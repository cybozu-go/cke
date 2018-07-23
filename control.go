package cke

import (
	"context"
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/coreos/etcd/clientv3/concurrency"
	"github.com/cybozu-go/cmd"
	"github.com/cybozu-go/log"
)

// Controller manage operations
type Controller struct {
	session  *concurrency.Session
	interval time.Duration
	client   *cmd.HTTPClient
}

// NewController construct controller instance
func NewController(s *concurrency.Session, interval time.Duration) Controller {
	client := &cmd.HTTPClient{
		Client: &http.Client{},
	}
	return Controller{s, interval, client}
}

// Run execute procedures with leader elections
func (c Controller) Run(ctx context.Context) error {
	hostname, err := os.Hostname()
	if err != nil {
		return err
	}

	e := concurrency.NewElection(c.session, KeyLeader)

RETRY:
	select {
	case <-c.session.Done():
		return errors.New("session has been orphaned")
	default:
	}

	err = e.Campaign(ctx, hostname)
	if err != nil {
		return err
	}

	leaderKey := e.Key()
	log.Info("I am the leader", map[string]interface{}{
		"session": c.session.Lease(),
	})

	err = c.runLoop(ctx, leaderKey)
	err2 := e.Resign(context.Background())
	if err2 != nil {
		return err2
	}
	if err == ErrNoLeader {
		log.Warn("lost the leadership", nil)
		goto RETRY
	}
	return err
}

func (c Controller) runLoop(ctx context.Context, leaderKey string) error {
	err := c.checkLastOp(ctx, leaderKey)
	if err != nil {
		return err
	}

	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		err := c.runOnce(ctx, leaderKey, ticker.C)
		if err != nil {
			return err
		}
	}
}

func (c Controller) checkLastOp(ctx context.Context, leaderKey string) error {
	storage := Storage{c.session.Client()}
	records, err := storage.GetRecords(ctx, 1)
	if err != nil {
		return err
	}
	if len(records) == 0 {
		return nil
	}

	r := records[0]
	if r.Status == StatusCancelled || r.Status == StatusCompleted {
		return nil
	}

	log.Warn("cancel the orphaned operation", map[string]interface{}{
		"id": r.ID,
		"op": r.Operation,
	})
	r.Cancel()
	return storage.UpdateRecord(ctx, leaderKey, r)
}

func (c Controller) runOnce(ctx context.Context, leaderKey string, tick <-chan time.Time) error {
	wait := false
	defer func() {
		if !wait {
			return
		}
		select {
		case <-ctx.Done():
		case <-tick:
		}
	}()

	storage := Storage{c.session.Client()}
	cluster, err := storage.GetCluster(ctx)
	switch err {
	case ErrNotFound:
		wait = true
		return nil
	case nil:
	default:
		return err
	}

	err = cluster.Validate()
	if err != nil {
		log.Error("invalid cluster configuration", map[string]interface{}{
			log.FnError: err,
		})
		wait = true
		return nil
	}

	status, err := c.GetClusterStatus(ctx, cluster)
	if err != nil {
		wait = true
		log.Warn("failed to get cluster status", map[string]interface{}{
			log.FnError: err,
		})
		return nil
	}
	defer status.Destroy()

	op := DecideToDo(ctx, cluster, status)
	if op == nil {
		wait = true
		return nil
	}

	// register operation record
	id, err := storage.NextRecordID(ctx)
	if err != nil {
		return err
	}
	record := NewRecord(id, op.Name())
	err = storage.RegisterRecord(ctx, leaderKey, record)
	if err != nil {
		return err
	}
	log.Info("begin new operation", map[string]interface{}{
		"op": op.Name(),
	})

	for {
		commander := op.NextCommand()
		if commander == nil {
			break
		}

		// check the context before proceed
		select {
		case <-ctx.Done():
			record.Cancel()
			err = storage.UpdateRecord(ctx, leaderKey, record)
			if err != nil {
				return err
			}
			log.Info("interrupt the operation due to cancellation", map[string]interface{}{
				"op": op.Name(),
			})
			return nil
		default:
		}

		record.SetCommand(commander.Command())
		err = storage.UpdateRecord(ctx, leaderKey, record)
		if err != nil {
			return err
		}
		log.Info("execute a command", map[string]interface{}{
			"op":      op.Name(),
			"command": commander.Command().String(),
		})
		err = commander.Run(ctx)
		if err == nil {
			continue
		}

		log.Error("command failed", map[string]interface{}{
			log.FnError: err,
			"op":        op.Name(),
			"command":   commander.Command().String(),
		})
		record.SetError(err)
		err2 := storage.UpdateRecord(ctx, leaderKey, record)
		if err2 != nil {
			return err2
		}

		// return nil instead of err as command failure is handled gracefully.
		wait = true
		return nil
	}

	record.Complete()
	err = storage.UpdateRecord(ctx, leaderKey, record)
	if err != nil {
		return err
	}
	log.Info("operation completed", map[string]interface{}{
		"op": op.Name(),
	})
	return nil
}

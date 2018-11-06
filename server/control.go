package server

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/coreos/etcd/clientv3/concurrency"
	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
)

var (
	errCommandFailure = errors.New("command failed")
)

// Controller manage operations
type Controller struct {
	session  *concurrency.Session
	interval time.Duration
	timeout  time.Duration
}

// NewController construct controller instance
func NewController(s *concurrency.Session, interval time.Duration, timeout time.Duration) Controller {
	return Controller{s, interval, timeout}
}

// Run execute procedures with leader elections
func (c Controller) Run(ctx context.Context) error {
	hostname, err := os.Hostname()
	if err != nil {
		return err
	}

	e := concurrency.NewElection(c.session, cke.KeyLeader)

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
	ctxWithTimeout, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	err2 := e.Resign(ctxWithTimeout)
	if err2 != nil {
		return err2
	}
	if err == cke.ErrNoLeader {
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

	watchChan := make(chan struct{})
	env := well.NewEnvironment(ctx)
	env.Go(func(ctx context.Context) error {
		return startWatcher(ctx, c.session.Client(), watchChan)
	})
	env.Stop()
	<-watchChan
	defer func() {
		env.Cancel(nil)
		env.Wait()
	}()

	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		err := c.runOnce(ctx, leaderKey, ticker.C, watchChan)
		if err != nil {
			return err
		}
	}
}

func (c Controller) checkLastOp(ctx context.Context, leaderKey string) error {
	storage := cke.Storage{c.session.Client()}
	records, err := storage.GetRecords(ctx, 1)
	if err != nil {
		return err
	}
	if len(records) == 0 {
		return nil
	}

	r := records[0]
	if r.Status == cke.StatusCancelled || r.Status == cke.StatusCompleted {
		return nil
	}

	log.Warn("cancel the orphaned operation", map[string]interface{}{
		"id": r.ID,
		"op": r.Operation,
	})
	r.Cancel()
	return storage.UpdateRecord(ctx, leaderKey, r)
}

func (c Controller) runOnce(ctx context.Context, leaderKey string, tick <-chan time.Time, watchChan <-chan struct{}) error {
	wait := false
	defer func() {
		if !wait {
			return
		}
		select {
		case <-watchChan:
		case <-ctx.Done():
		case <-tick:
		}
	}()

	storage := cke.Storage{c.session.Client()}
	cluster, err := storage.GetCluster(ctx)
	switch err {
	case cke.ErrNotFound:
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

	inf, err := cke.NewInfrastructure(ctx, cluster, storage)
	if err != nil {
		wait = true
		log.Error("failed to initialize infrastructure", map[string]interface{}{
			log.FnError: err,
		})
		return nil
	}
	defer inf.Close()

	// prepare service account signing
	_, err = storage.GetServiceAccountCert(ctx)
	switch err {
	case nil:
	case cke.ErrNotFound:
		crt, key, err := cke.KubernetesCA{}.IssueForServiceAccount(ctx, inf)
		if err != nil {
			return err
		}
		err = storage.PutServiceAccountData(ctx, leaderKey, crt, key)
		if err != nil {
			return err
		}
	default:
		return err
	}

	status, err := c.GetClusterStatus(ctx, cluster, inf)
	if err != nil {
		wait = true
		log.Warn("failed to get cluster status", map[string]interface{}{
			log.FnError: err,
		})
		return nil
	}

	ops := DecideOps(cluster, status)
	if len(ops) == 0 {
		wait = true
		return nil
	}

	for _, op := range ops {
		err := runOp(ctx, op, leaderKey, storage, inf)
		switch err {
		case nil:
		case errCommandFailure:
			wait = true
			return nil
		default:
			return err
		}
	}

	return nil
}

func runOp(ctx context.Context, op cke.Operator, leaderKey string, storage cke.Storage, inf cke.Infrastructure) error {
	// register operation record
	id, err := storage.NextRecordID(ctx)
	if err != nil {
		return err
	}
	record := cke.NewRecord(id, op.Name())
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
		err = commander.Run(ctx, inf)
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

		// return errCommandFailure instead of err as command failure need to be
		// handled gracefully.
		return errCommandFailure
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

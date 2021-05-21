package server

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/metrics"
	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
	"go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/clientv3/concurrency"
)

var (
	errCommandFailure = errors.New("command failed")
)

// Controller manage operations
type Controller struct {
	session         *concurrency.Session
	interval        time.Duration
	certsGCInterval time.Duration
	timeout         time.Duration
	addon           Integrator
}

// NewController construct controller instance
func NewController(s *concurrency.Session, interval, gcInterval, timeout time.Duration, addon Integrator) Controller {
	return Controller{s, interval, gcInterval, timeout, addon}
}

// Run execute procedures with leader elections
func (c Controller) Run(ctx context.Context) error {
	metrics.UpdateLeader(false)

	hostname, err := os.Hostname()
	if err != nil {
		return err
	}

	e := concurrency.NewElection(c.session, cke.KeyLeader)

	// When the etcd is stopping, the Campaign will hang up.
	// So check the session and exit if the session is closed.
	doneCh := make(chan error)
	go func() {
		doneCh <- e.Campaign(ctx, hostname)
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-c.session.Done():
		return errors.New("failed to campaign: session is closed")
	case err := <-doneCh:
		if err != nil {
			return fmt.Errorf("failed to campaign: %s", err.Error())
		}
	}

	leaderKey := e.Key()
	log.Info("I am the leader", map[string]interface{}{
		"session": c.session.Lease(),
	})
	metrics.UpdateLeader(true)

	// Release the leader before terminating.
	defer func() {
		select {
		case <-c.session.Done():
			log.Warn("session is closed, skip resign", nil)
			return
		default:
			ctxWithTimeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			err := e.Resign(ctxWithTimeout)
			if err != nil {
				log.Error("failed to resign", map[string]interface{}{
					log.FnError: err,
				})
			}
		}
	}()

	if c.addon != nil {
		if err := c.addon.Init(ctx, leaderKey); err != nil {
			return fmt.Errorf("failed to init the addon: %w", err)
		}
	}

	return c.runLoop(ctx, leaderKey)
}

func (c Controller) runLoop(ctx context.Context, leaderKey string) error {
	err := c.checkLastOp(ctx, leaderKey)
	if err != nil {
		return err
	}

	watchChan := make(chan struct{})
	var addonChan chan struct{}

	env := well.NewEnvironment(ctx)
	if c.addon != nil {
		addonChan = make(chan struct{}, 1)
		env.Go(func(ctx context.Context) error {
			return c.addon.StartWatch(ctx, addonChan)
		})
	}

	// Watch my leadership.
	env.Go(func(ctx context.Context) error {
		ch := c.session.Client().Watch(ctx, leaderKey, clientv3.WithFilterPut())
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-c.session.Done():
				return errors.New("session is closed")
			case resp, ok := <-ch:
				if !ok {
					return errors.New("watch is closed")
				}
				if resp.Err() != nil {
					return resp.Err()
				}
				for _, ev := range resp.Events {
					if ev.Type == clientv3.EventTypeDelete {
						return errors.New("leader key is deleted")
					}
				}
			}
		}
	})

	env.Go(func(ctx context.Context) error {
		return startWatcher(ctx, c.session.Client(), watchChan)
	})

	env.Go(func(ctx context.Context) error {
		select {
		case <-watchChan:
		case <-ctx.Done():
			return ctx.Err()
		}
		ticker := time.NewTicker(c.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return nil
			default:
			}
			err := c.runOnce(ctx, leaderKey, ticker.C, watchChan, addonChan)
			if err != nil {
				return err
			}
		}
	})

	env.Go(func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		ticker := time.NewTicker(c.certsGCInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-ticker.C:
			}
			err := c.runTidyExpiredCertificates(ctx)
			if err != nil {
				return err
			}
		}
	})

	env.Stop()
	return env.Wait()
}

func (c Controller) checkLastOp(ctx context.Context, leaderKey string) error {
	storage := cke.Storage{
		Client: c.session.Client(),
	}
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

func (c Controller) runOnce(ctx context.Context, leaderKey string, tick <-chan time.Time, watchChan, addonChan <-chan struct{}) error {
	wait := false
	defer func() {
		if !wait {
			return
		}
		select {
		case <-watchChan:
		case <-addonChan:
		case <-ctx.Done():
		case <-tick:
		}
	}()

	storage := cke.Storage{
		Client: c.session.Client(),
	}

	ts := time.Now().UTC()
	cluster, err := storage.GetCluster(ctx)
	switch err {
	case cke.ErrNotFound:
		wait = true
		if c.addon != nil {
			return c.addon.Do(ctx, leaderKey)
		}
		return nil
	case nil:
	default:
		return err
	}

	err = cluster.Validate(false)
	if err != nil {
		log.Error("invalid cluster configuration", map[string]interface{}{
			log.FnError: err,
		})
		wait = true
		// lint:ignore nilerr  Try again.
		return nil
	}

	inf, err := cke.NewInfrastructure(ctx, cluster, storage)
	if err != nil {
		// When the vault token is revoked, the following error will be returned. In this case, CKE can not continue any operations.
		// Error: "Error making API request.\n\nURL: GET <<URL>>\nCode: 403. Errors:\n\n* permission denied"
		if strings.Contains(err.Error(), "403") {
			log.Error("vault token was revoked", map[string]interface{}{
				log.FnError: err,
			})
			return err
		}

		wait = true
		log.Error("failed to initialize infrastructure", map[string]interface{}{
			log.FnError: err,
		})
		// lint:ignore nilerr  Try again.
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
		// lint:ignore nilerr  Try again.
		return nil
	}

	constraints, err := inf.Storage().GetConstraints(ctx)
	if err != nil {
		return err
	}

	rcs, err := inf.Storage().GetAllResources(ctx)
	if err != nil {
		return err
	}

	re, err := inf.Storage().GetRebootsEntries(ctx)
	if err != nil {
		return err
	}
	metrics.UpdateReboot(len(re))

	var reboot *cke.RebootQueueEntry
	if len(re) > 0 {
		disabled, err := inf.Storage().IsRebootQueueDisabled(ctx)
		if err != nil {
			return err
		}
		if !disabled {
			reboot = re[0]
		}
	}
	ops, phase := DecideOps(cluster, status, constraints, rcs, reboot)

	st := &cke.ServerStatus{
		Phase:     phase,
		Timestamp: ts,
	}
	err = storage.SetStatus(ctx, c.session.Lease(), st)
	if err != nil {
		return err
	}
	metrics.UpdateOperationPhase(phase, ts)

	if len(ops) == 0 {
		wait = true
		if c.addon != nil {
			return c.addon.Do(ctx, leaderKey)
		}
		return nil
	}

	// Reflect sabakan machine status when CKE does not need to do
	// anything except for rebooting nodes.
	if c.addon != nil && phase == cke.PhaseRebootNodes {
		if err := c.addon.Do(ctx, leaderKey); err != nil {
			return err
		}
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
	record := cke.NewRecord(id, op.Name(), op.Targets())
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

		log.Info("record targets", map[string]interface{}{
			"op":      op.Name(),
			"targets": strings.Join(op.Targets(), " "),
		})

		record.SetCommand(commander.Command())
		err = storage.UpdateRecord(ctx, leaderKey, record)
		if err != nil {
			return err
		}
		log.Info("execute a command", map[string]interface{}{
			"op":      op.Name(),
			"command": commander.Command().String(),
		})
		err = commander.Run(ctx, inf, leaderKey)
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

	if iop, ok := op.(cke.InfoOperator); ok {
		record.SetInfo(iop.Info())
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

func (c Controller) runTidyExpiredCertificates(ctx context.Context) error {
	storage := cke.Storage{
		Client: c.session.Client(),
	}

	cfg, err := storage.GetVaultConfig(ctx)
	if err != nil {
		log.Warn("failed to get vault config. skip tidy", map[string]interface{}{
			log.FnError: err,
		})
		// lint:ignore nilerr  Tidy is not mandatory.
		return nil
	}

	client, _, err := cke.VaultClient(cfg)
	if err != nil {
		return err
	}

	for _, ca := range cke.CAKeys {
		if err := c.TidyExpiredCertificates(ctx, client, cke.VaultPKIKey(ca)); err != nil {
			log.Warn("failed to tidy expired certificates", map[string]interface{}{
				log.FnError: err,
				"ca":        ca,
			})
		}
	}

	return nil
}

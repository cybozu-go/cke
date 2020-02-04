package metrics

import (
	"context"
	"time"

	v3 "github.com/coreos/etcd/clientv3"
	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/log"
	"golang.org/x/sync/errgroup"
)

// Updater updates Prometheus metrics periodically
type Updater struct {
	interval time.Duration
	storage  *cke.Storage
}

// NewUpdater is the constructor for Updater
func NewUpdater(interval time.Duration, client *v3.Client) *Updater {
	storage := cke.Storage{
		Client: client,
	}
	return &Updater{interval, &storage}
}

// UpdateLoop is the func to update all metrics continuously
func (u *Updater) UpdateLoop(ctx context.Context) error {
	ticker := time.NewTicker(u.interval)
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			err := u.UpdateAllMetrics(ctx)
			if err != nil {
				log.Warn("failed to update metrics", map[string]interface{}{
					log.FnError: err.Error(),
				})
			}
		}
	}
}

// UpdateAllMetrics is the func to update all metrics once
func (u *Updater) UpdateAllMetrics(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)
	tasks := map[string]func(ctx context.Context) error{
		"updateOperationRunning": u.updateOperationRunning,
	}
	for key, task := range tasks {
		key, task := key, task
		g.Go(func() error {
			err := task(ctx)
			if err != nil {
				log.Warn("unable to update metrics", map[string]interface{}{
					"funcname":  key,
					log.FnError: err,
				})
			}
			return err
		})
	}
	return g.Wait()
}

func (u *Updater) updateOperationRunning(ctx context.Context) error {
	st, err := u.storage.GetStatus(ctx)
	if err != nil {
		return err
	}
	if st.Phase == cke.PhaseCompleted {
		OperationRunning.Set(0)
	} else {
		OperationRunning.Set(1)
	}
	return nil
}

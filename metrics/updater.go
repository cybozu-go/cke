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
		"updateMachineStatus": u.updateMachineStatus,
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

func (u *Updater) updateMachineStatus(ctx context.Context) error {
	// machines, err := u.storage.Machine.Query(ctx, nil)
	// if err != nil {
	// 	return err
	// }
	// for _, m := range machines {
	// 	if len(m.Spec.IPv4) == 0 {
	// 		return fmt.Errorf("unable to expose metrics, because machine have no IPv4 address; serial: %s", m.Spec.Serial)
	// 	}
	// 	for _, st := range sabakan.StateList {
	// 		labelValues := []string{st.String(), m.Spec.IPv4[0], m.Spec.Serial, fmt.Sprint(m.Spec.Rack), m.Spec.Role, m.Spec.Labels["machine-type"]}
	// 		if st == m.Status.State {
	// 			MachineStatus.WithLabelValues(labelValues...).Set(1)
	// 		} else {
	// 			MachineStatus.WithLabelValues(labelValues...).Set(0)
	// 		}
	// 	}
	// }
	return nil
}

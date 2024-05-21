package sabakan

import (
	"context"
	"errors"
	"time"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/metrics"
	"github.com/cybozu-go/cke/server"
	"github.com/cybozu-go/log"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type sabakanContextKey string

const (
	// WaitSecs is a context key to pass to change the wait seconds
	// before removing retired nodes from the cluster.
	WaitSecs = sabakanContextKey("wait secs")
)

type integrator struct {
	etcd *clientv3.Client
}

// NewIntegrator returns server.Integrator to add sabakan integration
// feature to CKE.
func NewIntegrator(etcd *clientv3.Client) server.Integrator {
	return integrator{etcd: etcd}
}

func (ig integrator) StartWatch(ctx context.Context, ch chan<- struct{}) error {
	wch := ig.etcd.Watch(ctx, "", clientv3.WithPrefix(), clientv3.WithFilterDelete())
	for resp := range wch {
		if err := resp.Err(); err != nil {
			return err
		}

		for _, ev := range resp.Events {
			switch string(ev.Kv.Key) {
			case cke.KeyConstraints, cke.KeySabakanTemplate, cke.KeySabakanURL:
				select {
				case ch <- struct{}{}:
				default:
				}
			}
		}
	}
	return nil
}

func (ig integrator) Init(ctx context.Context, leaderKey string) error {
	return ig.runGenerator(ctx, leaderKey, nil, true)
}

func (ig integrator) Do(ctx context.Context, leaderKey string, clusterStatus *cke.ClusterStatus) error {
	if err := ig.runGenerator(ctx, leaderKey, clusterStatus, false); err != nil {
		return err
	}
	return ig.runRepairer(ctx, clusterStatus)
}

// runGenerator references WaitSecs in ctx to change the wait second value.
func (ig integrator) runGenerator(ctx context.Context, leaderKey string, clusterStatus *cke.ClusterStatus, onlyRegenerate bool) error {
	st := cke.Storage{Client: ig.etcd}

	disabled, err := st.IsSabakanDisabled(ctx)
	if err != nil {
		return err
	}
	if disabled {
		return nil
	}

	tmpl, rev, err := st.GetSabakanTemplate(ctx)
	switch err {
	case cke.ErrNotFound:
		return nil
	case nil:
	default:
		return err
	}

	machines, err := QueryAvailable(ctx, st)
	if err != nil {
		// the error is either harmless (cke.ErrNotFound) or already
		// logged by well.HTTPClient.
		if err != cke.ErrNotFound {
			log.Warn("sabakan: query failed", map[string]interface{}{
				log.FnError: err,
			})
		}
		return nil
	}

	cluster, crev, err := st.GetClusterWithRevision(ctx)
	if err != nil && err != cke.ErrNotFound {
		return err
	}

	tmplUpdated := (rev != crev)

	cstr, err := st.GetConstraints(ctx)
	switch err {
	case cke.ErrNotFound:
		cstr = cke.DefaultConstraints()
	case nil:
	default:
		return err
	}

	g := NewGenerator(tmpl, cstr, machines, clusterStatus, time.Now())

	val := ctx.Value(WaitSecs)
	if val != nil {
		if secs, ok := val.(float64); ok {
			g.SetWaitSeconds(secs)
		}
	}

	var newc *cke.Cluster
	if onlyRegenerate {
		if cluster != nil && tmplUpdated {
			newc, err = g.Regenerate(cluster)
		}
	} else {
		if cluster == nil {
			newc, err = g.Generate()
		} else {
			newc, err = g.Update(cluster)
			if newc == nil && err == nil && tmplUpdated {
				newc, err = g.Regenerate(cluster)
			}
		}
	}

	if err != nil {
		metrics.UpdateSabakanIntegration(false, nil, 0, time.Now().UTC())
		log.Warn("sabakan: failed to generate cluster", map[string]interface{}{
			log.FnError: err,
		})
		// lint:ignore nilerr  Some restriction was not satisfied. Try again.
		return nil
	}
	metrics.UpdateSabakanIntegration(true, g.countWorkerByRole, len(g.nextUnused), time.Now().UTC())

	if newc == nil {
		log.Debug("sabakan: nothing to do", nil)
		return nil
	}

	return st.PutClusterWithTemplateRevision(ctx, newc, rev, leaderKey)
}

func (ig integrator) runRepairer(ctx context.Context, clusterStatus *cke.ClusterStatus) error {
	st := cke.Storage{Client: ig.etcd}

	disabled, err := st.IsAutoRepairDisabled(ctx)
	if err != nil {
		return err
	}
	if disabled {
		return nil
	}

	machines, err := QueryNonHealthy(ctx, st)
	if err != nil {
		if !errors.Is(err, cke.ErrNotFound) {
			log.Warn("query failed", map[string]interface{}{
				log.FnError: err,
			})
		}
		return nil
	}

	rebootEntries, err := st.GetRebootsEntries(ctx)
	if err != nil {
		return err
	}
	rebootEntries = cke.DedupRebootQueueEntries(rebootEntries)

	constraints, err := st.GetConstraints(ctx)
	if err != nil {
		return err
	}

	entries := Repairer(machines, clusterStatus.RepairQueue.Entries, rebootEntries, constraints, time.Now().UTC())

	for _, entry := range entries {
		err := st.RegisterRepairsEntry(ctx, entry)
		if err != nil {
			return err
		}
	}

	return nil
}

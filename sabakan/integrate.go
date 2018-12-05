package sabakan

import (
	"context"

	"github.com/coreos/etcd/clientv3"
	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/server"
	"github.com/cybozu-go/log"
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

func (ig integrator) Do(ctx context.Context, leaderKey string) error {
	st := cke.Storage{Client: ig.etcd}

	tmpl, rev, err := st.GetSabakanTemplate(ctx)
	switch err {
	case cke.ErrNotFound:
		return nil
	case nil:
	default:
		return err
	}

	machines, err := Query(ctx, st)
	if err != nil {
		// the error is either harmless (cke.ErrNotFound) or already
		// logged by well.HTTPClient.
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

	g := NewGenerator(cluster, tmpl, cstr, machines)
	var newc *cke.Cluster
	if cluster == nil {
		newc, err = g.Generate()
	} else {
		newc, err = g.Update()
		if newc == nil && err == nil && tmplUpdated {
			newc, err = g.Regenerate()
		}
	}
	if err != nil {
		log.Warn("sabakan: failed to generate cluster", map[string]interface{}{
			log.FnError: err,
		})
	}
	if newc == nil {
		return nil
	}

	return st.PutClusterWithTemplateRevision(ctx, newc, rev, leaderKey)
}

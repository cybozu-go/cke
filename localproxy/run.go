package localproxy

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/log"
)

// LocalProxy is the controller of kube-proxy and unbound running on the same server as CKE.
type LocalProxy struct {
	Interval    time.Duration
	Storage     cke.Storage
	ProxyConfig *cke.ProxyParams

	currentAP string
}

// Run starts the controller.
func (c *LocalProxy) Run(ctx context.Context) error {
	tick := time.NewTicker(c.Interval)
	defer tick.Stop()

	for {
		select {
		case <-tick.C:
			if err := c.runOnce(ctx); err != nil {
				return err
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func (c *LocalProxy) runOnce(ctx context.Context) error {
	inf := newInfrastructure(c.Storage)
	defer inf.Close()

	cluster, err := c.Storage.GetCluster(ctx)
	if errors.Is(err, cke.ErrNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to get cluster: %w", err)
	}

	if c.ProxyConfig != nil {
		cluster.Options.Proxy = *c.ProxyConfig
	}

	st, err := getStatus(ctx, inf)
	if err != nil {
		log.Error("failed to get status", map[string]interface{}{
			log.FnError: err,
		})
		//lint:ignore nilerr intentional
		return nil
	}

	ap, ops := decideOps(cluster, c.currentAP, st)
	if len(ops) == 0 {
		return nil
	}

	c.currentAP = ap

	for _, op := range ops {
		if err := runOp(ctx, op, inf); err != nil {
			log.Error("failed to run an operation", map[string]interface{}{
				"op":        op.Name(),
				log.FnError: err,
			})
		}
	}
	return nil
}

func runOp(ctx context.Context, op cke.Operator, inf cke.Infrastructure) error {
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
			log.Info("interrupt the operation due to cancellation", map[string]interface{}{
				"op": op.Name(),
			})
			return nil
		default:
		}

		log.Info("execute a command", map[string]interface{}{
			"op":      op.Name(),
			"command": commander.Command().String(),
		})
		err := commander.Run(ctx, inf, "")
		if err == nil {
			continue
		}
		log.Error("command failed", map[string]interface{}{
			log.FnError: err,
			"op":        op.Name(),
			"command":   commander.Command().String(),
		})
		return err
	}

	log.Info("operation completed", map[string]interface{}{
		"op": op.Name(),
	})
	return nil

}

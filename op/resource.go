package op

import (
	"context"
	"time"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/log"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"
)

const (
	resourceApplyRetryCount    = 30
	resourceApplyRetryInterval = 10 * time.Second
)

type resourceApplyOp struct {
	apiserver      *cke.Node
	resource       cke.ResourceDefinition
	forceConflicts bool

	finished bool
}

// ResourceApplyOp creates or updates a Kubernetes object.
func ResourceApplyOp(apiServer *cke.Node, resource cke.ResourceDefinition, forceConflicts bool) cke.Operator {
	return &resourceApplyOp{
		apiserver:      apiServer,
		resource:       resource,
		forceConflicts: forceConflicts,
	}
}

func (o *resourceApplyOp) Name() string {
	return "resource-apply"
}

func (o *resourceApplyOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}
	o.finished = true
	return o
}

func (o *resourceApplyOp) Targets() []string {
	return []string{
		o.apiserver.Address,
	}
}

// isNonRetryableError returns true if the error is permanent and retrying will not help.
func isNonRetryableError(err error) bool {
	if err == nil {
		return false
	}

	return apierrors.IsForbidden(err) ||
		apierrors.IsUnauthorized(err) ||
		apierrors.IsInvalid(err)
}

func (o *resourceApplyOp) Run(ctx context.Context, inf cke.Infrastructure, _ string) error {
	cfg, err := inf.K8sConfig(ctx, o.apiserver)
	if err != nil {
		return err
	}
	dc, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return err
	}
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(dc))

	dyn, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return err
	}

	var lastErr error
	for i := range resourceApplyRetryCount {
		lastErr = cke.ApplyResource(ctx, dyn, mapper, inf, o.resource.Definition, o.resource.Revision, o.forceConflicts)
		if lastErr == nil {
			return nil
		}
		if isNonRetryableError(lastErr) {
			return lastErr
		}

		log.Warn("failed to apply resource, will retry", map[string]any{
			"resource":  o.resource.String(),
			"attempt":   i + 1,
			log.FnError: lastErr,
		})

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(resourceApplyRetryInterval):
		}
	}

	return lastErr
}

func (o *resourceApplyOp) Command() cke.Command {
	return cke.Command{
		Name:   "apply-resource",
		Target: o.resource.String(),
	}
}

package op

import (
	"context"

	"github.com/cybozu-go/cke"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"
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
	return cke.ApplyResource(ctx, dyn, mapper, inf, o.resource.Definition, o.resource.Revision, true)
}

func (o *resourceApplyOp) Command() cke.Command {
	return cke.Command{
		Name:   "apply-resource",
		Target: o.resource.String(),
	}
}

type resourceDeleteOp struct {
	apiserver    *cke.Node
	resourceKey  string
	resourceData []byte

	finished bool
}

// ResourceDeleteOp creates or updates a Kubernetes object.
func ResourceDeleteOp(apiServer *cke.Node, resourceKey string, resourceData []byte) cke.Operator {
	return &resourceDeleteOp{
		apiserver:    apiServer,
		resourceKey:  resourceKey,
		resourceData: resourceData,
	}
}

func (o *resourceDeleteOp) Name() string {
	return "resource-deletion"
}

func (o *resourceDeleteOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}
	o.finished = true
	return o
}

func (o *resourceDeleteOp) Targets() []string {
	return []string{
		o.apiserver.Address,
	}
}

func (o *resourceDeleteOp) Run(ctx context.Context, inf cke.Infrastructure, _ string) error {
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
	return cke.DeleteResource(ctx, dyn, mapper, inf, o.resourceKey, o.resourceData)
}

func (o *resourceDeleteOp) Command() cke.Command {
	return cke.Command{
		Name:   "delete-resource",
		Target: o.resourceKey,
	}
}

package op

import (
	"context"

	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"

	"github.com/cybozu-go/cke"
)

type kubeEndpointsDeleteOp struct {
	apiserver *cke.Node
	namespace string
	name      string
	finished  bool
}

// KubeEndpointsDeleteOp returns an Operator to delete Endpoints resource.
// CKE no longer maintains Endpoints, so it deletes the ones created by older versions.
func KubeEndpointsDeleteOp(apiserver *cke.Node, namespace, name string) cke.Operator {
	return &kubeEndpointsDeleteOp{
		apiserver: apiserver,
		namespace: namespace,
		name:      name,
	}
}

func (o *kubeEndpointsDeleteOp) Name() string {
	return "delete-" + o.name + "-endpoints"
}

func (o *kubeEndpointsDeleteOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}

	o.finished = true
	return deleteEndpointsCommand{o.apiserver, o.namespace, o.name}
}

func (o *kubeEndpointsDeleteOp) Targets() []string {
	return []string{
		o.apiserver.Address,
	}
}

type deleteEndpointsCommand struct {
	apiserver *cke.Node
	namespace string
	name      string
}

func (c deleteEndpointsCommand) Run(ctx context.Context, inf cke.Infrastructure, _ string) error {
	epAPI, err := legacyEndpointsAPI(ctx, inf, c.apiserver)
	if err != nil {
		return err
	}

	err = epAPI(c.namespace).Delete(ctx, c.name, metav1.DeleteOptions{})
	if k8serr.IsNotFound(err) {
		return nil
	}

	return err
}

func (c deleteEndpointsCommand) Command() cke.Command {
	return cke.Command{
		Name:   "deleteEndpointsCommand",
		Target: c.namespace + "/" + c.name,
	}
}

// legacyEndpointsAPI returns the Endpoints API that discards deprecation warnings.
func legacyEndpointsAPI(ctx context.Context, inf cke.Infrastructure, n *cke.Node) (func(namespace string) corev1client.EndpointsInterface, error) {
	cfg, err := inf.K8sConfig(ctx, n)
	if err != nil {
		return nil, err
	}
	cfg.WarningHandler = rest.NoWarnings{}
	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	return cs.CoreV1().Endpoints, nil
}

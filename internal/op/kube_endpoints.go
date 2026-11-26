package op

import (
	"context"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"

	"github.com/cybozu-go/cke"
)

type kubeEndpointsCreateOp struct {
	apiserver *cke.Node
	//nolint:staticcheck // code for Endpoints will be removed later
	endpoints *corev1.Endpoints
	finished  bool
}

// KubeEndpointsCreateOp returns an Operator to create Endpoints resource.
//
//nolint:staticcheck // code for Endpoints will be removed later
func KubeEndpointsCreateOp(apiserver *cke.Node, ep *corev1.Endpoints) cke.Operator {
	return &kubeEndpointsCreateOp{
		apiserver: apiserver,
		endpoints: ep,
	}
}

func (o *kubeEndpointsCreateOp) Name() string {
	return "create-" + o.endpoints.Name + "-endpoints"
}

func (o *kubeEndpointsCreateOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}

	o.finished = true
	return createEndpointsCommand{o.apiserver, o.endpoints}
}

func (o *kubeEndpointsCreateOp) Targets() []string {
	return []string{
		o.apiserver.Address,
	}
}

type kubeEndpointsUpdateOp struct {
	apiserver *cke.Node
	//nolint:staticcheck // code for Endpoints will be removed later
	endpoints *corev1.Endpoints
	finished  bool
}

// KubeEndpointsUpdateOp returns an Operator to update Endpoints resource.
//
//nolint:staticcheck // code for Endpoints will be removed later
func KubeEndpointsUpdateOp(apiserver *cke.Node, ep *corev1.Endpoints) cke.Operator {
	return &kubeEndpointsUpdateOp{
		apiserver: apiserver,
		endpoints: ep,
	}
}

func (o *kubeEndpointsUpdateOp) Name() string {
	return "update-" + o.endpoints.Name + "-endpoints"
}

func (o *kubeEndpointsUpdateOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}

	o.finished = true
	return updateEndpointsCommand{o.apiserver, o.endpoints}
}

func (o *kubeEndpointsUpdateOp) Targets() []string {
	return []string{
		o.apiserver.Address,
	}
}

type createEndpointsCommand struct {
	apiserver *cke.Node
	//nolint:staticcheck // code for Endpoints will be removed later
	endpoints *corev1.Endpoints
}

func (c createEndpointsCommand) Run(ctx context.Context, inf cke.Infrastructure, _ string) error {
	epAPI, err := legacyEndpointsAPI(ctx, inf, c.apiserver)
	if err != nil {
		return err
	}

	_, err = epAPI(c.endpoints.Namespace).Create(ctx, c.endpoints, metav1.CreateOptions{})

	return err
}

func (c createEndpointsCommand) Command() cke.Command {
	endpoints := make([]string, len(c.endpoints.Subsets[0].Addresses))
	for i, e := range c.endpoints.Subsets[0].Addresses {
		endpoints[i] = e.IP
	}
	return cke.Command{
		Name:   "createEndpointsCommand",
		Target: strings.Join(endpoints, ","),
	}
}

type updateEndpointsCommand struct {
	apiserver *cke.Node
	//nolint:staticcheck // code for Endpoints will be removed later
	endpoints *corev1.Endpoints
}

func (c updateEndpointsCommand) Run(ctx context.Context, inf cke.Infrastructure, _ string) error {
	epAPI, err := legacyEndpointsAPI(ctx, inf, c.apiserver)
	if err != nil {
		return err
	}

	_, err = epAPI(c.endpoints.Namespace).Update(ctx, c.endpoints, metav1.UpdateOptions{})

	return err
}

func (c updateEndpointsCommand) Command() cke.Command {
	endpoints := make([]string, len(c.endpoints.Subsets[0].Addresses))
	for i, e := range c.endpoints.Subsets[0].Addresses {
		endpoints[i] = e.IP
	}
	return cke.Command{
		Name:   "updateEndpointsCommand",
		Target: strings.Join(endpoints, ","),
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

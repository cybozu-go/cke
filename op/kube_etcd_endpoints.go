package op

import (
	"context"
	"strings"

	"github.com/cybozu-go/cke"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type kubeEtcdEndpointsCreateOp struct {
	apiserver *cke.Node
	endpoints []*cke.Node
	finished  bool
}

// KubeEtcdEndpointsCreateOp returns an Operator to create Endpoints resource for etcd.
func KubeEtcdEndpointsCreateOp(apiserver *cke.Node, cpNodes []*cke.Node) cke.Operator {
	return &kubeEtcdEndpointsCreateOp{
		apiserver: apiserver,
		endpoints: cpNodes,
	}
}

func (o *kubeEtcdEndpointsCreateOp) Name() string {
	return "create-etcd-endpoints"
}

func (o *kubeEtcdEndpointsCreateOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}

	o.finished = true
	return createEtcdEndpointsCommand{o.apiserver, o.endpoints}
}

func (o *kubeEtcdEndpointsCreateOp) Targets() []string {
	return []string{
		o.apiserver.Address,
	}
}

type kubeEtcdEndpointsUpdateOp struct {
	apiserver *cke.Node
	endpoints []*cke.Node
	finished  bool
}

// KubeEtcdEndpointsUpdateOp returns an Operator to update Endpoints resource for etcd.
func KubeEtcdEndpointsUpdateOp(apiserver *cke.Node, cpNodes []*cke.Node) cke.Operator {
	return &kubeEtcdEndpointsUpdateOp{
		apiserver: apiserver,
		endpoints: cpNodes,
	}
}

func (o *kubeEtcdEndpointsUpdateOp) Name() string {
	return "update-etcd-endpoints"
}

func (o *kubeEtcdEndpointsUpdateOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}

	o.finished = true
	return updateEtcdEndpointsCommand{o.apiserver, o.endpoints}
}

func (o *kubeEtcdEndpointsUpdateOp) Targets() []string {
	return []string{
		o.apiserver.Address,
	}
}

type createEtcdEndpointsCommand struct {
	apiserver *cke.Node
	endpoints []*cke.Node
}

func (c createEtcdEndpointsCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	cs, err := inf.K8sClient(ctx, c.apiserver)
	if err != nil {
		return err
	}

	// Endpoints needs a corresponding Service.
	// If an Endpoints lacks such a Service, it will be removed.
	// https://github.com/kubernetes/kubernetes/blob/b7c2d923ef4e166b9572d3aa09ca72231b59b28b/pkg/controller/endpoint/endpoints_controller.go#L392-L397
	services := cs.CoreV1().Services("kube-system")
	_, err = services.Get(EtcdEndpointsName, metav1.GetOptions{})
	switch {
	case err == nil:
	case errors.IsNotFound(err):
		_, err = services.Create(&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name: EtcdEndpointsName,
			},
			Spec: corev1.ServiceSpec{
				Ports:     []corev1.ServicePort{{Port: 2379}},
				ClusterIP: "None",
			},
		})
		if err != nil {
			return err
		}
	default:
		return err
	}

	subset := corev1.EndpointSubset{
		Addresses: make([]corev1.EndpointAddress, len(c.endpoints)),
		Ports:     []corev1.EndpointPort{{Port: 2379}},
	}
	for i, n := range c.endpoints {
		subset.Addresses[i].IP = n.Address
	}

	_, err = cs.CoreV1().Endpoints("kube-system").Create(&corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name: EtcdEndpointsName,
		},
		Subsets: []corev1.EndpointSubset{subset},
	})

	return err
}

func (c createEtcdEndpointsCommand) Command() cke.Command {
	endpoints := make([]string, len(c.endpoints))
	for i, e := range c.endpoints {
		endpoints[i] = e.Address
	}
	return cke.Command{
		Name:   "createEtcdEndpointsCommand",
		Target: strings.Join(endpoints, ","),
	}
}

type updateEtcdEndpointsCommand struct {
	apiserver *cke.Node
	endpoints []*cke.Node
}

func (c updateEtcdEndpointsCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	cs, err := inf.K8sClient(ctx, c.apiserver)
	if err != nil {
		return err
	}

	subset := corev1.EndpointSubset{
		Addresses: make([]corev1.EndpointAddress, len(c.endpoints)),
		Ports:     []corev1.EndpointPort{{Port: 2379}},
	}
	for i, n := range c.endpoints {
		subset.Addresses[i].IP = n.Address
	}

	_, err = cs.CoreV1().Endpoints("kube-system").Update(&corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name: EtcdEndpointsName,
		},
		Subsets: []corev1.EndpointSubset{subset},
	})

	return err
}

func (c updateEtcdEndpointsCommand) Command() cke.Command {
	endpoints := make([]string, len(c.endpoints))
	for i, e := range c.endpoints {
		endpoints[i] = e.Address
	}
	return cke.Command{
		Name:   "updateEtcdEndpointsCommand",
		Target: strings.Join(endpoints, ","),
	}
}

package op

import (
	"context"

	"github.com/cybozu-go/cke"
	corev1 "k8s.io/api/core/v1"
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

type createEtcdEndpointsCommand struct {
	apiserver *cke.Node
	endpoints []*cke.Node
}

func (c createEtcdEndpointsCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
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

	_, err = cs.CoreV1().Endpoints("kube-system").Create(&corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name: etcdEndpointsName,
		},
		Subsets: []corev1.EndpointSubset{subset},
	})

	return err
}

func (c createEtcdEndpointsCommand) Command() cke.Command {
	return cke.Command{
		Name:   "createEtcdEndpointsCommand",
		Target: "kube-system/" + etcdEndpointsName,
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
			Name: etcdEndpointsName,
		},
		Subsets: []corev1.EndpointSubset{subset},
	})

	return err
}

func (c updateEtcdEndpointsCommand) Command() cke.Command {
	return cke.Command{
		Name:   "updateEtcdEndpointsCommand",
		Target: "kube-system/" + etcdEndpointsName,
	}
}

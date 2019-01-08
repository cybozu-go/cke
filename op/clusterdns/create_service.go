package clusterdns

import (
	"context"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type kubeCreateServiceOp struct {
	apiserver *cke.Node
	finished  bool
}

// CreateOp returns an Operator to create cluster resolver.
func CreateOp(apiserver *cke.Node) cke.Operator {
	return &kubeCreateServiceOp{
		apiserver: apiserver,
	}
}

func (o *kubeCreateServiceOp) Name() string {
	return "create-cluster-dns-service"
}

func (o *kubeCreateServiceOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}
	o.finished = true
	return createServiceCommand{o.apiserver}
}

type createServiceCommand struct {
	apiserver *cke.Node
}

func (c createServiceCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	cs, err := inf.K8sClient(ctx, c.apiserver)
	if err != nil {
		return err
	}

	// Service
	services := cs.CoreV1().Services("kube-system")
	_, err = services.Get(op.ClusterDNSAppName, v1.GetOptions{})
	switch {
	case err == nil:
	case errors.IsNotFound(err):
		_, err = services.Create(getService())
		if err != nil {
			return err
		}
	default:
		return err
	}

	return nil
}

func (c createServiceCommand) Command() cke.Command {
	return cke.Command{
		Name:   "createServiceCommand",
		Target: "kube-system",
	}
}

func getService() *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      op.ClusterDNSAppName,
			Namespace: "kube-system",
			Labels: map[string]string{
				op.CKELabelAppName: op.ClusterDNSAppName,
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				op.CKELabelAppName: op.ClusterDNSAppName,
			},
			Ports: []corev1.ServicePort{
				{
					Name:     "dns",
					Port:     53,
					Protocol: corev1.ProtocolUDP,
				},
				{
					Name:     "dns-tcp",
					Port:     53,
					Protocol: corev1.ProtocolTCP,
				},
			},
		},
	}
}

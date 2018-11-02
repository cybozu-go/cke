package op

import (
	"context"

	"github.com/cybozu-go/cke"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type kubeCoreDNSCreateOp struct {
	clusterDomain string
	clusterDNS    []string
}

// KubeCoreDNSCreateOp returns an Operator to create CoreDNS.
func KubeCoreDNSCreateOp(clusterDomain string, clusterDNS []string) cke.Operator {
	return &kubeCoreDNSCreateOp{
		clusterDomain: clusterDomain,
		clusterDNS:    clusterDNS,
	}
}

func (o *kubeCoreDNSCreateOp) Name() string {
	return "create-coredns"
}

func (o *kubeCoreDNSCreateOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}

	o.finished = true
	return createCoreDNSCommand{o.clusterDomain, o.clusterDNS}
}

type kubeCoreDNSUpdateOp struct {
	clusterDomain string
	clusterDNS    []string
	finished      bool
}

// KubeCoreDNSUpdateOp returns an Operator to update CoreDNS.
func KubeCoreDNSUpdateOp(clusterDomain string, clusterDNS []string) cke.Operator {
	return &kubeCoreDNSCreateOp{
		clusterDomain: clusterDomain,
		clusterDNS:    clusterDNS,
	}
}

func (o *kubeCoreDNSUpdateOp) Name() string {
	return "update-coredns"
}

func (o *kubeCoreDNSUpdateOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}

	o.finished = true
	return updateCoreDNSCommand{o.clusterDomain, o.clusterDNS}
}

type createCoreDNSCommand struct {
	clusterDomain string
	clusterDNS    []string
	params        cke.KubeletParams
}

func (c createCoreDNSCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	cs, err := inf.K8sClient(ctx, c.apiserver)
	if err != nil {
		return err
	}

	// ServiceAccount

	// ClusterRole

	// ClusterRoleBinding

	// ConfigMap

	// Deployment

	// Service
	services := cs.CoreV1().Services("kube-system")
	_, err = services.Get(coreDNSAppName, metav1.GetOptions{})
	switch {
	case err == nil:
	case errors.IsNotFound(err):
		_, err = services.Create(&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      coreDNSAppName,
				NameSpace: "kube-system",
				Labels: map[string]string{
					"k8s-app":                       coreDNSAppName,
					"kubernetes.io/cluster-service": "true",
					"kubernetes.io/name":            "CoreDNS",
				},
			},
			Spec: corev1.ServiceSpec{
				Selector: map[string]string{
					"k8s-app": coreDNSAppName,
				},
				ClusterIP: c.params.ClusterDNS,
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
		})
		if err != nil {
			return err
		}
	default:
		return err
	}

	return err
}

func (c createCoreDNSCommand) Command() cke.Command {
	return cke.Command{
		Name:   "createCoreDNSCommand",
		Target: "kube-system",
	}
}

type updateCoreDNSCommand struct {
	clusterDomain string
	clusterDNS    []string
	params        cke.KubeletParams
}

func (c updateCoreDNSCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	cs, err := inf.K8sClient(ctx, c.apiserver)
	if err != nil {
		return err
	}

	// TODO

	return err
}

func (c updateEtcdEndpointsCommand) Command() cke.Command {
	return cke.Command{
		Name:   "updateCoreDNSCommand",
		Target: "kube-system",
	}
}

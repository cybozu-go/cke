package op

import (
	"context"

	"github.com/cybozu-go/cke"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type kubeCoreDNSCreateOp struct {
	apiserver *cke.Node
	finished  bool
	params    cke.KubeletParams
}

// KubeCoreDNSCreateOp returns an Operator to create CoreDNS.
func KubeCoreDNSCreateOp(apiserver *cke.Node, params cke.KubeletParams, finished bool) cke.Operator {
	return &kubeCoreDNSCreateOp{
		apiserver: apiserver,
		params:    params,
		finished:  finished,
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
	return createCoreDNSCommand{o.apiserver, o.params}
}

type createCoreDNSCommand struct {
	apiserver *cke.Node
	params    cke.KubeletParams
}

func (c createCoreDNSCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	cs, err := inf.K8sClient(ctx, c.apiserver)
	if err != nil {
		return err
	}

	// ServiceAccount
	accounts := cs.CoreV1().ServiceAccounts("kube-system")
	_, err = accounts.Get(coreDNSAppName, metav1.GetOptions{})
	switch {
	case err == nil:
	case errors.IsNotFound(err):
		_, err = accounts.Create(&corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      coreDNSAppName,
				Namespace: "kube-system",
			},
		})
		if err != nil {
			return err
		}
	default:
		return err
	}

	// ClusterRole
	_, err = cs.RbacV1().ClusterRoles().Create(&rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: coreDNSRBACRoleName,
			Labels: map[string]string{
				"kubernetes.io/bootstrapping": "rbac-defaults",
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{
					"endpoints",
					"services",
					"pods",
					"namespaces",
				},
				Verbs: []string{
					"list",
					"watch",
				},
			},
		},
	})
	if err != nil {
		return err
	}

	// ClusterRoleBinding
	_, err = cs.RbacV1().ClusterRoleBindings().Create(&rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: coreDNSRBACRoleName,
			Labels: map[string]string{
				"kubernetes.io/bootstrapping": "rbac-defaults",
			},
			Annotations: map[string]string{
				// turn on auto-reconciliation
				// https://kubernetes.io/docs/reference/access-authn-authz/rbac/#auto-reconciliation
				rbacv1.AutoUpdateAnnotationKey: "true",
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     coreDNSRBACRoleName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      coreDNSAppName,
				Namespace: "kube-system",
			},
		},
	})
	if err != nil {
		return err
	}

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
				Namespace: "kube-system",
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
				ClusterIP: c.params.DNS,
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

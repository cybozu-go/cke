package clusterdns

import (
	"context"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op"
	v12 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

type createRBACRoleBindingOp struct {
	apiserver  *cke.Node
	domain     string
	dnsServers []string
	finished   bool
}

// CreateRBACRoleBindingOp returns an Operator to create RBAC Role Binding for CoreDNS.
func CreateRBACRoleBindingOp(apiserver *cke.Node) cke.Operator {
	return &createRBACRoleBindingOp{
		apiserver: apiserver,
	}
}

func (o *createRBACRoleBindingOp) Name() string {
	return "create-cluster-dns-rbac-role-binding"
}

func (o *createRBACRoleBindingOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}
	o.finished = true
	return createRBACRoleBindingCommand{o.apiserver}
}

type createRBACRoleBindingCommand struct {
	apiserver *cke.Node
}

func (c createRBACRoleBindingCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	cs, err := inf.K8sClient(ctx, c.apiserver)
	if err != nil {
		return err
	}
	// ClusterRoleBinding
	clusterRoleBindings := cs.RbacV1().ClusterRoleBindings()
	_, err = clusterRoleBindings.Get(op.ClusterDNSRBACRoleName, v1.GetOptions{})
	switch {
	case err == nil:
	case errors.IsNotFound(err):
		_, err = clusterRoleBindings.Create(&v12.ClusterRoleBinding{
			ObjectMeta: v1.ObjectMeta{
				Name: op.ClusterDNSRBACRoleName,
				Labels: map[string]string{
					"kubernetes.io/bootstrapping": "rbac-defaults",
				},
				Annotations: map[string]string{
					// turn on auto-reconciliation
					// https://kubernetes.io/docs/reference/access-authn-authz/rbac/#auto-reconciliation
					v12.AutoUpdateAnnotationKey: "true",
				},
			},
			RoleRef: v12.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     op.ClusterDNSRBACRoleName,
			},
			Subjects: []v12.Subject{
				{
					Kind:      v12.ServiceAccountKind,
					Name:      op.ClusterDNSAppName,
					Namespace: "kube-system",
				},
			},
		})
		if err != nil {
			return err
		}
	default:
		return err
	}

	return nil
}

func (c createRBACRoleBindingCommand) Command() cke.Command {
	return cke.Command{
		Name:   "createRBACRoleBindingCommand",
		Target: "kube-system",
	}
}

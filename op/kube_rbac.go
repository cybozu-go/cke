package op

import (
	"context"

	"github.com/cybozu-go/cke"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type kubeRBACRoleInstallOp struct {
	apiserver     *cke.Node
	roleExists    bool
	bindingExists bool
}

// KubeRBACRoleInstallOp returns an Operator to install ClusterRole and binding for RBAC.
func KubeRBACRoleInstallOp(apiserver *cke.Node, roleExists bool) cke.Operator {
	return &kubeRBACRoleInstallOp{
		apiserver:  apiserver,
		roleExists: roleExists,
	}
}

func (o *kubeRBACRoleInstallOp) Name() string {
	return "install-rbac-role"
}

func (o *kubeRBACRoleInstallOp) NextCommand() cke.Commander {
	switch {
	case !o.roleExists:
		o.roleExists = true
		return makeRBACRoleCommand{o.apiserver}
	case !o.bindingExists:
		o.bindingExists = true
		return makeRBACRoleBindingCommand{o.apiserver}
	}
	return nil
}

type makeRBACRoleCommand struct {
	apiserver *cke.Node
}

func (c makeRBACRoleCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	cs, err := inf.K8sClient(ctx, c.apiserver)
	if err != nil {
		return err
	}

	_, err = cs.RbacV1().ClusterRoles().Create(&rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: rbacRoleName,
			Labels: map[string]string{
				"kubernetes.io/bootstrapping": "rbac-defaults",
			},
			Annotations: map[string]string{
				// turn on auto-reconciliation
				// https://kubernetes.io/docs/reference/access-authn-authz/rbac/#auto-reconciliation
				"rbac.authorization.kubernetes.io/autoupdate": "true",
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				// these are virtual resources.
				// see https://github.com/kubernetes/kubernetes/issues/44330#issuecomment-293768369
				Resources: []string{
					"nodes/proxy",
					"nodes/stats",
					"nodes/log",
					"nodes/spec",
					"nodes/metrics",
				},
				Verbs: []string{"*"},
			},
		},
	})

	return err
}

func (c makeRBACRoleCommand) Command() cke.Command {
	return cke.Command{
		Name:   "makeClusterRole",
		Target: rbacRoleName,
	}
}

type makeRBACRoleBindingCommand struct {
	apiserver *cke.Node
}

func (c makeRBACRoleBindingCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	cs, err := inf.K8sClient(ctx, c.apiserver)
	if err != nil {
		return err
	}

	_, err = cs.RbacV1().ClusterRoleBindings().Create(&rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: rbacRoleBindingName,
			Labels: map[string]string{
				"kubernetes.io/bootstrapping": "rbac-defaults",
			},
			Annotations: map[string]string{
				// turn on auto-reconciliation
				// https://kubernetes.io/docs/reference/access-authn-authz/rbac/#auto-reconciliation
				"rbac.authorization.kubernetes.io/autoupdate": "true",
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     rbacRoleName,
		},
		Subjects: []rbacv1.Subject{
			{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "User",
				Name:     "kubernetes",
			},
		},
	})

	return err
}

func (c makeRBACRoleBindingCommand) Command() cke.Command {
	return cke.Command{
		Name:   "makeClusterRoleBinding",
		Target: rbacRoleBindingName,
	}
}

package clusterdns

import (
	"context"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op"
	v12 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

type createRBACRoleOp struct {
	apiserver *cke.Node
	finished  bool
}

// CreateRBACRoleOp returns an Operator to create RBAC Role for CoreDNS.
func CreateRBACRoleOp(apiserver *cke.Node) cke.Operator {
	return &createRBACRoleOp{
		apiserver: apiserver,
	}
}

func (o *createRBACRoleOp) Name() string {
	return "create-cluster-dns-rbac-role"
}

func (o *createRBACRoleOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}
	o.finished = true
	return createRBACRoleCommand{o.apiserver}
}

type createRBACRoleCommand struct {
	apiserver *cke.Node
}

func (c createRBACRoleCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	cs, err := inf.K8sClient(ctx, c.apiserver)
	if err != nil {
		return err
	}

	// ClusterRole
	clusterRoles := cs.RbacV1().ClusterRoles()
	_, err = clusterRoles.Get(op.ClusterDNSRBACRoleName, v1.GetOptions{})
	switch {
	case err == nil:
	case errors.IsNotFound(err):
		_, err = clusterRoles.Create(&v12.ClusterRole{
			ObjectMeta: v1.ObjectMeta{
				Name: op.ClusterDNSRBACRoleName,
				Labels: map[string]string{
					"kubernetes.io/bootstrapping": "rbac-defaults",
				},
			},
			Rules: []v12.PolicyRule{
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
	default:
		return err
	}
	return nil
}

func (c createRBACRoleCommand) Command() cke.Command {
	return cke.Command{
		Name:   "createRBACRoleCommand",
		Target: "kube-system",
	}
}

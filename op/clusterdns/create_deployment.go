package clusterdns

import (
	"context"
	"strings"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op"
	v12 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

type createDeploymentOp struct {
	apiserver *cke.Node
	finished  bool
}

// CreateDeploymentOp returns an Operator to create deployment of CoreDNS.
func CreateDeploymentOp(apiserver *cke.Node) cke.Operator {
	return &createDeploymentOp{
		apiserver: apiserver,
	}
}

func (o *createDeploymentOp) Name() string {
	return "create-cluster-dns-deployment"
}

func (o *createDeploymentOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}
	o.finished = true
	return createDeploymentCommand{o.apiserver}
}

type createDeploymentCommand struct {
	apiserver *cke.Node
}

func (c createDeploymentCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	cs, err := inf.K8sClient(ctx, c.apiserver)
	if err != nil {
		return err
	}

	// Deployment
	deployments := cs.AppsV1().Deployments("kube-system")
	_, err = deployments.Get(op.ClusterDNSAppName, v1.GetOptions{})
	switch {
	case err == nil:
	case errors.IsNotFound(err):
		deployment := new(v12.Deployment)
		err = yaml.NewYAMLToJSONDecoder(strings.NewReader(deploymentText)).Decode(deployment)
		if err != nil {
			return err
		}
		_, err = deployments.Create(deployment)
		if err != nil {
			return err
		}
	default:
		return err
	}
	return nil
}

func (c createDeploymentCommand) Command() cke.Command {
	return cke.Command{
		Name:   "createDeploymentCommand",
		Target: "kube-system",
	}
}

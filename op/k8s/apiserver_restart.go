package k8s

import (
	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op"
	"github.com/cybozu-go/cke/op/common"
)

type apiServerRestartOp struct {
	nodes []*cke.Node
	cps   []*cke.Node

	serviceSubnet string
	params        cke.APIServerParams

	pulled bool
}

// APIServerRestartOp returns an Operator to restart kube-apiserver
func APIServerRestartOp(nodes, cps []*cke.Node, serviceSubnet string, params cke.APIServerParams) cke.Operator {
	return &apiServerRestartOp{
		nodes:         nodes,
		cps:           cps,
		serviceSubnet: serviceSubnet,
		params:        params,
	}
}

func (o *apiServerRestartOp) Name() string {
	return "kube-apiserver-restart"
}

func (o *apiServerRestartOp) NextCommand() cke.Commander {
	if !o.pulled {
		o.pulled = true
		return common.ImagePullCommand(o.nodes, cke.HyperkubeImage)
	}

	if len(o.nodes) == 0 {
		return nil
	}

	// API server should be restarted one by one.
	node := o.nodes[0]
	o.nodes = o.nodes[1:]
	opts := []string{
		"--mount", "type=tmpfs,dst=/run/kubernetes",
	}
	return common.RunContainerCommand([]*cke.Node{node},
		op.KubeAPIServerContainerName, cke.HyperkubeImage,
		common.WithOpts(opts),
		common.WithParams(APIServerParams(o.cps, node.Address, o.serviceSubnet, o.params.AuditLogEnabled, o.params.AuditLogPolicy)),
		common.WithExtra(o.params.ServiceParams),
		common.WithRestart())
}

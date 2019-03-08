package k8s

import (
	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op"
	"github.com/cybozu-go/cke/op/common"
)

type kubeProxyRestartOp struct {
	nodes []*cke.Node

	cluster string
	params  cke.ServiceParams

	pulled   bool
	finished bool
}

// KubeProxyRestartOp returns an Operator to restart kube-proxy.
func KubeProxyRestartOp(nodes []*cke.Node, cluster string, params cke.ServiceParams) cke.Operator {
	return &kubeProxyRestartOp{
		nodes:   nodes,
		cluster: cluster,
		params:  params,
	}
}

func (o *kubeProxyRestartOp) Name() string {
	return "kube-proxy-restart"
}

func (o *kubeProxyRestartOp) NextCommand() cke.Commander {
	if !o.pulled {
		o.pulled = true
		return common.ImagePullCommand(o.nodes, cke.HyperkubeImage)
	}

	if !o.finished {
		o.finished = true
		opts := []string{
			"--tmpfs=/run",
			"--privileged",
		}
		paramsMap := make(map[string]cke.ServiceParams)
		for _, n := range o.nodes {
			params := ProxyParams(n)
			paramsMap[n.Address] = params
		}
		return common.RunContainerCommand(o.nodes, op.KubeProxyContainerName, cke.HyperkubeImage,
			common.WithOpts(opts),
			common.WithParamsMap(paramsMap),
			common.WithExtra(o.params),
			common.WithRestart())
	}
	return nil
}

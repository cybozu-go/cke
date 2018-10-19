package op

import (
	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/common"
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
		return common.RunContainerCommand(o.nodes, kubeProxyContainerName, cke.HyperkubeImage,
			common.WithOpts(opts),
			common.WithParams(ProxyParams()),
			common.WithExtra(o.params),
			common.WithRestart())
	}
	return nil
}

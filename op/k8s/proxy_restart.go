package k8s

import (
	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op"
	"github.com/cybozu-go/cke/op/common"
)

type kubeProxyRestartOp struct {
	nodes []*cke.Node

	cluster string
	ap      string
	params  cke.ProxyParams

	step  int
	files *common.FilesBuilder
}

// KubeProxyRestartOp returns an Operator to restart kube-proxy.
func KubeProxyRestartOp(nodes []*cke.Node, cluster, ap string, params cke.ProxyParams) cke.Operator {
	return &kubeProxyRestartOp{
		nodes:   nodes,
		cluster: cluster,
		ap:      ap,
		params:  params,
		files:   common.NewFilesBuilder(nodes),
	}
}

func (o *kubeProxyRestartOp) Name() string {
	return "kube-proxy-restart"
}

func (o *kubeProxyRestartOp) NextCommand() cke.Commander {
	switch o.step {
	case 0:
		o.step++
		return common.ImagePullCommand(o.nodes, cke.KubernetesImage)
	case 1:
		o.step++
		return prepareProxyFilesCommand{cluster: o.cluster, ap: o.ap, files: o.files, params: o.params}
	case 2:
		o.step++
		return o.files
	case 3:
		o.step++
		opts := []string{
			"--tmpfs=/run",
			"--privileged",
		}
		paramsMap := make(map[string]cke.ServiceParams)
		for _, n := range o.nodes {
			params := ProxyParams()
			paramsMap[n.Address] = params
		}
		return common.RunContainerCommand(o.nodes, op.KubeProxyContainerName, cke.KubernetesImage,
			common.WithOpts(opts),
			common.WithParamsMap(paramsMap),
			common.WithExtra(o.params.ServiceParams),
			common.WithRestart())
	default:
		return nil
	}
}

func (o *kubeProxyRestartOp) Targets() []string {
	ips := make([]string, len(o.nodes))
	for i, n := range o.nodes {
		ips[i] = n.Address
	}
	return ips
}

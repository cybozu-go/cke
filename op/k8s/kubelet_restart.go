package k8s

import (
	"context"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op"
	"github.com/cybozu-go/cke/op/common"
	yaml "gopkg.in/yaml.v2"
)

type kubeletRestartOp struct {
	nodes []*cke.Node

	cluster   string
	podSubnet string
	params    cke.KubeletParams

	step  int
	files *common.FilesBuilder
}

// KubeletRestartOp returns an Operator to restart kubelet
func KubeletRestartOp(nodes []*cke.Node, cluster, podSubnet string, params cke.KubeletParams) cke.Operator {
	return &kubeletRestartOp{
		nodes:     nodes,
		cluster:   cluster,
		podSubnet: podSubnet,
		params:    params,
		files:     common.NewFilesBuilder(nodes),
	}
}

func (o *kubeletRestartOp) Name() string {
	return "kubelet-restart"
}

func (o *kubeletRestartOp) NextCommand() cke.Commander {
	switch o.step {
	case 0:
		o.step++
		return common.ImagePullCommand(o.nodes, cke.HyperkubeImage)
	case 1:
		o.step++
		return prepareKubeletConfigCommand{o.params, o.files}
	case 2:
		o.step++
		return o.files
	case 3:
		o.step++
		opts := []string{
			"--pid=host",
			"--privileged",
		}
		paramsMap := make(map[string]cke.ServiceParams)
		for _, n := range o.nodes {
			paramsMap[n.Address] = KubeletServiceParams(n, o.params)
		}
		return common.RunContainerCommand(o.nodes, op.KubeletContainerName, cke.HyperkubeImage,
			common.WithOpts(opts),
			common.WithParamsMap(paramsMap),
			common.WithExtra(o.params.ServiceParams),
			common.WithRestart())
	default:
		return nil
	}
}

type prepareKubeletConfigCommand struct {
	params cke.KubeletParams
	files  *common.FilesBuilder
}

func (c prepareKubeletConfigCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	const kubeletConfigPath = "/etc/kubernetes/kubelet/config.yml"
	caPath := op.K8sPKIPath("ca.crt")
	tlsCertPath := op.K8sPKIPath("kubelet.crt")
	tlsKeyPath := op.K8sPKIPath("kubelet.key")

	cfg := newKubeletConfiguration(tlsCertPath, tlsKeyPath, caPath, c.params.Domain,
		c.params.ContainerLogMaxSize, c.params.ContainerLogMaxFiles, c.params.AllowSwap)
	g := func(ctx context.Context, n *cke.Node) ([]byte, error) {
		cfg := cfg
		cfg.ClusterDNS = []string{n.Address}
		return yaml.Marshal(cfg)
	}
	return c.files.AddFile(ctx, kubeletConfigPath, g)
}

func (c prepareKubeletConfigCommand) Command() cke.Command {
	return cke.Command{
		Name: "prepare-kubelet-config",
	}
}

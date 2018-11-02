package op

import (
	"context"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/common"
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
		return common.ImagePullCommand(o.nodes, cke.PauseImage)
	case 2:
		o.step++
		return prepareKubeletConfigCommand{o.params, o.files}
	case 3:
		o.step++
		return o.files
	case 4:
		o.step++
		opts := []string{
			"--pid=host",
			"--mount=type=volume,src=dockershim,dst=/var/lib/dockershim",
			"--privileged",
		}
		paramsMap := make(map[string]cke.ServiceParams)
		for _, n := range o.nodes {
			paramsMap[n.Address] = KubeletServiceParams(n)
		}
		return common.RunContainerCommand(o.nodes, kubeletContainerName, cke.HyperkubeImage,
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
	caPath := K8sPKIPath("ca.crt")
	tlsCertPath := K8sPKIPath("kubelet.crt")
	tlsKeyPath := K8sPKIPath("kubelet.key")

	cfg := &KubeletConfiguration{
		APIVersion:            "kubelet.config.k8s.io/v1beta1",
		Kind:                  "KubeletConfiguration",
		ReadOnlyPort:          0,
		TLSCertFile:           tlsCertPath,
		TLSPrivateKeyFile:     tlsKeyPath,
		Authentication:        KubeletAuthentication{ClientCAFile: caPath},
		Authorization:         KubeletAuthorization{Mode: "Webhook"},
		HealthzBindAddress:    "0.0.0.0",
		ClusterDomain:         c.params.Domain,
		ClusterDNS:            c.params.DNS,
		RuntimeRequestTimeout: "15m",
		FailSwapOn:            !c.params.AllowSwap,
	}
	cfgData, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	g := func(ctx context.Context, n *cke.Node) ([]byte, error) {
		return cfgData, nil
	}
	return c.files.AddFile(ctx, kubeletConfigPath, g)
}

func (c prepareKubeletConfigCommand) Command() cke.Command {
	return cke.Command{
		Name: "prepare-kubelet-config",
	}
}

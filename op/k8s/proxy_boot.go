package k8s

import (
	"context"
	"fmt"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op"
	"github.com/cybozu-go/cke/op/common"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	proxyKubeconfigPath = "/etc/kubernetes/proxy/kubeconfig"
	proxyConfigPath     = "/etc/kubernetes/proxy/config.yml"
)

type kubeProxyBootOp struct {
	nodes []*cke.Node

	cluster string
	ap      string
	params  cke.ProxyParams

	step  int
	files *common.FilesBuilder
}

// KubeProxyBootOp returns an Operator to boot kube-proxy.
func KubeProxyBootOp(nodes []*cke.Node, cluster, ap string, params cke.ProxyParams) cke.Operator {
	return &kubeProxyBootOp{
		nodes:   nodes,
		ap:      ap,
		cluster: cluster,
		params:  params,
		files:   common.NewFilesBuilder(nodes),
	}
}

func (o *kubeProxyBootOp) Name() string {
	return "kube-proxy-bootstrap"
}

func (o *kubeProxyBootOp) NextCommand() cke.Commander {
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
			common.WithExtra(o.params.ServiceParams))
	default:
		return nil
	}
}

func (o *kubeProxyBootOp) Targets() []string {
	ips := make([]string, len(o.nodes))
	for i, n := range o.nodes {
		ips[i] = n.Address
	}
	return ips
}

type prepareProxyFilesCommand struct {
	cluster string
	ap      string
	files   *common.FilesBuilder
	params  cke.ProxyParams
}

func (c prepareProxyFilesCommand) Run(ctx context.Context, inf cke.Infrastructure, _ string) error {
	storage := inf.Storage()

	ca, err := storage.GetCACertificate(ctx, "kubernetes")
	if err != nil {
		return err
	}
	g := func(ctx context.Context, n *cke.Node) ([]byte, error) {
		crt, key, err := cke.KubernetesCA{}.IssueForProxy(ctx, inf)
		if err != nil {
			return nil, err
		}
		cfg := proxyKubeconfig(c.cluster, ca, crt, key, c.ap)
		return clientcmd.Write(*cfg)
	}
	if err := c.files.AddFile(ctx, proxyKubeconfigPath, g); err != nil {
		return err
	}

	return c.files.AddFile(ctx, proxyConfigPath, func(ctx context.Context, n *cke.Node) ([]byte, error) {
		cfg := GenerateProxyConfiguration(c.params, n)
		return encodeToYAML(cfg)
	})
}

func (c prepareProxyFilesCommand) Command() cke.Command {
	return cke.Command{
		Name: "prepare-proxy-files",
	}
}

// ProxyParams returns parameters for kube-proxy.
func ProxyParams() cke.ServiceParams {
	args := []string{
		"kube-proxy",
		fmt.Sprintf("--config=%s", proxyConfigPath),
	}
	return cke.ServiceParams{
		ExtraArguments: args,
		ExtraBinds: []cke.Mount{
			{
				Source:      "/etc/machine-id",
				Destination: "/etc/machine-id",
				ReadOnly:    true,
				Propagation: "",
				Label:       "",
			},
			{
				Source:      "/etc/kubernetes",
				Destination: "/etc/kubernetes",
				ReadOnly:    true,
				Propagation: "",
				Label:       cke.LabelShared,
			},
			{
				Source:      "/lib/modules",
				Destination: "/lib/modules",
				ReadOnly:    true,
				Propagation: "",
				Label:       "",
			},
		},
	}
}

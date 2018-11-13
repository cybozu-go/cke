package op

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/common"
	"github.com/cybozu-go/well"
	yaml "gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/retry"
)

type kubeletBootOp struct {
	nodes []*cke.Node

	registeredNodes []*cke.Node
	apiServer       *cke.Node

	cluster   string
	podSubnet string
	params    cke.KubeletParams

	step  int
	files *common.FilesBuilder
}

// KubeletBootOp returns an Operator to boot kubelet.
func KubeletBootOp(nodes, registeredNodes []*cke.Node, apiServer *cke.Node, cluster, podSubnet string, params cke.KubeletParams) cke.Operator {
	return &kubeletBootOp{
		nodes:           nodes,
		registeredNodes: registeredNodes,
		apiServer:       apiServer,
		cluster:         cluster,
		podSubnet:       podSubnet,
		params:          params,
		files:           common.NewFilesBuilder(nodes),
	}
}

func (o *kubeletBootOp) Name() string {
	return "kubelet-bootstrap"
}

func (o *kubeletBootOp) NextCommand() cke.Commander {
	switch o.step {
	case 0:
		o.step++
		return common.ImagePullCommand(o.nodes, cke.HyperkubeImage)
	case 1:
		o.step++
		return common.ImagePullCommand(o.nodes, cke.PauseImage)
	case 2:
		o.step++
		dirs := []string{
			cniBinDir,
			cniConfDir,
			cniVarDir,
			"/var/log/kubernetes/kubelet",
			"/var/log/pods",
			"/var/log/containers",
			"/opt/volume/bin",
		}
		return common.MakeDirsCommand(o.nodes, dirs)
	case 3:
		o.step++
		return prepareKubeletFilesCommand{o.cluster, o.podSubnet, o.params, o.files}
	case 4:
		o.step++
		return o.files
	case 5:
		o.step++
		return installCNICommand{o.nodes}
	case 6:
		o.step++
		if len(o.registeredNodes) > 0 && len(o.params.BootTaints) > 0 {
			return retaintBeforeKubeletBootCommand{o.registeredNodes, o.apiServer, o.params}
		}
		fallthrough
	case 7:
		o.step++
		return common.VolumeCreateCommand(o.nodes, "dockershim")
	case 8:
		o.step++
		opts := []string{
			"--pid=host",
			"--mount=type=volume,src=dockershim,dst=/var/lib/dockershim",
			"--privileged",
		}
		paramsMap := make(map[string]cke.ServiceParams)
		for _, n := range o.nodes {
			params := KubeletServiceParams(n)
			if len(o.params.BootTaints) > 0 {
				argl := make([]string, len(o.params.BootTaints))
				for i, t := range o.params.BootTaints {
					argl[i] = fmt.Sprintf("%s=%s:%s", t.Key, t.Value, t.Effect)
				}
				params.ExtraArguments = append(params.ExtraArguments,
					"--register-with-taints="+strings.Join(argl, ","))
			}
			paramsMap[n.Address] = params
		}
		return common.RunContainerCommand(o.nodes, kubeletContainerName, cke.HyperkubeImage,
			common.WithOpts(opts),
			common.WithParamsMap(paramsMap),
			common.WithExtra(o.params.ServiceParams))
	default:
		return nil
	}
}

type prepareKubeletFilesCommand struct {
	cluster   string
	podSubnet string
	params    cke.KubeletParams
	files     *common.FilesBuilder
}

func (c prepareKubeletFilesCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	const kubeletConfigPath = "/etc/kubernetes/kubelet/config.yml"
	const kubeconfigPath = "/etc/kubernetes/kubelet/kubeconfig"
	caPath := K8sPKIPath("ca.crt")
	tlsCertPath := K8sPKIPath("kubelet.crt")
	tlsKeyPath := K8sPKIPath("kubelet.key")
	storage := inf.Storage()

	bridgeConfData := []byte(cniBridgeConfig(c.podSubnet))
	g := func(ctx context.Context, n *cke.Node) ([]byte, error) {
		return bridgeConfData, nil
	}
	err := c.files.AddFile(ctx, filepath.Join(cniConfDir, "98-bridge.conf"), g)
	if err != nil {
		return err
	}

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
		RuntimeRequestTimeout: "15m",
		FailSwapOn:            !c.params.AllowSwap,
	}
	g = func(ctx context.Context, n *cke.Node) ([]byte, error) {
		cfg.ClusterDNS = []string{n.Address}
		return yaml.Marshal(cfg)
	}
	err = c.files.AddFile(ctx, kubeletConfigPath, g)
	if err != nil {
		return err
	}

	ca, err := storage.GetCACertificate(ctx, "kubernetes")
	if err != nil {
		return err
	}
	caData := []byte(ca)
	g = func(ctx context.Context, n *cke.Node) ([]byte, error) {
		return caData, nil
	}
	err = c.files.AddFile(ctx, caPath, g)
	if err != nil {
		return err
	}

	f := func(ctx context.Context, n *cke.Node) (cert, key []byte, err error) {
		c, k, e := cke.KubernetesCA{}.IssueForKubelet(ctx, inf, n)
		if e != nil {
			return nil, nil, e
		}
		return []byte(c), []byte(k), nil
	}
	err = c.files.AddKeyPair(ctx, K8sPKIPath("kubelet"), f)
	if err != nil {
		return err
	}

	g = func(ctx context.Context, n *cke.Node) ([]byte, error) {
		cfg := kubeletKubeconfig(c.cluster, n, caPath, tlsCertPath, tlsKeyPath)
		return clientcmd.Write(*cfg)
	}
	return c.files.AddFile(ctx, kubeconfigPath, g)
}

func (c prepareKubeletFilesCommand) Command() cke.Command {
	return cke.Command{
		Name: "prepare-kubelet-files",
	}
}

type installCNICommand struct {
	nodes []*cke.Node
}

func (c installCNICommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	env := well.NewEnvironment(ctx)

	binds := []cke.Mount{
		{Source: cniBinDir, Destination: "/host/bin", ReadOnly: false, Label: cke.LabelShared},
		{Source: cniConfDir, Destination: "/host/net.d", ReadOnly: false, Label: cke.LabelShared},
	}
	for _, n := range c.nodes {
		n := n
		ce := inf.Engine(n.Address)
		env.Go(func(ctx context.Context) error {
			return ce.Run(cke.ToolsImage, binds, "/usr/local/cke-tools/bin/install-cni")
		})
	}
	env.Stop()
	return env.Wait()
}

func (c installCNICommand) Command() cke.Command {
	targets := make([]string, len(c.nodes))
	for i, n := range c.nodes {
		targets[i] = n.Address
	}
	return cke.Command{
		Name:   "install-cni",
		Target: strings.Join(targets, ","),
	}
}

type retaintBeforeKubeletBootCommand struct {
	nodes     []*cke.Node
	apiServer *cke.Node
	params    cke.KubeletParams
}

func (c retaintBeforeKubeletBootCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	cs, err := inf.K8sClient(ctx, c.apiServer)
	if err != nil {
		return err
	}

	nodesAPI := cs.CoreV1().Nodes()
	for _, n := range c.nodes {
		err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			node, err := nodesAPI.Get(n.Nodename(), metav1.GetOptions{})
			if err != nil {
				return err
			}

			needUpdate := false
		OUTER:
			for _, bootTaint := range c.params.BootTaints {
				// append bootTaint except if the same taint is already there
				for _, nodeTaint := range node.Spec.Taints {
					if nodeTaint.Key == bootTaint.Key && nodeTaint.Value == bootTaint.Value && nodeTaint.Effect == bootTaint.Effect {
						continue OUTER
					}
				}
				node.Spec.Taints = append(node.Spec.Taints, bootTaint)
				needUpdate = true
			}
			if !needUpdate {
				return nil
			}

			_, err = nodesAPI.Update(node)
			return err
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (c retaintBeforeKubeletBootCommand) Command() cke.Command {
	targets := make([]string, len(c.nodes))
	for i, n := range c.nodes {
		targets[i] = n.Address
	}
	return cke.Command{
		Name:   "retaint-before-kubelet-boot",
		Target: strings.Join(targets, ","),
	}
}

// KubeletServiceParams returns parameters for kubelet.
func KubeletServiceParams(n *cke.Node) cke.ServiceParams {
	args := []string{
		"kubelet",
		"--config=/etc/kubernetes/kubelet/config.yml",
		"--kubeconfig=/etc/kubernetes/kubelet/kubeconfig",
		"--allow-privileged=true",
		"--hostname-override=" + n.Nodename(),
		"--pod-infra-container-image=" + cke.PauseImage.Name(),
		"--log-dir=/var/log/kubernetes/kubelet",
		"--logtostderr=false",
		"--network-plugin=cni",
		"--volume-plugin-dir=/opt/volume/bin",
	}
	return cke.ServiceParams{
		ExtraArguments: args,
		ExtraBinds: []cke.Mount{
			{"/etc/machine-id", "/etc/machine-id", true, "", ""},
			{"/etc/kubernetes", "/etc/kubernetes", true, "", cke.LabelShared},
			{"/var/lib/kubelet", "/var/lib/kubelet", false, cke.PropagationShared, cke.LabelShared},
			// TODO: /var/lib/docker is used by cAdvisor.
			// cAdvisor will be removed from kubelet. Then remove this bind mount.
			{"/var/lib/docker", "/var/lib/docker", false, "", cke.LabelPrivate},
			{"/opt/volume/bin", "/opt/volume/bin", false, cke.PropagationShared, cke.LabelShared},
			{"/var/log/pods", "/var/log/pods", false, "", cke.LabelShared},
			{"/var/log/containers", "/var/log/containers", false, "", cke.LabelShared},
			{"/var/log/kubernetes/kubelet", "/var/log/kubernetes/kubelet", false, "", cke.LabelPrivate},
			{"/run", "/run", false, "", ""},
			{"/sys", "/sys", true, "", ""},
			{"/dev", "/dev", false, "", ""},
			{cniBinDir, cniBinDir, true, "", cke.LabelShared},
			{cniConfDir, cniConfDir, true, "", cke.LabelShared},
			{cniVarDir, cniVarDir, false, "", cke.LabelShared},
		},
	}
}

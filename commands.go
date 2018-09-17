package cke

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/cybozu-go/cmd"
	yaml "gopkg.in/yaml.v2"
	"k8s.io/client-go/tools/clientcmd"
)

// Command represents some command
type Command struct {
	Name   string `json:"name"`
	Target string `json:"target"`
	Detail string `json:"detail"`
}

// String implements fmt.Stringer
func (c Command) String() string {
	if len(c.Detail) > 0 {
		return fmt.Sprintf("%s %s: %s", c.Name, c.Target, c.Detail)
	}
	return fmt.Sprintf("%s %s", c.Name, c.Target)
}

// Commander is a single step to proceed an operation
type Commander interface {
	// Run executes the command
	Run(ctx context.Context, inf Infrastructure) error
	// Command returns the command information
	Command() Command
}

type makeDirCommand struct {
	nodes     []*Node
	targetDir string
}

func (c makeDirCommand) Run(ctx context.Context, inf Infrastructure) error {
	env := cmd.NewEnvironment(ctx)
	binds := []Mount{{
		Source:      filepath.Dir(c.targetDir),
		Destination: filepath.Join("/mnt", filepath.Dir(c.targetDir)),
	}}
	mkdirCommand := "mkdir -p " + filepath.Join("/mnt", c.targetDir)
	for _, n := range c.nodes {
		ce := Docker(inf.Agent(n.Address))
		env.Go(func(ctx context.Context) error {
			return ce.Run("tools", binds, mkdirCommand)
		})
	}
	env.Stop()
	return env.Wait()
}

func (c makeDirCommand) Command() Command {
	return Command{
		Name:   "mkdir",
		Target: c.targetDir,
	}
}

type makeFileCommand struct {
	nodes  []*Node
	source string
	target string
}

func (c makeFileCommand) Run(ctx context.Context, inf Infrastructure) error {
	env := cmd.NewEnvironment(ctx)
	targetDir := filepath.Dir(c.target)
	binds := []Mount{{
		Source:      targetDir,
		Destination: filepath.Join("/mnt", targetDir),
	}}
	mkdirCommand := "mkdir -p " + filepath.Join("/mnt", targetDir)
	ddCommand := "dd of=" + filepath.Join("/mnt", c.target)
	for _, n := range c.nodes {
		ce := Docker(inf.Agent(n.Address))
		env.Go(func(ctx context.Context) error {
			err := ce.Run("tools", binds, mkdirCommand)
			if err != nil {
				return err
			}
			return ce.RunWithInput("tools", binds, ddCommand, c.source)
		})
	}
	env.Stop()
	return env.Wait()
}

func (c makeFileCommand) Command() Command {
	return Command{
		Name:   "make-file",
		Target: c.target,
	}
}

type removeFileCommand struct {
	nodes  []*Node
	target string
}

func (c removeFileCommand) Run(ctx context.Context, inf Infrastructure) error {
	env := cmd.NewEnvironment(ctx)
	dir := filepath.Dir(c.target)
	binds := []Mount{{
		Source:      dir,
		Destination: filepath.Join("/mnt", dir),
	}}
	command := "rm -f " + filepath.Join("/mnt", c.target)
	for _, n := range c.nodes {
		ce := Docker(inf.Agent(n.Address))
		env.Go(func(ctx context.Context) error {
			return ce.Run("tools", binds, command)
		})
	}
	env.Stop()
	return env.Wait()
}

func (c removeFileCommand) Command() Command {
	return Command{
		Name:   "rm",
		Target: c.target,
	}
}

type imagePullCommand struct {
	nodes []*Node
	name  string
}

func (c imagePullCommand) Run(ctx context.Context, inf Infrastructure) error {
	env := cmd.NewEnvironment(ctx)
	for _, n := range c.nodes {
		ce := Docker(inf.Agent(n.Address))
		env.Go(func(ctx context.Context) error {
			return ce.PullImage(c.name)
		})
	}
	env.Stop()
	return env.Wait()
}

func (c imagePullCommand) Command() Command {
	return Command{
		Name:   "image-pull",
		Target: c.name,
		Detail: Image(c.name),
	}
}

type volumeCreateCommand struct {
	nodes   []*Node
	volname string
}

func (c volumeCreateCommand) Run(ctx context.Context, inf Infrastructure) error {
	env := cmd.NewEnvironment(ctx)
	for _, n := range c.nodes {
		ce := Docker(inf.Agent(n.Address))
		env.Go(func(ctx context.Context) error {
			return ce.VolumeCreate(c.volname)
		})
	}
	env.Stop()
	return env.Wait()
}

func (c volumeCreateCommand) Command() Command {
	targets := make([]string, len(c.nodes))
	for i, n := range c.nodes {
		targets[i] = n.Address
	}
	return Command{
		Name:   "volume-create",
		Target: strings.Join(targets, ","),
		Detail: c.volname,
	}
}

type volumeRemoveCommand struct {
	nodes   []*Node
	volname string
}

func (c volumeRemoveCommand) Run(ctx context.Context, inf Infrastructure) error {
	env := cmd.NewEnvironment(ctx)
	for _, n := range c.nodes {
		ce := Docker(inf.Agent(n.Address))
		env.Go(func(ctx context.Context) error {
			exists, err := ce.VolumeExists(c.volname)
			if err != nil {
				return err
			}
			if exists {
				return ce.VolumeRemove(c.volname)
			}
			return nil
		})
	}
	env.Stop()
	return env.Wait()
}

func (c volumeRemoveCommand) Command() Command {
	targets := make([]string, len(c.nodes))
	for i, n := range c.nodes {
		targets[i] = n.Address
	}
	return Command{
		Name:   "volume-remove",
		Target: strings.Join(targets, ","),
		Detail: c.volname,
	}
}

type runContainerCommand struct {
	nodes  []*Node
	name   string
	opts   []string
	params ServiceParams
	extra  ServiceParams
}

func (c runContainerCommand) Run(ctx context.Context, inf Infrastructure) error {
	env := cmd.NewEnvironment(ctx)
	for _, n := range c.nodes {
		ce := Docker(inf.Agent(n.Address))
		env.Go(func(ctx context.Context) error {
			return ce.RunSystem(c.name, c.opts, c.params, c.extra)
		})
	}
	env.Stop()
	return env.Wait()
}

func (c runContainerCommand) Command() Command {
	targets := make([]string, len(c.nodes))
	for i, n := range c.nodes {
		targets[i] = n.Address
	}
	return Command{
		Name:   "run-container",
		Target: strings.Join(targets, ","),
		Detail: c.name,
	}
}

type runContainerParamsCommand struct {
	nodes  []*Node
	name   string
	opts   []string
	params map[string]ServiceParams
	extra  ServiceParams
}

func (c runContainerParamsCommand) Run(ctx context.Context, inf Infrastructure) error {
	env := cmd.NewEnvironment(ctx)
	for _, n := range c.nodes {
		ce := Docker(inf.Agent(n.Address))
		params := c.params[n.Address]
		env.Go(func(ctx context.Context) error {
			return ce.RunSystem(c.name, c.opts, params, c.extra)
		})
	}
	env.Stop()
	return env.Wait()
}

func (c runContainerParamsCommand) Command() Command {
	targets := make([]string, len(c.nodes))
	for i, n := range c.nodes {
		targets[i] = n.Address
	}
	return Command{
		Name:   "run-container",
		Target: strings.Join(targets, ","),
		Detail: c.name,
	}
}

type stopContainerCommand struct {
	node *Node
	name string
}

func (c stopContainerCommand) Run(ctx context.Context, inf Infrastructure) error {
	ce := Docker(inf.Agent(c.node.Address))
	exists, err := ce.Exists(c.name)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	err = ce.Stop(c.name)
	if err != nil {
		return err
	}
	// gofail: var dockerAfterContainerStop struct{}
	return ce.Remove(c.name)
}

func (c stopContainerCommand) Command() Command {
	return Command{
		Name:   "stop-container",
		Target: c.node.Address,
		Detail: c.name,
	}
}

type stopContainersCommand struct {
	nodes []*Node
	name  string
}

func (c stopContainersCommand) Run(ctx context.Context, inf Infrastructure) error {

	env := cmd.NewEnvironment(ctx)
	for _, n := range c.nodes {
		ce := Docker(inf.Agent(n.Address))
		env.Go(func(ctx context.Context) error {
			exists, err := ce.Exists(c.name)
			if err != nil {
				return err
			}
			if !exists {
				return nil
			}
			err = ce.Stop(c.name)
			if err != nil {
				return err
			}
			return ce.Remove(c.name)
		})
	}
	env.Stop()
	return env.Wait()
}

func (c stopContainersCommand) Command() Command {
	addrs := make([]string, len(c.nodes))
	for i, n := range c.nodes {
		addrs[i] = n.Address
	}
	return Command{
		Name:   "stop-containers",
		Target: strings.Join(addrs, ","),
		Detail: c.name,
	}
}

type setupEtcdCertificatesCommand struct {
	nodes []*Node
}

func (c setupEtcdCertificatesCommand) Run(ctx context.Context, inf Infrastructure) error {
	env := cmd.NewEnvironment(ctx)
	for _, node := range c.nodes {
		n := node
		env.Go(func(ctx context.Context) error {
			return EtcdCA{}.setupNode(ctx, inf, n)
		})
	}
	env.Stop()
	return env.Wait()
}

func (c setupEtcdCertificatesCommand) Command() Command {
	targets := make([]string, len(c.nodes))
	for i, n := range c.nodes {
		targets[i] = n.Address
	}
	return Command{
		Name:   "setup-etcd-certificates",
		Target: strings.Join(targets, ","),
	}
}

type issueAPIServerCertificatesCommand struct {
	nodes []*Node
}

func (c issueAPIServerCertificatesCommand) Run(ctx context.Context, inf Infrastructure) error {
	env := cmd.NewEnvironment(ctx)
	for _, node := range c.nodes {
		n := node
		env.Go(func(ctx context.Context) error {
			return EtcdCA{}.issueForAPIServer(ctx, inf, n)
		})
	}
	env.Stop()
	return env.Wait()
}

func (c issueAPIServerCertificatesCommand) Command() Command {
	targets := make([]string, len(c.nodes))
	for i, n := range c.nodes {
		targets[i] = n.Address
	}
	return Command{
		Name:   "issue-apiserver-certificates",
		Target: strings.Join(targets, ","),
	}
}

type setupAPIServerCertificatesCommand struct {
	nodes []*Node
}

func (c setupAPIServerCertificatesCommand) Run(ctx context.Context, inf Infrastructure) error {
	env := cmd.NewEnvironment(ctx)
	for _, node := range c.nodes {
		n := node
		env.Go(func(ctx context.Context) error {
			return KubernetesCA{}.setup(ctx, inf, n)
		})
	}
	env.Stop()
	return env.Wait()
}

func (c setupAPIServerCertificatesCommand) Command() Command {
	targets := make([]string, len(c.nodes))
	for i, n := range c.nodes {
		targets[i] = n.Address
	}
	return Command{
		Name:   "setup-apiserver-certificates",
		Target: strings.Join(targets, ","),
	}
}

type makeControllerManagerKubeconfigCommand struct {
	nodes   []*Node
	cluster string
}

func (c makeControllerManagerKubeconfigCommand) Run(ctx context.Context, inf Infrastructure) error {
	const path = "/etc/kubernetes/controller-manager/kubeconfig"

	ca, err := inf.Storage().GetCACertificate(ctx, "kubernetes")
	if err != nil {
		return err
	}
	crt, key, err := KubernetesCA{}.issueForControllerManager(ctx, inf)
	if err != nil {
		return err
	}
	cfg := controllerManagerKubeconfig(c.cluster, ca, crt, key)
	src, err := clientcmd.Write(*cfg)
	if err != nil {
		return err
	}
	return makeFileCommand{c.nodes, string(src), path}.Run(ctx, inf)
}

func (c makeControllerManagerKubeconfigCommand) Command() Command {
	targets := make([]string, len(c.nodes))
	for i, n := range c.nodes {
		targets[i] = n.Address
	}
	return Command{
		Name:   "make-controller-manager-kubeconfig",
		Target: strings.Join(targets, ","),
	}
}

type makeSchedulerKubeconfigCommand struct {
	nodes   []*Node
	cluster string
}

func (c makeSchedulerKubeconfigCommand) Run(ctx context.Context, inf Infrastructure) error {
	const path = "/etc/kubernetes/scheduler/kubeconfig"

	ca, err := inf.Storage().GetCACertificate(ctx, "kubernetes")
	if err != nil {
		return err
	}
	crt, key, err := KubernetesCA{}.issueForScheduler(ctx, inf)
	if err != nil {
		return err
	}
	cfg := schedulerKubeconfig(c.cluster, ca, crt, key)
	src, err := clientcmd.Write(*cfg)
	if err != nil {
		return err
	}
	return makeFileCommand{c.nodes, string(src), path}.Run(ctx, inf)
}

func (c makeSchedulerKubeconfigCommand) Command() Command {
	targets := make([]string, len(c.nodes))
	for i, n := range c.nodes {
		targets[i] = n.Address
	}
	return Command{
		Name:   "make-scheduler-kubeconfig",
		Target: strings.Join(targets, ","),
	}
}

type makeProxyKubeconfigCommand struct {
	nodes   []*Node
	cluster string
}

func (c makeProxyKubeconfigCommand) Run(ctx context.Context, inf Infrastructure) error {
	const path = "/etc/kubernetes/proxy/kubeconfig"

	ca, err := inf.Storage().GetCACertificate(ctx, "kubernetes")
	if err != nil {
		return err
	}
	crt, key, err := KubernetesCA{}.issueForProxy(ctx, inf)
	if err != nil {
		return err
	}
	cfg := proxyKubeconfig(c.cluster, ca, crt, key)
	src, err := clientcmd.Write(*cfg)
	if err != nil {
		return err
	}
	return makeFileCommand{c.nodes, string(src), path}.Run(ctx, inf)
}

func (c makeProxyKubeconfigCommand) Command() Command {
	targets := make([]string, len(c.nodes))
	for i, n := range c.nodes {
		targets[i] = n.Address
	}
	return Command{
		Name:   "make-proxy-kubeconfig",
		Target: strings.Join(targets, ","),
	}
}

type makeKubeletKubeconfigCommand struct {
	nodes   []*Node
	cluster string
	params  KubeletParams
}

func (c makeKubeletKubeconfigCommand) Run(ctx context.Context, inf Infrastructure) error {
	const kubeletConfigPath = "/etc/kubernetes/kubelet/config.yml"
	const kubeconfigPath = "/etc/kubernetes/kubelet/kubeconfig"
	caPath := K8sPKIPath("ca.crt")
	tlsCertPath := K8sPKIPath("kubelet.crt")
	tlsKeyPath := K8sPKIPath("kubelet.key")

	ca, err := inf.Storage().GetCACertificate(ctx, "kubernetes")
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
	cfgData, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	env := cmd.NewEnvironment(ctx)
	for _, n := range c.nodes {
		n := n

		env.Go(func(ctx context.Context) error {
			err := writeFile(inf, n, caPath, ca)
			if err != nil {
				return err
			}

			crt, key, err := KubernetesCA{}.issueForKubelet(ctx, inf, n)
			if err != nil {
				return err
			}
			cfg := kubeletKubeconfig(c.cluster, n, ca, crt, key)
			kubeconfig, err := clientcmd.Write(*cfg)
			if err != nil {
				return err
			}
			err = writeFile(inf, n, kubeconfigPath, string(kubeconfig))
			if err != nil {
				return err
			}

			err = writeFile(inf, n, tlsCertPath, crt)
			if err != nil {
				return err
			}
			err = writeFile(inf, n, tlsKeyPath, key)
			if err != nil {
				return err
			}
			return writeFile(inf, n, kubeletConfigPath, string(cfgData))
		})
	}
	env.Stop()
	return env.Wait()
}

func (c makeKubeletKubeconfigCommand) Command() Command {
	targets := make([]string, len(c.nodes))
	for i, n := range c.nodes {
		targets[i] = n.Address
	}
	return Command{
		Name:   "make-kubelet-kubeconfig",
		Target: strings.Join(targets, ","),
	}
}

type killContainersCommand struct {
	nodes []*Node
	name  string
}

func (c killContainersCommand) Run(ctx context.Context, inf Infrastructure) error {
	env := cmd.NewEnvironment(ctx)
	for _, n := range c.nodes {
		ce := Docker(inf.Agent(n.Address))
		env.Go(func(ctx context.Context) error {
			exists, err := ce.Exists(c.name)
			if err != nil {
				return err
			}
			if !exists {
				return nil
			}
			err = ce.Kill(c.name)
			if err != nil {
				return err
			}
			return ce.Remove(c.name)
		})
	}
	env.Stop()
	return env.Wait()
}

func (c killContainersCommand) Command() Command {
	addrs := make([]string, len(c.nodes))
	for i, n := range c.nodes {
		addrs[i] = n.Address
	}
	return Command{
		Name:   "kill-containers",
		Target: strings.Join(addrs, ","),
		Detail: c.name,
	}
}

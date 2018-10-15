package cke

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cybozu-go/cmd"
	"github.com/cybozu-go/log"
	yaml "gopkg.in/yaml.v2"
	rbac "k8s.io/api/rbac/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
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

type makeDirsCommand struct {
	nodes []*Node
	dirs  []string
}

func (c makeDirsCommand) Run(ctx context.Context, inf Infrastructure) error {
	bindMap := make(map[string]Mount)
	dests := make([]string, len(c.dirs))
	for i, d := range c.dirs {
		dests[i] = filepath.Join("/mnt", d)

		parentDir := filepath.Dir(d)
		if _, ok := bindMap[parentDir]; ok {
			continue
		}
		bindMap[parentDir] = Mount{
			Source:      parentDir,
			Destination: filepath.Join("/mnt", parentDir),
			Label:       LabelPrivate,
		}
	}
	binds := make([]Mount, 0, len(bindMap))
	for _, m := range bindMap {
		binds = append(binds, m)
	}

	arg := "/usr/local/cke-tools/bin/make_directories " + strings.Join(dests, " ")

	env := cmd.NewEnvironment(ctx)
	for _, n := range c.nodes {
		ce := Docker(inf.Agent(n.Address))
		env.Go(func(ctx context.Context) error {
			return ce.Run(ToolsImage, binds, arg)
		})
	}
	env.Stop()
	return env.Wait()
}

func (c makeDirsCommand) Command() Command {
	return Command{
		Name:   "make-dirs",
		Target: strings.Join(c.dirs, " "),
	}
}

type fileData struct {
	name    string
	dataMap map[string][]byte
}

type makeFilesCommand struct {
	nodes []*Node
	files []fileData
}

func (c *makeFilesCommand) AddFile(ctx context.Context, name string,
	f func(context.Context, *Node) ([]byte, error)) error {
	var mu sync.Mutex
	dataMap := make(map[string][]byte)

	env := cmd.NewEnvironment(ctx)
	for _, n := range c.nodes {
		n := n
		env.Go(func(ctx context.Context) error {
			data, err := f(ctx, n)
			if err != nil {
				return err
			}
			mu.Lock()
			dataMap[n.Address] = data
			mu.Unlock()
			return nil
		})
	}
	env.Stop()
	err := env.Wait()
	if err != nil {
		return err
	}

	c.files = append(c.files, fileData{name, dataMap})
	return nil
}

func (c *makeFilesCommand) AddKeyPair(ctx context.Context, name string,
	f func(context.Context, *Node) (cert, key []byte, err error)) error {
	var mu sync.Mutex
	certMap := make(map[string][]byte)
	keyMap := make(map[string][]byte)

	env := cmd.NewEnvironment(ctx)
	for _, n := range c.nodes {
		n := n
		env.Go(func(ctx context.Context) error {
			certData, keyData, err := f(ctx, n)
			if err != nil {
				return err
			}
			mu.Lock()
			certMap[n.Address] = certData
			keyMap[n.Address] = keyData
			mu.Unlock()
			return nil
		})
	}
	env.Stop()
	err := env.Wait()
	if err != nil {
		return err
	}

	c.files = append(c.files, fileData{name + ".crt", certMap})
	c.files = append(c.files, fileData{name + ".key", keyMap})
	return nil
}

func (c *makeFilesCommand) Run(ctx context.Context, inf Infrastructure) error {
	bindMap := make(map[string]Mount)
	for _, f := range c.files {
		parentDir := filepath.Dir(f.name)
		if _, ok := bindMap[parentDir]; ok {
			continue
		}
		bindMap[parentDir] = Mount{
			Source:      parentDir,
			Destination: filepath.Join("/mnt", parentDir),
			Label:       LabelPrivate,
		}
	}
	binds := make([]Mount, 0, len(bindMap))
	for _, m := range bindMap {
		binds = append(binds, m)
	}

	env := cmd.NewEnvironment(ctx)
	for _, n := range c.nodes {
		n := n
		env.Go(func(ctx context.Context) error {
			buf := new(bytes.Buffer)
			tw := tar.NewWriter(buf)
			for _, f := range c.files {
				data := f.dataMap[n.Address]
				hdr := &tar.Header{
					Name: f.name,
					Mode: 0644,
					Size: int64(len(data)),
				}
				if err := tw.WriteHeader(hdr); err != nil {
					return err
				}
				if _, err := tw.Write(data); err != nil {
					return err
				}
			}
			if err := tw.Close(); err != nil {
				return err
			}
			data := buf.String()

			arg := "/usr/local/cke-tools/bin/write_files /mnt"
			ce := Docker(inf.Agent(n.Address))
			return ce.RunWithInput(ToolsImage, binds, arg, data)
		})
	}
	env.Stop()
	return env.Wait()
}

func (c *makeFilesCommand) Command() Command {
	fileNames := make([]string, len(c.files))
	for i, f := range c.files {
		fileNames[i] = f.name
	}
	return Command{
		Name:   "make-files",
		Target: strings.Join(fileNames, ","),
	}
}

type imagePullCommand struct {
	nodes []*Node
	img   Image
}

func (c imagePullCommand) Run(ctx context.Context, inf Infrastructure) error {
	env := cmd.NewEnvironment(ctx)
	for _, n := range c.nodes {
		ce := Docker(inf.Agent(n.Address))
		env.Go(func(ctx context.Context) error {
			return ce.PullImage(c.img)
		})
	}
	env.Stop()
	return env.Wait()
}

func (c imagePullCommand) Command() Command {
	return Command{
		Name:   "image-pull",
		Target: c.img.Name(),
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
	nodes     []*Node
	name      string
	img       Image
	opts      []string
	optsMap   map[string][]string
	params    ServiceParams
	paramsMap map[string]ServiceParams
	extra     ServiceParams

	restart bool
}

func (c runContainerCommand) Run(ctx context.Context, inf Infrastructure) error {
	env := cmd.NewEnvironment(ctx)
	for _, n := range c.nodes {
		n := n
		ce := Docker(inf.Agent(n.Address))
		env.Go(func(ctx context.Context) error {
			params, ok := c.paramsMap[n.Address]
			if !ok {
				params = c.params
			}
			opts, ok := c.optsMap[n.Address]
			if !ok {
				opts = c.opts
			}
			if c.restart {
				err := ce.Kill(c.name)
				if err != nil {
					return err
				}
			}
			exists, err := ce.Exists(c.name)
			if err != nil {
				return err
			}
			if exists {
				err = ce.Remove(c.name)
				if err != nil {
					return err
				}
			}
			return ce.RunSystem(c.name, c.img, opts, params, c.extra)
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

type stopContainerCommand struct {
	node *Node
	name string
}

func (c stopContainerCommand) Run(ctx context.Context, inf Infrastructure) error {
	begin := time.Now()
	ce := Docker(inf.Agent(c.node.Address))
	exists, err := ce.Exists(c.name)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	// Inspect returns ServiceStatus for the named container.
	statuses, err := ce.Inspect([]string{c.name})
	if err != nil {
		return err
	}
	if st, ok := statuses[c.name]; ok && st.Running {
		err = ce.Stop(c.name)
		if err != nil {
			return err
		}
	}
	err = ce.Remove(c.name)
	log.Info("stop container", map[string]interface{}{
		"container": c.name,
		"elapsed":   time.Now().Sub(begin).Seconds(),
	})
	return err
}

func (c stopContainerCommand) Command() Command {
	return Command{
		Name:   "stop-container",
		Target: c.node.Address,
		Detail: c.name,
	}
}

type prepareEtcdCertificatesCommand struct {
	makeFiles *makeFilesCommand
}

func (c prepareEtcdCertificatesCommand) Run(ctx context.Context, inf Infrastructure) error {
	f := func(ctx context.Context, n *Node) (cert, key []byte, err error) {
		c, k, e := EtcdCA{}.issueServerCert(ctx, inf, n)
		if e != nil {
			return nil, nil, e
		}
		return []byte(c), []byte(k), nil
	}
	err := c.makeFiles.AddKeyPair(ctx, EtcdPKIPath("server"), f)
	if err != nil {
		return err
	}

	f = func(ctx context.Context, n *Node) (cert, key []byte, err error) {
		c, k, e := EtcdCA{}.issuePeerCert(ctx, inf, n)
		if e != nil {
			return nil, nil, e
		}
		return []byte(c), []byte(k), nil
	}
	err = c.makeFiles.AddKeyPair(ctx, EtcdPKIPath("peer"), f)
	if err != nil {
		return err
	}

	peerCA, err := inf.Storage().GetCACertificate(ctx, "etcd-peer")
	if err != nil {
		return err
	}
	f2 := func(ctx context.Context, node *Node) ([]byte, error) {
		return []byte(peerCA), nil
	}
	err = c.makeFiles.AddFile(ctx, EtcdPKIPath("ca-peer.crt"), f2)
	if err != nil {
		return err
	}

	clientCA, err := inf.Storage().GetCACertificate(ctx, "etcd-client")
	if err != nil {
		return err
	}
	f2 = func(ctx context.Context, node *Node) ([]byte, error) {
		return []byte(clientCA), nil
	}
	err = c.makeFiles.AddFile(ctx, EtcdPKIPath("ca-client.crt"), f2)
	if err != nil {
		return err
	}
	return nil
}

func (c prepareEtcdCertificatesCommand) Command() Command {
	return Command{
		Name: "prepare-etcd-certificates",
	}
}

type prepareAPIServerFilesCommand struct {
	makeFiles     *makeFilesCommand
	serviceSubnet string
	domain        string
}

func (c prepareAPIServerFilesCommand) Run(ctx context.Context, inf Infrastructure) error {
	storage := inf.Storage()

	// server (and client) certs of API server.
	f := func(ctx context.Context, n *Node) (cert, key []byte, err error) {
		c, k, e := KubernetesCA{}.issueForAPIServer(ctx, inf, n, c.serviceSubnet, c.domain)
		if e != nil {
			return nil, nil, e
		}
		return []byte(c), []byte(k), nil
	}
	err := c.makeFiles.AddKeyPair(ctx, K8sPKIPath("apiserver"), f)
	if err != nil {
		return err
	}

	// client certs for etcd auth.
	f = func(ctx context.Context, n *Node) (cert, key []byte, err error) {
		c, k, e := EtcdCA{}.issueForAPIServer(ctx, inf, n)
		if e != nil {
			return nil, nil, e
		}
		return []byte(c), []byte(k), nil
	}
	err = c.makeFiles.AddKeyPair(ctx, K8sPKIPath("apiserver-etcd-client"), f)
	if err != nil {
		return err
	}

	// CA of k8s cluster.
	ca, err := storage.GetCACertificate(ctx, "kubernetes")
	if err != nil {
		return err
	}
	caData := []byte(ca)
	g := func(ctx context.Context, n *Node) ([]byte, error) {
		return caData, nil
	}
	err = c.makeFiles.AddFile(ctx, K8sPKIPath("ca.crt"), g)
	if err != nil {
		return err
	}

	// CA of etcd server.
	etcdCA, err := storage.GetCACertificate(ctx, "server")
	if err != nil {
		return err
	}
	etcdCAData := []byte(etcdCA)
	g = func(ctx context.Context, n *Node) ([]byte, error) {
		return etcdCAData, nil
	}
	err = c.makeFiles.AddFile(ctx, K8sPKIPath("etcd-ca.crt"), g)
	if err != nil {
		return err
	}

	// ServiceAccount cert.
	saCert, err := storage.GetServiceAccountCert(ctx)
	if err != nil {
		return err
	}
	saCertData := []byte(saCert)
	g = func(ctx context.Context, n *Node) ([]byte, error) {
		return saCertData, nil
	}
	return c.makeFiles.AddFile(ctx, K8sPKIPath("service-account.crt"), g)
}

func (c prepareAPIServerFilesCommand) Command() Command {
	return Command{
		Name: "prepare-apiserver-files",
	}
}

type prepareControllerManagerFilesCommand struct {
	cluster   string
	makeFiles *makeFilesCommand
}

func (c prepareControllerManagerFilesCommand) Run(ctx context.Context, inf Infrastructure) error {
	const kubeconfigPath = "/etc/kubernetes/controller-manager/kubeconfig"
	storage := inf.Storage()

	ca, err := storage.GetCACertificate(ctx, "kubernetes")
	if err != nil {
		return err
	}
	g := func(ctx context.Context, n *Node) ([]byte, error) {
		crt, key, err := KubernetesCA{}.issueForControllerManager(ctx, inf)
		if err != nil {
			return nil, err
		}
		cfg := controllerManagerKubeconfig(c.cluster, ca, crt, key)
		return clientcmd.Write(*cfg)
	}
	err = c.makeFiles.AddFile(ctx, kubeconfigPath, g)
	if err != nil {
		return err
	}

	saKey, err := storage.GetServiceAccountKey(ctx)
	if err != nil {
		return err
	}
	saKeyData := []byte(saKey)
	g = func(ctx context.Context, n *Node) ([]byte, error) {
		return saKeyData, nil
	}
	return c.makeFiles.AddFile(ctx, K8sPKIPath("service-account.key"), g)
}

func (c prepareControllerManagerFilesCommand) Command() Command {
	return Command{
		Name: "prepare-controller-manager-files",
	}
}

type prepareSchedulerFilesCommand struct {
	cluster   string
	makeFiles *makeFilesCommand
}

func (c prepareSchedulerFilesCommand) Run(ctx context.Context, inf Infrastructure) error {
	const kubeconfigPath = "/etc/kubernetes/scheduler/kubeconfig"
	storage := inf.Storage()

	ca, err := storage.GetCACertificate(ctx, "kubernetes")
	if err != nil {
		return err
	}
	g := func(ctx context.Context, n *Node) ([]byte, error) {
		crt, key, err := KubernetesCA{}.issueForScheduler(ctx, inf)
		if err != nil {
			return nil, err
		}
		cfg := schedulerKubeconfig(c.cluster, ca, crt, key)
		return clientcmd.Write(*cfg)
	}
	return c.makeFiles.AddFile(ctx, kubeconfigPath, g)
}

func (c prepareSchedulerFilesCommand) Command() Command {
	return Command{
		Name: "prepare-scheduler-files",
	}
}

type prepareProxyFilesCommand struct {
	cluster   string
	makeFiles *makeFilesCommand
}

func (c prepareProxyFilesCommand) Run(ctx context.Context, inf Infrastructure) error {
	const kubeconfigPath = "/etc/kubernetes/proxy/kubeconfig"
	storage := inf.Storage()

	ca, err := storage.GetCACertificate(ctx, "kubernetes")
	if err != nil {
		return err
	}
	g := func(ctx context.Context, n *Node) ([]byte, error) {
		crt, key, err := KubernetesCA{}.issueForProxy(ctx, inf)
		if err != nil {
			return nil, err
		}
		cfg := proxyKubeconfig(c.cluster, ca, crt, key)
		return clientcmd.Write(*cfg)
	}
	return c.makeFiles.AddFile(ctx, kubeconfigPath, g)
}

func (c prepareProxyFilesCommand) Command() Command {
	return Command{
		Name: "prepare-proxy-files",
	}
}

type prepareKubeletFilesCommand struct {
	cluster   string
	podSubnet string
	params    KubeletParams
	makeFiles *makeFilesCommand
}

func (c prepareKubeletFilesCommand) Run(ctx context.Context, inf Infrastructure) error {
	const kubeletConfigPath = "/etc/kubernetes/kubelet/config.yml"
	const kubeconfigPath = "/etc/kubernetes/kubelet/kubeconfig"
	caPath := K8sPKIPath("ca.crt")
	tlsCertPath := K8sPKIPath("kubelet.crt")
	tlsKeyPath := K8sPKIPath("kubelet.key")
	storage := inf.Storage()

	bridgeConfData := []byte(cniBridgeConfig(c.podSubnet))
	g := func(ctx context.Context, n *Node) ([]byte, error) {
		return bridgeConfData, nil
	}
	err := c.makeFiles.AddFile(ctx, filepath.Join(cniConfDir, "98-bridge.conf"), g)
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
	g = func(ctx context.Context, n *Node) ([]byte, error) {
		return cfgData, nil
	}
	err = c.makeFiles.AddFile(ctx, kubeletConfigPath, g)
	if err != nil {
		return err
	}

	ca, err := storage.GetCACertificate(ctx, "kubernetes")
	if err != nil {
		return err
	}
	caData := []byte(ca)
	g = func(ctx context.Context, n *Node) ([]byte, error) {
		return caData, nil
	}
	err = c.makeFiles.AddFile(ctx, caPath, g)
	if err != nil {
		return err
	}

	f := func(ctx context.Context, n *Node) (cert, key []byte, err error) {
		c, k, e := KubernetesCA{}.issueForKubelet(ctx, inf, n)
		if e != nil {
			return nil, nil, e
		}
		return []byte(c), []byte(k), nil
	}
	err = c.makeFiles.AddKeyPair(ctx, K8sPKIPath("kubelet"), f)
	if err != nil {
		return err
	}

	g = func(ctx context.Context, n *Node) ([]byte, error) {
		cfg := kubeletKubeconfig(c.cluster, n, caPath, tlsCertPath, tlsKeyPath)
		return clientcmd.Write(*cfg)
	}
	return c.makeFiles.AddFile(ctx, kubeconfigPath, g)
}

func (c prepareKubeletFilesCommand) Command() Command {
	return Command{
		Name: "prepare-kubelet-files",
	}
}

type prepareKubeletConfigCommand struct {
	params    KubeletParams
	makeFiles *makeFilesCommand
}

func (c prepareKubeletConfigCommand) Run(ctx context.Context, inf Infrastructure) error {
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
		RuntimeRequestTimeout: "15m",
		FailSwapOn:            !c.params.AllowSwap,
	}
	cfgData, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	g := func(ctx context.Context, n *Node) ([]byte, error) {
		return cfgData, nil
	}
	return c.makeFiles.AddFile(ctx, kubeletConfigPath, g)
}

func (c prepareKubeletConfigCommand) Command() Command {
	return Command{
		Name: "prepare-kubelet-config",
	}
}

type installCNICommand struct {
	nodes []*Node
}

func (c installCNICommand) Run(ctx context.Context, inf Infrastructure) error {
	env := cmd.NewEnvironment(ctx)

	binds := []Mount{
		{Source: cniBinDir, Destination: "/host/bin", ReadOnly: false, Label: LabelShared},
		{Source: cniConfDir, Destination: "/host/net.d", ReadOnly: false, Label: LabelShared},
	}
	for _, n := range c.nodes {
		n := n
		ce := Docker(inf.Agent(n.Address))
		env.Go(func(ctx context.Context) error {
			return ce.Run(ToolsImage, binds, "/usr/local/cke-tools/bin/install-cni")
		})
	}
	env.Stop()
	return env.Wait()
}

func (c installCNICommand) Command() Command {
	targets := make([]string, len(c.nodes))
	for i, n := range c.nodes {
		targets[i] = n.Address
	}
	return Command{
		Name:   "install-cni",
		Target: strings.Join(targets, ","),
	}
}

type makeRBACRoleCommand struct {
	apiserver *Node
}

func (c makeRBACRoleCommand) Run(ctx context.Context, inf Infrastructure) error {
	cs, err := inf.K8sClient(ctx, c.apiserver)
	if err != nil {
		return err
	}

	_, err = cs.RbacV1().ClusterRoles().Create(&rbac.ClusterRole{
		ObjectMeta: meta.ObjectMeta{
			Name: rbacRoleName,
			Labels: map[string]string{
				"kubernetes.io/bootstrapping": "rbac-defaults",
			},
			Annotations: map[string]string{
				// turn on auto-reconciliation
				// https://kubernetes.io/docs/reference/access-authn-authz/rbac/#auto-reconciliation
				"rbac.authorization.kubernetes.io/autoupdate": "true",
			},
		},
		Rules: []rbac.PolicyRule{
			{
				APIGroups: []string{""},
				// these are virtual resources.
				// see https://github.com/kubernetes/kubernetes/issues/44330#issuecomment-293768369
				Resources: []string{
					"nodes/proxy",
					"nodes/stats",
					"nodes/log",
					"nodes/spec",
					"nodes/metrics",
				},
				Verbs: []string{"*"},
			},
		},
	})

	return err
}

func (c makeRBACRoleCommand) Command() Command {
	return Command{
		Name:   "makeClusterRole",
		Target: rbacRoleName,
	}
}

type makeRBACRoleBindingCommand struct {
	apiserver *Node
}

func (c makeRBACRoleBindingCommand) Run(ctx context.Context, inf Infrastructure) error {
	cs, err := inf.K8sClient(ctx, c.apiserver)
	if err != nil {
		return err
	}

	_, err = cs.RbacV1().ClusterRoleBindings().Create(&rbac.ClusterRoleBinding{
		ObjectMeta: meta.ObjectMeta{
			Name: rbacRoleBindingName,
			Labels: map[string]string{
				"kubernetes.io/bootstrapping": "rbac-defaults",
			},
			Annotations: map[string]string{
				// turn on auto-reconciliation
				// https://kubernetes.io/docs/reference/access-authn-authz/rbac/#auto-reconciliation
				"rbac.authorization.kubernetes.io/autoupdate": "true",
			},
		},
		RoleRef: rbac.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     rbacRoleName,
		},
		Subjects: []rbac.Subject{
			{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "User",
				Name:     "kubernetes",
			},
		},
	})

	return err
}

func (c makeRBACRoleBindingCommand) Command() Command {
	return Command{
		Name:   "makeClusterRoleBinding",
		Target: rbacRoleBindingName,
	}
}

type killContainersCommand struct {
	nodes []*Node
	name  string
}

func (c killContainersCommand) Run(ctx context.Context, inf Infrastructure) error {
	begin := time.Now()
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
			statuses, err := ce.Inspect([]string{c.name})
			if err != nil {
				return err
			}
			if st, ok := statuses[c.name]; ok && st.Running {
				err = ce.Kill(c.name)
				if err != nil {
					return err
				}
			}
			return ce.Remove(c.name)
		})
	}
	env.Stop()
	err := env.Wait()
	log.Info("kill container", map[string]interface{}{
		"container": c.name,
		"elapsed":   time.Now().Sub(begin).Seconds(),
	})
	return err
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

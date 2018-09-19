package cke

import (
	"strings"
)

const (
	kubeAPIServerContainerName         = "kube-apiserver"
	kubeControllerManagerContainerName = "kube-controller-manager"
	kubeProxyContainerName             = "kube-proxy"
	kubeSchedulerContainerName         = "kube-scheduler"
	kubeletContainerName               = "kubelet"
	pauseContainerName                 = "pause"
	riversContainerName                = "rivers"

	rbacRoleName        = "system:kube-apiserver-to-kubelet"
	rbacRoleBindingName = "system:kube-apiserver"
)

var (
	// admissionPlugins is the recommended list of admission plugins.
	// https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#is-there-a-recommended-set-of-admission-controllers-to-use
	admissionPlugins = []string{
		"NamespaceLifecycle",
		"LimitRanger",
		"ServiceAccount",
		"DefaultStorageClass",
		"DefaultTolerationSeconds",
		"MutatingAdmissionWebhook",
		"ValidatingAdmissionWebhook",
		"ResourceQuota",
	}
)

type riversBootOp struct {
	nodes     []*Node
	upstreams []*Node
	params    ServiceParams
	step      int
}

type kubeCPBootOp struct {
	cps []*Node

	apiserver         []*Node
	controllerManager []*Node
	scheduler         []*Node

	cluster       string
	serviceSubnet string
	options       Options

	step      int
	nodeIndex int
}

type kubeCPRestartOp struct {
	cps []*Node

	rivers            []*Node
	apiserver         []*Node
	controllerManager []*Node
	scheduler         []*Node

	cluster       string
	serviceSubnet string
	options       Options

	step1     int
	step2     int
	nodeIndex int
}

type kubeCPStopOp struct {
	apiserver         []*Node
	controllerManager []*Node
	scheduler         []*Node

	step int
}

type kubeWorkerBootOp struct {
	cps []*Node

	kubelets []*Node
	proxies  []*Node

	cluster string
	options Options

	step int
}

type kubeWorkerRestartOp struct {
	cps []*Node

	rivers   []*Node
	kubelets []*Node
	proxies  []*Node

	cluster string
	options Options

	step int
}

type kubeRBACRoleInstallOp struct {
	apiserver     *Node
	roleExists    bool
	bindingExists bool
}

// RiversBootOp returns an Operator to bootstrap rivers cluster.
func RiversBootOp(nodes []*Node, upstreams []*Node, params ServiceParams) Operator {
	return &riversBootOp{
		nodes:     nodes,
		upstreams: upstreams,
		params:    params,
		step:      0,
	}
}

func (o *riversBootOp) Name() string {
	return "rivers-bootstrap"
}

func (o *riversBootOp) NextCommand() Commander {
	extra := o.params
	var opts []string

	switch o.step {
	case 0:
		o.step++
		return imagePullCommand{o.nodes, ToolsImage}
	case 1:
		o.step++
		return makeDirCommand{o.nodes, "/var/log/rivers"}
	case 2:
		o.step++
		return runContainerCommand{o.nodes, "rivers", ToolsImage, opts, RiversParams(o.upstreams), extra}
	default:
		return nil
	}
}

// RiversParams returns a ServiceParams for rivers
func RiversParams(upstreams []*Node) ServiceParams {
	var ups []string
	for _, n := range upstreams {
		ups = append(ups, n.Address+":6443")
	}
	args := []string{
		"rivers",
		"--upstreams=" + strings.Join(ups, ","),
		"--listen=" + "127.0.0.1:16443",
	}
	return ServiceParams{
		ExtraArguments: args,
		ExtraBinds: []Mount{
			{"/var/log/rivers", "/var/log/rivers", false, ""},
		},
	}
}

// KubeCPBootOp returns an Operator to bootstrap kubernetes control planes
func KubeCPBootOp(cps []*Node, apiserver, controllerManager, scheduler []*Node, cluster string, serviceSubnet string, options Options) Operator {
	return &kubeCPBootOp{
		cps:               cps,
		apiserver:         apiserver,
		controllerManager: controllerManager,
		scheduler:         scheduler,
		cluster:           cluster,
		serviceSubnet:     serviceSubnet,
		options:           options,
	}
}

func (o *kubeCPBootOp) Name() string {
	return "kubernetes-control-plane-bootstrap"
}

func (o *kubeCPBootOp) NextCommand() Commander {
	var opts []string

	switch o.step {
	case 0:
		o.step++
		return imagePullCommand{o.cps, HyperkubeImage}
	case 1:
		o.step++
		if len(o.apiserver) == 0 {
			return o.NextCommand()
		}
		return makeDirCommand{o.apiserver, "/var/log/kubernetes/apiserver"}
	case 2:
		o.step++
		if len(o.controllerManager) == 0 {
			return o.NextCommand()
		}
		return makeDirCommand{o.controllerManager, "/var/log/kubernetes/controller-manager"}
	case 3:
		o.step++
		if len(o.scheduler) == 0 {
			return o.NextCommand()
		}
		return makeDirCommand{o.scheduler, "/var/log/kubernetes/scheduler"}
	case 4:
		o.step++
		if len(o.apiserver) == 0 {
			return o.NextCommand()
		}
		return issueAPIServerCertificatesCommand{o.apiserver}
	case 5:
		o.step++
		if len(o.apiserver) == 0 {
			return o.NextCommand()
		}
		return setupAPIServerCertificatesCommand{o.apiserver}
	case 6:
		o.step++
		if len(o.scheduler) == 0 {
			return o.NextCommand()
		}
		return makeControllerManagerKubeconfigCommand{o.controllerManager, o.cluster}
	case 7:
		o.step++
		if len(o.scheduler) == 0 {
			return o.NextCommand()
		}
		return makeSchedulerKubeconfigCommand{o.scheduler, o.cluster}
	case 8:
		if o.nodeIndex >= len(o.apiserver) {
			o.step++
			return o.NextCommand()
		}
		node := o.apiserver[o.nodeIndex]
		o.nodeIndex++

		opts := []string{
			// TODO pass keys from CKE
			"--mount", "type=tmpfs,dst=/run/kubernetes",
		}
		return runContainerCommand{[]*Node{node}, kubeAPIServerContainerName, HyperkubeImage,
			opts, APIServerParams(o.cps, node.Address, o.serviceSubnet), o.options.APIServer}
	case 9:
		o.step++
		if len(o.scheduler) == 0 {
			return o.NextCommand()
		}
		return runContainerCommand{o.scheduler, kubeSchedulerContainerName, HyperkubeImage,
			opts, SchedulerParams(), o.options.Scheduler}
	case 10:
		o.step++
		if len(o.controllerManager) == 0 {
			return o.NextCommand()
		}
		return runContainerCommand{o.controllerManager, kubeControllerManagerContainerName, HyperkubeImage,
			opts, ControllerManagerParams(o.cluster, o.serviceSubnet), o.options.ControllerManager}
	default:
		return nil
	}
}

// KubeCPStopOp returns an Operator to stop kubernetes control planes
func KubeCPStopOp(apiserver, controllerManager, scheduler []*Node) Operator {
	return &kubeCPStopOp{
		apiserver:         apiserver,
		controllerManager: controllerManager,
		scheduler:         scheduler,
	}
}

func (o *kubeCPStopOp) Name() string {
	return "kubernetes-control-plane-stop"
}

func (o *kubeCPStopOp) NextCommand() Commander {
	switch o.step {
	case 0:
		o.step++
		if len(o.apiserver) == 0 {
			return o.NextCommand()
		}
		return stopContainersCommand{o.apiserver, kubeAPIServerContainerName}
	case 1:
		o.step++
		if len(o.scheduler) == 0 {
			return o.NextCommand()
		}
		return stopContainersCommand{o.scheduler, kubeSchedulerContainerName}
	case 2:
		o.step++
		if len(o.controllerManager) == 0 {
			return o.NextCommand()
		}
		return stopContainersCommand{o.controllerManager, kubeControllerManagerContainerName}
	default:
		return nil
	}
}

// KubeCPRestartOp returns an Operator to restart kubernetes control planes
func KubeCPRestartOp(cps []*Node, rivers, apiserver, controllerManager, scheduler []*Node, cluster string, serviceSubnet string, options Options) Operator {
	return &kubeCPRestartOp{
		cps:               cps,
		rivers:            rivers,
		apiserver:         apiserver,
		controllerManager: controllerManager,
		scheduler:         scheduler,
		cluster:           cluster,
		serviceSubnet:     serviceSubnet,
		options:           options,
	}
}

func (o *kubeCPRestartOp) Name() string {
	return "kubernetes-control-plane-restart"
}

func (o *kubeCPRestartOp) NextCommand() Commander {
	var opts []string

	switch o.step1 {
	case 0:
		o.step1++
		return imagePullCommand{o.cps, ToolsImage}
	case 1:
		o.step1++
		return imagePullCommand{o.cps, HyperkubeImage}
	case 2:
		if o.nodeIndex >= len(o.rivers) {
			o.step1++
			o.nodeIndex = 0
			return o.NextCommand()
		}
		node := o.rivers[o.nodeIndex]

		switch o.step2 {
		case 0:
			o.step2++
			return killContainersCommand{[]*Node{node}, riversContainerName}
		case 1:
			o.step2++
			return runContainerCommand{[]*Node{node}, riversContainerName, ToolsImage,
				opts, RiversParams(o.cps), o.options.Rivers}
		default:
			o.step2 = 0
			o.nodeIndex++
			return o.NextCommand()
		}
	case 3:
		if o.nodeIndex >= len(o.apiserver) {
			o.step1++
			o.nodeIndex = 0
			return o.NextCommand()
		}
		node := o.apiserver[o.nodeIndex]

		switch o.step2 {
		case 0:
			o.step2++
			return stopContainersCommand{[]*Node{node}, kubeAPIServerContainerName}
		case 1:
			o.step2++
			opts = []string{
				// TODO pass keys from CKE
				"--mount", "type=tmpfs,dst=/run/kubernetes",
			}
			return runContainerCommand{[]*Node{node}, kubeAPIServerContainerName, HyperkubeImage,
				opts, APIServerParams(o.cps, node.Address, o.serviceSubnet), o.options.APIServer}
		default:
			o.step2 = 0
			o.nodeIndex++
			return o.NextCommand()
		}
	case 4:
		if o.nodeIndex >= len(o.controllerManager) {
			o.step1++
			o.nodeIndex = 0
			return o.NextCommand()
		}
		node := o.controllerManager[o.nodeIndex]

		switch o.step2 {
		case 0:
			o.step2++
			return stopContainersCommand{[]*Node{node}, kubeControllerManagerContainerName}
		case 1:
			o.step2++
			return runContainerCommand{[]*Node{node}, kubeControllerManagerContainerName, HyperkubeImage,
				opts, ControllerManagerParams(o.cluster, o.serviceSubnet), o.options.ControllerManager}
		default:
			o.step2 = 0
			o.nodeIndex++
			return o.NextCommand()
		}
	case 5:
		if o.nodeIndex >= len(o.scheduler) {
			o.step1++
			o.nodeIndex = 0
			return o.NextCommand()
		}
		node := o.scheduler[o.nodeIndex]

		switch o.step2 {
		case 0:
			o.step2++
			return stopContainersCommand{[]*Node{node}, kubeSchedulerContainerName}
		case 1:
			o.step2++
			return runContainerCommand{[]*Node{node}, kubeSchedulerContainerName, HyperkubeImage,
				opts, SchedulerParams(), o.options.Scheduler}
		default:
			o.step2 = 0
			o.nodeIndex++
			return o.NextCommand()
		}
	default:
		return nil
	}
}

// APIServerParams returns built-in a ServiceParams form kube-apiserver
func APIServerParams(controlPlanes []*Node, advertiseAddress, serviceSubnet string) ServiceParams {
	var etcdServers []string
	for _, n := range controlPlanes {
		etcdServers = append(etcdServers, "https://"+n.Address+":2379")
	}

	args := []string{
		"apiserver",
		"--allow-privileged",
		"--etcd-servers=" + strings.Join(etcdServers, ","),
		"--etcd-cafile=" + K8sPKIPath("etcd/ca.crt"),
		"--etcd-certfile=" + K8sPKIPath("apiserver-etcd-client.crt"),
		"--etcd-keyfile=" + K8sPKIPath("apiserver-etcd-client.key"),

		"--bind-address=0.0.0.0",
		"--insecure-port=0",
		"--client-ca-file=" + K8sPKIPath("ca.crt"),
		"--tls-cert-file=" + K8sPKIPath("apiserver.crt"),
		"--tls-private-key-file=" + K8sPKIPath("apiserver.key"),
		"--kubelet-certificate-authority=" + K8sPKIPath("ca.crt"),
		"--kubelet-client-certificate=" + K8sPKIPath("apiserver.crt"),
		"--kubelet-client-key=" + K8sPKIPath("apiserver.key"),
		"--kubelet-https=true",

		"--enable-admission-plugins=" + strings.Join(admissionPlugins, ","),

		// for service accounts
		"--service-account-key-file=" + K8sPKIPath("service-account.crt"),
		"--service-account-lookup",

		"--authorization-mode=Node,RBAC",

		"--advertise-address=" + advertiseAddress,
		"--service-cluster-ip-range=" + serviceSubnet,
		"--audit-log-path=/var/log/kubernetes/apiserver/audit.log",
		"--log-dir=/var/log/kubernetes/apiserver/",
		"--logtostderr=false",
		"--machine-id-file=/etc/machine-id",
	}
	return ServiceParams{
		ExtraArguments: args,
		ExtraBinds: []Mount{
			{"/etc/hostname", "/etc/machine-id", true, ""},
			{"/var/log/kubernetes/apiserver", "/var/log/kubernetes/apiserver", false, ""},
			{"/etc/kubernetes", "/etc/kubernetes", true, ""},
		},
	}
}

// ControllerManagerParams returns a ServiceParams for kube-controller-manager
func ControllerManagerParams(clusterName, serviceSubnet string) ServiceParams {
	args := []string{
		"controller-manager",
		"--cluster-name=" + clusterName,
		"--service-cluster-ip-range=" + serviceSubnet,
		"--kubeconfig=/etc/kubernetes/controller-manager/kubeconfig",
		"--log-dir=/var/log/kubernetes/controller-manager",
		"--logtostderr=false",

		// ToDo: cluster signing
		// https://kubernetes.io/docs/tasks/tls/managing-tls-in-a-cluster/#a-note-to-cluster-administrators
		// https://kubernetes.io/docs/reference/command-line-tools-reference/kubelet-tls-bootstrapping/
		//    Create an intermediate CA under cke/ca-kubernetes?
		//    or just an certficate/key pair?
		// "--cluster-signing-cert-file=..."
		// "--cluster-signing-key-file=..."

		// for service accounts
		"--root-ca-file=" + K8sPKIPath("ca.crt"),
		"--service-account-private-key-file=" + K8sPKIPath("service-account.key"),
		"--use-service-account-credentials=true",
	}
	return ServiceParams{
		ExtraArguments: args,
		ExtraBinds: []Mount{
			{"/etc/hostname", "/etc/machine-id", true, ""},
			{"/etc/kubernetes", "/etc/kubernetes", true, ""},
			{"/var/log/kubernetes/controller-manager", "/var/log/kubernetes/controller-manager", false, ""},
		},
	}
}

// SchedulerParams return a ServiceParams form kube-scheduler
func SchedulerParams() ServiceParams {
	args := []string{
		"scheduler",
		"--kubeconfig=/etc/kubernetes/scheduler/kubeconfig",
		"--log-dir=/var/log/kubernetes/scheduler",
		"--logtostderr=false",
	}
	return ServiceParams{
		ExtraArguments: args,
		ExtraBinds: []Mount{
			{"/etc/hostname", "/etc/machine-id", true, ""},
			{"/etc/kubernetes", "/etc/kubernetes", true, ""},
			{"/var/log/kubernetes/scheduler", "/var/log/kubernetes/scheduler", false, ""},
		},
	}
}

// KubeWorkerBootOp returns an Operator to boot kubernetes workers.
func KubeWorkerBootOp(cps []*Node, kubelets, proxies []*Node, options Options) Operator {
	return &kubeWorkerBootOp{
		cps:      cps,
		kubelets: kubelets,
		proxies:  proxies,
		options:  options,
	}
}

func (o *kubeWorkerBootOp) Name() string {
	return "worker-bootstrap"
}

func (o *kubeWorkerBootOp) NextCommand() Commander {
	var opts []string

	// Romana and Kubelet use these paths
	// https://github.com/romana/romana/blob/1aa6b9b8d43e0eb3830d581deec8f3ab4bba5833/docs/kubernetes/romana-kubeadm.yml#L259-L262
	cniBinDir := "/host/opt/cni/bin"
	cniConfDir := "/host/etc/cni/net.d"

	switch o.step {
	case 0:
		o.step++
		return imagePullCommand{o.kubelets, "cke-tools"}
	case 1:
		o.step++
		if len(o.kubelets) == 0 {
			return o.NextCommand()
		}
		return makeDirCommand{o.kubelets, cniBinDir}
	case 2:
		o.step++
		if len(o.kubelets) == 0 {
			return o.NextCommand()
		}
		return makeDirCommand{o.kubelets, cniConfDir}
	case 3:
		o.step++
		if len(o.kubelets) == 0 {
			return o.NextCommand()
		}
		opts = []string{
			"--mount", "type=bind,src=" + cniBinDir + ",target=/host/bin",
			"--mount", "type=bind,src=" + cniConfDir + ",target=/host/net.d"}
		return runContainerCommand{nodes: o.kubelets, name: "cke-tools", opts: opts,
			params: ServiceParams{ExtraArguments: []string{"/usr/local/cke-tools/bin/install-cni"}},
		}
	case 4:
		o.step++
		if len(o.proxies) == 0 {
			return o.NextCommand()
		}
		return imagePullCommand{o.proxies, HyperkubeImage}
	case 5:
		o.step++
		if len(o.kubelets) == 0 {
			return o.NextCommand()
		}
		return makeDirCommand{o.kubelets, "/var/log/kubernetes/kubelet"}
	case 6:
		o.step++
		if len(o.proxies) == 0 {
			return o.NextCommand()
		}
		return makeDirCommand{o.proxies, "/var/log/kubernetes/proxy"}
	case 7:
		o.step++
		if len(o.kubelets) == 0 {
			return o.NextCommand()
		}
		return makeKubeletKubeconfigCommand{o.kubelets, o.cluster, o.options.Kubelet}
	case 8:
		o.step++
		if len(o.proxies) == 0 {
			return o.NextCommand()
		}
		return makeProxyKubeconfigCommand{o.proxies, o.cluster}
	case 9:
		o.step++
		if len(o.kubelets) == 0 {
			return o.NextCommand()
		}
		return volumeCreateCommand{o.kubelets, "dockershim"}
	case 10:
		o.step++
		if len(o.kubelets) == 0 {
			return o.NextCommand()
		}
		opts = []string{
			"--pid=host",
			"--mount=type=volume,src=dockershim,dst=/var/lib/dockershim",
			"--mount=type=bind,src=" + cniBinDir + ",target=/opt/cni/bin",
			"--mount=type=bind,src=" + cniConfDir + ",target=/etc/cni/net.d",
			"--privileged",
		}
		params := make(map[string]ServiceParams)
		for _, n := range o.kubelets {
			params[n.Address] = KubeletServiceParams(n)
		}
		return runContainerParamsCommand{o.kubelets, kubeletContainerName, HyperkubeImage,
			opts, params, o.options.Kubelet.ServiceParams}
	case 11:
		o.step++
		if len(o.proxies) == 0 {
			return o.NextCommand()
		}
		opts = []string{
			"--tmpfs=/run",
			"--privileged",
		}
		return runContainerCommand{o.proxies, kubeProxyContainerName, HyperkubeImage,
			opts, ProxyParams(), o.options.Proxy}
	default:
		return nil
	}
}

// KubeWorkerRestartOp returns an Operator to restart kubernetes workers
func KubeWorkerRestartOp(cps []*Node, rivers, kubelets, proxies []*Node, options Options) Operator {
	return &kubeWorkerRestartOp{
		cps:      cps,
		rivers:   rivers,
		kubelets: kubelets,
		proxies:  proxies,
		options:  options,
	}
}

func (o *kubeWorkerRestartOp) Name() string {
	return "worker-restart"
}

func (o *kubeWorkerRestartOp) NextCommand() Commander {
	var opts []string

	switch o.step {
	case 0:
		o.step++
		if len(o.rivers) == 0 {
			return o.NextCommand()
		}
		return imagePullCommand{o.rivers, ToolsImage}
	case 1:
		o.step++
		if len(o.proxies)+len(o.kubelets) == 0 {
			return o.NextCommand()
		}
		return imagePullCommand{o.proxies, HyperkubeImage}
	case 2:
		o.step++
		if len(o.kubelets) == 0 {
			return o.NextCommand()
		}
		return makeKubeletKubeconfigCommand{o.kubelets, o.cluster, o.options.Kubelet}
	case 3:
		o.step++
		if len(o.proxies) == 0 {
			return o.NextCommand()
		}
		return makeProxyKubeconfigCommand{o.proxies, o.cluster}
	case 4:
		o.step++
		if len(o.rivers) == 0 {
			return o.NextCommand()
		}
		return killContainersCommand{o.rivers, riversContainerName}
	case 5:
		o.step++
		if len(o.rivers) == 0 {
			return o.NextCommand()
		}
		return runContainerCommand{o.rivers, riversContainerName, ToolsImage,
			opts, RiversParams(o.cps), o.options.Rivers}
	case 6:
		o.step++
		if len(o.kubelets) == 0 {
			return o.NextCommand()
		}
		return stopContainersCommand{o.kubelets, kubeletContainerName}
	case 7:
		o.step++
		if len(o.kubelets) == 0 {
			return o.NextCommand()
		}
		opts = []string{
			"--pid=host",
			"--mount=type=volume,src=dockershim,dst=/var/lib/dockershim",
			"--privileged",
		}
		params := make(map[string]ServiceParams)
		for _, n := range o.kubelets {
			params[n.Address] = KubeletServiceParams(n)
		}
		return runContainerParamsCommand{o.kubelets, kubeletContainerName, HyperkubeImage,
			opts, params, o.options.Kubelet.ServiceParams}
	case 8:
		o.step++
		if len(o.proxies) == 0 {
			return o.NextCommand()
		}
		return stopContainersCommand{o.proxies, kubeProxyContainerName}
	case 9:
		o.step++
		if len(o.proxies) == 0 {
			return o.NextCommand()
		}
		opts = []string{
			"--tmpfs=/run",
			"--privileged",
		}
		return runContainerCommand{o.proxies, kubeProxyContainerName, HyperkubeImage,
			opts, ProxyParams(), o.options.Proxy}
	default:
		return nil
	}
}

// KubeRBACRoleInstallOp returns an Operator to install ClusterRole and binding for RBAC.
func KubeRBACRoleInstallOp(apiserver *Node, roleExists bool) Operator {
	return &kubeRBACRoleInstallOp{
		apiserver:  apiserver,
		roleExists: roleExists,
	}
}

func (o *kubeRBACRoleInstallOp) Name() string {
	return "install-rbac-role"
}

func (o *kubeRBACRoleInstallOp) NextCommand() Commander {
	switch {
	case !o.roleExists:
		o.roleExists = true
		return makeRBACRoleCommand{o.apiserver}
	case !o.bindingExists:
		o.bindingExists = true
		return makeRBACRoleBindingCommand{o.apiserver}
	}
	return nil
}

// ProxyParams returns a ServiceParams form kube-proxy
func ProxyParams() ServiceParams {
	args := []string{
		"proxy",
		"--proxy-mode=ipvs",
		"--kubeconfig=/etc/kubernetes/proxy/kubeconfig",
		"--log-dir=/var/log/kubernetes/proxy",
		"--logtostderr=false",
	}
	return ServiceParams{
		ExtraArguments: args,
		ExtraBinds: []Mount{
			{"/etc/hostname", "/etc/machine-id", true, ""},
			{"/etc/kubernetes", "/etc/kubernetes", true, ""},
			{"/lib/modules", "/lib/modules", true, ""},
			{"/var/log/kubernetes/proxy", "/var/log/kubernetes/proxy", false, ""},
		},
	}
}

// KubeletServiceParams returns a ServiceParams for kubelet
func KubeletServiceParams(n *Node) ServiceParams {
	args := []string{
		"kubelet",
		"--config=/etc/kubernetes/kubelet/config.yml",
		"--kubeconfig=/etc/kubernetes/kubelet/kubeconfig",
		"--allow-privileged=true",
		"--hostname-override=" + n.Nodename(),
		"--pod-infra-container-image=" + PauseImage.Name(),
		"--log-dir=/var/log/kubernetes/kubelet",
		"--logtostderr=false",
		"--network-plugin=cni",
	}
	return ServiceParams{
		ExtraArguments: args,
		ExtraBinds: []Mount{
			{"/etc/hostname", "/etc/machine-id", true, ""},
			{"/etc/kubernetes", "/etc/kubernetes", true, ""},
			{"/var/lib/kubelet", "/var/lib/kubelet", false, "shared"},
			{"/var/lib/docker", "/var/lib/docker", false, "rslave"},
			{"/var/log/pods", "/var/log/pods", false, ""},
			{"/var/log/kubernetes/kubelet", "/var/log/kubernetes/kubelet", false, ""},
			{"/run", "/run", false, ""},
			{"/sys", "/sys", true, ""},
			{"/dev", "/dev", false, ""},
		},
	}
}

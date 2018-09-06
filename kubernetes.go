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

	serviceSubnet string
	options       Options

	step      int
	nodeIndex int
}

type kubeCPRestartOp struct {
	cps []*Node

	apiserver         []*Node
	controllerManager []*Node
	scheduler         []*Node

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

type kubeletBootOp struct {
	nodes  []*Node
	params KubeletParams
	step   int
}

type riversStopOp struct {
	nodes     []*Node
	nodeIndex int
}

type proxyBootOp struct {
	nodes  []*Node
	params ServiceParams
	step   int
}

type kubeletStopOp struct {
	nodes     []*Node
	step      int
	nodeIndex int
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
		return imagePullCommand{o.nodes, "rivers"}
	case 1:
		o.step++
		return makeDirCommand{o.nodes, "/var/log/rivers"}
	case 2:
		o.step++
		return runContainerCommand{o.nodes, "rivers", opts, riversParams(o.upstreams), extra}
	default:
		return nil
	}
}

func riversParams(upstreams []*Node) ServiceParams {
	var ups []string
	for _, n := range upstreams {
		ups = append(ups, n.Address+":8080")
	}
	args := []string{
		"rivers",
		"--upstreams=" + strings.Join(ups, ","),
		"--listen=" + "127.0.0.1:18080",
	}
	return ServiceParams{
		ExtraArguments: args,
		ExtraBinds: []Mount{
			{"/var/log/rivers", "/var/log/rivers", false},
		},
	}
}

// KubeCPBootOp returns an Operator to bootstrap kubernetes control planes
func KubeCPBootOp(cps []*Node, apiserver, controllerManager, scheduler []*Node, serviceSubnet string, options Options) Operator {
	return &kubeCPBootOp{
		cps:               cps,
		apiserver:         apiserver,
		controllerManager: controllerManager,
		scheduler:         scheduler,
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
		return imagePullCommand{o.cps, kubeAPIServerContainerName}
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
		return runContainerCommand{[]*Node{node}, kubeAPIServerContainerName, opts, apiServerParams(o.cps, node.Address, o.serviceSubnet), o.options.APIServer}
	case 6:
		o.step++
		if len(o.scheduler) == 0 {
			return o.NextCommand()
		}
		return runContainerCommand{o.scheduler, kubeSchedulerContainerName, opts, schedulerParams(), o.options.Scheduler}
	case 7:
		o.step++
		if len(o.controllerManager) == 0 {
			return o.NextCommand()
		}
		return runContainerCommand{o.controllerManager, kubeControllerManagerContainerName, opts, controllerManagerParams(), o.options.ControllerManager}
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
func KubeCPRestartOp(cps []*Node, apiserver, controllerManager, scheduler []*Node, serviceSubnet string, options Options) Operator {
	return &kubeCPRestartOp{
		cps:               cps,
		apiserver:         apiserver,
		controllerManager: controllerManager,
		scheduler:         scheduler,
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
		return imagePullCommand{o.cps, kubeAPIServerContainerName}
	case 1:
		if o.nodeIndex >= len(o.apiserver) {
			o.step1++
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
			return runContainerCommand{[]*Node{node}, kubeAPIServerContainerName, opts, apiServerParams(o.cps, node.Address, o.serviceSubnet), o.options.APIServer}
		default:
			o.step2 = 0
			o.nodeIndex++
			return o.NextCommand()
		}
	case 2:
		if o.nodeIndex >= len(o.controllerManager) {
			o.step1++
			return o.NextCommand()
		}
		node := o.controllerManager[o.nodeIndex]

		switch o.step2 {
		case 0:
			o.step2++
			return stopContainersCommand{[]*Node{node}, kubeControllerManagerContainerName}
		case 1:
			o.step2++
			return runContainerCommand{[]*Node{node}, kubeControllerManagerContainerName, opts, apiServerParams(o.cps, node.Address, o.serviceSubnet), o.options.ControllerManager}
		default:
			o.step2 = 0
			o.nodeIndex++
			return o.NextCommand()
		}
	case 3:
		if o.nodeIndex >= len(o.scheduler) {
			o.step1++
			return o.NextCommand()
		}
		node := o.scheduler[o.nodeIndex]

		switch o.step2 {
		case 0:
			o.step2++
			return stopContainersCommand{[]*Node{node}, kubeSchedulerContainerName}
		case 1:
			o.step2++
			return runContainerCommand{[]*Node{node}, kubeSchedulerContainerName, opts, apiServerParams(o.cps, node.Address, o.serviceSubnet), o.options.Scheduler}
		default:
			o.step2 = 0
			o.nodeIndex++
			return o.NextCommand()
		}
	default:
		return nil
	}

}

func apiServerParams(controlPlanes []*Node, advertiseAddress string, serviceSubnet string) ServiceParams {
	var etcdServers []string
	for _, n := range controlPlanes {
		etcdServers = append(etcdServers, "https://"+n.Address+":2379")
	}
	args := []string{
		"apiserver",
		"--allow-privileged",
		"--etcd-servers=" + strings.Join(etcdServers, ","),
		"--etcd-cafile=/etc/kubernetes/apiserver/ca-server.crt",
		"--etcd-certfile=/etc/kubernetes/apiserver/apiserver.crt",
		"--etcd-keyfile=/etc/kubernetes/apiserver/apiserver.key",

		// TODO use TLS
		"--insecure-bind-address=0.0.0.0",
		"--insecure-port=8080",

		"--advertise-address=" + advertiseAddress,
		"--service-cluster-ip-range=" + serviceSubnet,
		"--audit-log-path=/var/log/kubernetes/apiserver/audit.log",
		"--log-dir=/var/log/kubernetes/apiserver/",
		"--machine-id-file=/etc/machine-id",
	}
	return ServiceParams{
		ExtraArguments: args,
		ExtraBinds: []Mount{
			{"/etc/hostname", "/etc/machine-id", true},
			{"/var/log/kubernetes/apiserver", "/var/log/kubernetes/apiserver", false},
			{"/etc/kubernetes/apiserver", "/etc/kubernetes/apiserver", true},
		},
	}
}

func controllerManagerParams() ServiceParams {
	args := []string{
		"controller-manager",
		"--kubeconfig=/etc/kubernetes/controller-manager/kubeconfig",
		"--log-dir=/var/log/kubernetes/controller-manager",
	}
	return ServiceParams{
		ExtraArguments: args,
		ExtraBinds: []Mount{
			{"/etc/hostname", "/etc/machine-id", true},
			{"/etc/kubernetes/controller-manager", "/etc/kubernetes/controller-manager", true},
			{"/var/log/kubernetes/controller-manager", "/var/log/kubernetes/controller-manager", false},
		},
	}
}

// SchedulerBootOp returns an Operator to bootstrap Scheduler cluster.
func schedulerParams() ServiceParams {
	args := []string{
		"scheduler",
		"--kubeconfig=/etc/kubernetes/scheduler/kubeconfig",
		"--log-dir=/var/log/kubernetes/scheduler",
	}
	return ServiceParams{
		ExtraArguments: args,
		ExtraBinds: []Mount{
			{"/etc/hostname", "/etc/machine-id", true},
			{"/etc/kubernetes/scheduler", "/etc/kubernetes/scheduler", true},
			{"/var/log/kubernetes/scheduler", "/var/log/kubernetes/scheduler", false},
		},
	}
}

// KubeletBootOp returns an Operator to bootstrap Kubelet.
func KubeletBootOp(nodes []*Node, params KubeletParams) Operator {
	return &kubeletBootOp{
		nodes:  nodes,
		params: params,
		step:   0,
	}
}

func (o *kubeletBootOp) Name() string {
	return "kubelet-bootstrap"
}

func (o *kubeletBootOp) NextCommand() Commander {
	volName := "dockershim"
	opts := []string{
		"--tmpfs=/var/tmp/dockershim",
		"--privileged",
	}
	switch o.step {
	case 0:
		o.step++
		return makeFileCommand{o.nodes, kubeletKubeConfig(), "/etc/kubernetes/kubelet/kubeconfig"}
	case 1:
		o.step++
		return imagePullCommand{o.nodes, kubeletContainerName}
	case 2:
		o.step++
		return imagePullCommand{o.nodes, pauseContainerName}
	case 3:
		o.step++
		return makeDirCommand{o.nodes, "/var/log/kubernetes/kubelet"}
	case 4:
		o.step++
		return volumeCreateCommand{o.nodes, volName}
	case 5:
		o.step++
		return runContainerCommand{o.nodes, kubeletContainerName, opts, o.serviceParams(), o.extraParams()}
	default:
		return nil
	}
}

func (o *kubeletBootOp) serviceParams() ServiceParams {
	args := []string{
		"kubelet",
		"--allow-privileged=true",
		"--container-runtime-endpoint=/var/tmp/dockershim/dockershim.sock",
		"--pod-infra-container-image=" + Image(pauseContainerName),
		"--kubeconfig=/etc/kubernetes/kubelet/kubeconfig",
		"--log-dir=/var/log/kubernetes/kubelet",
	}
	return ServiceParams{
		ExtraArguments: args,
		ExtraBinds: []Mount{
			{"/etc/hostname", "/etc/machine-id", true},
			{"/etc/kubernetes/kubelet", "/etc/kubernetes/kubelet", true},
			{"/var/lib/kubelet", "/var/lib/kubelet", false},
			{"/var/lib/docker", "/var/lib/docker", false},
			{"/var/lib/dockershim", "/var/lib/dockershim", false},
			{"/var/log/pods", "/var/log/pods", false},
			{"/var/log/kubernetes/kubelet", "/var/log/kubernetes/kubelet", false},
			{"/var/run/docker.sock", "/var/run/docker.sock", false},
		},
	}
}

func (o *kubeletBootOp) extraParams() ServiceParams {
	extraArgs := o.params.ExtraArguments
	if len(o.params.Domain) > 0 {
		extraArgs = append(extraArgs, "--cluster-domain="+o.params.Domain)
	}
	if o.params.AllowSwap {
		extraArgs = append(extraArgs, "--fail-swap-on=false")
	}
	return ServiceParams{
		ExtraArguments: extraArgs,
		ExtraBinds:     o.params.ExtraBinds,
		ExtraEnvvar:    o.params.ExtraEnvvar,
	}
}

// RiversStopOp returns an Operator to stop Rivers.
func RiversStopOp(nodes []*Node) Operator {
	return &riversStopOp{
		nodes:     nodes,
		nodeIndex: 0,
	}
}

// ProxyBootOp returns an Operator to bootstrap Proxy
func ProxyBootOp(nodes []*Node, params ServiceParams) Operator {
	return &proxyBootOp{
		nodes:  nodes,
		params: params,
		step:   0,
	}
}

func (o *riversStopOp) Name() string {
	return "rivers-stop"
}

func (o *riversStopOp) NextCommand() Commander {
	if o.nodeIndex >= len(o.nodes) {
		return nil
	}

	node := o.nodes[o.nodeIndex]
	o.nodeIndex++

	return killContainerCommand{node, "rivers"}
}

func (o *proxyBootOp) Name() string {
	return "proxy-bootstrap"
}

func (o *proxyBootOp) NextCommand() Commander {
	extra := o.params
	opts := []string{
		"--tmpfs=/run",
		"--privileged",
	}

	switch o.step {
	case 0:
		o.step++
		return makeFileCommand{o.nodes, proxyKubeConfig(), "/etc/kubernetes/proxy/kubeconfig"}
	case 1:
		o.step++
		return imagePullCommand{o.nodes, kubeProxyContainerName}
	case 2:
		o.step++
		return makeDirCommand{o.nodes, "/var/log/kubernetes/proxy"}
	case 3:
		o.step++
		return runContainerCommand{o.nodes, kubeProxyContainerName, opts, o.serviceParams(), extra}
	default:
		return nil
	}
}

func (o *proxyBootOp) serviceParams() ServiceParams {
	args := []string{
		"proxy",
		"--proxy-mode=ipvs",
		"--kubeconfig=/etc/kubernetes/proxy/kubeconfig",
		"--log-dir=/var/log/kubernetes/proxy",
	}
	return ServiceParams{
		ExtraArguments: args,
		ExtraBinds: []Mount{
			{"/etc/hostname", "/etc/machine-id", true},
			{"/etc/kubernetes/proxy", "/etc/kubernetes/proxy", true},
			{"/lib/modules", "/lib/modules", true},
			{"/var/log/kubernetes/proxy", "/var/log/kubernetes/proxy", false},
		},
	}
}

// KubeletStopOp returns an Operator to stop Kubelet.
func KubeletStopOp(nodes []*Node) Operator {
	return &kubeletStopOp{
		nodes:     nodes,
		step:      0,
		nodeIndex: 0,
	}
}

func (o *kubeletStopOp) Name() string {
	return "kubelet-stop"
}

func (o *kubeletStopOp) NextCommand() Commander {
	switch o.step {
	case 0:
		o.step++
		return removeFileCommand{o.nodes, "/etc/kubernetes/kubelet/kubeconfig"}
	case 1:
		if o.nodeIndex >= len(o.nodes) {
			return nil
		}

		node := o.nodes[o.nodeIndex]
		o.nodeIndex++

		return stopContainerCommand{node, kubeletContainerName}
	default:
		return nil
	}
}

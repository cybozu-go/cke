package cke

import (
	"strings"
)

type riversBootOp struct {
	nodes     []*Node
	upstreams []*Node
	agents    map[string]Agent
	params    ServiceParams
	step      int
	nodeIndex int
}

type apiServerBootOp struct {
	nodes         []*Node
	controlPlanes []*Node
	agents        map[string]Agent
	params        ServiceParams
	step          int
	nodeIndex     int
	serviceSubnet string
}

type controllerManagerBootOp struct {
	nodes     []*Node
	agents    map[string]Agent
	params    ServiceParams
	step      int
	nodeIndex int
}

type schedulerBootOp struct {
	nodes     []*Node
	agents    map[string]Agent
	params    ServiceParams
	step      int
	nodeIndex int
}

type kubeletBootOp struct {
	nodes     []*Node
	agents    map[string]Agent
	params    KubeletParams
	step      int
	nodeIndex int
}

type riversStopOp struct {
	nodes     []*Node
	agents    map[string]Agent
	nodeIndex int
}

type proxyBootOp struct {
	nodes     []*Node
	agents    map[string]Agent
	params    ServiceParams
	step      int
	nodeIndex int
}

type apiServerStopOp struct {
	nodes     []*Node
	agents    map[string]Agent
	nodeIndex int
}

type controllerManagerStopOp struct {
	nodes     []*Node
	agents    map[string]Agent
	step      int
	nodeIndex int
}

type schedulerStopOp struct {
	nodes     []*Node
	agents    map[string]Agent
	step      int
	nodeIndex int
}

type kubeletStopOp struct {
	nodes     []*Node
	agents    map[string]Agent
	step      int
	nodeIndex int
}

// RiversBootOp returns an Operator to bootstrap rivers cluster.
func RiversBootOp(nodes []*Node, upstreams []*Node, agents map[string]Agent, params ServiceParams) Operator {
	return &riversBootOp{
		nodes:     nodes,
		upstreams: upstreams,
		agents:    agents,
		params:    params,
		step:      0,
		nodeIndex: 0,
	}
}

func (o *riversBootOp) Name() string {
	return "rivers-bootstrap"
}

func (o *riversBootOp) NextCommand() Commander {
	extra := o.params
	opts := []string{}

	switch o.step {
	case 0:
		o.step++
		return imagePullCommand{o.nodes, o.agents, "rivers"}
	case 1:
		o.step++
		return makeDirCommand{o.nodes, o.agents, "/var/log/rivers"}
	case 2:
		if o.nodeIndex >= len(o.nodes) {
			return nil
		}

		target := o.nodes[o.nodeIndex]
		o.nodeIndex++

		return runContainerCommand{target, o.agents[target.Address], "rivers", opts, riversParams(o.upstreams), extra}
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

// APIServerBootOp returns an Operator to bootstrap APIServer cluster.
func APIServerBootOp(nodes, controlPlanes []*Node, agents map[string]Agent, params ServiceParams, serviceSubnet string) Operator {
	return &apiServerBootOp{
		nodes:         nodes,
		controlPlanes: controlPlanes,
		agents:        agents,
		params:        params,
		step:          0,
		nodeIndex:     0,
		serviceSubnet: serviceSubnet,
	}
}

func (o *apiServerBootOp) Name() string {
	return "apiserver-bootstrap"
}

func (o *apiServerBootOp) NextCommand() Commander {
	extra := o.params

	switch o.step {
	case 0:
		o.step++
		return imagePullCommand{o.nodes, o.agents, "kube-apiserver"}
	case 1:
		o.step++
		return makeDirCommand{o.nodes, o.agents, "/var/log/kubernetes/apiserver"}
	case 2:
		opts := []string{
			// TODO pass keys from CKE
			"--mount", "type=tmpfs,dst=/run/kubernetes",
		}
		if o.nodeIndex >= len(o.nodes) {
			return nil
		}

		target := o.nodes[o.nodeIndex]
		o.nodeIndex++

		return runContainerCommand{target, o.agents[target.Address], "kube-apiserver", opts, apiServerParams(o.controlPlanes, target.Address, o.serviceSubnet), extra}
	default:
		return nil
	}
}

func apiServerParams(controlPlanes []*Node, advertiseAddress string, serviceSubnet string) ServiceParams {
	var etcdServers []string
	for _, n := range controlPlanes {
		etcdServers = append(etcdServers, "http://"+n.Address+":2379")
	}
	args := []string{
		"apiserver",
		"--allow-privileged",
		"--etcd-servers=" + strings.Join(etcdServers, ","),

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
		},
	}
}

// ControllerManagerBootOp returns an Operator to bootstrap ControllerManager cluster.
func ControllerManagerBootOp(nodes []*Node, agents map[string]Agent, params ServiceParams, serviceSubnet string) Operator {
	return &controllerManagerBootOp{
		nodes:     nodes,
		agents:    agents,
		params:    params,
		step:      0,
		nodeIndex: 0,
	}
}

func (o *controllerManagerBootOp) Name() string {
	return "controller-manager-bootstrap"
}

func (o *controllerManagerBootOp) NextCommand() Commander {
	extra := o.params
	opts := []string{}

	switch o.step {
	case 0:
		o.step++
		return makeFileCommand{o.nodes, o.agents, controllerManagerKubeconfig(), "/etc/kubernetes/controller-manager/kubeconfig"}
	case 1:
		o.step++
		return imagePullCommand{o.nodes, o.agents, "kube-controller-manager"}
	case 2:
		o.step++
		return makeDirCommand{o.nodes, o.agents, "/var/log/kubernetes/controller-manager"}
	case 3:
		if o.nodeIndex >= len(o.nodes) {
			return nil
		}

		target := o.nodes[o.nodeIndex]
		o.nodeIndex++

		return runContainerCommand{target, o.agents[target.Address], "kube-controller-manager", opts, controllerManagerParams(), extra}
	default:
		return nil
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
func SchedulerBootOp(nodes []*Node, agents map[string]Agent, params ServiceParams, serviceSubnet string) Operator {
	return &schedulerBootOp{
		nodes:     nodes,
		agents:    agents,
		params:    params,
		step:      0,
		nodeIndex: 0,
	}
}

func (o *schedulerBootOp) Name() string {
	return "scheduler-bootstrap"
}

func (o *schedulerBootOp) NextCommand() Commander {
	extra := o.params

	switch o.step {
	case 0:
		o.step++
		return makeFileCommand{o.nodes, o.agents, schedulerKubeconfig(), "/etc/kubernetes/scheduler/kubeconfig"}
	case 1:
		o.step++
		return imagePullCommand{o.nodes, o.agents, "kube-scheduler"}
	case 2:
		o.step++
		return makeDirCommand{o.nodes, o.agents, "/var/log/kubernetes/scheduler"}
	case 3:
		if o.nodeIndex >= len(o.nodes) {
			return nil
		}

		target := o.nodes[o.nodeIndex]
		o.nodeIndex++

		return runContainerCommand{target, o.agents[target.Address], "kube-scheduler", nil, schedulerParams(), extra}
	default:
		return nil
	}
}

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
func KubeletBootOp(nodes []*Node, agents map[string]Agent, params KubeletParams) Operator {
	return &kubeletBootOp{
		nodes:     nodes,
		agents:    agents,
		params:    params,
		step:      0,
		nodeIndex: 0,
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
		return makeFileCommand{o.nodes, o.agents, kubeletKubeConfig(), "/etc/kubernetes/kubelet/kubeconfig"}
	case 1:
		o.step++
		return imagePullCommand{o.nodes, o.agents, "kubelet"}
	case 2:
		o.step++
		return makeDirCommand{o.nodes, o.agents, "/var/log/kubernetes/kubelet"}
	case 3:
		o.step++
		return volumeCreateCommand{o.nodes, o.agents, volName}
	case 4:
		if o.nodeIndex >= len(o.nodes) {
			return nil
		}

		target := o.nodes[o.nodeIndex]
		o.nodeIndex++

		return runContainerCommand{target, o.agents[target.Address], "kubelet", opts, o.serviceParams(target.Address), o.extraParams()}
	default:
		return nil
	}
}

func (o *kubeletBootOp) serviceParams(targetAddress string) ServiceParams {
	args := []string{
		"kubelet",
		"--allow-privileged=true",
		"--container-runtime-endpoint=/var/tmp/dockershim/dockershim.sock",
		"--hostname-override=" + targetAddress,
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
func RiversStopOp(nodes []*Node, agents map[string]Agent) Operator {
	return &riversStopOp{
		nodes:     nodes,
		agents:    agents,
		nodeIndex: 0,
	}
}

// ProxyBootOp returns an Operator to bootstrap Proxy
func ProxyBootOp(nodes []*Node, agents map[string]Agent, params ServiceParams) Operator {
	return &proxyBootOp{
		nodes:     nodes,
		agents:    agents,
		params:    params,
		step:      0,
		nodeIndex: 0,
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

	return stopContainerCommand{node, o.agents[node.Address], "rivers"}
}

func (o *proxyBootOp) Name() string {
	return "proxy-bootstrap"
}

func (o *proxyBootOp) NextCommand() Commander {
	extra := o.params
	opts := []string{
		"--privileged",
	}

	switch o.step {
	case 0:
		o.step++
		return makeFileCommand{o.nodes, o.agents, proxyKubeConfig(), "/etc/kubernetes/proxy/config"}
	case 1:
		o.step++
		return imagePullCommand{o.nodes, o.agents, "kube-proxy"}
	case 2:
		o.step++
		return makeDirCommand{o.nodes, o.agents, "/var/log/kubernetes/proxy"}
	case 3:
		if o.nodeIndex >= len(o.nodes) {
			return nil
		}
		target := o.nodes[o.nodeIndex]
		o.nodeIndex++

		return runContainerCommand{target, o.agents[target.Address], "kube-proxy", opts, o.serviceParams(target.Address), extra}
	default:
		return nil
	}
}

func (o *proxyBootOp) serviceParams(targetAddress string) ServiceParams {
	args := []string{
		"proxy",
		"--proxy-mode ipvs",
		"--kubeconfig=/etc/kubernetes/proxy/kubeconfig",
		"--log-dir=/var/log/kubernetes/proxy",
	}
	return ServiceParams{
		ExtraArguments: args,
		ExtraBinds: []Mount{
			{"/etc/hostname", "/etc/macine-id", true},
			{"/etc/kubernetes/kubelet", "/etc/kubernetes/proxy", true},
			{"/var/log/kubernetes/proxy", "/var/log/kubernetes/proxy", false},
		},
	}
}

// APIServerStopOp returns an Operator to bootstrap Scheduler cluster.
func APIServerStopOp(nodes []*Node, agents map[string]Agent) Operator {
	return &apiServerStopOp{
		nodes:     nodes,
		agents:    agents,
		nodeIndex: 0,
	}
}

func (o *apiServerStopOp) Name() string {
	return "apiserver-stop"
}

func (o *apiServerStopOp) NextCommand() Commander {
	if o.nodeIndex >= len(o.nodes) {
		return nil
	}

	node := o.nodes[o.nodeIndex]
	o.nodeIndex++

	return stopContainerCommand{node, o.agents[node.Address], "kube-apiserver"}
}

// ControllerManagerStopOp returns an Operator to stop ControllerManager.
func ControllerManagerStopOp(nodes []*Node, agents map[string]Agent) Operator {
	return &controllerManagerStopOp{
		nodes:     nodes,
		agents:    agents,
		step:      0,
		nodeIndex: 0,
	}
}

func (o *controllerManagerStopOp) Name() string {
	return "controller-manager-stop"
}

func (o *controllerManagerStopOp) NextCommand() Commander {
	switch o.step {
	case 0:
		o.step++
		return removeFileCommand{o.nodes, o.agents, "/etc/kubernetes/controller-manager/kubeconfig"}
	case 1:
		if o.nodeIndex >= len(o.nodes) {
			return nil
		}

		node := o.nodes[o.nodeIndex]
		o.nodeIndex++

		return stopContainerCommand{node, o.agents[node.Address], "kube-controller-manager"}
	default:
		return nil
	}
}

// SchedulerStopOp returns an Operator to stop Scheduler.
func SchedulerStopOp(nodes []*Node, agents map[string]Agent) Operator {
	return &schedulerStopOp{
		nodes:     nodes,
		agents:    agents,
		step:      0,
		nodeIndex: 0,
	}
}

func (o *schedulerStopOp) Name() string {
	return "scheduler-stop"
}

func (o *schedulerStopOp) NextCommand() Commander {
	switch o.step {
	case 0:
		o.step++
		return removeFileCommand{o.nodes, o.agents, "/etc/kubernetes/scheduler/kubeconfig"}
	case 1:
		if o.nodeIndex >= len(o.nodes) {
			return nil
		}

		node := o.nodes[o.nodeIndex]
		o.nodeIndex++

		return stopContainerCommand{node, o.agents[node.Address], "kube-scheduler"}
	default:
		return nil
	}
}

// KubeletStopOp returns an Operator to stop Kubelet.
func KubeletStopOp(nodes []*Node, agents map[string]Agent) Operator {
	return &kubeletStopOp{
		nodes:     nodes,
		agents:    agents,
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
		return removeFileCommand{o.nodes, o.agents, "/etc/kubernetes/kubelet/kubeconfig"}
	case 1:
		if o.nodeIndex >= len(o.nodes) {
			return nil
		}

		node := o.nodes[o.nodeIndex]
		o.nodeIndex++

		return stopContainerCommand{node, o.agents[node.Address], "kubelet"}
	default:
		return nil
	}
}

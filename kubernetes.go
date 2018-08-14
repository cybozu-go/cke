package cke

import (
	"strings"
)

type riversBootOp struct {
	nodes     []*Node
	agents    map[string]Agent
	params    ServiceParams
	step      int
	nodeIndex int
}

type apiServerBootOp struct {
	nodes         []*Node
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

type riversStopOp struct {
	nodes     []*Node
	agents    map[string]Agent
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

// RiversBootOp returns an Operator to bootstrap rivers cluster.
func RiversBootOp(nodes []*Node, agents map[string]Agent, params ServiceParams) Operator {
	return &riversBootOp{
		nodes:     nodes,
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
	opts := []string{
		"--entrypoint", "/usr/local/cke-tools/bin/rivers",
	}
	extra := o.params

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

		var upstreams []string
		for _, n := range o.nodes {
			upstreams = append(upstreams, n.Address+":8080")
		}
		return runContainerCommand{target, o.agents[target.Address], "rivers", opts, riversParams(upstreams), extra}
	default:
		return nil
	}
}

func riversParams(upstreams []string) ServiceParams {
	args := []string{
		"--upstreams=" + strings.Join(upstreams, ","),
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
func APIServerBootOp(nodes []*Node, agents map[string]Agent, params ServiceParams, serviceSubnet string) Operator {
	return &apiServerBootOp{
		nodes:         nodes,
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
			"--entrypoint=/usr/local/kubernetes/bin/kube-apiserver",

			// TODO pass keys from CKE
			"--mount", "type=tmpfs,dst=/run/kubernetes",
		}
		if o.nodeIndex >= len(o.nodes) {
			return nil
		}

		target := o.nodes[o.nodeIndex]
		o.nodeIndex++

		var etcdServers []string
		for _, n := range o.nodes {
			etcdServers = append(etcdServers, "http://"+n.Address+":2379")
		}
		return runContainerCommand{target, o.agents[target.Address], "kube-apiserver", opts, o.apiServerParams(etcdServers, target.Address), extra}
	default:
		return nil
	}
}

func (o *apiServerBootOp) apiServerParams(etcdServers []string, addr string) ServiceParams {
	args := []string{
		"--etcd-servers=" + strings.Join(etcdServers, ","),

		// TODO use TLS
		"--insecure-bind-address=0.0.0.0",
		"--insecure-port=8080",

		"--advertise-address=" + addr,
		"--service-cluster-ip-range=" + o.serviceSubnet,
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
		opts := []string{
			"--entrypoint=/usr/local/kubernetes/bin/kube-controller-manager",
		}
		if o.nodeIndex >= len(o.nodes) {
			return nil
		}

		target := o.nodes[o.nodeIndex]
		o.nodeIndex++

		return runContainerCommand{target, o.agents[target.Address], "kube-controller-manager", opts, o.controllerManagerParams(), extra}
	default:
		return nil
	}
}

func (o *controllerManagerBootOp) controllerManagerParams() ServiceParams {
	args := []string{

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
		opts := []string{
			"--entrypoint=/usr/local/kubernetes/bin/kube-scheduler",
		}
		if o.nodeIndex >= len(o.nodes) {
			return nil
		}

		target := o.nodes[o.nodeIndex]
		o.nodeIndex++

		return runContainerCommand{target, o.agents[target.Address], "kube-scheduler", opts, o.schedulerParams(), extra}
	default:
		return nil
	}
}

func (o *schedulerBootOp) schedulerParams() ServiceParams {
	args := []string{
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

// RiversStopOp returns an Operator to bootstrap Scheduler cluster.
func RiversStopOp(nodes []*Node, agents map[string]Agent) Operator {
	return &riversStopOp{
		nodes:     nodes,
		agents:    agents,
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

	return stopContainerCommand{node, o.agents[node.Address], "kube-apiserver"}
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

// ControllerManagerStopOp returns an Operator to bootstrap Scheduler cluster.
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

// SchedulerStopOp returns an Operator to bootstrap Scheduler cluster.
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

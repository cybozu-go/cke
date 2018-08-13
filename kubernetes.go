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
		"--listen=" + "127.0.0.1:8080",
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

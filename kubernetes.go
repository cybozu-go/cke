package cke

import "strings"

type riversBootOp struct {
	nodes  []*Node
	agents map[string]Agent
	params ServiceParams
	step   int
}

// RiversBootOp returns an Operator to bootstrap rivers cluster.
func RiversBootOp(nodes []*Node, agents map[string]Agent, params ServiceParams) Operator {
	return &riversBootOp{
		nodes:  nodes,
		agents: agents,
		params: params,
		step:   0,
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
		o.step++
		var upstreams []string
		for _, n := range o.nodes {
			upstreams = append(upstreams, n.Address+":8080")
		}
		return runContainersCommand{o.nodes, o.agents, "rivers", opts, riversParams(upstreams), extra}
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

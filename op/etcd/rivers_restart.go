package etcd

import (
	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op"
	"github.com/cybozu-go/cke/op/common"
)

type riversRestartOp struct {
	nodes     []*cke.Node
	upstreams []*cke.Node
	params    cke.ServiceParams

	pulled   bool
	finished bool
}

// RiversRestartOp returns an Operator to restart rivers.
func RiversRestartOp(nodes, upstreams []*cke.Node, params cke.ServiceParams) cke.Operator {
	return &riversRestartOp{
		nodes:     nodes,
		upstreams: upstreams,
		params:    params,
	}
}

func (o *riversRestartOp) Name() string {
	return "etcd-rivers-restart"
}

func (o *riversRestartOp) NextCommand() cke.Commander {
	if !o.pulled {
		o.pulled = true
		return common.ImagePullCommand(o.nodes, cke.ToolsImage)
	}

	if !o.finished {
		o.finished = true
		return common.RunContainerCommand(o.nodes, op.EtcdRiversContainerName, cke.ToolsImage,
			common.WithParams(RiversParams(o.upstreams)),
			common.WithExtra(o.params),
			common.WithRestart())
	}
	return nil
}

func (o *riversRestartOp) Targets() []string {
	ips := make([]string, len(o.nodes))
	for i, n := range o.nodes {
		ips[i] = n.Address
	}
	return ips
}

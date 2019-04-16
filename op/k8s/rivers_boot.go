package k8s

import (
	"strings"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op"
	"github.com/cybozu-go/cke/op/common"
)

type riversBootOp struct {
	nodes     []*cke.Node
	upstreams []*cke.Node
	params    cke.ServiceParams
	step      int
}

// RiversBootOp returns an Operator to bootstrap rivers.
func RiversBootOp(nodes, upstreams []*cke.Node, params cke.ServiceParams) cke.Operator {
	return &riversBootOp{
		nodes:     nodes,
		upstreams: upstreams,
		params:    params,
	}
}

func (o *riversBootOp) Name() string {
	return "rivers-bootstrap"
}

func (o *riversBootOp) NextCommand() cke.Commander {
	switch o.step {
	case 0:
		o.step++
		return common.ImagePullCommand(o.nodes, cke.ToolsImage)
	case 1:
		o.step++
		return common.RunContainerCommand(o.nodes, op.RiversContainerName, cke.ToolsImage,
			common.WithParams(RiversParams(o.upstreams)),
			common.WithExtra(o.params))
	default:
		return nil
	}
}

// RiversParams returns parameters for rivers.
func RiversParams(upstreams []*cke.Node) cke.ServiceParams {
	var ups []string
	for _, n := range upstreams {
		ups = append(ups, n.Address+":6443")
	}
	args := []string{
		"rivers",
		"--upstreams=" + strings.Join(ups, ","),
		"--listen=" + "127.0.0.1:16443",
	}
	return cke.ServiceParams{ExtraArguments: args}
}

func (o *riversBootOp) Targets() []cke.Node {
	nodes := []cke.Node{}
	for _, v := range o.nodes {
		nodes = append(nodes, *v)
	}
	return nodes
}

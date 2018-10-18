package op

import (
	"strings"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/common"
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
		return common.MakeDirsCommand(o.nodes, []string{"/var/log/rivers"})
	case 2:
		o.step++
		return common.RunContainerCommand(o.nodes, riversContainerName, cke.ToolsImage,
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
	return cke.ServiceParams{
		ExtraArguments: args,
		ExtraBinds: []cke.Mount{
			{"/var/log/rivers", "/var/log/rivers", false, "", cke.LabelShared},
		},
	}
}

package etcd

import (
	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op"
	"github.com/cybozu-go/cke/op/common"
)

type markMemberOp struct {
	nodes    []*cke.Node
	executed bool
}

// MarkMemberOp returns an Operator to mark nodes as added members.
func MarkMemberOp(nodes []*cke.Node) cke.Operator {
	return &markMemberOp{
		nodes: nodes,
	}
}

func (o *markMemberOp) Name() string {
	return "etcd-mark-member"
}

func (o *markMemberOp) NextCommand() cke.Commander {
	if o.executed {
		return nil
	}
	o.executed = true

	return common.VolumeCreateCommand(o.nodes, op.EtcdAddedMemberVolumeName)
}

func (o *markMemberOp) Targets() []string {
	ips := make([]string, len(o.nodes))
	for i, n := range o.nodes {
		ips[i] = n.Address
	}
	return ips
}

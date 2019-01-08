package op

import (
	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op/common"
)

type etcdDestroyMemberOp struct {
	endpoints []string
	targets   []*cke.Node
	ids       []uint64
	params    cke.EtcdParams
	step      int
}

// EtcdDestroyMemberOp returns an Operator to remove and destroy a member.
func EtcdDestroyMemberOp(cp []*cke.Node, targets []*cke.Node, ids []uint64) cke.Operator {
	return &etcdDestroyMemberOp{
		endpoints: etcdEndpoints(cp),
		targets:   targets,
		ids:       ids,
	}
}

func (o *etcdDestroyMemberOp) Name() string {
	return "etcd-destroy-member"
}

func (o *etcdDestroyMemberOp) NextCommand() cke.Commander {
	switch o.step {
	case 0:
		o.step++
		return removeEtcdMemberCommand{o.endpoints, o.ids}
	case 1:
		o.step++
		return common.KillContainersCommand(o.targets, etcdContainerName)
	case 2:
		o.step++
		return common.VolumeRemoveCommand(o.targets, etcdVolumeName(o.params))
	case 3:
		o.step++
		return waitEtcdSyncCommand{o.endpoints, false}
	}
	return nil
}

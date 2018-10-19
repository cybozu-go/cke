package op

import (
	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/common"
)

type etcdRestartOp struct {
	cpNodes []*cke.Node
	target  *cke.Node
	params  cke.EtcdParams
	step    int
}

// EtcdRestartOp returns an Operator to restart an etcd member.
func EtcdRestartOp(cpNodes []*cke.Node, target *cke.Node, params cke.EtcdParams) cke.Operator {
	return &etcdRestartOp{
		cpNodes: cpNodes,
		target:  target,
		params:  params,
	}
}

func (o *etcdRestartOp) Name() string {
	return "etcd-restart"
}

func (o *etcdRestartOp) NextCommand() cke.Commander {
	switch o.step {
	case 0:
		o.step++
		return waitEtcdSyncCommand{etcdEndpoints(o.cpNodes), true}
	case 1:
		o.step++
		return common.ImagePullCommand([]*cke.Node{o.target}, cke.EtcdImage)
	case 2:
		o.step++
		return common.StopContainerCommand(o.target, etcdContainerName)
	case 3:
		o.step++
		opts := []string{
			"--mount",
			"type=volume,src=" + etcdVolumeName(o.params) + ",dst=/var/lib/etcd",
		}
		var initialCluster []string
		for _, n := range o.cpNodes {
			initialCluster = append(initialCluster, n.Address+"=https://"+n.Address+":2380")
		}
		return common.RunContainerCommand([]*cke.Node{o.target}, etcdContainerName, cke.EtcdImage,
			common.WithOpts(opts),
			common.WithParams(EtcdBuiltInParams(o.target, initialCluster, "new")),
			common.WithExtra(o.params.ServiceParams))
	}
	return nil
}

package k8s

import (
	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op"
	"github.com/cybozu-go/cke/op/common"
)

type schedulerRestartOp struct {
	nodes []*cke.Node

	cluster string
	params  cke.ServiceParams

	pulled   bool
	finished bool
}

// SchedulerRestartOp returns an Operator to restart kube-scheduler
func SchedulerRestartOp(nodes []*cke.Node, cluster string, params cke.ServiceParams) cke.Operator {
	return &schedulerRestartOp{
		nodes:   nodes,
		cluster: cluster,
		params:  params,
	}
}

func (o *schedulerRestartOp) Name() string {
	return "kube-scheduler-restart"
}

func (o *schedulerRestartOp) NextCommand() cke.Commander {
	if !o.pulled {
		o.pulled = true
		return common.ImagePullCommand(o.nodes, cke.HyperkubeImage)
	}

	if !o.finished {
		o.finished = true
		return common.RunContainerCommand(o.nodes, op.KubeSchedulerContainerName, cke.HyperkubeImage,
			common.WithParams(SchedulerParams()),
			common.WithExtra(o.params),
			common.WithRestart())
	}
	return nil
}

func (o *schedulerRestartOp) Nodes() []string {
	ips := []string{}
	for _, n := range o.nodes {
		ips = append(ips, n.Nodename())
	}
	return ips
}

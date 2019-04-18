package op

import (
	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op/common"
)

type containerStopOp struct {
	nodes    []*cke.Node
	name     string
	executed bool
}

func (o *containerStopOp) Name() string {
	return "stop-" + o.name
}

func (o *containerStopOp) NextCommand() cke.Commander {
	if o.executed {
		return nil
	}
	o.executed = true
	return common.KillContainersCommand(o.nodes, o.name)
}

func (o *containerStopOp) Nodes() []string {
	ips := []string{}
	for _, n := range o.nodes {
		ips = append(ips, n.Nodename())
	}
	return ips
}

// APIServerStopOp returns an Operator to stop API server
func APIServerStopOp(nodes []*cke.Node) cke.Operator {
	return &containerStopOp{
		nodes: nodes,
		name:  KubeAPIServerContainerName,
	}
}

// ControllerManagerStopOp returns an Operator to stop kube-controller-manager
func ControllerManagerStopOp(nodes []*cke.Node) cke.Operator {
	return &containerStopOp{
		nodes: nodes,
		name:  KubeControllerManagerContainerName,
	}
}

// SchedulerStopOp returns an Operator to stop kube-scheduler
func SchedulerStopOp(nodes []*cke.Node) cke.Operator {
	return &containerStopOp{
		nodes: nodes,
		name:  KubeSchedulerContainerName,
	}
}

// EtcdStopOp returns an Operator to stop etcd
func EtcdStopOp(nodes []*cke.Node) cke.Operator {
	return &containerStopOp{
		nodes: nodes,
		name:  EtcdContainerName,
	}
}

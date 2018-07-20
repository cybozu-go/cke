package cke

import (
	"context"
	"strings"

	"github.com/coreos/etcd/etcdserver/etcdserverpb"
)

const (
	defaultEtcdVolumeName = "etcd-cke"
)

func etcdVolumeName(c *Cluster) string {
	if len(c.Options.Etcd.VolumeName) == 0 {
		return defaultEtcdVolumeName
	}
	return c.Options.Etcd.VolumeName
}

type etcdBootOp struct {
	nodes   []*Node
	agents  map[string]Agent
	volname string
	extra   ServiceParams
	step    int
	cpIndex int
}

// EtcdBootOp returns an Operator to bootstrap etcd cluster.
func EtcdBootOp(nodes []*Node, agents map[string]Agent, volname string, extra ServiceParams) Operator {
	return &etcdBootOp{
		nodes:   nodes,
		agents:  agents,
		volname: volname,
		extra:   extra,
		step:    0,
		cpIndex: 0,
	}
}

func (o *etcdBootOp) Name() string {
	return "etcd-bootstrap"
}

func (o *etcdBootOp) NextCommand() Commander {
	switch o.step {
	case 0:
		o.step++
		return imagePullCommand{o.nodes, o.agents, "etcd"}
	case 1:
		o.step++
		return volumeCreateCommand{o.nodes, o.agents, o.volname}
	case 2:
		node := o.nodes[o.cpIndex]
		agent := o.agents[node.Address]

		o.cpIndex++
		if o.cpIndex == len(o.nodes) {
			o.step++
		}
		opts := []string{
			"--mount",
			"type=volume,src=" + o.volname + ",dst=/var/lib/etcd",
		}
		return runContainerCommand{node, agent, "etcd", opts, o.params(node), o.extra}
	default:
		return nil
	}
}

func (o *etcdBootOp) params(node *Node) ServiceParams {
	var initialCluster []string
	for _, n := range o.nodes {
		initialCluster = append(initialCluster, n.Address+"=http://"+n.Address+":2380")
	}
	args := []string{
		"--name=" + node.Address,
		"--listen-peer-urls=http://0.0.0.0:2380",
		"--listen-client-urls=http://0.0.0.0:2379",
		"--initial-advertise-peer-urls=http://" + node.Address + ":2380",
		"--advertise-client-urls=http://" + node.Address + ":2379",
		"--initial-cluster=" + strings.Join(initialCluster, ","),
		"--initial-cluster-token=cke",
		"--initial-cluster-state=new",
		"--enable-v2=false",
		"--enable-pprof=true",
		"--auto-compaction-mode=periodic",
		"--auto-compaction-retention=24",
	}
	params := ServiceParams{
		ExtraArguments: args,
	}

	return params
}

func (o *etcdBootOp) Cleanup(ctx context.Context) error {
	return nil
}

// EtcdAddMemberOp returns an Operator to add member to etcd cluster.
func EtcdAddMemberOp(nodes []*Node, agents map[string]Agent, volname string, extra ServiceParams) Operator {
	return &etcdAddMemberOp{
		nodes:   nodes,
		agents:  agents,
		volname: volname,
		extra:   extra,
		step:    0,
	}
}

type etcdAddMemberOp struct {
	nodes   []*Node
	agents  map[string]Agent
	volname string
	extra   ServiceParams
	step    int
}

func (o *etcdAddMemberOp) Name() string {
	return "etcd-add-member"
}

func (o *etcdAddMemberOp) NextCommand() Commander {
	// TODO return next command
	return nil
}

func (o *etcdAddMemberOp) Cleanup(ctx context.Context) error {
	return nil
}

// EtcdRemoveMemberOp returns an Operator to remove member from etcd cluster.
func EtcdRemoveMemberOp(nodes []*Node, unknown map[string]*etcdserverpb.Member, agents map[string]Agent, volname string, extra ServiceParams) Operator {
	return &etcdRemoveMemberOp{
		nodes:         nodes,
		unknownMember: unknown,
		agents:        agents,
		volname:       volname,
		extra:         extra,
		step:          0,
	}
}

type etcdRemoveMemberOp struct {
	nodes         []*Node
	unknownMember map[string]*etcdserverpb.Member
	agents        map[string]Agent
	volname       string
	extra         ServiceParams
	step          int
}

func (o *etcdRemoveMemberOp) Name() string {
	return "etcd-remove-member"
}

func (o *etcdRemoveMemberOp) NextCommand() Commander {
	// TODO return next command
	return nil
}
func (o *etcdRemoveMemberOp) Cleanup(ctx context.Context) error {
	return nil
}

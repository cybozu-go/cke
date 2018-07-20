package cke

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/coreos/etcd/clientv3"
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
		return runContainerCommand{node, agent, "etcd", opts, etcdParams(o.nodes, node, "new"), o.extra}
	default:
		return nil
	}
}

func etcdParams(cpNodes []*Node, node *Node, state string) ServiceParams {
	var initialCluster []string
	for _, n := range cpNodes {
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
		"--initial-cluster-state=" + state,
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
func EtcdAddMemberOp(cpNodes []*Node, targetNodes []*Node, endpoints []string, agents map[string]Agent, volname string, extra ServiceParams) Operator {
	return &etcdAddMemberOp{
		cpNodes:     cpNodes,
		targetNodes: targetNodes,
		endpoints:   endpoints,
		agents:      agents,
		volname:     volname,
		extra:       extra,
		step:        0,
		nodeIndex:   0,
	}
}

type etcdAddMemberOp struct {
	cpNodes     []*Node
	targetNodes []*Node
	endpoints   []string
	agents      map[string]Agent
	volname     string
	extra       ServiceParams
	step        int
	nodeIndex   int
}

func (o *etcdAddMemberOp) Name() string {
	return "etcd-add-member"
}

func (o *etcdAddMemberOp) NextCommand() Commander {
	switch o.step {
	case 0:
		o.step++
		return imagePullCommand{o.targetNodes, o.agents, "etcd"}
	case 1:
		o.step++
		return volumeCreateCommand{o.targetNodes, o.agents, o.volname}
	case 2:
		o.step++
		return addEtcdMemberCommand{o.targetNodes, o.endpoints}
	case 3:
		node := o.targetNodes[o.nodeIndex]
		agent := o.agents[node.Address]

		o.nodeIndex++
		if o.nodeIndex == len(o.targetNodes) {
			o.step++
		}
		opts := []string{
			"--mount",
			"type=volume,src=" + o.volname + ",dst=/var/lib/etcd",
		}
		return runContainerCommand{node, agent, "etcd", opts, etcdParams(o.cpNodes, node, "existing"), o.extra}
	default:
		return nil
	}
	return nil
}

type addEtcdMemberCommand struct {
	nodes     []*Node
	endpoints []string
}

func (c addEtcdMemberCommand) Run(ctx context.Context) error {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   c.endpoints,
		DialTimeout: 2 * time.Second,
	})
	if err != nil {
		return err
	}

	defer cli.Close()

	for _, n := range c.nodes {
		_, err := cli.MemberAdd(ctx, []string{fmt.Sprintf("http://%s:2380", n.Address)})
		if err != nil {
			return err
		}
	}

	return nil
}

func (c addEtcdMemberCommand) Command() Command {
	return Command{
		Name: "add-etcd-member",
	}
}

func (o *etcdAddMemberOp) Cleanup(ctx context.Context) error {
	// TODO: remove member from etcd cluster
	// TODO: stop etcd container
	// TODO: remove etcd data volume
	// TODO: remove etcd container image

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

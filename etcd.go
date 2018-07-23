package cke

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/etcdserver/etcdserverpb"
)

const (
	defaultEtcdVolumeName = "etcd-cke"
)

func etcdVolumeName(e EtcdParams) string {
	if len(e.VolumeName) == 0 {
		return defaultEtcdVolumeName
	}
	return e.VolumeName
}

type etcdBootOp struct {
	nodes   []*Node
	agents  map[string]Agent
	params  EtcdParams
	step    int
	cpIndex int
}

// EtcdBootOp returns an Operator to bootstrap etcd cluster.
func EtcdBootOp(nodes []*Node, agents map[string]Agent, params EtcdParams) Operator {
	return &etcdBootOp{
		nodes:   nodes,
		agents:  agents,
		params:  params,
		step:    0,
		cpIndex: 0,
	}
}

func (o *etcdBootOp) Name() string {
	return "etcd-bootstrap"
}

func (o *etcdBootOp) NextCommand() Commander {
	volname := etcdVolumeName(o.params)
	extra := o.params.ServiceParams

	switch o.step {
	case 0:
		o.step++
		return imagePullCommand{o.nodes, o.agents, "etcd"}
	case 1:
		o.step++
		return volumeCreateCommand{o.nodes, o.agents, volname}
	case 2:
		node := o.nodes[o.cpIndex]
		agent := o.agents[node.Address]

		o.cpIndex++
		if o.cpIndex == len(o.nodes) {
			o.step++
		}
		opts := []string{
			"--mount",
			"type=volume,src=" + volname + ",dst=/var/lib/etcd",
		}
		var initialCluster []string
		for _, n := range o.nodes {
			initialCluster = append(initialCluster, n.Address+"=http://"+n.Address+":2380")
		}
		return runContainerCommand{node, agent, "etcd", opts, etcdParams(node, initialCluster, "new"), extra}
	default:
		return nil
	}
}

func etcdParams(node *Node, initialCluster []string, state string) ServiceParams {
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
func EtcdAddMemberOp(endpoints []string, targetNodes []*Node, agents map[string]Agent, params EtcdParams) Operator {
	return &etcdAddMemberOp{
		endpoints:   endpoints,
		targetNodes: targetNodes,
		agents:      agents,
		params:      params,
		step:        0,
		nodeIndex:   0,
	}
}

type etcdAddMemberOp struct {
	endpoints   []string
	targetNodes []*Node
	agents      map[string]Agent
	params      EtcdParams
	step        int
	nodeIndex   int
}

func (o *etcdAddMemberOp) Name() string {
	return "etcd-add-member"
}

func (o *etcdAddMemberOp) NextCommand() Commander {
	volname := etcdVolumeName(o.params)
	extra := o.params.ServiceParams

	switch o.step {
	case 0:
		o.step++
		return imagePullCommand{o.targetNodes, o.agents, "etcd"}
	case 1:
		o.step++
		return volumeCreateCommand{o.targetNodes, o.agents, volname}
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
			"type=volume,src=" + volname + ",dst=/var/lib/etcd",
		}
		var initialCluster []string
		// TODO
		return runContainerCommand{node, agent, "etcd", opts, etcdParams(node, initialCluster, "existing"), extra}
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

type removeEtcdMemberCommand struct {
	endpoints []string
	ids       []uint64
}

func (c removeEtcdMemberCommand) Run(ctx context.Context) error {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   c.endpoints,
		DialTimeout: 2 * time.Second,
	})
	if err != nil {
		return err
	}
	defer cli.Close()

	for _, id := range c.ids {
		_, err := cli.MemberRemove(ctx, id)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c removeEtcdMemberCommand) Command() Command {
	idStrs := make([]string, len(c.ids))
	for i, id := range c.ids {
		idStrs[i] = strconv.FormatUint(id, 10)
	}
	return Command{
		Name:   "remove-etcd-member",
		Target: strings.Join(idStrs, ","),
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
func EtcdRemoveMemberOp(endpoints []string, targets map[string]*etcdserverpb.Member) Operator {
	return &etcdRemoveMemberOp{
		endpoints: endpoints,
		targets:   targets,
	}
}

type etcdRemoveMemberOp struct {
	endpoints []string
	targets   map[string]*etcdserverpb.Member
	executed  bool
}

func (o *etcdRemoveMemberOp) Name() string {
	return "etcd-remove-member"
}

func (o *etcdRemoveMemberOp) NextCommand() Commander {
	if o.executed {
		return nil
	}
	o.executed = true

	var ids []uint64
	for _, v := range o.targets {
		ids = append(ids, v.ID)
	}
	return removeEtcdMemberCommand{o.endpoints, ids}
}
func (o *etcdRemoveMemberOp) Cleanup(ctx context.Context) error {
	return nil
}

// EtcdDestroyMemberOp create new etcdDestroyMemberOp instance
func EtcdDestroyMemberOp(endpoints []string, targets []*Node, agents map[string]Agent, members map[string]*etcdserverpb.Member) Operator {
	return &etcdDestroyMemberOp{
		endpoints: endpoints,
		targets:   targets,
		agents:    agents,
		members:   members,
	}
	return nil
}

type etcdDestroyMemberOp struct {
	endpoints []string
	targets   []*Node
	agents    map[string]Agent
	members   map[string]*etcdserverpb.Member
}

func (o *etcdDestroyMemberOp) Name() string {
	return "etcd-destroy-member"
}

func (o *etcdDestroyMemberOp) NextCommand() Commander {
	// TODO destroy member from etcd cluster
	return nil
}
func (o *etcdDestroyMemberOp) Cleanup(ctx context.Context) error {
	return nil
}

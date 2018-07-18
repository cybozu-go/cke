package cke

import (
	"context"
	"strings"
)

const (
	defaultEtcdDataDir = "/var/lib/etcd-cke"
)

func etcdDataDir(c *Cluster) string {
	if len(c.Options.Etcd.DataDir) == 0 {
		return defaultEtcdDataDir
	}
	return c.Options.Etcd.DataDir
}

type etcdBootOperator struct {
	nodes     []*Node
	agents    map[string]Agent
	dataDir   string
	extra     ServiceParams
	step      int
	bootIndex int
}

func newEtcdBootOperator(nodes []*Node, agents map[string]Agent, dataDir string, extra ServiceParams) Operator {
	return &etcdBootOperator{
		nodes:     nodes,
		agents:    agents,
		dataDir:   dataDir,
		extra:     extra,
		step:      0,
		bootIndex: 0,
	}
}

func (o etcdBootOperator) Name() string {
	return "etcd-bootstrap"
}

func (o *etcdBootOperator) NextCommand() Commander {
	switch o.step {
	case 0:
		o.step++
		return makeDirCommand{o.nodes, o.agents, o.dataDir}
	case 1:
		o.step++
		return imagePullCommand{o.nodes, o.agents, "etcd"}
	case 2:
		node := o.nodes[o.bootIndex]
		agent := o.agents[node.Address]

		o.bootIndex++
		if o.bootIndex == len(o.nodes) {
			o.step++
		}
		return runContainerCommand{node, agent, "etcd", nil, o.params(node), o.extra}
	default:
		return nil
	}
}

func (o *etcdBootOperator) params(node *Node) ServiceParams {
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
		ExtraBinds: []Mount{
			{
				Source:      o.dataDir,
				Destination: "/var/lib/etcd",
				ReadOnly:    false,
			},
		},
		ExtraArguments: args,
	}

	return params
}

func (o etcdBootOperator) Cleanup(ctx context.Context) error {
	return nil
}

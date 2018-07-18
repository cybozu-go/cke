package cke

import (
	"context"
)

const (
	defaultEtcdDataDir = "/var/lib/etcd"
)

func etcdDataDir(c *Cluster) string {
	if len(c.Options.Etcd.DataDir) == 0 {
		return defaultEtcdDataDir
	}
	return c.Options.Etcd.DataDir
}

type etcdBootOperator struct {
	nodes   []*Node
	agents  map[string]Agent
	dataDir string
	step    int
}

func newEtcdBootOperator(nodes []*Node, agents map[string]Agent, dataDir string) Operator {
	return &etcdBootOperator{
		nodes:   nodes,
		agents:  agents,
		dataDir: dataDir,
		step:    0,
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
	default:
		return nil
	}
}

func (o etcdBootOperator) Cleanup(ctx context.Context) error {
	return nil
}

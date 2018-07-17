package cke

import (
	"context"

	"github.com/cybozu-go/cmd"
)

const (
	defaultEtcdDataDir = "/var/lib/etcd"
)

type EtcdBootOperator struct {
	nodes   []*Node
	agents  map[string]Agent
	dataDir string
	step    int
}

func etcdDataDir(c *Cluster) string {
	if len(c.Options.Etcd.DataDir) == 0 {
		return defaultEtcdDataDir
	}
	return c.Options.Etcd.DataDir
}

func NewEtcdBootOperator(nodes []*Node, agents map[string]Agent, dataDir string) *EtcdBootOperator {
	return &EtcdBootOperator{
		nodes:   nodes,
		agents:  agents,
		dataDir: dataDir,
		step:    0,
	}
}

func (o EtcdBootOperator) Name() string {
	return "etcd-bootstrap"
}

func (o *EtcdBootOperator) NextCommand() Commander {
	switch o.step {
	case 0:
		o.step++
		return MakeDirCommand{o.nodes, o.agents, o.dataDir}
	default:
		return nil
	}
}

func (o EtcdBootOperator) Cleanup(ctx context.Context) error {
	return nil
}

type MakeDirCommand struct {
	nodes     []*Node
	agents    map[string]Agent
	targetDir string
}

func (c MakeDirCommand) Run(ctx context.Context) error {
	env := cmd.NewEnvironment(ctx)
	for _, n := range c.nodes {
		a := c.agents[n.Address]
		env.Go(func(ctx context.Context) error {
			_, _, err := a.Run("mkdir -p " + c.targetDir)
			return err
		})
	}
	env.Stop()
	return env.Wait()
}

func (c MakeDirCommand) Command() Command {
	return Command{
		Name:   "mkdir",
		Target: c.targetDir,
		Detail: "",
	}
}

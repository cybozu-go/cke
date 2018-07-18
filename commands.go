package cke

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/cybozu-go/cmd"
)

// Command represents some command
type Command struct {
	Name   string `json:"name"`
	Target string `json:"target"`
	Detail string `json:"detail"`
}

// String implements fmt.Stringer
func (c Command) String() string {
	return fmt.Sprintf("%s@%s: %s", c.Name, c.Target, c.Detail)
}

// Commander is a single step to proceed an operation
type Commander interface {
	// Run executes the command
	Run(ctx context.Context) error
	// Command returns the command information
	Command() Command
}

type makeDirCommand struct {
	nodes     []*Node
	agents    map[string]Agent
	targetDir string
}

func (c makeDirCommand) Run(ctx context.Context) error {
	env := cmd.NewEnvironment(ctx)
	binds := []Mount{{
		Source:      filepath.Dir(c.targetDir),
		Destination: filepath.Join("/mnt", filepath.Dir(c.targetDir)),
	}}
	mkdirCommand := "mkdir -p " + filepath.Join("/mnt", c.targetDir)
	for _, n := range c.nodes {
		ctr := Docker("tools", c.agents[n.Address])
		env.Go(func(ctx context.Context) error {
			return ctr.Run(binds, mkdirCommand)
		})
	}
	env.Stop()
	return env.Wait()
}

func (c makeDirCommand) Command() Command {
	return Command{
		Name:   "mkdir",
		Target: c.targetDir,
	}
}

type imagePullCommand struct {
	nodes  []*Node
	agents map[string]Agent
	name   string
}

func (c imagePullCommand) Run(ctx context.Context) error {
	env := cmd.NewEnvironment(ctx)
	for _, n := range c.nodes {
		ctr := Docker(c.name, c.agents[n.Address])
		env.Go(func(ctx context.Context) error {
			return ctr.PullImage()
		})
	}
	env.Stop()
	return env.Wait()
}

func (c imagePullCommand) Command() Command {
	return Command{
		Name:   "image-pull",
		Target: c.name,
		Detail: Image(c.name),
	}
}

type runContainersCommand struct {
	nodes  []*Node
	agents map[string]Agent
	name   string
	opts   []string
	params ServiceParams
	extra  ServiceParams
}

func (c runContainersCommand) Run(ctx context.Context) error {
	env := cmd.NewEnvironment(ctx)
	for _, n := range c.nodes {
		ctr := Docker(c.name, c.agents[n.Address])
		env.Go(func(ctx context.Context) error {
			return ctr.RunSystem(c.opts, c.params, c.extra)
		})
	}
	env.Stop()
	return env.Wait()
}

func (c runContainersCommand) Command() Command {
	return Command{
		Name:   "run-containers",
		Target: c.name,
	}
}

type runContainerCommand struct {
	node   *Node
	agent  Agent
	name   string
	opts   []string
	params ServiceParams
	extra  ServiceParams
}

func (c runContainerCommand) Run(ctx context.Context) error {
	ctr := Docker(c.name, c.agent)
	return ctr.RunSystem(c.opts, c.params, c.extra)
}

func (c runContainerCommand) Command() Command {
	return Command{
		Name:   "run-container",
		Target: c.name,
	}
}

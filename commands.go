package cke

import (
	"context"
	"fmt"

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

func (c makeDirCommand) Command() Command {
	return Command{
		Name:   "mkdir",
		Target: c.targetDir,
		Detail: "",
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

package cke

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

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
	if len(c.Detail) > 0 {
		return fmt.Sprintf("%s %s: %s", c.Name, c.Target, c.Detail)
	}
	return fmt.Sprintf("%s %s", c.Name, c.Target)
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
		ce := Docker(c.agents[n.Address])
		env.Go(func(ctx context.Context) error {
			return ce.Run("tools", binds, mkdirCommand)
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

type makeFileCommand struct {
	nodes  []*Node
	agents map[string]Agent
	source string
	target string
}

func (c makeFileCommand) Run(ctx context.Context) error {
	env := cmd.NewEnvironment(ctx)
	targetDir := filepath.Dir(c.target)
	binds := []Mount{{
		Source:      targetDir,
		Destination: filepath.Join("/mnt", targetDir),
	}}
	mkdirCommand := "mkdir -p " + filepath.Join("/mnt", targetDir)
	ddCommand := "dd of=" + filepath.Join("/mnt", c.target)
	for _, n := range c.nodes {
		ce := Docker(c.agents[n.Address])
		env.Go(func(ctx context.Context) error {
			err := ce.Run("tools", binds, mkdirCommand)
			if err != nil {
				return err
			}
			return ce.RunWithInput("tools", binds, ddCommand, c.source)
		})
	}
	env.Stop()
	return env.Wait()
}

func (c makeFileCommand) Command() Command {
	return Command{
		Name:   "make-file",
		Target: c.target,
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
		ce := Docker(c.agents[n.Address])
		env.Go(func(ctx context.Context) error {
			return ce.PullImage(c.name)
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

type volumeCreateCommand struct {
	nodes   []*Node
	agents  map[string]Agent
	volname string
}

func (c volumeCreateCommand) Run(ctx context.Context) error {
	env := cmd.NewEnvironment(ctx)
	for _, n := range c.nodes {
		ce := Docker(c.agents[n.Address])
		env.Go(func(ctx context.Context) error {
			return ce.VolumeCreate(c.volname)
		})
	}
	env.Stop()
	return env.Wait()
}

func (c volumeCreateCommand) Command() Command {
	targets := make([]string, len(c.nodes))
	for i, n := range c.nodes {
		targets[i] = n.Address
	}
	return Command{
		Name:   "volume-create",
		Target: strings.Join(targets, ","),
		Detail: c.volname,
	}
}

type volumeRemoveCommand struct {
	nodes   []*Node
	agents  map[string]Agent
	volname string
}

func (c volumeRemoveCommand) Run(ctx context.Context) error {
	env := cmd.NewEnvironment(ctx)
	for _, n := range c.nodes {
		ce := Docker(c.agents[n.Address])
		env.Go(func(ctx context.Context) error {
			exists, err := ce.VolumeExists(c.volname)
			if err != nil {
				return err
			}
			if exists {
				return ce.VolumeRemove(c.volname)
			}
			return nil
		})
	}
	env.Stop()
	return env.Wait()
}

func (c volumeRemoveCommand) Command() Command {
	targets := make([]string, len(c.nodes))
	for i, n := range c.nodes {
		targets[i] = n.Address
	}
	return Command{
		Name:   "volume-remove",
		Target: strings.Join(targets, ","),
		Detail: c.volname,
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
	ce := Docker(c.agent)
	return ce.RunSystem(c.name, c.opts, c.params, c.extra)
}

func (c runContainerCommand) Command() Command {
	return Command{
		Name:   "run-container",
		Target: c.node.Address,
		Detail: c.name,
	}
}

type stopContainerCommand struct {
	node  *Node
	agent Agent
	name  string
}

func (c stopContainerCommand) Run(ctx context.Context) error {
	ce := Docker(c.agent)
	exists, err := ce.Exists(c.name)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	err = ce.Stop(c.name)
	if err != nil {
		return err
	}
	// gofail: var dockerAfterContainerStop struct{}
	return ce.Remove(c.name)
}

func (c stopContainerCommand) Command() Command {
	return Command{
		Name:   "stop-container",
		Target: c.node.Address,
		Detail: c.name,
	}
}

package common

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cmd"
)

type makeDirsCommand struct {
	nodes []*cke.Node
	dirs  []string
}

// MakeDirsCommand returns a Commander to make directories on nodes.
func MakeDirsCommand(nodes []*cke.Node, dirs []string) cke.Commander {
	return makeDirsCommand{nodes, dirs}
}

func (c makeDirsCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	bindMap := make(map[string]cke.Mount)
	dests := make([]string, len(c.dirs))
	for i, d := range c.dirs {
		dests[i] = filepath.Join("/mnt", d)

		parentDir := filepath.Dir(d)
		if _, ok := bindMap[parentDir]; ok {
			continue
		}
		bindMap[parentDir] = cke.Mount{
			Source:      parentDir,
			Destination: filepath.Join("/mnt", parentDir),
			Label:       cke.LabelPrivate,
		}
	}
	binds := make([]cke.Mount, 0, len(bindMap))
	for _, m := range bindMap {
		binds = append(binds, m)
	}

	arg := "/usr/local/cke-tools/bin/make_directories " + strings.Join(dests, " ")

	env := cmd.NewEnvironment(ctx)
	for _, n := range c.nodes {
		ce := inf.Engine(n.Address)
		env.Go(func(ctx context.Context) error {
			return ce.Run(cke.ToolsImage, binds, arg)
		})
	}
	env.Stop()
	return env.Wait()
}

func (c makeDirsCommand) Command() cke.Command {
	return cke.Command{
		Name:   "make-dirs",
		Target: strings.Join(c.dirs, " "),
	}
}

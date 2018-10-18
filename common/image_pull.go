package common

import (
	"context"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cmd"
)

type imagePullCommand struct {
	nodes []*cke.Node
	img   cke.Image
}

// ImagePullCommand returns a Commander to pull an image on nodes.
func ImagePullCommand(nodes []*cke.Node, img cke.Image) cke.Commander {
	return imagePullCommand{nodes, img}
}

func (c imagePullCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	env := cmd.NewEnvironment(ctx)
	for _, n := range c.nodes {
		ce := inf.Engine(n.Address)
		env.Go(func(ctx context.Context) error {
			return ce.PullImage(c.img)
		})
	}
	env.Stop()
	return env.Wait()
}

func (c imagePullCommand) Command() cke.Command {
	return cke.Command{
		Name:   "image-pull",
		Target: c.img.Name(),
	}
}

package cli

import (
	"context"
	"flag"
	"os"

	"github.com/cybozu-go/cke"
	"github.com/google/subcommands"
	"gopkg.in/yaml.v2"
)

type cluster struct{}

func (c cluster) SetFlags(f *flag.FlagSet) {}

func (c cluster) Execute(ctx context.Context, f *flag.FlagSet) subcommands.ExitStatus {
	newc := NewCommander(f, "cluster")
	newc.Register(clusterSetCommand(), "")
	newc.Register(clusterGetCommand(), "")
	return newc.Execute(ctx)
}

// ClusterCommand implements "cluster" subcommand
func ClusterCommand() subcommands.Command {
	return subcmd{
		cluster{},
		"cluster",
		"manage cluster configuration",
		"cluster ACTION ...",
	}
}

type clusterSet struct{}

func (c clusterSet) SetFlags(f *flag.FlagSet) {}

func (c clusterSet) Execute(ctx context.Context, f *flag.FlagSet) subcommands.ExitStatus {
	if f.NArg() != 1 {
		f.Usage()
		return subcommands.ExitUsageError
	}
	fileName := f.Arg(0)

	r, err := os.Open(fileName)
	if err != nil {
		return handleError(err)
	}
	defer r.Close()

	cfg := new(cke.Cluster)
	err = yaml.NewDecoder(r).Decode(cfg)
	if err != nil {
		return handleError(err)
	}
	err = cfg.Validate()
	if err != nil {
		return handleError(err)
	}

	constraints, err := storage.GetConstraints(ctx)
	switch err {
	case cke.ErrNotFound:
		constraints = cke.DefaultConstraints()
		fallthrough
	case nil:
		err = constraints.Check(cfg)
		if err != nil {
			return handleError(err)
		}
	default:
		return handleError(err)
	}

	// Put on etcd
	err = storage.PutCluster(ctx, cfg)
	return handleError(err)
}

func clusterSetCommand() subcommands.Command {
	return subcmd{
		clusterSet{},
		"set",
		"set cluster configuration",
		"cluster set FILE",
	}
}

type clusterGet struct{}

func (c clusterGet) SetFlags(f *flag.FlagSet) {}

func (c clusterGet) Execute(ctx context.Context, f *flag.FlagSet) subcommands.ExitStatus {
	cfg, err := storage.GetCluster(ctx)
	if err != nil {
		return handleError(err)
	}

	err = yaml.NewEncoder(os.Stdout).Encode(cfg)
	return handleError(err)
}

func clusterGetCommand() subcommands.Command {
	return subcmd{
		clusterGet{},
		"get",
		"get cluster configuration",
		"cluster get",
	}
}

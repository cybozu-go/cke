package cli

import (
	"context"
	"encoding/json"
	"flag"
	"os"
	"strconv"

	"github.com/cybozu-go/cke"
	"github.com/google/subcommands"
)

type constraints struct{}

func (c constraints) SetFlags(f *flag.FlagSet) {}

func (c constraints) Execute(ctx context.Context, f *flag.FlagSet) subcommands.ExitStatus {
	newc := NewCommander(f, "constraints")
	newc.Register(constraintsShowCommand(), "")
	newc.Register(constraintsSetCommand(), "")
	return newc.Execute(ctx)
}

// ConstraintsCommand implements "constraints" subcommand
func ConstraintsCommand() subcommands.Command {
	return subcmd{
		constraints{},
		"constraints",
		"set/show constraints on the cluster configuration",
		"constraints ACTION ...",
	}
}

type constraintsShow struct{}

func (c constraintsShow) SetFlags(f *flag.FlagSet) {}

func (c constraintsShow) Execute(ctx context.Context, f *flag.FlagSet) subcommands.ExitStatus {
	ctr, err := storage.GetConstraints(ctx)
	if err != nil {
		return handleError(err)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "    ")
	err = enc.Encode(ctr)
	return handleError(err)
}

func constraintsShowCommand() subcommands.Command {
	return subcmd{
		constraintsShow{},
		"show",
		"show constraints",
		"constraints show",
	}
}

type constraintsSet struct{}

func (c constraintsSet) SetFlags(f *flag.FlagSet) {}

func (c constraintsSet) Execute(ctx context.Context, f *flag.FlagSet) subcommands.ExitStatus {
	ctr, err := storage.GetConstraints(ctx)
	switch err {
	case cke.ErrNotFound:
		ctr = cke.DefaultConstraints()
	case nil:
	default:
		return handleError(err)
	}

	if f.NArg() != 2 {
		f.Usage()
		return subcommands.ExitUsageError
	}

	name := f.Arg(0)
	value := f.Arg(1)
	intVal, err := strconv.Atoi(value)
	if err != nil {
		return handleError(err)
	}

	switch name {
	case "control-plane-count":
		ctr.ControlPlaneCount = intVal
	case "minimum-workers":
		ctr.MinimumWorkers = intVal
	case "maximum-workers":
		ctr.MaximumWorkers = intVal
	default:
		f.Usage()
		return subcommands.ExitUsageError
	}

	err = storage.PutConstraints(ctx, ctr)
	return handleError(err)
}

func constraintsSetCommand() subcommands.Command {
	return subcmd{
		constraintsSet{},
		"set",
		"set a constraint",
		`constraints set NAME VALUE

NAME is one of:
    - control-plane-count
    - minimum-workers
    - maximum-workers
`,
	}
}

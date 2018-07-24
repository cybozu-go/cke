package cli

import (
	"context"
	"flag"
	"fmt"

	"github.com/google/subcommands"
)

type leader struct{}

func (c leader) SetFlags(f *flag.FlagSet) {}

func (c leader) Execute(ctx context.Context, f *flag.FlagSet) subcommands.ExitStatus {
	leader, err := storage.GetLeaderHostname(ctx)
	if err != nil {
		return handleError(err)
	}

	fmt.Println(leader)
	return handleError(nil)
}

// LeaderCommand implements "leader" subcommand
func LeaderCommand() subcommands.Command {
	return subcmd{
		leader{},
		"leader",
		"show the current leader",
		"leader",
	}
}

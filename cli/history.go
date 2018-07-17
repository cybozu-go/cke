package cli

import (
	"context"
	"encoding/json"
	"flag"
	"os"

	"github.com/google/subcommands"
)

type history struct {
	count int
}

func (c *history) SetFlags(f *flag.FlagSet) {
	f.IntVar(&c.count, "n", 0, "limit the number of operations to show")
}

func (c *history) Execute(ctx context.Context, f *flag.FlagSet) subcommands.ExitStatus {
	records, err := storage.GetRecords(ctx, int64(c.count))
	if err != nil {
		return handleError(err)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "    ")
	for _, r := range records {
		err = enc.Encode(r)
		if err != nil {
			return handleError(err)
		}
	}

	return handleError(nil)
}

// HistoryCommand implements "history" subcommand
func HistoryCommand() subcommands.Command {
	return subcmd{
		&history{},
		"history",
		"show the operation history",
		"history [-n COUNT]",
	}
}

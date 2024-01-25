package cmd

import (
	"context"
	"encoding/json"
	"os"

	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var repairQueueListCmd = &cobra.Command{
	Use:   "list",
	Short: "list the entries in the repair queue",
	Long: `List the entries in the repair queue.

The output is a list of RepairQueueEntry formatted in JSON.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		well.Go(func(ctx context.Context) error {
			entries, err := storage.GetRepairsEntries(ctx)
			if err != nil {
				return err
			}

			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "    ")
			return enc.Encode(entries)
		})
		well.Stop()
		return well.Wait()
	},
}

func init() {
	repairQueueCmd.AddCommand(repairQueueListCmd)
}

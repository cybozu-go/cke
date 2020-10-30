package cmd

import (
	"context"
	"encoding/json"
	"os"

	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var rebootQueueListCmd = &cobra.Command{
	Use:   "list",
	Short: "list the entries in the reboot queue",
	Long: `List the entries in the reboot queue.

The output is a list of RebootQueueEntry formatted in JSON.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		well.Go(func(ctx context.Context) error {
			entries, err := storage.GetRebootsEntries(ctx)
			if err != nil {
				return err
			}

			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "    ")
			if err := enc.Encode(entries); err != nil {
				return err
			}
			return nil
		})
		well.Stop()
		return well.Wait()
	},
}

func init() {
	rebootQueueCmd.AddCommand(rebootQueueListCmd)
}

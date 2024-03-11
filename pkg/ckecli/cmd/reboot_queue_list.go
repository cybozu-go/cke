package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var rebootQueueListOptions struct {
	Output string
}

var rebootQueueListCmd = &cobra.Command{
	Use:   "list",
	Short: "list the entries in the reboot queue",
	Long: `List the entries in the reboot queue.

The output is a list of RebootQueueEntry formatted in JSON.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		well.Go(func(ctx context.Context) error {
			if rebootQueueListOptions.Output != "json" && rebootQueueListOptions.Output != "simple" {
				return errors.New("invalid output format")
			}
			entries, err := storage.GetRebootsEntries(ctx)
			if err != nil {
				return err
			}
			if rebootQueueListOptions.Output == "simple" {
				w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 1, 1, ' ', 0)
				w.Write([]byte("Index\tNode\tStatus\tLastTransitionTime\tDrainBackOffCount\tDrainBackOffExpire\n"))
				for _, entry := range entries {
					w.Write([]byte(fmt.Sprintf("%v\t%v\t%v\t%v\t%v\t%v\t\n", entry.Index, entry.Node, entry.Status, entry.LastTransitionTime.Format(time.RFC3339), entry.DrainBackOffCount, entry.DrainBackOffExpire.Format(time.RFC3339))))
				}
				return w.Flush()
			} else {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "    ")
				if err := enc.Encode(entries); err != nil {
					return err
				}
			}
			return nil
		})
		well.Stop()
		return well.Wait()
	},
}

func init() {
	rebootQueueListCmd.Flags().StringVarP(&rebootQueueListOptions.Output, "output", "o", "json", "Output format [json,simple]")
	rebootQueueCmd.AddCommand(rebootQueueListCmd)
}

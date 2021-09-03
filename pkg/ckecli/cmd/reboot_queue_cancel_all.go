package cmd

import (
	"context"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var rebootQueueCancelAllCmd = &cobra.Command{
	Use:   "cancel-all",
	Short: "cancel all the reboot queue entries",
	Long:  `Cancel all the reboot queue entries.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		well.Go(func(ctx context.Context) error {
			entries, err := storage.GetRebootsEntries(ctx)
			if err != nil {
				return err
			}

			for _, entry := range entries {
				if entry.Status == cke.RebootStatusCancelled {
					continue
				}

				entry.Status = cke.RebootStatusCancelled
				err := storage.UpdateRebootsEntry(ctx, entry)
				if err == cke.ErrNotFound {
					// The entry has just finished
					continue
				}
				if err != nil {
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
	rebootQueueCmd.AddCommand(rebootQueueCancelAllCmd)
}

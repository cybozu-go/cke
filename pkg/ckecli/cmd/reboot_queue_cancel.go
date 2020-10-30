package cmd

import (
	"context"
	"strconv"

	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var rebootQueueCancelCmd = &cobra.Command{
	Use:   "cancel INDEX",
	Short: "cancel the specified reboot queue entry",
	Long:  `Cancel the specified reboot queue entry.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		index, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return err
		}

		well.Go(func(ctx context.Context) error {
			entry, err := storage.GetRebootsEntry(ctx, index)
			if err != nil {
				return err
			}

			entry.Cancel()
			return storage.UpdateRebootsEntry(ctx, entry)
		})
		well.Stop()
		return well.Wait()
	},
}

func init() {
	rebootQueueCmd.AddCommand(rebootQueueCancelCmd)
}

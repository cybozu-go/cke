package cmd

import (
	"strconv"

	"github.com/spf13/cobra"

	"github.com/cybozu-go/cke"
)

var rebootQueueCancelCmd = &cobra.Command{
	Use:   "cancel INDEX",
	Short: "cancel the specified reboot queue entry",
	Long:  `Cancel the specified reboot queue entry.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		index, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return err
		}

		entry, err := storage.GetRebootsEntry(ctx, index)
		if err != nil {
			return err
		}

		entry.Status = cke.RebootStatusCancelled
		return storage.UpdateRebootsEntry(ctx, entry)
	},
}

func init() {
	rebootQueueCmd.AddCommand(rebootQueueCancelCmd)
}

package cmd

import (
	"context"

	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var rebootQueueDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "disable reboot queue processing",
	Long:  `Disable reboot queue processing.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		well.Go(func(ctx context.Context) error {
			return storage.EnableRebootQueue(ctx, false)
		})
		well.Stop()
		return well.Wait()
	},
}

func init() {
	rebootQueueCmd.AddCommand(rebootQueueDisableCmd)
}

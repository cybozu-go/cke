package cmd

import (
	"context"

	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var rebootQueueEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "enable reboot queue processing",
	Long:  `Enable reboot queue processing.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		well.Go(func(ctx context.Context) error {
			return storage.EnableRebootQueue(ctx, true)
		})
		well.Stop()
		return well.Wait()
	},
}

func init() {
	rebootQueueCmd.AddCommand(rebootQueueEnableCmd)
}

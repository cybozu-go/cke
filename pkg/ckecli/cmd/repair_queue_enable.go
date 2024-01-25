package cmd

import (
	"context"

	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var repairQueueEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "enable repair queue processing",
	Long:  `Enable repair queue processing.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		well.Go(func(ctx context.Context) error {
			return storage.EnableRepairQueue(ctx, true)
		})
		well.Stop()
		return well.Wait()
	},
}

func init() {
	repairQueueCmd.AddCommand(repairQueueEnableCmd)
}

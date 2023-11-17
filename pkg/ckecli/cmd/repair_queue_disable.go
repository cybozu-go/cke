package cmd

import (
	"context"

	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var repairQueueDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "disable repair queue processing",
	Long:  `Disable repair queue processing.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		well.Go(func(ctx context.Context) error {
			return storage.EnableRepairQueue(ctx, false)
		})
		well.Stop()
		return well.Wait()
	},
}

func init() {
	repairQueueCmd.AddCommand(repairQueueDisableCmd)
}

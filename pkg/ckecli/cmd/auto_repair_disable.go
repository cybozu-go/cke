package cmd

import (
	"context"

	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var autoRepairDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "disable sabakan-triggered automatic repair",
	Long:  `Disable sabakan-triggered automatic repair.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		well.Go(func(ctx context.Context) error {
			return storage.EnableAutoRepair(ctx, false)
		})
		well.Stop()
		return well.Wait()
	},
}

func init() {
	autoRepairCmd.AddCommand(autoRepairDisableCmd)
}

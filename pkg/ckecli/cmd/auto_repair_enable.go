package cmd

import (
	"context"

	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var autoRepairEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "enable sabakan-triggered automatic repair",
	Long:  `Enable sabakan-triggered automatic repair.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		well.Go(func(ctx context.Context) error {
			return storage.EnableAutoRepair(ctx, true)
		})
		well.Stop()
		return well.Wait()
	},
}

func init() {
	autoRepairCmd.AddCommand(autoRepairEnableCmd)
}

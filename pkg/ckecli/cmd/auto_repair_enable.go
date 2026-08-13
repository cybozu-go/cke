package cmd

import (
	"github.com/spf13/cobra"
)

var autoRepairEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "enable sabakan-triggered automatic repair",
	Long:  `Enable sabakan-triggered automatic repair.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		return storage.EnableAutoRepair(cmd.Context(), true)
	},
}

func init() {
	autoRepairCmd.AddCommand(autoRepairEnableCmd)
}

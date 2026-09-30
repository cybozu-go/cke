package cmd

import (
	"github.com/spf13/cobra"
)

var autoRepairDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "disable sabakan-triggered automatic repair",
	Long:  `Disable sabakan-triggered automatic repair.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		return storage.EnableAutoRepair(cmd.Context(), false)
	},
}

func init() {
	autoRepairCmd.AddCommand(autoRepairDisableCmd)
}

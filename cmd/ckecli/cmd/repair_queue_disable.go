package cmd

import (
	"github.com/spf13/cobra"
)

var repairQueueDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "disable repair queue processing",
	Long:  `Disable repair queue processing.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		return storage.EnableRepairQueue(cmd.Context(), false)
	},
}

func init() {
	repairQueueCmd.AddCommand(repairQueueDisableCmd)
}

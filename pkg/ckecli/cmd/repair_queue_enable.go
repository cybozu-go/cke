package cmd

import (
	"github.com/spf13/cobra"
)

var repairQueueEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "enable repair queue processing",
	Long:  `Enable repair queue processing.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		return storage.EnableRepairQueue(cmd.Context(), true)
	},
}

func init() {
	repairQueueCmd.AddCommand(repairQueueEnableCmd)
}

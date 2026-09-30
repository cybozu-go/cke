package cmd

import (
	"github.com/spf13/cobra"
)

var rebootQueueDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "disable reboot queue processing",
	Long:  `Disable reboot queue processing.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		return storage.EnableRebootQueue(cmd.Context(), false)
	},
}

func init() {
	rebootQueueCmd.AddCommand(rebootQueueDisableCmd)
}

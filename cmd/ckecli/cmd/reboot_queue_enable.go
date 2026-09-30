package cmd

import (
	"github.com/spf13/cobra"
)

var rebootQueueEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "enable reboot queue processing",
	Long:  `Enable reboot queue processing.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		return storage.EnableRebootQueue(cmd.Context(), true)
	},
}

func init() {
	rebootQueueCmd.AddCommand(rebootQueueEnableCmd)
}

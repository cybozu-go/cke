package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var rebootQueueIsEnabledCmd = &cobra.Command{
	Use:   "is-enabled",
	Short: "show reboot queue status",
	Long:  `Show whether the processing of the reboot queue is enabled or not.  "true" if enabled.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		disabled, err := storage.IsRebootQueueDisabled(cmd.Context())
		if err != nil {
			return err
		}
		fmt.Println(!disabled)
		return nil
	},
}

func init() {
	rebootQueueCmd.AddCommand(rebootQueueIsEnabledCmd)
}

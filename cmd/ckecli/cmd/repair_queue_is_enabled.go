package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var repairQueueIsEnabledCmd = &cobra.Command{
	Use:   "is-enabled",
	Short: "show repair queue status",
	Long:  `Show whether the processing of the repair queue is enabled or not.  "true" if enabled.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		disabled, err := storage.IsRepairQueueDisabled(cmd.Context())
		if err != nil {
			return err
		}
		fmt.Println(!disabled)
		return nil
	},
}

func init() {
	repairQueueCmd.AddCommand(repairQueueIsEnabledCmd)
}

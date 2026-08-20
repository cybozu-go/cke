package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var autoRepairIsEnabledCmd = &cobra.Command{
	Use:   "is-enabled",
	Short: "show sabakan-triggered automatic repair status",
	Long:  `Show whether sabakan-triggered automatic repair is enabled or not.  "true" if enabled.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		disabled, err := storage.IsAutoRepairDisabled(cmd.Context())
		if err != nil {
			return err
		}
		fmt.Println(!disabled)
		return nil
	},
}

func init() {
	autoRepairCmd.AddCommand(autoRepairIsEnabledCmd)
}

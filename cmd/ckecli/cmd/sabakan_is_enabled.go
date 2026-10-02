package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var sabakanIsEnabledCmd = &cobra.Command{
	Use:   "is-enabled",
	Short: "show sabakan integration status",
	Long:  `Show whether sabakan integration is enabled or not.  "true" if enabled.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		disabled, err := storage.IsSabakanDisabled(cmd.Context())
		if err != nil {
			return err
		}
		fmt.Println(!disabled)
		return nil
	},
}

func init() {
	sabakanCmd.AddCommand(sabakanIsEnabledCmd)
}

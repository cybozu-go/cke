package cmd

import (
	"github.com/spf13/cobra"
)

var sabakanDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "disable sabakan integration",
	Long:  `Disable sabakan integration.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		return storage.EnableSabakan(cmd.Context(), false)
	},
}

func init() {
	sabakanCmd.AddCommand(sabakanDisableCmd)
}

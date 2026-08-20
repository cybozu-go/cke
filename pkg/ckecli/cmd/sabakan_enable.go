package cmd

import (
	"github.com/spf13/cobra"
)

var sabakanEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "enable sabakan integration",
	Long:  `Enable sabakan integration.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		return storage.EnableSabakan(cmd.Context(), true)
	},
}

func init() {
	sabakanCmd.AddCommand(sabakanEnableCmd)
}

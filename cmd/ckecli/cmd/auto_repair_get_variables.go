package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// autoRepairGetVariablesCmd represents the "auto-repair get-variables" command
var autoRepairGetVariablesCmd = &cobra.Command{
	Use:   "get-variables",
	Short: "get the query variables to search non-healthy machines in sabakan",
	Long:  `Get the query variables to search non-healthy machines in sabakan.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := storage.GetAutoRepairQueryVariables(cmd.Context())
		if err != nil {
			return err
		}
		os.Stdout.Write(data)
		return nil
	},
}

func init() {
	autoRepairCmd.AddCommand(autoRepairGetVariablesCmd)
}

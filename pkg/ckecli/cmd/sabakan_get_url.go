package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// sabakanGetURLCmd represents the "sabakan get-url" command
var sabakanGetURLCmd = &cobra.Command{
	Use:   "get-url",
	Short: "get stored URL of sabakan server",
	Long:  `get stored URL of sabakan server.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		u, err := storage.GetSabakanURL(cmd.Context())
		if err != nil {
			return err
		}
		fmt.Println(u)
		return nil
	},
}

func init() {
	sabakanCmd.AddCommand(sabakanGetURLCmd)
}

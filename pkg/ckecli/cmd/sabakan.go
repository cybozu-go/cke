package cmd

import (
	"github.com/spf13/cobra"
)

// sabakanCmd represents the sabakan command
var sabakanCmd = &cobra.Command{
	Use:   "sabakan",
	Short: "sabakan subcommand",
	Long:  `sabakan subcommand`,
}

func init() {
	rootCmd.AddCommand(sabakanCmd)
}

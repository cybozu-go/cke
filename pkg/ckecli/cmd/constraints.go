package cmd

import (
	"github.com/spf13/cobra"
)

// constraintsCmd represents the constraints command
var constraintsCmd = &cobra.Command{
	Use:     "constraints",
	Aliases: []string{"cstr"},
	Short:   "constraints subcommand",
	Long:    `constraints subcommand`,
}

func init() {
	rootCmd.AddCommand(constraintsCmd)
}

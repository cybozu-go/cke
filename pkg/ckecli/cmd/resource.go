package cmd

import (
	"github.com/spf13/cobra"
)

var resourceCmd = &cobra.Command{
	Use:   "resource",
	Short: "resource subcommand",
	Long:  `resource subcommand`,
}

func init() {
	rootCmd.AddCommand(resourceCmd)
}

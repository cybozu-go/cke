package cmd

import (
	"github.com/spf13/cobra"
)

// autoRepairCmd represents the auto-repair command
var autoRepairCmd = &cobra.Command{
	Use:   "auto-repair",
	Short: "auto-repair subcommand",
	Long:  `auto-repair subcommand`,
}

func init() {
	rootCmd.AddCommand(autoRepairCmd)
}

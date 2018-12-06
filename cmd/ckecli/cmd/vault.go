package cmd

import (
	"github.com/spf13/cobra"
)

// vaultCmd represents the vault command
var vaultCmd = &cobra.Command{
	Use:   "vault",
	Short: "vault subcommand",
	Long:  `vault subcommand`,
}

func init() {
	rootCmd.AddCommand(vaultCmd)
}

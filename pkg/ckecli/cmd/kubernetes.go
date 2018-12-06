package cmd

import (
	"github.com/spf13/cobra"
)

// kubernetesCmd represents the kubernetes command
var kubernetesCmd = &cobra.Command{
	Use:   "kubernetes",
	Short: "kubernetes subcommand",
	Long:  `kubernetes subcommand`,
}

func init() {
	rootCmd.AddCommand(kubernetesCmd)
}

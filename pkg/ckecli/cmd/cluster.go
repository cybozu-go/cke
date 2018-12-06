package cmd

import (
	"github.com/spf13/cobra"
)

// clusterCmd represents the cluster command
var clusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "cluster subcommand",
	Long:  `cluster subcommand`,
}

func init() {
	rootCmd.AddCommand(clusterCmd)
}

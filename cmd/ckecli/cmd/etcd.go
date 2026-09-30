package cmd

import (
	"github.com/spf13/cobra"
)

// etcdCmd represents the etcd command
var etcdCmd = &cobra.Command{
	Use:   "etcd",
	Short: "etcd subcommand",
	Long:  `etcd subcommand`,
}

func init() {
	rootCmd.AddCommand(etcdCmd)
}

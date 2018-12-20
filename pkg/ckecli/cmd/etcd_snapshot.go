package cmd

import "github.com/spf13/cobra"

// etcdSnapshotCmd represents the etcd snapshot command
var etcdSnapshotCmd = &cobra.Command{
	Use:   "etcd snapshot",
	Short: "etcd snapshot subcommand",
	Long:  `etcd snapshot subcommand`,
}

func init() {
	etcdCmd.AddCommand(etcdSnapshotCmd)
}

package cmd

import "github.com/spf13/cobra"

// etcdBackupCmd represents the etcd backup command
var etcdBackupCmd = &cobra.Command{
	Use:   "backup",
	Short: "etcd backup subcommand",
	Long:  `etcd backup subcommand`,
}

func init() {
	etcdCmd.AddCommand(etcdBackupCmd)
}

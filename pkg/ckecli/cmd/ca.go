package cmd

import (
	"github.com/spf13/cobra"
)

func isValidCAName(name string) bool {
	switch name {
	case "server", "etcd-peer", "etcd-client", "kubernetes":
		return true
	}
	return false
}

// caCmd represents the ca command
var caCmd = &cobra.Command{
	Use:   "ca",
	Short: "ca subcommand",
	Long:  `ca subcommand`,
}

func init() {
	rootCmd.AddCommand(caCmd)
}

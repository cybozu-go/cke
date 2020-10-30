package cmd

import (
	"github.com/spf13/cobra"
)

// rebootQueueCmd represents the reboot-queue command
var rebootQueueCmd = &cobra.Command{
	Use:     "reboot-queue",
	Aliases: []string{"rq"},
	Short:   "reboot-queue subcommand",
	Long:    `reboot-queue subcommand`,
}

func init() {
	rootCmd.AddCommand(rebootQueueCmd)
}

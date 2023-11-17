package cmd

import (
	"github.com/spf13/cobra"
)

var repairQueueCmd = &cobra.Command{
	Use:   "repair-queue",
	Short: "repair-queue subcommand",
	Long:  "repair-queue subcommand",
}

func init() {
	rootCmd.AddCommand(repairQueueCmd)
}

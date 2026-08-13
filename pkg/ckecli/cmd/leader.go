package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// leaderCmd represents the leader command
var leaderCmd = &cobra.Command{
	Use:   "leader",
	Short: "show the hostname of the current leader process",
	Long:  `Show the hostname of the current leader process.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		leader, err := storage.GetLeaderHostname(cmd.Context())
		if err != nil {
			return err
		}

		fmt.Println(leader)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(leaderCmd)
}

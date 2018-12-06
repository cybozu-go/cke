package cmd

import (
	"context"
	"fmt"

	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

// leaderCmd represents the leader command
var leaderCmd = &cobra.Command{
	Use:   "leader",
	Short: "show the hostname of the current leader process",
	Long:  `Show the hostname of the current leader process.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		well.Go(func(ctx context.Context) error {
			leader, err := storage.GetLeaderHostname(ctx)
			if err != nil {
				return err
			}

			fmt.Println(leader)
			return nil
		})
		well.Stop()
		return well.Wait()
	},
}

func init() {
	rootCmd.AddCommand(leaderCmd)
}

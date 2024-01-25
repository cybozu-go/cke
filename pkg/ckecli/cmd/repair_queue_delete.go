package cmd

import (
	"context"
	"strconv"

	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var repairQueueDeleteCmd = &cobra.Command{
	Use:   "delete INDEX",
	Short: "delete a repair queue entry",
	Long:  `Delete the specified repair queue entry.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		index, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return err
		}

		well.Go(func(ctx context.Context) error {
			entry, err := storage.GetRepairsEntry(ctx, index)
			if err != nil {
				return err
			}

			if entry.Deleted {
				return nil
			}

			entry.Deleted = true
			return storage.UpdateRepairsEntry(ctx, entry)
		})
		well.Stop()
		return well.Wait()
	},
}

func init() {
	repairQueueCmd.AddCommand(repairQueueDeleteCmd)
}

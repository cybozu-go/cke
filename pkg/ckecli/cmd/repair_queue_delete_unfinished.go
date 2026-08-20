package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cybozu-go/cke"
)

var repairQueueDeleteUnfinishedCmd = &cobra.Command{
	Use:   "delete-unfinished",
	Short: "delete all unfinished repair queue entries",
	Long: `Delete all unfinished repair queue entries.

Entries not in "succeeded" or "failed" status are deleted.
This displays the index numbers of deleted entries, one per line.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		entries, err := storage.GetRepairsEntries(ctx)
		if err != nil {
			return err
		}

		for _, entry := range entries {
			if entry.Deleted || entry.HasFinished() {
				continue
			}

			entry.Deleted = true
			err := storage.UpdateRepairsEntry(ctx, entry)
			if err == cke.ErrNotFound {
				// The entry has just been dequeued.
				continue
			}
			if err != nil {
				return err
			}

			fmt.Println(entry.Index)
		}

		return nil
	},
}

func init() {
	repairQueueCmd.AddCommand(repairQueueDeleteUnfinishedCmd)
}

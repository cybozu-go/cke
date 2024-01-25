package cmd

import (
	"context"
	"fmt"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var repairQueueDeleteFinishedCmd = &cobra.Command{
	Use:   "delete-finished",
	Short: "delete all finished repair queue entries",
	Long: `Delete all finished repair queue entries.

Entries in "succeeded" or "failed" status are deleted.
This displays the index numbers of deleted entries, one per line.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		well.Go(func(ctx context.Context) error {
			entries, err := storage.GetRepairsEntries(ctx)
			if err != nil {
				return err
			}

			for _, entry := range entries {
				if entry.Deleted || !entry.HasFinished() {
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
		})
		well.Stop()
		return well.Wait()
	},
}

func init() {
	repairQueueCmd.AddCommand(repairQueueDeleteFinishedCmd)
}

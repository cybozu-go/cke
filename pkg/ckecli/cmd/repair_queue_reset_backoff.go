package cmd

import (
	"context"
	"time"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var repairQueueResetBackoffCmd = &cobra.Command{
	Use:   "reset-backoff",
	Short: "Reset drain backoff of the entries in repair queue",
	Long:  `Reset drain_backoff_count and drain_backoff_expire of the entries in repair queue`,
	RunE: func(cmd *cobra.Command, args []string) error {
		well.Go(func(ctx context.Context) error {
			entries, err := storage.GetRepairsEntries(ctx)
			if err != nil {
				return err
			}
			for _, entry := range entries {
				entry.DrainBackOffCount = 0
				entry.DrainBackOffExpire = time.Time{}
				err := storage.UpdateRepairsEntry(ctx, entry)
				if err == cke.ErrNotFound {
					// The entry has just been dequeued.
					continue
				}
				if err != nil {
					return err
				}
			}
			return nil
		})
		well.Stop()
		return well.Wait()
	},
}

func init() {
	repairQueueCmd.AddCommand(repairQueueResetBackoffCmd)
}

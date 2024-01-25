package cmd

import (
	"context"
	"fmt"

	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var repairQueueIsEnabledCmd = &cobra.Command{
	Use:   "is-enabled",
	Short: "show repair queue status",
	Long:  `Show whether the processing of the repair queue is enabled or not.  "true" if enabled.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		well.Go(func(ctx context.Context) error {
			disabled, err := storage.IsRepairQueueDisabled(ctx)
			if err != nil {
				return err
			}
			fmt.Println(!disabled)
			return nil
		})
		well.Stop()
		return well.Wait()
	},
}

func init() {
	repairQueueCmd.AddCommand(repairQueueIsEnabledCmd)
}

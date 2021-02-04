package cmd

import (
	"context"
	"fmt"

	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var rebootQueueIsEnabledCmd = &cobra.Command{
	Use:   "is-enabled",
	Short: "show reboot queue status",
	Long:  `Show whether the processing of the reboot queue is enabled or not.  "true" if enabled.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		well.Go(func(ctx context.Context) error {
			disabled, err := storage.IsRebootQueueDisabled(ctx)
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
	rebootQueueCmd.AddCommand(rebootQueueIsEnabledCmd)
}

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
	Long:  `Show reboot queue status.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		well.Go(func(ctx context.Context) error {
			disabled, err := storage.IsRebootQueueDisabled(ctx)
			fmt.Println(!disabled)
			return err
		})
		well.Stop()
		return well.Wait()
	},
}

func init() {
	rebootQueueCmd.AddCommand(rebootQueueIsEnabledCmd)
}

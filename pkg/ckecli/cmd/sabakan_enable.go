package cmd

import (
	"context"

	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var sabakanEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "enable sabakan integration",
	Long:  `Enable sabakan integration.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		well.Go(func(ctx context.Context) error {
			return storage.EnableSabakan(ctx, true)
		})
		well.Stop()
		return well.Wait()
	},
}

func init() {
	sabakanCmd.AddCommand(sabakanEnableCmd)
}

package cmd

import (
	"context"
	"fmt"

	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var sabakanIsEnabledCmd = &cobra.Command{
	Use:   "is-enabled",
	Short: "show sabakan integration status",
	Long:  `Show whether sabakan integration is enabled or not.  "true" if enabled.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		well.Go(func(ctx context.Context) error {
			disabled, err := storage.IsSabakanDisabled(ctx)
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
	sabakanCmd.AddCommand(sabakanIsEnabledCmd)
}

package cmd

import (
	"context"
	"os"

	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

// autoRepairGetVariablesCmd represents the "auto-repair get-variables" command
var autoRepairGetVariablesCmd = &cobra.Command{
	Use:   "get-variables",
	Short: "get the query variables to search non-healthy machines in sabakan",
	Long:  `Get the query variables to search non-healthy machines in sabakan.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		well.Go(func(ctx context.Context) error {
			data, err := storage.GetAutoRepairQueryVariables(ctx)
			if err != nil {
				return err
			}
			os.Stdout.Write(data)
			return nil
		})
		well.Stop()
		return well.Wait()
	},
}

func init() {
	autoRepairCmd.AddCommand(autoRepairGetVariablesCmd)
}

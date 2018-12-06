package cmd

import (
	"context"
	"os"

	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

// sabakanGetVariablesCmd represents the "sabakan get-variables" command
var sabakanGetVariablesCmd = &cobra.Command{
	Use:   "get-variables",
	Short: "get the query variables to search machines in sabakan",
	Long:  `Get the query variables to search machines in sabakan.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		well.Go(func(ctx context.Context) error {
			data, err := storage.GetSabakanQueryVariables(ctx)
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
	sabakanCmd.AddCommand(sabakanGetVariablesCmd)
}

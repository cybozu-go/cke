package cmd

import (
	"context"
	"fmt"

	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

// sabakanGetURLCmd represents the "sabakan get-url" command
var sabakanGetURLCmd = &cobra.Command{
	Use:   "get-url",
	Short: "get stored URL of sabakan server",
	Long:  `get stored URL of sabakan server.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		well.Go(func(ctx context.Context) error {
			u, err := storage.GetSabakanURL(ctx)
			if err != nil {
				return err
			}
			fmt.Println(u)
			return nil
		})
		well.Stop()
		return well.Wait()
	},
}

func init() {
	sabakanCmd.AddCommand(sabakanGetURLCmd)
}

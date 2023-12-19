package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var resourceGetCmd = &cobra.Command{
	Use:   "get KEY",
	Short: "get a user-defined resource by key",
	Long:  `Get a user-defined resource by key.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		well.Go(func(ctx context.Context) error {
			data, _, err := storage.GetResource(ctx, args[0])
			if err != nil {
				return err
			}

			fmt.Println(strings.TrimSpace(string(data)))
			return nil
		})
		well.Stop()
		return well.Wait()
	},
}

func init() {
	resourceCmd.AddCommand(resourceGetCmd)
}

package cmd

import (
	"context"
	"fmt"

	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var resourceListCmd = &cobra.Command{
	Use:   "list",
	Short: "list keys of user resources",
	Long:  `List keys of registered user resources.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		well.Go(func(ctx context.Context) error {
			keys, err := storage.ListResources(ctx)
			if err != nil {
				return err
			}

			for _, key := range keys {
				fmt.Println(key)
			}
			return nil
		})
		well.Stop()
		return well.Wait()
	},
}

func init() {
	resourceCmd.AddCommand(resourceListCmd)
}

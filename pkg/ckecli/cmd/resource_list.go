package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var resourceListCmd = &cobra.Command{
	Use:   "list",
	Short: "list keys of user resources",
	Long:  `List keys of registered user resources.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		keys, err := storage.ListResources(cmd.Context())
		if err != nil {
			return err
		}

		for _, key := range keys {
			fmt.Println(key)
		}
		return nil
	},
}

func init() {
	resourceCmd.AddCommand(resourceListCmd)
}

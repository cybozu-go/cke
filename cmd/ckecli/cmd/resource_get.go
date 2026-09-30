package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var resourceGetCmd = &cobra.Command{
	Use:   "get KEY",
	Short: "get a user-defined resource by key",
	Long:  `Get a user-defined resource by key.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		data, _, err := storage.GetResource(cmd.Context(), args[0])
		if err != nil {
			return err
		}

		fmt.Println(strings.TrimSpace(string(data)))
		return nil
	},
}

func init() {
	resourceCmd.AddCommand(resourceGetCmd)
}

package cmd

import (
	"encoding/json"
	"os"

	"github.com/spf13/cobra"
)

// constraintsShowCmd represents the "constraints show" command
var constraintsShowCmd = &cobra.Command{
	Use:   "show",
	Short: "show current constraints",
	Long:  `Show the list of current constraint values.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		cstr, err := storage.GetConstraints(cmd.Context())
		if err != nil {
			return err
		}

		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "    ")
		return enc.Encode(cstr)
	},
}

func init() {
	constraintsCmd.AddCommand(constraintsShowCmd)
}

package cmd

import (
	"context"
	"encoding/json"
	"os"

	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

// constraintsShowCmd represents the "constraints show" command
var constraintsShowCmd = &cobra.Command{
	Use:   "show",
	Short: "show current constraints",
	Long:  `Show the list of current constraint values.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		well.Go(func(ctx context.Context) error {
			cstr, err := storage.GetConstraints(ctx)
			if err != nil {
				return err
			}

			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "    ")
			return enc.Encode(cstr)
		})
		well.Stop()
		return well.Wait()
	},
}

func init() {
	constraintsCmd.AddCommand(constraintsShowCmd)
}

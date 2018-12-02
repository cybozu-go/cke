package cmd

import (
	"context"
	"os"

	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v2"
)

// sabakanGetTemplateCmd represents the "sabakan get-template" command
var sabakanGetTemplateCmd = &cobra.Command{
	Use:   "get-template",
	Short: "get the cluster configuration template",
	Long:  `Get the cluster configuration template.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		well.Go(func(ctx context.Context) error {
			tmpl, _, err := storage.GetSabakanTemplate(ctx)
			if err != nil {
				return err
			}

			return yaml.NewEncoder(os.Stdout).Encode(tmpl)
		})
		well.Stop()
		return well.Wait()
	},
}

func init() {
	sabakanCmd.AddCommand(sabakanGetTemplateCmd)
}

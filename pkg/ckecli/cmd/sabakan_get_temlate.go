package cmd

import (
	"context"
	"os"

	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
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

			b, err := yaml.Marshal(tmpl)
			if err != nil {
				return nil
			}

			_, err = os.Stdout.Write(b)
			return err
		})
		well.Stop()
		return well.Wait()
	},
}

func init() {
	sabakanCmd.AddCommand(sabakanGetTemplateCmd)
}

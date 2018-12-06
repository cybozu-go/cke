package cmd

import (
	"context"
	"errors"
	"os"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v2"
)

func validateTemplate(tmpl *cke.Cluster) error {
	switch {
	case len(tmpl.Nodes) != 2:
		fallthrough
	case tmpl.Nodes[0].ControlPlane && tmpl.Nodes[1].ControlPlane:
		fallthrough
	case !tmpl.Nodes[0].ControlPlane && !tmpl.Nodes[1].ControlPlane:
		return errors.New("template must contain one control-plane and one non-control-plane nodes")
	}

	tmpl.Nodes[0].Address = "10.0.0.1"
	tmpl.Nodes[1].Address = "10.0.0.2"
	return tmpl.Validate()
}

// sabakanSetTemplateCmd represents the "sabakan set-template" command
var sabakanSetTemplateCmd = &cobra.Command{
	Use:   "set-template FILE",
	Short: "set the cluster configuration template",
	Long: `Set the cluster configuration template.

FILE should contain a YAML/JSON template of the cluster configuration.
The format is the same as the cluster configuration, but must contain
just one control-plane node and one non contorl-plane node.`,

	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := os.Open(args[0])
		if err != nil {
			return err
		}
		defer r.Close()

		tmpl := cke.NewCluster()
		err = yaml.NewDecoder(r).Decode(tmpl)
		if err != nil {
			return err
		}
		err = validateTemplate(tmpl)
		if err != nil {
			return err
		}

		well.Go(func(ctx context.Context) error {
			return storage.SetSabakanTemplate(ctx, tmpl)
		})
		well.Stop()
		return well.Wait()
	},
}

func init() {
	sabakanCmd.AddCommand(sabakanSetTemplateCmd)
}

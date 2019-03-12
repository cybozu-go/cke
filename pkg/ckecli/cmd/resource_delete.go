package cmd

import (
	"bufio"
	"context"
	"io"
	"os"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"

	k8sYaml "k8s.io/apimachinery/pkg/util/yaml"
)

var resourceDeleteCmd = &cobra.Command{
	Use:   "delete FILE",
	Short: "remove user-defined resources.",
	Long: `Remove user-defined resources.

FILE should contain multiple Kubernetes resources in YAML or JSON format.

Note that resources in Kubernetes will not be removed automatically.`,

	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := os.Open(args[0])
		if err != nil {
			return err
		}
		defer r.Close()

		well.Go(func(ctx context.Context) error {
			y := k8sYaml.NewYAMLReader(bufio.NewReader(r))
			for {
				data, err := y.Read()
				if err == io.EOF {
					break
				} else if err != nil {
					return err
				}
				key, _, _, err := cke.ParseResource(data)
				if err != nil {
					return err
				}
				err = storage.DeleteResource(ctx, key)
				if err != nil {
					return err
				}
			}
			return nil
		})
		well.Stop()
		return well.Wait()
	},
}

func init() {
	resourceCmd.AddCommand(resourceDeleteCmd)
}

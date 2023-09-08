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

var resourceSetCmd = &cobra.Command{
	Use:   "set FILE",
	Short: "register user-defined resources.",
	Long: `Register user-defined resources.

FILE should contain multiple Kubernetes resources in YAML or JSON format.
If FILE is "-", then data is read from stdin.

If a resource with the same key as a registered resource is specified, the resource will be overwritten.
If a resource exists in a registered resource but not in the specified resource, the resource will be deleted.`,

	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		r := os.Stdin
		if args[0] != "-" {
			f, err := os.Open(args[0])
			if err != nil {
				return err
			}
			defer f.Close()
			r = f
		}

		well.Go(func(ctx context.Context) error {
			newResources := make(map[string]string)
			y := k8sYaml.NewYAMLReader(bufio.NewReader(r))
			for {
				data, err := y.Read()
				if err == io.EOF {
					break
				}
				if err != nil {
					return err
				}
				key, err := cke.ParseResource(data)
				if err != nil {
					return err
				}
				newResources[key] = string(data)
			}
			return storage.ReplaceResources(ctx, newResources)
		})
		well.Stop()
		return well.Wait()
	},
}

func init() {
	resourceCmd.AddCommand(resourceSetCmd)
}

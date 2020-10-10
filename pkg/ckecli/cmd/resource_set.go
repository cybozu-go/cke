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

func updateResource(ctx context.Context, data []byte) error {
	key, err := cke.ParseResource(data)
	if err != nil {
		return err
	}

	return storage.SetResource(ctx, key, string(data))
}

var resourceSetCmd = &cobra.Command{
	Use:   "set FILE",
	Short: "register user-defined resources.",
	Long: `Register user-defined resources.

FILE should contain multiple Kubernetes resources in YAML or JSON format.
If FILE is "-", then data is read from stdin.`,

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
			y := k8sYaml.NewYAMLReader(bufio.NewReader(r))
			for {
				data, err := y.Read()
				if err == io.EOF {
					return nil
				}
				if err != nil {
					return err
				}

				err = updateResource(ctx, data)
				if err != nil {
					return err
				}
			}
		})
		well.Stop()
		return well.Wait()
	},
}

func init() {
	resourceCmd.AddCommand(resourceSetCmd)
}

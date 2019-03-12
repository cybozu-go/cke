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
	Short: "register user resources.",
	Long: `Register user resources.

FILE should contain multiple Kubernetes resources in YAML format.
Supports only the following types of resources.

  * Namespace
  * ServiceAccount
  * PodSecurityPolicy
  * NetworkPolicy
  * Role
  * RoleBinding
  * ClusterRole
  * ClusterRoleBinding
  * ConfigMap
  * Deployment
  * DaemonSet
  * CronJob
  * Service`,

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
				key, jd, err := cke.ParseResource(data)
				if err != nil {
					return err
				}
				_, _, err = storage.GetResource(ctx, key)
				if err == cke.ErrNotFound {
					err = storage.CreateResource(ctx, cke.KeyResourcePrefix+key, string(jd))
					if err != nil {
						return err
					}
					continue
				}
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
	resourceCmd.AddCommand(resourceSetCmd)
}

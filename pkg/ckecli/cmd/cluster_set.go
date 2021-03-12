package cmd

import (
	"context"
	"os"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
)

// clusterSetCmd represents the "cluster set" command
var clusterSetCmd = &cobra.Command{
	Use:   "set FILE",
	Short: "load cluster configuration",
	Long: `Load cluster configuration from FILE and store it in etcd.

The file must be either YAML or JSON.`,

	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		b, err := os.ReadFile(args[0])
		if err != nil {
			return err
		}

		cfg := cke.NewCluster()
		err = yaml.Unmarshal(b, cfg)
		if err != nil {
			return err
		}
		err = cfg.Validate(false)
		if err != nil {
			return err
		}

		well.Go(func(ctx context.Context) error {
			constraints, err := storage.GetConstraints(ctx)
			switch err {
			case cke.ErrNotFound:
				constraints = cke.DefaultConstraints()
				fallthrough
			case nil:
				err = constraints.Check(cfg)
				if err != nil {
					return err
				}
			default:
				return err
			}

			return storage.PutCluster(ctx, cfg)
		})
		well.Stop()
		return well.Wait()
	},
}

func init() {
	clusterCmd.AddCommand(clusterSetCmd)
}

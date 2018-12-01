package cmd

import (
	"context"
	"os"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v2"
)

// clusterSetCmd represents the "cluster set" command
var clusterSetCmd = &cobra.Command{
	Use:   "set FILE",
	Short: "load cluster configuration",
	Long: `Load cluster configuration from FILE and store it in etcd.

The file must be either YAML or JSON.`,

	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := os.Open(args[0])
		if err != nil {
			return err
		}
		defer r.Close()

		cfg := cke.NewCluster()
		err = yaml.NewDecoder(r).Decode(cfg)
		if err != nil {
			return err
		}
		err = cfg.Validate()
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

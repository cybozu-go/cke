package cmd

import (
	"context"
	"os"

	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v2"
)

// clusterGetCmd represents the "cluster get" command
var clusterGetCmd = &cobra.Command{
	Use:   "get",
	Short: "dump stored cluster configuration",
	Long:  `Dump cluster configuration stored in etcd.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		well.Go(func(ctx context.Context) error {
			cfg, err := storage.GetCluster(ctx)
			if err != nil {
				return err
			}

			return yaml.NewEncoder(os.Stdout).Encode(cfg)
		})
		well.Stop()
		return well.Wait()
	},
}

func init() {
	clusterCmd.AddCommand(clusterGetCmd)
}

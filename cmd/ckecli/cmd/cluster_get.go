package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
)

// clusterGetCmd represents the "cluster get" command
var clusterGetCmd = &cobra.Command{
	Use:   "get",
	Short: "dump stored cluster configuration",
	Long:  `Dump cluster configuration stored in etcd.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := storage.GetCluster(cmd.Context())
		if err != nil {
			return err
		}

		b, err := yaml.Marshal(cfg)
		if err != nil {
			return nil
		}

		_, err = os.Stdout.Write(b)
		return err
	},
}

func init() {
	clusterCmd.AddCommand(clusterGetCmd)
}

package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var rebootQueueAddCmd = &cobra.Command{
	Use:   "add FILE",
	Short: "append the nodes written in FILE to the reboot queue",
	Long: `Append the nodes written in FILE to the reboot queue.

The nodes should be specified with their IP addresses.
If FILE is -, the contents are read from stdin.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		f := os.Stdin
		if args[0] != "-" {
			var err error
			f, err = os.Open(args[0])
			if err != nil {
				return err
			}
			defer f.Close()
		}

		data, err := io.ReadAll(f)
		if err != nil {
			return err
		}
		nodes := strings.Fields(string(data))

		well.Go(func(ctx context.Context) error {
			for _, node := range nodes {
				entry := cke.NewRebootQueueEntry(node)
				cluster, err := storage.GetCluster(ctx)
				if err != nil {
					return err
				}
				err = validateNode(node, cluster)
				if err != nil {
					return err
				}

				err = storage.RegisterRebootsEntry(ctx, entry)
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

func validateNode(rebootNode string, cluster *cke.Cluster) error {
	for _, clusterNode := range cluster.Nodes {
		if rebootNode == clusterNode.Address {
			return nil
		}
	}
	return fmt.Errorf("%s is not a valid node IP address", rebootNode)
}

func init() {
	rebootQueueCmd.AddCommand(rebootQueueAddCmd)
}

package cmd

import (
	"context"
	"errors"
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
		entry := cke.NewRebootQueueEntry(nodes)

		well.Go(func(ctx context.Context) error {
			cluster, err := storage.GetCluster(ctx)
			if err != nil {
				return err
			}
			err = validateNodes(nodes, cluster)
			if err != nil {
				return err
			}

			return storage.RegisterRebootsEntry(ctx, entry)
		})
		well.Stop()
		return well.Wait()
	},
}

func validateNodes(nodes []string, cluster *cke.Cluster) error {
	numCPs := 0
OUTER:
	for _, rebootNode := range nodes {
		for _, clusterNode := range cluster.Nodes {
			if rebootNode == clusterNode.Address {
				if clusterNode.ControlPlane {
					numCPs++
				}
				continue OUTER
			}
		}
		return fmt.Errorf("%s is not a valid node IP address", rebootNode)
	}

	if numCPs > 1 {
		return errors.New("multiple control planes cannot be enqueued in one entry")
	}

	return nil
}

func init() {
	rebootQueueCmd.AddCommand(rebootQueueAddCmd)
}

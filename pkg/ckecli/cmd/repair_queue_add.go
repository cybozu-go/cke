package cmd

import (
	"context"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var repairQueueAddCmd = &cobra.Command{
	Use:   "add OPERATION MACHINE_TYPE ADDRESS",
	Short: "append a repair request to the repair queue",
	Long: `Append a repair request to the repair queue.

The repair target is a machine with an IP address ADDRESS and a machine type MACHINE_TYPE.
The machine should be processed with an operation OPERATION.`,
	Args: cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		operation := args[0]
		machineType := args[1]
		address := args[2]

		well.Go(func(ctx context.Context) error {
			entry := cke.NewRepairQueueEntry(operation, machineType, address)
			cluster, err := storage.GetCluster(ctx)
			if err != nil {
				return err
			}
			if _, err := entry.GetMatchingRepairOperation(cluster); err != nil {
				return err
			}

			return storage.RegisterRepairsEntry(ctx, entry)
		})
		well.Stop()
		return well.Wait()
	},
}

func init() {
	repairQueueCmd.AddCommand(repairQueueAddCmd)
}

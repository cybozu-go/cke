package cmd

import (
	"context"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var repairQueueAddCmd = &cobra.Command{
	Use:   "add OPERATION MACHINE_TYPE ADDRESS [SERIAL]",
	Short: "append a repair request to the repair queue",
	Long: `Append a repair request to the repair queue.

The repair target is a machine with an IP address ADDRESS and a machine type MACHINE_TYPE.
The machine should be processed with an operation OPERATION.
Optionally, you can specify the machine's serial number as the fourth argument.`,
	Args: cobra.RangeArgs(3, 4),
	RunE: func(cmd *cobra.Command, args []string) error {
		operation := args[0]
		machineType := args[1]
		address := args[2]
		serial := ""
		if len(args) > 3 {
			serial = args[3]
		}

		well.Go(func(ctx context.Context) error {
			entry := cke.NewRepairQueueEntry(operation, machineType, address, serial)
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

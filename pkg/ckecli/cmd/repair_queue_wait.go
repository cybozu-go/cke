package cmd

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

const defaultRepairQueueWaitTimeoutSeconds = 30

var repairQueueWaitCmd = &cobra.Command{
	Use:   "wait",
	Short: "wait until the repair queue is empty",
	Long: `Wait until the repair queue is empty.

The command exits successfully when no entries remain in the repair queue.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		timeoutSeconds, err := cmd.Flags().GetInt64("timeout-seconds")
		if err != nil {
			return err
		}
		if timeoutSeconds < 0 {
			return errors.New("--timeout-seconds must not be negative")
		}

		timeout := time.Duration(timeoutSeconds) * time.Second
		well.Go(func(ctx context.Context) error {
			if timeout == 0 {
				entries, err := storage.GetRepairsEntries(ctx)
				if err != nil {
					return err
				}
				if len(entries) != 0 {
					return errors.New("repair queue is not empty")
				}

				fmt.Fprintln(cmd.OutOrStdout(), "repair queue is empty")
				return nil
			}

			waitCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			err := storage.WaitRepairsEmpty(waitCtx)
			if errors.Is(err, context.DeadlineExceeded) {
				return fmt.Errorf(
					"timed out waiting for repair queue to become empty after %s",
					timeout,
				)
			}
			if err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), "repair queue is empty")
			return nil
		})

		well.Stop()
		return well.Wait()
	},
}

func init() {
	repairQueueWaitCmd.Flags().Int64(
		"timeout-seconds",
		defaultRepairQueueWaitTimeoutSeconds,
		"the number of seconds to wait before giving up; zero means check once and don't wait",
	)

	repairQueueCmd.AddCommand(repairQueueWaitCmd)
}

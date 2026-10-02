package cmd

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

const (
	defaultRepairQueueWaitTimeoutSeconds  = 30
	defaultRepairQueueWaitIntervalSeconds = 1
)

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

		intervalSeconds, err := cmd.Flags().GetInt64("interval-seconds")
		if err != nil {
			return err
		}
		if intervalSeconds <= 0 {
			return errors.New("--interval-seconds must be greater than zero")
		}

		timeout := time.Duration(timeoutSeconds) * time.Second
		interval := time.Duration(intervalSeconds) * time.Second

		well.Go(func(ctx context.Context) error {
			entries, err := storage.GetRepairsEntries(ctx)
			if err != nil {
				return err
			}
			if len(entries) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "repair queue is empty")
				return nil
			}
			if timeout == 0 {
				return errors.New("repair queue is not empty")
			}
			fmt.Fprintf(cmd.OutOrStdout(), "repair queue entries: %d\n", len(entries))

			timer := time.NewTimer(timeout)
			defer timer.Stop()
			ticker := time.NewTicker(interval)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-timer.C:
					return fmt.Errorf("timed out waiting for repair queue to become empty after %s", timeout)
				case <-ticker.C:
					entries, err := storage.GetRepairsEntries(ctx)
					if err != nil {
						return err
					}
					if len(entries) == 0 {
						fmt.Fprintln(cmd.OutOrStdout(), "repair queue is empty")
						return nil
					}
					fmt.Fprintf(cmd.OutOrStdout(), "repair queue entries: %d\n", len(entries))
				}
			}
		})

		well.Stop()
		return well.Wait()
	},
}

func init() {
	repairQueueWaitCmd.Flags().Int64("timeout-seconds", defaultRepairQueueWaitTimeoutSeconds, "the number of seconds to wait before giving up; zero means check once and don't wait")
	repairQueueWaitCmd.Flags().Int64("interval-seconds", defaultRepairQueueWaitIntervalSeconds, "the number of seconds between repair queue checks")

	repairQueueCmd.AddCommand(repairQueueWaitCmd)
}

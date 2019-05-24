package cmd

import (
	"context"
	"encoding/json"
	"os"
	"sort"

	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var historyCount int
var followFlag bool

// historyCmd represents the history command
var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "show the hostname of the current history process",
	Long:  `Show the hostname of the current history process.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		well.Go(func(ctx context.Context) error {
			records, err := storage.GetRecords(ctx, int64(historyCount))
			if err != nil {
				return err
			}

			if followFlag {
				sort.SliceStable(records, func(i, j int) bool { return records[i].ID < records[j].ID })
			}

			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "    ")
			for _, r := range records {
				err = enc.Encode(r)
				if err != nil {
					return err
				}
			}

			if !followFlag {
				return nil
			}

			recordCh := storage.WatchRecords(ctx)
			for r := range recordCh {
				err = enc.Encode(r)
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

func init() {
	historyCmd.Flags().IntVarP(&historyCount, "count", "n", 0, "limit the number of operations to show")
	historyCmd.Flags().BoolVarP(&followFlag, "follow", "f", false, "show operations continuously")
	rootCmd.AddCommand(historyCmd)
}

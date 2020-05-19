package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/cybozu-go/cke"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "show the server status",
	Long: `Show the server status if the server is running.
If no status is available, this command exits with status code 4.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		st, err := storage.GetStatus(context.Background())
		if err == cke.ErrNotFound {
			fmt.Fprintln(os.Stderr, "no status")
			os.Exit(4)
		}
		if err != nil {
			return err
		}
		enc := json.NewEncoder(os.Stdout)
		return enc.Encode(st)
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

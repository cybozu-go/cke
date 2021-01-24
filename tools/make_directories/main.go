package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/spf13/cobra"
)

var options struct {
	mode string
}

var rootCmd = &cobra.Command{
	Use:   "make_directories DIR [DIR ...]",
	Short: "make directories",
	Long:  `make directories with given permission flags`,
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		return subMain(args)
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func subMain(dirs []string) error {
	modeBits, err := strconv.ParseInt(options.mode, 8, 64)
	if err != nil {
		return fmt.Errorf("invalid mode %s: %w", options.mode, err)
	}

	for _, d := range dirs {
		if !filepath.IsAbs(d) {
			return fmt.Errorf("non-absolute path: %s", d)
		}

		if err := os.MkdirAll(d, os.FileMode(modeBits)); err != nil {
			return fmt.Errorf("failed to create %s: %w", d, err)
		}
	}
	return nil
}

func init() {
	rootCmd.Flags().StringVar(&options.mode, "mode", "755", "permission bits for directories")
}

package main

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/spf13/cobra"
)

var options struct {
	mode string
}

var rootCmd = &cobra.Command{
	Use:   "write_files DIR",
	Short: "read TAR from stdin and write out the contents",
	Long: `This command reads a TAR archive from stdin and extracts
the contents under DIR.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		return subMain(args[0])
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func subMain(prefix string) error {
	modeBits, err := strconv.ParseInt(options.mode, 8, 64)
	if err != nil {
		return fmt.Errorf("invalid mode %s: %w", options.mode, err)
	}

	r := tar.NewReader(os.Stdin)
	for {
		hdr, err := r.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar: %w", err)
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}

		dest := filepath.Join(prefix, hdr.Name)
		dir := filepath.Dir(dest)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to mkdir %s: %w", dir, err)
		}

		if err := copyFile(dest, r, os.FileMode(modeBits)); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(dest string, r io.Reader, mode os.FileMode) error {
	t := dest + ".tmp"
	f, err := os.OpenFile(t, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", t, err)
	}
	defer f.Close()

	if _, err := io.Copy(f, r); err != nil {
		return fmt.Errorf("failed to write contents of %s: %w", t, err)
	}
	if err := f.Sync(); err != nil {
		return fmt.Errorf("failed to fsync %s: %w", t, err)
	}

	return os.Rename(t, dest)
}

func init() {
	rootCmd.Flags().StringVar(&options.mode, "mode", "644", "permission bits for extracted files")
}

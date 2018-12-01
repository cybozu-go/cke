package cmd

import (
	"context"
	"errors"
	"os"

	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

// caGetCmd represents the "ca get" command
var caGetCmd = &cobra.Command{
	Use:   "get NAME",
	Short: "dump stored CA certificate to stdout",
	Long: `Dump stored CA certificate to stdout.

NAME is one of:
    server
    etcd-peer
    etcd-client
    kubernetes`,

	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New("wrong number of arguments")
		}

		if !isValidCAName(args[0]) {
			return errors.New("wrong CA name: " + args[0])
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		well.Go(func(ctx context.Context) error {
			pem, err := storage.GetCACertificate(ctx, args[0])
			if err != nil {
				return err
			}

			_, err = os.Stdout.WriteString(pem)
			return err
		})
		well.Stop()
		return well.Wait()
	},
}

func init() {
	caCmd.AddCommand(caGetCmd)
}

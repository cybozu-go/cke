package cmd

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io/ioutil"

	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var caData []byte

// caSetCmd represents the "ca set" command
var caSetCmd = &cobra.Command{
	Use:   "set NAME FILE",
	Short: "store CA certificate in etcd",
	Long: `Load PEM encoded x509 CA certificate from FILE, and stores it in etcd.

NAME is one of:
    server
    etcd-peer
    etcd-client
    kubernetes

In fact, these CA should be created in Vault.`,

	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 2 {
			return errors.New("wrong number of arguments")
		}

		if !isValidCAName(args[0]) {
			return errors.New("wrong CA name: " + args[0])
		}

		var err error
		caData, err = ioutil.ReadFile(args[1])
		if err != nil {
			return err
		}

		block, _ := pem.Decode(caData)
		if block == nil {
			return errors.New("invalid PEM data")
		}
		_, err = x509.ParseCertificate(block.Bytes)
		if err != nil {
			return errors.New("invalid certificate")
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		well.Go(func(ctx context.Context) error {
			return storage.PutCACertificate(ctx, args[0], string(caData))
		})
		well.Stop()
		return well.Wait()
	},
}

func init() {
	caCmd.AddCommand(caSetCmd)
}

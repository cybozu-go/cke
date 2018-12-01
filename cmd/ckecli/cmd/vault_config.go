package cmd

import (
	"context"
	"encoding/json"
	"os"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

// vaultConfigCmd represents the "vault config" command
var vaultConfigCmd = &cobra.Command{
	Use:   "config FILE|-",
	Short: "store parameters to connect Vault",
	Long: `Load parameters to connect Vault from a FILE or stdin,
and stores it in etcd.

The parameters are given by a JSON object having these fields:

    endpoint:  Vault URL.
    ca-cert:   PEM encoded CA certificate to verify server certificate.
    role-id:   AppRole ID to login to Vault.
    secret-id: AppRole secret to login to Vault.

If the argument is "-", the JSON is read from stdin.`,

	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		f := os.Stdin
		if args[0] != "-" {
			var err error
			f, err = os.Open(args[0])
			if err != nil {
				return err
			}
			defer f.Close()
		}

		cfg := new(cke.VaultConfig)
		err := json.NewDecoder(f).Decode(cfg)
		if err != nil {
			return err
		}
		err = cfg.Validate()
		if err != nil {
			return err
		}

		well.Go(func(ctx context.Context) error {
			return storage.PutVaultConfig(ctx, cfg)
		})
		well.Stop()
		return well.Wait()
	},
}

func init() {
	vaultCmd.AddCommand(vaultConfigCmd)
}

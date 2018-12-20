package cmd

import (
	"io/ioutil"
	"os"

	"github.com/cybozu-go/cke"
	"github.com/spf13/cobra"
)

var vaultSSHPrivKeyHost string

// vaultSSHPrivKeyCmd represents the "vault ssh-privkey" command
var vaultSSHPrivKeyCmd = &cobra.Command{
	Use:   "ssh-privkey FILE|-",
	Short: "store SSH private key into Vault",
	Long: `Store SSH private key for a host into Vault.

If --host is not specified, the key will be used as the default key.

FILE should be a SSH private key file.
If FILE is -, the contents are read from stdin.`,

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

		data, err := ioutil.ReadAll(f)
		if err != nil {
			return err
		}

		vc, err := inf.Vault()
		if err != nil {
			return err
		}

		secret, err := vc.Logical().Read(cke.SSHSecret)
		if err != nil {
			return err
		}

		var privkeys map[string]interface{}
		if secret != nil && secret.Data != nil {
			privkeys = secret.Data
		} else {
			privkeys = make(map[string]interface{})
		}
		privkeys[vaultSSHPrivKeyHost] = string(data)

		_, err = vc.Logical().Write(cke.SSHSecret, privkeys)
		return err
	},
}

func init() {
	vaultSSHPrivKeyCmd.Flags().StringVar(&vaultSSHPrivKeyHost, "host", "", "target host of SSH key")
	vaultCmd.AddCommand(vaultSSHPrivKeyCmd)
}

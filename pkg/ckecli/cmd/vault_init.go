package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/well"
	vault "github.com/hashicorp/vault/api"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const (
	ttl100Year = "876000h"
	ttl10Year  = "87600h"
)

type caParams struct {
	commonName string
	key        string
}

var (
	cas = []caParams{
		{
			commonName: "server CA",
			key:        cke.CAServer,
		},
		{
			commonName: "etcd peer CA",
			key:        cke.CAEtcdPeer,
		},
		{
			commonName: "etcd client CA",
			key:        cke.CAEtcdClient,
		},
		{
			commonName: "kubernetes CA",
			key:        cke.CAKubernetes,
		},
		{
			commonName: "kubernetes aggregation CA",
			key:        cke.CAKubernetesAggregation,
		},
		{
			commonName: "kubernetes webhook CA",
			key:        cke.CAWebhook,
		},
	}

	ckePolicy = `
path "cke/*"
{
  capabilities = ["create", "read", "update", "delete", "list", "sudo"]
}`
)

func readPasswordFromStdTerminal(prompt string) (string, error) {
	if !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stdout.Fd())) {
		return "", fmt.Errorf("stdin and stdout are not terminals")
	}

	fmt.Print(prompt)
	p, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return "", err
	}
	return string(p), nil
}

func connectVault(ctx context.Context) (*vault.Client, error) {
	cfg := vault.DefaultConfig()
	if len(vaultInitCfg.endpoint) > 0 {
		cfg.Address = vaultInitCfg.endpoint
	}

	if len(vaultInitCfg.caCertFile) > 0 {
		tlsCfg := &vault.TLSConfig{
			CACert: vaultInitCfg.caCertFile,
		}
		err := cfg.ConfigureTLS(tlsCfg)
		if err != nil {
			return nil, err
		}
	}

	vc, err := vault.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	if vc.Token() != "" {
		return vc, nil
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Vault username: ")
	username, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	username = username[0 : len(username)-1]

	password, err := readPasswordFromStdTerminal("Vault password: ")
	if err != nil {
		return nil, err
	}

	secret, err := vc.Logical().Write("/auth/userpass/login/"+username,
		map[string]interface{}{"password": password})
	if err != nil {
		return nil, err
	}
	vc.SetToken(secret.Auth.ClientToken)

	return vc, nil
}

func initVault(ctx context.Context) error {
	vc, err := connectVault(ctx)
	if err != nil {
		return err
	}

	for _, ca := range cas {
		err = createPKI(ctx, vc, ca)
		if err != nil {
			return err
		}
	}

	err = createKV(ctx, vc)
	if err != nil {
		return err
	}

	err = vc.Sys().PutPolicy("cke", ckePolicy)
	if err != nil {
		return err
	}

	cfg, err := storage.GetVaultConfig(ctx)
	switch err {
	case nil:
	case cke.ErrNotFound:
		_, err = vc.Logical().Write("auth/approle/role/cke", map[string]interface{}{
			"policies": "cke",
			"period":   "1h",
		})
		if err != nil {
			return err
		}
		secret, err := vc.Logical().Read("auth/approle/role/cke/role-id")
		if err != nil {
			return err
		}
		roleID := secret.Data["role_id"].(string)

		secret, err = vc.Logical().Write("auth/approle/role/cke/secret-id", map[string]interface{}{})
		if err != nil {
			return err
		}
		secretID := secret.Data["secret_id"].(string)

		cfg = new(cke.VaultConfig)
		cfg.Endpoint = vc.Address()
		cfg.RoleID = roleID
		cfg.SecretID = secretID
		if len(vaultInitCfg.caCertFile) > 0 {
			data, err := os.ReadFile(vaultInitCfg.caCertFile)
			if err != nil {
				return err
			}
			cfg.CACert = string(data)
		}

		err = storage.PutVaultConfig(ctx, cfg)
		if err != nil {
			return err
		}
	default:
		return err
	}

	vc2, _, err := cke.VaultClient(cfg)
	if err != nil {
		return err
	}

	for _, ca := range cas {
		err = createRootCA(ctx, vc2, ca)
		if err != nil {
			return err
		}
	}

	secret, err := vc2.Logical().Read(cke.K8sSecret)
	if err != nil {
		return err
	}
	if secret != nil && secret.Data != nil {
		return nil
	}

	return rotateK8sEncryptionKey(vc2)
}

func createPKI(ctx context.Context, vc *vault.Client, ca caParams) error {
	mounts, err := vc.Sys().ListMounts()
	if err != nil {
		return err
	}
	pkiKey := cke.VaultPKIKey(ca.key)
	if _, ok := mounts[pkiKey]; ok {
		return nil
	}
	if _, ok := mounts[pkiKey+"/"]; ok {
		return nil
	}

	err = vc.Sys().Mount(pkiKey, &vault.MountInput{
		Type: "pki",
		Config: vault.MountConfigInput{
			MaxLeaseTTL:     ttl100Year,
			DefaultLeaseTTL: ttl10Year,
		},
	})
	if err != nil {
		return err
	}

	fmt.Printf("mounted pki on %s\n", pkiKey)
	return nil
}

func createRootCA(ctx context.Context, vc *vault.Client, ca caParams) error {
	_, err := storage.GetCACertificate(ctx, ca.key)
	if err == nil {
		return nil
	}

	if err != cke.ErrNotFound {
		return err
	}

	pkiKey := cke.VaultPKIKey(ca.key)
	secret, err := vc.Logical().Write(path.Join(pkiKey, "/root/generate/internal"), map[string]interface{}{
		"common_name": ca.commonName,
		"ttl":         ttl100Year,
		"format":      "pem",
	})
	if err != nil {
		return err
	}
	_, ok := secret.Data["certificate"]
	if !ok {
		return fmt.Errorf("failed to issue ca: %#v", secret.Warnings)
	}

	fmt.Printf("issued root certificate for %s\n", pkiKey)
	return storage.PutCACertificate(ctx, ca.key, secret.Data["certificate"].(string))
}

func createKV(ctx context.Context, vc *vault.Client) error {
	mounts, err := vc.Sys().ListMounts()
	if err != nil {
		return err
	}
	if _, ok := mounts[cke.CKESecret]; ok {
		return nil
	}
	if _, ok := mounts[cke.CKESecret+"/"]; ok {
		return nil
	}

	kv1 := &vault.MountInput{
		Type:    "kv",
		Options: map[string]string{"version": "1"},
	}
	err = vc.Sys().Mount(cke.CKESecret, kv1)
	if err != nil {
		return err
	}

	fmt.Printf("mounted kv version 1 on %s\n", cke.CKESecret)
	return nil
}

var vaultInitCfg struct {
	caCertFile string
	endpoint   string
}

// vaultInitCmd represents the "vault init" command
var vaultInitCmd = &cobra.Command{
	Use:   "init",
	Short: "configure Vault for CKE",
	Long: `Configure HashiCorp Vault for CKE.

Vault will be configured to:

    * have "cke" policy that can use secrets under cke/.
    * have "ca-server", "ca-etcd-peer", "ca-etcd-client", "ca-kubernetes"
      PKI secrets under cke/.
    * creates AppRole for CKE.
    * have initial encryption key for Kubernetes Secrets.

This command will ask username and password for Vault authentication
when VAULT_TOKEN environment variable is not set.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		well.Go(initVault)
		well.Stop()
		return well.Wait()
	},
}

func init() {
	vaultInitCmd.Flags().StringVar(&vaultInitCfg.caCertFile, "cacert", "", "x509 CA certificate file")
	vaultInitCmd.Flags().StringVar(&vaultInitCfg.endpoint, "endpoint", "", "Vault URL")
	vaultCmd.AddCommand(vaultInitCmd)
}

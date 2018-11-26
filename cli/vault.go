package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path"

	"github.com/cybozu-go/cke"
	"github.com/google/subcommands"
	"github.com/hashicorp/vault/api"
	"github.com/howeyc/gopass"
)

const (
	ttl100Year = "876000h"
	ttl10Year  = "87600h"
)

var ckePolicy = `
path "cke/*"
{
  capabilities = ["create", "read", "update", "delete", "list", "sudo"]
}
`

type caParams struct {
	vaultPath  string
	commonName string
	key        string
}

var (
	cas = []caParams{
		{
			vaultPath:  "cke/ca-server",
			commonName: "server CA",
			key:        "server",
		},
		{

			vaultPath:  "cke/ca-etcd-peer",
			commonName: "etcd peer CA",
			key:        "etcd-peer",
		},
		{
			vaultPath:  "cke/ca-etcd-client",
			commonName: "etcd client CA",
			key:        "etcd-client",
		},
		{
			vaultPath:  "cke/ca-kubernetes",
			commonName: "kubernetes CA",
			key:        "kubernetes",
		},
	}
)

type vault struct{}

func (v vault) SetFlags(f *flag.FlagSet) {}

func (v vault) Execute(ctx context.Context, f *flag.FlagSet) subcommands.ExitStatus {
	newc := NewCommander(f, "vault")
	newc.Register(vaultConfigCommand(), "")
	newc.Register(vaultInitCommand(), "")
	return newc.Execute(ctx)
}

// VaultCommand implements "vault" subcommand
func VaultCommand() subcommands.Command {
	return subcmd{
		vault{},
		"vault",
		"manage the vault configuration",
		"vault ACTION ...",
	}
}

type vaultInit struct{}

func (c vaultInit) SetFlags(f *flag.FlagSet) {}

func (c vaultInit) Execute(ctx context.Context, f *flag.FlagSet) subcommands.ExitStatus {
	err := initVault(ctx)
	return handleError(err)
}

func vaultInitCommand() subcommands.Command {
	return subcmd{
		vaultInit{},
		"init",
		"initialize vault connection settings",
		"vault init",
	}
}

func connectVault(ctx context.Context) (*api.Client, error) {
	cfg := api.DefaultConfig()
	vc, err := api.NewClient(cfg)
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
	pass, err := gopass.GetPasswdPrompt("Vault password: ", false, os.Stdin, os.Stdout)
	if err != nil {
		return nil, err
	}
	password := string(pass)

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

	err = vc.Sys().PutPolicy("cke", ckePolicy)
	if err != nil {
		return err
	}

	_, err = vc.Logical().Write("auth/approle/role/cke", map[string]interface{}{
		"policies": "cke",
		"period":   "1h",
	})

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

	cfg := new(cke.VaultConfig)
	cfg.Endpoint = "https://localhost:8200"
	cfg.RoleID = roleID
	cfg.SecretID = secretID

	err = storage.PutVaultConfig(ctx, cfg)
	if err != nil {
		return err
	}

	for _, ca := range cas {
		err = createCA(ctx, vc, ca)
		if err != nil {
			return err
		}
	}

	return nil
}

func createCA(ctx context.Context, vc *api.Client, ca caParams) error {
	err := vc.Sys().Mount(ca.vaultPath, &api.MountInput{
		Type: "pki",
		Config: api.MountConfigInput{
			MaxLeaseTTL:     ttl100Year,
			DefaultLeaseTTL: ttl10Year,
		},
	})
	if err != nil {
		return err
	}

	secret, err := vc.Logical().Write(path.Join(ca.vaultPath, "/root/generate/internal"), map[string]interface{}{
		"common_name": ca.commonName,
		"ttl":         ttl100Year,
		"format":      "pem",
	})
	if err != nil {
		return err
	}
	return storage.PutCACertificate(ctx, ca.key, secret.Data["certificate"].(string))
}

type vaultConfig struct{}

func (c vaultConfig) SetFlags(f *flag.FlagSet) {}

func (c vaultConfig) Execute(ctx context.Context, f *flag.FlagSet) subcommands.ExitStatus {
	if f.NArg() != 1 {
		f.Usage()
		return subcommands.ExitUsageError
	}
	fileName := f.Arg(0)

	r := os.Stdin
	var err error
	if fileName != "-" {
		r, err = os.Open(fileName)
		if err != nil {
			return handleError(err)
		}
		defer r.Close()
	}

	cfg := new(cke.VaultConfig)
	err = json.NewDecoder(r).Decode(cfg)
	if err != nil {
		return handleError(err)
	}
	err = cfg.Validate()
	if err != nil {
		return handleError(err)
	}
	err = storage.PutVaultConfig(ctx, cfg)
	return handleError(err)
}

func vaultConfigCommand() subcommands.Command {
	return subcmd{
		vaultConfig{},
		"config",
		"set vault connection settings",
		"vault config JSON",
	}
}

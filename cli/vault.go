package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"os"
	"path"
	"time"

	"github.com/cybozu-go/cke"
	"github.com/google/subcommands"
)

type vault struct{}

func (v vault) SetFlags(f *flag.FlagSet) {}

func (v vault) Execute(ctx context.Context, f *flag.FlagSet) subcommands.ExitStatus {
	newc := NewCommander(f, "vault")
	newc.Register(vaultConfigCommand(), "")
	newc.Register(vaultIssueCommand(), "")
	return newc.Execute(ctx)
}

// VaultCommand implements "vault" subcommand
func VaultCommand() subcommands.Command {
	return subcmd{
		vault{},
		"vault",
		"manage the vault configuration, or issue client certificates for etcd connection",
		"vault ACTION ...",
	}
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

type vaultIssue struct{}

func (c vaultIssue) SetFlags(f *flag.FlagSet) {}

func (c vaultIssue) Execute(ctx context.Context, f *flag.FlagSet) subcommands.ExitStatus {
	if f.NArg() != 2 {
		f.Usage()
		return subcommands.ExitUsageError
	}

	commonName := f.Arg(0)
	ttl := f.Arg(1)

	if len(commonName) == 0 {
		return handleError(errors.New("common_name is empty"))
	}

	_, err := time.ParseDuration(ttl)
	if err != nil {
		return handleError(err)
	}

	cfg, err := storage.GetVaultConfig(ctx)
	if err != nil {
		return handleError(err)
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		return handleError(err)
	}

	client, err := cke.NewVaultClient(ctx, data)
	if err != nil {
		return handleError(err)
	}

	roleOpts := map[string]interface{}{
		"ttl":            ttl,
		"max_ttl":        ttl,
		"server_flag":    "false",
		"allow_any_name": "true",
	}
	err = cke.AddRole(client, cke.CAEtcdClient, "system", roleOpts)
	if err != nil {
		return handleError(err)
	}

	// Issue certificate
	certOpts := map[string]interface{}{
		"common_name":          commonName,
		"exclude_cn_from_sans": "true",
	}
	secret, err := client.Logical().Write(path.Join(cke.CAEtcdClient, "issue", "system"), certOpts)
	if err != nil {
		return handleError(err)
	}

	data, err = json.Marshal(secret.Data)
	if err != nil {
		return handleError(err)
	}
	_, err = os.Stdout.Write(data)
	return handleError(err)
}

func vaultIssueCommand() subcommands.Command {
	return subcmd{
		vaultIssue{},
		"issue",
		"Issue client certificate and add user/role for etcd",
		"vault issue COMMON_NAME TTL",
	}
}

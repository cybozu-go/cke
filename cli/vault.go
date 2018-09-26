package cli

import (
	"context"
	"encoding/json"
	"flag"
	"os"

	"github.com/cybozu-go/cke"
	"github.com/google/subcommands"
)

type vault struct{}

func (v vault) SetFlags(f *flag.FlagSet) {}

func (v vault) Execute(ctx context.Context, f *flag.FlagSet) subcommands.ExitStatus {
	newc := NewCommander(f, "vault")
	newc.Register(vaultConfigCommand(), "")
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

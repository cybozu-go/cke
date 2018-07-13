package main

import (
	"flag"
	"os"

	"context"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/cli"
	"github.com/cybozu-go/cmd"
	"github.com/cybozu-go/log"
	"github.com/google/subcommands"
	"gopkg.in/yaml.v2"
)

var (
	flgConfigPath = flag.String("config", "/etc/cke.yml", "configuration file path")
)

func loadConfig(p string) (*cke.EtcdConfig, error) {
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	cfg := cke.NewEtcdConfig()
	err = yaml.NewDecoder(f).Decode(cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func main() {
	subcommands.Register(subcommands.HelpCommand(), "misc")
	subcommands.Register(subcommands.FlagsCommand(), "misc")
	subcommands.Register(subcommands.CommandsCommand(), "misc")
	subcommands.Register(cli.ClusterCommand(), "")

	flag.Parse()
	cmd.LogConfig{}.Apply()

	cfg, err := loadConfig(*flgConfigPath)
	if err != nil {
		log.ErrorExit(err)
	}

	etcd, err := cfg.Client()
	if err != nil {
		log.ErrorExit(err)
	}
	defer etcd.Close()

	storage := cke.Storage{etcd}
	cli.Setup(storage)

	exitStatus := subcommands.ExitSuccess
	cmd.Go(func(ctx context.Context) error {
		exitStatus = subcommands.Execute(ctx)
		return nil
	})
	cmd.Stop()
	cmd.Wait()
	os.Exit(int(exitStatus))
}

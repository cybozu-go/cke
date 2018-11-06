package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/cli"
	"github.com/cybozu-go/etcdutil"
	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
	"github.com/google/subcommands"
	yaml "gopkg.in/yaml.v2"
)

var (
	flgConfigPath = flag.String("config", "/etc/cke/config.yml", "configuration file path")
	flgVersion    = flag.Bool("version", false, "show ckecli version")
)

func loadConfig(p string) (*etcdutil.Config, error) {
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
	subcommands.Register(cli.ConstraintsCommand(), "")
	subcommands.Register(cli.VaultCommand(), "")
	subcommands.Register(cli.CACommand(), "")
	subcommands.Register(cli.LeaderCommand(), "")
	subcommands.Register(cli.HistoryCommand(), "")
	subcommands.Register(cli.EtcdCommand(), "")
	subcommands.Register(cli.KubernetesCommand(), "")

	flag.Parse()
	well.LogConfig{}.Apply()

	if *flgVersion {
		fmt.Println(cke.Version)
		os.Exit(0)
	}

	cfg, err := loadConfig(*flgConfigPath)
	if err != nil {
		log.ErrorExit(err)
	}

	etcd, err := etcdutil.NewClient(cfg)
	if err != nil {
		log.ErrorExit(err)
	}
	defer etcd.Close()

	storage := cke.Storage{Client: etcd}
	cli.Setup(storage)

	exitStatus := subcommands.ExitSuccess
	well.Go(func(ctx context.Context) error {
		exitStatus = subcommands.Execute(ctx)
		return nil
	})
	well.Stop()
	well.Wait()
	os.Exit(int(exitStatus))
}

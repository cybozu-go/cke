package main

import (
	"flag"
	"os"
	"time"

	"github.com/coreos/etcd/clientv3/concurrency"
	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cmd"
	"github.com/cybozu-go/log"
	"gopkg.in/yaml.v2"
)

var (
	flgConfigPath = flag.String("config", "/etc/cke.yml", "configuration file path")
	flgInterval   = flag.String("interval", "10m", "check interval")
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
	flag.Parse()
	cmd.LogConfig{}.Apply()

	interval, err := time.ParseDuration(*flgInterval)
	if err != nil {
		log.ErrorExit(err)
	}

	cfg, err := loadConfig(*flgConfigPath)
	if err != nil {
		log.ErrorExit(err)
	}

	etcd, err := cfg.Client()
	if err != nil {
		log.ErrorExit(err)
	}
	defer etcd.Close()

	session, err := concurrency.NewSession(etcd)
	if err != nil {
		log.ErrorExit(err)
	}
	controller := cke.NewController(session, interval)

	cmd.Go(controller.Run)
	err = cmd.Wait()
	if err != nil && !cmd.IsSignaled(err) {
		log.ErrorExit(err)
	}
}

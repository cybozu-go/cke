package main

import (
	"os"
	"time"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/localproxy"
	"github.com/cybozu-go/etcdutil"
	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
	"github.com/spf13/pflag"
	"sigs.k8s.io/yaml"
)

var (
	flgConfigPath = pflag.String("config", "/etc/cke/config.yml", "configuration file path")
	flgInterval   = pflag.Duration("interval", 1*time.Minute, "check interval")
)

func loadConfig(p string) (*etcdutil.Config, error) {
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	cfg := cke.NewEtcdConfig()
	err = yaml.Unmarshal(b, cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func main() {
	pflag.Parse()
	well.LogConfig{}.Apply()

	cfg, err := loadConfig(*flgConfigPath)
	if err != nil {
		log.ErrorExit(err)
	}

	etcd, err := etcdutil.NewClient(cfg)
	if err != nil {
		log.ErrorExit(err)
	}
	defer etcd.Close()

	// Controller
	controller := localproxy.LocalProxy{Interval: *flgInterval, Storage: cke.Storage{Client: etcd}}
	well.Go(controller.Run)

	err = well.Wait()
	if err != nil && !well.IsSignaled(err) {
		log.ErrorExit(err)
	}
}

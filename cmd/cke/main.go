package main

import (
	"flag"
	"net/http"
	"os"
	"time"

	"github.com/coreos/etcd/clientv3/concurrency"
	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cmd"
	"github.com/cybozu-go/etcdutil"
	"github.com/cybozu-go/log"
	"gopkg.in/yaml.v2"
)

var (
	flgHTTP       = flag.String("http", "0.0.0.0:10180", "<Listen IP>:<Port number>")
	flgConfigPath = flag.String("config", "/etc/cke.yml", "configuration file path")
	flgInterval   = flag.String("interval", "10m", "check interval")
	flgSessionTTL = flag.String("session-ttl", "60s", "leader session's TTL")
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
	flag.Parse()
	cmd.LogConfig{}.Apply()

	interval, err := time.ParseDuration(*flgInterval)
	if err != nil {
		log.ErrorExit(err)
	}

	ttl, err := time.ParseDuration(*flgSessionTTL)
	if err != nil {
		log.ErrorExit(err)
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

	session, err := concurrency.NewSession(etcd, concurrency.WithTTL(int(ttl.Seconds())))
	if err != nil {
		log.ErrorExit(err)
	}
	controller := cke.NewController(session, interval)

	cmd.Go(controller.Run)
	server := cke.Server{
		EtcdClient: etcd,
	}
	s := &cmd.HTTPServer{
		Server: &http.Server{
			Addr:    *flgHTTP,
			Handler: server,
		},
		ShutdownTimeout: 3 * time.Minute,
	}
	s.ListenAndServe()
	err = cmd.Wait()
	if err != nil && !cmd.IsSignaled(err) {
		log.ErrorExit(err)
	}
}

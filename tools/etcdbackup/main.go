package main

import (
	"errors"
	"flag"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
	"sigs.k8s.io/yaml"
)

var flgConfig = flag.String("config", "", "path to configuration file")

func main() {
	flag.Parse()
	well.LogConfig{}.Apply()

	if *flgConfig == "" {
		log.ErrorExit(errors.New("usage: etcdbackup -config=<CONFIGFILE>"))
	}

	b, err := ioutil.ReadFile(*flgConfig)
	if err != nil {
		log.ErrorExit(err)
	}
	cfg := NewConfig()
	err = yaml.Unmarshal(b, cfg)
	if err != nil {
		log.ErrorExit(err)
	}

	server := NewServer(cfg)
	s := &well.HTTPServer{
		Server: &http.Server{
			Addr:    cfg.Listen,
			Handler: server,
		},
		ShutdownTimeout: 3 * time.Minute,
	}

	log.Info("started", map[string]interface{}{
		"listen": cfg.Listen,
	})

	err = s.ListenAndServe()
	if err != nil {
		log.ErrorExit(err)
	}

	err = well.Wait()
	if err != nil && !well.IsSignaled(err) {
		log.ErrorExit(err)
	}
}

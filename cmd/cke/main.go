package main

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/cybozu-go/etcdutil"
	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
	"github.com/spf13/pflag"
	"go.etcd.io/etcd/client/v3/concurrency"
	"sigs.k8s.io/yaml"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/internal/sabakan"
	"github.com/cybozu-go/cke/internal/server"
)

var (
	flgHTTP                 = pflag.String("http", "0.0.0.0:10180", "<Listen IP>:<Port number>")
	flgConfigPath           = pflag.String("config", "/etc/cke/config.yml", "configuration file path")
	flgInterval             = pflag.String("interval", "1m", "check interval")
	flgCertsGCInterval      = pflag.String("certs-gc-interval", "1h", "tidy interval for expired certificates")
	flgSessionTTL           = pflag.String("session-ttl", "60s", "leader session's TTL")
	flgDebugSabakan         = pflag.Bool("debug-sabakan", false, "debug sabakan integration")
	flgMaxConcurrentUpdates = pflag.Int("max-concurrent-updates", 10, "the maximum number of components that can be updated simultaneously")
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

func debugSabakan(addon server.Integrator) {
	well.Go(func(ctx context.Context) error {
		ctx = context.WithValue(ctx, sabakan.WaitSecs, float64(5))
		return server.RunIntegrator(ctx, addon)
	})
	well.Stop()
	err := well.Wait()
	if err != nil && !well.IsSignaled(err) {
		log.ErrorExit(err)
	}
}

func main() {
	pflag.Parse()
	well.LogConfig{}.Apply()

	interval, err := time.ParseDuration(*flgInterval)
	if err != nil {
		log.ErrorExit(err)
	}

	gcInterval, err := time.ParseDuration(*flgCertsGCInterval)
	if err != nil {
		log.ErrorExit(err)
	}
	if gcInterval <= 0 {
		log.ErrorExit(errors.New("certs-gc-interval must be greater than 0"))
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

	addon := sabakan.NewIntegrator(etcd)
	if *flgDebugSabakan {
		debugSabakan(addon)
		return
	}

	session, err := concurrency.NewSession(etcd, concurrency.WithTTL(int(ttl.Seconds())))
	if err != nil {
		log.ErrorExit(err)
	}
	defer func() {
		// Checking the session to avoid an error caused by duplicated closing.
		select {
		case <-session.Done():
			return
		default:
			session.Close()
		}
	}()

	maxConcurrentUpdates := *flgMaxConcurrentUpdates
	if maxConcurrentUpdates <= 0 {
		log.ErrorExit(errors.New("max-concurrent-updates must be greater than 0"))
	}

	// Controller
	controller := server.NewController(session, addon, &server.Config{
		Interval:             interval,
		CertsGCInterval:      gcInterval,
		MaxConcurrentUpdates: maxConcurrentUpdates,
	})
	well.Go(controller.Run)

	// API server
	well.Go(func(ctx context.Context) error {
		return runHTTPServer(ctx, *flgHTTP, etcd)
	})
	err = well.Wait()
	if err != nil && !well.IsSignaled(err) {
		log.ErrorExit(err)
	}
}

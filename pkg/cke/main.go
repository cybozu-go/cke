package main

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/metrics"
	"github.com/cybozu-go/cke/sabakan"
	"github.com/cybozu-go/cke/server"
	"github.com/cybozu-go/etcdutil"
	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
	"github.com/spf13/pflag"
	"go.etcd.io/etcd/clientv3/concurrency"
	"sigs.k8s.io/yaml"
)

var (
	flgHTTP            = pflag.String("http", "0.0.0.0:10180", "<Listen IP>:<Port number>")
	flgConfigPath      = pflag.String("config", "/etc/cke/config.yml", "configuration file path")
	flgInterval        = pflag.String("interval", "1m", "check interval")
	flgCertsGCInterval = pflag.String("certs-gc-interval", "1h", "tidy interval for expired certificates")
	flgSessionTTL      = pflag.String("session-ttl", "60s", "leader session's TTL")
	flgDebugSabakan    = pflag.Bool("debug-sabakan", false, "debug sabakan integration")
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

	timeout, err := time.ParseDuration(cfg.Timeout)
	if err != nil {
		log.ErrorExit(err)
	}

	// Controller
	controller := server.NewController(session, interval, gcInterval, timeout, addon)
	well.Go(controller.Run)

	// API server
	mux := http.NewServeMux()
	// Metrics
	collector := metrics.NewCollector(etcd)
	metricsHandler := metrics.GetHandler(collector)
	mux.Handle("/metrics", metricsHandler)
	// REST API
	server := server.Server{
		EtcdClient: etcd,
		Timeout:    timeout,
	}
	mux.Handle("/", server)
	s := &well.HTTPServer{
		Server: &http.Server{
			Addr:    *flgHTTP,
			Handler: mux,
		},
		ShutdownTimeout: 3 * time.Minute,
	}
	s.ListenAndServe()
	err = well.Wait()
	if err != nil && !well.IsSignaled(err) {
		log.ErrorExit(err)
	}
}

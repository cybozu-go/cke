package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"
)

var (
	flgListen          = flag.String("listen", "", "Listen address and port (address:port)")
	flgUpstreams       = flag.String("upstreams", "", "Comma-separated upstream servers (addr1:port1,addr2:port2)")
	flgShutdownTimeout = flag.String("shutdown-timeout", "10s", "Timeout for server shutting-down gracefully (disabled if specified \"0\")")
	flgDialTimeout     = flag.String("dial-timeout", "10s", "Timeout for dial to an upstream server")
	flgDialKeepAlive   = flag.String("dial-keep-alive", "15s", "Interval between keep-alive probes")
	flgCheckInterval   = flag.String("check-interval", "20s", "Interval for health check")
)

func main() {
	flag.Parse()

	if len(*flgUpstreams) == 0 {
		slog.Error("rivers exited with an error", "error", "--upstreams is blank")
		os.Exit(1)
	}
	upstreamAddrs := strings.Split(*flgUpstreams, ",")
	upstreams := make([]*Upstream, len(upstreamAddrs))
	for i, a := range upstreamAddrs {
		upstreams[i] = &Upstream{
			address: a,
			conns:   make(map[net.Conn]func()),
		}
	}

	dialer := &net.Dialer{}
	var err error
	dialer.Timeout, err = time.ParseDuration(*flgDialTimeout)
	if err != nil {
		slog.Error("rivers exited with an error", "error", fmt.Errorf("--dial-timeout: %w", err))
		os.Exit(1)
	}
	dialer.KeepAlive, err = time.ParseDuration(*flgDialKeepAlive)
	if err != nil {
		slog.Error("rivers exited with an error", "error", fmt.Errorf("--dial-keep-alive: %w", err))
		os.Exit(1)
	}

	serverCfg := ServerConfig{Dialer: dialer}
	serverCfg.ShutdownTimeout, err = time.ParseDuration(*flgShutdownTimeout)
	if err != nil {
		slog.Error("rivers exited with an error", "error", fmt.Errorf("--shutdown-timeout: %w", err))
		os.Exit(1)
	}

	healthCfg := HealthCheckerConfig{Dialer: dialer}
	healthCfg.CheckInterval, err = time.ParseDuration(*flgCheckInterval)
	if err != nil {
		slog.Error("rivers exited with an error", "error", fmt.Errorf("--check-interval: %w", err))
		os.Exit(1)
	}

	if len(*flgListen) == 0 {
		slog.Error("rivers exited with an error", "error", "--listen is blank")
		os.Exit(1)
	}

	slog.Info("rivers started", "listen", *flgListen, "upstreams", upstreamAddrs)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	waitSignalLog := logReceivedSignal(ctx)

	err = run(ctx, upstreams, healthCfg, *flgListen, serverCfg)

	// Stop and wait for the log output goroutine to finish.
	stop()
	waitSignalLog()

	if err != nil {
		slog.Error("rivers exited with an error", "error", err)
		os.Exit(1)
	}

	slog.Info("rivers stopped")
}

// logReceivedSignal starts a goroutine that logs a received signal.
// It returns a function that blocks until that goroutine has finished.
func logReceivedSignal(ctx context.Context) (wait func()) {
	var wg sync.WaitGroup
	wg.Go(func() {
		<-ctx.Done()
		if cause := context.Cause(ctx); cause != context.Canceled {
			slog.Warn("received signal, shutting down", "cause", cause)
		}
	})
	return wg.Wait
}

// run runs rivers until ctx is canceled or an unrecoverable error occurs.
func run(ctx context.Context, upstreams []*Upstream, healthCfg HealthCheckerConfig, listenAddr string, serverCfg ServerConfig) error {
	listen, err := (&net.ListenConfig{}).Listen(ctx, "tcp", listenAddr)
	if err != nil {
		return err
	}

	hc := NewHealthChecker(upstreams, healthCfg)
	s := NewServer(upstreams, serverCfg)

	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error { return hc.Run(ctx) })
	eg.Go(func() error { return s.Serve(ctx, listen) })
	return eg.Wait()
}

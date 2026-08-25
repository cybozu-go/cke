package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"sync"
	"time"
)

// HealthCheckerConfig represents configuration for health checker
type HealthCheckerConfig struct {
	Dialer        Dialer
	Logger        *slog.Logger
	CheckInterval time.Duration
}

// HealthChecker represents upstream health checker
type HealthChecker struct {
	dialer        Dialer
	logger        *slog.Logger
	upstreams     []*Upstream
	checkInterval time.Duration
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(upstreams []*Upstream, cfg HealthCheckerConfig) *HealthChecker {
	dialer := cfg.Dialer
	if dialer == nil {
		dialer = &net.Dialer{}
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return &HealthChecker{
		dialer:        dialer,
		logger:        logger,
		checkInterval: cfg.CheckInterval,
		upstreams:     upstreams,
	}
}

// Run runs health checking until ctx is canceled. It always returns nil.
func (hc *HealthChecker) Run(ctx context.Context) error {
	hc.doHealthCheck(ctx, true)
	ticker := time.NewTicker(hc.checkInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			hc.doHealthCheck(ctx, false)
		}
	}
}

// doHealthCheck do actual health checking at each interval
func (hc *HealthChecker) doHealthCheck(ctx context.Context, first bool) {
	var wg sync.WaitGroup

	for _, u := range hc.upstreams {
		wg.Go(func() {
			conn, err := hc.dialer.DialContext(ctx, "tcp", u.address)
			if errors.Is(err, context.Canceled) {
				return
			}

			if err != nil {
				switch {
				case first:
					hc.logger.Error("initial health check: upstream is unhealthy", "address", u.address, "error", err)
					u.SetHealthy(false)
				case u.IsHealthy():
					hc.logger.Error("upstream health changed: now unhealthy", "address", u.address, "error", err)
					u.SetHealthy(false)
				}
				return
			}

			conn.Close()
			switch {
			case first:
				hc.logger.Info("initial health check: upstream is healthy", "address", u.address)
				u.SetHealthy(true)
			case !u.IsHealthy():
				hc.logger.Info("upstream health changed: now healthy", "address", u.address)
				u.SetHealthy(true)
			}
		})
	}

	wg.Wait()
}

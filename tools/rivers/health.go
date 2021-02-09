package main

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
)

// HealthCheckerConfig represents configuration for health checker
type HealthCheckerConfig struct {
	CheckInterval time.Duration
	Logger        *log.Logger
	Dialer        *net.Dialer
}

// HealthChecker represents upstream health checker
type HealthChecker struct {
	upstreams     []*Upstream
	checkInterval time.Duration
	logger        *log.Logger
	dialer        *net.Dialer
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(upstreams []*Upstream, cfg HealthCheckerConfig) *HealthChecker {
	dialer := cfg.Dialer
	if dialer == nil {
		dialer = &net.Dialer{}
	}
	logger := cfg.Logger
	if logger == nil {
		logger = log.DefaultLogger()
	}

	return &HealthChecker{
		upstreams:     upstreams,
		checkInterval: cfg.CheckInterval,
		logger:        logger,
		dialer:        dialer,
	}
}

// Start starts health checking
func (hc *HealthChecker) Start() {
	well.Go(func(ctx context.Context) error {
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
	})
}

// doHealthCheck do actual health checking at each interval
func (hc *HealthChecker) doHealthCheck(ctx context.Context, first bool) {
	var wg sync.WaitGroup
	for _, tu := range hc.upstreams {
		wg.Add(1)

		go func(u *Upstream) {
			defer wg.Done()

			conn, err := hc.dialer.DialContext(ctx, "tcp", u.address)
			if errors.Is(err, context.Canceled) {
				return
			}

			if err == nil {
				conn.Close()
				if first || !u.IsHealthy() {
					hc.logger.Info("an upstream becomes healthy", map[string]interface{}{
						"address": u.address,
					})
					u.SetHealthy(true)
				}
				return
			}

			if first || u.IsHealthy() {
				hc.logger.Error("an upstream becomes unhealthy", map[string]interface{}{
					log.FnError: err.Error(),
					"address":   u.address,
				})
				u.SetHealthy(false)
			}
		}(tu)
	}

	wg.Wait()
}

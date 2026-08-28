package main

import (
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/onsi/gomega/gbytes"
)

func TestHealthChecker(t *testing.T) {
	upstreams := []*Upstream{NewUpstream("0")}
	dialer := &testDialer{}
	buf := gbytes.NewBuffer()
	logger := slog.New(slog.NewTextHandler(buf, nil))
	cfg := HealthCheckerConfig{
		Dialer:        dialer,
		Logger:        logger,
		CheckInterval: time.Millisecond * 100,
	}
	hc := NewHealthChecker(upstreams, cfg)
	go func() {
		_ = hc.Run(t.Context())
	}()

	time.Sleep(time.Millisecond * 200)
	if !upstreams[0].IsHealthy() {
		t.Errorf("HealthChecker did not change upstream healthy\n")
	}
	if !strings.Contains(string(buf.Contents()), "initial health check") {
		t.Errorf("HealthChecker did not output initial status log")
	}

	if err := buf.Clear(); err != nil {
		t.Fatal(err)
	}
	dialer.SetErrorAddress("0")
	time.Sleep(time.Millisecond * 300)
	if upstreams[0].IsHealthy() {
		t.Errorf("HealthChecker did not change upstream unhealthy\n")
	}
	if !strings.Contains(string(buf.Contents()), "health changed: now unhealthy") {
		t.Errorf("HealthChecker did not output status change log")
	}
}

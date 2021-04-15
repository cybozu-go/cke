package main

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/cybozu-go/log"
)

func TestHealthChecker(t *testing.T) {
	upstreams := []*Upstream{{
		address: "0",
	}}
	dialer := &testDialer{}
	logger := log.NewLogger()
	buf := &bytes.Buffer{}
	logger.SetOutput(buf)
	cfg := HealthCheckerConfig{
		CheckInterval: time.Millisecond * 100,
		Dialer:        dialer,
		Logger:        logger,
	}
	hc := NewHealthChecker(upstreams, cfg)
	hc.Start()

	time.Sleep(time.Millisecond * 200)
	if !upstreams[0].IsHealthy() {
		t.Errorf("HealthChecker did not change upstream healthy\n")
	}
	if !strings.Contains(buf.String(), "becomes healthy") {
		t.Errorf("HealthChecker did not output status change log")
	}

	buf = &bytes.Buffer{}
	logger.SetOutput(buf)
	dialer.SetErrorAddress("0")
	time.Sleep(time.Millisecond * 300)
	if upstreams[0].IsHealthy() {
		t.Errorf("HealthChecker did not change upstream unhealthy\n")
	}
	if !strings.Contains(buf.String(), "becomes unhealthy") {
		t.Errorf("HealthChecker did not output status change log")
	}
}

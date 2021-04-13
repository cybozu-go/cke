package main

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/cybozu-go/log"
)

func TestEmptyServer(t *testing.T) {
	cfg := Config{
		ShutdownTimeout: time.Second,
	}
	s := NewServer(nil, cfg)
	_, _, err := s.randomUpstream()
	if err == nil {
		t.Errorf("empty server should return error for randomUpstream()\n")
	}
}

func TestServerWithUnhealthyUpstream(t *testing.T) {
	upstreams := []*Upstream{{
		address: "0",
	}}
	logger := log.NewLogger()
	buf := &bytes.Buffer{}
	logger.SetOutput(buf)
	cfg := Config{
		ShutdownTimeout: time.Second,
		Dialer:          &testDialer{},
		Logger:          logger,
	}
	s := NewServer(upstreams, cfg)
	_, _, err := s.randomUpstream()
	if err == nil {
		t.Errorf("unhealthy upstream server should return error for randomUpstream()\n")
	}
	if buf.String() != "" {
		t.Errorf("unhealthy upstream server should not output any log\n")
	}
}

func TestServerWithUnconnectableUpstream(t *testing.T) {
	upstreams := []*Upstream{{
		address: "0",
	}}
	upstreams[0].SetHealthy(true)
	logger := log.NewLogger()
	buf := &bytes.Buffer{}
	logger.SetOutput(buf)
	cfg := Config{
		ShutdownTimeout: time.Second,
		Dialer: &testDialer{
			errorAddress: "0",
		},
		Logger: logger,
	}
	s := NewServer(upstreams, cfg)
	_, _, err := s.randomUpstream()
	if err == nil {
		t.Errorf("unconnectable upstream server should return error for randomUpstream()\n")
	}
	if !strings.Contains(buf.String(), "warning: \"failed to connect") {
		t.Errorf("unconnectable upstream server should output warning log\n")
	}
}

func TestServerRandomUpstream(t *testing.T) {
	upstreams := []*Upstream{
		{
			address: "0",
		},
		{
			address: "1",
		},
		{
			address: "2",
		},
	}
	for _, u := range upstreams {
		u.SetHealthy(true)
	}
	logger := log.NewLogger()
	logger.SetOutput(nil)
	cfg := Config{
		ShutdownTimeout: time.Second,
		Dialer: &testDialer{
			errorAddress: "1",
		},
		Logger: logger,
	}
	s := NewServer(upstreams, cfg)

	histogram := map[*Upstream]int{}
	for i := 0; i < 1000; i++ {
		conn, u, err := s.randomUpstream()
		if err != nil {
			t.Errorf("randomUpstream() should not return error in this case.\n")
			break
		}
		conn.Close()
		histogram[u]++
	}
	if len(histogram) != 2 {
		t.Errorf("randomUpstream() should not return non-connectable upstream.\n")
	}
	if histogram[upstreams[0]] < 400 || histogram[upstreams[2]] < 400 {
		t.Errorf("randomUpstream() should connect to each upstream uniformly\n")
	}

	upstreams[0].SetHealthy(false)
	for i := 0; i < 1000; i++ {
		conn, u, err := s.randomUpstream()
		if err != nil {
			t.Errorf("randomUpstream() should not return error in this case.\n")
			break
		}
		conn.Close()
		if u != upstreams[2] {
			t.Errorf("randomUpstream() should return healthy and connectable upstream.\n")
			break
		}
	}
}

package main

import (
	"bytes"
	"io"
	"log/slog"
	"strings"
	"testing"
)

func TestEmptyServer(t *testing.T) {
	s := NewServer(nil, ServerConfig{})
	_, _, err := s.randomUpstream()
	if err == nil {
		t.Errorf("empty server should return error for randomUpstream()\n")
	}
}

func TestServerWithUnhealthyUpstream(t *testing.T) {
	upstreams := []*Upstream{NewUpstream("0")}
	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(buf, nil))
	cfg := ServerConfig{
		Dialer: &testDialer{},
		Logger: logger,
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
	upstreams := []*Upstream{NewUpstream("0")}
	upstreams[0].SetHealthy(true)
	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(buf, nil))
	cfg := ServerConfig{
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
	if !strings.Contains(buf.String(), "level=WARN") || !strings.Contains(buf.String(), "failed to connect") {
		t.Errorf("unconnectable upstream server should output warning log\n")
	}
}

func TestServerRandomUpstream(t *testing.T) {
	upstreams := []*Upstream{
		NewUpstream("0"),
		NewUpstream("1"),
		NewUpstream("2"),
	}
	for _, u := range upstreams {
		u.SetHealthy(true)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := ServerConfig{
		Dialer: &testDialer{
			errorAddress: "1",
		},
		Logger: logger,
	}
	s := NewServer(upstreams, cfg)

	histogram := map[*Upstream]int{}
	for range 1000 {
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
	for range 1000 {
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

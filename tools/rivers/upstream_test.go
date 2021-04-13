package main

import (
	"net"
	"testing"
)

func TestUpstream(t *testing.T) {
	var upstream Upstream

	if upstream.IsHealthy() {
		t.Errorf("new upstream should be unhealthy\n")
	}

	upstream.SetHealthy(true)
	if !upstream.IsHealthy() {
		t.Errorf("upstream should become healthy by SetHealthy(true)\n")
	}

	upstream.SetHealthy(false)
	if upstream.IsHealthy() {
		t.Errorf("upstream should become unhealthy by SetHealthy(false)\n")
	}

	conn1, conn2 := net.Pipe()
	defer conn1.Close()
	defer conn2.Close()
	called1 := 0
	called2 := 0

	cancelFunc := func(x *int) func() {
		return func() {
			*x++
		}
	}

	upstream.AddConn(conn1, cancelFunc(&called1))
	upstream.SetHealthy(false)
	if called1 != 1 {
		t.Errorf("a cancel function should be called by SetHealthy(false): called1=%d\n", called1)
	}
	upstream.SetHealthy(false)
	if called1 != 1 {
		t.Errorf("all connections are removed by SetHealthy(false): called1=%d\n", called1)
	}

	upstream.AddConn(conn1, cancelFunc(&called1))
	upstream.AddConn(conn2, cancelFunc(&called2))
	upstream.SetHealthy(false)
	if called1 != 2 || called2 != 1 {
		t.Errorf("all cancel functions should be called by SetHealthy(false): called1=%d called2=%d\n", called1, called2)
	}

	upstream.AddConn(conn1, cancelFunc(&called1))
	upstream.AddConn(conn2, cancelFunc(&called2))
	upstream.RemoveConn(conn1)
	upstream.SetHealthy(false)
	if called1 != 2 || called2 != 2 {
		t.Errorf("the cancel function for removed conn should not be called by setHealthy(false): called1=%d called2=%d\n", called1, called2)
	}
}

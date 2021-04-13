package main

import (
	"context"
	"fmt"
	"net"
	"sync"
)

// testDialer is a pseudo Dialer used for DI
type testDialer struct {
	m            sync.Mutex
	errorAddress string
}

func (d *testDialer) Dial(network, address string) (net.Conn, error) {
	return d.DialContext(context.Background(), network, address)
}

func (d *testDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	d.m.Lock()
	defer d.m.Unlock()

	if address == d.errorAddress {
		return nil, fmt.Errorf("")
	}
	// return dummy connection
	conn1, conn2 := net.Pipe()
	defer conn2.Close()
	return conn1, nil
}

func (d *testDialer) SetErrorAddress(address string) {
	d.m.Lock()
	defer d.m.Unlock()

	d.errorAddress = address
}

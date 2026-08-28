package main

import (
	"net"
	"sync"
	"sync/atomic"
)

// Upstream represents upstream server
type Upstream struct {
	address string
	health  atomic.Int32 // must be accessed through SetHealthy / IsHealthy
	m       sync.Mutex
	conns   map[net.Conn]func()
}

// NewUpstream creates an Upstream for the given address.
func NewUpstream(address string) *Upstream {
	return &Upstream{address: address}
}

func (u *Upstream) SetHealthy(b bool) {
	if b {
		u.health.Store(1)
		return
	}

	u.health.Store(0)
	u.m.Lock()
	conns := u.conns
	u.conns = make(map[net.Conn]func())
	u.m.Unlock()

	for _, c := range conns {
		c()
	}
}

func (u *Upstream) IsHealthy() bool {
	return u.health.Load() != 0
}

func (u *Upstream) AddConn(conn net.Conn, cancelFunc func()) {
	u.m.Lock()
	defer u.m.Unlock()

	if u.conns == nil {
		u.conns = make(map[net.Conn]func())
	}
	u.conns[conn] = cancelFunc
}

func (u *Upstream) RemoveConn(conn net.Conn) {
	u.m.Lock()
	defer u.m.Unlock()

	delete(u.conns, conn)
}

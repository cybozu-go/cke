package main

import (
	"errors"
	"flag"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
)

var (
	flgListen          = flag.String("listen", "", "Listen address and port (address:port)")
	flgUpstreams       = flag.String("upstreams", "", "Comma-separated upstream servers (addr1:port1,addr2:port2)")
	flgShutdownTimeout = flag.String("shutdown-timeout", "10s", "Timeout for server shutting-down gracefully (disabled if specified \"0\")")
	flgDialTimeout     = flag.String("dial-timeout", "10s", "Timeout for dial to an upstream server")
	flgDialKeepAlive   = flag.String("dial-keep-alive", "15s", "Interval between keep-alive probes")
	flgCheckInterval   = flag.String("check-interval", "20s", "Interval for health check")
)

// Upstream represents upstream server
type Upstream struct {
	address string

	health int32 // must be accessed through SetHealthy / IsHealthy

	m     sync.Mutex
	conns map[net.Conn]func()
}

func (u *Upstream) SetHealthy(b bool) {
	if b {
		atomic.StoreInt32(&u.health, 1)
		return
	}

	atomic.StoreInt32(&u.health, 0)
	u.m.Lock()
	conns := u.conns
	u.conns = make(map[net.Conn]func())
	u.m.Unlock()

	for _, c := range conns {
		c()
	}
}

func (u *Upstream) IsHealthy() bool {
	return atomic.LoadInt32(&u.health) != 0
}

func (u *Upstream) AddConn(conn net.Conn, cancelFunc func()) {
	u.m.Lock()
	defer u.m.Unlock()

	u.conns[conn] = cancelFunc
}

func (u *Upstream) RemoveConn(conn net.Conn) {
	u.m.Lock()
	defer u.m.Unlock()

	delete(u.conns, conn)
}

func run() error {
	if len(*flgUpstreams) == 0 {
		return errors.New("--upstreams is blank")
	}
	upstreamAddresses := strings.Split(*flgUpstreams, ",")
	upstreams := make([]*Upstream, len(upstreamAddresses))
	for i, a := range upstreamAddresses {
		upstreams[i] = &Upstream{
			address: a,
			conns:   make(map[net.Conn]func()),
		}
	}

	var dialer = &net.Dialer{DualStack: true}
	var err error
	dialer.Timeout, err = time.ParseDuration(*flgDialTimeout)
	if err != nil {
		return err
	}
	dialer.KeepAlive, err = time.ParseDuration(*flgDialKeepAlive)
	if err != nil {
		return err
	}

	cfg := Config{Dialer: dialer}
	cfg.ShutdownTimeout, err = time.ParseDuration(*flgShutdownTimeout)
	if err != nil {
		return err
	}

	hcConfig := HealthCheckerConfig{Dialer: dialer}
	hcConfig.CheckInterval, err = time.ParseDuration(*flgCheckInterval)
	if err != nil {
		return err
	}

	if len(*flgListen) == 0 {
		return errors.New("--listen is blank")
	}
	listen, err := net.Listen("tcp", *flgListen)
	if err != nil {
		return err
	}

	hc := NewHealthChecker(upstreams, hcConfig)
	hc.Start()

	s := NewServer(upstreams, cfg)
	s.Serve(listen)

	well.Stop()
	return well.Wait()
}

func main() {
	flag.Parse()
	well.LogConfig{}.Apply()

	err := run()
	if err != nil && !well.IsSignaled(err) {
		log.ErrorExit(err)
	}
}

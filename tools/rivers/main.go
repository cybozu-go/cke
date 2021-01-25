package main

import (
	"errors"
	"flag"
	"net"
	"strings"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
)

var (
	flgListen          = flag.String("listen", "", "Listen address and port (address:port)")
	flgUpstreams       = flag.String("upstreams", "", "Comma-separated upstream servers (addr1:port1,addr2:port2")
	flgShutdownTimeout = flag.String("shutdown-timeout", "0", "Timeout for server shutting-down gracefully (disabled if specified \"0\")")
	flgDialTimeout     = flag.String("dial-timeout", "5s", "Timeout for dial to an upstream server")
	flgDialKeepAlive   = flag.String("dial-keep-alive", "3m", "Timeout for dial keepalive to an upstream server")
)

func run() error {
	if len(*flgUpstreams) == 0 {
		return errors.New("--upstreams is blank")
	}
	upstreams := strings.Split(*flgUpstreams, ",")

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

	if len(*flgListen) == 0 {
		return errors.New("--listen is blank")
	}
	listen, err := net.Listen("tcp", *flgListen)
	if err != nil {
		return err
	}

	s := NewServer(upstreams, cfg)
	s.Serve(listen)

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

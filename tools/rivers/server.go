package main

import (
	"context"
	"errors"
	"io"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/netutil"
	"github.com/cybozu-go/well"
)

const (
	copyBufferSize = 64 << 10
)

type Dialer interface {
	Dial(network, address string) (net.Conn, error)
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

// Config represents TCP servers
type Config struct {
	ShutdownTimeout time.Duration
	Logger          *log.Logger
	Dialer          Dialer
}

// Server represents TCP proxy server
type Server struct {
	well.Server

	upstreams []*Upstream
	logger    *log.Logger
	dialer    Dialer
	pool      sync.Pool
}

// NewServer creates a new Server
func NewServer(upstreams []*Upstream, cfg Config) *Server {
	dialer := cfg.Dialer
	if dialer == nil {
		dialer = &net.Dialer{}
	}
	logger := cfg.Logger
	if logger == nil {
		logger = log.DefaultLogger()
	}

	s := &Server{
		Server: well.Server{
			ShutdownTimeout: cfg.ShutdownTimeout,
		},

		upstreams: upstreams,
		logger:    logger,
		dialer:    dialer,
		pool: sync.Pool{
			New: func() interface{} {
				buf := make([]byte, copyBufferSize)
				return &buf
			},
		},
	}
	s.Server.Handler = s.handleConnection

	return s
}

func (s *Server) handleConnection(ctx context.Context, conn net.Conn) {
	fields := well.FieldsFromContext(ctx)
	fields[log.FnType] = "access"
	fields["client_addr"] = conn.RemoteAddr().String()

	tc, ok := conn.(*net.TCPConn)
	if !ok {
		s.logger.Error("non-TCP connection", map[string]interface{}{
			"conn": conn,
		})
		return
	}

	destConn, u, err := s.randomUpstream()
	if err != nil {
		fields[log.FnError] = err.Error()
		s.logger.Error("failed to connect to upstream servers", fields)
		return
	}
	defer destConn.Close()

	u.AddConn(conn, func() {
		conn.Close()
		destConn.Close()
	})
	defer u.RemoveConn(conn)

	st := time.Now()
	env := well.NewEnvironment(ctx)
	env.Go(func(_ context.Context) error {
		buf := s.pool.Get().(*[]byte)
		_, err := io.CopyBuffer(destConn, tc, *buf)
		s.pool.Put(buf)
		if hc, ok := destConn.(netutil.HalfCloser); ok {
			hc.CloseWrite()
		}
		tc.CloseRead()
		return err
	})
	env.Go(func(_ context.Context) error {
		buf := s.pool.Get().(*[]byte)
		_, err := io.CopyBuffer(tc, destConn, *buf)
		s.pool.Put(buf)
		tc.CloseWrite()
		if hc, ok := destConn.(netutil.HalfCloser); ok {
			hc.CloseRead()
		}
		return err
	})
	env.Stop()
	err = env.Wait()

	fields = well.FieldsFromContext(ctx)
	fields["elapsed"] = time.Since(st).Seconds()
	if err != nil {
		fields[log.FnError] = err.Error()
		s.logger.Error("proxy ends with an error", fields)
		return
	}
	s.logger.Info("proxy ends", fields)
}

func (s *Server) randomUpstream() (net.Conn, *Upstream, error) {
	ups := make([]*Upstream, len(s.upstreams))
	copy(ups, s.upstreams)
	rand.Shuffle(len(ups), func(i, j int) {
		ups[i], ups[j] = ups[j], ups[i]
	})
	for _, u := range ups {
		if !u.IsHealthy() {
			continue
		}

		a := u.address
		conn, err := s.dialer.Dial("tcp", a)
		if err == nil {
			return conn, u, nil
		}

		s.logger.Warn("failed to connect to proxy server", map[string]interface{}{
			"upstream": a,
		})
	}
	return nil, nil, errors.New("no available upstreams servers")
}

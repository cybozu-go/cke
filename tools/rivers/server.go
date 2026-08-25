package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"math/rand"
	"net"
	"slices"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

const (
	copyBufferSize     = 64 << 10
	tcpKeepAlivePeriod = 3 * time.Minute
)

type Dialer interface {
	Dial(network, address string) (net.Conn, error)
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

// halfCloser is satisfied by connections that can be half-closed, such as
// *net.TCPConn and *net.UnixConn. It lets randomUpstream's Dialer return
// non-TCP connections; half-close is just skipped for anything that
// doesn't support it.
type halfCloser interface {
	CloseRead() error
	CloseWrite() error
}

// ServerConfig represents TCP servers
type ServerConfig struct {
	Dialer          Dialer
	Logger          *slog.Logger
	ShutdownTimeout time.Duration
}

// Server represents TCP proxy server
type Server struct {
	dialer          Dialer
	logger          *slog.Logger
	shutdownTimeout time.Duration
	upstreams       []*Upstream
	pool            sync.Pool
}

// NewServer creates a new Server
func NewServer(upstreams []*Upstream, cfg ServerConfig) *Server {
	dialer := cfg.Dialer
	if dialer == nil {
		dialer = &net.Dialer{}
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return &Server{
		dialer:          dialer,
		logger:          logger,
		shutdownTimeout: cfg.ShutdownTimeout,
		upstreams:       upstreams,
		pool: sync.Pool{
			New: func() any {
				buf := make([]byte, copyBufferSize)
				return &buf
			},
		},
	}
}

// Serve accepts connections on l, handling each in its own goroutine, until
// ctx is canceled.  It then blocks until all connections are closed, or
// shutdownTimeout elapses, whichever comes first.
//
// It returns nil when ctx is canceled, or the error from l.Accept if
// accepting fails for any other reason.
func (s *Server) Serve(ctx context.Context, l net.Listener) error {
	go func() {
		<-ctx.Done()
		l.Close()
	}()

	var acceptErr error
	var wg sync.WaitGroup
	for {
		conn, err := l.Accept()
		if err != nil {
			if ctx.Err() == nil {
				acceptErr = err
			}
			break
		}

		if tc, ok := conn.(*net.TCPConn); ok {
			_ = tc.SetKeepAlive(true)
			_ = tc.SetKeepAlivePeriod(tcpKeepAlivePeriod)
		}

		wg.Go(func() {
			defer conn.Close()
			s.handleConnection(conn)
		})
	}

	s.waitShutdown(&wg)
	return acceptErr
}

// waitShutdown blocks until wg is done, or s.shutdownTimeout elapses,
// whichever comes first.  A zero shutdownTimeout disables the timeout and
// waits indefinitely.
func (s *Server) waitShutdown(wg *sync.WaitGroup) {
	if s.shutdownTimeout == 0 {
		wg.Wait()
		return
	}

	ch := make(chan struct{})
	go func() {
		wg.Wait()
		close(ch)
	}()

	select {
	case <-ch:
	case <-time.After(s.shutdownTimeout):
		s.logger.Warn("timeout waiting for shutdown")
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	logger := s.logger.With("client_addr", conn.RemoteAddr().String())

	tc, ok := conn.(*net.TCPConn)
	if !ok {
		logger.Error("non-TCP connection", "conn", conn)
		return
	}

	destConn, u, err := s.randomUpstream()
	if err != nil {
		logger.Error("failed to connect to upstream servers", "error", err)
		return
	}
	defer destConn.Close()

	u.AddConn(conn, func() {
		conn.Close()
		destConn.Close()
	})
	defer u.RemoveConn(conn)

	st := time.Now()

	var eg errgroup.Group
	eg.Go(func() error {
		buf := s.pool.Get().(*[]byte)
		_, err := io.CopyBuffer(destConn, tc, *buf)
		s.pool.Put(buf)
		if hc, ok := destConn.(halfCloser); ok {
			_ = hc.CloseWrite()
		}
		_ = tc.CloseRead()
		return err
	})
	eg.Go(func() error {
		buf := s.pool.Get().(*[]byte)
		_, err := io.CopyBuffer(tc, destConn, *buf)
		s.pool.Put(buf)
		_ = tc.CloseWrite()
		if hc, ok := destConn.(halfCloser); ok {
			_ = hc.CloseRead()
		}
		return err
	})
	err = eg.Wait()

	elapsed := time.Since(st).Seconds()

	if err != nil {
		logger.Error("proxy ends with an error", "elapsed", elapsed, "error", err)
		return
	}
	logger.Info("proxy ends", "elapsed", elapsed)
}

func (s *Server) randomUpstream() (net.Conn, *Upstream, error) {
	ups := slices.Clone(s.upstreams)
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

		s.logger.Warn("failed to connect to proxy server", "upstream", a)
	}
	return nil, nil, errors.New("no available upstreams servers")
}

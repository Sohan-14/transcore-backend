package httpserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/sohan-14/transcore-backend/internal/platform/config"
)

// Server wraps net/http.Server with readiness draining and graceful shutdown.
type Server struct {
	httpServer *http.Server
	httpCfg    *config.HTTP
	log        *slog.Logger
	draining   atomic.Bool
	onDrain    []func()
}

// New creates a Server bound to cfg and handler.
// ===============================================================
func New(httpCfg *config.HTTP, log *slog.Logger, handler http.Handler) *Server {
	if log == nil {
		log = slog.Default()
	}
	return &Server{
		httpCfg: httpCfg,
		log:     log,
		httpServer: &http.Server{
			Addr:              httpCfg.Addr,
			Handler:           handler,
			ReadHeaderTimeout: httpCfg.ReadHeaderTimeout,
			ReadTimeout:       httpCfg.ReadTimeout,
			WriteTimeout:      httpCfg.WriteTimeout,
			IdleTimeout:       httpCfg.IdleTimeout,
			MaxHeaderBytes:    httpCfg.MaxHeaderBytes,
			ErrorLog:          slog.NewLogLogger(log.Handler(), slog.LevelError),
		},
	}
}

// OnDrain registers a callback executed as soon as shutdown begins, before
// waiting for DrainDelay. Call before Run.
// ===============================================================
func (s *Server) OnDrain(fn func()) {
	if fn != nil {
		s.onDrain = append(s.onDrain, fn)
	}
}

// IsDraining reports whether the server has started shutting down.
// ===============================================================
func (s *Server) IsDraining() bool {
	return s.draining.Load()
}

// Run binds the TCP address synchronously and serves until ctx is cancelled
// or a fatal server error occurs.
// ===============================================================
func (s *Server) Run(ctx context.Context) error {
	ln, err := net.Listen("tcp", s.httpServer.Addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", s.httpServer.Addr, err)
	}

	return s.serve(ctx, ln)
}

func (s *Server) serve(ctx context.Context, ln net.Listener) error {
	errCh := make(chan error, 1)

	go func() {
		errCh <- s.httpServer.Serve(ln)
	}()

	s.log.InfoContext(ctx, "http server listening", "addr", ln.Addr().String())

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve http: %w", err)
	case <-ctx.Done():
	}

	return s.shutdown(ctx)
}

func (s *Server) shutdown(ctx context.Context) error {
	s.draining.Store(true)
	for _, fn := range s.onDrain {
		fn()
	}

	logCtx := context.WithoutCancel(ctx)

	s.log.InfoContext(logCtx, "shutting down http server",
		"drain_delay", s.httpCfg.DrainDelay.String(),
		"shutdown_timeout", s.httpCfg.ShutdownTimeout.String(),
	)

	if s.httpCfg.DrainDelay > 0 {
		timer := time.NewTimer(s.httpCfg.DrainDelay)
		select {
		case <-timer.C:
		case <-ctx.Done():
			timer.Stop()
		}
	}

	shutdownCtx, cancel := context.WithTimeout(logCtx, s.httpCfg.ShutdownTimeout)
	defer cancel()

	if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
		_ = s.httpServer.Close()
		return fmt.Errorf("graceful shutdown: %w", err)
	}

	s.log.InfoContext(logCtx, "http server stopped")
	return nil
}

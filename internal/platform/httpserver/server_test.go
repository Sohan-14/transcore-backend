package httpserver

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sohan-14/transcore-backend/internal/platform/config"
)

func baseHTTPCfg(addr string) *config.HTTP {
	return &config.HTTP{
		Addr:              addr,
		ReadHeaderTimeout: 2 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
		IdleTimeout:       10 * time.Second,
		MaxHeaderBytes:    1 << 16,
		DrainDelay:        0,
		ShutdownTimeout:   2 * time.Second,
	}
}

func TestNewAndNilLogger(t *testing.T) {
	cfg := baseHTTPCfg("127.0.0.1:0")
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	srv := New(cfg, nil, handler)
	if srv == nil {
		t.Fatal("expected non-nil Server")
	}
	if srv.IsDraining() {
		t.Fatal("expected IsDraining() to be false initially")
	}
	if srv.httpServer.Addr != cfg.Addr {
		t.Errorf("Addr = %q, want %q", srv.httpServer.Addr, cfg.Addr)
	}
	if srv.httpServer.ReadHeaderTimeout != cfg.ReadHeaderTimeout {
		t.Errorf("ReadHeaderTimeout = %v, want %v", srv.httpServer.ReadHeaderTimeout, cfg.ReadHeaderTimeout)
	}
}

func TestOnDrainHooksAndOrder(t *testing.T) {
	srv := New(baseHTTPCfg("127.0.0.1:0"), nil, http.NewServeMux())

	var mu sync.Mutex
	var order []string

	srv.OnDrain(nil)
	srv.OnDrain(func() {
		mu.Lock()
		defer mu.Unlock()
		order = append(order, "first")
	})
	srv.OnDrain(func() {
		mu.Lock()
		defer mu.Unlock()
		order = append(order, "second")
	})

	ctx, cancel := context.WithCancel(context.Background())
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen failed: %v", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- srv.serve(ctx, ln)
	}()

	cancel()
	if err := <-done; err != nil {
		t.Fatalf("serve returned unexpected error: %v", err)
	}

	if !srv.IsDraining() {
		t.Fatal("expected IsDraining() to be true after shutdown")
	}

	mu.Lock()
	defer mu.Unlock()
	if len(order) != 2 || order[0] != "first" || order[1] != "second" {
		t.Fatalf("unexpected drain hook execution order: %v", order)
	}
}

func TestRunBindError(t *testing.T) {
	occupied, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to reserve test port: %v", err)
	}
	defer func() {
		_ = occupied.Close()
	}()

	srv := New(baseHTTPCfg(occupied.Addr().String()), nil, http.NewServeMux())
	err = srv.Run(context.Background())
	if err == nil {
		t.Fatal("expected error when binding an already-in-use port, got nil")
	}
}

func TestServeGracefulShutdownDrainsInFlightRequest(t *testing.T) {
	type reqResult struct {
		status int
		body   string
	}

	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, nil))

	started := make(chan struct{})
	release := make(chan struct{})

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(started)
		<-release
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("drained"))
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen failed: %v", err)
	}

	srv := New(baseHTTPCfg(ln.Addr().String()), log, handler)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serveErrCh := make(chan error, 1)
	go func() {
		serveErrCh <- srv.serve(ctx, ln)
	}()

	client := &http.Client{Timeout: 3 * time.Second}
	respCh := make(chan reqResult, 1)
	reqErrCh := make(chan error, 1)

	go func() {
		resp, err := client.Get("http://" + ln.Addr().String() + "/work")
		if err != nil {
			if resp != nil {
				_ = resp.Body.Close()
			}
			reqErrCh <- err
			return
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			reqErrCh <- err
			return
		}

		respCh <- reqResult{
			status: resp.StatusCode,
			body:   string(body),
		}
	}()

	<-started
	cancel()
	close(release)

	select {
	case err := <-reqErrCh:
		t.Fatalf("in-flight request failed: %v", err)
	case res := <-respCh:
		if res.status != http.StatusAccepted || res.body != "drained" {
			t.Fatalf("unexpected response: status=%d body=%q", res.status, res.body)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for in-flight request")
	}

	select {
	case err := <-serveErrCh:
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Fatalf("serve returned error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for server shutdown")
	}

	logs := buf.String()
	if !strings.Contains(logs, "http server listening") || !strings.Contains(logs, "http server stopped") {
		t.Fatalf("expected lifecycle log messages, got: %s", logs)
	}
}

func TestShutdownTimeoutForcesClose(t *testing.T) {
	started := make(chan struct{})
	blockForever := make(chan struct{})
	defer close(blockForever)

	handler := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		close(started)
		<-blockForever
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen failed: %v", err)
	}

	cfg := baseHTTPCfg(ln.Addr().String())
	cfg.ShutdownTimeout = 40 * time.Millisecond

	srv := New(cfg, nil, handler)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serveErrCh := make(chan error, 1)
	go func() {
		serveErrCh <- srv.serve(ctx, ln)
	}()

	go func() {
		reqCtx, reqCancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer reqCancel()
		req, _ := http.NewRequestWithContext(reqCtx, http.MethodGet, "http://"+ln.Addr().String()+"/hang", nil)
		resp, err := http.DefaultClient.Do(req)
		if err == nil && resp != nil {
			_ = resp.Body.Close()
		}
	}()

	<-started
	cancel()

	select {
	case err := <-serveErrCh:
		if err == nil {
			t.Fatal("expected error when graceful shutdown times out, got nil")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("shutdown hung past ShutdownTimeout")
	}
}

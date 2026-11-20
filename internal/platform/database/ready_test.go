package database

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestWaitReady(t *testing.T) {
	t.Run("succeeds on first ping", func(t *testing.T) {
		var calls atomic.Int32
		err := waitReady(context.Background(), func(_ context.Context) error {
			calls.Add(1)
			return nil
		}, nil, 10*time.Millisecond)

		if err != nil {
			t.Fatalf("expected nil error, got: %v", err)
		}
		if calls.Load() != 1 {
			t.Fatalf("expected 1 ping call, got %d", calls.Load())
		}
	})

	t.Run("retries transient errors and logs warnings", func(t *testing.T) {
		var buf bytes.Buffer
		log := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn}))

		var calls atomic.Int32
		errBoom := errors.New("connection refused")

		err := waitReady(context.Background(), func(_ context.Context) error {
			if calls.Add(1) < 3 {
				return errBoom
			}
			return nil
		}, log, 5*time.Millisecond)

		if err != nil {
			t.Fatalf("expected nil error after retries, got: %v", err)
		}
		if calls.Load() != 3 {
			t.Fatalf("expected 3 ping calls, got %d", calls.Load())
		}
		if !strings.Contains(buf.String(), "database not ready") {
			t.Fatalf("expected warning log when retrying, got: %s", buf.String())
		}
	})

	t.Run("returns error when context expires", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
		defer cancel()

		errDown := errors.New("still starting")
		err := waitReady(ctx, func(_ context.Context) error {
			return errDown
		}, nil, 10*time.Millisecond)

		if err == nil {
			t.Fatal("expected error when context times out, got nil")
		}
		if !errors.Is(err, errDown) {
			t.Fatalf("expected wrapped error %v, got: %v", errDown, err)
		}
	})
}

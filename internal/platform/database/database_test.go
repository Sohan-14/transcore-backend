package database

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/sohan-14/transcore-backend/internal/platform/config"
)

func TestBuildPoolConfig(t *testing.T) {
	t.Run("maps pool settings and jitter from config", func(t *testing.T) {
		cfg := config.Database{
			URL:             "postgres://app:secret@localhost:5432/transcore?sslmode=disable",
			MaxConns:        20,
			MinConns:        4,
			MaxConnLifetime: 1 * time.Hour,
			MaxConnIdleTime: 10 * time.Minute,
			ConnectTimeout:  5 * time.Second,
		}

		poolCfg, err := buildPoolConfig(cfg)
		if err != nil {
			t.Fatalf("buildPoolConfig returned unexpected error: %v", err)
		}

		if poolCfg.MaxConns != 20 {
			t.Errorf("MaxConns = %d, want 20", poolCfg.MaxConns)
		}
		if poolCfg.MinConns != 4 {
			t.Errorf("MinConns = %d, want 4", poolCfg.MinConns)
		}
		if poolCfg.MaxConnLifetime != 1*time.Hour {
			t.Errorf("MaxConnLifetime = %v, want 1h", poolCfg.MaxConnLifetime)
		}
		if poolCfg.MaxConnLifetimeJitter != 6*time.Minute {
			t.Errorf("MaxConnLifetimeJitter = %v, want 6m", poolCfg.MaxConnLifetimeJitter)
		}
		if poolCfg.MaxConnIdleTime != 10*time.Minute {
			t.Errorf("MaxConnIdleTime = %v, want 10m", poolCfg.MaxConnIdleTime)
		}
		if poolCfg.ConnConfig.ConnectTimeout != 5*time.Second {
			t.Errorf("ConnectTimeout = %v, want 5s", poolCfg.ConnConfig.ConnectTimeout)
		}
	})

	t.Run("returns error for malformed url", func(t *testing.T) {
		_, err := buildPoolConfig(config.Database{
			URL: "://not-a-valid-url",
		})
		if err == nil {
			t.Fatal("expected error for invalid URL, got nil")
		}
	})
}

func TestDatabaseNilPoolGuards(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, nil))

	db := &Database{
		pool:          nil,
		log:           log,
		healthTimeout: 2 * time.Second,
	}

	if db.Pool() != nil {
		t.Fatal("expected nil pool")
	}

	if err := db.Health(context.Background()); err == nil {
		t.Fatal("expected Health to return error when pool is nil")
	}

	// Should return safely without panicking when pool is nil.
	db.Close()
	if buf.Len() != 0 {
		t.Fatalf("expected no log output when pool is nil, got: %s", buf.String())
	}

	// Should also not panic when both pool and log are nil.
	(&Database{}).Close()
}

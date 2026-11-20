package database

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sohan-14/transcore-backend/internal/platform/config"
)

const defaultReadyInitialWait = 250 * time.Millisecond

// Database wraps a PostgreSQL connection pool and readiness checks.
type Database struct {
	pool          *pgxpool.Pool
	log           *slog.Logger
	healthTimeout time.Duration
}

// New creates a pgx connection pool from cfg and waits until the database
// responds to ping or ctx/startup deadline expires.
// ===============================================================
func New(ctx context.Context, cfg config.Database, log *slog.Logger) (*Database, error) {
	poolCfg, err := buildPoolConfig(cfg)
	if err != nil {
		return nil, err
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create database pool: %w", err)
	}

	startupTimeout := max(cfg.ConnectTimeout, cfg.StartupTimeout)
	readyCtx, cancel := context.WithTimeout(ctx, startupTimeout)
	defer cancel()

	if err := waitReady(readyCtx, pool.Ping, log, defaultReadyInitialWait); err != nil {
		pool.Close()
		return nil, err
	}

	if log != nil {
		log.InfoContext(ctx, "database connected",
			"host", poolCfg.ConnConfig.Host,
			"port", poolCfg.ConnConfig.Port,
			"database", poolCfg.ConnConfig.Database,
			"max_conns", poolCfg.MaxConns,
			"min_conns", poolCfg.MinConns,
		)
	}

	return &Database{
		pool:          pool,
		log:           log,
		healthTimeout: cfg.HealthTimeout,
	}, nil
}

// Pool exposes the underlying pgx connection pool.
// ===============================================================
func (db *Database) Pool() *pgxpool.Pool {
	return db.pool
}

// Health checks database connectivity within a bounded timeout.
// Intended for readiness checks (/readyz).
// ===============================================================
func (db *Database) Health(ctx context.Context) error {
	if db.pool == nil {
		return errors.New("database pool is nil")
	}

	healthCtx, cancel := context.WithTimeout(ctx, db.healthTimeout)
	defer cancel()

	if err := db.pool.Ping(healthCtx); err != nil {
		return fmt.Errorf("database health check: %w", err)
	}

	return nil
}

// Close closes all connections in the pool.
// Call after active handlers stop.
// ===============================================================
func (db *Database) Close() {
	if db.pool == nil {
		return
	}

	if db.log != nil {
		db.log.Info("closing database connection pool")
	}

	db.pool.Close()
}

func buildPoolConfig(cfg config.Database) (*pgxpool.Config, error) {

	poolCfg, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}

	poolCfg.MaxConns = cfg.MaxConns
	poolCfg.MinConns = cfg.MinConns
	poolCfg.MaxConnLifetime = cfg.MaxConnLifetime
	poolCfg.MaxConnLifetimeJitter = cfg.MaxConnLifetime / 10
	poolCfg.MaxConnIdleTime = cfg.MaxConnIdleTime
	connectTimeout := cfg.ConnectTimeout
	poolCfg.ConnConfig.ConnectTimeout = connectTimeout

	return poolCfg, nil
}

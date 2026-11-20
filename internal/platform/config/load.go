package config

import (
	"errors"
	"fmt"
	"log/slog"
	"time"
)

// Load reads and validates configuration
// using a two-phase process:
//  1. Read every environment variable,
//     aggregating all parse errors.
//  2. Validate the result against business
//     rules aggregating all violations.
//
// ===============================================================
func Load(getenv func(string) string) (Config, error) {
	r := newReader(getenv)

	cfg := Config{
		App:      loadApp(r),
		HTTP:     loadHTTP(r),
		Logger:   loadLogger(r),
		Database: loadDatabase(r),
	}

	if err := errors.Join(r.errs...); err != nil {
		return Config{}, fmt.Errorf("config:\n%w", err)
	}

	if err := cfg.validate(); err != nil {
		return Config{}, fmt.Errorf("config:\n%w", err)
	}

	return cfg, nil
}

// load app configuration from env
// ===============================================================
func loadApp(r *reader) App {
	return App{
		Name:        optional(r, "APP_NAME", "transcore-backend", asString),
		Environment: required(r, "APP_ENV", oneOf(Development, Staging, Production)),
	}
}

// load http server configuration from env
// ===============================================================
func loadHTTP(r *reader) HTTP {
	return HTTP{
		Addr:              required(r, "HTTP_ADDR", asString),
		ReadTimeout:       optional(r, "HTTP_READ_TIMEOUT", 15*time.Second, asDuration),
		ReadHeaderTimeout: optional(r, "HTTP_READ_HEADER_TIMEOUT", 5*time.Second, asDuration),
		WriteTimeout:      optional(r, "HTTP_WRITE_TIMEOUT", 30*time.Second, asDuration),
		IdleTimeout:       optional(r, "HTTP_IDLE_TIMEOUT", 60*time.Second, asDuration),
		ShutdownTimeout:   optional(r, "HTTP_SHUTDOWN_TIMEOUT", 20*time.Second, asDuration),
		MaxBodyBytes:      optional(r, "HTTP_MAX_BODY_BYTES", 1<<20, asInt64),
	}
}

// load logger configuration from env
// ===============================================================
func loadLogger(r *reader) Logger {
	return Logger{
		Level:     optional(r, "LOG_LEVEL", slog.LevelDebug, asLevel),
		Format:    optional(r, "LOG_FORMAT", LogFormatJSON, oneOf(LogFormatJSON, LogFormatText)),
		AddSource: optional(r, "LOG_ADD_SOURCE", false, asBool),
	}
}

// load database configuration from env
// ===============================================================
func loadDatabase(r *reader) Database {
	return Database{
		URL:             required(r, "DATABASE_URL", asString),
		MaxConns:        optional(r, "DB_MAX_CONNS", 10, asInt32),
		MinConns:        optional(r, "DB_MIN_CONNS", 2, asInt32),
		MaxConnLifetime: optional(r, "DB_MAX_CONN_LIFETIME", 30*time.Minute, asDuration),
		MaxConnIdleTime: optional(r, "DB_MAX_CONN_IDLE_TIME", 5*time.Minute, asDuration),
		ConnectTimeout:  optional(r, "DB_CONNECT_TIMEOUT", 5*time.Second, asDuration),
		StartupTimeout:  optional(r, "DB_STARTUP_TIMEOUT", 15*time.Second, asDuration),
		HealthTimeout:   optional(r, "DB_HEALTH_TIMEOUT", 3*time.Second, asDuration),
	}
}

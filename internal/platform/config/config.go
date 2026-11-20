package config

import (
	"log/slog"
	"time"
)

// Config holds all configuration sections for the application.
// ===============================================================
type Config struct {
	App      App
	HTTP     HTTP
	Logger   Logger
	Database Database
}

// Environment is the deployment environment the process runs in.
// App holds application-level settings.
// ===============================================================
type Environment string

const (
	// Development is used for local development.
	Development Environment = "development"
	// Staging is used for pre-production testing.
	Staging Environment = "staging"
	// Production is used for live production traffic.
	Production Environment = "production"
)

// App holds application level settings.
// ===============================================================
type App struct {
	Name        string
	Environment Environment
}

// HTTP holds HTTP server settings.
// ===============================================================
type HTTP struct {
	Addr              string
	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ShutdownTimeout   time.Duration
	MaxBodyBytes      int64
}

// LogFormat is the log output encoding.
// Logger holds logger config settings.
// ===============================================================
type LogFormat string

const (
	// LogFormatJSON formats logs as structured JSON.
	LogFormatJSON LogFormat = "json"
	// LogFormatText formats logs as human-readable text.
	LogFormatText LogFormat = "text"
)

// Logger holds logger config settings.
// ===============================================================
type Logger struct {
	Level     slog.Level
	Format    LogFormat
	AddSource bool
}

// Database holds database config settings.
// ===============================================================
type Database struct {
	URL             string
	MaxConns        int
	MinConns        int
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
	ConnectTimeout  time.Duration
}

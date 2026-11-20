package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const maxPoolConns = 1000

// validate runs business-rule validation
// across all configuration sections.
// ===============================================================
func (c *Config) validate() error {
	errs := []error{
		c.App.validate(),
		c.HTTP.validate(),
		c.Logger.validate(c.App.Environment),
		c.Database.validate(c.App.Environment),
	}

	return errors.Join(errs...)
}

// app validation: check app name validation
// ===============================================================
func (a App) validate() error {
	if strings.TrimSpace(a.Name) == "" {
		return fmt.Errorf("%s: must not be empty", "APP_NAME")
	}
	return nil
}

// http validation: check http addr and other values validation
// ===============================================================
func (h HTTP) validate() error {
	errs := []error{
		positive("HTTP_READ_TIMEOUT", h.ReadTimeout),
		positive("HTTP_READ_HEADER_TIMEOUT", h.ReadHeaderTimeout),
		positive("HTTP_WRITE_TIMEOUT", h.WriteTimeout),
		positive("HTTP_IDLE_TIMEOUT", h.IdleTimeout),
		positive("HTTP_SHUTDOWN_TIMEOUT", h.ShutdownTimeout),
	}

	if err := h.validateAddr(); err != nil {
		errs = append(errs, err)
	}

	if h.MaxBodyBytes <= 0 {
		errs = append(errs, fmt.Errorf("%s: must be greater than 0, got %d", "HTTP_MAX_BODY_BYTES", h.MaxBodyBytes))
	}

	if h.ReadHeaderTimeout > h.ReadTimeout {
		errs = append(errs, fmt.Errorf("%s (%v) must not exceed %s (%v)",
			"HTTP_READ_HEADER_TIMEOUT", h.ReadHeaderTimeout, "HTTP_READ_TIMEOUT", h.ReadTimeout))
	}

	return errors.Join(errs...)
}

// validateAddr: validate http Addr
// ===============================================================
func (h HTTP) validateAddr() error {
	_, port, err := net.SplitHostPort(h.Addr)
	if err != nil {
		return fmt.Errorf("%s: want host:port such as :8080: %w", "HTTP_ADDR", err)
	}

	n, err := strconv.Atoi(port)
	if err != nil || n < 0 || n > 65535 {
		return fmt.Errorf("%s: port %q must be a number from 0 to 65535", "HTTP_ADDR", port)
	}
	return nil
}

// logger validation: production log format validation
// ===============================================================
func (l Logger) validate(env Environment) error {
	if env == Production && l.Format != LogFormatJSON {
		return fmt.Errorf("%s=%s: %s must be %q", "APP_ENV", env, "LOG_FORMAT", LogFormatJSON)
	}
	return nil
}

// database validation: development and production both
// ===============================================================
func (db Database) validate(env Environment) error {
	errs := []error{
		positive("DB_MAX_CONN_LIFETIME", db.MaxConnLifetime),
		positive("DB_MAX_CONN_IDLE_TIME", db.MaxConnIdleTime),
		positive("DB_CONNECT_TIMEOUT", db.ConnectTimeout),
	}

	if err := db.validateURL(env); err != nil {
		errs = append(errs, err)
	}

	if db.MaxConns < 1 || db.MaxConns > maxPoolConns {
		errs = append(errs, fmt.Errorf("%s: must be between 1 and %d, got %d", "DB_MAX_CONNS", maxPoolConns, db.MaxConns))
	}
	if db.MinConns < 0 || db.MinConns > db.MaxConns {
		errs = append(errs, fmt.Errorf("%s: must be between 0 and %s (%d), got %d",
			"DB_MIN_CONNS", "DB_MAX_CONNS", db.MaxConns, db.MinConns))
	}

	return errors.Join(errs...)
}

// database url validation: checks the connection URL. It never
// includes the parse error or the URL itself for the securiry
// ===============================================================
func (db Database) validateURL(env Environment) error {
	u, err := url.Parse(db.URL)
	if err != nil {
		return fmt.Errorf("%s: not a valid URL (value hidden)", "DATABASE_URL")
	}

	if u.Scheme != "postgres" && u.Scheme != "postgresql" {
		return fmt.Errorf("%s: scheme must be postgres:// or postgresql://, got %q", "DATABASE_URL", u.Scheme)
	}

	if strings.Trim(u.Path, "/") == "" {
		return fmt.Errorf("%s: database name is missing", "DATABASE_URL")
	}

	if env == Production {
		switch sslmode := u.Query().Get("sslmode"); sslmode {
		case "require", "verify-ca", "verify-full":
		default:
			if sslmode == "" {
				sslmode = "(not set)"
			}
			return fmt.Errorf("%s: sslmode must be require, verify-ca, or verify-full when %s=%s, got %s",
				"DATABASE_URL", "APP_ENV", env, sslmode)
		}
	}

	return nil
}

// position validate the duratioin is possitive and not zero
// ===============================================================
func positive(key string, d time.Duration) error {
	if d <= 0 {
		return fmt.Errorf("%s: must be greater than 0, got %v", key, d)
	}
	return nil
}

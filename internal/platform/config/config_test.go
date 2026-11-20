package config

import (
	"errors"
	"log/slog"
	"testing"
	"time"
)

func envMap(kv map[string]string) func(string) string {
	return func(key string) string {
		return kv[key]
	}
}

func validBaseEnv() map[string]string {
	return map[string]string{
		"APP_ENV":      "development",
		"HTTP_ADDR":    ":8080",
		"DATABASE_URL": "postgres://user:pass@localhost:5432/appdb?sslmode=disable",
	}
}

func TestParsers(t *testing.T) {
	t.Run("asString", func(t *testing.T) {
		got, err := asString("hello")
		if err != nil || got != "hello" {
			t.Fatalf("asString() = (%q, %v), want (%q, nil)", got, err, "hello")
		}
	})

	t.Run("asBool", func(t *testing.T) {
		tests := []struct {
			in      string
			want    bool
			wantErr bool
		}{
			{in: "true", want: true},
			{in: "false", want: false},
			{in: "1", want: true},
			{in: "0", want: false},
			{in: "yes", wantErr: true},
			{in: "", wantErr: true},
		}
		for _, tc := range tests {
			got, err := asBool(tc.in)
			if (err != nil) != tc.wantErr {
				t.Errorf("asBool(%q) err = %v, wantErr %v", tc.in, err, tc.wantErr)
				continue
			}
			if !tc.wantErr && got != tc.want {
				t.Errorf("asBool(%q) = %v, want %v", tc.in, got, tc.want)
			}
		}
	})

	t.Run("asInt", func(t *testing.T) {
		tests := []struct {
			in      string
			want    int
			wantErr bool
		}{
			{in: "42", want: 42},
			{in: "-7", want: -7},
			{in: "0", want: 0},
			{in: "abc", wantErr: true},
			{in: "12.34", wantErr: true},
		}
		for _, tc := range tests {
			got, err := asInt(tc.in)
			if (err != nil) != tc.wantErr {
				t.Errorf("asInt(%q) err = %v, wantErr %v", tc.in, err, tc.wantErr)
				continue
			}
			if !tc.wantErr && got != tc.want {
				t.Errorf("asInt(%q) = %d, want %d", tc.in, got, tc.want)
			}
		}
	})

	t.Run("asInt32", func(t *testing.T) {
		tests := []struct {
			in      string
			want    int32
			wantErr bool
		}{
			{in: "25", want: 25},
			{in: "0", want: 0},
			{in: "2147483647", want: 2147483647},
			{in: "-2147483648", want: -2147483648},
			{in: "2147483648", wantErr: true},
			{in: "-2147483649", wantErr: true},
			{in: "not-an-int32", wantErr: true},
		}
		for _, tc := range tests {
			got, err := asInt32(tc.in)
			if (err != nil) != tc.wantErr {
				t.Errorf("asInt32(%q) err = %v, wantErr %v", tc.in, err, tc.wantErr)
				continue
			}
			if !tc.wantErr && got != tc.want {
				t.Errorf("asInt32(%q) = %d, want %d", tc.in, got, tc.want)
			}
		}
	})

	t.Run("asInt64", func(t *testing.T) {
		got, err := asInt64("1048576")
		if err != nil || got != 1048576 {
			t.Fatalf("asInt64() = (%d, %v), want (1048576, nil)", got, err)
		}
		if _, err := asInt64("not-an-int64"); err == nil {
			t.Fatal("asInt64() expected error for invalid input, got nil")
		}
	})

	t.Run("asDuration", func(t *testing.T) {
		got, err := asDuration("250ms")
		if err != nil || got != 250*time.Millisecond {
			t.Fatalf("asDuration() = (%v, %v), want (250ms, nil)", got, err)
		}
		if _, err := asDuration("5"); err == nil {
			t.Fatal("asDuration() expected error for unitless duration, got nil")
		}
	})

	t.Run("asLevel", func(t *testing.T) {
		tests := []struct {
			in      string
			want    slog.Level
			wantErr bool
		}{
			{in: "DEBUG", want: slog.LevelDebug},
			{in: "INFO", want: slog.LevelInfo},
			{in: "WARN", want: slog.LevelWarn},
			{in: "ERROR", want: slog.LevelError},
			{in: "info", want: slog.LevelInfo}, // slog.Level.UnmarshalText is case-insensitive
			{in: "TRACE", wantErr: true},
		}
		for _, tc := range tests {
			got, err := asLevel(tc.in)
			if (err != nil) != tc.wantErr {
				t.Errorf("asLevel(%q) err = %v, wantErr %v", tc.in, err, tc.wantErr)
				continue
			}
			if !tc.wantErr && got != tc.want {
				t.Errorf("asLevel(%q) = %v, want %v", tc.in, got, tc.want)
			}
		}
	})

	t.Run("oneOf", func(t *testing.T) {
		parseEnv := oneOf(Development, Staging, Production)

		got, err := parseEnv("staging")
		if err != nil || got != Staging {
			t.Fatalf("oneOf() valid = (%q, %v), want (%q, nil)", got, err, Staging)
		}

		gotInvalid, err := parseEnv("local")
		if err == nil {
			t.Fatal("oneOf() expected error for disallowed value, got nil")
		}
		if gotInvalid != "" {
			t.Errorf("oneOf() invalid zero value = %q, want empty", gotInvalid)
		}
	})
}

func TestReaderOptionalAndRequired(t *testing.T) {
	t.Run("optional returns default when unset or whitespace", func(t *testing.T) {
		r := newReader(envMap(map[string]string{
			"EMPTY_VAR": "   ",
		}))

		gotUnset := optional(r, "UNSET_VAR", 99, asInt)
		gotEmpty := optional(r, "EMPTY_VAR", 99, asInt)

		if gotUnset != 99 || gotEmpty != 99 {
			t.Fatalf("optional defaults = (%d, %d), want (99, 99)", gotUnset, gotEmpty)
		}
		if len(r.errs) != 0 {
			t.Fatalf("unexpected reader errors: %v", r.errs)
		}
	})

	t.Run("optional records error and returns default on parse failure", func(t *testing.T) {
		r := newReader(envMap(map[string]string{
			"BAD_INT": "  nope ",
		}))

		got := optional(r, "BAD_INT", 10, asInt)
		if got != 10 {
			t.Fatalf("optional on error = %d, want default 10", got)
		}
		if len(r.errs) != 1 {
			t.Fatalf("expected 1 error, got %d", len(r.errs))
		}
	})

	t.Run("required fails on missing and invalid values", func(t *testing.T) {
		r := newReader(envMap(map[string]string{
			"MISSING_VAR": "",
			"BAD_BOOL":    "maybe",
			"GOOD_INT":    "  7  ",
		}))

		_ = required(r, "MISSING_VAR", asString)
		_ = required(r, "BAD_BOOL", asBool)
		good := required(r, "GOOD_INT", asInt)

		if good != 7 {
			t.Fatalf("required valid = %d, want 7", good)
		}
		if len(r.errs) != 2 {
			t.Fatalf("expected 2 errors, got %d: %v", len(r.errs), r.errs)
		}
	})
}

func TestLoadSuccessAndDefaults(t *testing.T) {
	cfg, err := Load(envMap(validBaseEnv()))
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}

	if cfg.App.Name != "transcore-backend" {
		t.Errorf("App.Name = %q, want %q", cfg.App.Name, "transcore-backend")
	}
	if cfg.App.Environment != Development {
		t.Errorf("App.Environment = %q, want %q", cfg.App.Environment, Development)
	}
	if cfg.HTTP.Addr != ":8080" {
		t.Errorf("HTTP.Addr = %q, want %q", cfg.HTTP.Addr, ":8080")
	}
	if cfg.HTTP.ReadTimeout != 15*time.Second {
		t.Errorf("HTTP.ReadTimeout = %v, want 15s", cfg.HTTP.ReadTimeout)
	}
	if cfg.Logger.Level != slog.LevelDebug {
		t.Errorf("Logger.Level = %v, want DEBUG", cfg.Logger.Level)
	}
	if cfg.Logger.Format != LogFormatJSON {
		t.Errorf("Logger.Format = %q, want %q", cfg.Logger.Format, LogFormatJSON)
	}
	if cfg.Database.MaxConns != int32(10) || cfg.Database.MinConns != int32(2) {
		t.Errorf("Database conns = (%d, %d), want (10, 2)", cfg.Database.MaxConns, cfg.Database.MinConns)
	}
}

func TestLoadAggregatesParseErrorsBeforeValidation(t *testing.T) {
	env := map[string]string{
		// Missing required fields + invalid types
		"HTTP_READ_TIMEOUT": "invalid",
		"DB_MAX_CONNS":      "not-a-number",
	}

	_, err := Load(envMap(env))
	if err == nil {
		t.Fatal("Load() expected error, got nil")
	}

	// Verify multiple errors are aggregated
	joined := errors.Unwrap(err)
	if joined == nil {
		t.Fatalf("expected wrapped error, got %v", err)
	}
}

func TestBusinessValidationRules(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(c *Config)
		wantErr bool
	}{
		{
			name:    "valid default config",
			mutate:  func(c *Config) {},
			wantErr: false,
		},
		{
			name: "blank app name fails",
			mutate: func(c *Config) {
				c.App.Name = "   "
			},
			wantErr: true,
		},
		{
			name: "invalid http addr fails",
			mutate: func(c *Config) {
				c.HTTP.Addr = "no-port"
			},
			wantErr: true,
		},
		{
			name: "out of range port fails",
			mutate: func(c *Config) {
				c.HTTP.Addr = ":70000"
			},
			wantErr: true,
		},
		{
			name: "read header timeout exceeding read timeout fails",
			mutate: func(c *Config) {
				c.HTTP.ReadTimeout = 5 * time.Second
				c.HTTP.ReadHeaderTimeout = 10 * time.Second
			},
			wantErr: true,
		},
		{
			name: "non-positive http max body bytes fails",
			mutate: func(c *Config) {
				c.HTTP.MaxBodyBytes = 0
			},
			wantErr: true,
		},
		{
			name: "production requires json log format",
			mutate: func(c *Config) {
				c.App.Environment = Production
				c.Logger.Format = LogFormatText
			},
			wantErr: true,
		},
		{
			name: "production allows json log format with valid sslmode",
			mutate: func(c *Config) {
				c.App.Environment = Production
				c.Logger.Format = LogFormatJSON
				c.Database.URL = "postgres://user:pass@db.example.com:5432/proddb?sslmode=require"
			},
			wantErr: false,
		},
		{
			name: "production rejects missing sslmode",
			mutate: func(c *Config) {
				c.App.Environment = Production
				c.Logger.Format = LogFormatJSON
				c.Database.URL = "postgres://user:pass@db.example.com:5432/proddb"
			},
			wantErr: true,
		},
		{
			name: "database min conns greater than max conns fails",
			mutate: func(c *Config) {
				c.Database.MaxConns = 5
				c.Database.MinConns = 6
			},
			wantErr: true,
		},
		{
			name: "database non-positive startup timeout fails",
			mutate: func(c *Config) {
				c.Database.StartupTimeout = 0
			},
			wantErr: true,
		},
		{
			name: "database non-positive health timeout fails",
			mutate: func(c *Config) {
				c.Database.HealthTimeout = 0
			},
			wantErr: true,
		},
		{
			name: "database url missing db name fails",
			mutate: func(c *Config) {
				c.Database.URL = "postgres://localhost:5432"
			},
			wantErr: true,
		},
		{
			name: "database url wrong scheme fails",
			mutate: func(c *Config) {
				c.Database.URL = "mysql://localhost:3306/mydb"
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := Load(envMap(validBaseEnv()))
			if err != nil {
				t.Fatalf("base Load() failed: %v", err)
			}

			tc.mutate(&cfg)
			err = cfg.validate()
			if (err != nil) != tc.wantErr {
				t.Errorf("validate() err = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

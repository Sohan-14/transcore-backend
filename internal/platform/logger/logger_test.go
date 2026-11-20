package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/sohan-14/transcore-backend/internal/platform/config"
)

func TestNewFormatsAndLevels(t *testing.T) {
	t.Run("json format outputs valid json", func(t *testing.T) {
		var buf bytes.Buffer
		l := New(&buf, config.Logger{
			Level:  slog.LevelInfo,
			Format: config.LogFormatJSON,
		})

		l.Info("hello json", slog.String("component", "auth"))

		var parsed map[string]any
		if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
			t.Fatalf("expected valid JSON output, got error: %v\noutput: %s", err, buf.String())
		}
		if parsed["msg"] != "hello json" {
			t.Errorf("msg = %v, want %q", parsed["msg"], "hello json")
		}
		if parsed["component"] != "auth" {
			t.Errorf("component = %v, want %q", parsed["component"], "auth")
		}
	})

	t.Run("text format outputs key-value pairs", func(t *testing.T) {
		var buf bytes.Buffer
		l := New(&buf, config.Logger{
			Level:  slog.LevelInfo,
			Format: config.LogFormatText,
		})

		l.Info("hello text", slog.String("service", "billing"))

		out := buf.String()
		first := strings.Contains(out, "msg=\"hello text\"")
		second := strings.Contains(out, "msg=\"hello text\"")

		if !first && !second {
			// slog TextHandler formats messages with spaces when needed
			if !strings.Contains(out, "hello text") {
				t.Errorf("expected message in text output, got: %s", out)
			}
		}
		if !strings.Contains(out, "service=billing") {
			t.Errorf("expected service=billing in text output, got: %s", out)
		}
	})

	t.Run("respects minimum log level", func(t *testing.T) {
		var buf bytes.Buffer
		l := New(&buf, config.Logger{
			Level:  slog.LevelWarn,
			Format: config.LogFormatJSON,
		})

		l.Debug("debug should be dropped")
		l.Info("info should be dropped")
		l.Warn("warn should appear")
		l.Error("error should appear")

		out := buf.String()
		if strings.Contains(out, "debug should be dropped") || strings.Contains(out, "info should be dropped") {
			t.Errorf("expected debug/info logs to be filtered out, got: %s", out)
		}
		if !strings.Contains(out, "warn should appear") || !strings.Contains(out, "error should appear") {
			t.Errorf("expected warn/error logs to be present, got: %s", out)
		}
	})

	t.Run("add source includes or omits source info", func(t *testing.T) {
		var withSrcBuf bytes.Buffer
		withSrc := New(&withSrcBuf, config.Logger{
			Level:     slog.LevelInfo,
			Format:    config.LogFormatJSON,
			AddSource: true,
		})
		withSrc.Info("with source")

		if !strings.Contains(withSrcBuf.String(), "logger_test.go") {
			t.Errorf("expected source file in output when AddSource=true, got: %s", withSrcBuf.String())
		}

		var withoutSrcBuf bytes.Buffer
		withoutSrc := New(&withoutSrcBuf, config.Logger{
			Level:     slog.LevelInfo,
			Format:    config.LogFormatJSON,
			AddSource: false,
		})
		withoutSrc.Info("without source")

		if strings.Contains(withoutSrcBuf.String(), "logger_test.go") {
			t.Errorf("did not expect source file in output when AddSource=false, got: %s", withoutSrcBuf.String())
		}
	})

	t.Run("nil writer falls back safely to discard", func(t *testing.T) {
		l := New(nil, config.Logger{
			Level:  slog.LevelInfo,
			Format: config.LogFormatJSON,
		})
		if l == nil {
			t.Fatal("New(nil, ...) returned nil logger")
		}
		l.Info("safe log with nil writer")
	})
}

func TestSecretRedaction(t *testing.T) {
	var buf bytes.Buffer
	l := New(&buf, config.Logger{
		Level:  slog.LevelDebug,
		Format: config.LogFormatJSON,
	})

	l.Info("auth check",
		slog.String("user_id", "usr_123"),
		slog.String("password", "super-secret-pw"),
		slog.String("Authorization", "Bearer tok_xyz"),
		slog.String("API_KEY", "ak_live_999"),
		slog.String("Refresh_Token", "rt_abc"),
		slog.String("Set-Cookie", "sid=12345"),
	)

	var parsed map[string]any
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("failed to unmarshal log JSON: %v", err)
	}

	if parsed["user_id"] != "usr_123" {
		t.Errorf("non-sensitive attr user_id = %v, want %q", parsed["user_id"], "usr_123")
	}

	for _, secretKey := range []string{"password", "Authorization", "API_KEY", "Refresh_Token", "Set-Cookie"} {
		if got := parsed[secretKey]; got != "[REDACTED]" {
			t.Errorf("key %q = %v, want %q", secretKey, got, "[REDACTED]")
		}
	}
}

func TestContextAttrsAndIsolation(t *testing.T) {
	t.Run("merges attributes across chained contexts without mutating siblings", func(t *testing.T) {
		var buf bytes.Buffer
		l := New(&buf, config.Logger{
			Level:  slog.LevelInfo,
			Format: config.LogFormatJSON,
		})

		baseCtx := context.Background()
		// Empty call should be a safe no-op
		emptyCtx := ContextWithAttrs(baseCtx)

		reqCtx := ContextWithAttrs(emptyCtx, slog.String("request_id", "req_1"))
		branchA := ContextWithAttrs(reqCtx, slog.String("step", "A"))
		branchB := ContextWithAttrs(reqCtx, slog.String("step", "B"))

		l.InfoContext(branchA, "handling branch A")
		lineA := lastLine(buf.String())

		l.InfoContext(branchB, "handling branch B")
		lineB := lastLine(buf.String())

		l.InfoContext(reqCtx, "handling base request")
		lineBase := lastLine(buf.String())

		var mapA, mapB, mapBase map[string]any
		_ = json.Unmarshal([]byte(lineA), &mapA)
		_ = json.Unmarshal([]byte(lineB), &mapB)
		_ = json.Unmarshal([]byte(lineBase), &mapBase)

		if mapA["request_id"] != "req_1" || mapA["step"] != "A" {
			t.Errorf("unexpected branchA fields: %v", mapA)
		}
		if mapB["request_id"] != "req_1" || mapB["step"] != "B" {
			t.Errorf("unexpected branchB fields: %v", mapB)
		}
		if mapBase["request_id"] != "req_1" || mapBase["step"] != nil {
			t.Errorf("expected base request to not have branch step, got: %v", mapBase)
		}
	})

	t.Run("preserves context attrs with WithAttrs and WithGroup", func(t *testing.T) {
		var buf bytes.Buffer
		l := New(&buf, config.Logger{
			Level:  slog.LevelInfo,
			Format: config.LogFormatJSON,
		}).With(slog.String("app", "transcore")).WithGroup("http")

		ctx := ContextWithAttrs(context.Background(), slog.String("trace_id", "tr_42"))
		l.InfoContext(ctx, "request completed", slog.Int("status", 200))

		var parsed map[string]any
		if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
			t.Fatalf("failed to unmarshal log JSON: %v", err)
		}
		if parsed["app"] != "transcore" {
			t.Errorf("app = %v, want %q", parsed["app"], "transcore")
		}
		httpGroup, ok := parsed["http"].(map[string]any)
		if !ok {
			t.Fatalf("expected http group object, got %T (%v)", parsed["http"], parsed["http"])
		}
		if httpGroup["trace_id"] != "tr_42" {
			t.Errorf("http.trace_id = %v, want %q", httpGroup["trace_id"], "tr_42")
		}
		if httpGroup["status"] != float64(200) {
			t.Errorf("http.status = %v, want 200", httpGroup["status"])
		}
	})
}

func TestSetupSetsDefaultLogger(t *testing.T) {
	prevDefault := slog.Default()
	t.Cleanup(func() {
		slog.SetDefault(prevDefault)
	})

	var buf bytes.Buffer
	l := Setup(&buf, config.Logger{
		Level:  slog.LevelInfo,
		Format: config.LogFormatJSON,
	})
	if l == nil {
		t.Fatal("Setup returned nil logger")
	}

	slog.Info("default logger check", slog.String("password", "hidden"))
	if !strings.Contains(buf.String(), `"password":"[REDACTED]"`) {
		t.Errorf("expected redacted password in default logger output, got: %s", buf.String())
	}
}

func lastLine(s string) string {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return ""
	}
	parts := strings.Split(trimmed, "\n")
	return parts[len(parts)-1]
}

// Compile-time check that io.Discard satisfies io.Writer in tests.
var _ io.Writer = io.Discard

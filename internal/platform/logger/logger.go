package logger

import (
	"io"
	"log/slog"
	"strings"

	"github.com/sohan-14/transcore-backend/internal/platform/config"
)

// redactedKeys are attribute names whose values are never written to logs,
var redactedKeys = map[string]struct{}{
	"password":      {},
	"passwd":        {},
	"secret":        {},
	"token":         {},
	"access_token":  {},
	"refresh_token": {},
	"api_key":       {},
	"apikey":        {},
	"authorization": {},
	"cookie":        {},
	"set-cookie":    {},
}

func redact(_ []string, a slog.Attr) slog.Attr {
	if _, ok := redactedKeys[strings.ToLower(a.Key)]; ok {
		return slog.String(a.Key, "[REDACTED]")
	}
	return a
}

// New builds a logger
// ===============================================================
func New(w io.Writer, loggerCfg *config.Logger) *slog.Logger {

	if w == nil {
		w = io.Discard
	}

	handlerOpts := &slog.HandlerOptions{
		AddSource:   loggerCfg.AddSource,
		Level:       loggerCfg.Level,
		ReplaceAttr: redact,
	}

	var handler slog.Handler
	switch loggerCfg.Format {
	case config.LogFormatText:
		handler = slog.NewTextHandler(w, handlerOpts)
	case config.LogFormatJSON:
		handler = slog.NewJSONHandler(w, handlerOpts)
	default:
		handler = slog.NewJSONHandler(w, handlerOpts)
	}

	return slog.New(contextHandler{handler})
}

// Setup builds a logger and installs it as the slog default.
// ===============================================================
func Setup(w io.Writer, loggerCfg *config.Logger) *slog.Logger {
	logger := New(w, loggerCfg)
	slog.SetDefault(logger)
	return logger
}

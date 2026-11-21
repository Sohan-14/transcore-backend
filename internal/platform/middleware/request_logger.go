package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/sohan-14/transcore-backend/internal/platform/requestid"
)

// quietPaths are probed constantly by load balancers and orchestrators; a
// successful probe is logged at debug level so it does not drown real traffic.
var quietPaths = map[string]struct{}{
	"/healthz": {},
	"/readyz":  {},
}

// RequestLogger logs completed HTTP requests with
// status, response size, and duration.
// ===============================================================
func RequestLogger(log *slog.Logger) Middleware {
	if log == nil {
		log = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := newResponseWriter(w)

			defer func() {
				level := slog.LevelInfo
				switch {
				case rw.status >= http.StatusInternalServerError:
					level = slog.LevelError
				case rw.status >= http.StatusBadRequest:
					level = slog.LevelWarn
				default:
					if _, quiet := quietPaths[r.URL.Path]; quiet {
						level = slog.LevelDebug
					}
				}
				log.LogAttrs(r.Context(), level, "http request",
					slog.String("request_id", requestid.FromContext(r.Context())),
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
					slog.String("route", r.Pattern),
					slog.Int("status", rw.status),
					slog.Duration("duration", time.Since(start)),
					slog.Int("bytes", rw.bytes),
					slog.String("remote_addr", r.RemoteAddr),
					slog.String("user_agent", r.UserAgent()),
				)
			}()

			next.ServeHTTP(rw, r)

		})
	}
}

// responseWriter records the status code and the number of bytes written.
type responseWriter struct {
	http.ResponseWriter
	status      int
	bytes       int
	wroteHeader bool
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{ResponseWriter: w, status: http.StatusOK}
}

// WriteHeader write the header with status code
func (rw *responseWriter) WriteHeader(code int) {
	if rw.wroteHeader {
		return
	}
	// Informational responses (1xx) are not final, except 101 Switching Protocols.
	if code >= 100 && code < 200 && code != http.StatusSwitchingProtocols {
		rw.ResponseWriter.WriteHeader(code)
		return
	}
	rw.status = code
	rw.wroteHeader = true
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(p []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
	n, err := rw.ResponseWriter.Write(p)
	rw.bytes += n
	return n, err
}

// Unwrap lets http.ResponseController reach the underlying writer's
// Flush/Hijack through this wrapper (needed for SSE, websockets).
func (rw *responseWriter) Unwrap() http.ResponseWriter {
	return rw.ResponseWriter
}

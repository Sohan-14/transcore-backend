package middleware

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/sohan-14/transcore-backend/internal/apperr"
	"github.com/sohan-14/transcore-backend/internal/platform/httpx"
	"github.com/sohan-14/transcore-backend/internal/platform/requestid"
)

// PanicRecover turns a panic in any downstream handler into a logged error and
// a 500 problem+json response, so one bad request cannot take the process down.
// ===============================================================
func PanicRecover(log *slog.Logger) Middleware {
	if log == nil {
		log = slog.Default()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					// net/http's own signal for "abort this response quietly".
					if err, ok := rec.(error); ok && errors.Is(err, http.ErrAbortHandler) {
						panic(rec)
					}

					reqID := requestid.FromContext(r.Context())

					log.ErrorContext(r.Context(), "panic recovered",
						"panic", rec,
						"method", r.Method,
						"path", r.URL.Path,
						"request_id", reqID,
						"stack", string(debug.Stack()),
					)

					httpx.WriteError(
						w, r,
						apperr.Internal(fmt.Errorf("panic: %v\n%s", rec, debug.Stack())),
					)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

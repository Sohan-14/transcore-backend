package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/sohan-14/transcore-backend/internal/apperr"
	"github.com/sohan-14/transcore-backend/internal/platform/httpx"
)

// PanicRecover turns a panic in any downstream handler into a logged error and
// a 500 problem+json response, so one bad request cannot take the process down.
// ===============================================================
func PanicRecover() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					// net/http's own signal for "abort this response quietly".
					if err, ok := rec.(error); ok && errors.Is(err, http.ErrAbortHandler) {
						panic(rec)
					}

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

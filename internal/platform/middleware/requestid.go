package middleware

import (
	"net/http"

	"github.com/sohan-14/transcore-backend/internal/platform/requestid"
)

// RequestID extracts a valid incoming request ID or generates a new one,
// stores it in the request context, and echoes it on the response header.
// ===============================================================
func RequestID() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Header.Get(requestid.Header)
			if !requestid.Valid(id) {
				id = requestid.Generate()
			}

			w.Header().Set(requestid.Header, id)
			ctx := requestid.NewContext(r.Context(), id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

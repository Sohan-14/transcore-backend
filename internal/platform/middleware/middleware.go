package middleware

import (
	"net/http"
	"slices"
)

// Middleware wraps an http.Handler to provide cross-cutting or scoped behavior.
type Middleware func(http.Handler) http.Handler

// Chain applies middlewares in left-to-right order:
// ===============================================================
func Chain(handler http.Handler, middlewares ...Middleware) http.Handler {
	for _, middleware := range slices.Backward(middlewares) {
		if middleware != nil {
			handler = middleware(handler)
		}
	}
	return handler
}

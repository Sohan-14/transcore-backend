package router

import (
	"net/http"
	"strings"

	"github.com/sohan-14/transcore-backend/internal/platform/middleware"
)

// Group registers routes under a prefix, sharing group middleware.
// ===============================================================
type Group struct {
	mux    *http.ServeMux
	prefix string
	mw     []middleware.Middleware
}

// New creates a group. prefix is owned by the module, e.g. "/api/v1/".
// Group middleware runs before per-route middleware.
// ===============================================================
func New(mux *http.ServeMux, prefix string, mw ...middleware.Middleware) *Group {
	return &Group{
		mux:    mux,
		prefix: strings.TrimSuffix(prefix, "/"),
		mw:     mw,
	}
}

// Handle registers "[METHOD ]/path" relative to the prefix.
// "/" means the group root exactly (no trailing-slash redirect).
// ===============================================================
func (g *Group) Handle(pattern string, h http.HandlerFunc, mw ...middleware.Middleware) {
	method, path, hasMethod := strings.Cut(pattern, " ")
	if !hasMethod {
		path = pattern
	}

	fullPath := g.prefix + path
	if path == "/" {
		fullPath = g.prefix
	}
	full := fullPath
	if hasMethod {
		full = method + " " + fullPath
	}

	// Copy: never mutate the group's slice.
	chain := make([]middleware.Middleware, 0, len(g.mw)+len(mw))
	chain = append(chain, g.mw...)
	chain = append(chain, mw...)

	g.mux.Handle(full, middleware.Chain(h, chain...))
}

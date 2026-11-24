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
// Panics on invalid input: wiring errors must surface
// at startup, not per-request.
// ===============================================================
func New(mux *http.ServeMux, prefix string, mw ...middleware.Middleware) *Group {
	if mux == nil {
		panic("router: nil mux")
	}

	if prefix == "" || !strings.HasPrefix(prefix, "/") {
		panic("router: prefix must be non-empty and start with '/', got " + prefix)
	}

	owned := make([]middleware.Middleware, len(mw))
	copy(owned, mw)

	return &Group{
		mux:    mux,
		prefix: strings.TrimSuffix(prefix, "/"),
		mw:     owned,
	}
}

// Handle registers "[METHOD ]/path" relative to the prefix.
// "/" means the group root exactly (no trailing-slash redirect).
// ===============================================================
func (g *Group) Handle(pattern string, h http.HandlerFunc, mw ...middleware.Middleware) {
	if h == nil {
		panic("router: nil handler for pattern " + pattern)
	}

	method, path, hasMethod := strings.Cut(pattern, " ")
	if !hasMethod {
		path = pattern
	}

	if !strings.HasPrefix(path, "/") {
		panic("router: path must start with '/', got " + pattern)
	}

	fullPath := g.prefix + path
	if path == "/" {
		fullPath = g.prefix
	}

	full := fullPath
	if hasMethod {
		full = method + " " + fullPath
	}

	chain := make([]middleware.Middleware, 0, len(g.mw)+len(mw))
	chain = append(chain, g.mw...)
	chain = append(chain, mw...)

	g.mux.Handle(full, middleware.Chain(h, chain...))

}

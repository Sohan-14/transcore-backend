package requestid

import (
	"context"
	"crypto/rand"
	"encoding/hex"
)

// Header is the HTTP header used to receive and return the request ID.
const Header = "X-Request-ID"

const maxLen = 64

type ctxKey struct{}

// NewContext returns a copy of ctx carrying id.
// ===============================================================
func NewContext(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKey{}, id)
}

// FromContext returns the request ID stored in ctx, or "" if none.
// ===============================================================
func FromContext(ctx context.Context) string {
	id, _ := ctx.Value(ctxKey{}).(string)
	return id
}

// Generate returns a new random request ID (32 hex characters).
// ===============================================================
func Generate() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// Valid reports whether id is safe to trust from a client: 1-64 characters of
// [A-Za-z0-9._-]. Anything else could be used for log injection or to blow up
// log-index cardinality.
// ===============================================================
func Valid(id string) bool {
	if id == "" || len(id) > maxLen {
		return false
	}
	for i := 0; i < len(id); i++ {
		c := id[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9',
			c == '.', c == '_', c == '-':
		default:
			return false
		}
	}
	return true
}

package health

import (
	"context"
	"net/http"
)

// Checker is satisfied by platform/database.Database.Health.
// ===============================================================
type Checker interface {
	Health(ctx context.Context) error
}

// Liveness returns 200 OK. The app is alive.
// ===============================================================
func Liveness() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}
}

// Readiness returns 200 if the DB is reachable, 503 otherwise.
// ===============================================================
func Readiness(checker Checker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := checker.Health(r.Context()); err != nil {
			http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}
}

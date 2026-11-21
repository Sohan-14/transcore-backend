package httpx

import (
	"encoding/json"
	"net/http"
)

// JSON writes v as the response body with the given status.
// For 204 use NoContent instead.
// ===============================================================
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

// NoContent sends 204 with no body.
// ===============================================================
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

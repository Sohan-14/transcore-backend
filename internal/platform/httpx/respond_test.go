package httpx

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestJSON(t *testing.T) {
	w := httptest.NewRecorder()
	JSON(w, http.StatusCreated, map[string]any{"id": "txn_1"})

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Fatalf("content-type = %q", ct)
	}
	var got map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil || got["id"] != "txn_1" {
		t.Fatalf("body = %q, err = %v", w.Body.String(), err)
	}
}

func TestJSONNilBody(t *testing.T) {
	w := httptest.NewRecorder()
	JSON(w, http.StatusAccepted, nil)
	if w.Code != http.StatusAccepted || w.Body.Len() != 0 {
		t.Fatalf("status = %d, body = %q", w.Code, w.Body.String())
	}
}

func TestNoContent(t *testing.T) {
	w := httptest.NewRecorder()
	NoContent(w)
	if w.Code != http.StatusNoContent || w.Body.Len() != 0 {
		t.Fatalf("status = %d, body = %q", w.Code, w.Body.String())
	}
}

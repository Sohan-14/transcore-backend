package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sohan-14/transcore-backend/internal/apperr"
	"github.com/sohan-14/transcore-backend/internal/platform/requestid"
)

func doWriteError(t *testing.T, err error) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(http.MethodGet, "/x", nil)
	r = r.WithContext(requestid.NewContext(r.Context(), "req-123"))
	w := httptest.NewRecorder()
	WriteError(w, r, err)
	return w
}

func envelopeOf(t *testing.T, w *httptest.ResponseRecorder) errorBody {
	t.Helper()
	var env errorEnvelope
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("bad envelope: %v", err)
	}
	return env.Error
}

func silenceLogs(t *testing.T) {
	t.Helper()
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })
}

func TestWriteErrorKindStatus(t *testing.T) {
	silenceLogs(t)
	cases := []struct {
		kind   apperr.Kind
		status int
	}{
		{apperr.KindInvalid, 400},
		{apperr.KindUnauthenticated, 401},
		{apperr.KindForbidden, 403},
		{apperr.KindNotFound, 404},
		{apperr.KindConflict, 409},
		{apperr.KindTooLarge, 413},
		{apperr.KindUnsupportedMedia, 415},
		{apperr.KindRateLimited, 429},
		{apperr.KindUnavailable, 503},
		{apperr.KindInternal, 500},
		{apperr.Kind("bogus"), 500},
	}
	for _, tc := range cases {
		t.Run(string(tc.kind), func(t *testing.T) {
			w := doWriteError(t, apperr.New(tc.kind, "test_code", "safe message"))
			if w.Code != tc.status {
				t.Fatalf("status = %d, want %d", w.Code, tc.status)
			}
			if ct := w.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
				t.Fatalf("content-type = %q", ct)
			}
			body := envelopeOf(t, w)
			if body.Code != "test_code" || body.Message != "safe message" || body.RequestID != "req-123" {
				t.Fatalf("body = %+v", body)
			}
		})
	}
}

func TestWriteErrorUnknownNeverLeaks(t *testing.T) {
	silenceLogs(t)
	w := doWriteError(t, errors.New("secret dsn password"))
	if w.Code != 500 {
		t.Fatalf("status = %d, want 500", w.Code)
	}
	if strings.Contains(w.Body.String(), "secret") {
		t.Fatal("cause leaked to client")
	}
	if body := envelopeOf(t, w); body.Code != "internal_error" {
		t.Fatalf("code = %q", body.Code)
	}
}

func TestWriteErrorMaxBytesFallback(t *testing.T) {
	w := doWriteError(t, &http.MaxBytesError{Limit: 10})
	if w.Code != 413 {
		t.Fatalf("status = %d, want 413", w.Code)
	}
	if body := envelopeOf(t, w); body.Code != "request_too_large" {
		t.Fatalf("code = %q", body.Code)
	}
}

func TestWriteErrorFields(t *testing.T) {
	e := apperr.New(apperr.KindInvalid, "validation_failed", "invalid fields").
		WithFields(apperr.Field{Name: "amount", Issue: "must be positive"})
	body := envelopeOf(t, doWriteError(t, e))
	if len(body.Fields) != 1 || body.Fields[0].Field != "amount" || body.Fields[0].Issue != "must be positive" {
		t.Fatalf("fields = %+v", body.Fields)
	}

	w := doWriteError(t, apperr.New(apperr.KindNotFound, "nf", "not found"))
	if strings.Contains(w.Body.String(), `"fields"`) {
		t.Fatal("empty fields must be omitted")
	}
}

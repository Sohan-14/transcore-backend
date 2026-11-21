package httpx

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sohan-14/transcore-backend/internal/apperr"
)

func jsonReq(body string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	return r
}

func TestDecodeJSONSuccess(t *testing.T) {
	bodies := []string{
		`{"amount":5}`,
		"{\"amount\":5}\n  ",             // trailing whitespace is fine
		`{"amount":5,"extra":"ignored"}`, // unknown fields are ignored
	}
	for _, body := range bodies {
		var in struct {
			Amount int `json:"amount"`
		}
		if err := DecodeJSON(jsonReq(body), &in); err != nil {
			t.Fatalf("body %q: %v", body, err)
		}
		if in.Amount != 5 {
			t.Fatalf("body %q: amount = %d", body, in.Amount)
		}
	}
}

func TestDecodeJSONFailures(t *testing.T) {
	cases := []struct {
		name        string
		req         func() *http.Request
		kind        apperr.Kind
		code        string
		msgContains string
	}{
		{"wrong content type", func() *http.Request {
			r := jsonReq(`{}`)
			r.Header.Set("Content-Type", "text/plain")
			return r
		}, apperr.KindUnsupportedMedia, "unsupported_media_type", ""},
		{"missing content type", func() *http.Request {
			r := jsonReq(`{}`)
			r.Header.Del("Content-Type")
			return r
		}, apperr.KindUnsupportedMedia, "unsupported_media_type", ""},
		{"empty body", func() *http.Request { return jsonReq("") },
			apperr.KindInvalid, "invalid_json", "required"},
		{"malformed", func() *http.Request { return jsonReq(`{`) },
			apperr.KindInvalid, "invalid_json", ""},
		{"type mismatch", func() *http.Request { return jsonReq(`{"amount":"x"}`) },
			apperr.KindInvalid, "invalid_json", "amount"},
		{"trailing data", func() *http.Request { return jsonReq(`{"amount":5}{"amount":6}`) },
			apperr.KindInvalid, "invalid_json", "single"},
		{"too large", func() *http.Request {
			r := jsonReq(`{"amount":123456}`)
			r.Body = http.MaxBytesReader(httptest.NewRecorder(), r.Body, 5)
			return r
		}, apperr.KindTooLarge, "request_too_large", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var in struct {
				Amount int `json:"amount"`
			}
			err := DecodeJSON(tc.req(), &in)
			e, ok := apperr.As(err)
			if !ok {
				t.Fatalf("err = %v, want *apperr.Error", err)
			}
			if e.Kind != tc.kind || e.Code != tc.code {
				t.Fatalf("kind/code = %s/%s, want %s/%s", e.Kind, e.Code, tc.kind, tc.code)
			}
			if tc.msgContains != "" && !strings.Contains(e.Message, tc.msgContains) {
				t.Fatalf("message = %q, want contains %q", e.Message, tc.msgContains)
			}
		})
	}
}

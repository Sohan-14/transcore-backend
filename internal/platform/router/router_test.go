package router

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sohan-14/transcore-backend/internal/platform/middleware"
)

func recordMW(name string, log *[]string) middleware.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			*log = append(*log, name+":before")
			next.ServeHTTP(w, r)
			*log = append(*log, name+":after")
		})
	}
}

func TestGroupRoutingAndMiddlewareOrder(t *testing.T) {
	mux := http.NewServeMux()
	var trace []string

	api := New(mux, "/api/v1/", recordMW("group1", &trace), recordMW("group2", &trace))

	api.Handle("GET /", func(w http.ResponseWriter, r *http.Request) {
		trace = append(trace, "root-handler")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("root"))
	})

	api.Handle("GET /orders/{id}", func(w http.ResponseWriter, r *http.Request) {
		trace = append(trace, "order-handler:"+r.PathValue("id"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("order-" + r.PathValue("id")))
	}, recordMW("route", &trace))

	// 1. Test group root ("GET /api/v1")
	trace = nil
	reqRoot := httptest.NewRequest(http.MethodGet, "/api/v1", nil)
	recRoot := httptest.NewRecorder()
	mux.ServeHTTP(recRoot, reqRoot)

	if recRoot.Code != http.StatusOK {
		t.Fatalf("root status = %d, want %d", recRoot.Code, http.StatusOK)
	}
	if got := strings.TrimSpace(recRoot.Body.String()); got != "root" {
		t.Fatalf("root body = %q, want %q", got, "root")
	}

	// 2. Test prefixed route + path param + group->route middleware ordering
	trace = nil
	reqOrder := httptest.NewRequest(http.MethodGet, "/api/v1/orders/42", nil)
	recOrder := httptest.NewRecorder()
	mux.ServeHTTP(recOrder, reqOrder)

	if recOrder.Code != http.StatusOK {
		t.Fatalf("order status = %d, want %d", recOrder.Code, http.StatusOK)
	}
	body, _ := io.ReadAll(recOrder.Body)
	if string(body) != "order-42" {
		t.Fatalf("order body = %q, want %q", string(body), "order-42")
	}

	wantTrace := []string{
		"group1:before",
		"group2:before",
		"route:before",
		"order-handler:42",
		"route:after",
		"group2:after",
		"group1:after",
	}
	if len(trace) != len(wantTrace) {
		t.Fatalf("trace len = %d (%v), want %d (%v)", len(trace), trace, len(wantTrace), wantTrace)
	}
	for i := range wantTrace {
		if trace[i] != wantTrace[i] {
			t.Fatalf("trace[%d] = %q, want %q (full: %v)", i, trace[i], wantTrace[i], trace)
		}
	}

	// 3. Ensure route middleware did not leak into earlier route's slice
	if strings.Contains(strings.Join(trace, ","), "route") && recRoot.Code == http.StatusOK {
		// already reset trace before order call; also verify another route has no route MW
		trace = nil
		api.Handle("POST /ping", func(w http.ResponseWriter, r *http.Request) {
			trace = append(trace, "ping")
			w.WriteHeader(http.StatusNoContent)
		})
		reqPing := httptest.NewRequest(http.MethodPost, "/api/v1/ping", nil)
		recPing := httptest.NewRecorder()
		mux.ServeHTTP(recPing, reqPing)

		for _, entry := range trace {
			if strings.HasPrefix(entry, "route:") {
				t.Fatalf("route middleware leaked into subsequent route: %v", trace)
			}
		}
	}
}

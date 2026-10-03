package httpx_test

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/httpx"
)

func handler(t *testing.T, h http.HandlerFunc) http.Handler {
	t.Helper()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	if h == nil {
		h = func(http.ResponseWriter, *http.Request) {}
	}
	mux := http.NewServeMux()
	mux.Handle("POST /echo", h)
	mux.Handle("GET /panic", http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("boom") }))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		httpx.Error(w, r, http.StatusNotFound, httpx.CodeNotFound, "no such endpoint")
	})
	return httpx.Chain(mux, httpx.Recover(log), httpx.RequestID(log), httpx.AccessLog(),
		httpx.CORS([]string{"http://ok.test"}), httpx.BodyLimit(64))
}

func do(h http.Handler, req *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestNullAndRefuse(t *testing.T) {
	h := handler(t, func(w http.ResponseWriter, r *http.Request) { httpx.Refuse(w, r, "wrong_otp") })
	rec := do(h, httptest.NewRequest("POST", "/echo", strings.NewReader("{}")))
	if rec.Code != 200 || strings.TrimSpace(rec.Body.String()) != "null" {
		t.Fatalf("got %d %q", rec.Code, rec.Body)
	}
	if rec.Header().Get("X-Waraqah-Error") != "wrong_otp" {
		t.Errorf("missing refusal header")
	}
}

func TestRequestIDEchoed(t *testing.T) {
	h := handler(t, func(w http.ResponseWriter, r *http.Request) { httpx.JSON(w, map[string]int{"a": 1}) })
	req := httptest.NewRequest("POST", "/echo", strings.NewReader("{}"))
	req.Header.Set("X-Request-ID", "abc-123")
	if got := do(h, req).Header().Get("X-Request-ID"); got != "abc-123" {
		t.Errorf("got %q", got)
	}
	if got := do(h, httptest.NewRequest("POST", "/echo", strings.NewReader("{}"))).Header().Get("X-Request-ID"); got == "" {
		t.Error("no request id generated")
	}
}

func TestUnknownPathAnswersErrorShape(t *testing.T) {
	rec := do(handler(t, nil), httptest.NewRequest("GET", "/v1/nope", nil))
	if rec.Code != 404 || !strings.Contains(rec.Body.String(), `"code":"not_found"`) || !strings.Contains(rec.Body.String(), `"requestId":"req_`) {
		t.Fatalf("got %d %s", rec.Code, rec.Body)
	}
}

func TestPanicBecomes500(t *testing.T) {
	rec := do(handler(t, nil), httptest.NewRequest("GET", "/panic", nil))
	if rec.Code != 500 || !strings.Contains(rec.Body.String(), `"code":"internal"`) || strings.Contains(rec.Body.String(), "boom") {
		t.Fatalf("got %d %s", rec.Code, rec.Body)
	}
}

func TestBodyTooLarge(t *testing.T) {
	h := handler(t, func(w http.ResponseWriter, r *http.Request) {
		var v map[string]any
		if httpx.Decode(w, r, &v) {
			httpx.JSON(w, v)
		}
	})
	big := `{"x":"` + strings.Repeat("a", 200) + `"}`
	if rec := do(h, httptest.NewRequest("POST", "/echo", strings.NewReader(big))); rec.Code != 413 {
		t.Errorf("content-length path: got %d", rec.Code)
	}
	req := httptest.NewRequest("POST", "/echo", io.NopCloser(bytes.NewReader([]byte(big))))
	req.ContentLength = -1
	if rec := do(h, req); rec.Code != 413 {
		t.Errorf("streaming path: got %d", rec.Code)
	}
}

func TestInvalidJSON(t *testing.T) {
	h := handler(t, func(w http.ResponseWriter, r *http.Request) {
		var v map[string]any
		if httpx.Decode(w, r, &v) {
			httpx.JSON(w, v)
		}
	})
	if rec := do(h, httptest.NewRequest("POST", "/echo", strings.NewReader("{nope"))); rec.Code != 400 {
		t.Errorf("got %d", rec.Code)
	}
}

func TestCORSPreflight(t *testing.T) {
	h := handler(t, func(w http.ResponseWriter, r *http.Request) {})
	req := httptest.NewRequest("OPTIONS", "/echo", nil)
	req.Header.Set("Origin", "http://ok.test")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rec := do(h, req)
	if rec.Code != 204 || rec.Header().Get("Access-Control-Allow-Origin") != "http://ok.test" {
		t.Fatalf("got %d %v", rec.Code, rec.Header())
	}
	if !strings.Contains(rec.Header().Get("Access-Control-Allow-Headers"), "Authorization") ||
		!strings.Contains(rec.Header().Get("Access-Control-Expose-Headers"), "X-Waraqah-Error") {
		t.Errorf("headers: %v", rec.Header())
	}
	req.Header.Set("Origin", "http://evil.test")
	if rec := do(h, req); rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("evil origin allowed")
	}
}

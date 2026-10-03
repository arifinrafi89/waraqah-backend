package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestBucketRefills(t *testing.T) {
	now := time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)
	l := New(3, time.Minute).WithClock(func() time.Time { return now })
	for i := 0; i < 3; i++ {
		if !l.Allow("a") {
			t.Fatalf("request %d refused inside the burst", i+1)
		}
	}
	if l.Allow("a") {
		t.Fatal("fourth request allowed")
	}
	if !l.Allow("b") {
		t.Fatal("another key must have its own bucket")
	}
	now = now.Add(21 * time.Second) // 3 per minute = one token every 20 s
	if !l.Allow("a") {
		t.Fatal("token should have refilled")
	}
	if l.Allow("a") {
		t.Fatal("only one token refilled")
	}
}

func TestZeroLimitAllowsAll(t *testing.T) {
	l := New(0, time.Minute)
	for i := 0; i < 100; i++ {
		if !l.Allow("x") {
			t.Fatal("limit 0 must not limit")
		}
	}
}

func TestMiddlewareAnswers429InErrorShape(t *testing.T) {
	l := New(1, time.Hour)
	h := l.ByIP()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))
	req := func() *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/x", nil)
		r.RemoteAddr = "10.0.0.1:5555"
		h.ServeHTTP(rec, r)
		return rec
	}
	if req().Code != 200 {
		t.Fatal("first request refused")
	}
	rec := req()
	if rec.Code != 429 || !strings.Contains(rec.Body.String(), `"code":"rate_limited"`) {
		t.Errorf("got %d %s", rec.Code, rec.Body)
	}
}

func TestIPKeyUsesLastForwardedEntry(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("X-Forwarded-For", "6.6.6.6, 203.0.113.9")
	if got := IPKey(r); got != "ip:203.0.113.9" {
		t.Errorf("got %s (a client-supplied entry must not choose the key)", got)
	}
}

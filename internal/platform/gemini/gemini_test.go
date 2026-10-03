package gemini

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewWithoutKeyIsNil(t *testing.T) {
	if New("", "m", time.Second) != nil {
		t.Error("expected nil client without a key")
	}
}

func TestWordParsesReplyAndKeepsKeyOutOfURL(t *testing.T) {
	var gotKey, gotURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey, gotURL = r.Header.Get("x-goog-api-key"), r.URL.String()
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":" Here are "},{"text":"two books. "}]}}]}`))
	}))
	defer srv.Close()
	c := &REST{APIKey: "secret-key", Model: "flash", BaseURL: srv.URL, HTTP: srv.Client()}
	got, err := c.Word(context.Background(), "hi")
	if err != nil || got != "Here are two books." {
		t.Fatalf("%q %v", got, err)
	}
	if gotKey != "secret-key" || strings.Contains(gotURL, "secret-key") || !strings.HasSuffix(gotURL, "/models/flash:generateContent") {
		t.Errorf("key=%q url=%q", gotKey, gotURL)
	}
}

func TestWordTimeoutAndErrors(t *testing.T) {
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
	}))
	defer slow.Close()
	c := &REST{APIKey: "k", Model: "m", BaseURL: slow.URL, HTTP: slow.Client(), Timeout: 50 * time.Millisecond}
	if _, err := c.Word(context.Background(), "hi"); err == nil {
		t.Error("slow answer must time out")
	}
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(429) }))
	defer bad.Close()
	c = &REST{APIKey: "k", Model: "m", BaseURL: bad.URL, HTTP: bad.Client()}
	if _, err := c.Word(context.Background(), "hi"); err == nil {
		t.Error("429 must be an error")
	}
	empty := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(`{}`)) }))
	defer empty.Close()
	c = &REST{APIKey: "k", Model: "m", BaseURL: empty.URL, HTTP: empty.Client()}
	if _, err := c.Word(context.Background(), "hi"); err != ErrEmptyAnswer {
		t.Errorf("got %v", err)
	}
}

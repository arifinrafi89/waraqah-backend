package email

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFactoryRefusesLogInProduction(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	if _, err := New("log", "", "", true, log); err == nil {
		t.Error("log provider allowed in production")
	}
	if _, err := New("log", "", "", false, log); err != nil {
		t.Error(err)
	}
	if _, err := New("smoke-signals", "", "", false, log); err == nil {
		t.Error("unknown provider accepted")
	}
}

func TestResendAndBrevoRequests(t *testing.T) {
	var path, auth, brevoKey string
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path, auth, brevoKey = r.URL.Path, r.Header.Get("Authorization"), r.Header.Get("api-key")
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.WriteHeader(200)
	}))
	defer srv.Close()
	ctx := context.Background()

	r := &Resend{APIKey: "rk", From: "Waraqah <no-reply@x.test>", URL: srv.URL + "/emails", Client: srv.Client()}
	if err := r.SendOTP(ctx, "a@x.test", "123456", "signup"); err != nil {
		t.Fatal(err)
	}
	if auth != "Bearer rk" || !strings.Contains(body["text"].(string), "123456") || body["from"] != "Waraqah <no-reply@x.test>" {
		t.Errorf("resend: %v %v", auth, body)
	}

	b := &Brevo{APIKey: "bk", From: "Waraqah <no-reply@x.test>", URL: srv.URL + "/v3", Client: srv.Client()}
	if err := b.SendOTP(ctx, "a@x.test", "654321", "reset"); err != nil {
		t.Fatal(err)
	}
	sender := body["sender"].(map[string]any)
	if brevoKey != "bk" || sender["email"] != "no-reply@x.test" || sender["name"] != "Waraqah" || path != "/v3" {
		t.Errorf("brevo: %v %v", brevoKey, body)
	}
	if !strings.Contains(body["subject"].(string), "reset") {
		t.Errorf("reset subject: %v", body["subject"])
	}
}

func TestProviderErrorIsReturned(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(401) }))
	defer srv.Close()
	r := &Resend{APIKey: "bad", URL: srv.URL, Client: srv.Client()}
	if err := r.SendOTP(context.Background(), "a@x.test", "1", "signup"); err == nil {
		t.Error("401 from the provider must be an error")
	}
}

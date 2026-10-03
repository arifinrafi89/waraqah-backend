package app_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/arifinrafi89/waraqah-backend/internal/app"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/config"
)

func newDeps(t *testing.T) *app.Deps {
	t.Helper()
	cfg, err := config.LoadFrom(func(string) (string, bool) { return "", false })
	if err != nil {
		t.Fatal(err)
	}
	d, err := app.NewDeps(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)), nil)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func get(h http.Handler, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
	return rec
}

func TestHealthEndpoints(t *testing.T) {
	h := app.Routes(newDeps(t))
	if rec := get(h, "/healthz"); rec.Code != 200 || !strings.Contains(rec.Body.String(), `"ok":true`) {
		t.Errorf("healthz: %d %s", rec.Code, rec.Body)
	}
	if rec := get(h, "/version"); rec.Code != 200 || !strings.Contains(rec.Body.String(), `"contract":"v1"`) {
		t.Errorf("version: %d %s", rec.Code, rec.Body)
	}
	if rec := get(h, "/v1/x"); rec.Code != 404 || !strings.Contains(rec.Body.String(), `"code":"not_found"`) {
		t.Errorf("404: %d %s", rec.Code, rec.Body)
	}
}

func TestReadyzReportsDatabaseDown(t *testing.T) {
	d := newDeps(t)
	d.Ready = func(context.Context) error { return errors.New("down") }
	rec := get(app.Routes(d), "/readyz")
	if rec.Code != 503 || strings.Contains(rec.Body.String(), "down") {
		t.Errorf("got %d %s", rec.Code, rec.Body)
	}
}

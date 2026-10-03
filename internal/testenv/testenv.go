// Package testenv boots the whole app on a seeded test database for handler tests. Everything a
// test writes is rolled back when the test ends. Only tests import it.
package testenv

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/arifinrafi89/waraqah-backend/internal/app"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/auth"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/clock"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/config"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/dbtest"
	"github.com/arifinrafi89/waraqah-backend/internal/seed"
)

// Accounts of the seeded demo users by short name.
var Accounts = map[string]struct {
	ID   string
	Role auth.Role
}{
	"reader":    {"u_reader", auth.RoleReader},
	"moderator": {"u_moderator", auth.RoleModerator},
	"catalog":   {"u_catalog", auth.RoleCatalogManager},
	"support":   {"u_support", auth.RoleSupport},
	"admin":     {"u_admin", auth.RoleSuperAdmin},
}

// Env is a running app with seeded data.
type Env struct {
	T    *testing.T
	Deps *app.Deps
	H    http.Handler
}

// New seeds the database (once per test, rolled back at the end) and builds the app.
func New(t *testing.T) *Env {
	t.Helper()
	d := dbtest.New(t)
	cfg, err := config.LoadFrom(os.LookupEnv)
	if err != nil {
		t.Fatal(err)
	}
	cfg.BcryptCost, cfg.SeedDemoPassword = 4, "Waraqah#Demo1"
	cfg.AppEnv, cfg.OTPDevCode = "development", "123456"
	cfg.AuthRatePerMin = 100000
	cfg.DemoBotDelay = 0       // the demo readers answer at once, so tests do not wait
	cfg.CourierPickupDelay = 0 // the courier collects a Sell Back at once
	cfg.GeminiAPIKey = ""      // rule-based assistant replies only: no network in tests
	loc, err := clock.Location(cfg.AppTimezone)
	if err != nil {
		t.Fatal(err)
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	if err := seed.All(context.Background(), d, seed.Options{Dir: seedDir(), DemoPassword: cfg.SeedDemoPassword, BcryptCost: cfg.BcryptCost, Loc: loc, Log: log}); err != nil {
		t.Fatal(err)
	}
	deps, err := app.NewDeps(cfg, log, d)
	if err != nil {
		t.Fatal(err)
	}
	return &Env{T: t, Deps: deps, H: app.Routes(deps)}
}

// seedDir finds the repo seed folder from any package directory.
func seedDir() string {
	for _, p := range []string{"seed", "../seed", "../../seed", "../../../seed", "../../../../seed"} {
		if _, err := os.Stat(p + "/users.json"); err == nil {
			return p
		}
	}
	return "seed"
}

// Rebuild builds the handler again from the (possibly changed) Deps, for tests that swap a
// service (a fake alerts sweeper) before sending requests.
func (e *Env) Rebuild() http.Handler { return app.Routes(e.Deps) }

// Token signs an access token for a demo account.
func (e *Env) Token(as string) string {
	a, ok := Accounts[as]
	if !ok {
		e.T.Fatalf("unknown account %q", as)
	}
	tok, _, err := e.Deps.JWT.Sign(a.ID, a.Role)
	if err != nil {
		e.T.Fatal(err)
	}
	return tok
}

// Result is one answer.
type Result struct {
	Status  int
	Refusal string // X-Waraqah-Error
	Body    any    // decoded JSON (nil for null)
	Raw     []byte
}

// Call sends a request as a demo account ("" for a guest). A query string may be part of path.
func (e *Env) Call(as, method, path string, body any) Result {
	e.T.Helper()
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, "/v1"+path, rd)
	if as != "" {
		req.Header.Set("Authorization", "Bearer "+e.Token(as))
	}
	rec := httptest.NewRecorder()
	e.H.ServeHTTP(rec, req)
	res := Result{Status: rec.Code, Refusal: rec.Header().Get("X-Waraqah-Error"), Raw: rec.Body.Bytes()}
	_ = json.Unmarshal(res.Raw, &res.Body)
	return res
}

// List returns the body as a list of objects (failing the test when it is not one).
func (r Result) List(t *testing.T) []map[string]any {
	t.Helper()
	l, ok := r.Body.([]any)
	if !ok {
		t.Fatalf("not a list: %s", r.Raw)
	}
	out := make([]map[string]any, 0, len(l))
	for _, v := range l {
		m, _ := v.(map[string]any)
		out = append(out, m)
	}
	return out
}

// Obj returns the body as an object (failing the test when it is not one).
func (r Result) Obj(t *testing.T) map[string]any {
	t.Helper()
	m, ok := r.Body.(map[string]any)
	if !ok {
		t.Fatalf("not an object: %s", r.Raw)
	}
	return m
}

// IDs lists the "id" of every object in a list body.
func (r Result) IDs(t *testing.T) []string {
	t.Helper()
	var out []string
	for _, m := range r.List(t) {
		out = append(out, m["id"].(string))
	}
	return out
}

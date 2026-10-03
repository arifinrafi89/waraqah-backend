package contract_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/arifinrafi89/waraqah-backend/internal/app"
	"github.com/arifinrafi89/waraqah-backend/internal/contract"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/auth"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/clock"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/config"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/dbtest"
	"github.com/arifinrafi89/waraqah-backend/internal/seed"
)

const (
	goldenDir   = "../../testdata/contract"
	pendingFile = "pending.txt"
	apiPrefix   = "/v1"
)

// accounts maps a golden's "as" to the seeded account that sends it.
var accounts = map[string]struct {
	id   string
	role auth.Role
}{
	"reader":    {"u_reader", auth.RoleReader},
	"moderator": {"u_moderator", auth.RoleModerator},
	"catalog":   {"u_catalog", auth.RoleCatalogManager},
	"support":   {"u_support", auth.RoleSupport},
	"admin":     {"u_admin", auth.RoleSuperAdmin},
}

type rig struct {
	deps *app.Deps
	h    http.Handler
	// subst maps ids the fake API made up (a new order number, a new list id) to the ids this
	// backend made for the same step, so a later golden that names one still finds it.
	subst map[string]string
}

// newRig boots the app on a seeded database that is rolled back when the test ends.
func newRig(t *testing.T) *rig {
	t.Helper()
	d := dbtest.New(t)
	cfg, err := config.LoadFrom(os.LookupEnv)
	if err != nil {
		t.Fatal(err)
	}
	cfg.BcryptCost = 4                                   // seeding hashes one password per account; keep the test quick
	cfg.SeedDemoPassword = "Waraqah#Demo1"               // the password the login golden sends
	cfg.AppEnv, cfg.OTPDevCode = "development", "123456" // the sign-up goldens use the dev code
	cfg.AuthRatePerMin = 100000
	cfg.DemoBotDelay = 0 // the demo readers answer at once, as the fake API does
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	loc, err := clock.Location(cfg.AppTimezone)
	if err != nil {
		t.Fatal(err)
	}
	err = seed.All(context.Background(), d, seed.Options{Dir: "../../seed", DemoPassword: cfg.SeedDemoPassword, BcryptCost: cfg.BcryptCost, Loc: loc, Log: log})
	if err != nil {
		t.Fatal(err)
	}
	deps, err := app.NewDeps(cfg, log, d)
	if err != nil {
		t.Fatal(err)
	}
	// Home's Season and the Ayah of the day depend on the day: replay on the day of the export.
	meta, err := contract.LoadMeta(goldenDir, loc)
	if err != nil {
		t.Fatal(err)
	}
	deps.Clock = clock.NewFake(meta.ExportedAt)
	// The google golden sends "sample-id-token"; the real verifier would refuse it.
	deps.Google = auth.FakeGoogle{Identities: map[string]auth.GoogleIdentity{
		"sample-id-token": {Subject: "g-sample", Email: "reader@waraqah.test", Name: "Reader"},
	}}
	return &rig{deps: deps, h: app.Routes(deps), subst: map[string]string{}}
}

func (r *rig) send(t *testing.T, g contract.Golden, ctxTimeout time.Duration) *httptest.ResponseRecorder {
	t.Helper()
	var body io.Reader
	if g.Request.Body != nil {
		b, _ := json.Marshal(g.Request.Body)
		body = bytes.NewReader([]byte(r.substitute(string(b))))
	}
	req := httptest.NewRequest(g.Request.Method, r.substitute(g.Request.URL(apiPrefix)), body)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if acc, ok := accounts[g.Request.As]; ok {
		tok, _, err := r.deps.JWT.Sign(acc.id, acc.role)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	if ctxTimeout > 0 {
		ctx, cancel := context.WithTimeout(req.Context(), ctxTimeout)
		defer cancel()
		req = req.WithContext(ctx)
	}
	rec := httptest.NewRecorder()
	r.h.ServeHTTP(rec, req)
	return rec
}

func (r *rig) substitute(s string) string {
	for from, to := range r.subst {
		s = strings.ReplaceAll(s, from, to)
	}
	return s
}

// idKeys are the fields whose values the backend makes up itself.
var idKeys = map[string]bool{"id": true, "number": true, "orderNumber": true, "lineId": true}

// learnIDs records where our answer used a different generated id than the golden.
func (r *rig) learnIDs(want, got any) {
	switch w := want.(type) {
	case map[string]any:
		g, ok := got.(map[string]any)
		if !ok {
			return
		}
		for k, wv := range w {
			if ws, isStr := wv.(string); isStr && idKeys[k] {
				if gs, ok := g[k].(string); ok && gs != ws && ws != "" {
					r.subst[ws] = gs
				}
				continue
			}
			r.learnIDs(wv, g[k])
		}
	case []any:
		if g, ok := got.([]any); ok {
			for i := range w {
				if i < len(g) {
					r.learnIDs(w[i], g[i])
				}
			}
		}
	}
}

// TestContract replays every golden in the exporter's order and compares shapes. Endpoints in
// pending.txt are skipped and counted; T19 requires that file to be empty.
func TestContract(t *testing.T) {
	goldens, err := contract.LoadGoldens(goldenDir)
	if err != nil {
		t.Fatal(err)
	}
	pending, err := contract.LoadPending(pendingFile)
	if err != nil {
		t.Fatal(err)
	}
	r := newRig(t)
	skipped, ran := 0, 0
	for _, g := range goldens {
		if pending[g.Request.Key()] {
			skipped++
			continue
		}
		ran++
		t.Run(filepath.Base(g.File), func(t *testing.T) {
			if g.SSE {
				rec := r.send(t, g, 150*time.Millisecond)
				if rec.Code != 200 || rec.Header().Get("Content-Type") != "text/event-stream" {
					t.Errorf("SSE: %d %s", rec.Code, rec.Header().Get("Content-Type"))
				}
				return
			}
			rec := r.send(t, g, 0)
			if os.Getenv("CONTRACT_DEBUG") != "" {
				t.Logf("%s %s -> %s", g.Request.Key(), g.Request.Body, rec.Body)
			}
			if rec.Code != 200 {
				t.Fatalf("status %d: %s", rec.Code, rec.Body)
			}
			var got any
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("answer is not JSON: %v", err)
			}
			for _, p := range contract.Shape(g.Response, got) {
				t.Error(p)
			}
			r.learnIDs(g.Response, got)
		})
	}
	t.Logf("contract: %d replayed, %d pending", ran, skipped)
}

// TestHealthzSample is the one always-on check while endpoints are still pending.
func TestHealthzSample(t *testing.T) {
	r := &rig{}
	cfg, _ := config.LoadFrom(os.LookupEnv)
	deps, err := app.NewDeps(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)), nil)
	if err != nil {
		t.Fatal(err)
	}
	r.h = app.Routes(deps)
	rec := httptest.NewRecorder()
	r.h.ServeHTTP(rec, httptest.NewRequest("GET", "/healthz", nil))
	var got any
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if problems := contract.Shape(map[string]any{"ok": true}, got); len(problems) > 0 || rec.Code != 200 {
		t.Errorf("healthz: %d %v", rec.Code, problems)
	}
}

package auth

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/httpx"
)

type fakeUsers map[string]User // id -> user; missing = deleted or banned

func (f fakeUsers) Lookup(_ context.Context, id string) (User, error) {
	if u, ok := f[id]; ok {
		return u, nil
	}
	return User{}, ErrNoUser
}

type rig struct {
	m      *Middleware
	j      *JWT
	logbuf *bytes.Buffer
}

func newRig(t *testing.T) *rig {
	t.Helper()
	clk := newClock()
	j, _ := NewJWT(testSecret, "waraqah", time.Hour, clk)
	users := fakeUsers{}
	for _, r := range []Role{RoleReader, RoleModerator, RoleCatalogManager, RoleSupport, RoleSuperAdmin} {
		users["u_"+string(r)] = User{ID: "u_" + string(r), Role: r}
	}
	return &rig{m: NewMiddleware(j, users), j: j, logbuf: &bytes.Buffer{}}
}

func (g *rig) token(t *testing.T, role Role) string {
	t.Helper()
	tok, _, err := g.j.Sign("u_"+string(role), role)
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

// call runs one request through the wrapped handler and says what the handler saw.
func (g *rig) call(level func(http.Handler) http.Handler, method, authHeader string) (code int, sawUser bool) {
	log := slog.New(slog.NewTextHandler(g.logbuf, nil))
	h := httpx.Chain(level(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, sawUser = UserFrom(r.Context())
	})), httpx.RequestID(log), httpx.AccessLog())
	req := httptest.NewRequest(method, "/x", strings.NewReader("{}"))
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Code, sawUser
}

func TestMiddlewareMatrix(t *testing.T) {
	g := newRig(t)
	bearer := func(r Role) string { return "Bearer " + g.token(t, r) }
	expired, _ := NewJWT(testSecret, "waraqah", -time.Hour, newClock())
	exp, _, _ := expired.Sign("u_reader", RoleReader)

	levels := map[string]func(http.Handler) http.Handler{
		"public": g.m.Public, "me": g.m.Me, "authed": g.m.Authed, "staff": g.m.Staff,
		"catalog":  func(h http.Handler) http.Handler { return g.m.StaffCan(PermCatalog, h) },
		"orders":   func(h http.Handler) http.Handler { return g.m.StaffCan(PermOrders, h) },
		"moderate": func(h http.Handler) http.Handler { return g.m.StaffCan(PermModerate, h) },
	}
	staffOK := map[string]map[Role]bool{
		"staff":    {RoleModerator: true, RoleCatalogManager: true, RoleSupport: true, RoleSuperAdmin: true},
		"catalog":  {RoleCatalogManager: true, RoleSuperAdmin: true},
		"orders":   {RoleSupport: true, RoleSuperAdmin: true},
		"moderate": {RoleModerator: true, RoleSuperAdmin: true},
	}
	for name, level := range levels {
		// no token
		code, user := g.call(level, "GET", "")
		switch name {
		case "public", "me":
			if code != 200 || user {
				t.Errorf("%s GET without token: %d user=%v", name, code, user)
			}
		default:
			if code != 401 {
				t.Errorf("%s GET without token: %d", name, code)
			}
		}
		// POST without a token: only public lets it through
		code, _ = g.call(level, "POST", "")
		if (name == "public") != (code == 200) {
			t.Errorf("%s POST without token: %d", name, code)
		}
		// bad tokens: public treats as a guest, every other level answers 401
		for _, bad := range []string{"Bearer garbage", "Bearer " + exp, "Basic abc", "Bearer "} {
			code, user = g.call(level, "GET", bad)
			if name == "public" {
				if code != 200 || user {
					t.Errorf("public with bad token %q: %d user=%v", bad, code, user)
				}
			} else if code != 401 {
				t.Errorf("%s with bad token %q: %d", name, bad, code)
			}
		}
		// each role
		for _, r := range []Role{RoleReader, RoleModerator, RoleCatalogManager, RoleSupport, RoleSuperAdmin} {
			code, user = g.call(level, "POST", bearer(r))
			ok, isStaffLevel := staffOK[name]
			wantCode := 200
			if isStaffLevel && !ok[r] {
				wantCode = 403
				if r == RoleReader && name == "staff" {
					wantCode = 403
				}
			}
			if code != wantCode || (code == 200 && !user) {
				t.Errorf("%s as %s: got %d user=%v want %d", name, r, code, user, wantCode)
			}
		}
	}
}

func TestBannedOrDeletedUserIsRefused(t *testing.T) {
	g := newRig(t)
	tok, _, _ := g.j.Sign("u_gone", RoleReader) // not in the user source
	h := "Bearer " + tok
	if code, _ := g.call(g.m.Me, "GET", h); code != 401 {
		t.Errorf("me: %d", code)
	}
	if code, user := g.call(g.m.Public, "GET", h); code != 200 || user {
		t.Errorf("public: %d user=%v", code, user)
	}
}

func TestRoleComesFromTheDatabaseNotTheToken(t *testing.T) {
	g := newRig(t)
	// A token claiming superAdmin for an account that is only a reader.
	tok, _, _ := g.j.Sign("u_reader", RoleSuperAdmin)
	if code, _ := g.call(g.m.Staff, "GET", "Bearer "+tok); code != 403 {
		t.Errorf("forged role claim: %d", code)
	}
}

func TestLogsNeverContainTokens(t *testing.T) {
	g := newRig(t)
	tok := g.token(t, RoleModerator)
	g.call(g.m.Staff, "POST", "Bearer "+tok)
	g.call(g.m.Staff, "POST", "Bearer "+tok+"tampered")
	logs := g.logbuf.String()
	if strings.Contains(logs, tok) || strings.Contains(logs, "Bearer") || strings.Contains(logs, "Authorization") {
		t.Errorf("log leaks a credential:\n%s", logs)
	}
	if !strings.Contains(logs, "user_id=u_moderator") {
		t.Errorf("log lacks user_id:\n%s", logs)
	}
}

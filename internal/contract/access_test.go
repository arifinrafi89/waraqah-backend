package contract_test

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/arifinrafi89/waraqah-backend/internal/contract"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/auth"
)

// accessRow is one Appendix A row with its auth level: public, me, staff or staff:<perm>.
var accessRow = regexp.MustCompile("^\\|\\s*(GET|POST)( \\(SSE\\))?\\s*\\|\\s*`([^`]+)`\\s*\\|\\s*([a-z:]+)\\s*\\|")

type endpoint struct {
	method, path, level string
	sse                 bool
}

func appendixEndpoints(t *testing.T) []endpoint {
	t.Helper()
	plan, err := os.ReadFile("../../BACKEND_PLAN.md")
	if err != nil {
		t.Fatal(err)
	}
	var out []endpoint
	for _, line := range strings.Split(string(plan), "\n") {
		if m := accessRow.FindStringSubmatch(strings.TrimRight(line, "\r")); m != nil {
			out = append(out, endpoint{method: m[1], sse: m[2] != "", path: m[3], level: m[4]})
		}
	}
	if len(out) != 174 {
		t.Fatalf("Appendix A lists %d endpoints with an auth level, want 174", len(out))
	}
	return out
}

// call sends one request as a role ("" for a guest) and answers the status and the decoded body.
// Live streams are cut after a moment.
func (r *rig) call(t *testing.T, e endpoint, role auth.Role) (int, any) {
	t.Helper()
	var body *strings.Reader
	if e.method == "POST" {
		body = strings.NewReader("{}")
	} else {
		body = strings.NewReader("")
	}
	req := httptest.NewRequest(e.method, apiPrefix+e.path, body)
	if role != "" {
		id := map[auth.Role]string{auth.RoleReader: "u_reader", auth.RoleModerator: "u_moderator", auth.RoleCatalogManager: "u_catalog",
			auth.RoleSupport: "u_support", auth.RoleSuperAdmin: "u_admin"}[role]
		tok, _, err := r.deps.JWT.Sign(id, role)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	if e.sse {
		ctx, cancel := context.WithTimeout(req.Context(), 50*time.Millisecond)
		defer cancel()
		req = req.WithContext(ctx)
	}
	rec := httptest.NewRecorder()
	r.h.ServeHTTP(rec, req)
	var v any
	if !e.sse {
		_ = json.Unmarshal(rec.Body.Bytes(), &v)
	}
	return rec.Code, v
}

// TestGuestWalk is the guest walk of plan §5.4 as a test: signed out, a public or `me` GET answers
// 200 (a `me` list or object comes back empty, never null), while a change, a live stream or a
// staff endpoint answers 401.
func TestGuestWalk(t *testing.T) {
	r := newRig(t)
	goldens, err := contract.LoadGoldens(goldenDir)
	if err != nil {
		t.Fatal(err)
	}
	shapeOf := map[string]any{}
	for _, g := range goldens {
		if len(g.Request.Query) == 0 && !strings.Contains(g.File, "__miss") {
			shapeOf[g.Request.Key()] = g.Response
		}
	}
	checked, empties := 0, 0
	for _, e := range appendixEndpoints(t) {
		status, body := r.call(t, e, "")
		want := 200
		if e.level != "public" && (e.method == "POST" || e.sse || strings.HasPrefix(e.level, "staff")) {
			want = 401
		}
		if strings.HasPrefix(e.path, "/auth/") && e.method == "POST" {
			continue // the sign-in endpoints answer refusals for an empty body; covered by the auth tests
		}
		checked++
		if status != want {
			t.Errorf("guest %s %s (%s): %d, want %d", e.method, e.path, e.level, status, want)
			continue
		}
		if e.level == "me" && e.method == "GET" && !e.sse {
			if golden, ok := shapeOf[e.method+" "+e.path]; ok && golden != nil {
				empties++
				if body == nil {
					t.Errorf("guest %s: null, want the empty value", e.path)
				}
			}
		}
	}
	t.Logf("guest walk: %d endpoints, %d me GETs answered an empty value", checked, empties)
}

// TestStaffRoleWalk is the staff role walk: each demo account reaches only the admin endpoints its
// role allows (403 otherwise), and a reader reaches none.
func TestStaffRoleWalk(t *testing.T) {
	r := newRig(t)
	checked := 0
	defer func() { t.Logf("staff role walk: %d endpoint and role pairs", checked) }()
	roles := []auth.Role{auth.RoleReader, auth.RoleModerator, auth.RoleCatalogManager, auth.RoleSupport, auth.RoleSuperAdmin}
	for _, e := range appendixEndpoints(t) {
		if !strings.HasPrefix(e.level, "staff") {
			continue
		}
		for _, role := range roles {
			allowed := role.IsStaff()
			if perm, ok := strings.CutPrefix(e.level, "staff:"); ok {
				allowed = role.Can(auth.Perm(perm))
			}
			status, _ := r.call(t, e, role)
			checked++
			if allowed && (status == 401 || status == 403) {
				t.Errorf("%s %s as %s: %d, want allowed", e.method, e.path, role, status)
			}
			if !allowed && status != 403 {
				t.Errorf("%s %s as %s: %d, want 403", e.method, e.path, role, status)
			}
		}
	}
}

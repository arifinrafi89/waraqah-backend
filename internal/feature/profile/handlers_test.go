package profile_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/arifinrafi89/waraqah-backend/internal/app"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/auth"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/config"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/dbtest"
)

type env struct {
	t    *testing.T
	h    http.Handler
	deps *app.Deps
	tok  map[string]string
}

func newEnv(t *testing.T) *env {
	t.Helper()
	d := dbtest.New(t)
	cfg, err := config.LoadFrom(os.LookupEnv)
	if err != nil {
		t.Fatal(err)
	}
	cfg.AppEnv, cfg.BcryptCost = "development", 4
	deps, err := app.NewDeps(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)), d)
	if err != nil {
		t.Fatal(err)
	}
	e := &env{t: t, h: app.Routes(deps), deps: deps, tok: map[string]string{}}
	for _, id := range []string{"u_ann", "u_bob"} {
		_, err := d.Conn.Exec(context.Background(), "INSERT INTO users (id, email, name) VALUES ($1, $2, $3)", id, id+"@example.test", "Name "+id)
		if err != nil {
			t.Fatal(err)
		}
		tok, _, _ := deps.JWT.Sign(id, auth.RoleReader)
		e.tok[id] = tok
	}
	return e
}

// call sends a request as a user ("" for a guest) and returns the status, refusal header and body.
func (e *env) call(as, method, path string, body any) (int, string, any) {
	e.t.Helper()
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, "/v1"+path, rd)
	if as != "" {
		req.Header.Set("Authorization", "Bearer "+e.tok[as])
	}
	rec := httptest.NewRecorder()
	e.h.ServeHTTP(rec, req)
	var out any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return rec.Code, rec.Header().Get("X-Waraqah-Error"), out
}

var addr = map[string]any{"label": "Home", "recipient": "Ann", "phone": "+8801712345678", "line": "House 1",
	"upazila": "Savar", "district": "Dhaka", "division": "Dhaka"}

func with(m map[string]any, k string, v any) map[string]any {
	out := map[string]any{}
	for kk, vv := range m {
		out[kk] = vv
	}
	out[k] = v
	return out
}

func ids(list any) []string {
	var out []string
	for _, a := range list.([]any) {
		out = append(out, a.(map[string]any)["id"].(string))
	}
	return out
}

func TestGuestsGetEmptyAnswers(t *testing.T) {
	e := newEnv(t)
	for path, want := range map[string]string{"/profile": `{"name":"","phone":"","photo":null}`, "/addresses": `[]`,
		"/profile/prefs": `{"muted":[],"profileVisible":true,"activityVisible":true}`, "/notifications": `[]`} {
		code, _, out := e.call("", "GET", path, nil)
		var wantV any
		_ = json.Unmarshal([]byte(want), &wantV)
		if code != 200 || !reflect.DeepEqual(out, wantV) {
			t.Errorf("guest %s: %d %v want %s", path, code, out, want)
		}
	}
	if code, _, _ := e.call("", "POST", "/profile/save", map[string]any{"name": "X"}); code != 401 {
		t.Errorf("guest POST: %d", code)
	}
	if code, _, out := e.call("", "GET", "/geo", nil); code != 200 || len(out.([]any)) != 8 {
		t.Errorf("geo: %d", code)
	}
}

func TestSaveProfile(t *testing.T) {
	e := newEnv(t)
	code, refusal, out := e.call("u_ann", "POST", "/profile/save", map[string]any{"name": "  Ann Hossain ", "phone": "+8801712345678"})
	m := out.(map[string]any)
	if code != 200 || refusal != "" || m["name"] != "Ann Hossain" || m["phone"] != "01712345678" {
		t.Fatalf("save: %d %q %v", code, refusal, out)
	}
	_, _, got := e.call("u_ann", "GET", "/profile", nil)
	if got.(map[string]any)["name"] != "Ann Hossain" {
		t.Errorf("not saved: %v", got)
	}
	for _, c := range []struct {
		name  string
		body  map[string]any
		refus string
	}{
		{"short name", map[string]any{"name": " A "}, "profile_invalid"},
		{"bad phone", map[string]any{"name": "Ann", "phone": "0121"}, "profile_invalid"},
		{"not an image", map[string]any{"name": "Ann", "photo": base64.StdEncoding.EncodeToString([]byte("<html>"))}, "photo_invalid"},
	} {
		if code, refusal, out := e.call("u_ann", "POST", "/profile/save", c.body); code != 200 || refusal != c.refus || out != nil {
			t.Errorf("%s: %d %q %v", c.name, code, refusal, out)
		}
	}
	png := append([]byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}, make([]byte, 16)...)
	b64 := base64.StdEncoding.EncodeToString(png)
	_, _, with := e.call("u_ann", "POST", "/profile/save", map[string]any{"name": "Ann", "photo": b64})
	if with.(map[string]any)["photo"] != b64 {
		t.Error("photo not kept")
	}
	_, _, without := e.call("u_ann", "POST", "/profile/save", map[string]any{"name": "Ann", "photo": nil})
	if without.(map[string]any)["photo"] != nil {
		t.Error("photo not removed")
	}
	// another reader's profile is untouched
	if _, _, bob := e.call("u_bob", "GET", "/profile", nil); bob.(map[string]any)["name"] != "Name u_bob" {
		t.Errorf("bob: %v", bob)
	}
}

func TestPrefsKeepOnlyKnownGroups(t *testing.T) {
	e := newEnv(t)
	_, _, out := e.call("u_ann", "POST", "/profile/prefs/save", map[string]any{"muted": []string{"community", "bogus", "orders", "orders"}, "profileVisible": false, "activityVisible": true})
	var want any
	_ = json.Unmarshal([]byte(`{"muted":["orders","community"],"profileVisible":false,"activityVisible":true}`), &want)
	if !reflect.DeepEqual(out, want) {
		t.Errorf("saved prefs: %v", out)
	}
	_, _, again := e.call("u_ann", "GET", "/profile/prefs", nil)
	if again.(map[string]any)["profileVisible"] != false {
		t.Errorf("not remembered: %v", again)
	}
	_, _, other := e.call("u_bob", "GET", "/profile/prefs", nil)
	if other.(map[string]any)["profileVisible"] != true {
		t.Error("prefs leaked to another reader")
	}
}

func TestAddressesLikeTheFakeStore(t *testing.T) {
	e := newEnv(t)
	_, _, first := e.call("u_ann", "POST", "/addresses/save", addr)
	a := ids(first)
	if len(a) != 1 || first.([]any)[0].(map[string]any)["isDefault"] != true || first.([]any)[0].(map[string]any)["phone"] != "01712345678" {
		t.Fatalf("first address: %v", first)
	}
	_, _, second := e.call("u_ann", "POST", "/addresses/save", with(addr, "label", "Office"))
	_, _, third := e.call("u_ann", "POST", "/addresses/save", with(addr, "label", "Family"))
	list := ids(third)
	if len(list) != 3 || list[0] != a[0] {
		t.Fatalf("default must come first, then in the order added: %v", list)
	}
	_ = second
	// make the third the default: it moves to the top
	_, _, moved := e.call("u_ann", "POST", "/addresses/default", map[string]any{"id": list[2]})
	if got := ids(moved); got[0] != list[2] || got[1] != list[0] || got[2] != list[1] {
		t.Errorf("after default: %v", got)
	}
	// deleting the default promotes the next one added
	_, _, after := e.call("u_ann", "POST", "/addresses/delete", map[string]any{"id": list[2]})
	if got := ids(after); len(got) != 2 || got[0] != list[0] || after.([]any)[0].(map[string]any)["isDefault"] != true {
		t.Errorf("after deleting the default: %v", after)
	}
	// edit keeps default flag and id
	_, _, edited := e.call("u_ann", "POST", "/addresses/save", with(with(addr, "id", list[1]), "label", "Work"))
	if edited.([]any)[1].(map[string]any)["label"] != "Work" || edited.([]any)[1].(map[string]any)["isDefault"] != false {
		t.Errorf("edit: %v", edited)
	}
	// refusals
	if _, refusal, out := e.call("u_ann", "POST", "/addresses/save", with(addr, "phone", "0171")); refusal != "address_invalid" || out != nil {
		t.Errorf("invalid: %q", refusal)
	}
	for _, path := range []string{"/addresses/delete", "/addresses/default"} {
		if _, refusal, out := e.call("u_ann", "POST", path, map[string]any{"id": "addr-nope"}); refusal != "address_unknown" || out != nil {
			t.Errorf("%s unknown: %q", path, refusal)
		}
	}
	// one reader cannot touch another reader's address
	if _, refusal, _ := e.call("u_bob", "POST", "/addresses/delete", map[string]any{"id": list[0]}); refusal != "address_unknown" {
		t.Errorf("bob deleting ann's address: %q", refusal)
	}
	if _, _, bobs := e.call("u_bob", "GET", "/addresses", nil); len(bobs.([]any)) != 0 {
		t.Error("addresses leaked")
	}
}

func TestDeleteAccount(t *testing.T) {
	e := newEnv(t)
	hooked := ""
	e.deps.Profile.Hooks = append(e.deps.Profile.Hooks, func(_ context.Context, id string) error { hooked = id; return nil })
	e.call("u_ann", "POST", "/addresses/save", addr)
	rt, _, _ := e.deps.Refresh.Issue(context.Background(), "u_ann")
	if code, _, out := e.call("u_ann", "POST", "/auth/delete", map[string]any{}); code != 200 || out.(map[string]any)["ok"] != true {
		t.Fatalf("delete: %d %v", code, out)
	}
	if hooked != "u_ann" {
		t.Error("the delete hook did not run")
	}
	// the old access token no longer works (the user is gone), the refresh token is revoked
	if code, _, _ := e.call("u_ann", "GET", "/profile", nil); code != 401 {
		t.Errorf("deleted user token: %d", code)
	}
	if _, _, _, err := e.deps.Refresh.Rotate(context.Background(), rt); err == nil {
		t.Error("refresh token survived the deletion")
	}
	var email *string
	var name string
	_ = e.deps.DB.Conn.QueryRow(context.Background(), "SELECT email, name FROM users WHERE id = 'u_ann'").Scan(&email, &name)
	if email != nil || strings.Contains(name, "ann") {
		t.Errorf("not anonymised: %v %q", email, name)
	}
	// the email can sign up again
	if code, _, _ := e.call("", "POST", "/auth/signup/request-otp", map[string]any{"name": "Ann", "contact": "u_ann@example.test", "password": "x"}); code != 200 {
		t.Error("email not freed")
	}
}

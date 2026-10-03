package auth_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/arifinrafi89/waraqah-backend/internal/app"
	pauth "github.com/arifinrafi89/waraqah-backend/internal/platform/auth"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/config"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/dbtest"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/email"
)

type env struct {
	h     http.Handler
	mail  *email.Recorder
	deps  *app.Deps
	t     *testing.T
	login string
}

func newEnv(t *testing.T, tweak func(*config.Config)) *env {
	t.Helper()
	d := dbtest.New(t)
	cfg, err := config.LoadFrom(os.LookupEnv)
	if err != nil {
		t.Fatal(err)
	}
	cfg.AppEnv, cfg.OTPDevCode, cfg.BcryptCost = "development", "", 4
	cfg.AuthRatePerMin = 1000
	if tweak != nil {
		tweak(cfg)
	}
	deps, err := app.NewDeps(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)), d)
	if err != nil {
		t.Fatal(err)
	}
	mail := &email.Recorder{}
	deps.Email = mail
	deps.Google = pauth.FakeGoogle{Identities: map[string]pauth.GoogleIdentity{
		"good": {Subject: "g1", Email: "Gina@Example.test", Name: "Gina", Picture: "https://example.invalid/p.png"},
	}}
	return &env{h: app.Routes(deps), mail: mail, deps: deps, t: t}
}

// post sends a JSON body and returns the status, the refusal header and the decoded answer.
func (e *env) post(path string, body any) (int, string, map[string]any) {
	e.t.Helper()
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/v1"+path, bytes.NewReader(b))
	req.RemoteAddr = "10.1.1.1:1111"
	rec := httptest.NewRecorder()
	e.h.ServeHTTP(rec, req)
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return rec.Code, rec.Header().Get("X-Waraqah-Error"), out
}

func (e *env) lastCode() string {
	e.t.Helper()
	if len(e.mail.Sent) == 0 {
		e.t.Fatal("no email was sent")
	}
	return e.mail.Sent[len(e.mail.Sent)-1].Code
}

func (e *env) signUp(addr string) map[string]any {
	e.t.Helper()
	if code, refusal, out := e.post("/auth/signup/request-otp", map[string]string{"name": "Rafiq", "contact": addr, "password": "Secret#1"}); code != 200 || refusal != "" || out["ok"] != true {
		e.t.Fatalf("request-otp: %d %q %v", code, refusal, out)
	}
	code, refusal, out := e.post("/auth/signup/verify-otp", map[string]string{"contact": addr, "otp": e.lastCode()})
	if code != 200 || refusal != "" || out == nil {
		e.t.Fatalf("verify-otp: %d %q %v", code, refusal, out)
	}
	return out
}

func TestSignUpLoginRefreshLogout(t *testing.T) {
	e := newEnv(t, nil)
	s := e.signUp("Rafiq@Example.test")
	if s["email"] != "rafiq@example.test" || s["role"] != "reader" || s["name"] != "Rafiq" || !strings.HasPrefix(s["id"].(string), "u_") {
		t.Fatalf("user: %v", s)
	}
	for _, k := range []string{"accessToken", "refreshToken", "expiresAt"} {
		if s[k] == nil || s[k] == "" {
			t.Errorf("%s missing", k)
		}
	}
	if !strings.Contains(s["expiresAt"].(string), "+06:00") {
		t.Errorf("expiresAt is not in the Dhaka offset: %v", s["expiresAt"])
	}
	// the new account signs in with its password
	code, refusal, out := e.post("/auth/login", map[string]string{"email": "RAFIQ@example.test", "password": "Secret#1"})
	if code != 200 || refusal != "" || out["id"] != s["id"] {
		t.Fatalf("login: %d %q %v", code, refusal, out)
	}
	// refresh rotates; the old token no longer works and reuse kills the family
	old := out["refreshToken"].(string)
	code, _, next := e.post("/auth/refresh", map[string]string{"refreshToken": old})
	if code != 200 || next["accessToken"] == nil || next["refreshToken"] == old {
		t.Fatalf("refresh: %d %v", code, next)
	}
	if code, _, _ := e.post("/auth/refresh", map[string]string{"refreshToken": old}); code != 401 {
		t.Errorf("reused refresh token: %d", code)
	}
	if code, _, _ := e.post("/auth/refresh", map[string]string{"refreshToken": next["refreshToken"].(string)}); code != 401 {
		t.Errorf("token issued before the theft still works: %d", code)
	}
	// logout
	_, _, fresh := e.post("/auth/login", map[string]string{"email": "rafiq@example.test", "password": "Secret#1"})
	rt := fresh["refreshToken"].(string)
	if code, _, out := e.post("/auth/logout", map[string]string{"refreshToken": rt}); code != 200 || out["ok"] != true {
		t.Errorf("logout: %d %v", code, out)
	}
	if code, _, _ := e.post("/auth/refresh", map[string]string{"refreshToken": rt}); code != 401 {
		t.Errorf("refresh after logout: %d", code)
	}
}

func TestSignUpRefusals(t *testing.T) {
	e := newEnv(t, nil)
	e.signUp("taken@example.test")
	cases := []struct{ name, contact, password, want string }{
		{"phone", "01712345678", "x", "phone_not_supported"},
		{"phone with plus", "+8801712345678", "x", "phone_not_supported"},
		{"not an email", "not-an-email", "x", "contact_invalid"},
		{"taken", "TAKEN@example.test", "x", "email_taken"},
		{"empty password", "ok@example.test", "", "password_invalid"},
	}
	for _, c := range cases {
		code, refusal, out := e.post("/auth/signup/request-otp", map[string]string{"name": "N", "contact": c.contact, "password": c.password})
		if code != 200 || out != nil || refusal != c.want {
			t.Errorf("%s: %d %q %v", c.name, code, refusal, out)
		}
	}
	// a wrong code is refused, and so is a contact with no pending sign-up
	e.post("/auth/signup/request-otp", map[string]string{"name": "N", "contact": "pending@example.test", "password": "Secret#1"})
	if _, refusal, out := e.post("/auth/signup/verify-otp", map[string]string{"contact": "pending@example.test", "otp": "000000"}); refusal != "wrong_otp" || out != nil {
		t.Errorf("wrong code: %q %v", refusal, out)
	}
	if _, refusal, _ := e.post("/auth/signup/verify-otp", map[string]string{"contact": "nobody@example.test", "otp": "123456"}); refusal != "wrong_otp" {
		t.Errorf("no pending sign-up: %q", refusal)
	}
}

func TestDevCodeWorksOnlyInDevelopment(t *testing.T) {
	dev := newEnv(t, func(c *config.Config) { c.OTPDevCode = "123456" })
	dev.post("/auth/signup/request-otp", map[string]string{"name": "D", "contact": "dev@example.test", "password": "Secret#1"})
	if code, refusal, out := dev.post("/auth/signup/verify-otp", map[string]string{"contact": "dev@example.test", "otp": "123456"}); code != 200 || refusal != "" || out == nil {
		t.Errorf("dev code in development: %d %q", code, refusal)
	}
	test := newEnv(t, func(c *config.Config) { c.OTPDevCode = "123456"; c.AppEnv = "test" })
	test.post("/auth/signup/request-otp", map[string]string{"name": "D", "contact": "dev2@example.test", "password": "Secret#1"})
	if _, refusal, _ := test.post("/auth/signup/verify-otp", map[string]string{"contact": "dev2@example.test", "otp": "123456"}); refusal != "wrong_otp" {
		t.Errorf("dev code outside development must be refused, got %q", refusal)
	}
}

func TestLoginRefusals(t *testing.T) {
	e := newEnv(t, nil)
	e.signUp("login@example.test")
	for _, c := range []map[string]string{
		{"email": "login@example.test", "password": "wrong"},
		{"email": "nobody@example.test", "password": "Secret#1"},
		{"email": "", "password": ""},
	} {
		if code, refusal, out := e.post("/auth/login", c); code != 200 || refusal != "wrong_credentials" || out != nil {
			t.Errorf("%v: %d %q %v", c, code, refusal, out)
		}
	}
}

func TestPasswordReset(t *testing.T) {
	e := newEnv(t, nil)
	s := e.signUp("reset@example.test")
	old := s["refreshToken"].(string)
	// an unknown account answers ok and sends nothing
	sent := len(e.mail.Sent)
	if _, _, out := e.post("/auth/password/request-otp", map[string]string{"contact": "ghost@example.test"}); out["ok"] != true || len(e.mail.Sent) != sent {
		t.Errorf("unknown contact: %v sent=%d", out, len(e.mail.Sent)-sent)
	}
	if _, _, out := e.post("/auth/password/request-otp", map[string]string{"contact": "reset@example.test"}); out["ok"] != true {
		t.Fatalf("request: %v", out)
	}
	if e.mail.Sent[len(e.mail.Sent)-1].Purpose != "reset" {
		t.Fatal("no reset email")
	}
	if _, refusal, out := e.post("/auth/password/reset", map[string]string{"contact": "reset@example.test", "otp": "000000", "password": "New#Pass1"}); refusal != "wrong_otp" || out != nil {
		t.Errorf("wrong code: %q %v", refusal, out)
	}
	code := e.lastCode()
	if _, refusal, out := e.post("/auth/password/reset", map[string]string{"contact": "reset@example.test", "otp": code, "password": "New#Pass1"}); refusal != "" || out["ok"] != true {
		t.Fatalf("reset: %q %v", refusal, out)
	}
	if _, refusal, _ := e.post("/auth/login", map[string]string{"email": "reset@example.test", "password": "Secret#1"}); refusal != "wrong_credentials" {
		t.Error("old password still works")
	}
	if _, refusal, _ := e.post("/auth/login", map[string]string{"email": "reset@example.test", "password": "New#Pass1"}); refusal != "" {
		t.Error("new password does not work")
	}
	if code, _, _ := e.post("/auth/refresh", map[string]string{"refreshToken": old}); code != 401 {
		t.Errorf("a reset must sign the account out everywhere, refresh gave %d", code)
	}
}

func TestGoogle(t *testing.T) {
	e := newEnv(t, nil)
	if _, refusal, out := e.post("/auth/google", map[string]string{}); refusal != "google_token_missing" || out != nil {
		t.Errorf("missing token: %q", refusal)
	}
	if _, refusal, _ := e.post("/auth/google", map[string]string{"idToken": "bad"}); refusal != "google_token_invalid" {
		t.Errorf("bad token: %q", refusal)
	}
	_, refusal, first := e.post("/auth/google", map[string]string{"idToken": "good"})
	if refusal != "" || first["email"] != "gina@example.test" || first["name"] != "Gina" || first["accessToken"] == nil {
		t.Fatalf("first sign-in: %q %v", refusal, first)
	}
	_, _, second := e.post("/auth/google", map[string]string{"idToken": "good"})
	if second["id"] != first["id"] {
		t.Error("the same Google account must map to the same user")
	}
}

func TestRateLimit(t *testing.T) {
	e := newEnv(t, func(c *config.Config) { c.AuthRatePerMin = 2 })
	var codes []int
	for i := 0; i < 4; i++ {
		code, _, _ := e.post("/auth/login", map[string]string{"email": "x@example.test", "password": "x"})
		codes = append(codes, code)
	}
	if codes[0] != 200 || codes[1] != 200 || codes[2] != 429 || codes[3] != 429 {
		t.Errorf("status codes: %v", codes)
	}
}

func TestAccessTokenWorksOnTheMiddleware(t *testing.T) {
	e := newEnv(t, nil)
	s := e.signUp("mw@example.test")
	var got string
	h := e.deps.Auth.Me(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, _ := pauth.UserFrom(r.Context())
		got = u.ID
	}))
	req := httptest.NewRequest("POST", "/x", nil)
	req.Header.Set("Authorization", "Bearer "+s["accessToken"].(string))
	h.ServeHTTP(httptest.NewRecorder(), req)
	if got != s["id"] {
		t.Errorf("middleware saw %q want %v", got, s["id"])
	}
}

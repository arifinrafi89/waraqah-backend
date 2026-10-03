package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestJWTRoundTrip(t *testing.T) {
	clk := newClock()
	j, err := NewJWT(testSecret, "waraqah", 15*time.Minute, clk)
	if err != nil {
		t.Fatal(err)
	}
	tok, exp, err := j.Sign("u_1", RoleModerator)
	if err != nil {
		t.Fatal(err)
	}
	if !exp.Equal(clk.now.Add(15 * time.Minute)) {
		t.Errorf("expiry %v", exp)
	}
	c, err := j.Parse(tok)
	if err != nil || c.UserID != "u_1" || c.Role != RoleModerator {
		t.Fatalf("got %+v %v", c, err)
	}
}

func TestJWTRejectsBadTokens(t *testing.T) {
	clk := newClock()
	j, _ := NewJWT(testSecret, "waraqah", time.Minute, clk)
	tok, _, _ := j.Sign("u_1", RoleReader)

	clk.Advance(2 * time.Minute)
	if _, err := j.Parse(tok); err == nil {
		t.Error("expired token accepted")
	}
	clk.now = clk.now.Add(-2 * time.Minute)

	tampered := tok[:len(tok)-2] + "xx"
	if _, err := j.Parse(tampered); err == nil {
		t.Error("tampered token accepted")
	}
	other, _ := NewJWT(testSecret, "someone-else", time.Minute, clk)
	otok, _, _ := other.Sign("u_1", RoleReader)
	if _, err := j.Parse(otok); err == nil {
		t.Error("wrong issuer accepted")
	}
	// alg=none must never pass.
	none := jwt.NewWithClaims(jwt.SigningMethodNone, accessClaims{Role: RoleSuperAdmin,
		RegisteredClaims: jwt.RegisteredClaims{Subject: "u_1", Issuer: "waraqah", ExpiresAt: jwt.NewNumericDate(clk.now.Add(time.Hour))}})
	ntok, _ := none.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if _, err := j.Parse(ntok); err == nil {
		t.Error("alg none accepted")
	}
	// HS384 with the same secret must not pass either.
	h384, _ := jwt.NewWithClaims(jwt.SigningMethodHS384, accessClaims{Role: RoleReader,
		RegisteredClaims: jwt.RegisteredClaims{Subject: "u_1", Issuer: "waraqah", ExpiresAt: jwt.NewNumericDate(clk.now.Add(time.Hour))}}).SignedString([]byte(testSecret))
	if _, err := j.Parse(h384); err == nil {
		t.Error("HS384 accepted")
	}
	if _, err := j.Parse(strings.Repeat("a", 10)); err == nil {
		t.Error("garbage accepted")
	}
}

func TestShortSecretRefused(t *testing.T) {
	if _, err := NewJWT("short", "waraqah", time.Minute, nil); err == nil {
		t.Error("short secret accepted")
	}
}

func TestPassword(t *testing.T) {
	h, err := HashPassword("correct horse", 4)
	if err != nil {
		t.Fatal(err)
	}
	if !CheckPassword(h, "correct horse") || CheckPassword(h, "wrong") || CheckPassword("", "x") {
		t.Error("password check wrong")
	}
}

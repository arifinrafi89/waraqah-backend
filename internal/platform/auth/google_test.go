package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func jwksServer(t *testing.T, key *rsa.PublicKey, hits *atomic.Int32) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Cache-Control", "public, max-age=3600")
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]string{{
			"kid": "k1", "kty": "RSA", "alg": "RS256",
			"n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
			"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
		}}})
	}))
	t.Cleanup(srv.Close)
	return srv
}

func googleToken(t *testing.T, key *rsa.PrivateKey, mod func(*googleClaims)) string {
	t.Helper()
	c := googleClaims{Email: "g@example.test", EmailVerified: true, Name: "G", RegisteredClaims: jwt.RegisteredClaims{
		Issuer: "https://accounts.google.com", Subject: "sub-1", Audience: jwt.ClaimStrings{"client-1"},
		ExpiresAt: jwt.NewNumericDate(newClock().now.Add(time.Hour)),
	}}
	if mod != nil {
		mod(&c)
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, c)
	tok.Header["kid"] = "k1"
	s, err := tok.SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestGoogleVerifier(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	var hits atomic.Int32
	srv := jwksServer(t, &key.PublicKey, &hits)
	v := NewGoogleVerifier(srv.URL, []string{"client-1", "client-2"}, newClock())
	ctx := context.Background()

	id, err := v.Verify(ctx, googleToken(t, key, nil))
	if err != nil || id.Email != "g@example.test" || id.Subject != "sub-1" {
		t.Fatalf("good token: %+v %v", id, err)
	}
	_, _ = v.Verify(ctx, googleToken(t, key, nil))
	if hits.Load() != 1 {
		t.Errorf("keys fetched %d times, want 1 (cached by Cache-Control)", hits.Load())
	}

	bad := map[string]func(*googleClaims){
		"wrong audience":   func(c *googleClaims) { c.Audience = jwt.ClaimStrings{"other"} },
		"wrong issuer":     func(c *googleClaims) { c.Issuer = "https://evil.example" },
		"unverified email": func(c *googleClaims) { c.EmailVerified = false },
		"expired":          func(c *googleClaims) { c.ExpiresAt = jwt.NewNumericDate(newClock().now.Add(-time.Hour)) },
	}
	for name, mod := range bad {
		if _, err := v.Verify(ctx, googleToken(t, key, mod)); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
	other, _ := rsa.GenerateKey(rand.Reader, 2048)
	if _, err := v.Verify(ctx, googleToken(t, other, nil)); err == nil {
		t.Error("token signed by an unknown key accepted")
	}
}

func TestFakeGoogle(t *testing.T) {
	f := FakeGoogle{Identities: map[string]GoogleIdentity{"ok": {Email: "x@example.test"}}}
	if _, err := f.Verify(context.Background(), "ok"); err != nil {
		t.Error(err)
	}
	if _, err := f.Verify(context.Background(), "no"); err == nil {
		t.Error("unknown token accepted")
	}
}

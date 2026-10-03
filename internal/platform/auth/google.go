package auth

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ErrInvalidGoogleToken is returned for any Google ID token that does not check out.
var ErrInvalidGoogleToken = errors.New("invalid google id token")

// GoogleIdentity is what we keep from a verified Google ID token.
type GoogleIdentity struct {
	Subject string
	Email   string
	Name    string
	Picture string
}

// GoogleVerifier checks a Google ID token and returns who it belongs to.
type GoogleVerifier interface {
	Verify(ctx context.Context, idToken string) (GoogleIdentity, error)
}

type googleClaims struct {
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	jwt.RegisteredClaims
}

// FakeGoogle accepts the tokens in its map (used in tests and local development).
type FakeGoogle struct {
	Identities map[string]GoogleIdentity
}

// Verify returns the identity registered for the token.
func (f FakeGoogle) Verify(_ context.Context, idToken string) (GoogleIdentity, error) {
	if id, ok := f.Identities[idToken]; ok {
		return id, nil
	}
	return GoogleIdentity{}, ErrInvalidGoogleToken
}

// GoogleJWKS is the real verifier: it checks the signature against Google's public keys.
type GoogleJWKS struct {
	keys     *jwksCache
	audience []string
	clock    Clock
}

// NewGoogleVerifier builds the real verifier. certsURL is Google's JWKS endpoint
// (https://www.googleapis.com/oauth2/v3/certs); it is a parameter so tests can serve their own.
func NewGoogleVerifier(certsURL string, audience []string, clock Clock) *GoogleJWKS {
	if clock == nil {
		clock = systemClock{}
	}
	return &GoogleJWKS{keys: newJWKSCache(certsURL, clock), audience: audience, clock: clock}
}

// Verify checks signature, issuer, audience, expiry and a verified email.
func (g *GoogleJWKS) Verify(ctx context.Context, idToken string) (GoogleIdentity, error) {
	var c googleClaims
	_, err := jwt.ParseWithClaims(idToken, &c, func(t *jwt.Token) (any, error) {
		kid, _ := t.Header["kid"].(string)
		return g.keys.key(ctx, kid)
	}, jwt.WithValidMethods([]string{"RS256"}), jwt.WithExpirationRequired(), jwt.WithTimeFunc(g.clock.Now), jwt.WithLeeway(time.Minute))
	if err != nil {
		return GoogleIdentity{}, ErrInvalidGoogleToken
	}
	if c.Issuer != "accounts.google.com" && c.Issuer != "https://accounts.google.com" {
		return GoogleIdentity{}, ErrInvalidGoogleToken
	}
	if !slices.ContainsFunc(c.Audience, func(a string) bool { return slices.Contains(g.audience, a) }) {
		return GoogleIdentity{}, ErrInvalidGoogleToken
	}
	if !c.EmailVerified || c.Email == "" || c.Subject == "" {
		return GoogleIdentity{}, ErrInvalidGoogleToken
	}
	return GoogleIdentity{Subject: c.Subject, Email: c.Email, Name: c.Name, Picture: c.Picture}, nil
}

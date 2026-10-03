package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Clock is the time source; platform/clock provides the real and the test one.
type Clock interface{ Now() time.Time }

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now() }

// ErrInvalidToken is returned for any token that is malformed, tampered, expired or from another issuer.
var ErrInvalidToken = errors.New("invalid token")

// Claims are what an access token carries.
type Claims struct {
	UserID string
	Role   Role
	Expiry time.Time
}

type accessClaims struct {
	Role Role `json:"role"`
	jwt.RegisteredClaims
}

// JWT signs and parses HS256 access tokens.
type JWT struct {
	secret []byte
	issuer string
	ttl    time.Duration
	clock  Clock
}

// NewJWT builds the signer. The secret must be at least 32 bytes (config enforces it too).
func NewJWT(secret, issuer string, ttl time.Duration, clock Clock) (*JWT, error) {
	if len(secret) < 32 {
		return nil, errors.New("jwt secret must be at least 32 bytes")
	}
	if clock == nil {
		clock = systemClock{}
	}
	return &JWT{secret: []byte(secret), issuer: issuer, ttl: ttl, clock: clock}, nil
}

// Sign makes an access token for a user and returns it with its expiry.
func (j *JWT) Sign(userID string, role Role) (string, time.Time, error) {
	now := j.clock.Now()
	exp := now.Add(j.ttl)
	c := accessClaims{Role: role, RegisteredClaims: jwt.RegisteredClaims{
		Subject:   userID,
		Issuer:    j.issuer,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(exp),
	}}
	s, err := jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(j.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign token: %w", err)
	}
	return s, exp, nil
}

// Parse checks the algorithm, signature, issuer and expiry.
func (j *JWT) Parse(token string) (Claims, error) {
	var c accessClaims
	_, err := jwt.ParseWithClaims(token, &c, func(*jwt.Token) (any, error) { return j.secret, nil },
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithIssuer(j.issuer),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(j.clock.Now),
	)
	if err != nil || c.Subject == "" {
		return Claims{}, ErrInvalidToken
	}
	return Claims{UserID: c.Subject, Role: c.Role, Expiry: c.ExpiresAt.Time}, nil
}

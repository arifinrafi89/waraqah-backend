package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/db"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
)

// ErrInvalidRefresh is returned for an unknown, expired, revoked or reused refresh token.
var ErrInvalidRefresh = errors.New("invalid refresh token")

// Refresh issues, rotates and revokes opaque refresh tokens. Only SHA-256 hashes are stored.
type Refresh struct {
	db    *db.DB
	ttl   time.Duration
	clock Clock
}

// NewRefresh builds the refresh token service.
func NewRefresh(d *db.DB, ttl time.Duration, clock Clock) *Refresh {
	if clock == nil {
		clock = systemClock{}
	}
	return &Refresh{db: d, ttl: ttl, clock: clock}
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func newToken() (token, id string, err error) {
	var b [32]byte
	if _, err = rand.Read(b[:]); err != nil {
		return "", "", err
	}
	return base64.RawURLEncoding.EncodeToString(b[:]), "rt_" + hex.EncodeToString(b[:8]), nil
}

func (r *Refresh) issue(ctx context.Context, q *sqlc.Queries, userID string) (string, time.Time, error) {
	token, id, err := newToken()
	if err != nil {
		return "", time.Time{}, err
	}
	now := r.clock.Now()
	exp := now.Add(r.ttl)
	err = q.InsertRefreshToken(ctx, sqlc.InsertRefreshTokenParams{
		ID: id, UserID: userID, TokenHash: hashToken(token), ExpiresAt: exp, CreatedAt: now,
	})
	return token, exp, err
}

// Issue makes a new refresh token for a user.
func (r *Refresh) Issue(ctx context.Context, userID string) (string, time.Time, error) {
	return r.issue(ctx, r.db.Q(), userID)
}

// Rotate revokes the presented token and issues a new one. Presenting an already revoked
// token is treated as theft: every token of that user is revoked.
func (r *Refresh) Rotate(ctx context.Context, token string) (newToken, userID string, exp time.Time, err error) {
	reused := false
	err = r.db.WithTx(ctx, func(q *sqlc.Queries) error {
		row, e := q.GetRefreshTokenForUpdate(ctx, hashToken(token))
		if errors.Is(e, pgx.ErrNoRows) {
			return ErrInvalidRefresh
		}
		if e != nil {
			return e
		}
		now := r.clock.Now()
		if row.RevokedAt.Valid {
			reused = true // commit the revoke-all, then report the reuse below
			return q.RevokeAllRefreshTokens(ctx, sqlc.RevokeAllRefreshTokensParams{UserID: row.UserID, RevokedAt: now})
		}
		if !row.ExpiresAt.After(now) || row.UserBanned || row.UserDeleted {
			return ErrInvalidRefresh
		}
		if e := q.RevokeRefreshToken(ctx, sqlc.RevokeRefreshTokenParams{ID: row.ID, RevokedAt: now}); e != nil {
			return e
		}
		userID = row.UserID
		newToken, exp, e = r.issue(ctx, q, userID)
		return e
	})
	if err != nil {
		return "", "", time.Time{}, err
	}
	if reused {
		return "", "", time.Time{}, ErrInvalidRefresh
	}
	return newToken, userID, exp, nil
}

// Revoke revokes one token (logout). An unknown token is not an error.
func (r *Refresh) Revoke(ctx context.Context, token string) error {
	return r.db.Q().RevokeRefreshTokenByHash(ctx, sqlc.RevokeRefreshTokenByHashParams{
		TokenHash: hashToken(token), RevokedAt: r.clock.Now()})
}

// RevokeAll revokes every token of a user (password reset, account deletion, reuse).
func (r *Refresh) RevokeAll(ctx context.Context, userID string) error {
	return r.db.Q().RevokeAllRefreshTokens(ctx, sqlc.RevokeAllRefreshTokensParams{UserID: userID, RevokedAt: r.clock.Now()})
}

package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/db"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
)

// OTP purposes.
const (
	PurposeSignup = "signup"
	PurposeReset  = "reset"
)

// ErrResendTooSoon is returned when a code was sent to the contact less than OTPResendSeconds ago.
var ErrResendTooSoon = errors.New("otp requested too soon")

// OTPConfig is the part of Config the OTP service needs.
type OTPConfig struct {
	TTL         time.Duration
	MaxAttempts int
	Resend      time.Duration
	// DevCode is accepted as a valid code only when DevEnabled (APP_ENV=development).
	DevCode    string
	DevEnabled bool
}

// OTP makes and checks one-time codes. Only hashes are stored.
type OTP struct {
	db    *db.DB
	cfg   OTPConfig
	clock Clock
}

// NewOTP builds the OTP service.
func NewOTP(d *db.DB, cfg OTPConfig, clock Clock) *OTP {
	if clock == nil {
		clock = systemClock{}
	}
	return &OTP{db: d, cfg: cfg, clock: clock}
}

func hashCode(contact, purpose, code string) string {
	sum := sha256.Sum256([]byte(contact + "|" + purpose + "|" + code))
	return hex.EncodeToString(sum[:])
}

// Issue stores a fresh 6-digit code for the contact and returns it so the caller can send it.
func (o *OTP) Issue(ctx context.Context, contact, purpose string) (string, error) {
	now := o.clock.Now()
	prev, err := o.db.Q().GetOTP(ctx, sqlc.GetOTPParams{Contact: contact, Purpose: purpose})
	if err == nil && now.Sub(prev.CreatedAt) < o.cfg.Resend {
		return "", ErrResendTooSoon
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	code := fmt.Sprintf("%06d", n.Int64())
	err = o.db.Q().UpsertOTP(ctx, sqlc.UpsertOTPParams{
		Contact: contact, Purpose: purpose, CodeHash: hashCode(contact, purpose, code),
		ExpiresAt: now.Add(o.cfg.TTL), CreatedAt: now,
	})
	return code, err
}

// Verify checks a code. A right code is used up; a wrong one counts an attempt and the
// code dies after MaxAttempts. The dev code works only in development.
func (o *OTP) Verify(ctx context.Context, contact, purpose, code string) (bool, error) {
	if o.cfg.DevEnabled && o.cfg.DevCode != "" && code == o.cfg.DevCode {
		_ = o.db.Q().DeleteOTP(ctx, sqlc.DeleteOTPParams{Contact: contact, Purpose: purpose})
		return true, nil
	}
	ok := false
	err := o.db.WithTx(ctx, func(q *sqlc.Queries) error {
		row, err := q.GetOTPForUpdate(ctx, sqlc.GetOTPForUpdateParams{Contact: contact, Purpose: purpose})
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		del := sqlc.DeleteOTPParams{Contact: contact, Purpose: purpose}
		if !row.ExpiresAt.After(o.clock.Now()) {
			return q.DeleteOTP(ctx, del)
		}
		want := hashCode(contact, purpose, code)
		if subtle.ConstantTimeCompare([]byte(want), []byte(row.CodeHash)) == 1 {
			ok = true
			return q.DeleteOTP(ctx, del)
		}
		attempts, err := q.BumpOTPAttempts(ctx, sqlc.BumpOTPAttemptsParams{Contact: contact, Purpose: purpose})
		if err != nil {
			return err
		}
		if int(attempts) >= o.cfg.MaxAttempts {
			return q.DeleteOTP(ctx, del)
		}
		return nil
	})
	return ok, err
}

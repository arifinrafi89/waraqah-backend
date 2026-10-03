// Package auth serves the /auth endpoints: sign-in, Google sign-in, e-mail sign-up and
// password reset with one-time codes, and token refresh. Identity plumbing (tokens, OTP,
// middleware) lives in internal/platform/auth.
package auth

import (
	"context"
	"errors"
	"log/slog"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	pauth "github.com/arifinrafi89/waraqah-backend/internal/platform/auth"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/clock"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/email"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/ids"
)

// Service holds the sign-in rules.
type Service struct {
	DB      *db.DB
	JWT     *pauth.JWT
	Refresh *pauth.Refresh
	OTP     *pauth.OTP
	Google  pauth.GoogleVerifier
	Email   email.Sender
	Clock   clock.Clock
	Loc     *time.Location
	Cost    int
	Log     *slog.Logger
}

var phonePattern = regexp.MustCompile(`^\+?[0-9][0-9\s-]{5,}$`)

// maxPasswordBytes is bcrypt's limit.
const maxPasswordBytes = 72

// dummyHash lets a login for an unknown email cost the same as a real one.
var dummyHash, _ = pauth.HashPassword("not-a-real-password", 10)

func normEmail(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

func validPassword(p string) bool { return p != "" && len(p) <= maxPasswordBytes }

func (s *Service) fmtTime(t time.Time) string { return clock.FormatRFC3339(t, s.Loc) }

func pgText(s string) pgtype.Text { return pgtype.Text{String: s, Valid: s != ""} }

func appUser(u sqlc.User) AppUser {
	return AppUser{ID: u.ID, Name: u.Name, Email: u.Email.String, Role: u.Role}
}

// session signs the user in: an access token and a new refresh token.
func (s *Service) session(ctx context.Context, u sqlc.User) (*Session, error) {
	role, _ := pauth.ParseRole(u.Role)
	access, exp, err := s.JWT.Sign(u.ID, role)
	if err != nil {
		return nil, err
	}
	refresh, _, err := s.Refresh.Issue(ctx, u.ID)
	if err != nil {
		return nil, err
	}
	return &Session{AppUser: appUser(u), AccessToken: access, RefreshToken: refresh, ExpiresAt: s.fmtTime(exp)}, nil
}

func (s *Service) userByEmail(ctx context.Context, addr string) (sqlc.User, bool, error) {
	u, err := s.DB.Q().GetUserByEmail(ctx, pgText(addr))
	if errors.Is(err, pgx.ErrNoRows) {
		return sqlc.User{}, false, nil
	}
	return u, err == nil, err
}

// Login checks an email and password.
func (s *Service) Login(ctx context.Context, emailAddr, password string) (*Session, error) {
	u, found, err := s.userByEmail(ctx, normEmail(emailAddr))
	if err != nil {
		return nil, err
	}
	hash := dummyHash
	if found && u.PasswordHash.Valid {
		hash = u.PasswordHash.String
	}
	good := pauth.CheckPassword(hash, password)
	if !found || !good || u.Banned {
		return nil, WrongCredentials
	}
	return s.session(ctx, u)
}

// LoginGoogle verifies a Google ID token and signs in, creating the account the first time.
func (s *Service) LoginGoogle(ctx context.Context, idToken string) (*Session, error) {
	if strings.TrimSpace(idToken) == "" {
		return nil, GoogleTokenMissing
	}
	id, err := s.Google.Verify(ctx, idToken)
	if err != nil {
		if errors.Is(err, pauth.ErrInvalidGoogleToken) {
			return nil, GoogleTokenInvalid
		}
		return nil, err
	}
	addr := normEmail(id.Email)
	u, found, err := s.userByEmail(ctx, addr)
	if err != nil {
		return nil, err
	}
	if found && u.Banned {
		return nil, WrongCredentials
	}
	if !found {
		name := id.Name
		if name == "" {
			name = strings.Split(addr, "@")[0]
		}
		u, err = s.DB.Q().CreateUser(ctx, sqlc.CreateUserParams{
			ID: ids.User(), Email: pgText(addr), Name: name, PhotoUrl: pgText(id.Picture), MemberSince: s.Clock.Now(),
		})
		if err != nil {
			return nil, err
		}
	}
	return s.session(ctx, u)
}

// RequestSignUpOTP stores the pending sign-up and e-mails a code.
func (s *Service) RequestSignUpOTP(ctx context.Context, name, contact, password string) error {
	contact = normEmail(contact)
	if phonePattern.MatchString(contact) {
		return PhoneNotSupported
	}
	if a, err := mail.ParseAddress(contact); err != nil || a.Address != contact || !strings.Contains(contact, "@") {
		return ContactInvalid
	}
	if !validPassword(password) {
		return PasswordInvalid
	}
	if _, found, err := s.userByEmail(ctx, contact); err != nil {
		return err
	} else if found {
		return EmailTaken
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = strings.Split(contact, "@")[0]
	}
	hash, err := pauth.HashPassword(password, s.Cost)
	if err != nil {
		return err
	}
	if err := s.DB.Q().UpsertPendingSignup(ctx, sqlc.UpsertPendingSignupParams{
		Contact: contact, Name: name, PasswordHash: hash, CreatedAt: s.Clock.Now(),
	}); err != nil {
		return err
	}
	return s.sendCode(ctx, contact, pauth.PurposeSignup)
}

// sendCode issues a code and e-mails it. A code sent a moment ago is not sent again.
func (s *Service) sendCode(ctx context.Context, contact, purpose string) error {
	code, err := s.OTP.Issue(ctx, contact, purpose)
	if errors.Is(err, pauth.ErrResendTooSoon) {
		return nil
	}
	if err != nil {
		return err
	}
	return s.Email.SendOTP(ctx, contact, code, purpose)
}

// VerifySignUp checks the code and creates the reader account.
func (s *Service) VerifySignUp(ctx context.Context, contact, code string) (*Session, error) {
	contact = normEmail(contact)
	good, err := s.OTP.Verify(ctx, contact, pauth.PurposeSignup, code)
	if err != nil {
		return nil, err
	}
	if !good {
		return nil, WrongOTP
	}
	pending, err := s.DB.Q().GetPendingSignup(ctx, contact)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, WrongOTP
	}
	if err != nil {
		return nil, err
	}
	var u sqlc.User
	err = s.DB.WithTx(ctx, func(q *sqlc.Queries) error {
		var e error
		u, e = q.CreateUser(ctx, sqlc.CreateUserParams{
			ID: ids.User(), Email: pgText(contact), Name: pending.Name,
			PasswordHash: pgText(pending.PasswordHash), MemberSince: s.Clock.Now(),
		})
		if e != nil {
			return e
		}
		return q.DeletePendingSignup(ctx, contact)
	})
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return nil, EmailTaken
	}
	if err != nil {
		return nil, err
	}
	return s.session(ctx, u)
}

// RequestPasswordReset sends a code when the account exists. It answers the same either way.
func (s *Service) RequestPasswordReset(ctx context.Context, contact string) error {
	contact = normEmail(contact)
	_, found, err := s.userByEmail(ctx, contact)
	if err != nil || !found {
		return err
	}
	return s.sendCode(ctx, contact, pauth.PurposeReset)
}

// ResetPassword sets a new password after a right code and signs the account out everywhere.
func (s *Service) ResetPassword(ctx context.Context, contact, code, password string) error {
	contact = normEmail(contact)
	if !validPassword(password) {
		return PasswordInvalid
	}
	good, err := s.OTP.Verify(ctx, contact, pauth.PurposeReset, code)
	if err != nil {
		return err
	}
	u, found, err := s.userByEmail(ctx, contact)
	if err != nil {
		return err
	}
	if !good || !found {
		return WrongOTP
	}
	hash, err := pauth.HashPassword(password, s.Cost)
	if err != nil {
		return err
	}
	if err := s.DB.Q().SetUserPassword(ctx, sqlc.SetUserPasswordParams{ID: u.ID, PasswordHash: pgText(hash)}); err != nil {
		return err
	}
	return s.Refresh.RevokeAll(ctx, u.ID)
}

// Rotate trades a refresh token for a new access token and a new refresh token.
func (s *Service) Rotate(ctx context.Context, token string) (*Tokens, error) {
	next, uid, _, err := s.Refresh.Rotate(ctx, token)
	if err != nil {
		return nil, err
	}
	u, err := s.DB.Q().GetUserByID(ctx, uid)
	if err != nil {
		return nil, pauth.ErrInvalidRefresh
	}
	role, _ := pauth.ParseRole(u.Role)
	access, exp, err := s.JWT.Sign(uid, role)
	if err != nil {
		return nil, err
	}
	return &Tokens{AccessToken: access, RefreshToken: next, ExpiresAt: s.fmtTime(exp)}, nil
}

// Logout revokes one refresh token.
func (s *Service) Logout(ctx context.Context, token string) error {
	return s.Refresh.Revoke(ctx, token)
}

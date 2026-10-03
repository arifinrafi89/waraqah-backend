package auth

import "github.com/arifinrafi89/waraqah-backend/internal/platform/httpx"

// Refusal is a rule the request broke. The handler answers 200 null with the code in
// X-Waraqah-Error (docs/error-codes.md).
type Refusal string

// Error makes a Refusal usable as an error.
func (r Refusal) Error() string { return string(r) }

// Refusal codes for the auth endpoints; the strings live in httpx/errors.go.
const (
	WrongCredentials   = Refusal(httpx.ErrWrongCredentials)
	WrongOTP           = Refusal(httpx.ErrWrongOTP)
	PhoneNotSupported  = Refusal(httpx.ErrPhoneNotSupported)
	ContactInvalid     = Refusal(httpx.ErrContactInvalid)
	EmailTaken         = Refusal(httpx.ErrEmailTaken)
	PasswordInvalid    = Refusal(httpx.ErrPasswordInvalid)
	GoogleTokenMissing = Refusal(httpx.ErrGoogleTokenMissing)
	GoogleTokenInvalid = Refusal(httpx.ErrGoogleTokenInvalid)
)

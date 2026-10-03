// Package email sends the one-time codes for sign-up and password reset.
package email

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/mail"
	"strings"
	"time"
)

// Sender sends OTP emails. Resend, Brevo and Log implement it.
type Sender interface {
	SendOTP(ctx context.Context, to, code, purpose string) error
}

// New picks the sender by EMAIL_PROVIDER: resend, brevo or log. The log provider prints
// codes and is refused in production.
func New(provider, apiKey, from string, production bool, log *slog.Logger) (Sender, error) {
	switch strings.ToLower(provider) {
	case "log":
		if production {
			return nil, fmt.Errorf("EMAIL_PROVIDER=log is not allowed in production")
		}
		return &Log{Logger: log}, nil
	case "resend":
		return &Resend{APIKey: apiKey, From: from, URL: "https://api.resend.com/emails", Client: httpClient()}, nil
	case "brevo":
		return &Brevo{APIKey: apiKey, From: from, URL: "https://api.brevo.com/v3/smtp/email", Client: httpClient()}, nil
	}
	return nil, fmt.Errorf("unknown EMAIL_PROVIDER %q (resend, brevo, log)", provider)
}

func httpClient() *http.Client { return &http.Client{Timeout: 10 * time.Second} }

// subjectAndText is the message for a purpose.
func subjectAndText(code, purpose string) (subject, text string) {
	if purpose == "reset" {
		return "Your Waraqah password reset code",
			fmt.Sprintf("Your Waraqah password reset code is %s. It expires soon. If you did not ask for it, ignore this email.", code)
	}
	return "Your Waraqah verification code",
		fmt.Sprintf("Your Waraqah verification code is %s. It expires soon. If you did not ask for it, ignore this email.", code)
}

// parseFrom splits "Name <addr>" into its parts.
func parseFrom(from string) (name, addr string) {
	a, err := mail.ParseAddress(from)
	if err != nil {
		return "", from
	}
	return a.Name, a.Address
}

// Log prints the code to the server log. Development only.
type Log struct{ Logger *slog.Logger }

// SendOTP logs the code instead of sending it.
func (l *Log) SendOTP(_ context.Context, to, code, purpose string) error {
	l.Logger.Info("otp email (log provider, development only)", "to", to, "purpose", purpose, "code", code)
	return nil
}

// Recorder keeps what was sent, for tests.
type Recorder struct{ Sent []Message }

// Message is one recorded OTP email.
type Message struct{ To, Code, Purpose string }

// SendOTP records the message.
func (r *Recorder) SendOTP(_ context.Context, to, code, purpose string) error {
	r.Sent = append(r.Sent, Message{to, code, purpose})
	return nil
}

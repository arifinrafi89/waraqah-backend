package auth

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/dbtest"
)

func otpCfg() OTPConfig {
	return OTPConfig{TTL: 10 * time.Minute, MaxAttempts: 3, Resend: 60 * time.Second}
}

func TestOTPIssueVerify(t *testing.T) {
	d := dbtest.New(t)
	ctx := context.Background()
	o := NewOTP(d, otpCfg(), newClock())
	code, err := o.Issue(ctx, "a@example.test", PurposeSignup)
	if err != nil || !regexp.MustCompile(`^\d{6}$`).MatchString(code) {
		t.Fatalf("code %q err %v", code, err)
	}
	if ok, _ := o.Verify(ctx, "a@example.test", PurposeSignup, code); !ok {
		t.Fatal("right code refused")
	}
	if ok, _ := o.Verify(ctx, "a@example.test", PurposeSignup, code); ok {
		t.Fatal("a code must work once")
	}
	if ok, _ := o.Verify(ctx, "a@example.test", PurposeReset, code); ok {
		t.Fatal("a signup code must not reset a password")
	}
}

func TestOTPExpiryAttemptsAndResend(t *testing.T) {
	d := dbtest.New(t)
	ctx := context.Background()
	clk := newClock()
	o := NewOTP(d, otpCfg(), clk)

	code, _ := o.Issue(ctx, "b@example.test", PurposeReset)
	if _, err := o.Issue(ctx, "b@example.test", PurposeReset); !errors.Is(err, ErrResendTooSoon) {
		t.Fatalf("resend: %v", err)
	}
	for i := 0; i < 3; i++ {
		if ok, _ := o.Verify(ctx, "b@example.test", PurposeReset, "000000x"); ok {
			t.Fatal("wrong code accepted")
		}
	}
	// Three wrong tries killed the code, even the right one fails now.
	if ok, _ := o.Verify(ctx, "b@example.test", PurposeReset, code); ok {
		t.Fatal("code survived too many attempts")
	}

	clk.Advance(2 * time.Minute)
	code, err := o.Issue(ctx, "b@example.test", PurposeReset)
	if err != nil {
		t.Fatal(err)
	}
	clk.Advance(11 * time.Minute)
	if ok, _ := o.Verify(ctx, "b@example.test", PurposeReset, code); ok {
		t.Fatal("expired code accepted")
	}
}

func TestDevCodeOnlyInDevelopment(t *testing.T) {
	d := dbtest.New(t)
	ctx := context.Background()
	cfg := otpCfg()
	cfg.DevCode = "123456"
	cfg.DevEnabled = true
	if ok, _ := NewOTP(d, cfg, newClock()).Verify(ctx, "c@example.test", PurposeSignup, "123456"); !ok {
		t.Error("dev code refused in development")
	}
	cfg.DevEnabled = false
	if ok, _ := NewOTP(d, cfg, newClock()).Verify(ctx, "c@example.test", PurposeSignup, "123456"); ok {
		t.Error("dev code accepted outside development")
	}
}

// Package app wires the server: it builds shared services and hands interfaces to features.
package app

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/auth"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/clock"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/cloudinary"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/config"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/email"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/gemini"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/jobs"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/ratelimit"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/sse"
)

const googleCertsURL = "https://www.googleapis.com/oauth2/v3/certs"

// Limits are the rate limiters named in the config.
type Limits struct {
	Auth   *ratelimit.Limiter
	AI     *ratelimit.Limiter
	Report *ratelimit.Limiter
}

// Deps is everything the features need. Cross-feature interfaces are built here (BACKEND_PLAN.md section 8).
type Deps struct {
	Cfg      *config.Config
	Log      *slog.Logger
	DB       *db.DB
	Clock    clock.Clock
	Location *time.Location

	// Identity
	JWT     *auth.JWT
	Refresh *auth.Refresh
	OTP     *auth.OTP
	Google  auth.GoogleVerifier
	Auth    *auth.Middleware

	// Platform services, each behind an interface with a fake
	SSE    *sse.Broker
	Images cloudinary.Uploader
	Email  email.Sender
	Gemini gemini.Client // nil without GEMINI_API_KEY: callers keep their rule-based text
	Jobs   *jobs.Runner
	Limits Limits

	// Ready reports whether the database answers (/readyz). Nil means always ready.
	Ready func(ctx context.Context) error
}

// NewDeps builds every shared service from the config.
func NewDeps(cfg *config.Config, log *slog.Logger, database *db.DB) (*Deps, error) {
	loc, err := clock.Location(cfg.AppTimezone)
	if err != nil {
		return nil, fmt.Errorf("APP_TIMEZONE: %w", err)
	}
	clk := clock.Real{}
	d := &Deps{Cfg: cfg, Log: log, DB: database, Clock: clk, Location: loc, SSE: sse.NewBroker()}

	if d.JWT, err = auth.NewJWT(cfg.JWTSecret, cfg.JWTIssuer, cfg.JWTAccessTTL, clk); err != nil {
		return nil, err
	}
	d.Refresh = auth.NewRefresh(database, cfg.JWTRefreshTTL, clk)
	d.OTP = auth.NewOTP(database, auth.OTPConfig{
		TTL: cfg.OTPTTL, MaxAttempts: cfg.OTPMaxAttempts,
		Resend:  time.Duration(cfg.OTPResendSeconds) * time.Second,
		DevCode: cfg.OTPDevCode, DevEnabled: cfg.IsDevelopment(),
	}, clk)
	d.Google = auth.NewGoogleVerifier(googleCertsURL, cfg.GoogleOAuthClientIDs, clk)
	d.Auth = auth.NewMiddleware(d.JWT, auth.DBUsers{DB: database})

	if cfg.CloudinaryFake {
		d.Images = cloudinary.NewFake(cfg.CloudinaryFolder)
	} else {
		d.Images = cloudinary.NewReal(cfg.CloudinaryCloudName, cfg.CloudinaryAPIKey, cfg.CloudinaryAPISecret, cfg.CloudinaryFolder)
	}
	if d.Email, err = email.New(cfg.EmailProvider, cfg.EmailAPIKey, cfg.EmailFrom, cfg.IsProduction(), log); err != nil {
		return nil, err
	}
	d.Gemini = gemini.New(cfg.GeminiAPIKey, cfg.GeminiModel, cfg.GeminiTimeout)
	d.Jobs = jobs.New(cfg.JobsTick, log)
	d.Limits = Limits{
		Auth:   ratelimit.New(cfg.AuthRatePerMin, time.Minute),
		AI:     ratelimit.New(cfg.AIRatePerMin, time.Minute),
		Report: ratelimit.New(cfg.ReportRatePerHour, time.Hour),
	}
	if database != nil {
		d.Ready = database.Ping
	}
	return d, nil
}

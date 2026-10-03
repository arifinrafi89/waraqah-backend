// Package app wires the server: it builds shared services and hands interfaces to features.
package app

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/arifinrafi89/waraqah-backend/internal/feature/alerts"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/cart"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/catalog"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/catalogadmin"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/checkout"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/deals"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/donate"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/loyalty"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/notifications"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/orders"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/profile"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/wallet"
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

	// Features other features talk to, through interfaces (BACKEND_PLAN.md section 8).
	Profile       *profile.Service
	Notifications *notifications.Service // also the notifications.Sender everyone uses
	Catalog       *catalog.Store         // also the catalog.Books everyone uses
	Sweeper       catalogadmin.Sweeper   // alerts.Sweeper: runs after price and stock changes
	Deals         *deals.Service
	Cart          *cart.Service
	Alerts        *alerts.Service
	Wallet        *wallet.Service  // also the wallet.Ledger checkout and orders use
	Points        *loyalty.Service // also the loyalty.Ledger
	Orders        *orders.Service
	Checkout      *checkout.Service
	Donate        *donate.Service

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
	d.Profile = &profile.Service{
		DB: database, MaxImageBytes: cfg.MaxImageMB << 20, RevokeTokens: d.Refresh.RevokeAll,
		Now: clk.Now, Log: log,
	}
	d.Notifications = &notifications.Service{DB: database, SSE: d.SSE, Prefs: d.Profile, Clock: clk, Loc: loc, Log: log}
	d.Catalog = &catalog.Store{DB: database, Loc: loc}
	d.Deals = &deals.Service{DB: database, Books: d.Catalog, Clock: clk, Loc: loc, Log: log}
	d.Cart = &cart.Service{DB: database, Books: d.Catalog, Deals: d.Deals, Used: cart.NoUsedStock{}, Clock: clk, Log: log}
	d.Alerts = &alerts.Service{DB: database, Books: d.Catalog, Notify: d.Notifications, Clock: clk, Log: log}
	d.Sweeper = d.Alerts
	d.Wallet = &wallet.Service{DB: database, Clock: clk, Loc: loc, Log: log}
	d.Points = &loyalty.Service{DB: database, Clock: clk, Loc: loc, Log: log}
	d.Orders = &orders.Service{DB: database, Wallet: d.Wallet, Points: d.Points, Notify: d.Notifications, Cart: d.Cart, Stock: d.Catalog,
		Clock: clk, Loc: loc, MaxImageBytes: cfg.MaxImageMB << 20, Log: log}
	d.Donate = &donate.Service{DB: database, Books: d.Catalog, Clock: clk, Log: log}
	d.Checkout = &checkout.Service{DB: database, Cart: d.Cart, Addresses: d.Profile, Wallet: d.Wallet, Points: d.Points, Stock: d.Catalog,
		Clock: clk, Loc: loc, Log: log}
	if database != nil {
		d.Ready = database.Ping
	}
	return d, nil
}

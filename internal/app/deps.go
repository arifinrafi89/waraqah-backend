// Package app wires the server: it builds shared services and hands interfaces to features.
package app

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/arifinrafi89/waraqah-backend/internal/feature/alerts"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/assistant"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/bites"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/bookrequest"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/cart"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/catalog"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/catalogadmin"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/checkout"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/dashboard"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/deals"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/donate"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/handledsale"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/inbox"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/loyalty"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/moderation"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/notifications"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/orders"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/p2p"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/profile"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/readers"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/report"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/reviews"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/sellback"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/shelves"
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
	P2P           *p2p.Service        // also listings.Status
	Report        *report.Service     // also blocks.Checker
	Moderation    *moderation.Service // also moderation.Bans
	Inbox         *inbox.Service
	BookRequests  *bookrequest.Service
	Sales         *handledsale.Service
	SellBack      *sellback.Service // also the Certified Used stock of the cart and the catalog
	Bites         *bites.Service
	Reviews       *reviews.Service
	Readers       *readers.Service // also the follows Bites read
	Shelves       *shelves.Service
	CatalogAdmin  *catalogadmin.Service
	SearchLog     *dashboard.SearchLog // search.Log: catalog searches counted for the dashboard
	Dashboard     *dashboard.Service
	Assistant     *assistant.Service

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
	d.P2P = &p2p.Service{DB: database, Images: d.Images, MaxImageBytes: cfg.MaxImageMB << 20, Catalog: d.Catalog, Clock: clk, Loc: loc, Log: log}
	d.Moderation = &moderation.Service{DB: database, Shop: d.P2P, Notify: d.Notifications, Clock: clk, Loc: loc, Log: log}
	d.P2P.Bans = d.Moderation
	d.Report = &report.Service{DB: database, Shop: d.P2P, Clock: clk, Loc: loc, Log: log}
	d.Inbox = &inbox.Service{DB: database, Market: d.P2P, Blocks: d.Report, SSE: d.SSE, Clock: clk, Loc: loc, Log: log, DemoMode: cfg.DemoMode, BotDelay: cfg.DemoBotDelay}
	d.Moderation.Register("message", inbox.Messages{S: d.Inbox})
	d.BookRequests = &bookrequest.Service{DB: database, Shop: d.P2P, Notify: d.Notifications, Clock: clk, Loc: loc, Log: log}
	d.Profile.Hooks = append(d.Profile.Hooks, d.P2P.OnAccountDeleted)
	d.Sales = &handledsale.Service{DB: database, Market: d.P2P, Wallet: d.Wallet, Notify: d.Notifications, Audit: d.Moderation, SSE: d.SSE,
		Clock: clk, Loc: loc, Log: log, MaxImageBytes: cfg.MaxImageMB << 20, DemoMode: cfg.DemoMode, BotDelay: cfg.DemoBotDelay}
	d.SellBack = &sellback.Service{DB: database, Books: d.Catalog, Wallet: d.Wallet, Notify: d.Notifications, Clock: clk, Loc: loc, Log: log,
		PickupDelay: cfg.CourierPickupDelay}
	d.Cart.Used = d.SellBack
	d.Jobs.Register("demo sellers send", d.Sales.SendDemoSales)
	d.Jobs.Register("courier pickups", d.SellBack.PickUp)
	d.Bites = &bites.Service{DB: database, Books: d.Catalog, Blocks: d.Report, Bans: d.Moderation, Notify: d.Notifications,
		Clock: clk, Loc: loc, Log: log}
	d.Readers = &readers.Service{DB: database, Prefs: d.Profile, Blocks: d.Report, Bites: d.Bites, Listings: d.P2P, Notify: d.Notifications,
		Clock: clk, Loc: loc, Log: log}
	d.Bites.Follows = d.Readers
	d.Reviews = &reviews.Service{DB: database, Catalog: d.Catalog, Delivered: d.Orders, Bans: d.Moderation, Clock: clk, Loc: loc, Log: log}
	d.Shelves = &shelves.Service{DB: database, Catalog: d.Catalog, Delivered: d.Orders, Clock: clk, Loc: loc, Log: log}
	d.Moderation.Register("bite", bites.Bites{S: d.Bites})
	d.Moderation.Register("comment", bites.Comments{S: d.Bites})
	d.Moderation.Register("review", reviews.Reviews{S: d.Reviews})
	d.Donate = &donate.Service{DB: database, Books: d.Catalog, Clock: clk, Log: log}
	d.Checkout = &checkout.Service{DB: database, Cart: d.Cart, Addresses: d.Profile, Wallet: d.Wallet, Points: d.Points, Stock: d.Catalog,
		Used: d.SellBack, Clock: clk, Loc: loc, Log: log}
	d.CatalogAdmin = &catalogadmin.Service{DB: database, Cache: d.Catalog, Sweeper: d.Sweeper, Clock: clk, Loc: loc, Log: log}
	d.SearchLog = &dashboard.SearchLog{DB: database, Clock: clk, Loc: loc, Log: log}
	d.Dashboard = &dashboard.Service{Orders: d.Orders, Listings: d.P2P, Reports: d.Moderation, Disputes: d.Sales, SellBacks: d.SellBack,
		Stock: d.CatalogAdmin, Requests: d.BookRequests, Searches: d.SearchLog, Clock: clk, Loc: loc, Log: log}
	d.Assistant = &assistant.Service{Catalog: d.Catalog, Gemini: d.Gemini, Log: log}
	if database != nil {
		d.Ready = database.Ping
	}
	return d, nil
}

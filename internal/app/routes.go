package app

import (
	"net/http"
	"strings"

	"github.com/arifinrafi89/waraqah-backend/internal/feature/alerts"
	authfeature "github.com/arifinrafi89/waraqah-backend/internal/feature/auth"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/cart"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/catalog"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/catalogadmin"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/checkout"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/deals"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/donate"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/home"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/loyalty"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/notifications"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/orders"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/profile"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/scan"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/wallet"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/wishlist"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/httpx"
	seeddata "github.com/arifinrafi89/waraqah-backend/seed"
)

// Routes builds the whole HTTP handler: the mux with every feature mounted, wrapped in middleware.
func Routes(d *Deps) http.Handler {
	mux := Mux(d)
	maxBody := int64(d.Cfg.MaxRequestBodyMB) << 20
	return httpx.Chain(mux,
		httpx.Recover(d.Log),
		httpx.RequestID(d.Log),
		httpx.AccessLog(),
		httpx.CORS(d.Cfg.CORSAllowedOrigins),
		httpx.BodyLimit(maxBody),
	)
}

// Mux registers every route without the middleware. The route coverage test asks it which
// endpoints exist.
func Mux(d *Deps) *http.ServeMux {
	mux := http.NewServeMux()
	api := httpx.Router{Mux: mux, Prefix: strings.TrimRight(d.Cfg.APIBasePath, "/")}

	mux.HandleFunc("GET /healthz", d.healthz)
	mux.HandleFunc("GET /readyz", d.readyz)
	mux.HandleFunc("GET /version", d.version)

	mountFeatures(api, d)

	// Anything else is an unknown path: the error shape, not Go's plain text 404.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		httpx.Error(w, r, http.StatusNotFound, httpx.CodeNotFound, "no such endpoint")
	})
	return mux
}

// mountFeatures is where each feature adds its routes, one line per feature.
func mountFeatures(api httpx.Router, d *Deps) {
	authfeature.NewHandler(authService(d)).Routes(api, d.Limits.Auth.ByIP())
	profile.NewHandler(d.Profile, seeddata.Geo).Routes(api, d.Auth)
	notifications.NewHandler(d.Notifications).Routes(api, d.Auth)
	home.NewHandler(&home.Service{DB: d.DB, Clock: d.Clock, Loc: d.Location}, d.Log).Routes(api, d.Auth)
	catalogadmin.NewHandler(&catalogadmin.Service{DB: d.DB, Cache: d.Catalog, Sweeper: d.Sweeper, Clock: d.Clock, Loc: d.Location, Log: d.Log}).Routes(api, d.Auth)
	(&scan.Handler{Books: d.Catalog}).Routes(api, d.Auth)
	deals.Handler{S: d.Deals}.Routes(api, d.Auth)
	cart.NewHandler(d.Cart).Routes(api, d.Auth)
	(&wishlist.Handler{S: &wishlist.Service{DB: d.DB, Books: d.Catalog, Clock: d.Clock, Log: d.Log}}).Routes(api, d.Auth)
	(&alerts.Handler{S: d.Alerts}).Routes(api, d.Auth)
	checkout.NewHandler(d.Checkout).Routes(api, d.Auth)
	donate.NewHandler(d.Donate).Routes(api, d.Auth)
	orders.NewHandler(d.Orders).Routes(api, d.Auth)
	wallet.Handler{S: d.Wallet}.Routes(api, d.Auth)
	loyalty.Handler{S: d.Points}.Routes(api, d.Auth)
	catalog.NewHandler(&catalog.Service{Store: d.Catalog, Used: catalog.NoUsedStock{}, Clock: d.Clock, Loc: d.Location, Log: d.Log}).Routes(api, d.Auth)
}

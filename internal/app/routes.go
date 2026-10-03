package app

import (
	"net/http"
	"strings"

	authfeature "github.com/arifinrafi89/waraqah-backend/internal/feature/auth"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/catalog"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/notifications"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/profile"
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
	catalog.NewHandler(&catalog.Service{Store: d.Catalog, Used: catalog.NoUsedStock{}, Clock: d.Clock, Loc: d.Location, Log: d.Log}).Routes(api, d.Auth)
}

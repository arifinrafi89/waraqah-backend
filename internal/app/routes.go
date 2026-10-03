package app

import (
	"net/http"
	"strings"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/httpx"
)

// Routes builds the whole HTTP handler: the mux with every feature mounted, wrapped in middleware.
func Routes(d *Deps) http.Handler {
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

	maxBody := int64(d.Cfg.MaxRequestBodyMB) << 20
	return httpx.Chain(mux,
		httpx.Recover(d.Log),
		httpx.RequestID(d.Log),
		httpx.AccessLog(),
		httpx.CORS(d.Cfg.CORSAllowedOrigins),
		httpx.BodyLimit(maxBody),
	)
}

// mountFeatures is where each feature adds its routes, one line per feature.
func mountFeatures(api httpx.Router, d *Deps) {
	_ = api
	_ = d
}

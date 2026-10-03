package app

import (
	"context"
	"net/http"
	"time"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/buildinfo"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/httpx"
)

func (d *Deps) healthz(w http.ResponseWriter, _ *http.Request) {
	httpx.JSON(w, map[string]bool{"ok": true})
}

func (d *Deps) readyz(w http.ResponseWriter, r *http.Request) {
	if d.Ready != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		if err := d.Ready(ctx); err != nil {
			d.Log.Error("readiness check failed", "error", err)
			httpx.Error(w, r, http.StatusServiceUnavailable, "unavailable", "database is not reachable")
			return
		}
	}
	httpx.JSON(w, map[string]bool{"ok": true})
}

func (d *Deps) version(w http.ResponseWriter, _ *http.Request) {
	httpx.JSON(w, map[string]string{
		"commit":   buildinfo.Commit,
		"builtAt":  buildinfo.BuiltAt,
		"contract": buildinfo.Contract,
	})
}

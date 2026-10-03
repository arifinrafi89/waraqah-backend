package dashboard

import (
	"net/http"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/auth"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/httpx"
)

// Handler is the HTTP side of the dashboard.
type Handler struct{ S *Service }

// NewHandler builds the handler.
func NewHandler(s *Service) *Handler { return &Handler{S: s} }

// Dashboard is DashboardFakeApi.dashboard: any staff role.
func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	v, err := h.S.Get(r.Context())
	if err != nil {
		h.S.Log.Error("dashboard failed", "error", err)
		httpx.Error(w, r, http.StatusInternalServerError, httpx.CodeInternal, "something went wrong")
		return
	}
	httpx.JSON(w, v)
}

// Routes registers the dashboard.
func (h *Handler) Routes(r httpx.Router, a *auth.Middleware) {
	r.Handle("GET /admin/dashboard", a.Staff(http.HandlerFunc(h.Dashboard))) // DashboardFakeApi.dashboard
}

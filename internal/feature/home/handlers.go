package home

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/auth"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/httpx"
)

// Handler is the HTTP side of the Home endpoints.
type Handler struct {
	S   *Service
	Log *slog.Logger
}

// NewHandler builds the handler.
func NewHandler(s *Service, log *slog.Logger) *Handler { return &Handler{S: s, Log: log} }

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	h.Log.Error("home endpoint failed", "error", err)
	httpx.Error(w, r, http.StatusInternalServerError, httpx.CodeInternal, "something went wrong")
}

// day reads the optional ?date=yyyy-mm-dd (default: today in the app timezone).
func (h *Handler) day(r *http.Request) time.Time {
	if v := httpx.Query(r, "date"); v != "" {
		if t, err := time.ParseInLocation("2006-01-02", v, h.S.Loc); err == nil {
			return t
		}
	}
	return h.S.Today()
}

// Banners is HomeFakeApi.banners.
func (h *Handler) Banners(w http.ResponseWriter, r *http.Request) {
	list, err := h.S.Banners(r.Context(), h.day(r))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, list)
}

// Season is HomeFakeApi.season: the hero card of the Season on, or null.
func (h *Handler) Season(w http.ResponseWriter, r *http.Request) {
	info, err := h.S.Season(r.Context(), h.day(r))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	if info == nil {
		httpx.Null(w)
		return
	}
	httpx.JSON(w, info)
}

// AyahOfTheDay is AyahFakeApi.ayahOfTheDay.
func (h *Handler) AyahOfTheDay(w http.ResponseWriter, r *http.Request) {
	a, err := h.S.AyahOfTheDay(r.Context(), h.S.Today())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	if a == nil {
		httpx.Null(w)
		return
	}
	httpx.JSON(w, a)
}

// Routes registers the Home endpoints (all public).
func (h *Handler) Routes(r httpx.Router, a *auth.Middleware) {
	r.Handle("GET /home/banners", a.Public(http.HandlerFunc(h.Banners)))                 // HomeFakeApi.banners
	r.Handle("GET /home/season", a.Public(http.HandlerFunc(h.Season)))                   // HomeFakeApi.season
	r.Handle("GET /islamic/ayah-of-the-day", a.Public(http.HandlerFunc(h.AyahOfTheDay))) // AyahFakeApi.ayahOfTheDay
}

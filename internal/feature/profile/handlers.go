package profile

import (
	"errors"
	"net/http"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/auth"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/httpx"
)

// Handler is the HTTP side of the profile endpoints.
type Handler struct {
	S       *Service
	GeoJSON []byte // the /geo answer, parsed once and kept (seed/geo.json)
}

// NewHandler builds the handler around a service and the geography JSON.
func NewHandler(s *Service, geo []byte) *Handler { return &Handler{S: s, GeoJSON: geo} }

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	var ref Refusal
	if errors.As(err, &ref) {
		httpx.Refuse(w, r, string(ref))
		return
	}
	h.S.Log.Error("profile endpoint failed", "error", err)
	httpx.Error(w, r, http.StatusInternalServerError, httpx.CodeInternal, "something went wrong")
}

// Profile is ProfileFakeApi.profile. A guest gets the empty profile.
func (h *Handler) Profile(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFrom(r.Context())
	if !ok {
		httpx.JSON(w, Details{})
		return
	}
	d, err := h.S.Profile(r.Context(), u.ID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, d)
}

// SaveProfile is ProfileFakeApi.saveProfile.
func (h *Handler) SaveProfile(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	var in Details
	if !httpx.Decode(w, r, &in) {
		return
	}
	d, err := h.S.SaveProfile(r.Context(), u.ID, in)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, d)
}

// Prefs is ProfileFakeApi.prefs. A guest gets the defaults.
func (h *Handler) Prefs(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFrom(r.Context())
	if !ok {
		httpx.JSON(w, DefaultPrefs())
		return
	}
	p, err := h.S.Prefs(r.Context(), u.ID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, p)
}

// SavePrefs is ProfileFakeApi.savePrefs.
func (h *Handler) SavePrefs(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	in := DefaultPrefs()
	if !httpx.Decode(w, r, &in) {
		return
	}
	p, err := h.S.SavePrefs(r.Context(), u.ID, in)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, p)
}

// Geo is ProfileFakeApi.geo: every division, its districts and their upazilas.
func (h *Handler) Geo(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	_, _ = w.Write(h.GeoJSON)
}

// DeleteAccount is ProfileFakeApi.deleteAccount (POST /auth/delete).
func (h *Handler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	if err := h.S.DeleteAccount(r.Context(), u.ID); err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, ok{true})
}

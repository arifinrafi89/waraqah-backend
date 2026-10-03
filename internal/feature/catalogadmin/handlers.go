package catalogadmin

import (
	"errors"
	"net/http"

	"github.com/arifinrafi89/waraqah-backend/internal/feature/home"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/httpx"
)

// Handler is the HTTP side of Admin → Catalog. Every route needs the catalog permission.
type Handler struct{ S *Service }

// NewHandler builds the handler.
func NewHandler(s *Service) *Handler { return &Handler{S: s} }

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	var ref Refusal
	if errors.As(err, &ref) {
		httpx.Refuse(w, r, string(ref))
		return
	}
	h.S.Log.Error("catalog admin endpoint failed", "error", err)
	httpx.Error(w, r, http.StatusInternalServerError, httpx.CodeInternal, "something went wrong")
}

// reply writes v, or the failure.
func (h *Handler) reply(w http.ResponseWriter, r *http.Request, v any, err error) {
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, v)
}

type idBody struct {
	ID string `json:"id"`
}

// SaveBook is CatalogAdminFakeApi.saveBook.
func (h *Handler) SaveBook(w http.ResponseWriter, r *http.Request) {
	var d BookDraft
	if !httpx.Decode(w, r, &d) {
		return
	}
	b, err := h.S.SaveBook(r.Context(), d)
	h.reply(w, r, b, err)
}

// HideBook is CatalogAdminFakeApi.hideBook.
func (h *Handler) HideBook(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ID     string `json:"id"`
		Hidden bool   `json:"hidden"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	b, err := h.S.HideBook(r.Context(), in.ID, in.Hidden)
	h.reply(w, r, b, err)
}

// Records returns the handler of GET /admin/catalog/<kind>.
func (h *Handler) Records(kind string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, err := h.S.Records(r.Context(), kind)
		h.reply(w, r, list, err)
	}
}

// SaveRecord returns the handler of POST /admin/catalog/<kind>/save.
func (h *Handler) SaveRecord(kind string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in RecordInput
		if !httpx.Decode(w, r, &in) {
			return
		}
		rec, err := h.S.SaveRecord(r.Context(), kind, in)
		h.reply(w, r, rec, err)
	}
}

// DeleteRecord returns the handler of POST /admin/catalog/<kind>/delete.
func (h *Handler) DeleteRecord(kind string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in idBody
		if !httpx.Decode(w, r, &in) {
			return
		}
		err := h.S.DeleteRecord(r.Context(), kind, in.ID)
		h.reply(w, r, ListID(in), err)
	}
}

// Banners is CatalogAdminFakeApi.banners.
func (h *Handler) Banners(w http.ResponseWriter, r *http.Request) {
	list, err := h.S.Banners(r.Context())
	h.reply(w, r, list, err)
}

// SaveBanner is CatalogAdminFakeApi.saveBanner.
func (h *Handler) SaveBanner(w http.ResponseWriter, r *http.Request) {
	var b home.Banner
	if !httpx.Decode(w, r, &b) {
		return
	}
	list, err := h.S.SaveBanner(r.Context(), b)
	h.reply(w, r, list, err)
}

// DeleteBanner is CatalogAdminFakeApi.deleteBanner.
func (h *Handler) DeleteBanner(w http.ResponseWriter, r *http.Request) {
	var in idBody
	if !httpx.Decode(w, r, &in) {
		return
	}
	list, err := h.S.DeleteBanner(r.Context(), in.ID)
	h.reply(w, r, list, err)
}

// MoveBanner is CatalogAdminFakeApi.moveBanner.
func (h *Handler) MoveBanner(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ID string `json:"id"`
		By int    `json:"by"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	list, err := h.S.MoveBanner(r.Context(), in.ID, in.By)
	h.reply(w, r, list, err)
}

type seasonBody struct {
	Season *string `json:"season"`
}

// Season is CatalogAdminFakeApi.season: the Season Staff forced on Home, null = automatic.
func (h *Handler) Season(w http.ResponseWriter, r *http.Request) {
	s, err := h.S.SeasonOverride(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, seasonBody{Season: nullable(s)})
}

// SaveSeason is CatalogAdminFakeApi.season + /save.
func (h *Handler) SaveSeason(w http.ResponseWriter, r *http.Request) {
	var in seasonBody
	if !httpx.Decode(w, r, &in) {
		return
	}
	name := ""
	if in.Season != nil {
		name = *in.Season
	}
	if err := h.S.SetSeasonOverride(r.Context(), name); err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, seasonBody{Season: nullable(name)})
}

func nullable(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

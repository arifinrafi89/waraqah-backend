package readers

import (
	"errors"
	"net/http"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/auth"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/httpx"
)

// Handler is the HTTP side of the reader endpoints.
type Handler struct{ S *Service }

// NewHandler builds the handler.
func NewHandler(s *Service) *Handler { return &Handler{S: s} }

func (h *Handler) answer(w http.ResponseWriter, r *http.Request, v *Reader, err error) {
	var ref Refusal
	switch {
	case errors.As(err, &ref):
		httpx.Refuse(w, r, string(ref))
	case err != nil:
		h.S.Log.Error("readers endpoint failed", "error", err)
		httpx.Error(w, r, http.StatusInternalServerError, httpx.CodeInternal, "something went wrong")
	case v == nil:
		httpx.Null(w)
	default:
		httpx.JSON(w, v)
	}
}

func viewer(r *http.Request) string {
	u, _ := auth.UserFrom(r.Context())
	return u.ID
}

// Detail is ReaderFakeApi.detail: `?id=`, one reader's page, or null.
func (h *Handler) Detail(w http.ResponseWriter, r *http.Request) {
	v, err := h.S.Page(r.Context(), viewer(r), httpx.Query(r, "id"))
	h.answer(w, r, v, err)
}

// Follow is ReaderFakeApi.follow: body `{id, follow}`; answers the reader's page.
func (h *Handler) Follow(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ID     string `json:"id"`
		Follow bool   `json:"follow"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	v, err := h.S.Follow(r.Context(), viewer(r), in.ID, in.Follow)
	h.answer(w, r, v, err)
}

// Routes registers the reader endpoints.
func (h *Handler) Routes(r httpx.Router, a *auth.Middleware) {
	r.Handle("GET /readers/detail", a.Public(http.HandlerFunc(h.Detail))) // ReaderFakeApi.detail
	r.Handle("POST /readers/follow", a.Me(http.HandlerFunc(h.Follow)))    // ReaderFakeApi.follow
}

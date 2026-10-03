package bookrequest

import (
	"errors"
	"net/http"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/auth"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/httpx"
)

// Handler is the HTTP side of the book request endpoints.
type Handler struct{ S *Service }

// NewHandler builds the handler.
func NewHandler(s *Service) *Handler { return &Handler{S: s} }

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	var ref Refusal
	if errors.As(err, &ref) {
		httpx.Refuse(w, r, string(ref))
		return
	}
	h.S.Log.Error("book request endpoint failed", "error", err)
	httpx.Error(w, r, http.StatusInternalServerError, httpx.CodeInternal, "something went wrong")
}

func viewer(r *http.Request) string {
	u, _ := auth.UserFrom(r.Context())
	return u.ID
}

func answer[T any](h *Handler, w http.ResponseWriter, r *http.Request, v T, err error) {
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, v)
}

// Create is BookRequestFakeApi.create: body `{title, author?, bookId?, maxPriceBdt?, note?}`.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var in Draft
	if !httpx.Decode(w, r, &in) {
		return
	}
	v, err := h.S.Create(r.Context(), viewer(r), in)
	answer(h, w, r, v, err)
}

// Mine is BookRequestFakeApi.mine.
func (h *Handler) Mine(w http.ResponseWriter, r *http.Request) {
	v, err := h.S.Mine(r.Context(), viewer(r))
	answer(h, w, r, v, err)
}

// Close is BookRequestFakeApi.close: body `{id}`; the reader requests.
func (h *Handler) Close(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ID string `json:"id"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	v, err := h.S.Close(r.Context(), viewer(r), in.ID)
	answer(h, w, r, v, err)
}

// Wanted is BookRequestFakeApi.wanted.
func (h *Handler) Wanted(w http.ResponseWriter, r *http.Request) {
	v, err := h.S.Wanted(r.Context(), viewer(r))
	answer(h, w, r, v, err)
}

// Demand is BookRequestFakeApi.demand: open requests per title, for staff.
func (h *Handler) Demand(w http.ResponseWriter, r *http.Request) {
	v, err := h.S.Demand(r.Context())
	answer(h, w, r, v, err)
}

// Routes registers the book request endpoints.
func (h *Handler) Routes(r httpx.Router, a *auth.Middleware) {
	r.Handle("POST /requests", a.Me(http.HandlerFunc(h.Create)))          // BookRequestFakeApi.create
	r.Handle("GET /requests/mine", a.Me(http.HandlerFunc(h.Mine)))        // BookRequestFakeApi.mine
	r.Handle("POST /requests/close", a.Me(http.HandlerFunc(h.Close)))     // BookRequestFakeApi.close
	r.Handle("GET /requests/wanted", a.Me(http.HandlerFunc(h.Wanted)))    // BookRequestFakeApi.wanted
	r.Handle("GET /requests/demand", a.Staff(http.HandlerFunc(h.Demand))) // BookRequestFakeApi.demand
}

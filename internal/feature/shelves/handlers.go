package shelves

import (
	"errors"
	"net/http"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/auth"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/httpx"
)

// Handler is the HTTP side of the shelves and reading stats.
type Handler struct{ S *Service }

// NewHandler builds the handler.
func NewHandler(s *Service) *Handler { return &Handler{S: s} }

func answer[T any](h *Handler, w http.ResponseWriter, r *http.Request, v T, err error) {
	var ref Refusal
	switch {
	case errors.As(err, &ref):
		httpx.Refuse(w, r, string(ref))
	case err != nil:
		h.S.Log.Error("shelves endpoint failed", "error", err)
		httpx.Error(w, r, http.StatusInternalServerError, httpx.CodeInternal, "something went wrong")
	default:
		httpx.JSON(w, v)
	}
}

func userID(r *http.Request) string {
	u, _ := auth.UserFrom(r.Context())
	return u.ID
}

// Mine is ShelfFakeApi.mine: the reader's shelves, newest first.
func (h *Handler) Mine(w http.ResponseWriter, r *http.Request) {
	v, err := h.S.Mine(r.Context(), userID(r))
	answer(h, w, r, v, err)
}

// Move is ShelfFakeApi.move: body `{bookId, shelf}`; no shelf takes the Book off.
func (h *Handler) Move(w http.ResponseWriter, r *http.Request) {
	var in struct {
		BookID string `json:"bookId"`
		Shelf  string `json:"shelf"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	v, err := h.S.Move(r.Context(), userID(r), in.BookID, in.Shelf)
	answer(h, w, r, v, err)
}

// Progress is ShelfFakeApi.progress: body `{bookId, percent, pagesRead?, totalPages?}`.
func (h *Handler) Progress(w http.ResponseWriter, r *http.Request) {
	in := Update{Percent: -1}
	if !httpx.Decode(w, r, &in) {
		return
	}
	v, err := h.S.Progress(r.Context(), userID(r), in)
	answer(h, w, r, v, err)
}

// Stats is ShelfFakeApi.stats: the reader's year in books.
func (h *Handler) Stats(w http.ResponseWriter, r *http.Request) {
	v, err := h.S.Stats(r.Context(), userID(r))
	answer(h, w, r, v, err)
}

// Goal is ShelfFakeApi.goal: body `{goal}` (1 to 365); answers the stats.
func (h *Handler) Goal(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Goal int `json:"goal"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	v, err := h.S.SetGoal(r.Context(), userID(r), in.Goal)
	answer(h, w, r, v, err)
}

// Routes registers the shelves endpoints.
func (h *Handler) Routes(r httpx.Router, a *auth.Middleware) {
	me := func(f http.HandlerFunc) http.Handler { return a.Me(f) }
	r.Handle("GET /shelves", me(h.Mine))               // ShelfFakeApi.mine
	r.Handle("POST /shelves/move", me(h.Move))         // ShelfFakeApi.move
	r.Handle("POST /shelves/progress", me(h.Progress)) // ShelfFakeApi.progress
	r.Handle("GET /reading/stats", me(h.Stats))        // ShelfFakeApi.stats
	r.Handle("POST /reading/goal", me(h.Goal))         // ShelfFakeApi.goal
}

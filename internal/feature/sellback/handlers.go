package sellback

import (
	"errors"
	"net/http"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/auth"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/httpx"
)

// Handler is the HTTP side of the Sell Back endpoints.
type Handler struct{ S *Service }

// NewHandler builds the handler.
func NewHandler(s *Service) *Handler { return &Handler{S: s} }

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	var ref Refusal
	if errors.As(err, &ref) {
		httpx.Refuse(w, r, string(ref))
		return
	}
	h.S.Log.Error("sell back endpoint failed", "error", err)
	httpx.Error(w, r, http.StatusInternalServerError, httpx.CodeInternal, "something went wrong")
}

// answer writes v, null when v is a nil pointer, or the refusal.
func answer[T any](h *Handler, w http.ResponseWriter, r *http.Request, v *T, err error) {
	switch {
	case err != nil:
		h.fail(w, r, err)
	case v == nil:
		httpx.Null(w)
	default:
		httpx.JSON(w, v)
	}
}

func userID(r *http.Request) string {
	u, _ := auth.UserFrom(r.Context())
	return u.ID
}

// Books is SellBackFakeApi.books: `?q=`, catalog Books Waraqah buys back.
func (h *Handler) Books(w http.ResponseWriter, r *http.Request) {
	v, err := h.S.SearchBooks(r.Context(), httpx.Query(r, "q"))
	answer(h, w, r, &v, err)
}

// Book is SellBackFakeApi.book: `?id=bk-zero`, or null.
func (h *Handler) Book(w http.ResponseWriter, r *http.Request) {
	v, err := h.S.FindBook(r.Context(), httpx.Query(r, "id"))
	answer(h, w, r, v, err)
}

// Create is SellBackFakeApi.create: body `{bookId, condition, flags, pickupAddress}`.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var in Draft
	if !httpx.Decode(w, r, &in) {
		return
	}
	v, err := h.S.Create(r.Context(), userID(r), in)
	answer(h, w, r, v, err)
}

// Mine is SellBackFakeApi.mine: the reader's Sell Backs, newest first.
func (h *Handler) Mine(w http.ResponseWriter, r *http.Request) {
	v, err := h.S.Mine(r.Context(), userID(r))
	answer(h, w, r, &v, err)
}

// Queue is SellBackFakeApi.queue: picked-up books waiting to be graded, for staff.
func (h *Handler) Queue(w http.ResponseWriter, r *http.Request) {
	v, err := h.S.Queue(r.Context())
	answer(h, w, r, &v, err)
}

// Grade is SellBackFakeApi.grade: body `{id, condition, accept, by}` (`by` is ignored). Answers
// the queue.
func (h *Handler) Grade(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ID        string `json:"id"`
		Condition string `json:"condition"`
		Accept    bool   `json:"accept"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	v, err := h.S.Grade(r.Context(), in.ID, in.Condition, in.Accept)
	answer(h, w, r, &v, err)
}

// Routes registers the Sell Back endpoints.
func (h *Handler) Routes(r httpx.Router, a *auth.Middleware) {
	me := func(f http.HandlerFunc) http.Handler { return a.Me(f) }
	catalog := func(f http.HandlerFunc) http.Handler { return a.StaffCan(auth.PermCatalog, f) }
	r.Handle("GET /sell-back/books", me(h.Books))       // SellBackFakeApi.books
	r.Handle("GET /sell-back/book", me(h.Book))         // SellBackFakeApi.book
	r.Handle("POST /sell-back", me(h.Create))           // SellBackFakeApi.create
	r.Handle("GET /sell-back/mine", me(h.Mine))         // SellBackFakeApi.mine
	r.Handle("GET /sell-back/queue", catalog(h.Queue))  // SellBackFakeApi.queue
	r.Handle("POST /sell-back/grade", catalog(h.Grade)) // SellBackFakeApi.grade
}

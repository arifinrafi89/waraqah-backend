package reviews

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/arifinrafi89/waraqah-backend/internal/feature/moderation"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/auth"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/httpx"
)

// Handler is the HTTP side of the review endpoints.
type Handler struct{ S *Service }

// NewHandler builds the handler.
func NewHandler(s *Service) *Handler { return &Handler{S: s} }

func viewer(r *http.Request) string {
	u, _ := auth.UserFrom(r.Context())
	return u.ID
}

func (h *Handler) answer(w http.ResponseWriter, r *http.Request, v *BookReviews, err error) {
	var ref Refusal
	switch {
	case errors.As(err, &ref):
		httpx.Refuse(w, r, string(ref))
	case err != nil:
		h.S.Log.Error("reviews endpoint failed", "error", err)
		httpx.Error(w, r, http.StatusInternalServerError, httpx.CodeInternal, "something went wrong")
	default:
		httpx.JSON(w, v)
	}
}

// Reviews is ReviewFakeApi.reviews: `?bookId=`.
func (h *Handler) Reviews(w http.ResponseWriter, r *http.Request) {
	v, err := h.S.ForBook(r.Context(), viewer(r), httpx.Query(r, "bookId"))
	h.answer(w, r, v, err)
}

// Save is ReviewFakeApi.save: body `{bookId, stars, text}`.
func (h *Handler) Save(w http.ResponseWriter, r *http.Request) {
	var in struct {
		BookID string `json:"bookId"`
		Stars  int    `json:"stars"`
		Text   string `json:"text"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	v, err := h.S.Save(r.Context(), viewer(r), in.BookID, in.Stars, in.Text)
	h.answer(w, r, v, err)
}

// Delete is ReviewFakeApi.delete: body `{bookId}`, the reader's own review.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	var in struct {
		BookID string `json:"bookId"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	v, err := h.S.Delete(r.Context(), viewer(r), in.BookID)
	h.answer(w, r, v, err)
}

// Routes registers the review endpoints.
func (h *Handler) Routes(r httpx.Router, a *auth.Middleware) {
	r.Handle("GET /reviews", a.Public(http.HandlerFunc(h.Reviews)))    // ReviewFakeApi.reviews
	r.Handle("POST /reviews/save", a.Me(http.HandlerFunc(h.Save)))     // ReviewFakeApi.save
	r.Handle("POST /reviews/delete", a.Me(http.HandlerFunc(h.Delete))) // ReviewFakeApi.delete
}

// Reviews is the moderation subject of kind "review" (ReviewFakeStore.remove).
type Reviews struct{ S *Service }

var _ moderation.Subject = Reviews{}

// Find looks a review up for a report.
func (rv Reviews) Find(ctx context.Context, id string) (*moderation.Found, error) {
	q := rv.S.DB.Q()
	r, err := q.FindReview(ctx, id)
	if isNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	name := "?"
	if people, err := q.ListUserNames(ctx, []string{r.UserID}); err == nil && len(people) > 0 {
		name = people[0].Name
	}
	return &moderation.Found{Preview: r.Text, OwnerID: r.UserID, OwnerName: name}, nil
}

// Remove deletes the review and keeps the Book's rating, in the moderator's transaction.
func (rv Reviews) Remove(ctx context.Context, q *sqlc.Queries, id string) error {
	r, err := q.FindReview(ctx, id)
	if isNoRows(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if err := q.DeleteReview(ctx, sqlc.DeleteReviewParams{BookID: r.BookID, UserID: r.UserID}); err != nil {
		return err
	}
	if err := rv.S.rate(ctx, q, r.BookID); err != nil {
		return err
	}
	// The moderator's transaction commits after this returns: drop the cached catalog a moment
	// later too, so a read in between cannot keep the old rating.
	rv.S.Catalog.Invalidate()
	time.AfterFunc(500*time.Millisecond, rv.S.Catalog.Invalidate)
	return nil
}

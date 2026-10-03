// Package wishlist keeps the books a reader saved and the share link friends open without an account.
package wishlist

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/arifinrafi89/waraqah-backend/internal/feature/catalog"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/auth"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/clock"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/httpx"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/ids"
)

// Shared is SharedWishlistModel: a list friends can see.
type Shared struct {
	ID        string         `json:"id"`
	OwnerName string         `json:"ownerName"`
	Books     []catalog.Book `json:"books"`
}

// Service holds the wishlist rules.
type Service struct {
	DB    *db.DB
	Books catalog.Books
	Clock clock.Clock
	Log   *slog.Logger
}

// books resolves ids to catalog books in the order given (unknown ids are dropped).
func (s *Service) books(ctx context.Context, bookIDs []string) ([]catalog.Book, error) {
	out := []catalog.Book{}
	for _, id := range bookIDs {
		b, ok, err := s.Books.Find(ctx, id)
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, b)
		}
	}
	return out, nil
}

// List returns the saved books, newest first.
func (s *Service) List(ctx context.Context, userID string) ([]catalog.Book, error) {
	return s.list(ctx, s.DB.Q(), userID)
}

func (s *Service) list(ctx context.Context, q *sqlc.Queries, userID string) ([]catalog.Book, error) {
	ids, err := q.ListWishlistBookIDs(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.books(ctx, ids)
}

// Save saves a book; saving it twice keeps one copy, moved to the top. An unknown book changes nothing.
func (s *Service) Save(ctx context.Context, userID, bookID string) ([]catalog.Book, error) {
	if _, ok, err := s.Books.Find(ctx, bookID); err != nil {
		return nil, err
	} else if ok {
		if err := s.DB.Q().SaveWishlistItem(ctx, sqlc.SaveWishlistItemParams{UserID: userID, BookID: bookID, AddedAt: s.Clock.Now()}); err != nil {
			return nil, err
		}
	}
	return s.List(ctx, userID)
}

// Remove drops a book from the list.
func (s *Service) Remove(ctx context.Context, userID, bookID string) ([]catalog.Book, error) {
	if err := s.DB.Q().RemoveWishlistItem(ctx, sqlc.RemoveWishlistItemParams{UserID: userID, BookID: bookID}); err != nil {
		return nil, err
	}
	return s.List(ctx, userID)
}

// Share creates (or reuses) the share link of the reader. ownerName is the name friends see.
// It answers nil when the name is blank.
func (s *Service) Share(ctx context.Context, userID, ownerName string) (*Shared, error) {
	name := strings.TrimSpace(ownerName)
	if name == "" {
		return nil, nil
	}
	q := s.DB.Q()
	id := ""
	if cur, err := q.GetShareByUser(ctx, userID); err == nil {
		id = cur.ID
	} else if errors.Is(err, pgx.ErrNoRows) {
		id = ids.New("wl")
	} else {
		return nil, err
	}
	if err := q.UpsertShare(ctx, sqlc.UpsertShareParams{ID: id, UserID: userID, OwnerName: name}); err != nil {
		return nil, err
	}
	books, err := s.List(ctx, userID)
	return &Shared{ID: id, OwnerName: name, Books: books}, err
}

// SharedByID returns the list behind a share link as it is now, or nil.
func (s *Service) SharedByID(ctx context.Context, id string) (*Shared, error) {
	share, err := s.DB.Q().GetShareByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	books, err := s.List(ctx, share.UserID)
	return &Shared{ID: share.ID, OwnerName: share.OwnerName, Books: books}, err
}

// Handler is the HTTP side of the wishlist endpoints.
type Handler struct{ S *Service }

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	h.S.Log.Error("wishlist endpoint failed", "error", err)
	httpx.Error(w, r, http.StatusInternalServerError, httpx.CodeInternal, "something went wrong")
}

func (h *Handler) books(w http.ResponseWriter, r *http.Request, books []catalog.Book, err error) {
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, books)
}

// Wishlist is WishlistFakeApi.wishlist. A guest gets an empty list.
func (h *Handler) Wishlist(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFrom(r.Context())
	if !ok {
		httpx.JSON(w, []catalog.Book{})
		return
	}
	books, err := h.S.List(r.Context(), u.ID)
	h.books(w, r, books, err)
}

type bookBody struct {
	BookID string `json:"bookId"`
}

// Save is WishlistFakeApi.save.
func (h *Handler) Save(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	var in bookBody
	if !httpx.Decode(w, r, &in) {
		return
	}
	books, err := h.S.Save(r.Context(), u.ID, in.BookID)
	h.books(w, r, books, err)
}

// Remove is WishlistFakeApi.remove.
func (h *Handler) Remove(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	var in bookBody
	if !httpx.Decode(w, r, &in) {
		return
	}
	books, err := h.S.Remove(r.Context(), u.ID, in.BookID)
	h.books(w, r, books, err)
}

// Share is WishlistFakeApi.share. The owner is the signed-in reader; ownerName is only the name
// friends see.
func (h *Handler) Share(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	var in struct {
		OwnerName string `json:"ownerName"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	shared, err := h.S.Share(r.Context(), u.ID, in.OwnerName)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	if shared == nil {
		httpx.Refuse(w, r, httpx.ErrWishlistNameMissing)
		return
	}
	httpx.JSON(w, shared)
}

// Shared is WishlistFakeApi.shared: a friend list, or null.
func (h *Handler) Shared(w http.ResponseWriter, r *http.Request) {
	shared, err := h.S.SharedByID(r.Context(), httpx.Query(r, "id"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	if shared == nil {
		httpx.Null(w)
		return
	}
	httpx.JSON(w, shared)
}

// Routes registers the wishlist endpoints.
func (h *Handler) Routes(r httpx.Router, a *auth.Middleware) {
	r.Handle("GET /wishlist", a.Me(http.HandlerFunc(h.Wishlist)))          // WishlistFakeApi.wishlist
	r.Handle("POST /wishlist/save", a.Me(http.HandlerFunc(h.Save)))        // WishlistFakeApi.save
	r.Handle("POST /wishlist/remove", a.Me(http.HandlerFunc(h.Remove)))    // WishlistFakeApi.remove
	r.Handle("POST /wishlist/share", a.Me(http.HandlerFunc(h.Share)))      // WishlistFakeApi.share
	r.Handle("GET /wishlist/shared", a.Public(http.HandlerFunc(h.Shared))) // WishlistFakeApi.shared
}

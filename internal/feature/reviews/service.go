// Package reviews is the readers' reviews of Books: one per reader per Book, Verified Purchase
// from delivered orders, and the Book's rating kept as their average. Port of
// review_fake_store.dart and review_fake_api.dart.
package reviews

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/arifinrafi89/waraqah-backend/internal/feature/catalog"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/orders"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/clock"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/httpx"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/ids"
)

// Refusal is a rule the request broke; the handler answers 200 null with its code.
type Refusal string

func (r Refusal) Error() string { return string(r) }

// Refusal codes (strings live in httpx/errors.go, documented in docs/error-codes.md).
const (
	ErrInvalid     = Refusal(httpx.ErrReviewInvalid)
	ErrUnknown     = Refusal(httpx.ErrReviewUnknown)
	ErrBookUnknown = Refusal(httpx.ErrBookUnknown)
	ErrBanned      = Refusal(httpx.ErrReaderBanned)
)

// Catalog is the part of catalog.Books reviews use: find a Book, and keep its rating.
type Catalog interface {
	Find(ctx context.Context, id string) (catalog.Book, bool, error)
	SetRating(ctx context.Context, q *sqlc.Queries, bookID string, rating float64) error
	Invalidate()
}

// Bans is moderation.Bans.
type Bans interface {
	IsBanned(ctx context.Context, userID string) (bool, error)
}

// Review is ReviewModel: whose it is and Verified Purchase come from the server.
type Review struct {
	ID         string     `json:"id"`
	BookID     string     `json:"bookId"`
	AuthorID   string     `json:"authorId"`
	AuthorName string     `json:"authorName"`
	Stars      int        `json:"stars"`
	CreatedAt  time.Time  `json:"createdAt"`
	Text       string     `json:"text"`
	EditedAt   *time.Time `json:"editedAt"`
	Verified   bool       `json:"verified"`
	IsMine     bool       `json:"isMine"`
}

// BookReviews is BookReviewsModel: the average, the count, the viewer's own and all, newest first.
type BookReviews struct {
	Average float64  `json:"average"`
	Count   int      `json:"count"`
	Mine    *Review  `json:"mine"`
	Reviews []Review `json:"reviews"`
}

// Service holds the review rules.
type Service struct {
	DB        *db.DB
	Catalog   Catalog
	Delivered orders.DeliveredBooks
	Bans      Bans
	Clock     clock.Clock
	Loc       *time.Location
	Log       *slog.Logger
}

func isNoRows(err error) bool { return errors.Is(err, pgx.ErrNoRows) }

// roundRating is ReviewFakeStore.average: one decimal, half away from zero.
func roundRating(avg float64) float64 { return math.Round(avg*10) / 10 }

// bought says whether a reader received the Book in a delivered order that was not a donation.
func (s *Service) bought(ctx context.Context, userID string) (map[string]bool, error) {
	lines, err := s.Delivered.DeliveredLines(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := map[string]bool{}
	for _, l := range lines {
		if !l.IsDonation {
			out[l.BookID] = true
		}
	}
	return out, nil
}

// ForBook answers the reviews of a Book as the viewer sees them.
func (s *Service) ForBook(ctx context.Context, viewer, bookID string) (*BookReviews, error) {
	rows, err := s.DB.Q().ListReviews(ctx, bookID)
	if err != nil {
		return nil, err
	}
	out := &BookReviews{Reviews: []Review{}}
	sum, bought := 0, map[string]map[string]bool{}
	for _, r := range rows {
		verified := r.SeedVerified
		if !verified {
			if bought[r.UserID] == nil {
				if bought[r.UserID], err = s.bought(ctx, r.UserID); err != nil {
					return nil, err
				}
			}
			verified = bought[r.UserID][bookID]
		}
		rv := Review{ID: r.ID, BookID: r.BookID, AuthorID: r.UserID, AuthorName: r.AuthorName, Stars: int(r.Stars),
			CreatedAt: r.CreatedAt.In(s.Loc).Truncate(time.Second), Text: r.Text, Verified: verified, IsMine: viewer != "" && r.UserID == viewer}
		if r.EditedAt.Valid {
			at := r.EditedAt.Time.In(s.Loc).Truncate(time.Second)
			rv.EditedAt = &at
		}
		if rv.IsMine {
			mine := rv
			out.Mine = &mine
		}
		sum += rv.Stars
		out.Reviews = append(out.Reviews, rv)
	}
	out.Count = len(out.Reviews)
	if out.Count > 0 {
		out.Average = roundRating(float64(sum) / float64(out.Count))
	}
	return out, nil
}

// rate keeps the Book's rating as the average of its reviews, in the same transaction. A Book
// whose last review goes keeps the rating it had (ReviewFakeStore._rate).
func (s *Service) rate(ctx context.Context, q *sqlc.Queries, bookID string) error {
	st, err := q.ReviewStats(ctx, bookID)
	if err != nil || st.Count == 0 {
		return err
	}
	return s.Catalog.SetRating(ctx, q, bookID, roundRating(st.Average))
}

// Save is ReviewFakeStore.save: saving again edits the viewer's review.
func (s *Service) Save(ctx context.Context, viewer, bookID string, stars int, text string) (*BookReviews, error) {
	if banned, err := s.Bans.IsBanned(ctx, viewer); err != nil || banned {
		if err == nil {
			err = ErrBanned
		}
		return nil, err
	}
	if Check(stars, text) != ProblemNone {
		return nil, ErrInvalid
	}
	if _, ok, err := s.Catalog.Find(ctx, bookID); err != nil || !ok {
		if err == nil {
			err = ErrBookUnknown
		}
		return nil, err
	}
	text = strings.TrimSpace(text)
	now := s.Clock.Now()
	err := s.DB.WithTx(ctx, func(q *sqlc.Queries) error {
		_, err := q.GetReviewOf(ctx, sqlc.GetReviewOfParams{BookID: bookID, UserID: viewer})
		switch {
		case err == nil:
			err = q.EditReview(ctx, sqlc.EditReviewParams{BookID: bookID, UserID: viewer, Stars: int32(stars), Text: text,
				EditedAt: pgtype.Timestamptz{Time: now, Valid: true}})
		case isNoRows(err):
			err = q.InsertReview(ctx, sqlc.InsertReviewParams{ID: ids.New("rv"), BookID: bookID, UserID: viewer, Stars: int32(stars),
				Text: text, CreatedAt: now})
		}
		if err != nil {
			return err
		}
		return s.rate(ctx, q, bookID)
	})
	if err != nil {
		return nil, err
	}
	s.Catalog.Invalidate()
	return s.ForBook(ctx, viewer, bookID)
}

// Delete is ReviewFakeStore.delete: the viewer's review of the Book; refused when there is none.
func (s *Service) Delete(ctx context.Context, viewer, bookID string) (*BookReviews, error) {
	err := s.DB.WithTx(ctx, func(q *sqlc.Queries) error {
		if _, err := q.GetReviewOf(ctx, sqlc.GetReviewOfParams{BookID: bookID, UserID: viewer}); err != nil {
			if isNoRows(err) {
				return ErrUnknown
			}
			return err
		}
		if err := q.DeleteReview(ctx, sqlc.DeleteReviewParams{BookID: bookID, UserID: viewer}); err != nil {
			return err
		}
		return s.rate(ctx, q, bookID)
	})
	if err != nil {
		return nil, err
	}
	s.Catalog.Invalidate()
	return s.ForBook(ctx, viewer, bookID)
}

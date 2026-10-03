// Package sellback is Sell Back and Certified Used: a reader sells a used book to Waraqah for an
// instant quote, a courier picks it up, staff grade it, Waraqah pays into the reader's wallet and
// puts the copy on sale as Certified Used. Port of sell_back_fake_store.dart,
// sell_back_books.dart and certified_used_stock.dart.
package sellback

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/arifinrafi89/waraqah-backend/internal/feature/catalog"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/notifications"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/wallet"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/clock"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/httpx"
)

// Sell Back statuses (SellBackStatus): a courier picks the book up, staff grade it, and Waraqah
// pays (or sends the book back).
const (
	Scheduled = "scheduled"
	PickedUp  = "pickedUp"
	PaidOut   = "paid"
	Returned  = "returned"
)

// Refusal is a rule the request broke; the handler answers 200 null with its code.
type Refusal string

func (r Refusal) Error() string { return string(r) }

// Refusal codes (strings live in httpx/errors.go, documented in docs/error-codes.md).
const (
	ErrBookUnknown = Refusal(httpx.ErrSellBackBookUnknown)
	ErrInvalid     = Refusal(httpx.ErrSellBackInvalid)
	ErrNotWaiting  = Refusal(httpx.ErrSellBackNotWaiting)
)

// Books is the part of catalog.Books Sell Back reads.
type Books interface {
	Find(ctx context.Context, id string) (catalog.Book, bool, error)
	Visible(ctx context.Context) ([]catalog.Book, error)
}

// Service holds the Sell Back rules and the Certified Used stock.
type Service struct {
	DB     *db.DB
	Books  Books
	Wallet wallet.Ledger
	Notify notifications.Sender
	Clock  clock.Clock
	Loc    *time.Location
	Log    *slog.Logger
	// PickupDelay is COURIER_PICKUP_DELAY: how long after booking the courier collects a book.
	PickupDelay time.Duration
}

func isNoRows(err error) bool { return errors.Is(err, pgx.ErrNoRows) }

func (s *Service) warn(what string, err error) {
	if err != nil {
		s.Log.Error("sell back side effect failed", "what", what, "error", err)
	}
}

// bookOf is SellBackBooks._of: a catalog Book Waraqah buys back, priced from its cheapest printed
// Edition. A book with only eBooks is not bought back.
func bookOf(b catalog.Book) (Book, bool) {
	cheapest := -1
	for _, e := range b.Editions {
		if e.Format == "ebook" {
			continue
		}
		if cheapest < 0 || e.PriceBdt < cheapest {
			cheapest = e.PriceBdt
		}
	}
	if cheapest < 0 {
		return Book{}, false
	}
	return Book{BookID: b.ID, Title: b.Title, Author: b.Author, NewPriceBdt: cheapest, CoverSeed: b.CoverSeed}, true
}

// FindBook is SellBackBooks.find: the quote basis of one book, or nil.
func (s *Service) FindBook(ctx context.Context, id string) (*Book, error) {
	b, ok, err := s.Books.Find(ctx, id)
	if err != nil || !ok || b.Hidden {
		return nil, err
	}
	out, ok := bookOf(b)
	if !ok {
		return nil, nil
	}
	return &out, nil
}

// SearchBooks is SellBackBooks.search: up to eight books whose title or author contains q
// (two letters at least).
func (s *Service) SearchBooks(ctx context.Context, q string) ([]Book, error) {
	out := []Book{}
	q = strings.ToLower(strings.TrimSpace(q))
	if len([]rune(q)) < 2 {
		return out, nil
	}
	all, err := s.Books.Visible(ctx)
	if err != nil {
		return nil, err
	}
	for _, b := range all {
		if !strings.Contains(strings.ToLower(b.Title), q) && !strings.Contains(strings.ToLower(b.Author), q) {
			continue
		}
		if sb, ok := bookOf(b); ok {
			out = append(out, sb)
		}
		if len(out) == 8 {
			break
		}
	}
	return out, nil
}

// Mine lists the reader's Sell Backs, newest first. A guest has none.
func (s *Service) Mine(ctx context.Context, userID string) ([]SellBack, error) {
	out := []SellBack{}
	if userID == "" {
		return out, nil
	}
	rows, err := s.DB.Q().ListSellBacksOf(ctx, userID)
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		out = append(out, toSellBack(r, nil, s.Loc))
	}
	return out, nil
}

// Queue lists the books picked up and waiting for staff, oldest first, with who is selling.
func (s *Service) Queue(ctx context.Context) ([]SellBack, error) {
	rows, err := s.DB.Q().ListSellBackQueue(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]SellBack, 0, len(rows))
	for _, r := range rows {
		name := r.ReaderName
		out = append(out, toSellBack(r.SellBack, &name, s.Loc))
	}
	return out, nil
}

// QueueLength counts the books waiting to be graded (dashboard).
func (s *Service) QueueLength(ctx context.Context) (int, error) {
	n, err := s.DB.Q().CountSellBackQueue(ctx)
	return int(n), err
}

// PickUp is the courier job (BACKEND_PLAN.md section 12): every pickup booked at least
// COURIER_PICKUP_DELAY ago is collected. It is idempotent and catches up after the server slept.
func (s *Service) PickUp(ctx context.Context) error {
	_, err := s.DB.Q().PickUpDue(ctx, s.Clock.Now().Add(-s.PickupDelay))
	return err
}

// schedulePickUp runs the courier when a new pickup is due instead of waiting for the next tick
// of the jobs. With no delay (tests) it runs at once.
func (s *Service) schedulePickUp() {
	run := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		s.warn("courier", s.PickUp(ctx))
	}
	if s.PickupDelay <= 0 {
		run()
		return
	}
	time.AfterFunc(s.PickupDelay+100*time.Millisecond, run)
}

// queries is q, or the plain connection when q is nil.
func (s *Service) queries(q *sqlc.Queries) *sqlc.Queries {
	if q != nil {
		return q
	}
	return s.DB.Q()
}

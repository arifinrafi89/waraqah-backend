// Package shelves is the reader's shelves (Want to Read, Reading, Finished), their reading
// progress, the days they read and their yearly goal. Port of shelf_fake_store.dart and
// reading_log.dart.
package shelves

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/arifinrafi89/waraqah-backend/internal/feature/catalog"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/orders"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/clock"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/httpx"
)

// Refusal is a rule the request broke; the handler answers 200 null with its code.
type Refusal string

func (r Refusal) Error() string { return string(r) }

// Refusal codes (strings live in httpx/errors.go, documented in docs/error-codes.md).
const (
	ErrBookUnknown     = Refusal(httpx.ErrBookUnknown)
	ErrProgressInvalid = Refusal(httpx.ErrProgressInvalid)
	ErrNotOnShelf      = Refusal(httpx.ErrNotOnShelf)
	ErrGoalInvalid     = Refusal(httpx.ErrGoalInvalid)
)

// Catalog gives the Books and Categories shelves show (catalog.Books).
type Catalog interface {
	Snapshot(ctx context.Context) (*catalog.Snapshot, error)
}

// Entry is ShelfEntryModel, with its Book inside.
type Entry struct {
	Book       catalog.Book `json:"book"`
	Shelf      string       `json:"shelf"`
	AddedAt    time.Time    `json:"addedAt"`
	FinishedAt *time.Time   `json:"finishedAt"`
	Progress   int          `json:"progress"`
	PagesRead  *int         `json:"pagesRead"`
	TotalPages *int         `json:"totalPages"`
}

// Service holds the shelf rules.
type Service struct {
	DB        *db.DB
	Catalog   Catalog
	Delivered orders.DeliveredBooks
	Clock     clock.Clock
	Loc       *time.Location
	Log       *slog.Logger
}

func isNoRows(err error) bool { return errors.Is(err, pgx.ErrNoRows) }

func valid(shelf string) bool { return shelf == WantToRead || shelf == Reading || shelf == Finished }

func ts(t time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: t, Valid: true} }

func int4(p *int) pgtype.Int4 {
	if p == nil {
		return pgtype.Int4{}
	}
	return pgtype.Int4{Int32: int32(*p), Valid: true}
}

func intp(v pgtype.Int4) *int {
	if !v.Valid {
		return nil
	}
	n := int(v.Int32)
	return &n
}

// syncDelivered is ShelfFakeStore._addDelivered: each Book of a delivered order the reader kept
// (not a donation or a gift) goes on Want to Read once; taking it off later is respected.
func (s *Service) syncDelivered(ctx context.Context, q *sqlc.Queries, userID string) error {
	lines, err := s.Delivered.DeliveredLines(ctx, userID)
	if err != nil {
		return err
	}
	for _, l := range lines {
		if l.IsDonation || l.IsGift {
			continue
		}
		n, err := q.MarkShelfSynced(ctx, sqlc.MarkShelfSyncedParams{UserID: userID, BookID: l.BookID})
		if err != nil {
			return err
		}
		if n > 0 {
			err = q.AddShelfEntryIfAbsent(ctx, sqlc.AddShelfEntryIfAbsentParams{UserID: userID, BookID: l.BookID, AddedAt: l.DeliveredAt})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Mine is ShelfFakeStore.mine: the reader's shelves, newest first, only Books still in the
// catalog. A guest has none.
func (s *Service) Mine(ctx context.Context, userID string) ([]Entry, error) {
	out := []Entry{}
	if userID == "" {
		return out, nil
	}
	q := s.DB.Q()
	if err := s.syncDelivered(ctx, q, userID); err != nil {
		return nil, err
	}
	snap, err := s.Catalog.Snapshot(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListShelf(ctx, userID)
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		book, ok := snap.Book(r.BookID)
		if !ok {
			continue
		}
		e := Entry{Book: book, Shelf: r.Shelf, AddedAt: r.AddedAt.In(s.Loc).Truncate(time.Second), Progress: int(r.Progress),
			PagesRead: intp(r.PagesRead), TotalPages: intp(r.TotalPages)}
		if r.FinishedAt.Valid {
			at := r.FinishedAt.Time.In(s.Loc).Truncate(time.Second)
			e.FinishedAt = &at
		}
		out = append(out, e)
	}
	return out, nil
}

// Move is ShelfFakeStore.move: puts a Book on a shelf (Finished sets 100%), or takes it off
// when the shelf is not one of the three. A Book the catalog does not have is refused.
func (s *Service) Move(ctx context.Context, userID, bookID, shelf string) ([]Entry, error) {
	snap, err := s.Catalog.Snapshot(ctx)
	if err != nil {
		return nil, err
	}
	if _, ok := snap.Book(bookID); !ok {
		return nil, ErrBookUnknown
	}
	q := s.DB.Q()
	if err := s.syncDelivered(ctx, q, userID); err != nil {
		return nil, err
	}
	if !valid(shelf) {
		if err := q.DeleteShelfEntry(ctx, sqlc.DeleteShelfEntryParams{UserID: userID, BookID: bookID}); err != nil {
			return nil, err
		}
		return s.Mine(ctx, userID)
	}
	old, err := q.GetShelfEntry(ctx, sqlc.GetShelfEntryParams{UserID: userID, BookID: bookID})
	if err != nil && !isNoRows(err) {
		return nil, err
	}
	if err == nil && old.Shelf == shelf {
		return s.Mine(ctx, userID)
	}
	now := s.Clock.Now()
	p := sqlc.PutShelfEntryParams{UserID: userID, BookID: bookID, Shelf: shelf, AddedAt: now, Progress: old.Progress,
		PagesRead: old.PagesRead, TotalPages: old.TotalPages}
	if shelf == Finished {
		p.FinishedAt, p.Progress = ts(now), 100
	}
	if err := q.PutShelfEntry(ctx, p); err != nil {
		return nil, err
	}
	return s.Mine(ctx, userID)
}

// Progress is ShelfFakeStore.progress: how far the reader got puts the Book on Reading (or
// Finished at 100%) and counts today as a reading day when it moved forward. Refused when it
// breaks ProgressRules or the Book is on no shelf.
func (s *Service) Progress(ctx context.Context, userID string, u Update) ([]Entry, error) {
	q := s.DB.Q()
	old, err := q.GetShelfEntry(ctx, sqlc.GetShelfEntryParams{UserID: userID, BookID: u.BookID})
	if isNoRows(err) {
		return nil, ErrNotOnShelf
	}
	if err != nil {
		return nil, err
	}
	if Check(u) != ProblemNone {
		return nil, ErrProgressInvalid
	}
	percent := u.Percent
	if u.TotalPages != nil {
		percent = PercentOf(*u.PagesRead, *u.TotalPages)
	}
	now := s.Clock.Now()
	p := sqlc.PutShelfEntryParams{UserID: userID, BookID: u.BookID, Shelf: Reading, AddedAt: now, Progress: int32(percent),
		PagesRead: int4(u.PagesRead), TotalPages: int4(u.TotalPages)}
	if old.Shelf == Reading {
		p.AddedAt = old.AddedAt
	}
	if percent == 100 {
		p.Shelf, p.FinishedAt = Finished, ts(now)
	}
	err = s.DB.WithTx(ctx, func(q *sqlc.Queries) error {
		if err := q.PutShelfEntry(ctx, p); err != nil {
			return err
		}
		if int32(percent) > old.Progress {
			return q.AddReadingDay(ctx, sqlc.AddReadingDayParams{UserID: userID, Day: day(now.In(s.Loc))})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.Mine(ctx, userID)
}

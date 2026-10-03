package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"math"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/clock"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
)

// Books is catalog.Books (BACKEND_PLAN.md section 8): how every other feature reads the catalog.
type Books interface {
	// Find returns a book by id, hidden ones too.
	Find(ctx context.Context, id string) (Book, bool, error)
	// FindEdition returns an edition and the book it belongs to.
	FindEdition(ctx context.Context, editionID string) (Book, Edition, bool, error)
	// Visible lists the books on the storefront, in storefront order.
	Visible(ctx context.Context) ([]Book, error)
}

// Find implements Books.
func (st *Store) Find(ctx context.Context, id string) (Book, bool, error) {
	s, err := st.Snapshot(ctx)
	if err != nil {
		return Book{}, false, err
	}
	b, ok := s.Book(id)
	return b, ok, nil
}

// FindEdition implements Books.
func (st *Store) FindEdition(ctx context.Context, editionID string) (Book, Edition, bool, error) {
	s, err := st.Snapshot(ctx)
	if err != nil {
		return Book{}, Edition{}, false, err
	}
	for _, b := range s.Books {
		for _, e := range b.Editions {
			if e.ID == editionID {
				return b, e, true, nil
			}
		}
	}
	return Book{}, Edition{}, false, nil
}

// Visible implements Books.
func (st *Store) Visible(ctx context.Context) ([]Book, error) {
	s, err := st.Snapshot(ctx)
	if err != nil {
		return nil, err
	}
	return s.Visible(), nil
}

// UsedStock answers with the Certified Used copy Waraqah has for a book (Sell Back, T16).
type UsedStock interface {
	CertifiedFor(ctx context.Context, bookID string) (*UsedCopy, error)
}

// NoUsedStock is the stand-in until Sell Back lands: Waraqah has no Certified Used copies.
type NoUsedStock struct{}

// CertifiedFor never finds a copy.
func (NoUsedStock) CertifiedFor(context.Context, string) (*UsedCopy, error) { return nil, nil }

// SearchLog is search.Log: it counts what readers search for (the dashboard implements it).
type SearchLog interface {
	Record(ctx context.Context, searcher, query string)
}

// NoSearchLog counts nothing.
type NoSearchLog struct{}

// Record does nothing.
func (NoSearchLog) Record(context.Context, string, string) {}

// Service answers the catalog read endpoints.
type Service struct {
	Store *Store
	Used  UsedStock
	// Searches counts reader searches of /books (staff includeHidden lists are not counted).
	Searches SearchLog
	Clock    clock.Clock
	Loc      *time.Location
	Log      *slog.Logger
}

func (s *Service) q() *sqlc.Queries { return s.Store.DB.Q() }

// salesWindowStart is the first day of the month 30 days ago: sales are kept per month, so the
// 30-day window is counted in whole months.
func (s *Service) salesWindowStart() time.Time {
	t := s.Clock.Now().In(s.Loc).AddDate(0, 0, -30)
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, s.Loc)
}

// Sold returns the copies sold per Edition in the last 30 days.
func (s *Service) Sold(ctx context.Context) (map[string]int, error) {
	rows, err := s.q().ListSalesSince(ctx, s.salesWindowStart())
	if err != nil {
		return nil, err
	}
	out := make(map[string]int, len(rows))
	for _, r := range rows {
		out[r.EditionID] = int(r.Copies)
	}
	return out, nil
}

// Details returns the summary and page count of a book, or nil.
func (s *Service) Details(ctx context.Context, id string) (*Details, error) {
	row, err := s.q().GetBookDetails(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &Details{BookID: row.BookID, Description: row.Description, Pages: int(row.Pages)}, nil
}

// LookInside returns the contents and sample pages of a book, or nil.
func (s *Service) LookInside(ctx context.Context, id string) (*LookInside, error) {
	row, err := s.q().GetLookInside(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	out := &LookInside{Contents: []ContentsEntry{}, SamplePages: []string{}}
	if err := json.Unmarshal(row.Contents, &out.Contents); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(row.SamplePages, &out.SamplePages); err != nil {
		return nil, err
	}
	return out, nil
}

// PriceLows returns each Edition lowest price in the last 30 days: the earlier low when it was
// lower than today price, the price itself otherwise.
func (s *Service) PriceLows(ctx context.Context, b Book) (map[string]int, error) {
	rows, err := s.q().ListPriceLows(ctx)
	if err != nil {
		return nil, err
	}
	earlier := map[string]int{}
	cutoff := s.Clock.Now().AddDate(0, 0, -30)
	for _, r := range rows {
		if r.Since.After(cutoff) {
			earlier[r.EditionID] = int(r.LowBdt)
		}
	}
	out := map[string]int{}
	for _, e := range b.Editions {
		low := e.PriceBdt
		if v, ok := earlier[e.ID]; ok && v < low {
			low = v
		}
		out[e.ID] = low
	}
	return out, nil
}

// SeriesOfBook returns the series a book is in, or nil.
func (s *Service) SeriesOfBook(snap *Snapshot, bookID string) *Series {
	for _, sr := range snap.Series {
		for _, e := range sr.Entries {
			if e.BookID != nil && *e.BookID == bookID {
				return withCovers(snap, sr)
			}
		}
	}
	return nil
}

// SeriesByID returns one series, or nil.
func (s *Service) SeriesByID(snap *Snapshot, id string) *Series {
	for _, sr := range snap.Series {
		if sr.ID == id {
			return withCovers(snap, sr)
		}
	}
	return nil
}

// withCovers gives each entry the cover of its book in the catalog (its position when it has none).
func withCovers(snap *Snapshot, sr Series) *Series {
	out := Series{ID: sr.ID, Name: sr.Name, Entries: make([]SeriesEntry, len(sr.Entries))}
	for i, e := range sr.Entries {
		e.CoverSeed = e.Position
		if e.BookID != nil {
			if b, ok := snap.Book(*e.BookID); ok {
				e.CoverSeed = b.CoverSeed
			}
		}
		out.Entries[i] = e
	}
	return &out
}

// ResaleValue is about 45% of the cheapest printed edition, to the nearest 10 taka; nil for
// eBook-only books, which cannot be resold.
func ResaleValue(b Book) *int {
	cheapest, found := 0, false
	for _, e := range b.Editions {
		if e.Format == "ebook" {
			continue
		}
		if !found || e.PriceBdt < cheapest {
			cheapest, found = e.PriceBdt, true
		}
	}
	if !found {
		return nil
	}
	v := int(math.Round(float64(cheapest)*0.45/10)) * 10
	return &v
}

// UsedOptionsOf combines the Certified Used copy and the resale estimate of a book.
func (s *Service) UsedOptionsOf(ctx context.Context, b Book) (UsedOptions, error) {
	copy, err := s.Used.CertifiedFor(ctx, b.ID)
	if err != nil {
		return UsedOptions{}, err
	}
	return UsedOptions{CertifiedUsed: copy, ResaleValueBdt: ResaleValue(b)}, nil
}

// Inventory is how orders change the stock and the sales of the catalog, inside their own
// transaction (BACKEND_PLAN.md section 8: catalog.Books covers price and stock changes).
// The cache is dropped by the caller after the commit (Invalidate).

// Take removes copies of a printed Edition (never below zero) and counts them as sold this month.
func (st *Store) Take(ctx context.Context, q *sqlc.Queries, editionID string, quantity int, at time.Time) error {
	if err := q.TakeStock(ctx, sqlc.TakeStockParams{ID: editionID, Stock: int32(quantity)}); err != nil {
		return err
	}
	month := time.Date(at.In(st.Loc).Year(), at.In(st.Loc).Month(), 1, 0, 0, 0, 0, st.Loc)
	return q.AddSale(ctx, sqlc.AddSaleParams{EditionID: editionID, Month: month, Copies: int32(quantity)})
}

// Restock gives copies of a printed Edition back (a cancelled order).
func (st *Store) Restock(ctx context.Context, q *sqlc.Queries, editionID string, quantity int) error {
	return q.RestockEdition(ctx, sqlc.RestockEditionParams{ID: editionID, Stock: int32(quantity)})
}

// SetRating saves a book's average review rating inside the caller's transaction (reviews). Call
// Invalidate after the commit so the next read sees it.
func (st *Store) SetRating(ctx context.Context, q *sqlc.Queries, bookID string, rating float64) error {
	return q.SetBookRating(ctx, sqlc.SetBookRatingParams{ID: bookID, Rating: rating})
}

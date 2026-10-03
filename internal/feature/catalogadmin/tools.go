package catalogadmin

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
)

// IsbnResult is the answer of the ISBN lookup: a Book already in the catalog, or one from outside.
type IsbnResult struct {
	InCatalog    bool    `json:"inCatalog"`
	BookID       *string `json:"bookId,omitempty"`
	ISBN         string  `json:"isbn,omitempty"`
	Title        string  `json:"title,omitempty"`
	TitleBn      *string `json:"titleBn,omitempty"`
	Author       string  `json:"author,omitempty"`
	Publisher    string  `json:"publisher,omitempty"`
	Language     string  `json:"language,omitempty"`
	Format       string  `json:"format,omitempty"`
	ListPriceBdt *int    `json:"listPriceBdt,omitempty"`
}

// LookUpIsbn is CatalogToolsFakeApi._lookUp: {inCatalog: true, bookId} when an Edition has the
// ISBN, a Book from outside the catalog (inCatalog false), or nil.
func (s *Service) LookUpIsbn(ctx context.Context, isbn string) (*IsbnResult, error) {
	books, err := s.DB.Q().ListBooks(ctx)
	if err != nil {
		return nil, err
	}
	editions, err := s.DB.Q().ListEditions(ctx)
	if err != nil {
		return nil, err
	}
	for _, e := range editions {
		if e.Isbn.Valid && e.Isbn.String == isbn {
			id := e.BookID
			for _, b := range books {
				if b.ID == e.BookID {
					return &IsbnResult{InCatalog: true, BookID: &id}, nil
				}
			}
		}
	}
	row, err := s.DB.Q().GetIsbnLookup(ctx, isbn)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	price := int(row.ListPriceBdt)
	out := &IsbnResult{ISBN: isbn, Title: row.Title, Author: row.Author, Publisher: row.Publisher, Language: row.Language,
		Format: row.Format, ListPriceBdt: &price}
	if row.TitleBn.Valid {
		out.TitleBn = &row.TitleBn.String
	}
	return out, nil
}

// LowStockItem is one printed Edition that needs more copies.
type LowStockItem struct {
	BookID    string `json:"bookId"`
	Title     string `json:"title"`
	CoverSeed int    `json:"coverSeed"`
	EditionID string `json:"editionId"`
	Format    string `json:"format"`
	Language  string `json:"language"`
	Stock     int    `json:"stock"`
}

// LowStock lists the printed Editions (no eBooks or pre-orders) at or under LowStock, lowest first.
func (s *Service) LowStock(ctx context.Context) ([]LowStockItem, error) {
	rows, err := s.DB.Q().LowStockEditions(ctx, LowStock)
	if err != nil {
		return nil, err
	}
	out := make([]LowStockItem, 0, len(rows))
	for _, r := range rows {
		out = append(out, LowStockItem{BookID: r.BookID, Title: r.Title, CoverSeed: int(r.CoverSeed), EditionID: r.EditionID,
			Format: r.Format, Language: r.Language, Stock: int(r.Stock)})
	}
	return out, nil
}

// StockResult is the answer of an edition stock change: the same body.
type StockResult struct {
	EditionID string `json:"editionId"`
	Stock     int    `json:"stock"`
}

// SetStock is CatalogToolsFakeApi._setStock: refused for an eBook, a negative stock or an unknown Edition.
func (s *Service) SetStock(ctx context.Context, editionID string, stock int) (StockResult, error) {
	err := s.DB.WithTx(ctx, func(q *sqlc.Queries) error {
		e, err := q.GetEditionForUpdate(ctx, editionID)
		if errors.Is(err, pgx.ErrNoRows) || stock < 0 || (err == nil && e.Format == "ebook") {
			return ErrStockInvalid
		}
		if err != nil {
			return err
		}
		_, err = q.SetEditionStock(ctx, sqlc.SetEditionStockParams{ID: editionID, Stock: int32(stock)})
		return err
	})
	if err != nil {
		return StockResult{}, err
	}
	s.changed(ctx, true)
	return StockResult{EditionID: editionID, Stock: stock}, nil
}

// ImportRow is one row of the CSV import: a BookDraft plus the names of its Author and Publisher.
type ImportRow struct {
	BookDraft
	Row       int    `json:"row"`
	Author    string `json:"author"`
	Publisher string `json:"publisher"`
}

// Skipped is a row the import refused.
type Skipped struct {
	Row    int    `json:"row"`
	Reason string `json:"reason"`
}

// ImportResult is the answer of the import.
type ImportResult struct {
	Imported int       `json:"imported"`
	Skipped  []Skipped `json:"skipped"`
}

// Import is CatalogImportFake.run: saves each row as a new Book, checked again. An Author or
// Publisher no record is named is created, and the row is undone with it when the Book is refused.
func (s *Service) Import(ctx context.Context, rows []ImportRow) (ImportResult, error) {
	out := ImportResult{Skipped: []Skipped{}}
	for _, r := range rows {
		err := s.DB.WithTx(ctx, func(q *sqlc.Queries) error {
			authorID, err := s.recordFor(ctx, q, "author", r.Author)
			if err != nil {
				return err
			}
			publisherID, err := s.recordFor(ctx, q, "publisher", r.Publisher)
			if err != nil {
				return err
			}
			d := r.BookDraft
			d.ID, d.AuthorID, d.PublisherID = nil, authorID, publisherID
			_, err = s.saveBook(ctx, q, d)
			return err
		})
		var ref Refusal
		switch {
		case err == nil:
			out.Imported++
		case errors.As(err, &ref):
			out.Skipped = append(out.Skipped, Skipped{Row: r.Row, Reason: "refused"})
		default:
			return out, err
		}
	}
	if out.Imported > 0 {
		s.changed(ctx, true)
	}
	return out, nil
}

// recordFor finds the record with that name (English or Bangla, any case) or creates it.
func (s *Service) recordFor(ctx context.Context, q *sqlc.Queries, kind, name string) (string, error) {
	key := strings.ToLower(strings.TrimSpace(name))
	list, err := s.listRecords(ctx, q, kind)
	if err != nil {
		return "", err
	}
	for _, r := range list {
		if strings.ToLower(r.Name) == key || (r.NameBn != "" && strings.ToLower(r.NameBn) == key) {
			return r.ID, nil
		}
	}
	rec, err := s.saveRecord(ctx, q, kind, RecordInput{Name: name})
	if err != nil {
		return "", nil // an empty name creates nothing; the Book is refused for it
	}
	return rec.ID, nil
}

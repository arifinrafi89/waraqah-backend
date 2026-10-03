package sellback

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/arifinrafi89/waraqah-backend/internal/feature/catalog"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
)

// The Certified Used stock (CertifiedUsedStock): copies bought back, graded and on sale. The
// book page offers the cheapest copy of each Book (catalog.UsedStock) and the cart sells it
// (cart.UsedStock); an order takes the copy off sale (SellCopy).

// publish puts a graded copy on sale: cu-<book>-<n>, n counting every copy the book ever had.
func (s *Service) publish(ctx context.Context, q *sqlc.Queries, bookID string, priceBdt int, condition string) error {
	n, err := q.CountCertifiedOf(ctx, bookID)
	if err != nil {
		return err
	}
	id := fmt.Sprintf("cu-%s-%d", strings.TrimPrefix(bookID, "bk-"), n+1)
	return q.InsertCertified(ctx, sqlc.InsertCertifiedParams{ID: id, BookID: bookID, Condition: condition,
		PriceBdt: int32(priceBdt), CreatedAt: s.Clock.Now()})
}

func toCopy(c sqlc.CertifiedUsed) catalog.UsedCopy {
	return catalog.UsedCopy{ID: c.ID, PriceBdt: int(c.PriceBdt), Condition: c.Condition}
}

// CertifiedFor is catalog.UsedStock: the cheapest copy of a book on sale, or nil.
func (s *Service) CertifiedFor(ctx context.Context, bookID string) (*catalog.UsedCopy, error) {
	c, err := s.DB.Q().CheapestCertified(ctx, bookID)
	if isNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	out := toCopy(c)
	return &out, nil
}

// Copy is cart.UsedStock: a copy on sale by its id, with its Book.
func (s *Service) Copy(ctx context.Context, copyID string) (catalog.Book, catalog.UsedCopy, bool, error) {
	c, err := s.DB.Q().GetCertified(ctx, copyID)
	if isNoRows(err) {
		return catalog.Book{}, catalog.UsedCopy{}, false, nil
	}
	if err != nil {
		return catalog.Book{}, catalog.UsedCopy{}, false, err
	}
	b, ok, err := s.Books.Find(ctx, c.BookID)
	if err != nil || !ok {
		return catalog.Book{}, catalog.UsedCopy{}, false, err
	}
	return b, toCopy(c), true, nil
}

// SellCopy takes a copy off sale inside the order's transaction. It answers false when the copy
// was sold already.
func (s *Service) SellCopy(ctx context.Context, q *sqlc.Queries, copyID string, at time.Time) (bool, error) {
	n, err := s.queries(q).SellCertified(ctx, sqlc.SellCertifiedParams{ID: copyID, SoldAt: pgtype.Timestamptz{Time: at, Valid: true}})
	return n > 0, err
}

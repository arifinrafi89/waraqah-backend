// Package cart keeps the cart of a reader: one line per item, quantities capped per order, nothing
// that can not be ordered. Prices come from the catalog and the deals when the cart is read.
package cart

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/arifinrafi89/waraqah-backend/internal/feature/catalog"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/deals"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/clock"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/httpx"
)

// PerOrderCap is the most copies of one printed Edition per order, however much is in stock.
const PerOrderCap = 10

// BundleCap is the most copies of one bundle.
const BundleCap = 5

// ErrItemUnknown is the refusal of an item that can not be added: unknown, not orderable, or a
// kind the cart does not sell (reader listings are bought with an offer).
const ErrItemUnknown = httpx.ErrCartItemUnknown

// Line is CartLineModel: one line of the cart, priced today.
type Line struct {
	ID           string  `json:"id"`
	Kind         string  `json:"kind"`
	ItemID       string  `json:"itemId"`
	BookID       string  `json:"bookId"`
	Title        string  `json:"title"`
	Author       string  `json:"author"`
	UnitPriceBdt int     `json:"unitPriceBdt"`
	Quantity     int     `json:"quantity"`
	MaxQuantity  int     `json:"maxQuantity"`
	ListPriceBdt *int    `json:"listPriceBdt"`
	Format       *string `json:"format"`
	Language     *string `json:"language"`
	IsPreorder   bool    `json:"isPreorder"`
	Condition    *string `json:"condition"`
	CoverSeed    int     `json:"coverSeed"`
}

// Cart is CartModel: every cart endpoint answers with it.
type Cart struct {
	Lines []Line `json:"lines"`
}

// Prices says which deal prices apply.
type Prices interface {
	FlashPrice(ctx context.Context, editionID string) (int, bool, error)
	BundleByID(ctx context.Context, id string) (*deals.Bundle, error)
}

// UsedStock finds a Certified Used copy by its id (Sell Back, T16).
type UsedStock interface {
	Copy(ctx context.Context, copyID string) (catalog.Book, catalog.UsedCopy, bool, error)
}

// NoUsedStock is the stand-in until Sell Back plugs in: there are no Certified Used copies.
type NoUsedStock struct{}

// Copy never finds a copy.
func (NoUsedStock) Copy(context.Context, string) (catalog.Book, catalog.UsedCopy, bool, error) {
	return catalog.Book{}, catalog.UsedCopy{}, false, nil
}

// Service holds the cart rules.
type Service struct {
	DB    *db.DB
	Books catalog.Books
	Deals Prices
	Used  UsedStock
	Clock clock.Clock
	Log   *slog.Logger
}

// maxFor: one copy of an eBook is enough; printed books cap at stock, or at PerOrderCap for pre-orders.
func maxFor(e catalog.Edition) int {
	if e.Format == "ebook" {
		return 1
	}
	if e.Stock > 0 {
		return min(e.Stock, PerOrderCap)
	}
	return PerOrderCap
}

func lineID(kind, itemID string) string { return kind + "-" + itemID }

func strp(s string) *string { return &s }
func intp(n int) *int       { return &n }

// resolve prices one item today, or returns false when it can not be sold (any more).
func (s *Service) resolve(ctx context.Context, kind, itemID string) (Line, bool, error) {
	switch kind {
	case "edition":
		b, e, ok, err := s.Books.FindEdition(ctx, itemID)
		if err != nil || !ok || !e.IsOrderable() {
			return Line{}, false, err
		}
		unit, list := e.PriceBdt, e.ListPriceBdt
		if flash, on, err := s.Deals.FlashPrice(ctx, e.ID); err != nil {
			return Line{}, false, err
		} else if on {
			unit, list = flash, intp(e.PriceBdt) // a flash price counts against the usual price
		}
		return Line{ID: lineID(kind, itemID), Kind: kind, ItemID: itemID, BookID: b.ID, Title: b.Title, Author: b.Author,
			UnitPriceBdt: unit, Quantity: 1, MaxQuantity: maxFor(e), ListPriceBdt: list, Format: strp(e.Format),
			Language: strp(e.Language), IsPreorder: e.Stock == 0 && e.IsPreorder, CoverSeed: b.CoverSeed}, true, nil
	case "bundle":
		bundle, err := s.Deals.BundleByID(ctx, itemID)
		if err != nil || bundle == nil || len(bundle.Items) == 0 {
			return Line{}, false, err
		}
		titles, regular := make([]string, 0, len(bundle.Items)), 0
		for _, it := range bundle.Items {
			titles = append(titles, it.Title)
			regular += it.RegularPriceBdt
		}
		first := bundle.Items[0] // the bundle is bought for the price of all its books
		return Line{ID: lineID(kind, itemID), Kind: kind, ItemID: itemID, BookID: first.BookID, Title: bundle.Title,
			Author: strings.Join(titles, " · "), UnitPriceBdt: bundle.PriceBdt, ListPriceBdt: intp(regular), Quantity: 1,
			MaxQuantity: BundleCap, CoverSeed: first.CoverSeed}, true, nil
	case "certifiedUsed":
		book, copy, ok, err := s.Used.Copy(ctx, itemID)
		if err != nil || !ok {
			return Line{}, false, err
		}
		// A used copy is one of a kind; its "list price" is the book new, so the cart shows the saving.
		return Line{ID: lineID(kind, itemID), Kind: kind, ItemID: itemID, BookID: book.ID, Title: book.Title, Author: book.Author,
			UnitPriceBdt: copy.PriceBdt, ListPriceBdt: intp(book.FromPriceBdt()), Quantity: 1, MaxQuantity: 1,
			Condition: strp(copy.Condition), CoverSeed: book.CoverSeed}, true, nil
	}
	return Line{}, false, nil // listings are bought with an offer, not through the cart
}

// Get returns the cart. Lines that can no longer be sold are left out, and a quantity above
// today maximum is shown at the maximum.
func (s *Service) Get(ctx context.Context, userID string) (Cart, error) {
	return s.cart(ctx, s.DB.Q(), userID)
}

func (s *Service) cart(ctx context.Context, q *sqlc.Queries, userID string) (Cart, error) {
	rows, err := q.ListCartLines(ctx, userID)
	if err != nil {
		return Cart{}, err
	}
	out := Cart{Lines: []Line{}}
	for _, r := range rows {
		line, ok, err := s.resolve(ctx, r.Kind, r.ItemID)
		if err != nil {
			return Cart{}, err
		}
		if !ok {
			continue
		}
		line.Quantity = max(1, min(int(r.Quantity), line.MaxQuantity))
		out.Lines = append(out.Lines, line)
	}
	return out, nil
}

// Add puts one more of an item in the cart (a new line, or +1 up to the maximum). refused is the
// refusal code when the item can not be added; the cart is answered either way.
func (s *Service) Add(ctx context.Context, userID, kind, itemID string) (cart Cart, refused string, err error) {
	err = s.DB.WithTx(ctx, func(q *sqlc.Queries) error {
		line, ok, err := s.resolve(ctx, kind, itemID)
		if err != nil {
			return err
		}
		if !ok {
			refused = ErrItemUnknown
		} else {
			cur, err := q.GetCartLine(ctx, sqlc.GetCartLineParams{UserID: userID, Kind: kind, ItemID: itemID})
			switch {
			case errors.Is(err, pgx.ErrNoRows):
				err = q.InsertCartLine(ctx, sqlc.InsertCartLineParams{UserID: userID, Kind: kind, ItemID: itemID, Quantity: 1, AddedAt: s.Clock.Now()})
			case err == nil:
				err = q.SetCartQuantity(ctx, sqlc.SetCartQuantityParams{UserID: userID, Kind: kind, ItemID: itemID,
					Quantity: int32(min(int(cur.Quantity)+1, line.MaxQuantity))})
			}
			if err != nil {
				return err
			}
		}
		cart, err = s.cart(ctx, q, userID)
		return err
	})
	return cart, refused, err
}

// find returns the stored line with that line id.
func (s *Service) find(ctx context.Context, q *sqlc.Queries, userID, id string) (sqlc.CartLine, bool, error) {
	rows, err := q.ListCartLines(ctx, userID)
	if err != nil {
		return sqlc.CartLine{}, false, err
	}
	for _, r := range rows {
		if lineID(r.Kind, r.ItemID) == id {
			return r, true, nil
		}
	}
	return sqlc.CartLine{}, false, nil
}

// SetQuantity changes a line quantity, kept between 1 and the maximum. An unknown line changes nothing.
func (s *Service) SetQuantity(ctx context.Context, userID, id string, quantity int) (Cart, error) {
	var out Cart
	err := s.DB.WithTx(ctx, func(q *sqlc.Queries) error {
		if row, ok, err := s.find(ctx, q, userID, id); err != nil {
			return err
		} else if ok {
			if line, ok, err := s.resolve(ctx, row.Kind, row.ItemID); err != nil {
				return err
			} else if ok {
				err := q.SetCartQuantity(ctx, sqlc.SetCartQuantityParams{UserID: userID, Kind: row.Kind, ItemID: row.ItemID,
					Quantity: int32(max(1, min(quantity, line.MaxQuantity)))})
				if err != nil {
					return err
				}
			}
		}
		var err error
		out, err = s.cart(ctx, q, userID)
		return err
	})
	return out, err
}

// Remove drops a line. An unknown line changes nothing.
func (s *Service) Remove(ctx context.Context, userID, id string) (Cart, error) {
	var out Cart
	err := s.DB.WithTx(ctx, func(q *sqlc.Queries) error {
		if row, ok, err := s.find(ctx, q, userID, id); err != nil {
			return err
		} else if ok {
			if err := q.DeleteCartLine(ctx, sqlc.DeleteCartLineParams{UserID: userID, Kind: row.Kind, ItemID: row.ItemID}); err != nil {
				return err
			}
		}
		var err error
		out, err = s.cart(ctx, q, userID)
		return err
	})
	return out, err
}

// CartIn reads the cart inside a transaction (checkout, T12).
func (s *Service) CartIn(ctx context.Context, q *sqlc.Queries, userID string) (Cart, error) {
	return s.cart(ctx, q, userID)
}

// ClearIn empties the cart inside a transaction (placing an order empties it).
func (s *Service) ClearIn(ctx context.Context, q *sqlc.Queries, userID string) error {
	return q.ClearCart(ctx, userID)
}

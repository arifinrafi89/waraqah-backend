// Package deals serves the flash sale, bundles and pre-orders running now, and answers the
// cart which flash price or bundle price applies.
package deals

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/arifinrafi89/waraqah-backend/internal/feature/catalog"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/auth"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/clock"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/httpx"
)

// PreorderLeadDays is how far away the release date of a pre-order is shown (the fake API always
// said 40 days out).
const PreorderLeadDays = 40

// DealItem is DealItemModel: one Edition on a deal.
type DealItem struct {
	BookID          string `json:"bookId"`
	EditionID       string `json:"editionId"`
	Title           string `json:"title"`
	RegularPriceBdt int    `json:"regularPriceBdt"`
	PriceBdt        int    `json:"priceBdt"`
	CoverSeed       int    `json:"coverSeed"`
}

// FlashSale is FlashSaleModel.
type FlashSale struct {
	Title  string     `json:"title"`
	EndsAt time.Time  `json:"endsAt"`
	Items  []DealItem `json:"items"`
}

// Bundle is BundleModel: Editions sold together at one price.
type Bundle struct {
	ID       string     `json:"id"`
	Title    string     `json:"title"`
	Items    []DealItem `json:"items"`
	PriceBdt int        `json:"priceBdt"`
}

// Preorder is PreorderModel.
type Preorder struct {
	Item        DealItem  `json:"item"`
	ReleaseDate time.Time `json:"releaseDate"`
}

// Deals is DealsModel: everything running now.
type Deals struct {
	FlashSale FlashSale  `json:"flashSale"`
	Bundles   []Bundle   `json:"bundles"`
	Preorders []Preorder `json:"preorders"`
}

// Service reads the deals and prices them from the catalog.
type Service struct {
	DB    *db.DB
	Books catalog.Books
	Clock clock.Clock
	Loc   *time.Location
	Log   *slog.Logger
}

func (s *Service) item(ctx context.Context, editionID string, price *int) (DealItem, bool, error) {
	b, e, ok, err := s.Books.FindEdition(ctx, editionID)
	if err != nil || !ok {
		return DealItem{}, false, err
	}
	p := e.PriceBdt
	if price != nil {
		p = *price
	}
	return DealItem{BookID: b.ID, EditionID: e.ID, Title: b.Title, RegularPriceBdt: e.PriceBdt, PriceBdt: p, CoverSeed: b.CoverSeed}, true, nil
}

// endOfDay is the end of the current day in the app timezone: the flash sale runs all day.
func (s *Service) endOfDay() time.Time {
	return clock.Today(s.Clock, s.Loc).AddDate(0, 0, 1)
}

// Current returns the deals running now.
func (s *Service) Current(ctx context.Context) (Deals, error) {
	out := Deals{FlashSale: FlashSale{Title: "Flash sale", EndsAt: s.endOfDay(), Items: []DealItem{}}, Bundles: []Bundle{}, Preorders: []Preorder{}}
	flash, err := s.DB.Q().ListFlashItems(ctx)
	if err != nil {
		return out, err
	}
	for _, f := range flash {
		price := int(f.PriceBdt)
		if it, ok, err := s.item(ctx, f.EditionID, &price); err != nil {
			return out, err
		} else if ok {
			out.FlashSale.Items = append(out.FlashSale.Items, it)
		}
	}
	bundles, err := s.bundles(ctx)
	if err != nil {
		return out, err
	}
	out.Bundles = bundles
	pre, err := s.DB.Q().ListPreorders(ctx)
	if err != nil {
		return out, err
	}
	release := clock.Today(s.Clock, s.Loc).AddDate(0, 0, PreorderLeadDays)
	for _, p := range pre {
		if it, ok, err := s.item(ctx, p.EditionID, nil); err != nil {
			return out, err
		} else if ok {
			out.Preorders = append(out.Preorders, Preorder{Item: it, ReleaseDate: release})
		}
	}
	return out, nil
}

func (s *Service) bundles(ctx context.Context) ([]Bundle, error) {
	rows, err := s.DB.Q().ListBundles(ctx)
	if err != nil {
		return nil, err
	}
	out := []Bundle{}
	for _, r := range rows {
		b := Bundle{ID: r.ID, Title: r.Title, PriceBdt: int(r.PriceBdt), Items: []DealItem{}}
		for _, id := range r.EditionIds {
			if it, ok, err := s.item(ctx, id, nil); err != nil {
				return nil, err
			} else if ok {
				b.Items = append(b.Items, it)
			}
		}
		out = append(out, b)
	}
	return out, nil
}

// FlashPrice is the flash price of an Edition while the sale is on.
func (s *Service) FlashPrice(ctx context.Context, editionID string) (int, bool, error) {
	items, err := s.DB.Q().ListFlashItems(ctx)
	if err != nil {
		return 0, false, err
	}
	for _, f := range items {
		if f.EditionID == editionID {
			return int(f.PriceBdt), true, nil
		}
	}
	return 0, false, nil
}

// BundleByID returns a bundle, or nil.
func (s *Service) BundleByID(ctx context.Context, id string) (*Bundle, error) {
	all, err := s.bundles(ctx)
	if err != nil {
		return nil, err
	}
	for i := range all {
		if all[i].ID == id {
			return &all[i], nil
		}
	}
	return nil, nil
}

// Handler is the HTTP side of GET /deals.
type Handler struct{ S *Service }

// Deals is DealsFakeApi.deals.
func (h Handler) Deals(w http.ResponseWriter, r *http.Request) {
	d, err := h.S.Current(r.Context())
	if err != nil {
		h.S.Log.Error("deals failed", "error", err)
		httpx.Error(w, r, http.StatusInternalServerError, httpx.CodeInternal, "something went wrong")
		return
	}
	httpx.JSON(w, d)
}

// Routes registers GET /deals (public).
func (h Handler) Routes(r httpx.Router, a *auth.Middleware) {
	r.Handle("GET /deals", a.Public(http.HandlerFunc(h.Deals))) // DealsFakeApi.deals
}

package p2p

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/arifinrafi89/waraqah-backend/internal/feature/catalog"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/clock"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/cloudinary"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/httpx"
)

// Refusal is a rule the request broke; the handler answers 200 null with its code.
type Refusal string

func (r Refusal) Error() string { return string(r) }

// Refusal codes (strings live in httpx/errors.go, documented in docs/error-codes.md).
const (
	ErrUnknown      = Refusal(httpx.ErrListingUnknown)
	ErrNotEditable  = Refusal(httpx.ErrListingNotEditable)
	ErrInvalid      = Refusal(httpx.ErrListingInvalid)
	ErrPhotoInvalid = Refusal(httpx.ErrListingPhotoInvalid)
	ErrBanned       = Refusal(httpx.ErrReaderBanned)
)

// Catalog is the part of the catalog a listing needs: the Category of its book.
type Catalog interface {
	Snapshot(ctx context.Context) (*catalog.Snapshot, error)
}

// Bans is moderation.Bans: whether a reader may use the marketplace.
type Bans interface {
	IsBanned(ctx context.Context, userID string) (bool, error)
}

// Listing is P2pListingModel, as every listing endpoint sends it.
type Listing struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	SellerID   string   `json:"sellerId"`
	SellerName string   `json:"sellerName"`
	PriceBdt   int      `json:"priceBdt"`
	Condition  string   `json:"condition"`
	Flags      []string `json:"flags"`
	Photos     []string `json:"photos"`
	// PhotoURLs is contract v1.1 (F7): the Cloudinary thumbnail of each slot in photos. Slots
	// without an uploaded image (seeded listings) are left out.
	PhotoURLs       map[string]string `json:"photoUrls"`
	IsNegotiable    bool              `json:"isNegotiable"`
	Handover        string            `json:"handover"`
	Status          string            `json:"status"`
	IsMine          bool              `json:"isMine"`
	IsMyDeal        bool              `json:"isMyDeal"`
	RejectionReason *string           `json:"rejectionReason"`
	BookID          *string           `json:"bookId"`
	CoverSeed       int               `json:"coverSeed"`
	District        *string           `json:"district"`
	Area            *string           `json:"area"`
	CategoryID      *string           `json:"categoryId"`
	Section         *string           `json:"section"`
	NewPriceBdt     *int              `json:"newPriceBdt"`
	Note            *string           `json:"note"`

	// BuyerID is who it is reserved for or was sold to; never sent.
	BuyerID string `json:"-"`
}

// Service holds the marketplace rules.
type Service struct {
	DB            *db.DB
	Images        cloudinary.Uploader
	MaxImageBytes int
	Catalog       Catalog
	Bans          Bans
	Clock         clock.Clock
	Loc           *time.Location
	Log           *slog.Logger
}

func isNoRows(err error) bool { return errors.Is(err, pgx.ErrNoRows) }

func text(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	return &t.String
}

func orEmpty(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

type row struct {
	l          sqlc.Listing
	sellerName string
}

// build turns rows into the JSON the app reads, for viewer ("" for a guest): photo slots, the
// catalog Category and Section, and the per-viewer fields.
func (s *Service) build(ctx context.Context, q *sqlc.Queries, viewer string, rows []row) ([]Listing, error) {
	out := make([]Listing, 0, len(rows))
	if len(rows) == 0 {
		return out, nil
	}
	idList := make([]string, len(rows))
	for i, r := range rows {
		idList[i] = r.l.ID
	}
	photoRows, err := q.ListPhotosOf(ctx, idList)
	if err != nil {
		return nil, err
	}
	photos, urls := map[string][]string{}, map[string]map[string]string{}
	for _, p := range photoRows {
		photos[p.ListingID] = append(photos[p.ListingID], p.Slot)
		if p.Url != "" {
			if urls[p.ListingID] == nil {
				urls[p.ListingID] = map[string]string{}
			}
			urls[p.ListingID][p.Slot] = cloudinary.Thumb(p.Url)
		}
	}
	snap, err := s.Catalog.Snapshot(ctx)
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		l := r.l
		item := Listing{ID: l.ID, Title: l.Title, SellerID: l.SellerID, SellerName: r.sellerName, PriceBdt: int(l.PriceBdt), Condition: l.Condition,
			Flags: orEmpty(l.Flags), Photos: orEmpty(photos[l.ID]), PhotoURLs: orEmptyMap(urls[l.ID]), IsNegotiable: l.IsNegotiable, Handover: l.Handover, Status: l.Status,
			IsMine: viewer != "" && l.SellerID == viewer, IsMyDeal: viewer != "" && l.BuyerID.String == viewer,
			RejectionReason: text(l.RejectionReason), BookID: text(l.BookID), CoverSeed: int(l.CoverSeed), District: text(l.District),
			Area: text(l.Area), CategoryID: text(l.CategoryID), Note: text(l.Note), BuyerID: l.BuyerID.String}
		if l.NewPriceBdt.Valid {
			v := int(l.NewPriceBdt.Int32)
			item.NewPriceBdt = &v
		}
		if item.CategoryID == nil && item.BookID != nil {
			if b, ok := snap.Book(*item.BookID); ok && b.CategoryID != "" {
				c := b.CategoryID
				item.CategoryID = &c
			}
		}
		if item.CategoryID != nil {
			if i := slices.IndexFunc(snap.Categories, func(c catalog.Category) bool { return c.ID == *item.CategoryID }); i >= 0 {
				sec := snap.Categories[i].Section
				item.Section = &sec
			}
		}
		out = append(out, item)
	}
	return out, nil
}

func rowsOf[T any](in []T, f func(T) row) []row {
	out := make([]row, len(in))
	for i, v := range in {
		out[i] = f(v)
	}
	return out
}

// Open lists the listings on sale or reserved, newest first. available keeps the live ones of
// other readers; limit caps the list (0 for no cap). Blocked sellers stay out.
func (s *Service) Open(ctx context.Context, viewer string, available bool, limit int) ([]Listing, error) {
	q := s.DB.Q()
	lim := pgtype.Int4{}
	if limit > 0 {
		lim = pgtype.Int4{Int32: int32(limit), Valid: true}
	}
	rows, err := q.ListOpenListings(ctx, sqlc.ListOpenListingsParams{OnlyAvailable: available, Viewer: viewer, Lim: lim})
	if err != nil {
		return nil, err
	}
	return s.build(ctx, q, viewer, rowsOf(rows, func(r sqlc.ListOpenListingsRow) row { return row{r.Listing, r.SellerName} }))
}

// Mine lists the own listings of a reader, in any status.
func (s *Service) Mine(ctx context.Context, viewer string) ([]Listing, error) {
	if viewer == "" {
		return []Listing{}, nil
	}
	q := s.DB.Q()
	rows, err := q.ListMineListings(ctx, viewer)
	if err != nil {
		return nil, err
	}
	return s.build(ctx, q, viewer, rowsOf(rows, func(r sqlc.ListMineListingsRow) row { return row{r.Listing, r.SellerName} }))
}

// ForBook lists the copies of a catalog book other readers are selling.
func (s *Service) ForBook(ctx context.Context, viewer, bookID string) ([]Listing, error) {
	q := s.DB.Q()
	rows, err := q.ListListingsForBook(ctx, sqlc.ListListingsForBookParams{BookID: bookID, Viewer: viewer})
	if err != nil {
		return nil, err
	}
	return s.build(ctx, q, viewer, rowsOf(rows, func(r sqlc.ListListingsForBookRow) row { return row{r.Listing, r.SellerName} }))
}

// One finds a listing as viewer sees it; nil when unknown.
func (s *Service) One(ctx context.Context, viewer, id string) (*Listing, error) {
	return s.oneIn(ctx, s.DB.Q(), viewer, id)
}

func (s *Service) oneIn(ctx context.Context, q *sqlc.Queries, viewer, id string) (*Listing, error) {
	r, err := q.GetListing(ctx, id)
	if isNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	l, err := s.build(ctx, q, viewer, []row{{r.Listing, r.SellerName}})
	if err != nil || len(l) == 0 {
		return nil, err
	}
	return &l[0], nil
}

func orEmptyMap(m map[string]string) map[string]string {
	if m == nil {
		return map[string]string{}
	}
	return m
}

package bookrequest

import (
	"context"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/arifinrafi89/waraqah-backend/internal/feature/notifications"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/p2p"
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
	ErrInvalid = Refusal(httpx.ErrRequestInvalid)
	ErrUnknown = Refusal(httpx.ErrRequestUnknown)
)

// Market is the part of the marketplace requests are matched against.
type Market interface {
	Candidates(ctx context.Context, excludedSeller string, onlyLive bool) ([]p2p.Candidate, error)
	OfSeller(ctx context.Context, sellerID string) ([]p2p.Candidate, error)
}

// Request is BookRequestModel, as its reader sees it, with the matches counted now.
type Request struct {
	ID              string    `json:"id"`
	Title           string    `json:"title"`
	CreatedAt       time.Time `json:"createdAt"`
	Author          *string   `json:"author"`
	BookID          *string   `json:"bookId"`
	MaxPriceBdt     *int      `json:"maxPriceBdt"`
	Note            *string   `json:"note"`
	IsOpen          bool      `json:"isOpen"`
	MatchCount      int       `json:"matchCount"`
	NotifiedSellers int       `json:"notifiedSellers"`
}

// Wanted is WantedBookModel: another reader asks for a book the signed-in reader sells.
type Wanted struct {
	RequestID   string    `json:"requestId"`
	ReaderName  string    `json:"readerName"`
	Title       string    `json:"title"`
	ListingID   string    `json:"listingId"`
	CreatedAt   time.Time `json:"createdAt"`
	MaxPriceBdt *int      `json:"maxPriceBdt"`
}

// Demand is BookDemandModel: how many open requests a title has.
type Demand struct {
	Title    string `json:"title"`
	Requests int    `json:"requests"`
}

// Service holds the book request rules.
type Service struct {
	DB     *db.DB
	Shop   Market
	Notify notifications.Sender
	Clock  clock.Clock
	Loc    *time.Location
	Log    *slog.Logger
}

func str(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	return &t.String
}

func (s *Service) json(r sqlc.BookRequest, candidates []p2p.Candidate) Request {
	out := Request{ID: r.ID, Title: r.Title, CreatedAt: r.CreatedAt.In(s.Loc).Truncate(time.Second), Author: str(r.Author), BookID: str(r.BookID),
		Note: str(r.Note), IsOpen: r.IsOpen}
	if r.MaxPriceBdt.Valid {
		v := int(r.MaxPriceBdt.Int32)
		out.MaxPriceBdt = &v
	}
	sellers := map[string]bool{}
	for _, c := range candidates {
		if c.SellerID == r.RequesterID || !Matches(r.Title, r.BookID.String, c.Title, c.BookID) {
			continue
		}
		sellers[c.SellerID] = true
		if c.Status == p2p.Live {
			out.MatchCount++
		}
	}
	out.NotifiedSellers = len(sellers)
	return out
}

// Create takes a request and tells the sellers who have a copy. Refused when it breaks RequestRules.
func (s *Service) Create(ctx context.Context, userID string, d Draft) (*Request, error) {
	if !Valid(d) {
		return nil, ErrInvalid
	}
	opt := func(p *string) pgtype.Text {
		if p == nil || strings.TrimSpace(*p) == "" {
			return pgtype.Text{}
		}
		return pgtype.Text{String: strings.TrimSpace(*p), Valid: true}
	}
	price := pgtype.Int4{}
	if d.MaxPriceBdt != nil {
		price = pgtype.Int4{Int32: int32(*d.MaxPriceBdt), Valid: true}
	}
	row := sqlc.BookRequest{ID: ids.Request(), RequesterID: userID, Title: strings.TrimSpace(d.Title), Author: opt(d.Author), BookID: opt(d.BookID),
		MaxPriceBdt: price, Note: opt(d.Note), IsOpen: true, CreatedAt: s.Clock.Now()}
	err := s.DB.Q().InsertBookRequest(ctx, sqlc.InsertBookRequestParams{ID: row.ID, RequesterID: userID, Title: row.Title, Author: row.Author,
		BookID: row.BookID, MaxPriceBdt: price, Note: row.Note, CreatedAt: row.CreatedAt})
	if err != nil {
		return nil, err
	}
	cands, err := s.Shop.Candidates(ctx, userID, false)
	if err != nil {
		return nil, err
	}
	out := s.json(row, cands)
	var sellers []string
	seen := map[string]bool{}
	for _, c := range cands {
		if Matches(row.Title, row.BookID.String, c.Title, c.BookID) && !seen[c.SellerID] {
			seen[c.SellerID] = true
			sellers = append(sellers, c.SellerID)
		}
	}
	if err := notifications.WantedBook(ctx, s.Notify, sellers, row.Title); err != nil {
		s.Log.Error("book request notification failed", "error", err)
	}
	return &out, nil
}

// Mine lists the requests of a reader, newest first.
func (s *Service) Mine(ctx context.Context, userID string) ([]Request, error) {
	out := []Request{}
	if userID == "" {
		return out, nil
	}
	rows, err := s.DB.Q().ListBookRequestsOf(ctx, userID)
	if err != nil {
		return nil, err
	}
	cands, err := s.Shop.Candidates(ctx, userID, false)
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		out = append(out, s.json(r, cands))
	}
	return out, nil
}

// Close ends a request of the reader and answers their requests; refused for an unknown one or
// a request of someone else.
func (s *Service) Close(ctx context.Context, userID, id string) ([]Request, error) {
	n, err := s.DB.Q().CloseBookRequest(ctx, sqlc.CloseBookRequestParams{ID: id, RequesterID: userID})
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, ErrUnknown
	}
	return s.Mine(ctx, userID)
}

// Wanted lists other readers' open requests for books the reader is selling (not sold yet).
func (s *Service) Wanted(ctx context.Context, userID string) ([]Wanted, error) {
	out := []Wanted{}
	if userID == "" {
		return out, nil
	}
	rows, err := s.DB.Q().ListOpenBookRequestsOfOthers(ctx, userID)
	if err != nil {
		return nil, err
	}
	mine, err := s.Shop.OfSeller(ctx, userID)
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		for _, l := range mine {
			if Matches(r.Title, r.BookID.String, l.Title, l.BookID) {
				w := Wanted{RequestID: r.ID, ReaderName: r.RequesterName, Title: l.Title, ListingID: l.ID, CreatedAt: r.CreatedAt.In(s.Loc).Truncate(time.Second)}
				if r.MaxPriceBdt.Valid {
					v := int(r.MaxPriceBdt.Int32)
					w.MaxPriceBdt = &v
				}
				out = append(out, w)
			}
		}
	}
	return out, nil
}

// Demand counts the open requests per title, most asked first (for staff).
func (s *Service) Demand(ctx context.Context) ([]Demand, error) {
	rows, err := s.DB.Q().ListOpenBookRequests(ctx)
	if err != nil {
		return nil, err
	}
	counts := map[string]int{}
	var order []string
	for _, r := range rows {
		if counts[r.Title] == 0 {
			order = append(order, r.Title)
		}
		counts[r.Title]++
	}
	out := make([]Demand, 0, len(order))
	for _, t := range order {
		out = append(out, Demand{Title: t, Requests: counts[t]})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Requests > out[j].Requests })
	return out, nil
}

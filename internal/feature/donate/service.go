package donate

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5"

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
	ErrCOD            = Refusal(httpx.ErrDonateCOD)
	ErrTooMany        = Refusal(httpx.ErrDonateTooMany)
	ErrUnknown        = Refusal(httpx.ErrDonateUnknown)
	ErrPaymentInvalid = Refusal(httpx.ErrPaymentInvalid)
	ErrPlaceInvalid   = Refusal(httpx.ErrPlaceInvalid)
	ErrPlaceUnknown   = Refusal(httpx.ErrPlaceUnknown)
)

var payments = []string{"bkash", "nagad", "cashOnDelivery", "card"}

// Books is the part of the catalog donate needs.
type Books interface {
	Find(ctx context.Context, id string) (catalog.Book, bool, error)
}

// Need is one book a place asked for, as the app reads it.
type Need struct {
	Book      catalog.Book `json:"book"`
	EditionID string       `json:"editionId"`
	PriceBdt  int          `json:"priceBdt"`
	Wanted    int          `json:"wanted"`
	Received  int          `json:"received"`
}

// Recipient is RecipientModel: a verified place and what it asked for.
type Recipient struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	District string `json:"district"`
	Area     string `json:"area"`
	Story    string `json:"story"`
	Needs    []Need `json:"needs"`
}

// Donation is DonationModel: what the donor gets back.
type Donation struct {
	OrderNumber string `json:"orderNumber"`
	TotalBdt    int    `json:"totalBdt"`
}

// GiveInput is the body of /donate/give.
type GiveInput struct {
	RecipientID string `json:"recipientId"`
	BookID      string `json:"bookId"`
	Quantity    int    `json:"quantity"`
	Payment     string `json:"payment"`
	Note        string `json:"note"`
}

// Service holds the donate rules.
type Service struct {
	DB    *db.DB
	Books Books
	Clock clock.Clock
	Log   *slog.Logger
}

func isNoRows(err error) bool { return errors.Is(err, pgx.ErrNoRows) }

// printed is the edition donors buy: the cheapest printed one that can be ordered.
func printed(b catalog.Book) (catalog.Edition, bool) {
	var best catalog.Edition
	found := false
	for _, e := range b.Editions {
		if e.Format != "ebook" && e.IsOrderable() && (!found || e.PriceBdt < best.PriceBdt) {
			best, found = e, true
		}
	}
	return best, found
}

func (s *Service) recipients(ctx context.Context, q *sqlc.Queries, places []sqlc.DonatePlace) ([]Recipient, error) {
	idList := make([]string, len(places))
	for i, p := range places {
		idList[i] = p.ID
	}
	rows, err := q.ListDonateNeeds(ctx, idList)
	if err != nil {
		return nil, err
	}
	byPlace := map[string][]sqlc.DonateNeed{}
	for _, n := range rows {
		byPlace[n.PlaceID] = append(byPlace[n.PlaceID], n)
	}
	out := make([]Recipient, 0, len(places))
	for _, p := range places {
		r := Recipient{ID: p.ID, Name: p.Name, Kind: p.Kind, District: p.District, Area: p.Area, Story: p.Story, Needs: []Need{}}
		for _, n := range byPlace[p.ID] {
			book, ok, err := s.Books.Find(ctx, n.BookID)
			if err != nil {
				return nil, err
			}
			if !ok {
				continue
			}
			if e, ok := printed(book); ok {
				r.Needs = append(r.Needs, Need{Book: book, EditionID: e.ID, PriceBdt: e.PriceBdt, Wanted: int(n.Wanted), Received: int(n.Received)})
			}
		}
		out = append(out, r)
	}
	return out, nil
}

// All lists every verified place.
func (s *Service) All(ctx context.Context) ([]Recipient, error) {
	q := s.DB.Q()
	places, err := q.ListDonatePlaces(ctx)
	if err != nil {
		return nil, err
	}
	return s.recipients(ctx, q, places)
}

// One finds a place, or nil.
func (s *Service) One(ctx context.Context, id string) (*Recipient, error) {
	q := s.DB.Q()
	p, err := q.GetDonatePlace(ctx, id)
	if isNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	rs, err := s.recipients(ctx, q, []sqlc.DonatePlace{p})
	if err != nil || len(rs) == 0 {
		return nil, err
	}
	return &rs[0], nil
}

// Give saves an order delivered free to the place. The copies count as received at once. Cash on
// delivery is refused, and so is more than the place still needs of the book.
func (s *Service) Give(ctx context.Context, userID string, in GiveInput) (*Donation, error) {
	if !slices.Contains(payments, in.Payment) {
		return nil, ErrPaymentInvalid
	}
	var out Donation
	err := s.DB.WithTx(ctx, func(q *sqlc.Queries) error {
		place, err := q.LockDonatePlace(ctx, in.RecipientID)
		if isNoRows(err) {
			return ErrUnknown
		}
		if err != nil {
			return err
		}
		needs, err := q.ListDonateNeeds(ctx, []string{place.ID})
		if err != nil {
			return err
		}
		i := slices.IndexFunc(needs, func(n sqlc.DonateNeed) bool { return n.BookID == in.BookID })
		book, found, err := s.Books.Find(ctx, in.BookID)
		if err != nil {
			return err
		}
		edition, ok := printed(book)
		if i < 0 || !found || !ok {
			return ErrUnknown
		}
		if in.Payment == "cashOnDelivery" {
			return ErrCOD
		}
		if in.Quantity < 1 || in.Quantity > int(needs[i].Wanted-needs[i].Received) {
			return ErrTooMany
		}
		number, err := ids.OrderNumber(ctx, s.DB)
		if err != nil {
			return err
		}
		now := s.Clock.Now()
		total := edition.PriceBdt * in.Quantity
		format, language, eid := edition.Format, edition.Language, edition.ID
		err = orders.Insert(ctx, q, orders.NewOrder{Number: number, UserID: userID, PlacedAt: now, AddressLabel: place.Name,
			AddressLine: place.Area + ", " + place.District, Payment: in.Payment, SubtotalBdt: total, TotalBdt: total, NeedsDelivery: true,
			Gift: &orders.Gift{RecipientName: place.Name, Message: strings.TrimSpace(in.Note)}, IsDonation: true, DonatePlaceID: place.ID,
			Lines: []orders.NewLine{{BookID: book.ID, EditionID: &eid, Title: book.Title, Author: book.Author, Quantity: in.Quantity,
				UnitPriceBdt: edition.PriceBdt, Format: &format, Language: &language, CoverSeed: book.CoverSeed}}})
		if err != nil {
			return err
		}
		if err := q.AddDonationReceived(ctx, sqlc.AddDonationReceivedParams{PlaceID: place.ID, BookID: in.BookID, Received: int32(in.Quantity)}); err != nil {
			return err
		}
		out = Donation{OrderNumber: number, TotalBdt: total}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Save adds or replaces a place (Staff) and answers every place. A replaced place starts counting
// received copies again, as in the app. Refused when it breaks PlaceRules, names a book the
// catalog does not have, or changes an unknown place.
func (s *Service) Save(ctx context.Context, d PlaceDraft) ([]Recipient, error) {
	d.Kind = kindOrDefault(d.Kind)
	if Check(d) != ProblemNone {
		return nil, ErrPlaceInvalid
	}
	for _, n := range d.Needs {
		if _, ok, err := s.Books.Find(ctx, n.BookID); err != nil {
			return nil, err
		} else if !ok {
			return nil, ErrPlaceInvalid
		}
	}
	err := s.DB.WithTx(ctx, func(q *sqlc.Queries) error {
		id := d.ID
		if id == "" {
			id = ids.Place()
			err := q.InsertDonatePlace(ctx, sqlc.InsertDonatePlaceParams{ID: id, Name: strings.TrimSpace(d.Name), Kind: d.Kind,
				District: strings.TrimSpace(d.District), Area: strings.TrimSpace(d.Area), Story: strings.TrimSpace(d.Story)})
			if err != nil {
				return err
			}
		} else {
			if _, err := q.LockDonatePlace(ctx, id); isNoRows(err) {
				return ErrPlaceUnknown
			} else if err != nil {
				return err
			}
			err := q.UpdateDonatePlace(ctx, sqlc.UpdateDonatePlaceParams{ID: id, Name: strings.TrimSpace(d.Name), Kind: d.Kind,
				District: strings.TrimSpace(d.District), Area: strings.TrimSpace(d.Area), Story: strings.TrimSpace(d.Story)})
			if err != nil {
				return err
			}
			if err := q.DeleteDonateNeeds(ctx, id); err != nil {
				return err
			}
		}
		seen := map[string]bool{}
		for i, n := range d.Needs {
			if seen[n.BookID] {
				continue
			}
			seen[n.BookID] = true
			if err := q.InsertDonateNeed(ctx, sqlc.InsertDonateNeedParams{PlaceID: id, Position: int32(i), BookID: n.BookID, Wanted: int32(n.Wanted)}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.All(ctx)
}

// Remove takes a place off the list (Staff) and answers the places left.
func (s *Service) Remove(ctx context.Context, id string) ([]Recipient, error) {
	n, err := s.DB.Q().RemoveDonatePlace(ctx, id)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, ErrPlaceUnknown
	}
	return s.All(ctx)
}

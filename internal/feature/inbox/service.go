package inbox

import (
	"context"
	"errors"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/arifinrafi89/waraqah-backend/internal/feature/p2p"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/clock"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/httpx"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/sse"
)

// Refusal is a rule the request broke; the handler answers 200 null with its code.
type Refusal string

func (r Refusal) Error() string { return string(r) }

// Refusal codes (strings live in httpx/errors.go, documented in docs/error-codes.md).
const (
	ErrThreadUnknown      = Refusal(httpx.ErrThreadUnknown)
	ErrBlocked            = Refusal(httpx.ErrBlockedReader)
	ErrMessageInvalid     = Refusal(httpx.ErrMessageInvalid)
	ErrOfferInvalid       = Refusal(httpx.ErrOfferInvalid)
	ErrOfferPending       = Refusal(httpx.ErrOfferPending)
	ErrOfferUnknown       = Refusal(httpx.ErrOfferUnknown)
	ErrListingUnavailable = Refusal(httpx.ErrListingUnavailable)
	ErrListingOwn         = Refusal(httpx.ErrListingOwn)
	ErrListingClosed      = Refusal(httpx.ErrListingClosed)
	ErrListingUnknown     = Refusal(httpx.ErrListingUnknown)
	ErrDealUnknown        = Refusal(httpx.ErrDealUnknown)
	ErrRatingInvalid      = Refusal(httpx.ErrRatingInvalid)
	ErrRatingNotAllowed   = Refusal(httpx.ErrRatingNotAllowed)
)

// Market is listings.Status and the ratings of the marketplace (p2p implements it).
type Market interface {
	p2p.Status
	RatingsFor(ctx context.Context, q *sqlc.Queries, listingIDs []string) (map[string]map[string]int, error)
	AddRating(ctx context.Context, q *sqlc.Queries, fromID, toID, listingID string, stars int, comment string, at time.Time) error
}

// Blocks is blocks.Checker: whether either of two readers blocked the other.
type Blocks interface {
	IsBlocked(ctx context.Context, viewer, other string) (bool, error)
}

// Service holds the inbox rules.
type Service struct {
	DB     *db.DB
	Market Market
	Blocks Blocks
	SSE    *sse.Broker
	Clock  clock.Clock
	Loc    *time.Location
	Log    *slog.Logger

	// DemoMode lets the seeded demo readers (ids "p-...") answer, after BotDelay, like
	// inbox_fake_replies.dart and inbox_fake_rating.dart; real readers never get an automatic answer.
	DemoMode bool
	BotDelay time.Duration
}

func isNoRows(err error) bool { return errors.Is(err, pgx.ErrNoRows) }

// isDemo: the seeded demo readers have ids like "p-nabila"; accounts made in the app have "u_...".
func isDemo(userID string) bool { return strings.HasPrefix(userID, "p-") }

func otherOf(t sqlc.Thread, viewer string) string {
	if viewer == t.BuyerID {
		return t.SellerID
	}
	return t.BuyerID
}

func readMarker(t sqlc.Thread, viewer string) int64 {
	if viewer == t.BuyerID {
		return t.BuyerRead
	}
	return t.SellerRead
}

// view builds the threads as viewer sees them. With lastOnly each thread carries only its latest
// message (the list), otherwise all of them.
func (s *Service) view(ctx context.Context, q *sqlc.Queries, viewer string, threads []sqlc.Thread, lastOnly bool) ([]Thread, error) {
	out := make([]Thread, 0, len(threads))
	if len(threads) == 0 {
		return out, nil
	}
	threadIDs, listingIDs, userIDs := make([]string, len(threads)), make([]string, 0, len(threads)), make([]string, 0, len(threads))
	for i, t := range threads {
		threadIDs[i] = t.ID
		listingIDs = append(listingIDs, t.ListingID)
		userIDs = append(userIDs, otherOf(t, viewer))
	}
	rows, err := q.ListMessagesOf(ctx, threadIDs)
	if err != nil {
		return nil, err
	}
	byThread := map[string][]sqlc.ListMessagesOfRow{}
	for _, m := range rows {
		byThread[m.ThreadID] = append(byThread[m.ThreadID], m)
	}
	names := map[string]string{}
	people, err := q.ListUserNames(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	for _, p := range people {
		names[p.ID] = p.Name
	}
	ratings, err := s.Market.RatingsFor(ctx, q, listingIDs)
	if err != nil {
		return nil, err
	}
	for _, t := range threads {
		listing, err := s.Market.Find(ctx, viewer, t.ListingID)
		if err != nil {
			return nil, err
		}
		if listing == nil {
			continue
		}
		other := otherOf(t, viewer)
		role := "seller"
		if t.BuyerID == viewer {
			role = "buyer"
		}
		th := Thread{ID: t.ID, Role: role, OtherID: other, OtherName: names[other], Messages: []Message{},
			Listing: ThreadListing{ID: listing.ID, Title: listing.Title, PriceBdt: listing.PriceBdt, Status: listing.Status, CoverSeed: listing.CoverSeed,
				IsNegotiable: listing.IsNegotiable, Handover: listing.Handover},
			DealHere: listing.BuyerID == t.BuyerID && (listing.Status == p2p.Reserved || listing.Status == p2p.Sold)}
		marker := readMarker(t, viewer)
		msgs := byThread[t.ID]
		for i, m := range msgs {
			if m.AuthorID != viewer && m.Position > marker {
				th.Unread++
			}
			th.lastAt = m.At
			if lastOnly && i < len(msgs)-1 {
				continue
			}
			th.Messages = append(th.Messages, s.message(m, viewer))
		}
		if listing.Status == p2p.Sold && listing.BuyerID == t.BuyerID {
			if v, ok := ratings[listing.ID][viewer]; ok {
				th.MyRating = &v
			}
			if v, ok := ratings[listing.ID][other]; ok {
				th.TheirRating = &v
			}
		}
		out = append(out, th)
	}
	return out, nil
}

func (s *Service) message(m sqlc.ListMessagesOfRow, viewer string) Message {
	from := "them"
	switch m.AuthorID {
	case viewer:
		from = "me"
	case System:
		from = "system"
	}
	msg := Message{ID: m.ID, From: from, At: m.At.In(s.Loc).Truncate(time.Second)}
	if m.Text.Valid {
		msg.Text = &m.Text.String
	}
	if m.OfferID.Valid {
		msg.Offer = &Offer{ID: m.OfferID.String, AmountBdt: int(m.OfferAmountBdt.Int32), Handover: m.OfferHandover.String, Status: m.OfferStatus.String}
	}
	if m.Event.Valid {
		msg.Event = &m.Event.String
	}
	if m.AmountBdt.Valid {
		v := int(m.AmountBdt.Int32)
		msg.AmountBdt = &v
	}
	return msg
}

// List answers the threads of a reader with messages, newest activity first, each with its latest
// message. listingID keeps those about one listing.
func (s *Service) List(ctx context.Context, viewer, listingID string) ([]Thread, error) {
	if viewer == "" {
		return []Thread{}, nil
	}
	q := s.DB.Q()
	all, err := q.ListThreadsOf(ctx, viewer)
	if err != nil {
		return nil, err
	}
	var mine []sqlc.Thread
	for _, t := range all {
		if listingID == "" || t.ListingID == listingID {
			mine = append(mine, t)
		}
	}
	threads, err := s.view(ctx, q, viewer, mine, true)
	if err != nil {
		return nil, err
	}
	withMessages := threads[:0]
	for _, t := range threads {
		if len(t.Messages) > 0 {
			withMessages = append(withMessages, t)
		}
	}
	sort.SliceStable(withMessages, func(i, j int) bool { return withMessages[i].lastAt.After(withMessages[j].lastAt) })
	return withMessages, nil
}

// Get answers the whole thread, or nil when it is unknown or the reader is not in it.
func (s *Service) Get(ctx context.Context, viewer, id string) (*Thread, error) {
	return s.getIn(ctx, s.DB.Q(), viewer, id)
}

func (s *Service) getIn(ctx context.Context, q *sqlc.Queries, viewer, id string) (*Thread, error) {
	t, err := q.GetThread(ctx, id)
	if isNoRows(err) || (err == nil && viewer != t.BuyerID && viewer != t.SellerID) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	v, err := s.view(ctx, q, viewer, []sqlc.Thread{t}, false)
	if err != nil || len(v) == 0 {
		return nil, err
	}
	return &v[0], nil
}

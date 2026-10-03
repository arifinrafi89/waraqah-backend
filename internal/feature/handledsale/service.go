// Package handledsale is a used book sold through Waraqah: the buyer pays in the app, the seller
// sends the book, and the money waits with Waraqah until the buyer confirms or a moderator settles
// a dispute. Port of handled_sale_fake_store.dart, _steps.dart and _money.dart.
package handledsale

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/arifinrafi89/waraqah-backend/internal/feature/notifications"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/p2p"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/wallet"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/clock"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/sse"
)

// Audit writes a line of the Moderation Center's audit log with the name of the staff member of
// the token (moderation implements it).
type Audit interface {
	Record(ctx context.Context, q *sqlc.Queries, staffID, action, subject, why string) error
}

// Audit actions of a settled dispute (AuditAction).
const (
	AuditRefunded   = "refunded"
	AuditPaidSeller = "paidSeller"
)

// ModeratorsTopic is the live topic moderators follow for disputes.
const ModeratorsTopic = "sales:moderators"

// Service holds the handled sale rules.
type Service struct {
	DB            *db.DB
	Market        p2p.Status
	Wallet        wallet.Ledger
	Notify        notifications.Sender
	Audit         Audit
	SSE           *sse.Broker
	Clock         clock.Clock
	Loc           *time.Location
	Log           *slog.Logger
	MaxImageBytes int

	// DemoMode lets the seeded demo sellers (ids "p-...") send a paid book after BotDelay, as the
	// fake API does; real sellers send it themselves with POST /sales/step.
	DemoMode bool
	BotDelay time.Duration
}

func isNoRows(err error) bool { return errors.Is(err, pgx.ErrNoRows) }

// view builds sales as viewer sees them, with the listing's title and cover and the names.
func (s *Service) view(ctx context.Context, q *sqlc.Queries, viewer string, rows []sqlc.HandledSale) ([]Sale, error) {
	out := make([]Sale, 0, len(rows))
	if len(rows) == 0 {
		return out, nil
	}
	userIDs := make([]string, 0, 2*len(rows))
	for _, r := range rows {
		userIDs = append(userIDs, r.BuyerID, r.SellerID)
	}
	names := map[string]string{}
	people, err := q.ListUserNames(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	for _, p := range people {
		names[p.ID] = p.Name
	}
	for _, r := range rows {
		f := facts{names: names}
		l, err := s.Market.Find(ctx, "", r.ListingID)
		if err != nil {
			return nil, err
		}
		if l != nil {
			f.title, f.coverSeed = l.Title, l.CoverSeed
		}
		out = append(out, toSale(r, viewer, f, s.Loc))
	}
	return out, nil
}

func (s *Service) one(ctx context.Context, q *sqlc.Queries, viewer string, r sqlc.HandledSale) (*Sale, error) {
	v, err := s.view(ctx, q, viewer, []sqlc.HandledSale{r})
	if err != nil {
		return nil, err
	}
	return &v[0], nil
}

// Mine lists the sales the reader buys or sells, newest first. A guest has none.
func (s *Service) Mine(ctx context.Context, viewer string) ([]Sale, error) {
	if viewer == "" {
		return []Sale{}, nil
	}
	q := s.DB.Q()
	rows, err := q.ListSalesOf(ctx, viewer)
	if err != nil {
		return nil, err
	}
	return s.view(ctx, q, viewer, rows)
}

// Get answers one sale of the reader, or nil when unknown or not theirs.
func (s *Service) Get(ctx context.Context, viewer, id string) (*Sale, error) {
	q := s.DB.Q()
	r, err := q.GetSale(ctx, id)
	if isNoRows(err) || (err == nil && viewer != r.BuyerID && viewer != r.SellerID) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return s.one(ctx, q, viewer, r)
}

// Earnings is the seller's money: held until buyers confirm, earned all time, paid out, and the
// payouts newest first (HandledSaleFakeMoney.earningsJson). A guest has nothing.
func (s *Service) Earnings(ctx context.Context, viewer string) (Earnings, error) {
	return s.earningsIn(ctx, s.DB.Q(), viewer)
}

func (s *Service) earningsIn(ctx context.Context, q *sqlc.Queries, viewer string) (Earnings, error) {
	out := Earnings{Payouts: []Payout{}}
	if viewer == "" {
		return out, nil
	}
	sums, err := q.SellerEarnings(ctx, viewer)
	if err != nil {
		return out, err
	}
	rows, err := q.ListPayouts(ctx, viewer)
	if err != nil {
		return out, err
	}
	out.HeldBdt, out.EarnedBdt = int(sums.HeldBdt), int(sums.EarnedBdt)
	for _, p := range rows {
		out.PaidOutBdt += int(p.AmountBdt)
		out.Payouts = append(out.Payouts, Payout{AmountBdt: int(p.AmountBdt), At: p.At.In(s.Loc).Truncate(time.Second)})
	}
	return out, nil
}

// Disputes lists the open disputes for moderators, oldest first, each from the buyer's side.
func (s *Service) Disputes(ctx context.Context) ([]Dispute, error) {
	q := s.DB.Q()
	rows, err := q.ListDisputedSales(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Dispute, 0, len(rows))
	for _, r := range rows {
		sale, err := s.one(ctx, q, r.BuyerID, r)
		if err != nil {
			return nil, err
		}
		people, err := q.ListUserNames(ctx, []string{r.BuyerID, r.SellerID})
		if err != nil {
			return nil, err
		}
		d := Dispute{Sale: *sale, BuyerName: "?", SellerName: "?"}
		for _, p := range people {
			if p.ID == r.BuyerID {
				d.BuyerName = p.Name
			}
			if p.ID == r.SellerID {
				d.SellerName = p.Name
			}
		}
		out = append(out, d)
	}
	return out, nil
}

// OpenDisputes counts the disputes waiting for a moderator (dashboard).
func (s *Service) OpenDisputes(ctx context.Context) (int, error) {
	n, err := s.DB.Q().CountDisputedSales(ctx)
	return int(n), err
}

// publish tells both readers of a sale, and moderators when it concerns a dispute, that it changed.
func (s *Service) publish(r sqlc.HandledSale, moderators bool) {
	for _, user := range []string{r.BuyerID, r.SellerID} {
		s.SSE.PublishSeq("sales:"+user, map[string]any{"saleId": r.ID})
	}
	if moderators {
		s.SSE.PublishSeq(ModeratorsTopic, map[string]any{"saleId": r.ID})
	}
}

func (s *Service) title(ctx context.Context, listingID, fallback string) string {
	if l, err := s.Market.Find(ctx, "", listingID); err == nil && l != nil {
		return l.Title
	}
	return fallback
}

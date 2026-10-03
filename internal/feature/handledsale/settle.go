package handledsale

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/arifinrafi89/waraqah-backend/internal/feature/notifications"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/p2p"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/wallet"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
)

func text(p *string) pgtype.Text {
	if p == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *p, Valid: true}
}

// Settle is a moderator's decision on a dispute (HandledSaleFakeMoney.settle): refund the buyer
// (the book goes back on sale) or pay the seller (the book is sold). Both readers are told, the
// audit log records it with the staff member of the token, and the open disputes are answered.
func (s *Service) Settle(ctx context.Context, staffID, id string, refund bool) ([]Dispute, error) {
	var r sqlc.HandledSale
	var title string
	err := s.DB.WithTx(ctx, func(q *sqlc.Queries) error {
		var err error
		r, err = q.LockSale(ctx, id)
		if isNoRows(err) || (err == nil && r.Status != Disputed) {
			return ErrNotDisputed
		}
		if err != nil {
			return err
		}
		title = s.title(ctx, r.ListingID, r.ID)
		action := AuditPaidSeller
		if refund {
			action = AuditRefunded
			if err := s.Market.SetStatus(ctx, q, r.ListingID, p2p.Live, ""); err != nil {
				return err
			}
			if err := s.Wallet.Credit(ctx, q, r.BuyerID, buyerPays(r), wallet.SaleRefund, r.ID, title); err != nil {
				return err
			}
			err = s.setStatus(ctx, q, &r, Refunded)
		} else {
			if err := s.Market.SetStatus(ctx, q, r.ListingID, p2p.Sold, r.BuyerID); err != nil {
				return err
			}
			err = s.setStatus(ctx, q, &r, Released)
		}
		if err != nil {
			return err
		}
		return s.Audit.Record(ctx, q, staffID, action, title, r.DisputeReason.String)
	})
	if err != nil {
		return nil, err
	}
	amount := sellerGets(r)
	if refund {
		amount = buyerPays(r)
	}
	s.warn("sale settled", notifications.SaleSettledNotice(ctx, s.Notify, r.BuyerID, r.SellerID, r.ID, title, refund, amount))
	s.publish(r, true)
	return s.Disputes(ctx)
}

// isDemo: the seeded demo readers have ids like "p-tanvir"; accounts made in the app have "u_...".
func isDemo(userID string) bool { return strings.HasPrefix(userID, "p-") }

// SendDemoSales is the demo bot (BACKEND_PLAN.md section 12): with DEMO_MODE, a seeded demo seller
// hands a paid book to the courier once the sale is DEMO_BOT_DELAY old, and the buyer is told. It
// runs as a job, so it is idempotent and catches up after the server slept.
func (s *Service) SendDemoSales(ctx context.Context) error {
	if !s.DemoMode {
		return nil
	}
	now := s.Clock.Now()
	var sent []sqlc.HandledSale
	err := s.DB.WithTx(ctx, func(q *sqlc.Queries) error {
		var err error
		sent, err = q.SendDemoSales(ctx, sqlc.SendDemoSalesParams{Now: now, Due: now.Add(-s.BotDelay)})
		return err
	})
	if err != nil {
		return err
	}
	for _, r := range sent {
		s.warn("demo sale sent", notifications.SaleSentNotice(ctx, s.Notify, r.BuyerID, r.ID, s.title(ctx, r.ListingID, "")))
		s.publish(r, false)
	}
	return nil
}

// scheduleDemoSend runs the demo bot when a sale from a demo seller is due, instead of waiting for
// the next tick of the jobs. With no delay (tests) it runs at once.
func (s *Service) scheduleDemoSend(r sqlc.HandledSale) {
	if !s.DemoMode || !isDemo(r.SellerID) {
		return
	}
	run := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		s.warn("demo bot", s.SendDemoSales(ctx))
	}
	if s.BotDelay <= 0 {
		run()
		return
	}
	time.AfterFunc(s.BotDelay+100*time.Millisecond, run)
}

package handledsale

import (
	"context"
	"encoding/json"
	"slices"
	"strings"

	"github.com/arifinrafi89/waraqah-backend/internal/feature/notifications"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/p2p"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/wallet"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/cloudinary"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/ids"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/textutil"
)

// change runs fn on a sale of viewer, locked, in a transaction. fn answers the new status and
// may queue work for after the commit (notifications). The sale is then published and answered.
func (s *Service) change(ctx context.Context, viewer, id string, moderators bool,
	fn func(q *sqlc.Queries, r *sqlc.HandledSale, after *[]func()) error) (*Sale, error) {
	var r sqlc.HandledSale
	var after []func()
	err := s.DB.WithTx(ctx, func(q *sqlc.Queries) error {
		var err error
		r, err = q.LockSale(ctx, id)
		if isNoRows(err) || (err == nil && viewer != r.BuyerID && viewer != r.SellerID) {
			return ErrSaleUnknown
		}
		if err != nil {
			return err
		}
		return fn(q, &r, &after)
	})
	if err != nil {
		return nil, err
	}
	s.publish(r, moderators)
	for _, f := range after {
		f()
	}
	return s.Get(ctx, viewer, id)
}

func (s *Service) setStatus(ctx context.Context, q *sqlc.Queries, r *sqlc.HandledSale, status string) error {
	r.Status = status
	return q.SetSaleStatus(ctx, sqlc.SetSaleStatusParams{ID: r.ID, Status: status, UpdatedAt: s.Clock.Now()})
}

func (s *Service) warn(what string, err error) {
	if err != nil {
		s.Log.Error("handled sale side effect failed", "what", what, "error", err)
	}
}

// Buy pays for a live listing of another reader and reserves it (HandledSaleFakeStore.buy).
// Cash on delivery is refused: Waraqah can only hold money paid up front.
func (s *Service) Buy(ctx context.Context, viewer, listingID, method string) (*Sale, error) {
	if !isPrepaid(method) {
		return nil, ErrPrepaidOnly
	}
	var r sqlc.HandledSale
	err := s.DB.WithTx(ctx, func(q *sqlc.Queries) error {
		l, err := q.LockListing(ctx, listingID)
		if isNoRows(err) {
			return ErrListingUnavailable
		}
		if err != nil {
			return err
		}
		if l.SellerID == viewer {
			return ErrListingOwn
		}
		if l.Status != p2p.Live {
			return ErrListingUnavailable
		}
		id, err := ids.HandledSaleID(ctx, s.DB)
		if err != nil {
			return err
		}
		price := int(l.PriceBdt)
		err = q.InsertSale(ctx, sqlc.InsertSaleParams{ID: id, ListingID: listingID, BuyerID: viewer, SellerID: l.SellerID,
			PriceBdt: l.PriceBdt, FeeBdt: int32(FeeFor(price)), DeliveryBdt: DeliveryBdt, Method: method, CreatedAt: s.Clock.Now()})
		if err != nil {
			return err
		}
		if err := s.Market.SetStatus(ctx, q, listingID, p2p.Reserved, viewer); err != nil {
			return err
		}
		r, err = q.GetSale(ctx, id)
		return err
	})
	if err != nil {
		return nil, err
	}
	sale, err := s.one(ctx, s.DB.Q(), viewer, r)
	if err != nil {
		return nil, err
	}
	s.publish(r, false)
	s.scheduleDemoSend(r)
	return sale, nil
}

// Step is the seller sending, or the buyer cancelling before that or confirming after
// (HandledSaleFakeSteps.step). Anything else is not the reader's move.
func (s *Service) Step(ctx context.Context, viewer, id, step string) (*Sale, error) {
	return s.change(ctx, viewer, id, false, func(q *sqlc.Queries, r *sqlc.HandledSale, after *[]func()) error {
		buying := r.BuyerID == viewer
		title := s.title(ctx, r.ListingID, "")
		switch {
		case step == StepSend && !buying && r.Status == Paid:
			buyer, saleID := r.BuyerID, r.ID
			*after = append(*after, func() { s.warn("sale sent", notifications.SaleSentNotice(ctx, s.Notify, buyer, saleID, title)) })
			return s.setStatus(ctx, q, r, Sent)
		case step == StepCancel && buying && r.Status == Paid:
			if err := s.Wallet.Credit(ctx, q, r.BuyerID, buyerPays(*r), wallet.SaleRefund, r.ID, title); err != nil {
				return err
			}
			if err := s.Market.SetStatus(ctx, q, r.ListingID, p2p.Live, ""); err != nil {
				return err
			}
			return s.setStatus(ctx, q, r, Cancelled)
		case step == StepConfirm && buying && r.Status == Sent:
			if err := s.Market.SetStatus(ctx, q, r.ListingID, p2p.Sold, r.BuyerID); err != nil {
				return err
			}
			seller, saleID, amount := r.SellerID, r.ID, sellerGets(*r)
			*after = append(*after, func() {
				s.warn("sale completed", notifications.SaleCompletedNotice(ctx, s.Notify, seller, saleID, title, amount))
			})
			return s.setStatus(ctx, q, r, Completed)
		}
		return ErrStepRefused
	})
}

// Dispute is the buyer saying a sent book is not as described (HandledSaleFakeMoney.dispute).
// The reason must be a DisputeReason, the note at most 300 characters and the photos at most 3
// real images; they are kept as the base64 the app sends, which it reads back (contract v1).
func (s *Service) Dispute(ctx context.Context, viewer, id, reason string, note *string, photos []string) (*Sale, error) {
	if !slices.Contains(reasons, reason) || len(photos) > MaxDisputePhotos {
		return nil, ErrDisputeInvalid
	}
	var noteText *string
	if note != nil {
		if t := strings.TrimSpace(*note); t != "" {
			if textutil.Len(t) > MaxDisputeNote {
				return nil, ErrDisputeInvalid
			}
			noteText = &t
		}
	}
	for _, p := range photos {
		if _, err := cloudinary.DecodeImage(p, s.MaxImageBytes); err != nil {
			return nil, ErrPhotoInvalid
		}
	}
	if photos == nil {
		photos = []string{}
	}
	pj, _ := json.Marshal(photos)
	return s.change(ctx, viewer, id, true, func(q *sqlc.Queries, r *sqlc.HandledSale, _ *[]func()) error {
		if r.BuyerID != viewer || r.Status != Sent {
			return ErrStepRefused
		}
		r.Status = Disputed
		return q.SetSaleDispute(ctx, sqlc.SetSaleDisputeParams{ID: r.ID, DisputeReason: text(&reason), DisputeNote: text(noteText),
			DisputePhotos: pj, UpdatedAt: s.Clock.Now()})
	})
}

// Payout sends everything earned and not yet paid out to the seller (HandledSaleFakeMoney.payout)
// and answers the earnings. Nothing to pay out is refused.
func (s *Service) Payout(ctx context.Context, viewer string) (*Earnings, error) {
	var out Earnings
	err := s.DB.WithTx(ctx, func(q *sqlc.Queries) error {
		if _, err := q.LockUser(ctx, viewer); err != nil {
			return err
		}
		e, err := s.earningsIn(ctx, q, viewer)
		if err != nil {
			return err
		}
		available := e.EarnedBdt - e.PaidOutBdt
		if available <= 0 {
			return ErrNothingToPayOut
		}
		if err := q.InsertPayout(ctx, sqlc.InsertPayoutParams{UserID: viewer, AmountBdt: int32(available), At: s.Clock.Now()}); err != nil {
			return err
		}
		out, err = s.earningsIn(ctx, q, viewer)
		return err
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

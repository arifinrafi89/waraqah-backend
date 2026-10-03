package orders

import (
	"context"
	"encoding/json"
	"errors"
	"slices"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/arifinrafi89/waraqah-backend/internal/feature/notifications"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/wallet"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/cloudinary"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
)

func isNoRows(err error) bool { return errors.Is(err, pgx.ErrNoRows) }

// Cancel cancels an order of the reader that has not shipped. It gives back the points spent,
// takes back the points earned, puts what was paid back in the wallet and restocks the books.
func (s *Service) Cancel(ctx context.Context, userID, number string) (*Order, error) {
	var out Order
	err := s.DB.WithTx(ctx, func(q *sqlc.Queries) error {
		o, ok, err := s.one(ctx, q, number)
		if err != nil {
			return err
		}
		if !ok || o.UserID != userID {
			return ErrOrderUnknown
		}
		if !CanCancel(o.Status) {
			return ErrOrderNotCancel
		}
		now := s.Clock.Now()
		if err := q.SetOrderStatus(ctx, sqlc.SetOrderStatusParams{Number: number, Status: Cancelled}); err != nil {
			return err
		}
		if err := q.InsertOrderHistory(ctx, sqlc.InsertOrderHistoryParams{OrderNumber: number, Status: Cancelled, At: now}); err != nil {
			return err
		}
		if err := s.Points.Undo(ctx, q, userID, number, o.PointsUsed, o.PointsEarned); err != nil {
			return err
		}
		refund := o.CancelRefund()
		if err := s.Wallet.Credit(ctx, q, userID, refund, wallet.CancelRefund, number, ""); err != nil {
			return err
		}
		if refund > 0 {
			if err := q.AddOrderRefund(ctx, sqlc.AddOrderRefundParams{Number: number, RefundedBdt: int32(refund)}); err != nil {
				return err
			}
		}
		for _, l := range o.Lines {
			if l.EditionID != nil {
				if err := s.Stock.Restock(ctx, q, *l.EditionID, l.Quantity); err != nil {
					return err
				}
			}
		}
		again, _, err := s.one(ctx, q, number)
		out = again
		return err
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

var returnReasons = []string{"damaged", "wrongBook", "other"}

// RequestReturn asks to send a delivered order back: once, within 7 days of delivery. The photos
// are checked as images (type and size) and at most 3 are kept; they stay base64, as the app reads them.
func (s *Service) RequestReturn(ctx context.Context, userID, number, reason, note string, photos []string) (*Order, error) {
	if !slices.Contains(returnReasons, reason) {
		return nil, ErrReturnReason
	}
	if len(photos) > MaxReturnPhotos {
		photos = photos[:MaxReturnPhotos]
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
	var out Order
	err := s.DB.WithTx(ctx, func(q *sqlc.Queries) error {
		o, ok, err := s.one(ctx, q, number)
		if err != nil {
			return err
		}
		if !ok || o.UserID != userID {
			return ErrOrderUnknown
		}
		if !o.CanRequestReturn(s.Clock.Now()) {
			return ErrReturnNotAllowed
		}
		if err := q.InsertReturn(ctx, sqlc.InsertReturnParams{OrderNumber: number, Reason: reason, RequestedAt: s.Clock.Now(), Note: note, Photos: pj}); err != nil {
			return err
		}
		out, _, err = s.one(ctx, q, number)
		return err
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ReorderResult is how many books went back in the cart, and how many could not.
type ReorderResult struct {
	Added   int `json:"added"`
	Skipped int `json:"skipped"`
}

// Reorder is "buy again": each Edition of the order goes back in the cart in the same quantity,
// as far as stock and the cart limits allow. Used copies and bundles cannot be bought again.
func (s *Service) Reorder(ctx context.Context, userID, number string) (*ReorderResult, error) {
	o, err := s.Details(ctx, userID, number)
	if err != nil {
		return nil, err
	}
	if o == nil {
		return nil, ErrOrderUnknown
	}
	res := &ReorderResult{}
	for _, l := range o.Lines {
		if l.EditionID == nil {
			res.Skipped += l.Quantity
			continue
		}
		inCart := false
		for i := 0; i < l.Quantity; i++ {
			c, _, err := s.Cart.Add(ctx, userID, "edition", *l.EditionID)
			if err != nil {
				return nil, err
			}
			inCart = false
			for _, cl := range c.Lines {
				if cl.Kind == "edition" && cl.ItemID == *l.EditionID {
					inCart = true
				}
			}
		}
		if inCart {
			res.Added += l.Quantity
		} else {
			res.Skipped += l.Quantity
		}
	}
	return res, nil
}

// Advance moves an order to its next step (Staff). The step must be the next one; otherwise it
// is refused, for example when someone else moved the order first. The reader is told.
func (s *Service) Advance(ctx context.Context, number, status string) (*Order, error) {
	var out Order
	err := s.DB.WithTx(ctx, func(q *sqlc.Queries) error {
		o, ok, err := s.one(ctx, q, number)
		if err != nil {
			return err
		}
		if !ok {
			return ErrOrderUnknown
		}
		if Next(o.Status) != status {
			return ErrStepUnavailable
		}
		if err := q.SetOrderStatus(ctx, sqlc.SetOrderStatusParams{Number: number, Status: status}); err != nil {
			return err
		}
		if err := q.InsertOrderHistory(ctx, sqlc.InsertOrderHistoryParams{OrderNumber: number, Status: status, At: s.Clock.Now()}); err != nil {
			return err
		}
		out, _, err = s.one(ctx, q, number)
		return err
	})
	if err != nil {
		return nil, err
	}
	if err := notifications.OrderChanged(ctx, s.Notify, out.UserID, out.Number, status); err != nil {
		s.Log.Error("order notification failed", "error", err)
	}
	return &out, nil
}

// DecideReturn approves or rejects a waiting return (Staff). Approving refunds the books to the
// wallet of the reader. The reader is told.
func (s *Service) DecideReturn(ctx context.Context, number string, approve bool) (*Order, error) {
	var out Order
	err := s.DB.WithTx(ctx, func(q *sqlc.Queries) error {
		o, ok, err := s.one(ctx, q, number)
		if err != nil {
			return err
		}
		if !ok {
			return ErrOrderUnknown
		}
		if o.ReturnRequest == nil || o.ReturnRequest.Status != ReturnRequested {
			return ErrReturnNotWaiting
		}
		status := ReturnRejected
		if approve {
			status = ReturnApproved
		}
		now := s.Clock.Now()
		if err := q.DecideReturn(ctx, sqlc.DecideReturnParams{OrderNumber: number, Status: status, DecidedAt: pgtype.Timestamptz{Time: now, Valid: true}}); err != nil {
			return err
		}
		if approve {
			refund := o.ReturnRefund()
			if err := s.Wallet.Credit(ctx, q, o.UserID, refund, wallet.ReturnRefund, number, ""); err != nil {
				return err
			}
			if refund > 0 {
				if err := q.AddOrderRefund(ctx, sqlc.AddOrderRefundParams{Number: number, RefundedBdt: int32(refund)}); err != nil {
					return err
				}
			}
		}
		out, _, err = s.one(ctx, q, number)
		return err
	})
	if err != nil {
		return nil, err
	}
	if err := notifications.ReturnDecided(ctx, s.Notify, out.UserID, out.Number, approve); err != nil {
		s.Log.Error("return notification failed", "error", err)
	}
	return &out, nil
}

// DeliveredLine is a book that reached a reader, for Verified Purchase reviews and shelves.
type DeliveredLine struct {
	BookID      string
	EditionID   string
	OrderNumber string
}

// DeliveredBooks is orders.Delivered (BACKEND_PLAN.md section 8): the delivered lines of a reader.
type DeliveredBooks interface {
	DeliveredLines(ctx context.Context, userID string) ([]DeliveredLine, error)
}

// DeliveredLines lists the books of delivered orders of a reader.
func (s *Service) DeliveredLines(ctx context.Context, userID string) ([]DeliveredLine, error) {
	rows, err := s.DB.Q().DeliveredLinesOf(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]DeliveredLine, 0, len(rows))
	for _, r := range rows {
		out = append(out, DeliveredLine{BookID: r.BookID, EditionID: r.EditionID.String, OrderNumber: r.Number})
	}
	return out, nil
}

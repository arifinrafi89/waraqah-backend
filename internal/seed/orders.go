package seed

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/arifinrafi89/waraqah-backend/internal/feature/orders"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
)

// loadOrders loads the coupons and the demo reader's two orders, wallet credit and points.
// Orders are written once (by number); the wallet and points entries once by what they are.
func loadOrders(ctx context.Context, r *Run) error {
	var d struct {
		Coupons []struct {
			Code, Kind         string
			Value, MinOrderBdt int
			MaxDiscountBdt     *int
		}
		Orders []struct {
			Number     string
			AgeMinutes int
			StepHours  []int
			Payment    string
			Gift       *orders.Gift
			Lines      []orders.NewLine
		}
		Wallet []struct {
			AmountBdt, AgeMinutes int
			Reason, Note          string
		}
		Points []struct {
			Points, AgeMinutes int
			Reason             string
			OrderNumber        string
		}
	}
	if err := r.read("orders.json", &d); err != nil {
		return err
	}
	now := time.Now()
	return r.DB.WithTx(ctx, func(q *sqlc.Queries) error {
		for _, c := range d.Coupons {
			max := pgtype.Int4{}
			if c.MaxDiscountBdt != nil {
				max = pgtype.Int4{Int32: int32(*c.MaxDiscountBdt), Valid: true}
			}
			err := q.UpsertSeedCoupon(ctx, sqlc.UpsertSeedCouponParams{Code: c.Code, Kind: c.Kind, Value: int32(c.Value),
				MinOrderBdt: int32(c.MinOrderBdt), MaxDiscountBdt: max})
			if err != nil {
				return err
			}
		}
		for _, o := range d.Orders {
			if exists, err := q.SeedOrderExists(ctx, o.Number); err != nil || exists {
				if err != nil {
					return err
				}
				continue
			}
			placed := now.Add(-time.Duration(o.AgeMinutes) * time.Minute)
			history := make([]orders.Change, len(o.StepHours))
			subtotal := 0
			for i, h := range o.StepHours {
				history[i] = orders.Change{Status: orders.Steps[i], At: placed.Add(time.Duration(h) * time.Hour)}
			}
			for _, l := range o.Lines {
				subtotal += l.UnitPriceBdt * l.Quantity
			}
			err := orders.Insert(ctx, q, orders.NewOrder{Number: o.Number, UserID: r.Me, PlacedAt: placed, Status: history[len(history)-1].Status,
				History: history, AddressLabel: "Home", AddressLine: "House 12, Road 5, Dhanmondi, Dhaka", Payment: o.Payment, Lines: o.Lines,
				SubtotalBdt: subtotal, DeliveryFeeBdt: 60, TotalBdt: subtotal + 60, NeedsDelivery: true, PointsEarned: subtotal / 100, Gift: o.Gift})
			if err != nil {
				return err
			}
		}
		for _, w := range d.Wallet {
			note := pgtype.Text{String: w.Note, Valid: true}
			if exists, err := q.SeedWalletEntryExists(ctx, sqlc.SeedWalletEntryExistsParams{UserID: r.Me, Reason: w.Reason, Note: note}); err != nil || exists {
				if err != nil {
					return err
				}
				continue
			}
			err := q.AddWalletEntry(ctx, sqlc.AddWalletEntryParams{UserID: r.Me, AmountBdt: int32(w.AmountBdt), Reason: w.Reason, Note: note,
				At: now.Add(-time.Duration(w.AgeMinutes) * time.Minute)})
			if err != nil {
				return err
			}
		}
		for _, p := range d.Points {
			order := pgtype.Text{String: p.OrderNumber, Valid: true}
			if exists, err := q.SeedPointsEntryExists(ctx, sqlc.SeedPointsEntryExistsParams{UserID: r.Me, Reason: p.Reason, OrderNumber: order}); err != nil || exists {
				if err != nil {
					return err
				}
				continue
			}
			err := q.AddPointsEntry(ctx, sqlc.AddPointsEntryParams{UserID: r.Me, Points: int32(p.Points), Reason: p.Reason,
				OrderNumber: pgText(p.OrderNumber), At: now.Add(-time.Duration(p.AgeMinutes) * time.Minute)})
			if err != nil {
				return err
			}
		}
		return nil
	})
}

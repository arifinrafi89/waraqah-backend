package seed

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/arifinrafi89/waraqah-backend/internal/feature/handledsale"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/sellback"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
)

// loadSales loads the handled sales already under way (handled_sale_seed.dart), the payout of
// the demo reader, the Sell Backs (sell_back_seed.dart) and the Certified Used copies on sale
// (certified_used_stock.dart). Rows already there are left alone. A sale not completed keeps its
// listing reserved for the buyer, as the fake store does.
func loadSales(ctx context.Context, r *Run) error {
	var d struct {
		Sales []struct {
			ID, ListingID, BuyerID, SellerID, Method, Status string
			PriceBdt, AgeMinutes                             int
			DisputeReason, DisputeNote                       *string
		}
		Payouts []struct {
			UserID                string
			AmountBdt, AgeMinutes int
		}
		SellBacks []struct {
			ID, UserID, BookID, Condition, Status string
			AgeMinutes                            int
			PaidBdt                               *int
		}
		CertifiedUsed []struct {
			ID, BookID, Condition string
			PriceBdt              int
		}
	}
	if err := r.read("sales.json", &d); err != nil {
		return err
	}
	now := time.Now()
	ago := func(m int) time.Time { return now.Add(-time.Duration(m) * time.Minute) }
	return r.DB.WithTx(ctx, func(q *sqlc.Queries) error {
		for _, s := range d.Sales {
			err := q.SeedSale(ctx, sqlc.SeedSaleParams{ID: s.ID, ListingID: s.ListingID, BuyerID: r.who(s.BuyerID), SellerID: r.who(s.SellerID),
				PriceBdt: int32(s.PriceBdt), FeeBdt: int32(handledsale.FeeFor(s.PriceBdt)), DeliveryBdt: handledsale.DeliveryBdt,
				Method: s.Method, Status: s.Status, CreatedAt: ago(s.AgeMinutes), DisputeReason: optText(s.DisputeReason), DisputeNote: optText(s.DisputeNote)})
			if err != nil {
				return err
			}
			status := "reserved"
			if s.Status == handledsale.Completed {
				status = "sold"
			}
			err = q.SeedListingBuyer(ctx, sqlc.SeedListingBuyerParams{ID: s.ListingID, Status: status, BuyerID: pgtype.Text{String: r.who(s.BuyerID), Valid: true}})
			if err != nil {
				return err
			}
		}
		for _, p := range d.Payouts {
			if err := q.SeedPayout(ctx, sqlc.SeedPayoutParams{UserID: r.who(p.UserID), AmountBdt: int32(p.AmountBdt), At: ago(p.AgeMinutes)}); err != nil {
				return err
			}
		}
		for _, b := range d.SellBacks {
			book, err := q.SeedBookQuoteBasis(ctx, b.BookID)
			if err != nil {
				return err
			}
			graded, paid := pgtype.Text{}, pgtype.Int4{}
			if b.PaidBdt != nil {
				graded = pgtype.Text{String: b.Condition, Valid: true}
				paid = pgtype.Int4{Int32: int32(*b.PaidBdt), Valid: true}
			}
			err = q.SeedSellBack(ctx, sqlc.SeedSellBackParams{ID: b.ID, UserID: r.who(b.UserID), BookID: b.BookID, Title: book.Title, Author: book.Author,
				NewPriceBdt: book.NewPriceBdt, CoverSeed: book.CoverSeed, Condition: b.Condition,
				QuoteBdt: int32(sellback.Quote(int(book.NewPriceBdt), b.Condition, 0)), Status: b.Status, PickupAddress: "Road 7, Dhanmondi, Dhaka",
				CreatedAt: ago(b.AgeMinutes), GradedCondition: graded, PaidBdt: paid})
			if err != nil {
				return err
			}
		}
		for _, c := range d.CertifiedUsed {
			err := q.SeedCertified(ctx, sqlc.SeedCertifiedParams{ID: c.ID, BookID: c.BookID, Condition: c.Condition, PriceBdt: int32(c.PriceBdt), CreatedAt: now})
			if err != nil {
				return err
			}
		}
		return nil
	})
}

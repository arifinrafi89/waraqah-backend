package seed

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
)

// who maps "me" in the fixtures to the demo reader account.
func (r *Run) who(id string) string {
	if id == "me" {
		return r.Me
	}
	return id
}

func optText(p *string) pgtype.Text {
	if p == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *p, Valid: true}
}

// loadMarketplace loads the listings, the ratings readers gave each other, the books sold before
// the records of the app start, and the reports waiting in the Moderation Center. A listing
// already there is left alone, so what readers did since is kept.
func loadMarketplace(ctx context.Context, r *Run) error {
	var d struct {
		Listings []struct {
			ID, Title, SellerID, Condition, Handover, Status string
			PriceBdt, CoverSeed                              int
			Flags, Photos                                    []string
			IsNegotiable                                     bool
			RejectionReason, BookID, District, Area          *string
			CategoryID, Note, BuyerID                        *string
			NewPriceBdt                                      *int
		}
		Ratings []struct {
			FromID, ToID string
			Stars        int
			AgeMinutes   int
			ListingID    *string
			Comment      *string
		}
		SoldBefore map[string]int
	}
	var reports []struct {
		ID, Kind, TargetID, Reason, ReporterID string
		AgeMinutes                             int
		Note                                   *string
	}
	if err := r.read("p2p.json", &d); err != nil {
		return err
	}
	if err := r.read("reports.json", &reports); err != nil {
		return err
	}
	now := time.Now()
	return r.DB.WithTx(ctx, func(q *sqlc.Queries) error {
		for id, n := range d.SoldBefore {
			if err := q.SeedSoldBefore(ctx, sqlc.SeedSoldBeforeParams{ID: r.who(id), SoldBefore: int32(n)}); err != nil {
				return err
			}
		}
		// The fixtures list the newest first; position grows with every new listing.
		for i, l := range d.Listings {
			flags := l.Flags
			if flags == nil {
				flags = []string{}
			}
			buyer := pgtype.Text{}
			if l.BuyerID != nil {
				buyer = pgtype.Text{String: r.who(*l.BuyerID), Valid: true}
			}
			newPrice := pgtype.Int4{}
			if l.NewPriceBdt != nil {
				newPrice = pgtype.Int4{Int32: int32(*l.NewPriceBdt), Valid: true}
			}
			err := q.SeedListing(ctx, sqlc.SeedListingParams{ID: l.ID, Position: int64(len(d.Listings) - i), SellerID: r.who(l.SellerID), Title: l.Title,
				PriceBdt: int32(l.PriceBdt), Condition: l.Condition, Flags: flags, IsNegotiable: l.IsNegotiable, Handover: l.Handover, Status: l.Status,
				RejectionReason: optText(l.RejectionReason), BookID: optText(l.BookID), CoverSeed: int32(l.CoverSeed), District: optText(l.District),
				Area: optText(l.Area), CategoryID: optText(l.CategoryID), NewPriceBdt: newPrice, Note: optText(l.Note), BuyerID: buyer})
			if err != nil {
				return err
			}
			for _, slot := range l.Photos {
				if err := q.SeedListingPhoto(ctx, sqlc.SeedListingPhotoParams{ListingID: l.ID, Slot: slot, Position: int32(slotPosition(slot))}); err != nil {
					return err
				}
			}
		}
		for _, x := range d.Ratings {
			err := q.SeedRating(ctx, sqlc.SeedRatingParams{FromID: r.who(x.FromID), ToID: r.who(x.ToID), ListingID: pgString(x.ListingID),
				Stars: int32(x.Stars), Comment: pgString(x.Comment), At: now.Add(-time.Duration(x.AgeMinutes) * time.Minute)})
			if err != nil {
				return err
			}
		}
		for _, p := range reports {
			err := q.SeedReport(ctx, sqlc.SeedReportParams{ID: p.ID, Kind: p.Kind, TargetID: p.TargetID, Reason: p.Reason, Note: optText(p.Note),
				ReporterID: r.who(p.ReporterID), CreatedAt: now.Add(-time.Duration(p.AgeMinutes) * time.Minute)})
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func slotPosition(slot string) int {
	for i, s := range []string{"front", "back", "spine", "inside", "damage"} {
		if s == slot {
			return i
		}
	}
	return 0
}

func pgString(p *string) pgtype.Text { return optText(p) }

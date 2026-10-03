package seed

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
)

// loadInbox loads the four demo conversations of the demo reader and the book requests of other
// readers. A thread or request already there is left alone, so what readers did since is kept.
func loadInbox(ctx context.Context, r *Run) error {
	var d struct {
		Threads []struct {
			ID, ListingID, BuyerID, SellerID string
			Replied                          bool
			ReadByMe                         int
			Messages                         []struct {
				ID, AuthorID string
				AgeMinutes   int
				Text, Event  *string
				AmountBdt    *int
				Offer        *struct {
					ID, Handover, Status string
					AmountBdt            int
				}
			}
		}
	}
	var requests []struct {
		ID, RequesterID, Title string
		Author, BookID, Note   *string
		MaxPriceBdt            *int
		AgeMinutes             int
	}
	if err := r.read("inbox.json", &d); err != nil {
		return err
	}
	if err := r.read("book_requests.json", &requests); err != nil {
		return err
	}
	now := time.Now()
	return r.DB.WithTx(ctx, func(q *sqlc.Queries) error {
		for _, t := range d.Threads {
			if exists, err := threadExists(ctx, q, t.ID); err != nil || exists {
				if err != nil {
					return err
				}
				continue
			}
			oldest := now
			for _, m := range t.Messages {
				if at := now.Add(-time.Duration(m.AgeMinutes) * time.Minute); at.Before(oldest) {
					oldest = at
				}
			}
			err := q.SeedThread(ctx, sqlc.SeedThreadParams{ID: t.ID, ListingID: t.ListingID, BuyerID: r.who(t.BuyerID), SellerID: r.who(t.SellerID),
				BotReplied: t.Replied, CreatedAt: oldest})
			if err != nil {
				return err
			}
			for _, m := range t.Messages {
				offerID := pgtype.Text{}
				if m.Offer != nil {
					offerID = pgtype.Text{String: m.Offer.ID, Valid: true}
					err := q.SeedOffer(ctx, sqlc.SeedOfferParams{ID: m.Offer.ID, ThreadID: t.ID, AmountBdt: int32(m.Offer.AmountBdt), Handover: m.Offer.Handover, Status: m.Offer.Status})
					if err != nil {
						return err
					}
				}
				amount := pgtype.Int4{}
				if m.AmountBdt != nil {
					amount = pgtype.Int4{Int32: int32(*m.AmountBdt), Valid: true}
				}
				err := q.SeedMessage(ctx, sqlc.SeedMessageParams{ID: m.ID, ThreadID: t.ID, AuthorID: r.who(m.AuthorID), At: now.Add(-time.Duration(m.AgeMinutes) * time.Minute),
					Text: optText(m.Text), OfferID: offerID, Event: optText(m.Event), AmountBdt: amount})
				if err != nil {
					return err
				}
			}
			// The demo reader has seen the first readByMe messages.
			positions, err := q.ListMessagePositions(ctx, t.ID)
			if err != nil {
				return err
			}
			var seen int64
			if t.ReadByMe > 0 && t.ReadByMe <= len(positions) {
				seen = positions[t.ReadByMe-1]
			}
			if r.who(t.BuyerID) == r.Me {
				err = q.SeedSetReadBuyer(ctx, sqlc.SeedSetReadBuyerParams{ID: t.ID, BuyerRead: seen})
			} else {
				err = q.SeedSetReadSeller(ctx, sqlc.SeedSetReadSellerParams{ID: t.ID, SellerRead: seen})
			}
			if err != nil {
				return err
			}
		}
		for _, p := range requests {
			price := pgtype.Int4{}
			if p.MaxPriceBdt != nil {
				price = pgtype.Int4{Int32: int32(*p.MaxPriceBdt), Valid: true}
			}
			err := q.SeedBookRequest(ctx, sqlc.SeedBookRequestParams{ID: p.ID, RequesterID: r.who(p.RequesterID), Title: p.Title, Author: optText(p.Author),
				BookID: optText(p.BookID), MaxPriceBdt: price, Note: optText(p.Note), CreatedAt: now.Add(-time.Duration(p.AgeMinutes) * time.Minute)})
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func threadExists(ctx context.Context, q *sqlc.Queries, id string) (bool, error) {
	_, err := q.GetThread(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

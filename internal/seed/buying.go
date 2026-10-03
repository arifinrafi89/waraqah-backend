package seed

import (
	"context"
	"strings"
	"time"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
)

// loadPeople creates the readers of the demo marketplace (Tanvir, Nabila, ...) as reader
// accounts, so the marketplace and the shared wishlist are not empty. They can sign in with the
// demo password.
func loadPeople(ctx context.Context, r *Run) error {
	var people []struct {
		ID, Name, Area, District, MemberSince string
		BooksSold                             int
	}
	if err := r.read("people.json", &people); err != nil {
		return err
	}
	hash, err := r.hash()
	if err != nil {
		return err
	}
	return r.DB.WithTx(ctx, func(q *sqlc.Queries) error {
		for _, p := range people {
			since, err := r.parseNaive(p.MemberSince)
			if err != nil {
				return err
			}
			err = q.UpsertSeedPerson(ctx, sqlc.UpsertSeedPersonParams{ID: p.ID, Email: pgText(strings.ToLower(p.Name) + "@waraqah.test"),
				Name: p.Name, PasswordHash: pgText(hash), Area: p.Area, District: p.District, MemberSince: since, SoldBefore: int32(p.BooksSold)})
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// loadBuying loads the deals and the friend wishlist.
func loadBuying(ctx context.Context, r *Run) error {
	var deals struct {
		Flash []struct {
			EditionID string
			PriceBdt  int
		}
		Bundles []struct {
			ID, Title  string
			EditionIDs []string
			PriceBdt   int
		}
		Preorders []string
	}
	var shares []struct {
		ID, OwnerID, OwnerName string
		BookIDs                []string
	}
	if err := r.read("deals.json", &deals); err != nil {
		return err
	}
	if err := r.read("wishlist_shared.json", &shares); err != nil {
		return err
	}
	now := time.Now()
	return r.DB.WithTx(ctx, func(q *sqlc.Queries) error {
		for i, f := range deals.Flash {
			if err := q.UpsertSeedFlashItem(ctx, sqlc.UpsertSeedFlashItemParams{EditionID: f.EditionID, Position: int32(i), PriceBdt: int32(f.PriceBdt)}); err != nil {
				return err
			}
		}
		for i, b := range deals.Bundles {
			err := q.UpsertSeedBundle(ctx, sqlc.UpsertSeedBundleParams{ID: b.ID, Position: int32(i), Title: b.Title, EditionIds: b.EditionIDs, PriceBdt: int32(b.PriceBdt)})
			if err != nil {
				return err
			}
		}
		for i, e := range deals.Preorders {
			if err := q.UpsertSeedPreorder(ctx, sqlc.UpsertSeedPreorderParams{EditionID: e, Position: int32(i)}); err != nil {
				return err
			}
		}
		for _, s := range shares {
			if err := q.UpsertSeedShare(ctx, sqlc.UpsertSeedShareParams{ID: s.ID, UserID: s.OwnerID, OwnerName: s.OwnerName}); err != nil {
				return err
			}
			for i, book := range s.BookIDs { // the first book is the newest
				err := q.UpsertSeedWishlistItem(ctx, sqlc.UpsertSeedWishlistItemParams{UserID: s.OwnerID, BookID: book, AddedAt: now.Add(-time.Duration(i) * time.Minute)})
				if err != nil {
					return err
				}
			}
		}
		return nil
	})
}

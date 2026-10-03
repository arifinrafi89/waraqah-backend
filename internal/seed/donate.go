package seed

import (
	"context"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
)

// loadDonate loads the verified donation places. A place already there is left alone, so the
// copies donors sent since are kept.
func loadDonate(ctx context.Context, r *Run) error {
	var places []struct {
		ID, Name, Kind, District, Area, Story string
		Needs                                 []struct {
			BookID           string
			Wanted, Received int
		}
	}
	if err := r.read("donate_places.json", &places); err != nil {
		return err
	}
	return r.DB.WithTx(ctx, func(q *sqlc.Queries) error {
		for _, p := range places {
			if exists, err := q.SeedDonatePlaceExists(ctx, p.ID); err != nil {
				return err
			} else if exists {
				continue
			}
			err := q.SeedDonatePlace(ctx, sqlc.SeedDonatePlaceParams{ID: p.ID, Name: p.Name, Kind: p.Kind, District: p.District, Area: p.Area, Story: p.Story})
			if err != nil {
				return err
			}
			for i, n := range p.Needs {
				err := q.SeedDonateNeed(ctx, sqlc.SeedDonateNeedParams{PlaceID: p.ID, Position: int32(i), BookID: n.BookID, Wanted: int32(n.Wanted), Received: int32(n.Received)})
				if err != nil {
					return err
				}
			}
		}
		return nil
	})
}

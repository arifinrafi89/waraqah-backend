package seed

import (
	"context"
	"time"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
)

// loadDashboard loads the demo search counts of search_log.dart, so the dashboard's top searches
// are not empty on a fresh start. They are dated the day of seeding; a term already there that day
// is left alone.
func loadDashboard(ctx context.Context, r *Run) error {
	var d struct{ Searches map[string]int }
	if err := r.read("dashboard.json", &d); err != nil {
		return err
	}
	now := time.Now().In(r.Loc)
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	return r.DB.WithTx(ctx, func(q *sqlc.Queries) error {
		for term, n := range d.Searches {
			if err := q.SeedSearch(ctx, sqlc.SeedSearchParams{Term: term, Day: day, Count: int32(n)}); err != nil {
				return err
			}
		}
		return nil
	})
}

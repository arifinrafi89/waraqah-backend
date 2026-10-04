package seed

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
)

// coverURL is the Open Library cover of a book (https://openlibrary.org/dev/docs/api/covers),
// used for this non-commercial project with credit.
func coverURL(id int) string { return fmt.Sprintf("https://covers.openlibrary.org/b/id/%d-L.jpg", id) }

// loadCovers sets the cover picture of the books and of the seed listings that have none. Books and
// listings that are not named stay with the generated cover.
func loadCovers(ctx context.Context, r *Run) error {
	var c struct {
		Books           map[string]int
		ListingsByTitle map[string]int
	}
	if err := r.read("covers.json", &c); err != nil {
		return err
	}
	return r.DB.WithTx(ctx, func(q *sqlc.Queries) error {
		for id, n := range c.Books {
			if err := q.SetBookCover(ctx, sqlc.SetBookCoverParams{ID: id, CoverUrl: pgtype.Text{String: coverURL(n), Valid: true}}); err != nil {
				return err
			}
		}
		for title, n := range c.ListingsByTitle {
			if err := q.SetListingCoverByTitle(ctx, sqlc.SetListingCoverByTitleParams{Title: title, CoverUrl: pgtype.Text{String: coverURL(n), Valid: true}}); err != nil {
				return err
			}
		}
		return nil
	})
}

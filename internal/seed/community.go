package seed

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
)

// loadCommunity loads the demo feed (bite_fixtures.dart: Bites, likes and comments), the reviews
// (review_seed.dart), who follows whom (follow_fake_store.dart, p-rafi's private profile) and the
// demo reader's shelves, reading days and goal (shelf_seed.dart). Rows already there are left alone.
func loadCommunity(ctx context.Context, r *Run) error {
	var d struct {
		Bites []struct {
			ID, AuthorID, Text string
			AgeHours           int
			BookID             *string
			Spoiler            bool
			Likes              []string
		}
		Comments []struct {
			ID, BiteID, AuthorID, Text string
			ParentID                   *string
		}
		Reviews []struct {
			ID, BookID, AuthorID, Text string
			Stars, AgeDays             int
			Verified                   bool
		}
		Follows         [][2]string
		PrivateProfiles []string
		Shelves         struct {
			UserID  string
			Entries []struct {
				BookID, Shelf          string
				AddedDaysAgo, Progress int
				FinishedDaysAgo        *int
				PagesRead, TotalPages  *int
			}
			ReadingDaysAgo []int
			Goal           int
		}
	}
	if err := r.read("community.json", &d); err != nil {
		return err
	}
	now := time.Now()
	daysAgo := func(n int) time.Time { return now.AddDate(0, 0, -n) }
	int4 := func(p *int) pgtype.Int4 {
		if p == nil {
			return pgtype.Int4{}
		}
		return pgtype.Int4{Int32: int32(*p), Valid: true}
	}
	return r.DB.WithTx(ctx, func(q *sqlc.Queries) error {
		for _, b := range d.Bites {
			err := q.SeedBite(ctx, sqlc.SeedBiteParams{ID: b.ID, AuthorID: r.who(b.AuthorID), Text: b.Text, BookID: optText(b.BookID),
				Spoiler: b.Spoiler, CreatedAt: now.Add(-time.Duration(b.AgeHours) * time.Hour)})
			if err != nil {
				return err
			}
			for _, u := range b.Likes {
				if err := q.SeedLike(ctx, sqlc.SeedLikeParams{BiteID: b.ID, UserID: r.who(u)}); err != nil {
					return err
				}
			}
		}
		for _, c := range d.Comments {
			err := q.SeedComment(ctx, sqlc.SeedCommentParams{ID: c.ID, BiteID: c.BiteID, ParentID: optText(c.ParentID), AuthorID: r.who(c.AuthorID),
				Text: c.Text, CreatedAt: now.Add(-time.Hour)})
			if err != nil {
				return err
			}
		}
		for _, v := range d.Reviews {
			err := q.SeedReview(ctx, sqlc.SeedReviewParams{ID: v.ID, BookID: v.BookID, UserID: r.who(v.AuthorID), Stars: int32(v.Stars), Text: v.Text,
				SeedVerified: v.Verified, CreatedAt: daysAgo(v.AgeDays)})
			if err != nil {
				return err
			}
		}
		for _, f := range d.Follows {
			if err := q.SeedFollow(ctx, sqlc.SeedFollowParams{FollowerID: r.who(f[0]), FolloweeID: r.who(f[1]), At: now}); err != nil {
				return err
			}
		}
		for _, id := range d.PrivateProfiles {
			if err := q.SeedPrivateProfile(ctx, r.who(id)); err != nil {
				return err
			}
		}
		user := r.who(d.Shelves.UserID)
		for _, e := range d.Shelves.Entries {
			finished := pgtype.Timestamptz{}
			if e.FinishedDaysAgo != nil {
				finished = pgtype.Timestamptz{Time: daysAgo(*e.FinishedDaysAgo), Valid: true}
			}
			err := q.SeedShelfEntry(ctx, sqlc.SeedShelfEntryParams{UserID: user, BookID: e.BookID, Shelf: e.Shelf, AddedAt: daysAgo(e.AddedDaysAgo),
				FinishedAt: finished, Progress: int32(e.Progress), PagesRead: int4(e.PagesRead), TotalPages: int4(e.TotalPages)})
			if err != nil {
				return err
			}
		}
		local := now.In(r.Loc)
		for _, n := range d.Shelves.ReadingDaysAgo {
			day := time.Date(local.Year(), local.Month(), local.Day()-n, 0, 0, 0, 0, time.UTC)
			if err := q.AddReadingDay(ctx, sqlc.AddReadingDayParams{UserID: user, Day: day}); err != nil {
				return err
			}
		}
		return q.SeedReadingGoal(ctx, sqlc.SeedReadingGoalParams{UserID: user, Year: int32(local.Year()), Goal: int32(d.Shelves.Goal)})
	})
}

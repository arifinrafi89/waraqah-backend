package shelves

import (
	"context"
	"sort"
	"time"

	"github.com/arifinrafi89/waraqah-backend/internal/feature/catalog"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
)

// CategoryCount is CategoryCountModel.
type CategoryCount struct {
	NameEn string `json:"nameEn"`
	NameBn string `json:"nameBn"`
	Count  int    `json:"count"`
}

// Stats is ReadingStatsModel: the reader's year in books.
type Stats struct {
	Year             int             `json:"year"`
	Goal             *int            `json:"goal"`
	FinishedThisYear int             `json:"finishedThisYear"`
	StreakDays       int             `json:"streakDays"`
	ReadToday        bool            `json:"readToday"`
	PerMonth         []int           `json:"perMonth"`
	TopCategories    []CategoryCount `json:"topCategories"`
}

// Stats is ReadingLog.statsJson: this year's goal, the Books finished per month, the streak and
// the three Categories read most. A guest gets the empty year.
func (s *Service) Stats(ctx context.Context, userID string) (Stats, error) {
	now := s.Clock.Now().In(s.Loc)
	out := Stats{Year: now.Year(), PerMonth: make([]int, 12), TopCategories: []CategoryCount{}}
	if userID == "" {
		return out, nil
	}
	q := s.DB.Q()
	if goal, err := q.GetReadingGoal(ctx, sqlc.GetReadingGoalParams{UserID: userID, Year: int32(out.Year)}); err == nil {
		g := int(goal)
		out.Goal = &g
	} else if !isNoRows(err) {
		return out, err
	}
	snap, err := s.Catalog.Snapshot(ctx)
	if err != nil {
		return out, err
	}
	rows, err := q.ListShelf(ctx, userID)
	if err != nil {
		return out, err
	}
	counts, order := map[string]int{}, []string{}
	for _, r := range rows {
		if r.Shelf != Finished || !r.FinishedAt.Valid {
			continue
		}
		at := r.FinishedAt.Time.In(s.Loc)
		if at.Year() != out.Year {
			continue
		}
		out.PerMonth[at.Month()-1]++
		out.FinishedThisYear++
		if b, ok := snap.Book(r.BookID); ok {
			if counts[b.CategoryID] == 0 {
				order = append(order, b.CategoryID)
			}
			counts[b.CategoryID]++
		}
	}
	sort.SliceStable(order, func(i, j int) bool { return counts[order[i]] > counts[order[j]] })
	names := map[string]catalog.Category{}
	for _, c := range snap.Categories {
		names[c.ID] = c
	}
	for _, id := range order[:min(3, len(order))] {
		if c, ok := names[id]; ok {
			out.TopCategories = append(out.TopCategories, CategoryCount{NameEn: c.NameEn, NameBn: c.NameBn, Count: counts[id]})
		}
	}
	list, err := q.ListReadingDays(ctx, sqlc.ListReadingDaysParams{UserID: userID, Day: day(now).AddDate(-1, 0, -1)})
	if err != nil {
		return out, err
	}
	days := map[time.Time]bool{}
	for _, d := range list {
		days[day(d)] = true
	}
	out.StreakDays, out.ReadToday = Streak(days, now), days[day(now)]
	return out, nil
}

// SetGoal is ShelfFakeApi.goal: 1 to 365 books this year; answers the stats.
func (s *Service) SetGoal(ctx context.Context, userID string, goal int) (Stats, error) {
	if !GoalIsValid(goal) {
		return Stats{}, ErrGoalInvalid
	}
	year := s.Clock.Now().In(s.Loc).Year()
	if err := s.DB.Q().SetReadingGoal(ctx, sqlc.SetReadingGoalParams{UserID: userID, Year: int32(year), Goal: int32(goal)}); err != nil {
		return Stats{}, err
	}
	return s.Stats(ctx, userID)
}

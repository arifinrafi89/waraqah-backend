package dashboard

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/clock"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
)

// growWindow is how long a live search may keep growing and still count once.
const growWindow = 2 * time.Minute

// SearchLog is search.Log (BACKEND_PLAN.md section 8, port of search_log.dart): what readers search
// the catalog for, counted per term and day. Live search sends every keystroke, so a term that
// grows from (or shrinks back to) the same searcher's last one within two minutes replaces it.
type SearchLog struct {
	DB    *db.DB
	Clock clock.Clock
	Loc   *time.Location
	Log   *slog.Logger

	mu   sync.Mutex
	last map[string]lastTerm // by searcher: a user id, or the address of a guest
}

type lastTerm struct {
	term string
	day  time.Time
	at   time.Time
}

func dayOf(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// Record counts one search (SearchLog.record). Terms under three characters are not counted.
// Failures are logged, never shown to the reader.
func (l *SearchLog) Record(ctx context.Context, searcher, query string) {
	term := strings.ToLower(strings.TrimSpace(query))
	if len([]rune(term)) < 3 {
		return
	}
	now := l.Clock.Now()
	day := dayOf(now.In(l.Loc))
	l.mu.Lock()
	if l.last == nil {
		l.last = map[string]lastTerm{}
	}
	prev, had := l.last[searcher]
	l.last[searcher] = lastTerm{term: term, day: day, at: now}
	if len(l.last) > 10000 { // forget old searchers so the map cannot grow without end
		for k, v := range l.last {
			if now.Sub(v.at) > growWindow {
				delete(l.last, k)
			}
		}
	}
	l.mu.Unlock()
	err := l.DB.WithTx(ctx, func(q *sqlc.Queries) error {
		if had && now.Sub(prev.at) <= growWindow && prev.term != term &&
			(strings.HasPrefix(term, prev.term) || strings.HasPrefix(prev.term, term)) {
			if err := q.DropSearch(ctx, sqlc.DropSearchParams{Term: prev.term, Day: prev.day}); err != nil {
				return err
			}
		}
		if had && now.Sub(prev.at) <= growWindow && prev.term == term {
			return nil // the same search sent again (a filter changed) counts once
		}
		return q.BumpSearch(ctx, sqlc.BumpSearchParams{Term: term, Day: day})
	})
	if err != nil {
		l.Log.Warn("search log failed", "error", err)
	}
}

// Search is one line of the top searches.
type Search struct {
	Term  string `json:"term"`
	Count int    `json:"count"`
}

// Top is SearchLog.top: the most searched terms, most first.
func (l *SearchLog) Top(ctx context.Context, n int) ([]Search, error) {
	rows, err := l.DB.Q().TopSearches(ctx, int32(n))
	if err != nil {
		return nil, err
	}
	out := make([]Search, 0, len(rows))
	for _, r := range rows {
		out = append(out, Search{Term: r.Term, Count: int(r.Count)})
	}
	return out, nil
}

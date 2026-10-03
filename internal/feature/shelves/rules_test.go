package shelves

import (
	"testing"
	"time"
)

func ip(v int) *int { return &v }

// Ported from test/reading_stats_test.dart ("progress is a percentage or pages; streaks count back").
func TestProgressRules(t *testing.T) {
	if got := PercentOf(155, 310); got != 50 {
		t.Errorf("percentOf(155, 310) = %d", got)
	}
	cases := []struct {
		u    Update
		want Problem
	}{
		{Update{Percent: 101}, ProblemBadPercent},
		{Update{Percent: 0, PagesRead: ip(20), TotalPages: ip(10)}, ProblemBadPages},
		{Update{Percent: 0, PagesRead: ip(5)}, ProblemBadPages},
		{Update{Percent: 40, PagesRead: ip(4), TotalPages: ip(10)}, ProblemNone},
		{Update{Percent: 10, PagesRead: ip(1), TotalPages: ip(MaxPages + 1)}, ProblemBadPages},
	}
	for i, c := range cases {
		if got := Check(c.u); got != c.want {
			t.Errorf("case %d: %q, want %q", i, got, c.want)
		}
	}
	d := func(m time.Month, dd int) time.Time { return time.Date(2026, m, dd, 0, 0, 0, 0, time.UTC) }
	now := time.Date(2026, 10, 3, 9, 0, 0, 0, time.UTC)
	days := map[time.Time]bool{d(10, 2): true, d(10, 1): true}
	if got := Streak(days, now); got != 2 {
		t.Errorf("streak ending yesterday = %d", got)
	}
	days[d(10, 3)] = true
	if got := Streak(days, now); got != 3 {
		t.Errorf("streak ending today = %d", got)
	}
	if got := Streak(map[time.Time]bool{d(9, 30): true}, now); got != 0 {
		t.Errorf("broken streak = %d", got)
	}
	if GoalIsValid(0) || !GoalIsValid(1) || !GoalIsValid(365) || GoalIsValid(366) {
		t.Error("goal is 1 to 365")
	}
}

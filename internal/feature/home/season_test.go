package home

import (
	"testing"
	"time"
)

func date(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

// Ported from test/season_picker_test.dart.
func TestOneSeasonByDateRamadanFirst(t *testing.T) {
	cases := []struct {
		when time.Time
		want Season
	}{
		{date(2027, 2, 10), Ramadan}, // beats Boi Mela
		{date(2026, 3, 19), Ramadan}, // last day counts
		{date(2026, 2, 18), Ramadan}, // first day counts
		{date(2026, 3, 20), ""},      // the day after
		{date(2026, 2, 5), BoiMela},
		{date(2026, 1, 10), BackToSchool},
		{date(2026, 11, 1), Admission},
		{date(2026, 6, 1), ""},
	}
	for _, c := range cases {
		if got := ActiveOn(c.when, ""); got != c.want {
			t.Errorf("%v: got %q want %q", c.when.Format("2006-01-02"), got, c.want)
		}
	}
}

func TestStaffOverrideBeatsTheDate(t *testing.T) {
	if got := ActiveOn(date(2026, 6, 1), BoiMela); got != BoiMela {
		t.Errorf("override: %q", got)
	}
	if got := ActiveOn(date(2026, 11, 1), ""); got != Admission {
		t.Errorf("automatic: %q", got)
	}
}

func TestEveryActiveSeasonHasAHeroCard(t *testing.T) {
	for _, s := range Seasons {
		if Info[s].CollectionID == "" || Info[s].Season != s {
			t.Errorf("season %s has no card", s)
		}
	}
}

package home_test

import (
	"testing"

	"github.com/arifinrafi89/waraqah-backend/internal/testenv"
)

func TestAyahRotatesByDayOfYear(t *testing.T) {
	e := testenv.New(t)
	got := e.Call("", "GET", "/islamic/ayah-of-the-day", nil).Obj(t)
	for _, k := range []string{"arabic", "translation", "surahEn", "surahBn", "surahNumber", "verseNumber"} {
		if got[k] == nil {
			t.Errorf("missing %s", k)
		}
	}
}

func TestSeasonByDateOverHTTP(t *testing.T) {
	e := testenv.New(t)
	cases := map[string]string{"2027-02-10": "ramadan", "2026-02-05": "boiMela", "2026-01-10": "backToSchool", "2026-11-01": "admission", "2026-06-01": ""}
	for date, want := range cases {
		r := e.Call("", "GET", "/home/season?date="+date, nil)
		if want == "" {
			if r.Body != nil {
				t.Errorf("%s: %s", date, r.Raw)
			}
			continue
		}
		if r.Obj(t)["season"] != want {
			t.Errorf("%s: %s", date, r.Raw)
		}
	}
	// banners: the active Season first, then the all-year ones, never another Season
	for _, b := range e.Call("", "GET", "/home/banners?date=2026-06-01", nil).List(t) {
		if b["season"] != nil {
			t.Errorf("a Season banner out of season: %v", b["id"])
		}
	}
}

package clock

import (
	"testing"
	"time"
)

func TestDhakaDayBoundary(t *testing.T) {
	loc, err := Location("Asia/Dhaka")
	if err != nil {
		t.Fatal(err)
	}
	// 20:00 UTC on 2 Oct is already 3 Oct in Dhaka (UTC+6).
	f := NewFake(time.Date(2026, 10, 2, 20, 0, 0, 0, time.UTC))
	if got := DayString(f.Now(), loc); got != "2026-10-03" {
		t.Errorf("DayString = %s", got)
	}
	today := Today(f, loc)
	if today.Hour() != 0 || today.Day() != 3 {
		t.Errorf("Today = %v", today)
	}
	f.Advance(5 * time.Hour)
	if Today(f, loc).Day() != 3 {
		t.Error("still the same Dhaka day")
	}
	f.Advance(20 * time.Hour)
	if Today(f, loc).Day() != 4 {
		t.Error("should be the next Dhaka day")
	}
	f.Set(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	if f.Now().Year() != 2026 {
		t.Error("Set failed")
	}
}

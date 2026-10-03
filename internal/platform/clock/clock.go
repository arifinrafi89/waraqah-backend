// Package clock is the time source. Services take a Clock so tests can set the time, and
// "today" is always the app timezone (Asia/Dhaka), not the server's.
package clock

import (
	"sync"
	"time"
	_ "time/tzdata" // the timezone database travels with the binary (Windows and slim images have none)
)

// Clock tells the time.
type Clock interface{ Now() time.Time }

// Real is the wall clock.
type Real struct{}

// Now returns the current time.
func (Real) Now() time.Time { return time.Now() }

// Fake is a settable clock for tests.
type Fake struct {
	mu sync.Mutex
	t  time.Time
}

// NewFake starts a fake clock at t.
func NewFake(t time.Time) *Fake { return &Fake{t: t} }

// Now returns the fake time.
func (f *Fake) Now() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.t
}

// Set moves the clock to t.
func (f *Fake) Set(t time.Time) {
	f.mu.Lock()
	f.t = t
	f.mu.Unlock()
}

// Advance moves the clock forward by d.
func (f *Fake) Advance(d time.Duration) {
	f.mu.Lock()
	f.t = f.t.Add(d)
	f.mu.Unlock()
}

// Location loads the app timezone (APP_TIMEZONE).
func Location(name string) (*time.Location, error) { return time.LoadLocation(name) }

// DayOf returns midnight (start of the day) of t in loc.
func DayOf(t time.Time, loc *time.Location) time.Time {
	y, m, d := t.In(loc).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, loc)
}

// Today returns the start of the current day in loc.
func Today(c Clock, loc *time.Location) time.Time { return DayOf(c.Now(), loc) }

// DayString formats the calendar day of t in loc as 2006-01-02.
func DayString(t time.Time, loc *time.Location) string { return t.In(loc).Format("2006-01-02") }

// FormatRFC3339 formats t as RFC 3339 with the offset of loc, for example 2026-10-03T14:05:00+06:00.
func FormatRFC3339(t time.Time, loc *time.Location) string { return t.In(loc).Format(time.RFC3339) }

package auth

import (
	"context"
	"testing"
	"time"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/db"
)

const testSecret = "0123456789abcdef0123456789abcdef-test"

type testClock struct{ now time.Time }

func (c *testClock) Now() time.Time          { return c.now }
func (c *testClock) Advance(d time.Duration) { c.now = c.now.Add(d) }

func newClock() *testClock { return &testClock{now: time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)} }

func addUser(t *testing.T, d *db.DB, id string, role Role, banned bool) {
	t.Helper()
	_, err := d.Conn.Exec(context.Background(),
		"INSERT INTO users (id, email, name, role, banned) VALUES ($1, $2, $1, $3, $4)",
		id, id+"@example.test", string(role), banned)
	if err != nil {
		t.Fatal(err)
	}
}

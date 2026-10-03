// Package ids makes the readable ids the app and the seed data use (BACKEND_PLAN.md section 4.3):
// u_<ulid>, p2p-..., th-..., WQ-100231, HS-201.
package ids

import (
	"context"
	"crypto/rand"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/db"
)

var (
	mu      sync.Mutex
	entropy = ulid.Monotonic(rand.Reader, 0)
)

func newULID() string {
	mu.Lock()
	defer mu.Unlock()
	return strings.ToLower(ulid.MustNew(ulid.Timestamp(time.Now()), entropy).String())
}

// New returns "<prefix>-<ulid>", for example New("p2p") = p2p-01j9....
func New(prefix string) string { return prefix + "-" + newULID() }

// User returns a new user id: u_<ulid>.
func User() string { return "u_" + newULID() }

// Prefixes of the ids the fake stores make, one function per kind so callers read well.
func Listing() string      { return New("p2p") }
func Thread() string       { return New("th") }
func Message() string      { return New("m") }
func Offer() string        { return New("o") }
func Request() string      { return New("rq") }
func Notification() string { return New("nt") }
func Alert() string        { return New("al") }
func Address() string      { return New("ad") }
func Bite() string         { return New("bt") }
func Comment() string      { return New("cm") }
func Question() string     { return New("q") }
func Answer() string       { return New("a") }
func Book() string         { return New("bk") }
func Place() string        { return New("rc") }
func Booklist() string     { return New("bl") }
func Sale() string         { return New("sb") }
func Certified() string    { return New("cu") }

// OrderNumber takes the next order number from the sequence: WQ-100231, WQ-100232, ...
func OrderNumber(ctx context.Context, d *db.DB) (string, error) {
	n, err := d.Q().NextOrderNumber(ctx)
	if err != nil {
		return "", fmt.Errorf("next order number: %w", err)
	}
	return fmt.Sprintf("WQ-%d", n), nil
}

// HandledSaleID takes the next handled sale id from the sequence: HS-201, HS-202, ...
func HandledSaleID(ctx context.Context, d *db.DB) (string, error) {
	n, err := d.Q().NextHandledSaleNumber(ctx)
	if err != nil {
		return "", fmt.Errorf("next handled sale id: %w", err)
	}
	return fmt.Sprintf("HS-%d", n), nil
}

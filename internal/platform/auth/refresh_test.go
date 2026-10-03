package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/dbtest"
)

func TestRefreshRotateAndReuse(t *testing.T) {
	d := dbtest.New(t)
	ctx := context.Background()
	clk := newClock()
	addUser(t, d, "u_r1", RoleReader, false)
	r := NewRefresh(d, 24*time.Hour, clk)

	t1, _, err := r.Issue(ctx, "u_r1")
	if err != nil {
		t.Fatal(err)
	}
	t2, uid, _, err := r.Rotate(ctx, t1)
	if err != nil || uid != "u_r1" || t2 == "" || t2 == t1 {
		t.Fatalf("rotate: %q %q %v", t2, uid, err)
	}
	// Reusing the revoked token is theft: it fails and kills the new token too.
	if _, _, _, err := r.Rotate(ctx, t1); !errors.Is(err, ErrInvalidRefresh) {
		t.Fatalf("reuse: %v", err)
	}
	if _, _, _, err := r.Rotate(ctx, t2); !errors.Is(err, ErrInvalidRefresh) {
		t.Fatalf("token issued before the theft still works: %v", err)
	}
}

func TestRefreshExpiredUnknownAndBanned(t *testing.T) {
	d := dbtest.New(t)
	ctx := context.Background()
	clk := newClock()
	addUser(t, d, "u_r2", RoleReader, false)
	addUser(t, d, "u_r3", RoleReader, true)
	r := NewRefresh(d, time.Hour, clk)

	tok, _, _ := r.Issue(ctx, "u_r2")
	clk.Advance(2 * time.Hour)
	if _, _, _, err := r.Rotate(ctx, tok); !errors.Is(err, ErrInvalidRefresh) {
		t.Errorf("expired: %v", err)
	}
	if _, _, _, err := r.Rotate(ctx, "not-a-token"); !errors.Is(err, ErrInvalidRefresh) {
		t.Errorf("unknown: %v", err)
	}
	clk.now = newClock().now
	bt, _, _ := r.Issue(ctx, "u_r3")
	if _, _, _, err := r.Rotate(ctx, bt); !errors.Is(err, ErrInvalidRefresh) {
		t.Errorf("banned: %v", err)
	}
}

func TestRefreshRevokeAndLogout(t *testing.T) {
	d := dbtest.New(t)
	ctx := context.Background()
	addUser(t, d, "u_r4", RoleReader, false)
	r := NewRefresh(d, time.Hour, newClock())
	a, _, _ := r.Issue(ctx, "u_r4")
	b, _, _ := r.Issue(ctx, "u_r4")
	if err := r.Revoke(ctx, a); err != nil {
		t.Fatal(err)
	}
	if err := r.Revoke(ctx, "unknown"); err != nil {
		t.Errorf("unknown token on logout must not fail: %v", err)
	}
	if _, _, _, err := r.Rotate(ctx, b); err != nil {
		t.Fatalf("other token should still work: %v", err)
	}
	if err := r.RevokeAll(ctx, "u_r4"); err != nil {
		t.Fatal(err)
	}
}

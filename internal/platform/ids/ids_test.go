package ids_test

import (
	"context"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/dbtest"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/ids"
)

func TestPrefixesAndUniqueness(t *testing.T) {
	if !regexp.MustCompile(`^u_[0-9a-z]{26}$`).MatchString(ids.User()) {
		t.Error("user id shape")
	}
	if !strings.HasPrefix(ids.Listing(), "p2p-") || !strings.HasPrefix(ids.Thread(), "th-") || !strings.HasPrefix(ids.Place(), "rc-") {
		t.Error("prefixes")
	}
	seen := map[string]bool{}
	for i := 0; i < 1000; i++ {
		id := ids.Message()
		if seen[id] {
			t.Fatalf("duplicate id %s", id)
		}
		seen[id] = true
	}
}

func TestSequencesCount(t *testing.T) {
	d := dbtest.New(t)
	ctx := context.Background()
	a, err := ids.OrderNumber(ctx, d)
	if err != nil || !strings.HasPrefix(a, "WQ-") {
		t.Fatalf("%q %v", a, err)
	}
	b, _ := ids.OrderNumber(ctx, d)
	na, _ := strconv.Atoi(strings.TrimPrefix(a, "WQ-"))
	nb, _ := strconv.Atoi(strings.TrimPrefix(b, "WQ-"))
	if na < 100231 || nb != na+1 {
		t.Errorf("order numbers %s %s", a, b)
	}
	h, err := ids.HandledSaleID(ctx, d)
	if err != nil || !strings.HasPrefix(h, "HS-") {
		t.Errorf("%q %v", h, err)
	}
}

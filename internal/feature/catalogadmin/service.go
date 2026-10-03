package catalogadmin

import (
	"context"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/clock"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/httpx"
)

// Refusal is a rule the request broke; the handler answers 200 null with its code.
type Refusal string

func (r Refusal) Error() string { return string(r) }

// Refusal codes (strings live in httpx/errors.go, documented in docs/error-codes.md).
const (
	ErrBookInvalid   = Refusal(httpx.ErrBookInvalid)
	ErrBookUnknown   = Refusal(httpx.ErrBookUnknown)
	ErrRecordInvalid = Refusal(httpx.ErrRecordInvalid)
	ErrRecordUnknown = Refusal(httpx.ErrRecordUnknown)
	ErrRecordInUse   = Refusal(httpx.ErrRecordInUse)
	ErrBannerInvalid = Refusal(httpx.ErrBannerInvalid)
	ErrBannerUnknown = Refusal(httpx.ErrBannerUnknown)
	ErrListInvalid   = Refusal(httpx.ErrListInvalid)
	ErrListUnknown   = Refusal(httpx.ErrListUnknown)
	ErrStockInvalid  = Refusal(httpx.ErrStockInvalid)
	ErrSeasonUnknown = Refusal(httpx.ErrSeasonUnknown)
)

// Cache is the catalog read cache; every change to the catalog tables drops it.
type Cache interface{ Invalidate() }

// Sweeper checks price and stock alerts after a change (alerts.Sweeper, BACKEND_PLAN.md section 8).
type Sweeper interface {
	Sweep(ctx context.Context) error
}

// NoSweeper is the stand-in until alerts (T11) plugs in.
type NoSweeper struct{}

// Sweep does nothing.
func (NoSweeper) Sweep(context.Context) error { return nil }

// Service holds the Admin → Catalog rules.
type Service struct {
	DB      *db.DB
	Cache   Cache
	Sweeper Sweeper
	Clock   clock.Clock
	Loc     *time.Location
	Log     *slog.Logger
}

var nonSlug = regexp.MustCompile("[^a-z0-9]+")
var edgeDash = regexp.MustCompile("^-+|-+$")

// uniqueID is `<prefix>-<slug of text>`, with -2, -3... when taken has it (unique_id.dart).
func uniqueID(prefix, text string, taken []string) string {
	slug := edgeDash.ReplaceAllString(nonSlug.ReplaceAllString(strings.ToLower(text), "-"), "")
	if slug == "" {
		slug = "new"
	}
	base := prefix + "-" + slug
	used := map[string]bool{}
	for _, t := range taken {
		used[t] = true
	}
	id := base
	for n := 2; used[id]; n++ {
		id = base + "-" + itoa(n)
	}
	return id
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for ; n > 0; n /= 10 {
		b = append([]byte{byte('0' + n%10)}, b...)
	}
	return string(b)
}

// changed runs after a commit that touched prices, stock or the catalog lists.
func (s *Service) changed(ctx context.Context, alerts bool) {
	s.Cache.Invalidate()
	if alerts {
		if err := s.Sweeper.Sweep(ctx); err != nil {
			s.Log.Error("alert sweep failed", "error", err)
		}
	}
}

// Package seed loads seed/*.json (exported from the frontend fixtures by `make contract-export`)
// into Postgres. Every step is an idempotent upsert, run in dependency order, so running it
// twice changes nothing (BACKEND_PLAN.md section 14).
package seed

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/auth"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db"
)

// Options configure a seeding run.
type Options struct {
	Dir          string // the seed/ folder
	DemoPassword string // SEED_DEMO_PASSWORD, given to every demo account
	BcryptCost   int
	Loc          *time.Location // APP_TIMEZONE: how the exporter naive dates are read
	Log          *slog.Logger
}

// Run is what each step receives.
type Run struct {
	Options
	DB     *db.DB
	pwHash string
	// Me is the id of the account the fake API calls "me" (reader@waraqah.test).
	Me string
}

// MeID is the seeded demo reader. Fake data that mentions "me" is loaded as this account.
const MeID = "u_reader"

type step struct {
	name string
	fn   func(ctx context.Context, r *Run) error
}

// steps run in this order. Each feature adds its loader here, after the tables it needs.
var steps = []step{
	{"users", loadUsers},
	{"addresses", loadAddresses},
	{"notifications", loadNotifications},
	{"catalog records", loadCatalogRecords},
	{"books", loadBooks},
	{"book extras", loadBookExtras},
	{"collections", loadCollections},
	{"home", loadHome},
	{"people", loadPeople},
	{"buying", loadBuying},
	{"orders", loadOrders},
	{"donate", loadDonate},
	{"marketplace", loadMarketplace},
	{"inbox", loadInbox},
	{"sales", loadSales},
	{"community", loadCommunity},
}

// All runs every step.
func All(ctx context.Context, d *db.DB, opts Options) error {
	if opts.Loc == nil {
		opts.Loc = time.FixedZone("Asia/Dhaka", 6*3600)
	}
	r := &Run{Options: opts, DB: d, Me: MeID}
	for _, s := range steps {
		opts.Log.Info("seeding", "step", s.name)
		if err := s.fn(ctx, r); err != nil {
			return fmt.Errorf("seed %s: %w", s.name, err)
		}
	}
	return nil
}

// hash is the bcrypt hash of the demo password, made once per run.
func (r *Run) hash() (string, error) {
	if r.pwHash != "" {
		return r.pwHash, nil
	}
	if r.DemoPassword == "" {
		return "", errors.New("SEED_DEMO_PASSWORD is empty")
	}
	h, err := auth.HashPassword(r.DemoPassword, r.BcryptCost)
	r.pwHash = h
	return h, err
}

// read decodes seed/<name>.json into v. A missing file is an error, so a stale checkout is noticed.
func (r *Run) read(name string, v any) error {
	b, err := os.ReadFile(filepath.Join(r.Dir, name))
	if err != nil {
		return fmt.Errorf("read %s: %w (run make contract-export)", name, err)
	}
	return json.Unmarshal(b, v)
}

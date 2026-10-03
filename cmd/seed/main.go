// Command seed loads seed/*.json into Postgres. It is idempotent and refuses a production
// database unless SEED_ALLOW_PRODUCTION=true.
package main

import (
	"context"
	"flag"
	"log/slog"
	"os"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/clock"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/config"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/logx"
	"github.com/arifinrafi89/waraqah-backend/internal/seed"
)

func main() {
	dir := flag.String("dir", "seed", "folder with the seed JSON files")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("cannot start", "error", err)
		os.Exit(1)
	}
	log := logx.New(os.Stderr, cfg.LogLevel, cfg.LogFormat)
	if cfg.IsProduction() && !cfg.SeedAllowProduction {
		log.Error("refusing to seed a production database; set SEED_ALLOW_PRODUCTION=true to override")
		os.Exit(1)
	}
	ctx := context.Background()
	url := cfg.DatabaseURLDirect
	if url == "" {
		url = cfg.DatabaseURL
	}
	if err := db.Migrate(ctx, url, "up"); err != nil {
		log.Error("migrate failed", "error", err)
		os.Exit(1)
	}
	database, err := db.Open(ctx, cfg.DatabaseURL, 2, false, log)
	if err != nil {
		log.Error("cannot open database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	loc, err := clock.Location(cfg.AppTimezone)
	if err != nil {
		log.Error("APP_TIMEZONE", "error", err)
		os.Exit(1)
	}
	err = seed.All(ctx, database, seed.Options{
		Dir: *dir, DemoPassword: cfg.SeedDemoPassword, BcryptCost: cfg.BcryptCost, Loc: loc, Log: log,
	})
	if err != nil {
		log.Error("seed failed", "error", err)
		os.Exit(1)
	}
	log.Info("seed complete")
}

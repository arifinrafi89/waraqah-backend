// Command api is the Waraqah HTTP server: config, logger, database, services, routes, server.
package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/arifinrafi89/waraqah-backend/internal/app"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/config"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/logx"
)

func main() {
	migrate := flag.String("migrate", "", "run a migration command (up, down, down-up, status) and exit")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("cannot start", "error", err)
		os.Exit(1)
	}
	log := logx.New(os.Stderr, cfg.LogLevel, cfg.LogFormat)
	slog.SetDefault(log)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	migrateURL := cfg.DatabaseURLDirect
	if migrateURL == "" {
		migrateURL = cfg.DatabaseURL
	}
	if *migrate != "" {
		if err := db.Migrate(ctx, migrateURL, *migrate); err != nil {
			log.Error("migrate failed", "error", err)
			os.Exit(1)
		}
		return
	}
	if cfg.RunMigrationsOnStart {
		if err := db.Migrate(ctx, migrateURL, "up"); err != nil {
			log.Error("migrate on start failed", "error", err)
			os.Exit(1)
		}
	}

	database, err := db.Open(ctx, cfg.DatabaseURL, cfg.DBMaxConns, cfg.DebugSQL, log)
	if err != nil {
		log.Error("cannot open database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	deps := &app.Deps{Cfg: cfg, Log: log, DB: database, Ready: database.Ping}
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           app.Routes(deps),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
		// No WriteTimeout: server-sent events keep responses open.
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	log.Info("listening", "port", cfg.Port, "env", cfg.AppEnv)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

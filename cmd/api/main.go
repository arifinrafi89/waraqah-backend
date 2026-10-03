// Command api is the Waraqah HTTP server: config, logger, services, routes, server.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/arifinrafi89/waraqah-backend/internal/app"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/config"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/logx"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("cannot start", "error", err)
		os.Exit(1)
	}
	log := logx.New(os.Stderr, cfg.LogLevel, cfg.LogFormat)
	slog.SetDefault(log)

	deps := &app.Deps{Cfg: cfg, Log: log}
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           app.Routes(deps),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
		// No WriteTimeout: server-sent events keep responses open.
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
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

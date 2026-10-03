package db

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" database/sql driver
	"github.com/pressly/goose/v3"

	migrations "github.com/arifinrafi89/waraqah-backend/db"
)

// advisoryLockID serialises concurrent migrators (tests in several packages, two app instances).
const advisoryLockID = 727_001

// Migrate runs a goose command ("up", "down", "down-up", "status") against url.
func Migrate(ctx context.Context, url, command string) error {
	conn, err := sql.Open("pgx", url)
	if err != nil {
		return fmt.Errorf("open for migrate: %w", err)
	}
	defer func() { _ = conn.Close() }()
	// A single connection keeps the session-level advisory lock and goose on the same session.
	conn.SetMaxOpenConns(1)
	if _, err := conn.ExecContext(ctx, "SELECT pg_advisory_lock($1)", advisoryLockID); err != nil {
		return fmt.Errorf("migration lock: %w", err)
	}
	defer func() { _, _ = conn.ExecContext(context.Background(), "SELECT pg_advisory_unlock($1)", advisoryLockID) }()

	goose.SetBaseFS(migrations.Migrations)
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	switch command {
	case "up":
		return goose.UpContext(ctx, conn, "migrations")
	case "down":
		return goose.DownContext(ctx, conn, "migrations")
	case "down-up":
		if err := goose.DownContext(ctx, conn, "migrations"); err != nil {
			return err
		}
		return goose.UpContext(ctx, conn, "migrations")
	case "status":
		return goose.StatusContext(ctx, conn, "migrations")
	}
	return fmt.Errorf("unknown migrate command %q (up, down, down-up, status)", command)
}

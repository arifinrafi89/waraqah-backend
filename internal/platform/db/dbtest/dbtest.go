// Package dbtest gives tests a migrated database in a transaction that is rolled back at the end.
package dbtest

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/db"
)

var (
	once    sync.Once
	pool    *pgxpool.Pool
	initErr error
)

func setup(url string) {
	ctx := context.Background()
	if initErr = db.Migrate(ctx, url, "up"); initErr != nil {
		return
	}
	pool, initErr = pgxpool.New(ctx, url)
}

// New returns a DB bound to one transaction. Anything the test writes is undone at cleanup,
// and WithTx inside the test uses savepoints. The test is skipped when DATABASE_URL_TEST is unset.
func New(t testing.TB) *db.DB {
	t.Helper()
	loadDotEnv()
	url := os.Getenv("DATABASE_URL_TEST")
	if url == "" {
		t.Skip("DATABASE_URL_TEST is not set")
	}
	once.Do(func() { setup(url) })
	if initErr != nil {
		t.Fatalf("test database: %v", initErr)
	}
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	t.Cleanup(func() { _ = tx.Rollback(context.Background()) })
	return db.FromConn(tx)
}

// loadDotEnv reads the repo-root .env (found by walking up to go.mod) so a local
// `go test ./...` works without exporting variables. Real environment variables win.
func loadDotEnv() {
	dir, err := os.Getwd()
	if err != nil {
		return
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			_ = godotenv.Load(filepath.Join(dir, ".env"))
			return
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return
		}
		dir = parent
	}
}

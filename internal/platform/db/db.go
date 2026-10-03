// Package db is the pgx pool, the transaction helper and the migration runner.
package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
)

// Conn is what the stores run on: a pool in production, a transaction in tests.
// pgx.Tx can Begin a nested (savepoint) transaction, so WithTx works on both.
type Conn interface {
	sqlc.DBTX
	Begin(ctx context.Context) (pgx.Tx, error)
}

// DB bundles a connection with the generated queries.
type DB struct {
	Conn Conn
	pool *pgxpool.Pool
}

// New wraps a pool.
func New(pool *pgxpool.Pool) *DB { return &DB{Conn: pool, pool: pool} }

// FromConn wraps any Conn (used by dbtest with a rolled-back transaction).
func FromConn(c Conn) *DB { return &DB{Conn: c} }

// Q returns the queries bound to the current connection (no transaction).
func (d *DB) Q() *sqlc.Queries { return sqlc.New(d.Conn) }

// InTx runs fn in one transaction: commit when it returns nil, roll back otherwise.
func (d *DB) InTx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	tx, err := d.Conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// WithTx is InTx with the generated queries bound to the transaction.
func (d *DB) WithTx(ctx context.Context, fn func(q *sqlc.Queries) error) error {
	return d.InTx(ctx, func(tx pgx.Tx) error { return fn(sqlc.New(tx)) })
}

// Ping checks the database answers (used by /readyz).
func (d *DB) Ping(ctx context.Context) error {
	if d.pool != nil {
		return d.pool.Ping(ctx)
	}
	_, err := d.Conn.Exec(ctx, "SELECT 1")
	return err
}

// Close closes the pool, if there is one.
func (d *DB) Close() {
	if d.pool != nil {
		d.pool.Close()
	}
}

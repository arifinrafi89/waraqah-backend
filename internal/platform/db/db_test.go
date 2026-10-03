package db_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/db"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/dbtest"
)

func users(t *testing.T, d *db.DB) int {
	t.Helper()
	var n int
	if err := d.Conn.QueryRow(context.Background(), "SELECT count(*) FROM users WHERE id LIKE 'dbtest-%'").Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func insert(ctx context.Context, tx pgx.Tx, id string) error {
	_, err := tx.Exec(ctx, "INSERT INTO users (id, email) VALUES ($1, $2)", id, id+"@example.test")
	return err
}

func TestInTxCommitsAndRollsBack(t *testing.T) {
	d := dbtest.New(t)
	ctx := context.Background()
	if err := d.InTx(ctx, func(tx pgx.Tx) error { return insert(ctx, tx, "dbtest-a") }); err != nil {
		t.Fatal(err)
	}
	boom := errors.New("boom")
	err := d.InTx(ctx, func(tx pgx.Tx) error {
		if e := insert(ctx, tx, "dbtest-b"); e != nil {
			return e
		}
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("got %v", err)
	}
	if n := users(t, d); n != 1 {
		t.Fatalf("want only the committed row, have %d", n)
	}
}

func TestGeneratedQueryAndSequences(t *testing.T) {
	d := dbtest.New(t)
	ctx := context.Background()
	if _, err := d.Q().GetUserByEmail(ctx, pgtype.Text{String: "nobody@example.test", Valid: true}); err != pgx.ErrNoRows {
		t.Fatalf("got %v", err)
	}
	var n int64
	if err := d.Conn.QueryRow(ctx, "SELECT nextval('order_number_seq')").Scan(&n); err != nil || n < 100231 {
		t.Fatalf("order_number_seq: %d %v", n, err)
	}
}

func TestPing(t *testing.T) {
	if err := dbtest.New(t).Ping(context.Background()); err != nil {
		t.Fatal(err)
	}
}

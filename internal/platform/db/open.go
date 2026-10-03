package db

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Open builds a pool from the URL. With debugSQL it logs each statement and its timing
// (never the argument values).
func Open(ctx context.Context, url string, maxConns int, debugSQL bool, log *slog.Logger) (*DB, error) {
	if url == "" {
		return nil, fmt.Errorf("DATABASE_URL is empty")
	}
	pc, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("parse DATABASE_URL: %w", err)
	}
	if maxConns > 0 {
		pc.MaxConns = int32(maxConns)
	}
	if debugSQL {
		pc.ConnConfig.Tracer = sqlTracer{log: log}
	}
	pool, err := pgxpool.NewWithConfig(ctx, pc)
	if err != nil {
		return nil, fmt.Errorf("open pool: %w", err)
	}
	return New(pool), nil
}

type sqlTracer struct{ log *slog.Logger }

type queryStart struct {
	sql   string
	start time.Time
}

type queryKey struct{}

func (t sqlTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, d pgx.TraceQueryStartData) context.Context {
	return context.WithValue(ctx, queryKey{}, queryStart{sql: d.SQL, start: time.Now()})
}

func (t sqlTracer) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, d pgx.TraceQueryEndData) {
	q, ok := ctx.Value(queryKey{}).(queryStart)
	if !ok {
		return
	}
	attrs := []any{"sql", q.sql, "ms", time.Since(q.start).Milliseconds()}
	if d.Err != nil {
		attrs = append(attrs, "error", d.Err.Error())
	}
	t.log.Debug("sql", attrs...)
}

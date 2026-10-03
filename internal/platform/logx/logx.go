// Package logx sets up slog and carries a request-scoped logger in the context.
package logx

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"sync"
)

// New builds the process logger: JSON or text, at the given level.
func New(w io.Writer, level, format string) *slog.Logger {
	var lv slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lv = slog.LevelDebug
	case "warn":
		lv = slog.LevelWarn
	case "error":
		lv = slog.LevelError
	default:
		lv = slog.LevelInfo
	}
	opts := &slog.HandlerOptions{Level: lv}
	if strings.ToLower(format) == "json" {
		return slog.New(slog.NewJSONHandler(w, opts))
	}
	return slog.New(slog.NewTextHandler(w, opts))
}

type ctxKey struct{}

// reqLog is the per-request logger state. Attributes can be added after it is
// created (for example user_id once the token is read), so it is mutable.
type reqLog struct {
	base  *slog.Logger
	mu    sync.Mutex
	attrs []any
}

// WithRequest returns a context carrying a logger for one request.
func WithRequest(ctx context.Context, base *slog.Logger, attrs ...any) context.Context {
	return context.WithValue(ctx, ctxKey{}, &reqLog{base: base, attrs: attrs})
}

// Annotate adds attributes (user_id, role, ...) to every later log line of the request.
func Annotate(ctx context.Context, attrs ...any) {
	if rl, ok := ctx.Value(ctxKey{}).(*reqLog); ok {
		rl.mu.Lock()
		rl.attrs = append(rl.attrs, attrs...)
		rl.mu.Unlock()
	}
}

// From returns the request logger, or the default logger outside a request.
func From(ctx context.Context) *slog.Logger {
	rl, ok := ctx.Value(ctxKey{}).(*reqLog)
	if !ok {
		return slog.Default()
	}
	rl.mu.Lock()
	defer rl.mu.Unlock()
	return rl.base.With(rl.attrs...)
}

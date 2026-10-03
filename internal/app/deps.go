// Package app wires the server: it builds shared services and hands interfaces to features.
package app

import (
	"context"
	"log/slog"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/config"
)

// Deps is everything the features need. Cross-feature interfaces are built here (BACKEND_PLAN.md §8).
type Deps struct {
	Cfg *config.Config
	Log *slog.Logger
	// Ready reports whether the database answers (/readyz). Nil means always ready.
	Ready func(ctx context.Context) error
}

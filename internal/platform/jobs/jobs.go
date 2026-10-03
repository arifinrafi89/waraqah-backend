// Package jobs runs background work on a ticker (BACKEND_PLAN.md section 12). The free host
// sleeps when idle, so every job must be idempotent and catch up on wake.
package jobs

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// Job is one unit of background work.
type Job func(ctx context.Context) error

type named struct {
	name string
	fn   Job
}

// Runner calls its jobs one at a time on every tick.
type Runner struct {
	tick time.Duration
	log  *slog.Logger
	mu   sync.Mutex
	jobs []named
}

// New makes a runner that ticks every tick.
func New(tick time.Duration, log *slog.Logger) *Runner {
	return &Runner{tick: tick, log: log}
}

// Register adds a job. Call it before Run.
func (r *Runner) Register(name string, fn Job) {
	r.mu.Lock()
	r.jobs = append(r.jobs, named{name, fn})
	r.mu.Unlock()
}

// RunOnce runs every job once, in order. A failing or panicking job is logged and the rest still run.
func (r *Runner) RunOnce(ctx context.Context) {
	r.mu.Lock()
	jobs := append([]named(nil), r.jobs...)
	r.mu.Unlock()
	for _, j := range jobs {
		if ctx.Err() != nil {
			return
		}
		r.runJob(ctx, j)
	}
}

func (r *Runner) runJob(ctx context.Context, j named) {
	defer func() {
		if v := recover(); v != nil {
			r.log.Error("job panicked", "job", j.name, "panic", fmt.Sprint(v))
		}
	}()
	if err := j.fn(ctx); err != nil {
		r.log.Error("job failed", "job", j.name, "error", err)
	}
}

// Run runs the jobs once at start (to catch up after a sleep) and then on every tick, until ctx ends.
func (r *Runner) Run(ctx context.Context) {
	r.RunOnce(ctx)
	t := time.NewTicker(r.tick)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			r.RunOnce(ctx)
		}
	}
}

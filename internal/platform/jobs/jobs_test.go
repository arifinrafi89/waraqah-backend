package jobs

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"
)

func quiet() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func TestFailingAndPanickingJobsDoNotStopOthers(t *testing.T) {
	r := New(time.Hour, quiet())
	var ran atomic.Int32
	r.Register("fails", func(context.Context) error { return errors.New("nope") })
	r.Register("panics", func(context.Context) error { panic("boom") })
	r.Register("works", func(context.Context) error { ran.Add(1); return nil })
	r.RunOnce(context.Background())
	if ran.Load() != 1 {
		t.Errorf("ran = %d", ran.Load())
	}
}

func TestRunCatchesUpAtStartAndTicks(t *testing.T) {
	r := New(10*time.Millisecond, quiet())
	var n atomic.Int32
	r.Register("count", func(context.Context) error { n.Add(1); return nil })
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { r.Run(ctx); close(done) }()
	time.Sleep(80 * time.Millisecond)
	cancel()
	<-done
	if n.Load() < 3 {
		t.Errorf("job ran %d times, expected a start run plus ticks", n.Load())
	}
}

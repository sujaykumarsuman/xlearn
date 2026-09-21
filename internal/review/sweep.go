package review

import (
	"context"
	"log/slog"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/review/store"
)

// Sweeper is the periodic due-sweep worker (events.md flow 4, ADR-0004 hybrid): on a
// ~15-minute tick it materialises revision_item rows whose due_date has passed but
// which have not been surfaced, emitting revision_due per item. It is the offline-safe
// half of the scheduler — even after the learner is away for days, the sweep catches
// up every missed due date on the next tick (R-SR6). The work is idempotent
// (surfaced_at guards re-emission), so a missed or overlapping tick is harmless.
type Sweeper struct {
	store    store.Store
	log      *slog.Logger
	interval time.Duration
	batch    int
}

// NewSweeper builds the sweep worker. Call Run in a goroutine.
func (s *Service) NewSweeper(interval time.Duration, batch int) *Sweeper {
	if interval <= 0 {
		interval = 15 * time.Minute
	}
	if batch <= 0 {
		batch = 500
	}
	return &Sweeper{store: s.store, log: s.log, interval: interval, batch: batch}
}

// Run sweeps until ctx is cancelled. It sweeps once immediately (so a just-started
// pod catches up any backlog), then on each tick.
func (w *Sweeper) Run(ctx context.Context) {
	t := time.NewTicker(w.interval)
	defer t.Stop()
	w.sweep(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			w.sweep(ctx)
		}
	}
}

func (w *Sweeper) sweep(ctx context.Context) {
	n, err := w.store.Sweep(ctx, w.batch)
	if err != nil {
		w.log.Error("revision sweep failed", "err", err)
		return
	}
	if n > 0 {
		w.log.Info("revision sweep surfaced due touches", "count", n)
	}
}

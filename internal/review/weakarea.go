package review

import (
	"context"
	"log/slog"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/review/store"
)

// This file computes the weekly weak-area rollup (R-MJ3, ADR-0016). It runs on the
// same periodic tick as the S06 due-sweep: for each (account, course) with mistakes
// (m1-03: a weak area is per course, over mistake_entry.path_slug) it counts this
// week's open entries per category and upserts one snapshot, with the WEEK BOUNDARY
// defined in the account's timezone (resolved via the identity internal API) so a
// snapshot never straddles the wrong day for a user far from UTC. The upsert is
// idempotent per (course, week_of), so re-running the tick never double-counts.

// weakAreaStore is the store seam the recompute needs (the full store.Store satisfies it).
type weakAreaStore interface {
	MistakeScopes(ctx context.Context) ([]store.MistakeScope, error)
	CountOpenMistakesByCategory(ctx context.Context, accountID, pathSlug string, start, end time.Time) (map[string]int, error)
	SaveWeakAreaSnapshot(ctx context.Context, accountID, pathSlug string, weekOf time.Time, topCategory string, counts map[string]int) error
}

// accountResolver resolves an account's timezone + study budget + reminder prefs (the
// identity client in prod; nil-safe — a nil resolver falls back to UTC). The weak-area
// recompute uses only the timezone; the notifications worker uses all three (S10).
type accountResolver interface {
	ResolveAccount(ctx context.Context, accountID string) (timezone string, studyBudget, reminders []byte, err error)
}

// weakAreaComputer builds weak-area snapshots on the periodic tick.
type weakAreaComputer struct {
	store    weakAreaStore
	accounts accountResolver
	log      *slog.Logger
}

// WeakAreaWorker runs the weekly weak-area recompute on its own periodic loop —
// deliberately SEPARATE from the due-sweep goroutine (ADR-0016) so a slow identity or a
// large account count can never stall the core offline-safe due-sweep (R-SR6). Same
// cadence as the sweep by default; idempotent per week_of, so overlapping/missed ticks
// are harmless.
type WeakAreaWorker struct {
	computer *weakAreaComputer
	interval time.Duration
	log      *slog.Logger
}

// NewWeakAreaWorker builds the weak-area recompute worker. Call Run in a goroutine.
func (s *Service) NewWeakAreaWorker(interval time.Duration) *WeakAreaWorker {
	if interval <= 0 {
		interval = 15 * time.Minute
	}
	return &WeakAreaWorker{computer: s.weakArea, interval: interval, log: s.log}
}

// Run recomputes once immediately (fresh snapshots on pod start), then on each tick,
// until ctx is cancelled.
func (w *WeakAreaWorker) Run(ctx context.Context) {
	t := time.NewTicker(w.interval)
	defer t.Stop()
	w.recompute(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			w.recompute(ctx)
		}
	}
}

func (w *WeakAreaWorker) recompute(ctx context.Context) {
	if _, err := w.computer.Recompute(ctx); err != nil {
		w.log.Error("weak-area recompute failed", "err", err)
	}
}

// Recompute rebuilds the current-week weak-area snapshot of every (account, course)
// with mistakes. It never fails the whole run for one scope — an account whose timezone
// can't be resolved falls back to a UTC week window (logged, resolved once per account
// whatever its course count), and one scope's error is logged and skipped. Returns the
// number of snapshots written.
func (c *weakAreaComputer) Recompute(ctx context.Context) (int, error) {
	if c == nil || c.store == nil {
		return 0, nil
	}
	scopes, err := c.store.MistakeScopes(ctx)
	if err != nil {
		return 0, err
	}
	now := time.Now()
	done := 0
	tzOf := map[string]string{}
	for _, sc := range scopes {
		tz, seen := tzOf[sc.AccountID]
		if !seen {
			tz = "UTC"
			if c.accounts != nil {
				if resolved, _, _, rerr := c.accounts.ResolveAccount(ctx, sc.AccountID); rerr != nil {
					c.log.Warn("weak-area: timezone resolve failed; using UTC", "account_id", sc.AccountID, "err", rerr)
				} else if resolved != "" {
					tz = resolved
				}
			}
			tzOf[sc.AccountID] = tz
		}
		weekOf, start, end := weekWindow(now, tz)
		counts, err := c.store.CountOpenMistakesByCategory(ctx, sc.AccountID, sc.PathSlug, start, end)
		if err != nil {
			c.log.Error("weak-area: count failed; skipping", "account_id", sc.AccountID, "path_slug", sc.PathSlug, "err", err)
			continue
		}
		top := topCategory(counts)
		if err := c.store.SaveWeakAreaSnapshot(ctx, sc.AccountID, sc.PathSlug, weekOf, top, counts); err != nil {
			// Until m1-08 drops the v1 UNIQUE (account_id, week_of), a second course's
			// snapshot in the same week lands here (a unique violation) and is skipped.
			c.log.Error("weak-area: save snapshot failed; skipping", "account_id", sc.AccountID, "path_slug", sc.PathSlug, "err", err)
			continue
		}
		done++
	}
	if done > 0 {
		c.log.Info("weak-area snapshots recomputed", "snapshots", done)
	}
	return done, nil
}

// weekWindow returns, for the calendar week containing `now` in timezone tzName:
//   - weekOf: the Monday of that week as a UTC-midnight date (the snapshot key),
//   - start/end: the [Monday 00:00, next Monday 00:00) bounds IN tzName, as instants
//     to range-filter the UTC-stored created_at against.
//
// An unknown/empty timezone falls back to UTC. Weeks start Monday.
func weekWindow(now time.Time, tzName string) (weekOf, start, end time.Time) {
	loc, err := time.LoadLocation(tzName)
	if err != nil || loc == nil {
		loc = time.UTC
	}
	local := now.In(loc)
	// Go's Weekday: Sunday=0..Saturday=6. Days since Monday (Mon=0..Sun=6).
	daysSinceMonday := (int(local.Weekday()) + 6) % 7
	midnight := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
	monday := midnight.AddDate(0, 0, -daysSinceMonday)
	start = monday
	end = monday.AddDate(0, 0, 7)
	// The snapshot key is Monday's calendar date, stored tz-agnostically at UTC midnight.
	weekOf = time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, time.UTC)
	return weekOf, start, end
}

// topCategory returns the category with the highest count. Ties break by the canonical
// R-MJ2 order (deterministic across runs). Empty counts → "".
func topCategory(counts map[string]int) string {
	top := ""
	best := 0
	for _, cat := range store.MistakeCategories {
		if n, ok := counts[cat]; ok && n > best {
			best = n
			top = cat
		}
	}
	return top
}

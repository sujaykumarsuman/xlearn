// Package store is the review service's persistence layer: pgx/pgxpool over the
// sqlc-generated queries in ./gen, goose migrations in ./migrations, the
// transactional outbox and the idempotent inbox (ADR-0004/0005). It exposes small
// domain types with plain Go scalars so the HTTP/consumer layers never touch
// pgtype, keeps the xlearn_review role scoped to schema review (all SQL is
// schema-qualified), and owns the five-touch invariants — the Day 1/3/7/21/45
// ladder, the R-SR2 auto-score, and the fail→reset — inside single transactions
// that also append the outbox rows and record the consumed event in the inbox.
package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sujaykumarsuman/xlearn/internal/review/store/gen"
)

// The five spaced-repetition touches (R-SR1): touch_level 1..5 maps to a re-solve
// this many days after the anchor (first clean solve, or the reset date on a fail).
var touchDays = [MaxTouchLevel]int{1, 3, 7, 21, 45}

const (
	// MaxTouchLevel is the last touch (Day 45); passing it completes the ladder.
	MaxTouchLevel = 5

	// NamePatternMaxSecs is the R-SR2 pattern-naming threshold: a re-solve passes
	// only if the pattern is named in under two minutes.
	NamePatternMaxSecs = 120

	// OwedAttemptTouchLevel / OwedAttemptDays: revealing the solution early owes a
	// re-solve in 3 days (R-PF2), scheduled as the Day-3 (level 2) touch.
	OwedAttemptTouchLevel = 2
	OwedAttemptDays       = 3

	// Outcome value that gates the five-touch schedule (R-SR1: from first clean solve).
	OutcomeClean = "clean"
)

// Revision-item status values.
const (
	StatusPending = "pending"
	StatusPassed  = "passed"
	StatusFailed  = "failed"
)

// Event subjects + envelope version (events.md). Consumed practice subjects and the
// review subjects the relay publishes to XLEARN_REVIEW.
const (
	SubjectProblemSolved         = "xlearn.practice.problem_solved"
	SubjectSolutionRevealedEarly = "xlearn.practice.solution_revealed_early"

	SubjectRevisionScheduled = "xlearn.review.revision_scheduled"
	SubjectRevisionDue       = "xlearn.review.revision_due"

	eventVersion = 1
)

// Errors mapped to HTTP status by the handlers.
var (
	ErrNotFound = errors.New("review: not found")
)

// ScoreInput is one re-solve's auto-score inputs (R-SR2).
type ScoreInput struct {
	NamedPatternSecs int
	SolvedInTimer    bool
	StatedComplexity bool
}

// ScoreResult is the outcome of auto-scoring a re-solve: whether it passed, the new
// status, and the next scheduled touch (the advance on a pass, or Day 1 on a reset).
type ScoreResult struct {
	ItemID         string
	ProblemID      string
	TouchLevel     int // the touch that was scored
	AutoPass       bool
	MockMode       bool // Day 21 / Day 45 (R-SR4)
	Reset          bool // true when a fail reset the ladder to Day 1
	NextTouchLevel int  // the touch now due (0 = ladder complete after a Day-45 pass)
	NextDueDate    time.Time
}

// DueItem is one entry in the prioritised revision queue.
type DueItem struct {
	ItemID     string
	ProblemID  string
	TouchLevel int
	DueDate    time.Time
	Due        bool // due_date <= now (vs an upcoming "coming up" touch)
	MockMode   bool // Day 21 / Day 45
	Status     string
}

// OutboxRow is one unsent domain event awaiting relay to NATS.
type OutboxRow struct {
	EventID string
	Subject string
	Payload []byte
}

// Store is the review persistence seam. Handlers + consumers depend on this
// interface so they can be unit-tested against an in-memory fake.
type Store interface {
	// HandleProblemSolved reacts to xlearn.practice.problem_solved: on the first
	// clean solve it schedules the five touches (Day 1/3/7/21/45 from occurredAt) and
	// emits revision_scheduled per newly-scheduled touch, all in one transaction that
	// also records eventID in the inbox. A re-delivered event (eventID already in the
	// inbox) is a no-op. Returns the number of touches scheduled.
	HandleProblemSolved(ctx context.Context, eventID, accountID, problemID, outcome string, firstSolve bool, occurredAt time.Time) (int, error)
	// HandleSolutionRevealedEarly reacts to xlearn.practice.solution_revealed_early:
	// it schedules the owed re-solve 3 days out (R-PF2) as the Day-3 touch and emits
	// revision_scheduled, deduping on eventID. Returns 1 if newly scheduled, else 0.
	HandleSolutionRevealedEarly(ctx context.Context, eventID, accountID, problemID string, occurredAt time.Time) (int, error)
	// Score auto-scores a re-solve (R-SR2): records a touch_result, then on a pass
	// advances to the next touch (emitting revision_scheduled) or, on a fail, resets
	// the problem's ladder to Day 1 (R-SR3). One transaction.
	Score(ctx context.Context, accountID, itemID string, in ScoreInput) (ScoreResult, error)
	// DueQueue returns the account's prioritised queue (most-overdue first), capped at
	// limit, for the Revision screen.
	DueQueue(ctx context.Context, accountID string, limit int) ([]DueItem, error)
	// Sweep materialises due-but-unsurfaced touches across all accounts (flow 4):
	// it latches surfaced_at (idempotent) and emits revision_due per item, each in one
	// transaction. Returns the number of items surfaced this run.
	Sweep(ctx context.Context, batch int) (int, error)
	ListUnsentOutbox(ctx context.Context, limit int32) ([]OutboxRow, error)
	MarkOutboxSent(ctx context.Context, eventID string) error
	Ping(ctx context.Context) error
}

// PgStore is the pgxpool-backed Store.
type PgStore struct {
	pool *pgxpool.Pool
	q    *gen.Queries
}

// New wraps a pgxpool with the generated queries.
func New(pool *pgxpool.Pool) *PgStore {
	return &PgStore{pool: pool, q: gen.New(pool)}
}

var _ Store = (*PgStore)(nil)

// Ping verifies the database is reachable (drives /readyz).
func (s *PgStore) Ping(ctx context.Context) error { return s.pool.Ping(ctx) }

// HandleProblemSolved schedules the five-touch ladder on the first clean solve.
func (s *PgStore) HandleProblemSolved(ctx context.Context, eventID, accountID, problemID, outcome string, firstSolve bool, occurredAt time.Time) (int, error) {
	aid, err := parseUUID(accountID)
	if err != nil {
		return 0, fmt.Errorf("parse account id: %w", err)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := s.q.WithTx(tx)

	fresh, err := claimInbox(ctx, qtx, eventID)
	if err != nil {
		return 0, err
	}
	if !fresh {
		return 0, nil // duplicate delivery — already processed
	}

	// The five-touch ladder is scheduled from the FIRST CLEAN solve (R-SR1). A
	// below-clean or repeat solve is consumed (inbox recorded) but schedules nothing
	// here — mistake handling lands in S07.
	scheduled := 0
	if firstSolve && outcome == OutcomeClean {
		anchor := occurredAt
		for level := 1; level <= MaxTouchLevel; level++ {
			due := touchDueDate(level, anchor)
			_, err := qtx.ScheduleTouch(ctx, gen.ScheduleTouchParams{
				AccountID:  aid,
				ProblemID:  problemID,
				TouchLevel: int32(level),
				DueDate:    tsz(due),
			})
			switch {
			case err == nil:
				if eerr := emitRevisionScheduled(ctx, qtx, accountID, problemID, level, due); eerr != nil {
					return 0, eerr
				}
				scheduled++
			case errors.Is(err, pgx.ErrNoRows):
				// Touch already scheduled (out-of-order / partial re-delivery) — skip.
			default:
				return 0, fmt.Errorf("schedule touch %d: %w", level, err)
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit tx: %w", err)
	}
	return scheduled, nil
}

// HandleSolutionRevealedEarly schedules the owed 3-day re-solve (R-PF2).
func (s *PgStore) HandleSolutionRevealedEarly(ctx context.Context, eventID, accountID, problemID string, occurredAt time.Time) (int, error) {
	aid, err := parseUUID(accountID)
	if err != nil {
		return 0, fmt.Errorf("parse account id: %w", err)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := s.q.WithTx(tx)

	fresh, err := claimInbox(ctx, qtx, eventID)
	if err != nil {
		return 0, err
	}
	if !fresh {
		return 0, nil
	}

	// The owed attempt is the Day-3 touch, due 3 days from the reveal. ScheduleTouch
	// is idempotent (DO NOTHING): if the five-touch ladder already placed a Day-3
	// touch, this is a no-op and re-delivery never double-schedules.
	due := occurredAt.AddDate(0, 0, OwedAttemptDays)
	scheduled := 0
	_, err = qtx.ScheduleTouch(ctx, gen.ScheduleTouchParams{
		AccountID:  aid,
		ProblemID:  problemID,
		TouchLevel: OwedAttemptTouchLevel,
		DueDate:    tsz(due),
	})
	switch {
	case err == nil:
		if eerr := emitRevisionScheduled(ctx, qtx, accountID, problemID, OwedAttemptTouchLevel, due); eerr != nil {
			return 0, eerr
		}
		scheduled = 1
	case errors.Is(err, pgx.ErrNoRows):
		// Day-3 touch already exists — nothing to schedule.
	default:
		return 0, fmt.Errorf("schedule owed attempt: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit tx: %w", err)
	}
	return scheduled, nil
}

// Score auto-scores a re-solve and advances or resets the ladder.
func (s *PgStore) Score(ctx context.Context, accountID, itemID string, in ScoreInput) (ScoreResult, error) {
	aid, err := parseUUID(accountID)
	if err != nil {
		return ScoreResult{}, ErrNotFound
	}
	iid, err := parseUUID(itemID)
	if err != nil {
		return ScoreResult{}, ErrNotFound
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ScoreResult{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := s.q.WithTx(tx)

	item, err := qtx.GetRevisionItem(ctx, gen.GetRevisionItemParams{ID: iid, AccountID: aid})
	if errors.Is(err, pgx.ErrNoRows) {
		return ScoreResult{}, ErrNotFound
	}
	if err != nil {
		return ScoreResult{}, fmt.Errorf("get revision item: %w", err)
	}

	level := int(item.TouchLevel)
	mockMode := isMockTouch(level)

	// Idempotency guard: only a PENDING touch can be scored. Re-scoring a settled
	// (passed) touch — a double-submit or a replay — is a no-op that returns the prior
	// outcome, so it never writes a duplicate touch_result or re-emits
	// revision_scheduled with a fresh event_id (which would defeat event_id dedupe). A
	// fail re-anchors the touch back to pending in the same tx, so a legitimate later
	// re-solve of that level is still allowed.
	if item.Status != StatusPending {
		res, rerr := s.settledResult(ctx, qtx, aid, itemID, item)
		if rerr != nil {
			return ScoreResult{}, rerr
		}
		if err := tx.Commit(ctx); err != nil {
			return ScoreResult{}, fmt.Errorf("commit tx: %w", err)
		}
		return res, nil
	}

	autoPass := AutoPass(in)

	if err := qtx.InsertTouchResult(ctx, gen.InsertTouchResultParams{
		RevisionItemID:   item.ID,
		NamedPatternSecs: int32(in.NamedPatternSecs),
		SolvedInTimer:    in.SolvedInTimer,
		StatedComplexity: in.StatedComplexity,
		AutoPass:         autoPass,
		MockMode:         mockMode,
	}); err != nil {
		return ScoreResult{}, fmt.Errorf("insert touch result: %w", err)
	}

	res := ScoreResult{
		ItemID:     itemID,
		ProblemID:  item.ProblemID,
		TouchLevel: level,
		AutoPass:   autoPass,
		MockMode:   mockMode,
	}

	if autoPass {
		if _, err := qtx.MarkTouchPassed(ctx, item.ID); err != nil {
			return ScoreResult{}, fmt.Errorf("mark passed: %w", err)
		}
		// Advance: the next touch already exists (scheduled at first solve); emit
		// revision_scheduled for it. If it is somehow missing, create it now.
		if level < MaxTouchLevel {
			next := level + 1
			nextDue, err := s.ensureNextTouch(ctx, qtx, aid, item.ProblemID, next)
			if err != nil {
				return ScoreResult{}, err
			}
			if eerr := emitRevisionScheduled(ctx, qtx, accountID, item.ProblemID, next, nextDue); eerr != nil {
				return ScoreResult{}, eerr
			}
			res.NextTouchLevel = next
			res.NextDueDate = nextDue
		}
		// level == MaxTouchLevel: the ladder is complete (no further touch).
	} else {
		// Fail (R-SR3): reset the whole problem to Day 1. Re-anchor every touch to a
		// fresh Day 1/3/7/21/45 schedule from now (all pending, surfaced_at cleared),
		// so the problem re-progresses from the start. The failure is durably recorded
		// in touch_result above.
		now := time.Now()
		for l := 1; l <= MaxTouchLevel; l++ {
			due := touchDueDate(l, now)
			if _, err := qtx.ReanchorTouch(ctx, gen.ReanchorTouchParams{
				AccountID:  aid,
				ProblemID:  item.ProblemID,
				TouchLevel: int32(l),
				DueDate:    tsz(due),
			}); err != nil {
				return ScoreResult{}, fmt.Errorf("reanchor touch %d: %w", l, err)
			}
		}
		day1Due := touchDueDate(1, now)
		if eerr := emitRevisionScheduled(ctx, qtx, accountID, item.ProblemID, 1, day1Due); eerr != nil {
			return ScoreResult{}, eerr
		}
		res.Reset = true
		res.NextTouchLevel = 1
		res.NextDueDate = day1Due
	}

	if err := tx.Commit(ctx); err != nil {
		return ScoreResult{}, fmt.Errorf("commit tx: %w", err)
	}
	return res, nil
}

// ensureNextTouch returns the next touch's due date, scheduling it (anchored to now)
// if it does not already exist.
func (s *PgStore) ensureNextTouch(ctx context.Context, qtx *gen.Queries, aid pgtype.UUID, problemID string, level int) (time.Time, error) {
	touch, err := qtx.GetTouch(ctx, gen.GetTouchParams{AccountID: aid, ProblemID: problemID, TouchLevel: int32(level)})
	if err == nil {
		return touch.DueDate.Time, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, fmt.Errorf("get next touch: %w", err)
	}
	due := touchDueDate(level, time.Now())
	if _, err := qtx.ScheduleTouch(ctx, gen.ScheduleTouchParams{
		AccountID:  aid,
		ProblemID:  problemID,
		TouchLevel: int32(level),
		DueDate:    tsz(due),
	}); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, fmt.Errorf("schedule next touch: %w", err)
	}
	return due, nil
}

// settledResult reconstructs the score result for an already-settled touch (the
// idempotent re-score path) with no writes. A passed touch reports the next touch it
// had advanced to, if any — the ladder position is read, not re-created or re-emitted.
func (s *PgStore) settledResult(ctx context.Context, qtx *gen.Queries, aid pgtype.UUID, itemID string, item gen.ReviewRevisionItem) (ScoreResult, error) {
	level := int(item.TouchLevel)
	res := ScoreResult{
		ItemID:     itemID,
		ProblemID:  item.ProblemID,
		TouchLevel: level,
		AutoPass:   item.Status == StatusPassed,
		MockMode:   isMockTouch(level),
	}
	if item.Status == StatusPassed && level < MaxTouchLevel {
		next, err := qtx.GetTouch(ctx, gen.GetTouchParams{AccountID: aid, ProblemID: item.ProblemID, TouchLevel: int32(level + 1)})
		switch {
		case err == nil:
			res.NextTouchLevel = level + 1
			res.NextDueDate = next.DueDate.Time
		case errors.Is(err, pgx.ErrNoRows):
			// The next touch doesn't exist (edge case) — leave NextTouchLevel 0.
		default:
			return ScoreResult{}, fmt.Errorf("get next touch (settled): %w", err)
		}
	}
	return res, nil
}

// DueQueue returns the account's prioritised revision queue.
func (s *PgStore) DueQueue(ctx context.Context, accountID string, limit int) ([]DueItem, error) {
	aid, err := parseUUID(accountID)
	if err != nil {
		return nil, ErrNotFound
	}
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.q.ListActiveTouches(ctx, gen.ListActiveTouchesParams{AccountID: aid, Limit: int32(limit)})
	if err != nil {
		return nil, fmt.Errorf("list active touches: %w", err)
	}
	now := time.Now()
	out := make([]DueItem, 0, len(rows))
	for _, r := range rows {
		out = append(out, DueItem{
			ItemID:     uuidString(r.ID),
			ProblemID:  r.ProblemID,
			TouchLevel: int(r.TouchLevel),
			DueDate:    r.DueDate.Time,
			Due:        !r.DueDate.Time.After(now),
			MockMode:   isMockTouch(int(r.TouchLevel)),
			Status:     r.Status,
		})
	}
	return out, nil
}

// Sweep surfaces due-but-unsurfaced touches and emits revision_due per item.
func (s *PgStore) Sweep(ctx context.Context, batch int) (int, error) {
	if batch <= 0 {
		batch = 500
	}
	candidates, err := s.q.SweepDueCandidates(ctx, int32(batch))
	if err != nil {
		return 0, fmt.Errorf("sweep candidates: %w", err)
	}
	surfaced := 0
	for _, c := range candidates {
		emitted, err := s.surfaceOne(ctx, c.ID)
		if err != nil {
			return surfaced, err
		}
		if emitted {
			surfaced++
		}
	}
	return surfaced, nil
}

// surfaceOne latches surfaced_at and emits revision_due in one transaction. The
// surfaced_at IS NULL guard makes it idempotent — a touch surfaced by a concurrent
// or earlier run yields no row and no duplicate event.
func (s *PgStore) surfaceOne(ctx context.Context, id pgtype.UUID) (bool, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := s.q.WithTx(tx)

	row, err := qtx.MarkSurfaced(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil // already surfaced — no-op
	}
	if err != nil {
		return false, fmt.Errorf("mark surfaced: %w", err)
	}
	if err := emitRevisionDue(ctx, qtx, uuidString(row.AccountID), row.ProblemID, int(row.TouchLevel)); err != nil {
		return false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit tx: %w", err)
	}
	return true, nil
}

// ListUnsentOutbox returns up to limit unsent outbox rows for the relay.
func (s *PgStore) ListUnsentOutbox(ctx context.Context, limit int32) ([]OutboxRow, error) {
	rows, err := s.q.ListUnsentOutbox(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("list unsent outbox: %w", err)
	}
	out := make([]OutboxRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, OutboxRow{EventID: uuidString(r.EventID), Subject: r.Subject, Payload: r.PayloadJson})
	}
	return out, nil
}

// MarkOutboxSent stamps sent_at on a relayed outbox row.
func (s *PgStore) MarkOutboxSent(ctx context.Context, eventID string) error {
	uid, err := parseUUID(eventID)
	if err != nil {
		return fmt.Errorf("parse event id: %w", err)
	}
	if err := s.q.MarkOutboxSent(ctx, uid); err != nil {
		return fmt.Errorf("mark outbox sent: %w", err)
	}
	return nil
}

// --- helpers ---

// AutoPass is the R-SR2 rule: pattern named in under two minutes AND solved within
// the timer AND complexity stated. Any one failing is a fail.
func AutoPass(in ScoreInput) bool {
	return in.NamedPatternSecs < NamePatternMaxSecs && in.SolvedInTimer && in.StatedComplexity
}

// isMockTouch reports whether a touch runs under mock conditions (Day 21 / Day 45,
// levels 4 / 5) per R-SR4.
func isMockTouch(level int) bool { return level == 4 || level == 5 }

// touchDueDate returns the due date for a touch level anchored at t (Day 1/3/7/21/45).
func touchDueDate(level int, t time.Time) time.Time {
	if level < 1 || level > MaxTouchLevel {
		return t
	}
	return t.AddDate(0, 0, touchDays[level-1])
}

// claimInbox records eventID in the inbox for idempotency, returning whether it was
// freshly claimed (true) or already present (false → duplicate delivery).
func claimInbox(ctx context.Context, qtx *gen.Queries, eventID string) (bool, error) {
	_, err := qtx.InsertInbox(ctx, eventID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("claim inbox: %w", err)
	}
	return true, nil
}

// emitRevisionScheduled appends a xlearn.review.revision_scheduled event to the outbox.
func emitRevisionScheduled(ctx context.Context, qtx *gen.Queries, accountID, problemID string, touchLevel int, dueDate time.Time) error {
	return insertEvent(ctx, qtx, SubjectRevisionScheduled, accountID, map[string]any{
		"problem_id":  problemID,
		"touch_level": touchLevel,
		"due_date":    dueDate.UTC().Format(time.RFC3339),
	})
}

// emitRevisionDue appends a xlearn.review.revision_due event to the outbox (no
// due_date — the sweep event carries only problem_id + touch_level per events.md).
func emitRevisionDue(ctx context.Context, qtx *gen.Queries, accountID, problemID string, touchLevel int) error {
	return insertEvent(ctx, qtx, SubjectRevisionDue, accountID, map[string]any{
		"problem_id":  problemID,
		"touch_level": touchLevel,
	})
}

// insertEvent marshals the events.md envelope and appends it to the outbox inside
// the caller's transaction (transactional outbox — never published inline).
func insertEvent(ctx context.Context, qtx *gen.Queries, subject, accountID string, data map[string]any) error {
	eventID := newUUIDv4()
	payload, err := marshalEnvelope(eventID, subject, accountID, data)
	if err != nil {
		return fmt.Errorf("marshal %s: %w", subject, err)
	}
	if err := qtx.InsertOutbox(ctx, gen.InsertOutboxParams{
		EventID:     mustUUID(eventID),
		Subject:     subject,
		PayloadJson: payload,
	}); err != nil {
		return fmt.Errorf("insert outbox %s: %w", subject, err)
	}
	return nil
}

// marshalEnvelope builds the events.md event envelope.
func marshalEnvelope(eventID, subject, accountID string, data map[string]any) ([]byte, error) {
	return json.Marshal(map[string]any{
		"event_id":    eventID,
		"subject":     subject,
		"occurred_at": time.Now().UTC().Format(time.RFC3339Nano),
		"version":     eventVersion,
		"account_id":  accountID,
		"data":        data,
	})
}

// tsz wraps a time in a non-null pgtype.Timestamptz.
func tsz(t time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: t, Valid: true} }

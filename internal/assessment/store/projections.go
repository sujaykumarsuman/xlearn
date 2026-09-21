package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sujaykumarsuman/xlearn/internal/assessment/store/gen"
)

// This file is the S09 progress-projection read model (ADR-0018): the write path
// (ApplyProjection, applied by the durable pull consumers on XLEARN_PRACTICE /
// XLEARN_REVIEW) and the read path (the Progress + Dashboard aggregations).
//
// Projections are keyed at the EVENT grain — per (account, problem), per (account,
// day), per (account, outcome) — because the practice/review events carry only a bare
// problem_id (never pattern or week_n; those are curriculum facts). The gateway
// composes the by-week / by-phase / by-pattern roll-ups from curriculum. Every write is
// an idempotent UPSERT with commutative accumulation (addition / GREATEST / LEAST), so
// out-of-order delivery is safe and a drop-and-replay from the stream start rebuilds an
// identical result — a pure function of the event log (no external reads here).

// Consumed subjects (events.md). assessment reacts to practice solves + review
// schedules; the other captured subjects are recorded in the inbox (dedupe) but change
// no projection this sprint (weak-area context is served live from review).
const (
	SubjectProblemSolved         = "xlearn.practice.problem_solved"
	SubjectAttemptLogged         = "xlearn.practice.attempt_logged"
	SubjectSolutionRevealedEarly = "xlearn.practice.solution_revealed_early"
	SubjectRevisionScheduled     = "xlearn.review.revision_scheduled"
	SubjectRevisionDue           = "xlearn.review.revision_due"
	SubjectMistakeOpened         = "xlearn.review.mistake_opened"
	SubjectMistakeClosed         = "xlearn.review.mistake_closed"
)

// day1TouchLevel is the Day-1 revision touch; a Day-1 re-schedule after the first is a
// ladder reset (a failed re-solve). Mirrors review's touch levels 1..5 = Day 1/3/7/21/45.
const day1TouchLevel = 1

// outcomeRank orders solve outcomes best-to-worst for the mastery max (a commutative
// GREATEST under replay). Unknown / empty outcomes rank 0 (recorded but not scored).
func outcomeRank(outcome string) int16 {
	switch outcome {
	case OutcomeClean:
		return 4
	case OutcomeRough:
		return 3
	case OutcomeAssisted:
		return 2
	case OutcomeMiss:
		return 1
	default:
		return 0
	}
}

// The four first-solve outcomes (also the proj_outcome_mix CHECK set).
const (
	OutcomeClean    = "clean"
	OutcomeRough    = "rough"
	OutcomeAssisted = "assisted"
	OutcomeMiss     = "miss"
)

// validOutcome reports whether o is one of the four scored outcomes.
func validOutcome(o string) bool {
	switch o {
	case OutcomeClean, OutcomeRough, OutcomeAssisted, OutcomeMiss:
		return true
	default:
		return false
	}
}

// ProjectionEvent is a decoded practice/review event the consumer applies to the read
// model. The handler fills only the fields its subject uses; OccurredAt anchors the
// heatmap day (UTC). It carries no wall-clock-derived state so replay is deterministic.
type ProjectionEvent struct {
	EventID    string
	Subject    string
	AccountID  string
	ProblemID  string
	Outcome    string
	FirstSolve bool
	TouchLevel int
	OccurredAt time.Time
}

// HeatmapDay is one day's revision activity (solves + reviews).
type HeatmapDay struct {
	Date    time.Time
	Solves  int
	Reviews int
}

// ProblemMastery is one solved problem's coverage + solve quality. BestRank orders the
// best outcome (clean=4 … miss=1); the gateway maps it to a mastery weight.
type ProblemMastery struct {
	ProblemID     string
	FirstSolvedAt *time.Time
	BestOutcome   string
	BestRank      int
	CleanSolves   int
	SolveCount    int
}

// MockStats is the scored-mock roll-up for the Progress/Dashboard tiles.
type MockStats struct {
	Count   int
	Average int
	Best    int
}

// ApplyProjection applies ev to the read-model projections inside one transaction that
// also claims the inbox, so inbox <-> projected is atomic (effectively-once). A
// duplicate delivery (inbox conflict) is a no-op returning fresh=false. It only ever
// touches schema assessment.
func (s *PgStore) ApplyProjection(ctx context.Context, ev ProjectionEvent) (bool, error) {
	aid, err := parseUUID(ev.AccountID)
	if err != nil {
		return false, fmt.Errorf("parse account id: %w", err)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := s.q.WithTx(tx)

	fresh, err := claimInbox(ctx, qtx, ev.EventID)
	if err != nil {
		return false, err
	}
	if !fresh {
		// Duplicate delivery: the projection was already applied in the tx that first
		// claimed this event_id. Nothing to do (effectively-once).
		return false, nil
	}

	if err := applyProjection(ctx, qtx, aid, ev); err != nil {
		return false, err
	}

	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit tx: %w", err)
	}
	return true, nil
}

// applyProjection routes ev to the right upserts (all within the caller's tx). It is a
// pure function of ev — no reads, no clock branching — so replay is deterministic.
func applyProjection(ctx context.Context, qtx *gen.Queries, aid pgtype.UUID, ev ProjectionEvent) error {
	switch ev.Subject {
	case SubjectProblemSolved:
		return applyProblemSolved(ctx, qtx, aid, ev)
	case SubjectRevisionScheduled:
		return applyRevisionScheduled(ctx, qtx, aid, ev)
	default:
		// attempt_logged / solution_revealed_early / revision_due / mistake_opened /
		// mistake_closed: captured by the stream filters + recorded in the inbox for
		// dedupe, but they mutate no shown projection this sprint.
		return nil
	}
}

// applyProblemSolved projects a solve: coverage (solved + earliest solve time), mastery
// (quality), the heatmap (a solve on the event day) and — for a first solve — the
// outcome mix.
func applyProblemSolved(ctx context.Context, qtx *gen.Queries, aid pgtype.UUID, ev ProjectionEvent) error {
	if ev.ProblemID == "" {
		return nil // nothing to attribute the solve to
	}
	solvedAt := tsz(ev.OccurredAt)
	if err := qtx.UpsertCoverageSolve(ctx, gen.UpsertCoverageSolveParams{
		AccountID: aid, ProblemID: ev.ProblemID, FirstSolvedAt: solvedAt,
	}); err != nil {
		return fmt.Errorf("upsert coverage solve: %w", err)
	}

	clean := int32(0)
	if ev.Outcome == OutcomeClean {
		clean = 1
	}
	if err := qtx.UpsertMastery(ctx, gen.UpsertMasteryParams{
		AccountID:   aid,
		ProblemID:   ev.ProblemID,
		BestOutcome: ev.Outcome,
		BestRank:    outcomeRank(ev.Outcome),
		CleanSolves: clean,
	}); err != nil {
		return fmt.Errorf("upsert mastery: %w", err)
	}

	if err := qtx.UpsertHeatmapSolve(ctx, gen.UpsertHeatmapSolveParams{
		AccountID: aid, ActivityDate: dateOf(ev.OccurredAt),
	}); err != nil {
		return fmt.Errorf("upsert heatmap solve: %w", err)
	}

	// Outcome mix counts a problem's FIRST solve by its (valid) outcome, so the totals
	// sum to the solved count.
	if ev.FirstSolve && validOutcome(ev.Outcome) {
		if err := qtx.UpsertOutcomeMix(ctx, gen.UpsertOutcomeMixParams{
			AccountID: aid, Outcome: ev.Outcome,
		}); err != nil {
			return fmt.Errorf("upsert outcome mix: %w", err)
		}
	}
	return nil
}

// applyRevisionScheduled projects a scheduled review: a review on the event day, and —
// for a Day-1 touch — a ladder anchor (a second+ Day-1 is a reset).
func applyRevisionScheduled(ctx context.Context, qtx *gen.Queries, aid pgtype.UUID, ev ProjectionEvent) error {
	if err := qtx.UpsertHeatmapReview(ctx, gen.UpsertHeatmapReviewParams{
		AccountID: aid, ActivityDate: dateOf(ev.OccurredAt),
	}); err != nil {
		return fmt.Errorf("upsert heatmap review: %w", err)
	}
	if ev.TouchLevel == day1TouchLevel && ev.ProblemID != "" {
		if err := qtx.UpsertCoverageLevel1(ctx, gen.UpsertCoverageLevel1Params{
			AccountID: aid, ProblemID: ev.ProblemID,
		}); err != nil {
			return fmt.Errorf("upsert coverage level1: %w", err)
		}
	}
	return nil
}

// --- read path ---

// SolvedCount returns the number of distinct problems the account has solved.
func (s *PgStore) SolvedCount(ctx context.Context, accountID string) (int, error) {
	aid, err := parseUUID(accountID)
	if err != nil {
		return 0, ErrNotFound
	}
	n, err := s.q.CountSolvedProblems(ctx, aid)
	if err != nil {
		return 0, fmt.Errorf("count solved: %w", err)
	}
	return int(n), nil
}

// Retention returns the ladder + reset counts for the Day-7-retention tile.
func (s *PgStore) Retention(ctx context.Context, accountID string) (ladders, resets int, err error) {
	aid, err := parseUUID(accountID)
	if err != nil {
		return 0, 0, ErrNotFound
	}
	row, err := s.q.RetentionStats(ctx, aid)
	if err != nil {
		return 0, 0, fmt.Errorf("retention stats: %w", err)
	}
	return int(row.Ladders), int(row.Resets), nil
}

// Heatmap returns the account's per-day revision-activity rows on/after since (UTC).
func (s *PgStore) Heatmap(ctx context.Context, accountID string, since time.Time) ([]HeatmapDay, error) {
	aid, err := parseUUID(accountID)
	if err != nil {
		return nil, ErrNotFound
	}
	rows, err := s.q.ListHeatmap(ctx, gen.ListHeatmapParams{AccountID: aid, ActivityDate: dateOf(since)})
	if err != nil {
		return nil, fmt.Errorf("list heatmap: %w", err)
	}
	out := make([]HeatmapDay, 0, len(rows))
	for _, r := range rows {
		out = append(out, HeatmapDay{Date: r.ActivityDate.Time, Solves: int(r.Solves), Reviews: int(r.Reviews)})
	}
	return out, nil
}

// Mastery returns every solved problem with its solve quality.
func (s *PgStore) Mastery(ctx context.Context, accountID string) ([]ProblemMastery, error) {
	aid, err := parseUUID(accountID)
	if err != nil {
		return nil, ErrNotFound
	}
	rows, err := s.q.ListSolvedMastery(ctx, aid)
	if err != nil {
		return nil, fmt.Errorf("list mastery: %w", err)
	}
	out := make([]ProblemMastery, 0, len(rows))
	for _, r := range rows {
		pm := ProblemMastery{
			ProblemID:   r.ProblemID,
			BestOutcome: r.BestOutcome,
			BestRank:    int(r.BestRank),
			CleanSolves: int(r.CleanSolves),
			SolveCount:  int(r.SolveCount),
		}
		if r.FirstSolvedAt.Valid {
			t := r.FirstSolvedAt.Time
			pm.FirstSolvedAt = &t
		}
		out = append(out, pm)
	}
	return out, nil
}

// OutcomeMix returns the first-solve outcome counts keyed by outcome.
func (s *PgStore) OutcomeMix(ctx context.Context, accountID string) (map[string]int, error) {
	aid, err := parseUUID(accountID)
	if err != nil {
		return nil, ErrNotFound
	}
	rows, err := s.q.ListOutcomeMix(ctx, aid)
	if err != nil {
		return nil, fmt.Errorf("list outcome mix: %w", err)
	}
	out := make(map[string]int, len(rows))
	for _, r := range rows {
		out[r.Outcome] = int(r.Cnt)
	}
	return out, nil
}

// MockStats returns the scored-mock count, average /35 and best /35.
func (s *PgStore) MockStats(ctx context.Context, accountID string) (MockStats, error) {
	aid, err := parseUUID(accountID)
	if err != nil {
		return MockStats{}, ErrNotFound
	}
	row, err := s.q.MockAggregate(ctx, aid)
	if err != nil {
		return MockStats{}, fmt.Errorf("mock aggregate: %w", err)
	}
	return MockStats{Count: int(row.ScoredCount), Average: int(row.Average35), Best: int(row.Best35)}, nil
}

// dateOf truncates a time to its UTC calendar day as a non-null pgtype.Date.
func dateOf(t time.Time) pgtype.Date {
	u := t.UTC()
	return pgtype.Date{Time: time.Date(u.Year(), u.Month(), u.Day(), 0, 0, 0, 0, time.UTC), Valid: true}
}

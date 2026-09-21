// Package store is the practice service's persistence layer: pgx/pgxpool over the
// sqlc-generated queries in ./gen, goose migrations in ./migrations, and the
// transactional outbox (ADR-0004/0005). It exposes small domain types with plain
// Go scalars so the HTTP layer never touches pgtype, keeps the xlearn_practice role
// scoped to schema practice (all SQL is schema-qualified), and owns the guided-flow
// invariants — stage gating, server-authoritative timers, the reveal penalty, and
// outcome logging — inside single transactions that also append the outbox rows.
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

	"github.com/sujaykumarsuman/xlearn/internal/practice/store/gen"
)

// Content stages of the guided flow (R-PF1). attempt is the statement (always
// visible); hint and solution are gated behind explicit reveals. re-implement and
// log are client/terminal steps with no gated content, so they are not stages here.
const (
	StageAttempt  = "attempt"
	StageHint     = "hint"
	StageSolution = "solution"
)

// Timer kinds + durations (R-PF3). The attempt is 15 min, the hint 10 min;
// solution/re-implement/log are untimed.
const (
	TimerAttempt = "attempt"
	TimerHint    = "hint"

	AttemptTimer = 15 * time.Minute
	HintTimer    = 10 * time.Minute

	// EarlyRevealPenaltyDays is how far out the owed re-attempt is queued when the
	// solution is revealed before the attempt timer elapses (R-PF2).
	EarlyRevealPenaltyDays = 3
)

// Outcome values (R-OL1). Anything below clean opens a mistake downstream.
const (
	OutcomeClean    = "clean"
	OutcomeRough    = "rough"
	OutcomeAssisted = "assisted"
	OutcomeMiss     = "miss"
)

// Event subjects + envelope version (events.md). The relay publishes these to the
// XLEARN_PRACTICE JetStream stream.
const (
	SubjectProblemSolved         = "xlearn.practice.problem_solved"
	SubjectAttemptLogged         = "xlearn.practice.attempt_logged"
	SubjectSolutionRevealedEarly = "xlearn.practice.solution_revealed_early"
	eventVersion                 = 1
)

// contentStages is the ordered content ladder used to build UnlockedStages.
var contentStages = []string{StageAttempt, StageHint, StageSolution}

// Errors mapped to HTTP status by the handlers.
var (
	ErrNotFound        = errors.New("practice: not found")
	ErrNoAttempt       = errors.New("practice: no active attempt")
	ErrAlreadySolved   = errors.New("practice: already solved")
	ErrNothingToReveal = errors.New("practice: nothing left to reveal")
	ErrInvalidOutcome  = errors.New("practice: invalid outcome")
)

// Timer is the server-authoritative countdown mirror for the active stage. Expired
// is computed from DeadlineAt so a refresh always resumes the same clock (R-PF3).
type Timer struct {
	Kind       string
	DeadlineAt time.Time
	Expired    bool
}

// State is a learner's computed state for one problem.
type State struct {
	ProblemID      string
	Status         string   // available | attempting | solved
	StageReached   string   // "" | attempt | hint | solution (deepest content stage)
	UnlockedStages []string // subset of contentStages, in order (R-PF1 gate)
	CurrentTouch   int
	LastOutcome    string    // "" until logged
	FirstSolvedAt  time.Time // zero until first solve
	RevealedEarly  bool
	Timer          *Timer // active timer for the current stage, or nil
}

// Penalty is the reveal-penalty acknowledgement (R-PF2).
type Penalty struct {
	OwedAttempt bool
	DueInDays   int
}

// RevealResult is the outcome of a reveal: the new state, the stage just unlocked,
// and the penalty ack when the solution was revealed early.
type RevealResult struct {
	State    State
	Revealed string
	Penalty  *Penalty
}

// OutboxRow is one unsent domain event awaiting relay to NATS.
type OutboxRow struct {
	EventID string
	Subject string
	Payload []byte
}

// Store is the practice persistence seam. Handlers depend on this interface so they
// can be unit-tested against an in-memory fake.
type Store interface {
	// GetState returns the full computed state (+ active timer) for one problem; a
	// problem the learner has never touched is "available" with only attempt unlocked.
	GetState(ctx context.Context, accountID, problemID string) (State, error)
	// ListStates returns the (lightweight) states for the given problems the learner
	// has a row for; problems without a row are absent (the caller defaults them).
	ListStates(ctx context.Context, accountID string, problemIDs []string) (map[string]State, error)
	// StartAttempt creates or resumes the attempt, starting the 15-min timer, and
	// moves the problem to "attempting" — all in one transaction.
	StartAttempt(ctx context.Context, accountID, problemID string) (State, error)
	// Reveal unlocks the next content stage (hint → solution), starting the hint
	// timer, and — if the solution is revealed early — flags it and emits
	// solution_revealed_early via the outbox, in one transaction.
	Reveal(ctx context.Context, accountID, problemID string) (RevealResult, error)
	// LogOutcome records the outcome, marks the problem solved, and appends the
	// problem_solved + attempt_logged outbox rows, all in one transaction.
	LogOutcome(ctx context.Context, accountID, problemID, value string) (State, error)
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

// GetState computes the current state for one problem.
func (s *PgStore) GetState(ctx context.Context, accountID, problemID string) (State, error) {
	aid, err := parseUUID(accountID)
	if err != nil {
		return State{}, ErrNotFound
	}
	ups, err := s.q.GetUserProblemState(ctx, gen.GetUserProblemStateParams{AccountID: aid, ProblemID: problemID})
	if errors.Is(err, pgx.ErrNoRows) {
		return defaultState(problemID), nil
	}
	if err != nil {
		return State{}, fmt.Errorf("get state: %w", err)
	}
	return s.composeState(ctx, ups)
}

// ListStates returns lightweight states (status / last outcome / touch) for a set
// of problems — enough to drive the Week five-touch dots without the per-attempt
// timer/unlocked-stage detail.
func (s *PgStore) ListStates(ctx context.Context, accountID string, problemIDs []string) (map[string]State, error) {
	out := map[string]State{}
	if len(problemIDs) == 0 {
		return out, nil
	}
	aid, err := parseUUID(accountID)
	if err != nil {
		return out, nil
	}
	rows, err := s.q.ListUserProblemStates(ctx, gen.ListUserProblemStatesParams{AccountID: aid, ProblemIds: problemIDs})
	if err != nil {
		return nil, fmt.Errorf("list states: %w", err)
	}
	for _, ups := range rows {
		out[ups.ProblemID] = State{
			ProblemID:     ups.ProblemID,
			Status:        ups.Status,
			CurrentTouch:  int(ups.CurrentTouch),
			LastOutcome:   ups.LastOutcome.String,
			FirstSolvedAt: ups.FirstSolvedAt.Time,
		}
	}
	return out, nil
}

// StartAttempt creates or resumes an attempt in one transaction.
func (s *PgStore) StartAttempt(ctx context.Context, accountID, problemID string) (State, error) {
	aid, err := parseUUID(accountID)
	if err != nil {
		return State{}, ErrNotFound
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return State{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := s.q.WithTx(tx)

	ups, err := qtx.UpsertUserProblemState(ctx, gen.UpsertUserProblemStateParams{AccountID: aid, ProblemID: problemID})
	if err != nil {
		return State{}, fmt.Errorf("upsert state: %w", err)
	}

	// A solved problem is terminal for this sprint: return its state unchanged
	// rather than opening a fresh attempt (re-attempts arrive with review, S06).
	if ups.Status == "solved" {
		if err := tx.Commit(ctx); err != nil {
			return State{}, fmt.Errorf("commit tx: %w", err)
		}
		return s.GetState(ctx, accountID, problemID)
	}

	// Resume an in-progress attempt (server-authoritative: its timer is not reset).
	if _, err := qtx.GetOpenAttempt(ctx, ups.ID); err == nil {
		if _, err := qtx.SetStateAttempting(ctx, gen.SetStateAttemptingParams{AccountID: aid, ProblemID: problemID}); err != nil {
			return State{}, fmt.Errorf("set attempting: %w", err)
		}
		if err := tx.Commit(ctx); err != nil {
			return State{}, fmt.Errorf("commit tx: %w", err)
		}
		return s.GetState(ctx, accountID, problemID)
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return State{}, fmt.Errorf("get open attempt: %w", err)
	}

	// Fresh attempt: create it, enter the attempt stage, start the 15-min timer.
	att, err := qtx.CreateAttempt(ctx, ups.ID)
	if err != nil {
		return State{}, fmt.Errorf("create attempt: %w", err)
	}
	if err := qtx.CreateStageEvent(ctx, gen.CreateStageEventParams{AttemptID: att.ID, Stage: StageAttempt}); err != nil {
		return State{}, fmt.Errorf("create stage event: %w", err)
	}
	if _, err := qtx.CreateTimer(ctx, gen.CreateTimerParams{
		AttemptID:  att.ID,
		Kind:       TimerAttempt,
		DeadlineAt: pgtype.Timestamptz{Time: time.Now().Add(AttemptTimer), Valid: true},
	}); err != nil {
		return State{}, fmt.Errorf("create timer: %w", err)
	}
	if _, err := qtx.SetStateAttempting(ctx, gen.SetStateAttemptingParams{AccountID: aid, ProblemID: problemID}); err != nil {
		return State{}, fmt.Errorf("set attempting: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return State{}, fmt.Errorf("commit tx: %w", err)
	}
	return s.GetState(ctx, accountID, problemID)
}

// Reveal unlocks the next content stage in one transaction.
func (s *PgStore) Reveal(ctx context.Context, accountID, problemID string) (RevealResult, error) {
	aid, err := parseUUID(accountID)
	if err != nil {
		return RevealResult{}, ErrNotFound
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return RevealResult{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := s.q.WithTx(tx)

	ups, err := qtx.GetUserProblemState(ctx, gen.GetUserProblemStateParams{AccountID: aid, ProblemID: problemID})
	if errors.Is(err, pgx.ErrNoRows) {
		return RevealResult{}, ErrNoAttempt
	}
	if err != nil {
		return RevealResult{}, fmt.Errorf("get state: %w", err)
	}
	if ups.Status == "solved" {
		return RevealResult{}, ErrAlreadySolved
	}
	att, err := qtx.GetOpenAttempt(ctx, ups.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		return RevealResult{}, ErrNoAttempt
	}
	if err != nil {
		return RevealResult{}, fmt.Errorf("get open attempt: %w", err)
	}

	next, err := nextStage(att.StageReached)
	if err != nil {
		return RevealResult{}, err
	}

	if err := qtx.CreateStageEvent(ctx, gen.CreateStageEventParams{
		AttemptID:    att.ID,
		Stage:        next,
		UnlockedFrom: pgtype.Text{String: att.StageReached, Valid: true},
	}); err != nil {
		return RevealResult{}, fmt.Errorf("create stage event: %w", err)
	}
	if err := qtx.SetAttemptStage(ctx, gen.SetAttemptStageParams{ID: att.ID, StageReached: next}); err != nil {
		return RevealResult{}, fmt.Errorf("set attempt stage: %w", err)
	}

	var penalty *Penalty
	switch next {
	case StageHint:
		if _, err := qtx.CreateTimer(ctx, gen.CreateTimerParams{
			AttemptID:  att.ID,
			Kind:       TimerHint,
			DeadlineAt: pgtype.Timestamptz{Time: time.Now().Add(HintTimer), Valid: true},
		}); err != nil {
			return RevealResult{}, fmt.Errorf("create hint timer: %w", err)
		}
	case StageSolution:
		early, err := s.revealIsEarly(ctx, qtx, att.ID)
		if err != nil {
			return RevealResult{}, err
		}
		if early {
			if err := qtx.SetAttemptRevealedEarly(ctx, att.ID); err != nil {
				return RevealResult{}, fmt.Errorf("set revealed early: %w", err)
			}
			if err := insertEvent(ctx, qtx, SubjectSolutionRevealedEarly, accountID, map[string]any{
				"problem_id": problemID,
			}); err != nil {
				return RevealResult{}, err
			}
			penalty = &Penalty{OwedAttempt: true, DueInDays: EarlyRevealPenaltyDays}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return RevealResult{}, fmt.Errorf("commit tx: %w", err)
	}
	st, err := s.GetState(ctx, accountID, problemID)
	if err != nil {
		return RevealResult{}, err
	}
	return RevealResult{State: st, Revealed: next, Penalty: penalty}, nil
}

// LogOutcome records the outcome and emits the domain events in one transaction.
func (s *PgStore) LogOutcome(ctx context.Context, accountID, problemID, value string) (State, error) {
	if !validOutcome(value) {
		return State{}, ErrInvalidOutcome
	}
	aid, err := parseUUID(accountID)
	if err != nil {
		return State{}, ErrNotFound
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return State{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := s.q.WithTx(tx)

	ups, err := qtx.GetUserProblemState(ctx, gen.GetUserProblemStateParams{AccountID: aid, ProblemID: problemID})
	if errors.Is(err, pgx.ErrNoRows) {
		return State{}, ErrNoAttempt
	}
	if err != nil {
		return State{}, fmt.Errorf("get state: %w", err)
	}
	if ups.Status == "solved" {
		return State{}, ErrAlreadySolved
	}
	att, err := qtx.GetOpenAttempt(ctx, ups.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		return State{}, ErrNoAttempt
	}
	if err != nil {
		return State{}, fmt.Errorf("get open attempt: %w", err)
	}

	firstSolve := !ups.FirstSolvedAt.Valid

	if err := qtx.CreateOutcome(ctx, gen.CreateOutcomeParams{
		AttemptID:     att.ID,
		Value:         value,
		RevealedEarly: att.RevealedEarly,
	}); err != nil {
		return State{}, fmt.Errorf("create outcome: %w", err)
	}
	if err := qtx.EndAttempt(ctx, att.ID); err != nil {
		return State{}, fmt.Errorf("end attempt: %w", err)
	}
	if _, err := qtx.SetStateSolved(ctx, gen.SetStateSolvedParams{
		AccountID:   aid,
		ProblemID:   problemID,
		LastOutcome: pgtype.Text{String: value, Valid: true},
	}); err != nil {
		return State{}, fmt.Errorf("set solved: %w", err)
	}

	durationS := int(time.Since(att.StartedAt.Time).Seconds())
	if durationS < 0 {
		durationS = 0
	}

	// problem_solved — review schedules five-touch (on first clean solve) and opens
	// a mistake when below_clean; assessment updates coverage/mastery projections.
	if err := insertEvent(ctx, qtx, SubjectProblemSolved, accountID, map[string]any{
		"problem_id":  problemID,
		"outcome":     value,
		"first_solve": firstSolve,
		"below_clean": value != OutcomeClean,
	}); err != nil {
		return State{}, err
	}
	// attempt_logged — assessment updates outcome-mix / coverage projections.
	if err := insertEvent(ctx, qtx, SubjectAttemptLogged, accountID, map[string]any{
		"problem_id":    problemID,
		"stage_reached": att.StageReached,
		"duration_s":    durationS,
	}); err != nil {
		return State{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return State{}, fmt.Errorf("commit tx: %w", err)
	}
	return s.GetState(ctx, accountID, problemID)
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

// --- state composition ---

// composeState builds the full State for a problem the learner has a row for,
// reading the latest attempt, its stage events and the active-stage timer.
func (s *PgStore) composeState(ctx context.Context, ups gen.PracticeUserProblemState) (State, error) {
	st := State{
		ProblemID:      ups.ProblemID,
		Status:         ups.Status,
		CurrentTouch:   int(ups.CurrentTouch),
		LastOutcome:    ups.LastOutcome.String,
		FirstSolvedAt:  ups.FirstSolvedAt.Time,
		UnlockedStages: []string{StageAttempt},
	}
	att, err := s.q.GetLatestAttempt(ctx, ups.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		return st, nil // a state row with no attempt yet: statement only
	}
	if err != nil {
		return State{}, fmt.Errorf("get latest attempt: %w", err)
	}
	st.StageReached = att.StageReached
	st.RevealedEarly = att.RevealedEarly

	events, err := s.q.ListStageEvents(ctx, att.ID)
	if err != nil {
		return State{}, fmt.Errorf("list stage events: %w", err)
	}
	st.UnlockedStages = unlockedFromEvents(events)

	// The active timer mirrors the current stage's countdown, and only while an
	// attempt is in progress; solution/re-implement/log and solved states show none.
	if ups.Status == "attempting" && (att.StageReached == StageAttempt || att.StageReached == StageHint) {
		tm, err := s.q.GetTimer(ctx, gen.GetTimerParams{AttemptID: att.ID, Kind: att.StageReached})
		switch {
		case err == nil:
			st.Timer = &Timer{
				Kind:       tm.Kind,
				DeadlineAt: tm.DeadlineAt.Time,
				Expired:    time.Now().After(tm.DeadlineAt.Time),
			}
		case errors.Is(err, pgx.ErrNoRows):
			// no timer row (shouldn't happen for these stages) — leave nil
		default:
			return State{}, fmt.Errorf("get timer: %w", err)
		}
	}
	return st, nil
}

// revealIsEarly reports whether the attempt timer is still running (now before its
// deadline). Revealing the solution then is "early" and carries the penalty.
func (s *PgStore) revealIsEarly(ctx context.Context, qtx *gen.Queries, attemptID pgtype.UUID) (bool, error) {
	tm, err := qtx.GetTimer(ctx, gen.GetTimerParams{AttemptID: attemptID, Kind: TimerAttempt})
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil // no attempt timer: treat as not early
	}
	if err != nil {
		return false, fmt.Errorf("get attempt timer: %w", err)
	}
	return time.Now().Before(tm.DeadlineAt.Time), nil
}

// --- helpers ---

// defaultState is the state of a problem the learner has never touched: available,
// with only the statement (attempt stage) unlocked.
func defaultState(problemID string) State {
	return State{
		ProblemID:      problemID,
		Status:         "available",
		UnlockedStages: []string{StageAttempt},
	}
}

// unlockedFromEvents derives the ordered set of unlocked content stages from an
// attempt's stage events (R-PF1). The attempt stage is always included.
func unlockedFromEvents(events []gen.PracticeStageEvent) []string {
	seen := map[string]bool{StageAttempt: true}
	for _, e := range events {
		if isContentStage(e.Stage) {
			seen[e.Stage] = true
		}
	}
	out := make([]string, 0, len(contentStages))
	for _, st := range contentStages {
		if seen[st] {
			out = append(out, st)
		}
	}
	return out
}

// nextStage returns the content stage a reveal unlocks from the current deepest one.
func nextStage(current string) (string, error) {
	switch current {
	case StageAttempt:
		return StageHint, nil
	case StageHint:
		return StageSolution, nil
	case StageSolution:
		return "", ErrNothingToReveal
	default:
		return StageHint, nil // defensive: unknown current → offer the hint
	}
}

func isContentStage(s string) bool {
	return s == StageAttempt || s == StageHint || s == StageSolution
}

func validOutcome(v string) bool {
	switch v {
	case OutcomeClean, OutcomeRough, OutcomeAssisted, OutcomeMiss:
		return true
	}
	return false
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

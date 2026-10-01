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
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sujaykumarsuman/xlearn/internal/platform/events"
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
// XLEARN_PRACTICE JetStream stream. Every practice subject is course-scoped, so since
// m1-03 (M1b) each is a v2 envelope carrying the problem state's path_slug (the
// v1.6.0 consumers already decode v2, m1-02).
const (
	SubjectProblemSolved         = "xlearn.practice.problem_solved"
	SubjectAttemptLogged         = "xlearn.practice.attempt_logged"
	SubjectSolutionRevealedEarly = "xlearn.practice.solution_revealed_early"
	eventVersion                 = events.EnvelopeV2
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
	// ErrAttemptClosed: the attempt exists and is the account's, but it has concluded
	// (MarkCoachAssist → 409 attempt_closed).
	ErrAttemptClosed = errors.New("practice: attempt closed")
)

// PurposeCourse is the purpose of every attempt until M2a: a counted course attempt.
// m2-01 adds attempt.purpose and reports open touches as "touch" in the same list.
const PurposeCourse = "course"

// CappedByCoach is OutcomeResult.CappedBy when a self-reported clean/rough was recorded
// as assisted because the coach was used on the attempt (D27).
const CappedByCoach = "coach"

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
	// CoachAssistAt is when the coach was first used on the OPEN attempt (D27); zero when
	// it never was, or when no attempt is open. It caps that attempt's grade at assisted.
	CoachAssistAt time.Time
}

// OpenAttempt is one open (not yet concluded) attempt of the account (GET
// /attempts/open). CoachAssistAt is zero until the coach is used on it.
type OpenAttempt struct {
	ID            string
	ProblemID     string
	PathSlug      string
	Purpose       string
	StartedAt     time.Time
	StageReached  string
	CoachAssistAt time.Time
}

// OutcomeResult is a logged outcome: the new state, and CappedBy = CappedByCoach when
// the D27 clamp recorded a self-reported clean/rough as assisted.
type OutcomeResult struct {
	State    State
	CappedBy string
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
	// moves the problem to "attempting" — all in one transaction. pathSlug is the
	// problem's course (the gateway's `?path=`), written when the problem state row is
	// CREATED; a resume keeps the row's own course.
	StartAttempt(ctx context.Context, accountID, problemID, pathSlug string) (State, error)
	// Reveal unlocks the next content stage (hint → solution), starting the hint
	// timer, and — if the solution is revealed early — flags it and emits
	// solution_revealed_early via the outbox, in one transaction.
	Reveal(ctx context.Context, accountID, problemID string) (RevealResult, error)
	// LogOutcome records the outcome, marks the problem solved, and appends the
	// problem_solved + attempt_logged outbox rows, all in one transaction. A clean/rough
	// on an attempt the coach was used on is recorded as assisted (D27; CappedBy).
	LogOutcome(ctx context.Context, accountID, problemID, value string) (OutcomeResult, error)
	// MarkCoachAssist records the first coach use on the account's open attempt
	// (idempotent: the first time is kept) and returns it. ErrNotFound for an unknown
	// attempt or another account's; ErrAttemptClosed for a concluded one.
	MarkCoachAssist(ctx context.Context, accountID, attemptID string) (time.Time, error)
	// ListOpenAttempts lists the account's open attempts, newest first; problemID ""
	// lists every problem's.
	ListOpenAttempts(ctx context.Context, accountID, problemID string) ([]OpenAttempt, error)
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
func (s *PgStore) StartAttempt(ctx context.Context, accountID, problemID, pathSlug string) (State, error) {
	aid, err := parseUUID(accountID)
	if err != nil {
		return State{}, ErrNotFound
	}
	if pathSlug == "" {
		// Never write '' or lean on the column default (m1-08 drops it): the handler
		// always resolves a course.
		return State{}, errors.New("practice: start attempt without a course")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return State{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := s.q.WithTx(tx)

	// The course is written explicitly on create; the upsert's conflict branch leaves
	// an existing row's course alone (a resume keeps it).
	ups, err := qtx.UpsertUserProblemState(ctx, gen.UpsertUserProblemStateParams{AccountID: aid, ProblemID: problemID, PathSlug: pathSlug})
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
	// The attempt denormalises its problem state's account, course and problem (M1a).
	att, err := qtx.CreateAttempt(ctx, gen.CreateAttemptParams{
		UserProblemStateID: ups.ID,
		AccountID:          ups.AccountID,
		PathSlug:           pgtype.Text{String: ups.PathSlug, Valid: true},
		ProblemID:          pgtype.Text{String: ups.ProblemID, Valid: true},
	})
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
			if err := insertEvent(ctx, qtx, SubjectSolutionRevealedEarly, accountID, ups.PathSlug, map[string]any{
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
func (s *PgStore) LogOutcome(ctx context.Context, accountID, problemID, value string) (OutcomeResult, error) {
	if !validOutcome(value) {
		return OutcomeResult{}, ErrInvalidOutcome
	}
	aid, err := parseUUID(accountID)
	if err != nil {
		return OutcomeResult{}, ErrNotFound
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return OutcomeResult{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := s.q.WithTx(tx)

	ups, err := qtx.GetUserProblemState(ctx, gen.GetUserProblemStateParams{AccountID: aid, ProblemID: problemID})
	if errors.Is(err, pgx.ErrNoRows) {
		return OutcomeResult{}, ErrNoAttempt
	}
	if err != nil {
		return OutcomeResult{}, fmt.Errorf("get state: %w", err)
	}
	if ups.Status == "solved" {
		return OutcomeResult{}, ErrAlreadySolved
	}
	// Row-locked (m1-07): a concurrent MarkCoachAssist waits for this conclusion and then
	// finds the attempt closed, so an assist this transaction can't see was refused (and
	// its chat never forwarded).
	att, err := qtx.GetOpenAttemptForUpdate(ctx, ups.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		return OutcomeResult{}, ErrNoAttempt
	}
	if err != nil {
		return OutcomeResult{}, fmt.Errorf("get open attempt: %w", err)
	}

	firstSolve := !ups.FirstSolvedAt.Valid

	// D27 / ADR-0029: coach help during the attempt caps it at assisted. The lock point is
	// this conclusion (the outcome's logged_at); the attempt is locked and still open, so
	// a coach_assist_at it carries was recorded before it (coach_assist_at <= lock_at).
	coachUsed := att.CoachAssistAt.Valid
	effective, cappedBy := clampOutcome(value, coachUsed)

	if err := qtx.CreateOutcome(ctx, gen.CreateOutcomeParams{
		AttemptID:     att.ID,
		Value:         effective,
		RevealedEarly: att.RevealedEarly,
	}); err != nil {
		return OutcomeResult{}, fmt.Errorf("create outcome: %w", err)
	}
	if err := qtx.EndAttempt(ctx, att.ID); err != nil {
		return OutcomeResult{}, fmt.Errorf("end attempt: %w", err)
	}
	if _, err := qtx.SetStateSolved(ctx, gen.SetStateSolvedParams{
		AccountID:   aid,
		ProblemID:   problemID,
		LastOutcome: pgtype.Text{String: effective, Valid: true},
	}); err != nil {
		return OutcomeResult{}, fmt.Errorf("set solved: %w", err)
	}

	durationS := int(time.Since(att.StartedAt.Time).Seconds())
	if durationS < 0 {
		durationS = 0
	}

	// problem_solved — review schedules five-touch (on first clean solve) and opens
	// a mistake when below_clean; assessment updates coverage/mastery projections.
	// `assist` (m1-07) is additive on an existing subject: the v1.6.0 consumers decode
	// with DecodeEnvelope, which ignores unknown fields (events.md).
	if err := insertEvent(ctx, qtx, SubjectProblemSolved, accountID, ups.PathSlug, map[string]any{
		"problem_id":  problemID,
		"outcome":     effective,
		"first_solve": firstSolve,
		"below_clean": effective != OutcomeClean,
		"assist":      assistData(att.StageReached, coachUsed),
	}); err != nil {
		return OutcomeResult{}, err
	}
	// attempt_logged — assessment updates outcome-mix / coverage projections.
	if err := insertEvent(ctx, qtx, SubjectAttemptLogged, accountID, ups.PathSlug, map[string]any{
		"problem_id":    problemID,
		"stage_reached": att.StageReached,
		"duration_s":    durationS,
	}); err != nil {
		return OutcomeResult{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return OutcomeResult{}, fmt.Errorf("commit tx: %w", err)
	}
	st, err := s.GetState(ctx, accountID, problemID)
	if err != nil {
		return OutcomeResult{}, err
	}
	return OutcomeResult{State: st, CappedBy: cappedBy}, nil
}

// clampOutcome applies the D27 ceiling: with the coach used on the attempt, a
// self-reported clean or rough is recorded as assisted (cappedBy "coach"); assisted and
// miss are unchanged (miss is already below the ceiling).
func clampOutcome(value string, coachUsed bool) (effective, cappedBy string) {
	if coachUsed && (value == OutcomeClean || value == OutcomeRough) {
		return OutcomeAssisted, CappedByCoach
	}
	return value, ""
}

// assistData is problem_solved's `assist` object (m1-07): hint = the attempt went past
// the statement (a hint or the solution was revealed); coach = the coach was used on it.
func assistData(stageReached string, coachUsed bool) map[string]bool {
	return map[string]bool{"hint": stageReached != StageAttempt, "coach": coachUsed}
}

// MarkCoachAssist records the first coach use on an open attempt (D27).
func (s *PgStore) MarkCoachAssist(ctx context.Context, accountID, attemptID string) (time.Time, error) {
	aid, err := parseUUID(accountID)
	if err != nil {
		return time.Time{}, ErrNotFound
	}
	id, err := parseUUID(attemptID)
	if err != nil {
		return time.Time{}, ErrNotFound
	}
	at, err := s.q.MarkCoachAssist(ctx, gen.MarkCoachAssistParams{ID: id, AccountID: aid})
	if err == nil {
		return at.Time, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, fmt.Errorf("mark coach assist: %w", err)
	}
	// No row: unknown or another account's (404), or concluded (409).
	if _, err := s.q.GetAttemptOwned(ctx, gen.GetAttemptOwnedParams{ID: id, AccountID: aid}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return time.Time{}, ErrNotFound
		}
		return time.Time{}, fmt.Errorf("get attempt: %w", err)
	}
	return time.Time{}, ErrAttemptClosed
}

// ListOpenAttempts lists the account's open attempts (every one is a course attempt
// until M2a).
func (s *PgStore) ListOpenAttempts(ctx context.Context, accountID, problemID string) ([]OpenAttempt, error) {
	aid, err := parseUUID(accountID)
	if err != nil {
		return []OpenAttempt{}, nil
	}
	rows, err := s.q.ListOpenAttempts(ctx, gen.ListOpenAttemptsParams{
		AccountID: aid,
		ProblemID: pgtype.Text{String: problemID, Valid: problemID != ""},
	})
	if err != nil {
		return nil, fmt.Errorf("list open attempts: %w", err)
	}
	out := make([]OpenAttempt, 0, len(rows))
	for _, r := range rows {
		out = append(out, OpenAttempt{
			ID:            uuidString(r.ID),
			ProblemID:     r.ProblemID.String,
			PathSlug:      r.PathSlug.String,
			Purpose:       PurposeCourse,
			StartedAt:     r.StartedAt.Time,
			StageReached:  r.StageReached,
			CoachAssistAt: r.CoachAssistAt.Time,
		})
	}
	return out, nil
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
	if !att.EndedAt.Valid {
		// Only the OPEN attempt's assist is reported (D27 caps that attempt; the HUD chip
		// and the outcome step's cap line read it). A concluded attempt's grade already
		// carries the cap.
		st.CoachAssistAt = att.CoachAssistAt.Time
	}

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

// insertEvent marshals the event envelope — v2, carrying pathSlug (the problem state's
// course); events.NewEnvelope also enforces the size cap — and appends it to the
// outbox inside the caller's transaction (transactional outbox — never published
// inline). occurred_at is the write time, as in v1.
func insertEvent(ctx context.Context, qtx *gen.Queries, subject, accountID, pathSlug string, data map[string]any) error {
	eventID := newUUIDv4()
	payload, err := events.NewEnvelope(eventVersion, eventID, subject, accountID, pathSlug, time.Now(), data)
	if err != nil {
		return fmt.Errorf("marshal %s: %w", subject, err)
	}
	aid, err := parseUUID(accountID)
	if err != nil {
		return fmt.Errorf("outbox account id: %w", err)
	}
	if err := qtx.InsertOutbox(ctx, gen.InsertOutboxParams{
		EventID:     mustUUID(eventID),
		Subject:     subject,
		PayloadJson: payload,
		AccountID:   aid,
	}); err != nil {
		return fmt.Errorf("insert outbox %s: %w", subject, err)
	}
	return nil
}

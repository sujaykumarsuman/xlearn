// Package store is the assessment service's persistence layer: pgx/pgxpool over the
// sqlc-generated queries in ./gen, goose migrations in ./migrations, the
// transactional outbox and the idempotent inbox (ADR-0004/0005). It exposes small
// domain types with plain Go scalars so the HTTP/consumer layers never touch
// pgtype, keeps the xlearn_assessment role scoped to schema assessment (all SQL is
// schema-qualified), and owns the mock-interview invariants — the rubric total is
// summed server-side, scoring transitions live -> scored exactly once, and the
// mock_completed event is appended to the outbox inside the same transaction.
package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sujaykumarsuman/xlearn/internal/assessment/store/gen"
	"github.com/sujaykumarsuman/xlearn/internal/platform/events"
)

// Dimensions is the 7-value rubric enum in canonical PRD order (R-MK2). Scores are
// stored under these text keys; human display names live in the http layer. The
// server sums exactly these seven into the total — a client total is never trusted.
var Dimensions = []string{
	"communication",
	"problem_understanding",
	"brute_force",
	"optimisation",
	"code_quality",
	"edge_cases",
	"complexity",
}

const (
	// NumDimensions is the fixed rubric width (R-MK2: 7 dims x 1..5 = /35).
	NumDimensions = 7
	// MinScore / MaxScore bound each dimension's integer score (R-MK2).
	MinScore = 1
	MaxScore = 5

	// MockDuration is the timed session length (R-MK1): 45 minutes, server-authoritative.
	MockDuration = 45 * time.Minute

	// Session status values.
	StatusLive   = "live"
	StatusScored = "scored"

	// MaxTotal is the v1 rubric's maximum total (7 dims x 5 = 35). m1-02 (M1a) writes it
	// as mock_session.max_total, and readers default a NULL max_total to it.
	MaxTotal = NumDimensions * MaxScore

	// RubricID is the id the DSA course's manifest gives the v1 mock rubric (its
	// course.json mock.rubric.id; pinned by TestManifestGoldenMirror). New sessions
	// record it with RubricSnapshot; it is the only rubric this service scores.
	RubricID = "dsa-mock@1"

	// ScoredBySelf is the only scorer v1 has: the learner's own rubric (m1-02; 'ai-byo'
	// arrives with the AI scorer).
	ScoredBySelf = "self"
)

// RubricSnapshot is the rubric a session is scored against, frozen on the session row
// (mock_session.rubric_snapshot): the DSA manifest's mock.rubric, byte-for-byte the
// literal migration 00004 backfills (pinned to the manifest by TestManifestGoldenMirror).
const RubricSnapshot = `{"id": "dsa-mock@1", "dims": [{"id": "communication", "label": "Communication"}, {"id": "problem_understanding", "label": "Problem understanding"}, {"id": "brute_force", "label": "Brute force"}, {"id": "optimisation", "label": "Optimisation"}, {"id": "code_quality", "label": "Code quality"}, {"id": "edge_cases", "label": "Edge cases"}, {"id": "complexity", "label": "Complexity"}], "scale": [1, 5]}`

// dimensionSet indexes Dimensions for O(1) membership checks.
var dimensionSet = func() map[string]bool {
	m := make(map[string]bool, len(Dimensions))
	for _, d := range Dimensions {
		m[d] = true
	}
	return m
}()

// validDifficulties are the mock-setup difficulty values (difficulty -> UI token).
var validDifficulties = map[string]bool{"easy": true, "med": true, "hard": true}

// ValidDifficulty reports whether d is one of easy/med/hard.
func ValidDifficulty(d string) bool { return validDifficulties[d] }

// Event subject + envelope version (events.md). assessment emits mock_completed to
// its own XLEARN_ASSESSMENT stream via the outbox relay. mock_completed is
// course-scoped, so since m1-03 (M1b) it is a v2 envelope carrying the session's
// path_slug (the v1.6.0 consumers already decode v2, m1-02).
const (
	SubjectMockCompleted = "xlearn.assessment.mock_completed"

	eventVersion = events.EnvelopeV2
)

// Errors mapped to HTTP status by the handlers.
var (
	// ErrNotFound is returned when a session is not the account's (or absent).
	ErrNotFound = errors.New("assessment: not found")
	// ErrInvalidRubric is returned when a rubric submission is not exactly the seven
	// dimensions each in [1,5] (the server-authoritative validation; R-MK2).
	ErrInvalidRubric = errors.New("assessment: invalid rubric")
)

// ValidateRubric checks scores contains EXACTLY the seven dimensions, each an integer
// in [1,5]. It rejects partial submissions and unknown keys — the server never trusts
// a client-supplied total and computes /35 itself (R-MK2).
func ValidateRubric(scores map[string]int) error {
	if len(scores) != NumDimensions {
		return ErrInvalidRubric
	}
	for _, dim := range Dimensions {
		v, ok := scores[dim]
		if !ok || v < MinScore || v > MaxScore {
			return ErrInvalidRubric
		}
	}
	return nil
}

// TotalScore sums a validated rubric into its total (R-MK2: /35 for the v1 rubric).
func TotalScore(scores map[string]int) int {
	t := 0
	for _, dim := range Dimensions {
		t += scores[dim]
	}
	return t
}

// MockSession is one timed mock, with plain Go scalars. Total is nil until scored and
// reads only mock_session.total (m1-03: m1-02 backfilled it for every v1 row and v1.6.0
// dual-writes it, so the v1 /35 column is never read); MaxTotal reads
// COALESCE(max_total, 35). RubricID is the rubric the session is scored against.
type MockSession struct {
	ID         string
	AccountID  string
	SetID      string
	ProblemID  string
	Difficulty string
	Date       time.Time
	Status     string
	Total      *int
	MaxTotal   int
	RubricID   string
	PathSlug   string
	Notes      string
	StartedAt  time.Time
	DeadlineAt time.Time
}

// RubricScore is one dimension's recorded score.
type RubricScore struct {
	Dimension string
	Score     int
}

// TrendPoint is one scored mock in the account's trend series in one course (R-MK3).
type TrendPoint struct {
	MockID     string
	SetID      string
	ProblemID  string
	Difficulty string
	Date       time.Time
	StartedAt  time.Time
	Total      int
	MaxTotal   int
}

// OutboxRow is one unsent domain event awaiting relay to NATS.
type OutboxRow struct {
	EventID string
	Subject string
	Payload []byte
}

// Store is the assessment persistence seam. Handlers + consumers depend on this
// interface so they can be unit-tested against an in-memory fake.
type Store interface {
	// CreateMock inserts a live session (status=live) in course pathSlug with the server
	// clock and returns it. startedAt / deadlineAt are computed by the caller so the
	// 45-minute window is server-authoritative (deadlineAt = startedAt + MockDuration).
	CreateMock(ctx context.Context, accountID, pathSlug, setID, problemID, difficulty string, startedAt, deadlineAt time.Time) (MockSession, error)
	// GetMock returns a session scoped to its owner plus its rubric scores (empty
	// until scored). ErrNotFound if it isn't the account's.
	GetMock(ctx context.Context, accountID, mockID string) (MockSession, []RubricScore, error)
	// ScoreMock records the seven rubric scores, computes the total server-side,
	// transitions the session live -> scored, and appends a mock_completed outbox row —
	// all in one transaction (R-MK2). It is idempotent: a re-submit on an
	// already-scored session returns the stored result without re-inserting or
	// re-emitting. ErrInvalidRubric for a bad rubric; ErrNotFound if not the account's.
	ScoreMock(ctx context.Context, accountID, mockID string, scores map[string]int, notes string) (MockSession, []RubricScore, error)
	// Trend returns the account's scored mocks in one course, oldest-first (R-MK3).
	Trend(ctx context.Context, accountID, pathSlug string) ([]TrendPoint, error)

	// ApplyProjection applies a decoded practice/review event to the S09 read-model
	// projections (coverage / mastery / heatmap / outcome-mix), deduped on event_id via
	// the inbox — all in ONE transaction so inbox <-> projected stays atomic
	// (effectively-once, ADR-0017). Handlers are a pure function of the event log (no
	// external reads, no wall-clock state branching) and every write is an idempotent
	// UPSERT, so out-of-order delivery is safe and a drop-and-replay from the stream
	// start rebuilds an identical result (ADR-0018). Returns whether the event was
	// freshly applied (false = a duplicate delivery that was a no-op).
	ApplyProjection(ctx context.Context, ev ProjectionEvent) (bool, error)

	// --- progress read model (S09) ---

	// SolvedCount is the "solved / 151" numerator: distinct problems solved.
	SolvedCount(ctx context.Context, accountID string) (int, error)
	// Retention returns the Day-7-retention inputs: `ladders` is problems that started a
	// spaced-repetition ladder; `resets` is how many times a ladder was reset by a fail.
	Retention(ctx context.Context, accountID string) (ladders, resets int, err error)
	// Heatmap returns the per-day revision-activity rows on/after `since` (UTC days).
	Heatmap(ctx context.Context, accountID string, since time.Time) ([]HeatmapDay, error)
	// Mastery returns every solved problem with its solve quality; the gateway rolls
	// these up by curriculum pattern (mastery bars) and week -> phase (completion table).
	Mastery(ctx context.Context, accountID string) ([]ProblemMastery, error)
	// OutcomeMix returns the first-solve outcome counts (clean/rough/assisted/miss).
	OutcomeMix(ctx context.Context, accountID string) (map[string]int, error)
	// MockStats returns the scored-mock roll-up in one course (count, average total,
	// best total).
	MockStats(ctx context.Context, accountID, pathSlug string) (MockStats, error)

	ListUnsentOutbox(ctx context.Context, limit int32) ([]OutboxRow, error)
	MarkOutboxSent(ctx context.Context, eventID string) error
	Ping(ctx context.Context) error

	// RecordDeadLetter is the consumers' dead-letter sink (events.DeadLetterSink):
	// one ids-only assessment.event_dead_letter row per (durable, event_id).
	RecordDeadLetter(ctx context.Context, dl events.DeadLetter) error
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

// CreateMock starts a live session in course pathSlug with a server-authoritative
// 45-minute window. The session row and its ordinal-1 mock_session_item row are written
// in ONE transaction (m1-02, M1a), both with the course written explicitly: the item is
// the session's problem, or NULL for a mixed set (problemID "" — v1 never pins an item).
func (s *PgStore) CreateMock(ctx context.Context, accountID, pathSlug, setID, problemID, difficulty string, startedAt, deadlineAt time.Time) (MockSession, error) {
	aid, err := parseUUID(accountID)
	if err != nil {
		return MockSession{}, fmt.Errorf("parse account id: %w", err)
	}
	if pathSlug == "" {
		// Never write '' or lean on the column default (m1-08 drops it): the handler
		// always resolves a course.
		return MockSession{}, errors.New("assessment: create mock without a course")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return MockSession{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := s.q.WithTx(tx)

	m, err := qtx.InsertMockSession(ctx, gen.InsertMockSessionParams{
		AccountID:      aid,
		SetID:          setID,
		ProblemID:      problemID,
		Difficulty:     pgtype.Text{String: difficulty, Valid: true},
		StartedAt:      tsz(startedAt),
		DeadlineAt:     tsz(deadlineAt),
		PathSlug:       pathSlug,
		RubricID:       pgtype.Text{String: RubricID, Valid: true},
		RubricSnapshot: []byte(RubricSnapshot),
		MaxTotal:       i4(MaxTotal),
	})
	if err != nil {
		return MockSession{}, fmt.Errorf("insert mock session: %w", err)
	}
	if err := qtx.InsertMockSessionItem(ctx, gen.InsertMockSessionItemParams{
		SessionID: m.ID,
		Ordinal:   1,
		ItemID:    pgtype.Text{String: problemID, Valid: problemID != ""},
		PathSlug:  m.PathSlug,
	}); err != nil {
		return MockSession{}, fmt.Errorf("insert mock session item: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return MockSession{}, fmt.Errorf("commit tx: %w", err)
	}
	return toMockSession(gen.GetMockSessionRow(m)), nil
}

// GetMock returns a session + its rubric scores, scoped to the owner.
func (s *PgStore) GetMock(ctx context.Context, accountID, mockID string) (MockSession, []RubricScore, error) {
	aid, err := parseUUID(accountID)
	if err != nil {
		return MockSession{}, nil, ErrNotFound
	}
	mid, err := parseUUID(mockID)
	if err != nil {
		return MockSession{}, nil, ErrNotFound
	}
	m, err := s.q.GetMockSession(ctx, gen.GetMockSessionParams{ID: mid, AccountID: aid})
	if errors.Is(err, pgx.ErrNoRows) {
		return MockSession{}, nil, ErrNotFound
	}
	if err != nil {
		return MockSession{}, nil, fmt.Errorf("get mock session: %w", err)
	}
	var scores []RubricScore
	if m.Status == StatusScored {
		scores, err = s.readRubric(ctx, s.q, m.ID)
		if err != nil {
			return MockSession{}, nil, err
		}
	}
	return toMockSession(m), scores, nil
}

// ScoreMock validates + records the seven-dimension rubric, computes the total
// server-side, latches the session scored, and emits mock_completed via the outbox —
// one tx.
func (s *PgStore) ScoreMock(ctx context.Context, accountID, mockID string, scores map[string]int, notes string) (MockSession, []RubricScore, error) {
	if err := ValidateRubric(scores); err != nil {
		return MockSession{}, nil, err
	}
	aid, err := parseUUID(accountID)
	if err != nil {
		return MockSession{}, nil, ErrNotFound
	}
	mid, err := parseUUID(mockID)
	if err != nil {
		return MockSession{}, nil, ErrNotFound
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return MockSession{}, nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := s.q.WithTx(tx)

	// Row-lock the session so two concurrent submits serialise: the second waits, then
	// re-reads status='scored' below and returns the stored result idempotently.
	m, err := qtx.GetMockSessionForUpdate(ctx, gen.GetMockSessionForUpdateParams{ID: mid, AccountID: aid})
	if errors.Is(err, pgx.ErrNoRows) {
		return MockSession{}, nil, ErrNotFound
	}
	if err != nil {
		return MockSession{}, nil, fmt.Errorf("lock mock session: %w", err)
	}

	// Idempotent re-submit: already scored -> return the stored rubric, no re-insert /
	// re-emit (a fresh event_id would defeat the downstream event_id dedupe).
	if m.Status == StatusScored {
		stored, rerr := s.readRubric(ctx, qtx, m.ID)
		if rerr != nil {
			return MockSession{}, nil, rerr
		}
		if err := tx.Commit(ctx); err != nil {
			return MockSession{}, nil, fmt.Errorf("commit tx: %w", err)
		}
		return toMockSession(gen.GetMockSessionRow(m)), stored, nil
	}

	total := TotalScore(scores)
	maxTotal := MaxTotal
	if m.MaxTotal.Valid {
		maxTotal = int(m.MaxTotal.Int32)
	}
	for _, dim := range Dimensions {
		if err := qtx.InsertRubricScore(ctx, gen.InsertRubricScoreParams{
			MockSessionID: m.ID,
			Dimension:     dim,
			Score:         int32(scores[dim]),
		}); err != nil {
			return MockSession{}, nil, fmt.Errorf("insert rubric score %s: %w", dim, err)
		}
	}
	if _, err := qtx.MarkMockScored(ctx, gen.MarkMockScoredParams{
		ID:        mid,
		AccountID: aid,
		Total:     i4(total),
		Notes:     notes,
		MaxTotal:  i4(MaxTotal),
	}); err != nil {
		// The FOR UPDATE lock + the status='live' read above guarantee this updates one
		// row; ErrNoRows here would mean a concurrent scorer beat us despite the lock.
		if errors.Is(err, pgx.ErrNoRows) {
			return MockSession{}, nil, ErrNotFound
		}
		return MockSession{}, nil, fmt.Errorf("mark mock scored: %w", err)
	}

	// Transactional outbox: the mock_completed fact is written in this same tx (R-MK2 /
	// events.md). The relay publishes it to XLEARN_ASSESSMENT.
	rubricID := RubricID
	if m.RubricID.Valid {
		rubricID = m.RubricID.String
	}
	if err := emitMockCompleted(ctx, qtx, accountID, m.PathSlug, mockID, rubricID, total, maxTotal, scores); err != nil {
		return MockSession{}, nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return MockSession{}, nil, fmt.Errorf("commit tx: %w", err)
	}

	// Reflect the committed transition without another round trip.
	m.Status = StatusScored
	m.Total = i4(total)
	m.MaxTotal = i4(maxTotal)
	m.ScoredBy = pgtype.Text{String: ScoredBySelf, Valid: true}
	m.Notes = notes
	out := make([]RubricScore, 0, NumDimensions)
	for _, dim := range Dimensions {
		out = append(out, RubricScore{Dimension: dim, Score: scores[dim]})
	}
	return toMockSession(gen.GetMockSessionRow(m)), out, nil
}

// Trend returns the account's scored mocks in one course, oldest-first.
func (s *PgStore) Trend(ctx context.Context, accountID, pathSlug string) ([]TrendPoint, error) {
	aid, err := parseUUID(accountID)
	if err != nil {
		return nil, ErrNotFound
	}
	rows, err := s.q.ListScoredMocks(ctx, gen.ListScoredMocksParams{AccountID: aid, PathSlug: pathSlug})
	if err != nil {
		return nil, fmt.Errorf("list scored mocks: %w", err)
	}
	out := make([]TrendPoint, 0, len(rows))
	for _, r := range rows {
		p := TrendPoint{
			MockID:     uuidString(r.ID),
			SetID:      r.SetID,
			ProblemID:  r.ProblemID,
			Difficulty: r.Difficulty.String,
			Date:       r.Date.Time,
			StartedAt:  r.StartedAt.Time,
			Total:      int(r.Total.Int32),
			MaxTotal:   MaxTotal,
		}
		if r.MaxTotal.Valid {
			p.MaxTotal = int(r.MaxTotal.Int32)
		}
		out = append(out, p)
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

// --- helpers ---

// readRubric reads a session's rubric rows into domain scores (via the given queries,
// so it works both on the pool and inside a tx).
func (s *PgStore) readRubric(ctx context.Context, q *gen.Queries, mockID pgtype.UUID) ([]RubricScore, error) {
	rows, err := q.ListRubricScores(ctx, mockID)
	if err != nil {
		return nil, fmt.Errorf("list rubric scores: %w", err)
	}
	out := make([]RubricScore, 0, len(rows))
	for _, r := range rows {
		out = append(out, RubricScore{Dimension: r.Dimension, Score: int(r.Score)})
	}
	return out, nil
}

// toMockSession maps a generated session row (every session read shares GetMockSession's
// column list; the other row types convert to it) to the plain-scalar domain type. The
// total reads total only; the max reads COALESCE(max_total, 35) and the rubric
// COALESCE(rubric_id, RubricID) — both are backfilled for every v1 row (m1-02).
func toMockSession(m gen.GetMockSessionRow) MockSession {
	ms := MockSession{
		ID:         uuidString(m.ID),
		AccountID:  uuidString(m.AccountID),
		SetID:      m.SetID,
		ProblemID:  m.ProblemID,
		Difficulty: m.Difficulty.String,
		Date:       m.Date.Time,
		Status:     m.Status,
		MaxTotal:   MaxTotal,
		RubricID:   RubricID,
		PathSlug:   m.PathSlug,
		Notes:      m.Notes,
		StartedAt:  m.StartedAt.Time,
		DeadlineAt: m.DeadlineAt.Time,
	}
	if m.Total.Valid {
		v := int(m.Total.Int32)
		ms.Total = &v
	}
	if m.MaxTotal.Valid {
		ms.MaxTotal = int(m.MaxTotal.Int32)
	}
	if m.RubricID.Valid {
		ms.RubricID = m.RubricID.String
	}
	return ms
}

// claimInbox records eventID in the inbox for idempotency, returning whether it was
// freshly claimed (true) or already present (false -> duplicate delivery).
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

// mockCompletedData is the xlearn.assessment.mock_completed payload. It is append-only
// (events.md): v1's mock_id, total_35 and rubric stay, and m1-03 (M1b; t1 §9) adds
// rubric_id, total, max_total and scored_by. total_35 is the same total, kept for a
// 35-point rubric (every v1 and DSA session); a session scored against another rubric
// width omits it rather than report a total that isn't out of 35.
func mockCompletedData(mockID, rubricID string, total, maxTotal int, rubric map[string]int) map[string]any {
	data := map[string]any{
		"mock_id":   mockID,
		"rubric":    rubric,
		"rubric_id": rubricID,
		"total":     total,
		"max_total": maxTotal,
		"scored_by": ScoredBySelf,
	}
	if maxTotal == MaxTotal {
		data["total_35"] = total
	}
	return data
}

// emitMockCompleted appends a xlearn.assessment.mock_completed event (the session's
// course in path_slug; mockCompletedData's payload) to the outbox inside the caller's
// transaction (events.md).
func emitMockCompleted(ctx context.Context, qtx *gen.Queries, accountID, pathSlug, mockID, rubricID string, total, maxTotal int, rubric map[string]int) error {
	return insertEvent(ctx, qtx, SubjectMockCompleted, accountID, pathSlug, mockCompletedData(mockID, rubricID, total, maxTotal, rubric))
}

// insertEvent marshals the event envelope — v2, carrying pathSlug (the session's
// course); events.NewEnvelope also enforces the size cap — and appends it to the outbox
// inside the caller's transaction (transactional outbox — never published inline).
// occurred_at is the write time, as in v1.
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

// tsz wraps a time in a non-null pgtype.Timestamptz.
func tsz(t time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: t, Valid: true} }

// i4 wraps an int in a non-null pgtype.Int4.
func i4(v int) pgtype.Int4 { return pgtype.Int4{Int32: int32(v), Valid: true} }

// ValidDimension reports whether d is one of the seven rubric dimensions.
func ValidDimension(d string) bool { return dimensionSet[d] }

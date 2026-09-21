package assessment

import (
	"context"
	"io"
	"log/slog"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/assessment/store"
	"github.com/sujaykumarsuman/xlearn/internal/platform/auth"
)

// testLogger discards output (handler/consumer tests assert on responses, not logs).
func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// fakeStore is an in-memory store.Store for handler + consumer unit tests (no
// database). Each method delegates to an overridable func so a test sets only what it
// exercises; an unset func panics, surfacing an unexpected call.
type fakeStore struct {
	createMock      func(ctx context.Context, accountID, setID, problemID, difficulty string, startedAt, deadlineAt time.Time) (store.MockSession, error)
	getMock         func(ctx context.Context, accountID, mockID string) (store.MockSession, []store.RubricScore, error)
	scoreMock       func(ctx context.Context, accountID, mockID string, scores map[string]int, notes string) (store.MockSession, []store.RubricScore, error)
	trend           func(ctx context.Context, accountID string) ([]store.TrendPoint, error)
	applyProjection func(ctx context.Context, ev store.ProjectionEvent) (bool, error)
	solvedCount     func(ctx context.Context, accountID string) (int, error)
	retention       func(ctx context.Context, accountID string) (int, int, error)
	heatmap         func(ctx context.Context, accountID string, since time.Time) ([]store.HeatmapDay, error)
	mastery         func(ctx context.Context, accountID string) ([]store.ProblemMastery, error)
	outcomeMix      func(ctx context.Context, accountID string) (map[string]int, error)
	mockStats       func(ctx context.Context, accountID string) (store.MockStats, error)

	pingErr error
}

func (f *fakeStore) CreateMock(ctx context.Context, accountID, setID, problemID, difficulty string, startedAt, deadlineAt time.Time) (store.MockSession, error) {
	return f.createMock(ctx, accountID, setID, problemID, difficulty, startedAt, deadlineAt)
}

func (f *fakeStore) GetMock(ctx context.Context, accountID, mockID string) (store.MockSession, []store.RubricScore, error) {
	return f.getMock(ctx, accountID, mockID)
}

func (f *fakeStore) ScoreMock(ctx context.Context, accountID, mockID string, scores map[string]int, notes string) (store.MockSession, []store.RubricScore, error) {
	return f.scoreMock(ctx, accountID, mockID, scores, notes)
}

func (f *fakeStore) Trend(ctx context.Context, accountID string) ([]store.TrendPoint, error) {
	return f.trend(ctx, accountID)
}

func (f *fakeStore) ApplyProjection(ctx context.Context, ev store.ProjectionEvent) (bool, error) {
	return f.applyProjection(ctx, ev)
}

func (f *fakeStore) SolvedCount(ctx context.Context, accountID string) (int, error) {
	return f.solvedCount(ctx, accountID)
}

func (f *fakeStore) Retention(ctx context.Context, accountID string) (int, int, error) {
	return f.retention(ctx, accountID)
}

func (f *fakeStore) Heatmap(ctx context.Context, accountID string, since time.Time) ([]store.HeatmapDay, error) {
	return f.heatmap(ctx, accountID, since)
}

func (f *fakeStore) Mastery(ctx context.Context, accountID string) ([]store.ProblemMastery, error) {
	return f.mastery(ctx, accountID)
}

func (f *fakeStore) OutcomeMix(ctx context.Context, accountID string) (map[string]int, error) {
	return f.outcomeMix(ctx, accountID)
}

func (f *fakeStore) MockStats(ctx context.Context, accountID string) (store.MockStats, error) {
	return f.mockStats(ctx, accountID)
}

func (f *fakeStore) ListUnsentOutbox(context.Context, int32) ([]store.OutboxRow, error) {
	return nil, nil
}

func (f *fakeStore) MarkOutboxSent(context.Context, string) error { return nil }

func (f *fakeStore) Ping(context.Context) error { return f.pingErr }

// fakeVerifier accepts any non-empty token, returning claims for a fixed subject,
// unless err is set (to exercise the invalid-token path).
type fakeVerifier struct {
	subject string
	err     error
}

func (v fakeVerifier) Verify(_ context.Context, _ string) (auth.Claims, error) {
	if v.err != nil {
		return auth.Claims{}, v.err
	}
	return auth.Claims{Subject: v.subject, Audience: "assessment", Roles: []string{"learner"}}, nil
}

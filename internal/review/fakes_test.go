package review

import (
	"context"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/platform/auth"
	"github.com/sujaykumarsuman/xlearn/internal/review/store"
)

// fakeStore is an in-memory store.Store for handler + consumer unit tests (no
// database). Each method delegates to an overridable func so a test sets only what it
// exercises.
type fakeStore struct {
	problemSolved func(ctx context.Context, eventID, accountID, problemID, outcome string, firstSolve bool, occurredAt time.Time) (int, error)
	revealedEarly func(ctx context.Context, eventID, accountID, problemID string, occurredAt time.Time) (int, error)
	score         func(ctx context.Context, accountID, itemID string, in store.ScoreInput) (store.ScoreResult, error)
	dueQueue      func(ctx context.Context, accountID string, limit int) ([]store.DueItem, error)
	sweep         func(ctx context.Context, batch int) (int, error)

	// S07 seams.
	listMistakes    func(ctx context.Context, accountID, status string) ([]store.Mistake, error)
	getMistake      func(ctx context.Context, accountID, id string) (store.Mistake, error)
	createMistake   func(ctx context.Context, accountID string, in store.MistakeInput) (store.Mistake, error)
	updateMistake   func(ctx context.Context, accountID, id string, in store.MistakePatch) (store.Mistake, error)
	weakAreaCurrent func(ctx context.Context, accountID string) (store.WeakArea, bool, error)
	listReminders   func(ctx context.Context, accountID string, limit int) ([]store.Reminder, error)
	handleDue       func(ctx context.Context, eventID, accountID, kind string, dueAt time.Time) (bool, error)

	pingErr error
}

func (f *fakeStore) HandleProblemSolved(ctx context.Context, eventID, accountID, problemID, outcome string, firstSolve bool, occurredAt time.Time) (int, error) {
	return f.problemSolved(ctx, eventID, accountID, problemID, outcome, firstSolve, occurredAt)
}

func (f *fakeStore) HandleSolutionRevealedEarly(ctx context.Context, eventID, accountID, problemID string, occurredAt time.Time) (int, error) {
	return f.revealedEarly(ctx, eventID, accountID, problemID, occurredAt)
}

func (f *fakeStore) Score(ctx context.Context, accountID, itemID string, in store.ScoreInput) (store.ScoreResult, error) {
	return f.score(ctx, accountID, itemID, in)
}

func (f *fakeStore) DueQueue(ctx context.Context, accountID string, limit int) ([]store.DueItem, error) {
	return f.dueQueue(ctx, accountID, limit)
}

func (f *fakeStore) Sweep(ctx context.Context, batch int) (int, error) { return f.sweep(ctx, batch) }

func (f *fakeStore) ListMistakes(ctx context.Context, accountID, status string) ([]store.Mistake, error) {
	return f.listMistakes(ctx, accountID, status)
}

func (f *fakeStore) GetMistake(ctx context.Context, accountID, id string) (store.Mistake, error) {
	return f.getMistake(ctx, accountID, id)
}

func (f *fakeStore) CreateMistake(ctx context.Context, accountID string, in store.MistakeInput) (store.Mistake, error) {
	return f.createMistake(ctx, accountID, in)
}

func (f *fakeStore) UpdateMistake(ctx context.Context, accountID, id string, in store.MistakePatch) (store.Mistake, error) {
	return f.updateMistake(ctx, accountID, id, in)
}

func (f *fakeStore) AccountsWithMistakes(context.Context) ([]string, error) { return nil, nil }

func (f *fakeStore) SaveWeakAreaSnapshot(context.Context, string, time.Time, string, map[string]int) error {
	return nil
}

func (f *fakeStore) CountOpenMistakesByCategory(context.Context, string, time.Time, time.Time) (map[string]int, error) {
	return nil, nil
}

func (f *fakeStore) WeakAreaCurrent(ctx context.Context, accountID string) (store.WeakArea, bool, error) {
	return f.weakAreaCurrent(ctx, accountID)
}

func (f *fakeStore) HandleRevisionDue(ctx context.Context, eventID, accountID, kind string, dueAt time.Time) (bool, error) {
	return f.handleDue(ctx, eventID, accountID, kind, dueAt)
}

func (f *fakeStore) ListDueReminders(ctx context.Context, accountID string, limit int) ([]store.Reminder, error) {
	return f.listReminders(ctx, accountID, limit)
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
	return auth.Claims{Subject: v.subject, Audience: "review", Roles: []string{"learner"}}, nil
}

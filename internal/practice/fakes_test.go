package practice

import (
	"context"

	"github.com/sujaykumarsuman/xlearn/internal/platform/auth"
	"github.com/sujaykumarsuman/xlearn/internal/practice/store"
)

// fakeStore is an in-memory store.Store for handler unit tests (no database). Each
// method delegates to an overridable func so a test sets only what it exercises.
type fakeStore struct {
	getState   func(ctx context.Context, accountID, problemID string) (store.State, error)
	listStates func(ctx context.Context, accountID string, ids []string) (map[string]store.State, error)
	start      func(ctx context.Context, accountID, problemID string) (store.State, error)
	reveal     func(ctx context.Context, accountID, problemID string) (store.RevealResult, error)
	logOutcome func(ctx context.Context, accountID, problemID, value string) (store.State, error)
	pingErr    error
}

func (f *fakeStore) GetState(ctx context.Context, a, p string) (store.State, error) {
	return f.getState(ctx, a, p)
}

func (f *fakeStore) ListStates(ctx context.Context, a string, ids []string) (map[string]store.State, error) {
	return f.listStates(ctx, a, ids)
}

func (f *fakeStore) StartAttempt(ctx context.Context, a, p string) (store.State, error) {
	return f.start(ctx, a, p)
}

func (f *fakeStore) Reveal(ctx context.Context, a, p string) (store.RevealResult, error) {
	return f.reveal(ctx, a, p)
}

func (f *fakeStore) LogOutcome(ctx context.Context, a, p, v string) (store.State, error) {
	return f.logOutcome(ctx, a, p, v)
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
	return auth.Claims{Subject: v.subject, Audience: "practice", Roles: []string{"learner"}}, nil
}

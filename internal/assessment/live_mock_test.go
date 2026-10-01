package assessment

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/assessment/store"
)

// m1-07: GET /mocks/live — the account's live session, {"live": null} without one, a
// 500 when the store fails (the gateway then fails the coach chat closed), and JWT-scoped.
func TestLiveMock(t *testing.T) {
	started := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
	var gotAccount string
	live := true
	fs := &fakeStore{liveMock: func(_ context.Context, accountID string) (store.MockSession, bool, error) {
		gotAccount = accountID
		if !live {
			return store.MockSession{}, false, nil
		}
		return store.MockSession{ID: "m-1", PathSlug: "dsa", Status: store.StatusLive,
			StartedAt: started, DeadlineAt: started.Add(store.MockDuration)}, true, nil
	}}
	h := newTestService(fs).Handler()

	w := do(t, h, http.MethodGet, "/mocks/live", "", true)
	if w.Code != http.StatusOK || gotAccount != testAccount {
		t.Fatalf("status %d account %q (%s)", w.Code, gotAccount, w.Body.String())
	}
	var body struct {
		Live *struct {
			ID, PathSlug, StartedAt, DeadlineAt string
		} `json:"live"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || body.Live == nil {
		t.Fatalf("body %s: %v", w.Body.String(), err)
	}
	if body.Live.ID != "m-1" || body.Live.PathSlug != "dsa" || body.Live.StartedAt != "2026-10-01T09:00:00Z" ||
		body.Live.DeadlineAt != started.Add(store.MockDuration).Format(time.RFC3339) {
		t.Errorf("live = %+v", body.Live)
	}

	live = false
	w = do(t, h, http.MethodGet, "/mocks/live", "", true)
	if w.Code != http.StatusOK || w.Body.String() != "{\"live\":null}\n" {
		t.Errorf("no live mock: %d %q", w.Code, w.Body.String())
	}

	fs.liveMock = func(context.Context, string) (store.MockSession, bool, error) {
		return store.MockSession{}, false, errors.New("db down")
	}
	if w = do(t, h, http.MethodGet, "/mocks/live", "", true); w.Code != http.StatusInternalServerError {
		t.Errorf("store failure: %d, want 500", w.Code)
	}
	if w = do(t, h, http.MethodGet, "/mocks/live", "", false); w.Code != http.StatusUnauthorized {
		t.Errorf("no token: %d, want 401", w.Code)
	}
}

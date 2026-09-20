package identity

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/identity/store"
	"github.com/sujaykumarsuman/xlearn/internal/platform/auth"
)

// fakeStore is an in-memory store.Store for handler tests (no database).
type fakeStore struct {
	mu         sync.Mutex
	seq        int
	accounts   map[string]store.Account
	byProvider map[string]string // provider|providerUserID -> accountID
	onboarding map[string]store.Onboarding
	sessions   map[string]store.Session
	outbox     []store.OutboxRow
	pingErr    error
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		accounts:   map[string]store.Account{},
		byProvider: map[string]string{},
		onboarding: map[string]store.Onboarding{},
		sessions:   map[string]store.Session{},
	}
}

func (f *fakeStore) FindOrCreateAccount(_ context.Context, in store.OAuthUpsert) (store.Account, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := in.Provider + "|" + in.ProviderUserID
	if id, ok := f.byProvider[key]; ok {
		return f.accounts[id], false, nil
	}
	f.seq++
	id := fmt.Sprintf("acct-%d", f.seq)
	acct := store.Account{ID: id, DisplayName: in.DisplayName, Email: in.Email, Timezone: "UTC", CreatedAt: time.Now()}
	f.accounts[id] = acct
	f.byProvider[key] = id
	f.onboarding[id] = store.Onboarding{AccountID: id}
	f.outbox = append(f.outbox, store.OutboxRow{
		EventID: fmt.Sprintf("evt-%d", f.seq),
		Subject: store.SubjectAccountCreated,
		Payload: []byte(`{"account_id":"` + id + `","data":{"provider":"` + in.Provider + `"}}`),
	})
	return acct, true, nil
}

func (f *fakeStore) GetAccount(_ context.Context, id string) (store.Account, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	a, ok := f.accounts[id]
	if !ok {
		return store.Account{}, store.ErrNotFound
	}
	return a, nil
}

func (f *fakeStore) GetOnboarding(_ context.Context, accountID string) (store.Onboarding, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	o, ok := f.onboarding[accountID]
	if !ok {
		return store.Onboarding{}, store.ErrNotFound
	}
	return o, nil
}

func (f *fakeStore) SetOnboardingPath(_ context.Context, accountID, path string) (store.Onboarding, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	o, ok := f.onboarding[accountID]
	if !ok {
		return store.Onboarding{}, store.ErrNotFound
	}
	o.PathChosen = path
	f.onboarding[accountID] = o
	return o, nil
}

func (f *fakeStore) CreateSession(_ context.Context, id, accountID string, expiresAt time.Time) (store.Session, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s := store.Session{ID: id, AccountID: accountID, CreatedAt: time.Now(), ExpiresAt: expiresAt}
	f.sessions[id] = s
	return s, nil
}

func (f *fakeStore) GetValidSession(_ context.Context, id string) (store.Session, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s, ok := f.sessions[id]
	if !ok || time.Now().After(s.ExpiresAt) {
		return store.Session{}, store.ErrNotFound
	}
	return s, nil
}

func (f *fakeStore) RevokeSession(_ context.Context, id string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.sessions[id]; !ok {
		return false, nil
	}
	delete(f.sessions, id)
	return true, nil
}

func (f *fakeStore) ListUnsentOutbox(_ context.Context, limit int32) ([]store.OutboxRow, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if int(limit) < len(f.outbox) {
		return append([]store.OutboxRow(nil), f.outbox[:limit]...), nil
	}
	return append([]store.OutboxRow(nil), f.outbox...), nil
}

func (f *fakeStore) MarkOutboxSent(_ context.Context, eventID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := f.outbox[:0]
	for _, r := range f.outbox {
		if r.EventID != eventID {
			out = append(out, r)
		}
	}
	f.outbox = out
	return nil
}

func (f *fakeStore) Ping(_ context.Context) error { return f.pingErr }

// fakeVerifier accepts one known token and returns fixed claims (crypto is tested
// in the auth package).
type fakeVerifier struct {
	token  string
	claims auth.Claims
}

func (v fakeVerifier) Verify(_ context.Context, token string) (auth.Claims, error) {
	if token != v.token {
		return auth.Claims{}, auth.ErrUnauthenticated
	}
	return v.claims, nil
}

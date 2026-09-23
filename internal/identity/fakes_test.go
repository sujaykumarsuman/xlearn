package identity

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/identity/store"
	"github.com/sujaykumarsuman/xlearn/internal/platform/auth"
)

// fakeStore is an in-memory store.Store for handler tests (no database).
type fakeStore struct {
	mu          sync.Mutex
	seq         int
	accounts    map[string]store.Account
	byProvider  map[string]string   // provider|providerUserID -> accountID
	emailIndex  map[string]string   // lower(email) -> accountID
	usernameIdx map[string]string   // lower(username) -> accountID
	providers   map[string][]string // accountID -> linked providers
	onboarding  map[string]store.Onboarding
	sessions    map[string]store.Session
	enrollments map[string][]store.Enrollment
	outbox      []store.OutboxRow
	pingErr     error
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		accounts:    map[string]store.Account{},
		byProvider:  map[string]string{},
		emailIndex:  map[string]string{},
		usernameIdx: map[string]string{},
		providers:   map[string][]string{},
		onboarding:  map[string]store.Onboarding{},
		sessions:    map[string]store.Session{},
		enrollments: map[string][]store.Enrollment{},
	}
}

func (f *fakeStore) FindOrCreateAccount(_ context.Context, in store.OAuthUpsert) (store.Account, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := in.Provider + "|" + in.ProviderUserID
	if id, ok := f.byProvider[key]; ok {
		return f.accounts[id], false, nil
	}
	// Auto-link by verified email (mirrors the real store).
	if in.Email != "" {
		if id, ok := f.emailIndex[strings.ToLower(in.Email)]; ok {
			f.byProvider[key] = id
			f.providers[id] = append(f.providers[id], in.Provider)
			return f.accounts[id], false, nil
		}
	}
	f.seq++
	id := fmt.Sprintf("acct-%d", f.seq)
	acct := store.Account{ID: id, DisplayName: in.DisplayName, Email: in.Email, Timezone: "UTC", CreatedAt: time.Now()}
	f.accounts[id] = acct
	f.byProvider[key] = id
	if in.Email != "" {
		f.emailIndex[strings.ToLower(in.Email)] = id
	}
	f.providers[id] = append(f.providers[id], in.Provider)
	f.onboarding[id] = store.Onboarding{AccountID: id}
	f.outbox = append(f.outbox, store.OutboxRow{
		EventID: fmt.Sprintf("evt-%d", f.seq),
		Subject: store.SubjectAccountCreated,
		Payload: []byte(`{"account_id":"` + id + `","data":{"provider":"` + in.Provider + `"}}`),
	})
	return acct, true, nil
}

func (f *fakeStore) GetAccountByEmail(_ context.Context, email string) (store.Account, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	id, ok := f.emailIndex[strings.ToLower(strings.TrimSpace(email))]
	if !ok {
		return store.Account{}, store.ErrNotFound
	}
	return f.accounts[id], nil
}

func (f *fakeStore) GetAccountByUsername(_ context.Context, username string) (store.Account, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	id, ok := f.usernameIdx[strings.ToLower(strings.TrimSpace(username))]
	if !ok {
		return store.Account{}, store.ErrNotFound
	}
	return f.accounts[id], nil
}

func (f *fakeStore) SetUsername(_ context.Context, id, username string) (store.Account, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	a, ok := f.accounts[id]
	if !ok {
		return store.Account{}, store.ErrNotFound
	}
	lk := strings.ToLower(username)
	if owner, taken := f.usernameIdx[lk]; taken && owner != id {
		return store.Account{}, store.ErrUsernameTaken
	}
	if a.Username != "" {
		delete(f.usernameIdx, strings.ToLower(a.Username))
	}
	a.Username = username
	f.accounts[id] = a
	f.usernameIdx[lk] = id
	return a, nil
}

func (f *fakeStore) CreateEmailAccount(_ context.Context, email, passwordHash, displayName string) (store.Account, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	lk := strings.ToLower(email)
	if _, ok := f.emailIndex[lk]; ok {
		return store.Account{}, store.ErrEmailTaken
	}
	f.seq++
	id := fmt.Sprintf("acct-%d", f.seq)
	acct := store.Account{ID: id, DisplayName: displayName, Email: email, PasswordHash: passwordHash, Timezone: "UTC", CreatedAt: time.Now()}
	f.accounts[id] = acct
	f.emailIndex[lk] = id
	f.onboarding[id] = store.Onboarding{AccountID: id}
	f.outbox = append(f.outbox, store.OutboxRow{
		EventID: fmt.Sprintf("evt-%d", f.seq),
		Subject: store.SubjectAccountCreated,
		Payload: []byte(`{"account_id":"` + id + `","data":{"provider":"email"}}`),
	})
	return acct, nil
}

func (f *fakeStore) SetAccountPassword(_ context.Context, id, passwordHash string) (store.Account, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	a, ok := f.accounts[id]
	if !ok {
		return store.Account{}, store.ErrNotFound
	}
	a.PasswordHash = passwordHash
	f.accounts[id] = a
	return a, nil
}

func (f *fakeStore) LinkOAuth(_ context.Context, accountID, provider, providerUserID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := provider + "|" + providerUserID
	if _, ok := f.byProvider[key]; ok {
		return store.ErrConflict
	}
	if _, ok := f.accounts[accountID]; !ok {
		return store.ErrNotFound
	}
	f.byProvider[key] = accountID
	f.providers[accountID] = append(f.providers[accountID], provider)
	return nil
}

func (f *fakeStore) UnlinkOAuth(_ context.Context, accountID, provider string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	removed := false
	kept := make([]string, 0, len(f.providers[accountID]))
	for _, p := range f.providers[accountID] {
		if p == provider {
			removed = true
			continue
		}
		kept = append(kept, p)
	}
	f.providers[accountID] = kept
	for k, id := range f.byProvider {
		if id == accountID && strings.HasPrefix(k, provider+"|") {
			delete(f.byProvider, k)
		}
	}
	return removed, nil
}

func (f *fakeStore) ListOAuthProviders(_ context.Context, accountID string) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.providers[accountID]...), nil
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

func (f *fakeStore) UpdateAccount(_ context.Context, id string, in store.AccountUpdate) (store.Account, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	a, ok := f.accounts[id]
	if !ok {
		return store.Account{}, store.ErrNotFound
	}
	if in.DisplayName != nil {
		a.DisplayName = *in.DisplayName
	}
	if in.Timezone != nil {
		a.Timezone = *in.Timezone
	}
	if in.StudyBudget != nil {
		a.StudyBudget = in.StudyBudget
	}
	if in.Reminders != nil {
		a.Reminders = in.Reminders
	}
	f.accounts[id] = a
	return a, nil
}

func (f *fakeStore) SetOnboardingBudget(_ context.Context, accountID string, budgetJSON []byte) (store.Onboarding, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	o, ok := f.onboarding[accountID]
	if !ok {
		return store.Onboarding{}, store.ErrNotFound
	}
	if a, ok := f.accounts[accountID]; ok {
		a.StudyBudget = budgetJSON
		f.accounts[accountID] = a
	}
	o.BudgetSet = true
	f.onboarding[accountID] = o
	return o, nil
}

func (f *fakeStore) CompleteOnboarding(_ context.Context, accountID string) (store.Onboarding, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	o, ok := f.onboarding[accountID]
	if !ok {
		return store.Onboarding{}, store.ErrNotFound
	}
	if o.CompletedAt.IsZero() {
		o.CompletedAt = time.Now()
	}
	f.onboarding[accountID] = o
	return o, nil
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

func (f *fakeStore) StartEnrollment(_ context.Context, accountID, pathSlug string) (store.Enrollment, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, e := range f.enrollments[accountID] {
		if e.PathSlug == pathSlug {
			e.Status = "active"
			return e, nil // idempotent: keep the original started_at
		}
	}
	e := store.Enrollment{AccountID: accountID, PathSlug: pathSlug, Status: "active", StartedAt: time.Now()}
	f.enrollments[accountID] = append(f.enrollments[accountID], e)
	return e, nil
}

func (f *fakeStore) ListEnrollments(_ context.Context, accountID string) ([]store.Enrollment, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]store.Enrollment(nil), f.enrollments[accountID]...), nil
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

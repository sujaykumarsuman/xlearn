package coach

import (
	"context"
	"crypto/rand"
	"io"
	"log/slog"
	"sort"
	"sync"

	"github.com/sujaykumarsuman/xlearn/internal/coach/store"
	"github.com/sujaykumarsuman/xlearn/internal/platform/auth"
	"github.com/sujaykumarsuman/xlearn/internal/platform/secrets"
)

// fakeVerifier returns fixed claims for any non-empty token, so handler tests don't need
// a live JWKS endpoint. The subject is the account id under test.
type fakeVerifier struct{ subject string }

func (f fakeVerifier) Verify(_ context.Context, token string) (auth.Claims, error) {
	if token == "" {
		return auth.Claims{}, auth.ErrUnauthenticated
	}
	return auth.Claims{Subject: f.subject, Audience: "coach", Roles: []string{"learner"}}, nil
}

// memStore is an in-memory Store for unit tests. keys is account -> provider -> config,
// mirroring the multi-provider schema (one key per provider, exactly one default).
type memStore struct {
	mu       sync.Mutex
	keys     map[string]map[string]store.KeyConfig // accountID -> provider -> config
	threads  map[string]string                     // accountID|context -> threadID
	messages map[string][]store.Message            // threadID -> messages
	threadOf map[string]string                     // threadID -> accountID|context (bookkeeping)
	nextID   int
}

func newMemStore() *memStore {
	return &memStore{
		keys:     map[string]map[string]store.KeyConfig{},
		threads:  map[string]string{},
		messages: map[string][]store.Message{},
		threadOf: map[string]string{},
	}
}

func (m *memStore) ListKeys(_ context.Context, accountID string) ([]store.KeyConfig, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ps := m.keys[accountID]
	out := make([]store.KeyConfig, 0, len(ps))
	for _, k := range ps {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Provider < out[j].Provider })
	return out, nil
}

func (m *memStore) GetKey(_ context.Context, accountID, provider string) (store.KeyConfig, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	k, ok := m.keys[accountID][provider]
	if !ok {
		return store.KeyConfig{}, store.ErrNotFound
	}
	return k, nil
}

func (m *memStore) GetDefaultKey(_ context.Context, accountID string) (store.KeyConfig, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, k := range m.keys[accountID] {
		if k.IsDefault {
			return k, nil
		}
	}
	return store.KeyConfig{}, store.ErrNotFound
}

func (m *memStore) PutKey(_ context.Context, k store.KeyConfig) (store.KeyConfig, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.keys[k.AccountID] == nil {
		m.keys[k.AccountID] = map[string]store.KeyConfig{}
	}
	ps := m.keys[k.AccountID]
	k.Enabled = true
	if existing, ok := ps[k.Provider]; ok {
		k.IsDefault = existing.IsDefault // replacing a provider keeps its default status
	} else {
		k.IsDefault = len(ps) == 0 // the account's first key becomes the default
	}
	ps[k.Provider] = k
	return k, nil
}

func (m *memStore) UpdateKeyMeta(_ context.Context, accountID, provider, model, name string) (store.KeyConfig, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	k, ok := m.keys[accountID][provider]
	if !ok {
		return store.KeyConfig{}, store.ErrNotFound
	}
	k.DefaultModel = model
	k.Name = name
	m.keys[accountID][provider] = k
	return k, nil
}

func (m *memStore) SetKeyEnabled(_ context.Context, accountID, provider string, enabled bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	k, ok := m.keys[accountID][provider]
	if !ok {
		return store.ErrNotFound
	}
	k.Enabled = enabled
	m.keys[accountID][provider] = k
	return nil
}

func (m *memStore) SetDefault(_ context.Context, accountID, provider string) (store.KeyConfig, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ps := m.keys[accountID]
	if _, ok := ps[provider]; !ok {
		return store.KeyConfig{}, store.ErrNotFound
	}
	var def store.KeyConfig
	for p, k := range ps {
		k.IsDefault = p == provider
		ps[p] = k
		if k.IsDefault {
			def = k
		}
	}
	return def, nil
}

func (m *memStore) DeleteKey(_ context.Context, accountID, provider string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	ps := m.keys[accountID]
	k, ok := ps[provider]
	if !ok {
		return store.ErrNotFound
	}
	wasDefault := k.IsDefault
	delete(ps, provider)
	if wasDefault && len(ps) > 0 {
		// Promote a survivor (lowest provider id, for deterministic tests) to default.
		pick := ""
		for p := range ps {
			if pick == "" || p < pick {
				pick = p
			}
		}
		pk := ps[pick]
		pk.IsDefault = true
		ps[pick] = pk
	}
	return nil
}

func (m *memStore) EnsureThread(_ context.Context, accountID, pageContext string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := accountID + "|" + pageContext
	if id, ok := m.threads[key]; ok {
		return id, nil
	}
	m.nextID++
	id := "thread-" + itoa(m.nextID)
	m.threads[key] = id
	m.threadOf[id] = key
	return id, nil
}

func (m *memStore) ThreadHistory(_ context.Context, accountID, pageContext string) ([]store.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.threads[accountID+"|"+pageContext]
	if !ok {
		return []store.Message{}, nil
	}
	return append([]store.Message(nil), m.messages[id]...), nil
}

func (m *memStore) AppendMessage(_ context.Context, threadID, role, content string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages[threadID] = append(m.messages[threadID], store.Message{Role: role, Content: content})
	return nil
}

func (m *memStore) Ping(context.Context) error { return nil }

// messagesFor returns the persisted messages for (account, context).
func (m *memStore) messagesFor(accountID, pageContext string) []store.Message {
	msgs, _ := m.ThreadHistory(context.Background(), accountID, pageContext)
	return msgs
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

// testCipher builds a Cipher with a random master key.
func testCipher() *secrets.Cipher {
	m := make([]byte, secrets.MasterKeySize)
	_, _ = rand.Read(m)
	c, _ := secrets.NewCipher(m)
	return c
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

package coach

import (
	"context"
	"crypto/rand"
	"io"
	"log/slog"
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

// memStore is an in-memory Store for unit tests.
type memStore struct {
	mu       sync.Mutex
	keys     map[string]store.KeyConfig // accountID -> config
	threads  map[string]string          // accountID|context -> threadID
	messages map[string][]store.Message // threadID -> messages
	threadOf map[string]string          // threadID -> accountID|context (bookkeeping)
	nextID   int
}

func newMemStore() *memStore {
	return &memStore{
		keys:     map[string]store.KeyConfig{},
		threads:  map[string]string{},
		messages: map[string][]store.Message{},
		threadOf: map[string]string{},
	}
}

func (m *memStore) PutKey(_ context.Context, k store.KeyConfig) (store.KeyConfig, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	k.Enabled = true
	m.keys[k.AccountID] = k
	return k, nil
}

func (m *memStore) GetKey(_ context.Context, accountID string) (store.KeyConfig, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	k, ok := m.keys[accountID]
	if !ok {
		return store.KeyConfig{}, store.ErrNotFound
	}
	return k, nil
}

func (m *memStore) DeleteKey(_ context.Context, accountID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.keys[accountID]; !ok {
		return store.ErrNotFound
	}
	delete(m.keys, accountID)
	return nil
}

func (m *memStore) SetKeyEnabled(_ context.Context, accountID string, enabled bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	k, ok := m.keys[accountID]
	if !ok {
		return store.ErrNotFound
	}
	k.Enabled = enabled
	m.keys[accountID] = k
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

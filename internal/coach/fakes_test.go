package coach

import (
	"context"
	"crypto/rand"
	"io"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/coach/store"
	"github.com/sujaykumarsuman/xlearn/internal/platform/auth"
	"github.com/sujaykumarsuman/xlearn/internal/platform/secrets"
)

// fakeVerifier returns fixed claims for any non-empty token, so handler tests don't need
// a live JWKS endpoint. The subject is the account id under test; roles overrides the
// default ["learner"] role set (the role-check tests).
type fakeVerifier struct {
	subject string
	roles   []string
}

func (f fakeVerifier) Verify(_ context.Context, token string) (auth.Claims, error) {
	if token == "" {
		return auth.Claims{}, auth.ErrUnauthenticated
	}
	roles := f.roles
	if roles == nil {
		roles = []string{"learner"}
	}
	return auth.Claims{Subject: f.subject, Audience: "coach", Roles: roles}, nil
}

// memStore is an in-memory Store for unit tests. keys is account -> provider -> config,
// mirroring the multi-provider schema (one key per provider). defaults is
// account -> feature -> default row, mirroring coach.key_default — including the
// asymmetry the handler tests exercise: `coach` is maintained implicitly (first key in,
// survivor promoted on delete) while `interview` is only ever set explicitly.
//
// IsDefault is DERIVED from the coach default row on every read, never stored on the key,
// so the fake can't accidentally pass a test the real store (which reads key_default)
// would fail.
type memStore struct {
	mu       sync.Mutex
	keys     map[string]map[string]store.KeyConfig      // accountID -> provider -> config
	defaults map[string]map[string]store.FeatureDefault // accountID -> feature -> default
	threads  map[string]string                          // accountID|context -> threadID
	messages map[string][]store.Message                 // threadID -> messages
	usage    map[string][]store.MessageUsage            // threadID -> assistant-turn usage
	threadOf map[string]string                          // threadID -> accountID|context (bookkeeping)
	pathOf   map[string]string                          // threadID -> path_slug ("" = NULL)
	metas    map[string][]store.MessageMeta             // threadID -> each message's meta, parallel to messages
	quota    map[string]int                             // accountID|YYYY-MM-DD (UTC) -> n (coach.message_quota_day)
	// quotaErr, when set, fails every daily-quota call (the store-error path of L18).
	quotaErr error
	nextID   int
}

func newMemStore() *memStore {
	return &memStore{
		keys:     map[string]map[string]store.KeyConfig{},
		defaults: map[string]map[string]store.FeatureDefault{},
		threads:  map[string]string{},
		messages: map[string][]store.Message{},
		usage:    map[string][]store.MessageUsage{},
		threadOf: map[string]string{},
		pathOf:   map[string]string{},
		metas:    map[string][]store.MessageMeta{},
		quota:    map[string]int{},
	}
}

// setDefaultLocked writes one feature's default row (callers hold m.mu). An empty model
// takes the key's own default_model, as UpsertKeyDefault does.
func (m *memStore) setDefaultLocked(accountID, feature string, k store.KeyConfig, model string) {
	if m.defaults[accountID] == nil {
		m.defaults[accountID] = map[string]store.FeatureDefault{}
	}
	if model == "" {
		model = k.DefaultModel
	}
	m.defaults[accountID][feature] = store.FeatureDefault{
		Feature: feature, Provider: k.Provider, Model: model, KeyID: k.Provider,
	}
}

// withDefaultFlagLocked stamps IsDefault from the coach default row.
func (m *memStore) withDefaultFlagLocked(accountID string, k store.KeyConfig) store.KeyConfig {
	d, ok := m.defaults[accountID][store.FeatureCoach]
	k.IsDefault = ok && d.Provider == k.Provider
	return k
}

func (m *memStore) ListKeys(_ context.Context, accountID string) ([]store.KeyConfig, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ps := m.keys[accountID]
	out := make([]store.KeyConfig, 0, len(ps))
	for _, k := range ps {
		out = append(out, m.withDefaultFlagLocked(accountID, k))
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
	return m.withDefaultFlagLocked(accountID, k), nil
}

func (m *memStore) GetDefaultKey(_ context.Context, accountID, feature string) (store.KeyConfig, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	d, ok := m.defaults[accountID][feature]
	if !ok {
		return store.KeyConfig{}, store.ErrNotFound
	}
	k, ok := m.keys[accountID][d.Provider]
	if !ok {
		return store.KeyConfig{}, store.ErrNotFound
	}
	k = m.withDefaultFlagLocked(accountID, k)
	k.FeatureModel = d.Model
	k.IsDefault = feature == store.FeatureCoach
	return k, nil
}

func (m *memStore) ListDefaults(_ context.Context, accountID string) ([]store.FeatureDefault, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]store.FeatureDefault, 0, len(m.defaults[accountID]))
	for _, d := range m.defaults[accountID] {
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Feature < out[j].Feature })
	return out, nil
}

func (m *memStore) PutKey(_ context.Context, k store.KeyConfig) (store.KeyConfig, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.keys[k.AccountID] == nil {
		m.keys[k.AccountID] = map[string]store.KeyConfig{}
	}
	ps := m.keys[k.AccountID]
	k.Enabled = true
	first := len(ps) == 0
	ps[k.Provider] = k
	d, hasCoach := m.defaults[k.AccountID][store.FeatureCoach]
	switch {
	case first:
		// The account's first key becomes its coach default.
		m.setDefaultLocked(k.AccountID, store.FeatureCoach, k, k.DefaultModel)
	case hasCoach && d.Provider == k.Provider:
		// A replaced key that already backs the coach default carries its (possibly new)
		// model over — SyncKeyDefaultModel. An `interview` default pointing at the same
		// key keeps its own chosen brain.
		m.setDefaultLocked(k.AccountID, store.FeatureCoach, k, k.DefaultModel)
	}
	return m.withDefaultFlagLocked(k.AccountID, k), nil
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
	if d, ok := m.defaults[accountID][store.FeatureCoach]; ok && d.Provider == provider {
		m.setDefaultLocked(accountID, store.FeatureCoach, k, model)
	}
	return m.withDefaultFlagLocked(accountID, k), nil
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

func (m *memStore) SetDefault(_ context.Context, accountID, provider, feature, model string) (store.KeyConfig, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	k, ok := m.keys[accountID][provider]
	if !ok {
		return store.KeyConfig{}, store.ErrNotFound
	}
	m.setDefaultLocked(accountID, feature, k, model)
	return m.withDefaultFlagLocked(accountID, k), nil
}

func (m *memStore) DeleteKey(_ context.Context, accountID, provider string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	ps := m.keys[accountID]
	if _, ok := ps[provider]; !ok {
		return store.ErrNotFound
	}
	delete(ps, provider)
	// ON DELETE CASCADE: every default row pointing at the key dies with it.
	for feature, d := range m.defaults[accountID] {
		if d.Provider == provider {
			delete(m.defaults[accountID], feature)
		}
	}
	// Only `coach` is promoted (lowest provider id, for deterministic tests);
	// `interview` is deliberately left unset rather than moved to a model the learner
	// never chose.
	if _, ok := m.defaults[accountID][store.FeatureCoach]; !ok && len(ps) > 0 {
		pick := ""
		for p := range ps {
			if pick == "" || p < pick {
				pick = p
			}
		}
		m.setDefaultLocked(accountID, store.FeatureCoach, ps[pick], ps[pick].DefaultModel)
	}
	return nil
}

func (m *memStore) UsageMonth(_ context.Context, accountID string) (store.Usage, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out store.Usage
	for id, key := range m.threadOf {
		if a, _, _ := strings.Cut(key, "|"); a != accountID {
			continue
		}
		for _, u := range m.usage[id] {
			out.Messages++
			out.InputTokens += int64(u.InputTokens)
			out.OutputTokens += int64(u.OutputTokens)
			if u.EstCostMicros == nil {
				// An un-priced turn makes the total a floor, not a sum — the same thing
				// bool_or(est_cost_micros IS NULL) reports in SQL.
				out.HasUnknownCost = true
				continue
			}
			out.EstCostMicros += *u.EstCostMicros
		}
	}
	return out, nil
}

func (m *memStore) EnsureThread(_ context.Context, accountID, pageContext, pathSlug string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := accountID + "|" + pageContext
	if id, ok := m.threads[key]; ok {
		if m.pathOf[id] == "" {
			m.pathOf[id] = pathSlug // keep an existing path_slug, fill a NULL one (the real upsert)
		}
		return id, nil
	}
	m.nextID++
	id := "thread-" + itoa(m.nextID)
	m.threads[key] = id
	m.threadOf[id] = key
	m.pathOf[id] = pathSlug
	return id, nil
}

// thread returns the (account, thread key) thread's id and path_slug; ok is false when
// the account never chatted under that key.
func (m *memStore) thread(accountID, key string) (id, pathSlug string, ok bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok = m.threads[accountID+"|"+key]
	return id, m.pathOf[id], ok
}

// threadKeys returns every thread key the account has, sorted.
func (m *memStore) threadKeys(accountID string) []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []string
	for k := range m.threads {
		if a, key, _ := strings.Cut(k, "|"); a == accountID {
			out = append(out, key)
		}
	}
	sort.Strings(out)
	return out
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

// RecentMessages mirrors ListRecentMessages: the last limit messages, oldest-first.
func (m *memStore) RecentMessages(_ context.Context, threadID string, limit int) ([]store.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	msgs := m.messages[threadID]
	if limit <= 0 {
		return []store.Message{}, nil
	}
	if len(msgs) > limit {
		msgs = msgs[len(msgs)-limit:]
	}
	return append([]store.Message{}, msgs...), nil
}

func (m *memStore) AppendMessage(_ context.Context, threadID, role, content string, meta store.MessageMeta) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages[threadID] = append(m.messages[threadID], store.Message{Role: role, Content: content})
	m.metas[threadID] = append(m.metas[threadID], meta)
	return nil
}

func (m *memStore) AppendAssistantMessage(_ context.Context, threadID, content string, u store.MessageUsage, meta store.MessageMeta) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages[threadID] = append(m.messages[threadID], store.Message{Role: store.RoleAssistant, Content: content})
	m.metas[threadID] = append(m.metas[threadID], meta)
	m.usage[threadID] = append(m.usage[threadID], u)
	return nil
}

// quotaKey is the fake's coach.message_quota_day primary key: the account and its UTC day.
func quotaKey(accountID string, day time.Time) string {
	return accountID + "|" + day.UTC().Format(time.DateOnly)
}

// TakeDailyMessage mirrors the conditional upsert: insert 1, or increment while below the
// cap; at the cap nothing changes and ok is false.
func (m *memStore) TakeDailyMessage(_ context.Context, accountID string, day time.Time, limit int) (int, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.quotaErr != nil {
		return 0, false, m.quotaErr
	}
	k := quotaKey(accountID, day)
	n, exists := m.quota[k]
	if exists && n >= limit {
		return limit, false, nil
	}
	m.quota[k] = n + 1
	return n + 1, true, nil
}

func (m *memStore) RefundDailyMessage(_ context.Context, accountID string, day time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.quotaErr != nil {
		return m.quotaErr
	}
	if k := quotaKey(accountID, day); m.quota[k] > 0 {
		m.quota[k]--
	}
	return nil
}

func (m *memStore) DailyMessages(_ context.Context, accountID string, day time.Time) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.quotaErr != nil {
		return 0, m.quotaErr
	}
	return m.quota[quotaKey(accountID, day)], nil
}

// dailyCount / setDailyCount read and seed the fake's quota row for (account, UTC day).
func (m *memStore) dailyCount(accountID string, day time.Time) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.quota[quotaKey(accountID, day)]
}

func (m *memStore) setDailyCount(accountID string, day time.Time, n int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.quota[quotaKey(accountID, day)] = n
}

// setQuotaErr makes every daily-quota call fail (nil clears it).
func (m *memStore) setQuotaErr(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.quotaErr = err
}

// metasFor returns each persisted message's meta for (account, context), in order.
func (m *memStore) metasFor(accountID, pageContext string) []store.MessageMeta {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]store.MessageMeta(nil), m.metas[m.threads[accountID+"|"+pageContext]]...)
}

// messageCount is how many coach_message rows the account has across all its threads.
func (m *memStore) messageCount(accountID string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for id, key := range m.threadOf {
		if a, _, _ := strings.Cut(key, "|"); a == accountID {
			n += len(m.messages[id])
		}
	}
	return n
}

// usageFor returns the usage rows recorded against (account, context)'s assistant turns.
func (m *memStore) usageFor(accountID, pageContext string) []store.MessageUsage {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]store.MessageUsage(nil), m.usage[m.threads[accountID+"|"+pageContext]]...)
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

// fakeClock is the L18 clock seam for handler tests: it only moves when a test advances
// it, so token refills and UTC-day boundaries are deterministic.
type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func newFakeClock(t time.Time) *fakeClock { return &fakeClock{t: t} }

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

func (c *fakeClock) Set(t time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = t
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

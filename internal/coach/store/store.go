// Package store is the coach service's persistence layer: pgx/pgxpool over the
// sqlc-generated queries in ./gen and the goose migrations in ./migrations
// (ADR-0005/0007). It exposes small domain types with plain Go scalars so the HTTP
// layer never touches pgtype, and keeps the xlearn_coach role scoped to schema `coach`
// (all SQL is schema-qualified). It stores each provider key ONLY as envelope-encrypted
// material (enc_key + enc_data_key + masked_key) — the raw key never reaches a column —
// and owns the per-page-context chat threads/messages. coach has no outbox/inbox
// (it emits no events and consumes none via NATS).
//
// v1 round 2 (F006): an account can connect one key PER PROVIDER (Anthropic and/or
// OpenAI), with exactly one marked the DEFAULT — the provider whose model the coach
// answers with. The store maintains the "exactly one default" invariant (first key
// becomes default; set-default moves it; delete promotes a survivor).
package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sujaykumarsuman/xlearn/internal/coach/store/gen"
)

// Providers is the set of provider ids coach supports in v1 (ADR-0007). Additional
// providers stay behind the Coach interface and land later.
const (
	ProviderOpenAI    = "openai"
	ProviderAnthropic = "anthropic"

	// Message roles (coach_message.role).
	RoleUser      = "user"
	RoleAssistant = "assistant"
)

// validProviders indexes Providers for O(1) membership checks.
var validProviders = map[string]bool{ProviderOpenAI: true, ProviderAnthropic: true}

// ValidProvider reports whether p is a supported provider id.
func ValidProvider(p string) bool { return validProviders[p] }

// Errors mapped to HTTP status by the handlers.
var (
	// ErrNotFound is returned when the account has no matching key (or the row isn't theirs).
	ErrNotFound = errors.New("coach: not found")
)

// KeyConfig is one provider-key configuration for an account. EncKey / EncDataKey are the
// sealed material used to decrypt in memory for a provider call — they are NEVER
// serialised to a client (only Masked / Provider / DefaultModel / Name / Enabled /
// IsDefault are). An account has at most one KeyConfig per provider, exactly one of which
// is the default (IsDefault).
type KeyConfig struct {
	AccountID    string
	Provider     string
	EncKey       []byte
	EncDataKey   []byte
	Masked       string
	DefaultModel string
	Name         string
	Enabled      bool
	IsDefault    bool
}

// Message is one persisted chat turn (role user|assistant).
type Message struct {
	Role      string
	Content   string
	CreatedAt time.Time
}

// Store is the coach persistence seam. Handlers depend on this interface so they can be
// unit-tested against an in-memory fake.
type Store interface {
	// ListKeys returns all of an account's provider key configs (0..2, incl. sealed
	// material), stable-ordered. Empty (not ErrNotFound) when the account has none.
	ListKeys(ctx context.Context, accountID string) ([]KeyConfig, error)
	// GetKey returns one (account, provider) config, or ErrNotFound.
	GetKey(ctx context.Context, accountID, provider string) (KeyConfig, error)
	// GetDefaultKey returns the account's DEFAULT provider config (the one the coach uses),
	// or ErrNotFound when the account has no keys at all.
	GetDefaultKey(ctx context.Context, accountID string) (KeyConfig, error)
	// PutKey upserts the (account, provider) config with pre-sealed material, re-enabling it.
	// The account's FIRST key becomes the default; replacing a key keeps its default status.
	PutKey(ctx context.Context, k KeyConfig) (KeyConfig, error)
	// UpdateKeyMeta changes a provider's model + name WITHOUT re-sealing the key. ErrNotFound
	// when that provider isn't connected.
	UpdateKeyMeta(ctx context.Context, accountID, provider, model, name string) (KeyConfig, error)
	// SetKeyEnabled flips one provider's enabled flag (Settings toggle / provider-auth
	// failure). ErrNotFound when that provider isn't connected.
	SetKeyEnabled(ctx context.Context, accountID, provider string, enabled bool) error
	// SetDefault makes provider the account's default (clearing the others) and returns the
	// new default config. ErrNotFound when that provider isn't connected.
	SetDefault(ctx context.Context, accountID, provider string) (KeyConfig, error)
	// DeleteKey removes one provider's key, promoting the earliest-created survivor to
	// default when the deleted key was the default. ErrNotFound when it wasn't connected.
	DeleteKey(ctx context.Context, accountID, provider string) error

	// EnsureThread get-or-creates the thread for (account, page context) and returns its
	// id (used before appending a message).
	EnsureThread(ctx context.Context, accountID, pageContext string) (string, error)
	// ThreadHistory returns the messages for (account, page context) oldest-first, or an
	// empty slice when the account has never chatted on that page.
	ThreadHistory(ctx context.Context, accountID, pageContext string) ([]Message, error)
	// AppendMessage appends a message to a thread.
	AppendMessage(ctx context.Context, threadID, role, content string) error

	Ping(ctx context.Context) error
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

// ListKeys returns all of an account's provider key configs.
func (s *PgStore) ListKeys(ctx context.Context, accountID string) ([]KeyConfig, error) {
	aid, err := parseUUID(accountID)
	if err != nil {
		return nil, ErrNotFound
	}
	rows, err := s.q.ListApiKeyConfigs(ctx, aid)
	if err != nil {
		return nil, fmt.Errorf("list api key configs: %w", err)
	}
	out := make([]KeyConfig, 0, len(rows))
	for _, r := range rows {
		out = append(out, toKeyConfig(r))
	}
	return out, nil
}

// GetKey returns one (account, provider) config or ErrNotFound.
func (s *PgStore) GetKey(ctx context.Context, accountID, provider string) (KeyConfig, error) {
	aid, err := parseUUID(accountID)
	if err != nil {
		return KeyConfig{}, ErrNotFound
	}
	row, err := s.q.GetApiKeyConfig(ctx, gen.GetApiKeyConfigParams{AccountID: aid, Provider: provider})
	if errors.Is(err, pgx.ErrNoRows) {
		return KeyConfig{}, ErrNotFound
	}
	if err != nil {
		return KeyConfig{}, fmt.Errorf("get api key config: %w", err)
	}
	return toKeyConfig(row), nil
}

// GetDefaultKey returns the account's default provider config or ErrNotFound.
func (s *PgStore) GetDefaultKey(ctx context.Context, accountID string) (KeyConfig, error) {
	aid, err := parseUUID(accountID)
	if err != nil {
		return KeyConfig{}, ErrNotFound
	}
	row, err := s.q.GetDefaultApiKeyConfig(ctx, aid)
	if errors.Is(err, pgx.ErrNoRows) {
		return KeyConfig{}, ErrNotFound
	}
	if err != nil {
		return KeyConfig{}, fmt.Errorf("get default api key config: %w", err)
	}
	return toKeyConfig(row), nil
}

// PutKey upserts a provider key; the account's first key becomes the default.
func (s *PgStore) PutKey(ctx context.Context, k KeyConfig) (KeyConfig, error) {
	aid, err := parseUUID(k.AccountID)
	if err != nil {
		return KeyConfig{}, fmt.Errorf("parse account id: %w", err)
	}
	// The first key an account connects becomes its default. A replacement keeps whatever
	// default status the row already had (the upsert ORs is_default), so passing false for a
	// non-first key never demotes an existing default.
	n, err := s.q.CountApiKeyConfigs(ctx, aid)
	if err != nil {
		return KeyConfig{}, fmt.Errorf("count api key configs: %w", err)
	}
	row, err := s.q.UpsertApiKeyConfig(ctx, gen.UpsertApiKeyConfigParams{
		AccountID:    aid,
		Provider:     k.Provider,
		EncKey:       k.EncKey,
		EncDataKey:   k.EncDataKey,
		MaskedKey:    k.Masked,
		DefaultModel: k.DefaultModel,
		Name:         k.Name,
		IsDefault:    n == 0,
	})
	if err != nil {
		return KeyConfig{}, fmt.Errorf("upsert api key config: %w", err)
	}
	return toKeyConfig(row), nil
}

// UpdateKeyMeta changes a provider's model + name without re-sealing the key.
func (s *PgStore) UpdateKeyMeta(ctx context.Context, accountID, provider, model, name string) (KeyConfig, error) {
	aid, err := parseUUID(accountID)
	if err != nil {
		return KeyConfig{}, ErrNotFound
	}
	row, err := s.q.UpdateApiKeyMeta(ctx, gen.UpdateApiKeyMetaParams{AccountID: aid, Provider: provider, DefaultModel: model, Name: name})
	if errors.Is(err, pgx.ErrNoRows) {
		return KeyConfig{}, ErrNotFound
	}
	if err != nil {
		return KeyConfig{}, fmt.Errorf("update api key meta: %w", err)
	}
	return toKeyConfig(row), nil
}

// SetKeyEnabled flips a provider's enabled flag (ErrNotFound if not connected).
func (s *PgStore) SetKeyEnabled(ctx context.Context, accountID, provider string, enabled bool) error {
	aid, err := parseUUID(accountID)
	if err != nil {
		return ErrNotFound
	}
	n, err := s.q.SetApiKeyEnabled(ctx, gen.SetApiKeyEnabledParams{AccountID: aid, Provider: provider, Enabled: enabled})
	if err != nil {
		return fmt.Errorf("set api key enabled: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// SetDefault makes provider the account's default and returns the new default config.
func (s *PgStore) SetDefault(ctx context.Context, accountID, provider string) (KeyConfig, error) {
	aid, err := parseUUID(accountID)
	if err != nil {
		return KeyConfig{}, ErrNotFound
	}
	// Verify the target provider is connected first — a single-statement default swap would
	// otherwise blank the default when the provider has no row.
	if _, err := s.q.GetApiKeyConfig(ctx, gen.GetApiKeyConfigParams{AccountID: aid, Provider: provider}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return KeyConfig{}, ErrNotFound
		}
		return KeyConfig{}, fmt.Errorf("get api key config: %w", err)
	}
	if _, err := s.q.SetDefaultProvider(ctx, gen.SetDefaultProviderParams{AccountID: aid, Provider: provider}); err != nil {
		return KeyConfig{}, fmt.Errorf("set default provider: %w", err)
	}
	row, err := s.q.GetApiKeyConfig(ctx, gen.GetApiKeyConfigParams{AccountID: aid, Provider: provider})
	if err != nil {
		return KeyConfig{}, fmt.Errorf("get api key config: %w", err)
	}
	return toKeyConfig(row), nil
}

// DeleteKey removes a provider's key, promoting a survivor to default if needed.
func (s *PgStore) DeleteKey(ctx context.Context, accountID, provider string) error {
	aid, err := parseUUID(accountID)
	if err != nil {
		return ErrNotFound
	}
	n, err := s.q.DeleteApiKeyConfig(ctx, gen.DeleteApiKeyConfigParams{AccountID: aid, Provider: provider})
	if err != nil {
		return fmt.Errorf("delete api key config: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	// If we removed the default, promote the earliest-created survivor so the account keeps
	// exactly one default (a no-op — ErrNoRows — when a default remains or no keys are left).
	if _, err := s.q.PromoteEarliestDefault(ctx, aid); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("promote default: %w", err)
	}
	return nil
}

// EnsureThread get-or-creates the (account, page context) thread and returns its id.
func (s *PgStore) EnsureThread(ctx context.Context, accountID, pageContext string) (string, error) {
	aid, err := parseUUID(accountID)
	if err != nil {
		return "", fmt.Errorf("parse account id: %w", err)
	}
	id, err := s.q.UpsertThread(ctx, gen.UpsertThreadParams{AccountID: aid, PageContext: pageContext})
	if err != nil {
		return "", fmt.Errorf("upsert thread: %w", err)
	}
	return uuidString(id), nil
}

// ThreadHistory returns the (account, page context) messages oldest-first (empty if the
// thread doesn't exist yet).
func (s *PgStore) ThreadHistory(ctx context.Context, accountID, pageContext string) ([]Message, error) {
	aid, err := parseUUID(accountID)
	if err != nil {
		return nil, ErrNotFound
	}
	tid, err := s.q.GetThread(ctx, gen.GetThreadParams{AccountID: aid, PageContext: pageContext})
	if errors.Is(err, pgx.ErrNoRows) {
		return []Message{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get thread: %w", err)
	}
	rows, err := s.q.ListMessages(ctx, tid)
	if err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}
	out := make([]Message, 0, len(rows))
	for _, r := range rows {
		out = append(out, Message{Role: r.Role, Content: r.Content, CreatedAt: r.CreatedAt.Time})
	}
	return out, nil
}

// AppendMessage appends a message to a thread.
func (s *PgStore) AppendMessage(ctx context.Context, threadID, role, content string) error {
	tid, err := parseUUID(threadID)
	if err != nil {
		return fmt.Errorf("parse thread id: %w", err)
	}
	if _, err := s.q.InsertMessage(ctx, gen.InsertMessageParams{ThreadID: tid, Role: role, Content: content}); err != nil {
		return fmt.Errorf("insert message: %w", err)
	}
	return nil
}

// toKeyConfig maps the generated row to the plain-scalar domain type.
func toKeyConfig(r gen.CoachApiKeyConfig) KeyConfig {
	return KeyConfig{
		AccountID:    uuidString(r.AccountID),
		Provider:     r.Provider,
		EncKey:       r.EncKey,
		EncDataKey:   r.EncDataKey,
		Masked:       r.MaskedKey,
		DefaultModel: r.DefaultModel,
		Name:         r.Name,
		Enabled:      r.Enabled,
		IsDefault:    r.IsDefault,
	}
}

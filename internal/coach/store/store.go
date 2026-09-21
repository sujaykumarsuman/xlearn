// Package store is the coach service's persistence layer: pgx/pgxpool over the
// sqlc-generated queries in ./gen and the goose migrations in ./migrations
// (ADR-0005/0007). It exposes small domain types with plain Go scalars so the HTTP
// layer never touches pgtype, and keeps the xlearn_coach role scoped to schema `coach`
// (all SQL is schema-qualified). It stores the provider key ONLY as envelope-encrypted
// material (enc_key + enc_data_key + masked_key) — the raw key never reaches a column —
// and owns the per-page-context chat threads/messages. coach has no outbox/inbox
// (it emits no events and consumes none via NATS).
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
	// ErrNotFound is returned when the account has no key (or the row isn't theirs).
	ErrNotFound = errors.New("coach: not found")
)

// KeyConfig is an account's provider-key configuration. EncKey / EncDataKey are the
// sealed material used to decrypt in memory for a provider call — they are NEVER
// serialised to a client (only Masked / Provider / DefaultModel / Enabled are).
type KeyConfig struct {
	AccountID    string
	Provider     string
	EncKey       []byte
	EncDataKey   []byte
	Masked       string
	DefaultModel string
	Enabled      bool
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
	// PutKey stores or replaces the account's key config (single-key per account) with
	// the pre-sealed material, re-enabling it, and returns the stored config.
	PutKey(ctx context.Context, k KeyConfig) (KeyConfig, error)
	// GetKey returns the account's key config (incl. sealed material) or ErrNotFound.
	GetKey(ctx context.Context, accountID string) (KeyConfig, error)
	// DeleteKey removes the account's key. ErrNotFound if there was none.
	DeleteKey(ctx context.Context, accountID string) error
	// SetKeyEnabled flips enabled (Settings toggle / provider-auth failure). ErrNotFound
	// if the account has no key.
	SetKeyEnabled(ctx context.Context, accountID string, enabled bool) error

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

// PutKey upserts the account's key config with pre-sealed material.
func (s *PgStore) PutKey(ctx context.Context, k KeyConfig) (KeyConfig, error) {
	aid, err := parseUUID(k.AccountID)
	if err != nil {
		return KeyConfig{}, fmt.Errorf("parse account id: %w", err)
	}
	row, err := s.q.UpsertApiKeyConfig(ctx, gen.UpsertApiKeyConfigParams{
		AccountID:    aid,
		Provider:     k.Provider,
		EncKey:       k.EncKey,
		EncDataKey:   k.EncDataKey,
		MaskedKey:    k.Masked,
		DefaultModel: k.DefaultModel,
	})
	if err != nil {
		return KeyConfig{}, fmt.Errorf("upsert api key config: %w", err)
	}
	return toKeyConfig(row), nil
}

// GetKey returns the account's key config or ErrNotFound.
func (s *PgStore) GetKey(ctx context.Context, accountID string) (KeyConfig, error) {
	aid, err := parseUUID(accountID)
	if err != nil {
		return KeyConfig{}, ErrNotFound
	}
	row, err := s.q.GetApiKeyConfig(ctx, aid)
	if errors.Is(err, pgx.ErrNoRows) {
		return KeyConfig{}, ErrNotFound
	}
	if err != nil {
		return KeyConfig{}, fmt.Errorf("get api key config: %w", err)
	}
	return toKeyConfig(row), nil
}

// DeleteKey removes the account's key (ErrNotFound if none).
func (s *PgStore) DeleteKey(ctx context.Context, accountID string) error {
	aid, err := parseUUID(accountID)
	if err != nil {
		return ErrNotFound
	}
	n, err := s.q.DeleteApiKeyConfig(ctx, aid)
	if err != nil {
		return fmt.Errorf("delete api key config: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// SetKeyEnabled flips enabled (ErrNotFound if none).
func (s *PgStore) SetKeyEnabled(ctx context.Context, accountID string, enabled bool) error {
	aid, err := parseUUID(accountID)
	if err != nil {
		return ErrNotFound
	}
	n, err := s.q.SetApiKeyEnabled(ctx, gen.SetApiKeyEnabledParams{AccountID: aid, Enabled: enabled})
	if err != nil {
		return fmt.Errorf("set api key enabled: %w", err)
	}
	if n == 0 {
		return ErrNotFound
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
		Enabled:      r.Enabled,
	}
}

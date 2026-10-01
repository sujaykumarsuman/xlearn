// Package store is the coach service's persistence layer: pgx/pgxpool over the
// sqlc-generated queries in ./gen and the goose migrations in ./migrations
// (ADR-0005/0007). It exposes small domain types with plain Go scalars so the HTTP
// layer never touches pgtype, and keeps the xlearn_coach role scoped to schema `coach`
// (all SQL is schema-qualified). It stores each provider key ONLY as envelope-encrypted
// material — the raw key never reaches a column — and owns the per-page-context chat
// threads/messages. coach has no outbox/inbox (it emits no events and consumes none via
// NATS).
//
// v1 round 2 (F006): an account can connect one key PER PROVIDER (Anthropic and/or
// OpenAI), with one of them the default — the provider whose model the coach answers with.
//
// v2 M1a (m1-02): the default moved to coach.key_default(account_id, feature), dual-written
// beside api_key_config.is_default.
//
// v2 M1b (m1-10): key_default is the ONLY default source. coach neither reads nor writes
// is_default (m1-08 drops it in M1c), every query names its columns explicitly, and the
// default is PER FEATURE:
//
//   - FeatureCoach is implicit: the account's first key becomes it, and deleting it
//     promotes the earliest surviving key, so an account with any key always has one.
//   - FeatureInterview is explicit only. It is never auto-assigned and never
//     auto-promoted (t6 §11); "not set" is a real state the UI renders.
//
// v2 M1b key crypto (m1-10 task 3): each key carries TWO sealed pairs. The legacy pair
// (EncKey, EncDataKey) is unbound and still readable by v1.6.0, which keeps the rollback
// floor at 1.6.0; the AD pair (EncKeyAD, EncDataKeyAD) is bound to (account, provider) by
// AEAD associated data under keyring entry KEKID. ADSrcDigest = sha256 of the legacy
// EncDataKey the AD pair was sealed beside, which is how a pair left STALE by a v1.6.0
// key replace during a rollback is detected (see rewrap.go). Both pairs are written in a
// single statement, so no v1.7.0+ writer can make them diverge.
//
// v2 M1b (m1-03): a thread's page_context is the normalized key the handlers build
// (course.NormalizeCoachContext: `<course>:<ctx>` for a course-scoped context), and the
// thread and each of its messages carry the thread's course in path_slug (NULL for an
// account-wide context). Migration 00005 moved the v1 rows onto that shape.
package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sujaykumarsuman/xlearn/internal/coach/store/gen"
)

// Providers is the set of provider ids coach supports (ADR-0007).
const (
	ProviderOpenAI    = "openai"
	ProviderAnthropic = "anthropic"

	// Message roles (coach_message.role).
	RoleUser      = "user"
	RoleAssistant = "assistant"
)

// Features are the per-feature default keys (coach.key_default.feature; the CHECK in
// migration 00004 holds exactly these two).
const (
	// FeatureCoach is the key the chat coach answers with. Implicitly maintained.
	FeatureCoach = "coach"
	// FeatureInterview is the brain the text interviewer uses (m6a-02). Explicit only.
	FeatureInterview = "interview"
)

// validProviders indexes Providers for O(1) membership checks.
var validProviders = map[string]bool{ProviderOpenAI: true, ProviderAnthropic: true}

// ValidProvider reports whether p is a supported provider id.
//
// This is a STATIC MIRROR of the `CHECK (provider IN ('openai','anthropic'))` in
// migration 00001, kept here because store cannot import coach (coach imports store — a
// cycle) and so cannot see the service's provider registry. The registry is the one the
// HTTP layer validates against; this mirror exists for the store's own guard rails and
// for tests. A test in package coach asserts the registry, this mirror and the CHECK all
// list the same providers, so widening one without the others fails CI.
func ValidProvider(p string) bool { return validProviders[p] }

// ValidFeature reports whether f is a key_default feature (mirrors 00004's CHECK).
func ValidFeature(f string) bool { return f == FeatureCoach || f == FeatureInterview }

// Errors mapped to HTTP status by the handlers.
var (
	// ErrNotFound is returned when the account has no matching key (or the row isn't theirs).
	ErrNotFound = errors.New("coach: not found")
)

// KeyConfig is one provider-key configuration for an account. The sealed material
// (EncKey/EncDataKey and EncKeyAD/EncDataKeyAD) is used to decrypt in memory for a
// provider call and is NEVER serialised to a client — only Masked / Provider /
// DefaultModel / Name / Enabled / IsDefault are. An account has at most one KeyConfig per
// provider.
type KeyConfig struct {
	ID        string
	AccountID string
	Provider  string

	// The legacy, UNBOUND sealed pair (v1 format; v1.6.0 can open it). Dual-written.
	EncKey     []byte
	EncDataKey []byte

	// The AD-BOUND pair and the keyring entry that wrapped it. Empty on a row the re-wrap
	// job has not reached yet.
	EncKeyAD     []byte
	EncDataKeyAD []byte
	KEKID        string
	// ADSrcDigest is sha256(EncDataKey) as it stood when the AD pair was sealed. A
	// mismatch means the legacy pair was rewritten underneath the AD pair, which makes the
	// AD pair STALE (see ADPairCurrent).
	ADSrcDigest []byte

	Masked       string
	DefaultModel string
	Name         string
	Enabled      bool

	// IsDefault reports whether this key backs the account's FeatureCoach default. It is
	// derived from key_default, never from the dead api_key_config.is_default column.
	IsDefault bool

	// FeatureModel is the model the requested feature's default row chose. Set only by
	// GetDefaultKey; it may differ from DefaultModel (an interview brain, say) and is ""
	// when the row stored none.
	FeatureModel string
}

// ADPairCurrent reports whether the key's AD-bound pair describes the legacy pair that is
// stored with it right now, i.e. whether it is safe to answer a provider call from.
//
// It is current when the pair exists AND either
//   - the legacy pair is gone (from l-01 on, when coach stops writing it), or
//   - the digest still matches the legacy wrapped data key beside it.
//
// It is NOT current when the legacy pair was rewritten underneath it — which is exactly
// what a v1.6.0 image does when it replaces a key during a rollback, since its upsert
// touches only the legacy columns. Reading a stale AD pair would hand the provider the
// PREVIOUS key: if the owner rotated because the old key was revoked, the 401 would
// disable the key they just pasted; if the old key still worked, it would be billed
// silently. Callers fall back to the legacy pair in that case and let the re-wrap job
// repair the row.
func (k KeyConfig) ADPairCurrent() bool {
	if len(k.EncKeyAD) == 0 || len(k.EncDataKeyAD) == 0 {
		return false
	}
	if len(k.EncDataKey) == 0 {
		return true // l-01 onwards: no legacy pair left to be stale against
	}
	return digestMatches(k.ADSrcDigest, k.EncDataKey)
}

// FeatureDefault is one row of coach.key_default resolved for display: which provider's
// key a feature uses and the model it chose. A feature with no default is simply absent
// from ListDefaults.
type FeatureDefault struct {
	Feature   string
	Provider  string
	Model     string
	KeyID     string
	UpdatedAt time.Time
}

// Usage is an account's month-to-date spend on its own keys (UTC calendar month).
// HasUnknownCost marks that at least one answered turn carried no price — a custom model
// id, or a stream that reported no usage — so EstCostMicros is a FLOOR, not a total. It
// is display-only and always an estimate.
type Usage struct {
	Messages       int64
	InputTokens    int64
	OutputTokens   int64
	EstCostMicros  int64
	HasUnknownCost bool
}

// MessageUsage is what one provider turn cost, stored on the assistant message it
// produced. EstCostMicros is nil when no price is known (a custom model id, or missing
// usage), which is what makes Usage.HasUnknownCost true.
type MessageUsage struct {
	Provider      string
	Model         string
	StopReason    string
	InputTokens   int
	OutputTokens  int
	EstCostMicros *int64
	// HasTokens reports whether the stream actually reported token counts (so zero
	// tokens is distinguishable from "the provider told us nothing").
	HasTokens bool
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
	// GetDefaultKey returns the key backing one FEATURE's default, with that feature's
	// chosen model in FeatureModel. ErrNotFound when the feature has no default — for
	// FeatureCoach that means the account has no keys at all; for FeatureInterview it
	// means none was ever chosen.
	GetDefaultKey(ctx context.Context, accountID, feature string) (KeyConfig, error)
	// ListDefaults returns every feature default the account has set, feature-ordered.
	ListDefaults(ctx context.Context, accountID string) ([]FeatureDefault, error)
	// PutKey upserts the (account, provider) config with BOTH pre-sealed pairs, re-enabling
	// it. The account's FIRST key becomes its FeatureCoach default.
	PutKey(ctx context.Context, k KeyConfig) (KeyConfig, error)
	// UpdateKeyMeta changes a provider's model + name WITHOUT re-sealing the key. ErrNotFound
	// when that provider isn't connected.
	UpdateKeyMeta(ctx context.Context, accountID, provider, model, name string) (KeyConfig, error)
	// SetKeyEnabled flips one provider's enabled flag (Settings toggle / provider-AUTH
	// failure — never a quota or rate limit). ErrNotFound when that provider isn't connected.
	SetKeyEnabled(ctx context.Context, accountID, provider string, enabled bool) error
	// SetDefault points one feature's default at a provider's key, with the model the
	// feature should use (empty takes the key's own default_model). ErrNotFound when that
	// provider isn't connected.
	SetDefault(ctx context.Context, accountID, provider, feature, model string) (KeyConfig, error)
	// DeleteKey removes one provider's key. Its key_default rows go with it (cascade); the
	// FeatureCoach default is then re-pointed at the earliest-created survivor.
	// FeatureInterview is left unset. ErrNotFound when it wasn't connected.
	DeleteKey(ctx context.Context, accountID, provider string) error
	// UsageMonth returns the account's month-to-date usage across its keys.
	UsageMonth(ctx context.Context, accountID string) (Usage, error)

	// EnsureThread get-or-creates the thread for (account, page context) and returns its
	// id (used before appending a message). pageContext is the normalized thread key
	// (course.NormalizeCoachContext); pathSlug is the thread's course, "" for NULL (an
	// account-wide context). An existing thread keeps its path_slug; a NULL one is filled.
	EnsureThread(ctx context.Context, accountID, pageContext, pathSlug string) (string, error)
	// ThreadHistory returns the messages for (account, page context) oldest-first, or an
	// empty slice when the account has never chatted on that page.
	ThreadHistory(ctx context.Context, accountID, pageContext string) ([]Message, error)
	// AppendMessage appends a message to a thread, with the thread's path_slug.
	AppendMessage(ctx context.Context, threadID, role, content string) error
	// AppendAssistantMessage appends the coach's reply together with what the provider
	// turn cost. A nil-cost usage stores NULL, which the month summary reports as unknown.
	AppendAssistantMessage(ctx context.Context, threadID, content string, u MessageUsage) error

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
	def, err := coachDefaultKeyID(ctx, s.q, aid)
	if err != nil {
		return nil, err
	}
	out := make([]KeyConfig, 0, len(rows))
	for _, r := range rows {
		out = append(out, keyConfigFrom(keyRow{
			ID: r.ID, AccountID: r.AccountID, Provider: r.Provider,
			EncKey: r.EncKey, EncDataKey: r.EncDataKey,
			EncKeyAd: r.EncKeyAd, EncDataKeyAd: r.EncDataKeyAd, KekID: r.KekID, AdSrcDigest: r.AdSrcDigest,
			MaskedKey: r.MaskedKey, DefaultModel: r.DefaultModel, Name: r.Name, Enabled: r.Enabled,
		}, def))
	}
	return out, nil
}

// GetKey returns one (account, provider) config or ErrNotFound.
func (s *PgStore) GetKey(ctx context.Context, accountID, provider string) (KeyConfig, error) {
	aid, err := parseUUID(accountID)
	if err != nil {
		return KeyConfig{}, ErrNotFound
	}
	return getKey(ctx, s.q, aid, provider)
}

// getKey reads one (account, provider) config with its effective coach-default flag, on
// the pool or inside a transaction.
func getKey(ctx context.Context, q *gen.Queries, aid pgtype.UUID, provider string) (KeyConfig, error) {
	r, err := q.GetApiKeyConfig(ctx, gen.GetApiKeyConfigParams{AccountID: aid, Provider: provider})
	if errors.Is(err, pgx.ErrNoRows) {
		return KeyConfig{}, ErrNotFound
	}
	if err != nil {
		return KeyConfig{}, fmt.Errorf("get api key config: %w", err)
	}
	def, err := coachDefaultKeyID(ctx, q, aid)
	if err != nil {
		return KeyConfig{}, err
	}
	return keyConfigFrom(keyRow{
		ID: r.ID, AccountID: r.AccountID, Provider: r.Provider,
		EncKey: r.EncKey, EncDataKey: r.EncDataKey,
		EncKeyAd: r.EncKeyAd, EncDataKeyAd: r.EncDataKeyAd, KekID: r.KekID, AdSrcDigest: r.AdSrcDigest,
		MaskedKey: r.MaskedKey, DefaultModel: r.DefaultModel, Name: r.Name, Enabled: r.Enabled,
	}, def), nil
}

// GetDefaultKey returns the key backing one feature's default, with that feature's model
// in FeatureModel. key_default is the only source consulted (m1-10): a default that only
// ever existed as an is_default row was migrated into key_default by 00004's backfill.
func (s *PgStore) GetDefaultKey(ctx context.Context, accountID, feature string) (KeyConfig, error) {
	aid, err := parseUUID(accountID)
	if err != nil {
		return KeyConfig{}, ErrNotFound
	}
	if !ValidFeature(feature) {
		return KeyConfig{}, ErrNotFound
	}
	r, err := s.q.GetKeyDefault(ctx, gen.GetKeyDefaultParams{AccountID: aid, Feature: feature})
	if errors.Is(err, pgx.ErrNoRows) {
		return KeyConfig{}, ErrNotFound
	}
	if err != nil {
		return KeyConfig{}, fmt.Errorf("get key default: %w", err)
	}
	kc := keyConfigFrom(keyRow{
		ID: r.ID, AccountID: r.AccountID, Provider: r.Provider,
		EncKey: r.EncKey, EncDataKey: r.EncDataKey,
		EncKeyAd: r.EncKeyAd, EncDataKeyAd: r.EncDataKeyAd, KekID: r.KekID, AdSrcDigest: r.AdSrcDigest,
		MaskedKey: r.MaskedKey, DefaultModel: r.DefaultModel, Name: r.Name, Enabled: r.Enabled,
	}, r.ID)
	kc.FeatureModel = r.FeatureModel
	// IsDefault means "backs the COACH default"; only say so for that feature.
	kc.IsDefault = feature == FeatureCoach
	return kc, nil
}

// ListDefaults returns the account's per-feature defaults, feature-ordered.
func (s *PgStore) ListDefaults(ctx context.Context, accountID string) ([]FeatureDefault, error) {
	aid, err := parseUUID(accountID)
	if err != nil {
		return nil, ErrNotFound
	}
	rows, err := s.q.ListKeyDefaults(ctx, aid)
	if err != nil {
		return nil, fmt.Errorf("list key defaults: %w", err)
	}
	out := make([]FeatureDefault, 0, len(rows))
	for _, r := range rows {
		out = append(out, FeatureDefault{
			Feature:   r.Feature,
			Provider:  r.Provider,
			Model:     r.Model,
			KeyID:     uuidString(r.KeyID),
			UpdatedAt: r.UpdatedAt.Time,
		})
	}
	return out, nil
}

// PutKey upserts a provider key with both sealed pairs; the account's first key becomes
// its coach default. One transaction: the key row and its key_default row commit together,
// so an account can never end up with a key and no default (or the reverse).
func (s *PgStore) PutKey(ctx context.Context, k KeyConfig) (KeyConfig, error) {
	aid, err := parseUUID(k.AccountID)
	if err != nil {
		return KeyConfig{}, fmt.Errorf("parse account id: %w", err)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return KeyConfig{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := s.q.WithTx(tx)

	// The first key an account connects becomes its coach default. A replacement leaves
	// whatever default rows exist pointing where they already point.
	n, err := qtx.CountApiKeyConfigs(ctx, aid)
	if err != nil {
		return KeyConfig{}, fmt.Errorf("count api key configs: %w", err)
	}
	first := n == 0
	r, err := qtx.UpsertApiKeyConfig(ctx, gen.UpsertApiKeyConfigParams{
		AccountID:    aid,
		Provider:     k.Provider,
		EncKey:       k.EncKey,
		EncDataKey:   k.EncDataKey,
		MaskedKey:    k.Masked,
		DefaultModel: k.DefaultModel,
		Name:         k.Name,
		EncKeyAd:     k.EncKeyAD,
		EncDataKeyAd: k.EncDataKeyAD,
		KekID:        pgtype.Text{String: k.KEKID, Valid: k.KEKID != ""},
		AdSrcDigest:  k.ADSrcDigest,
	})
	if err != nil {
		return KeyConfig{}, fmt.Errorf("upsert api key config: %w", err)
	}
	if first {
		err = qtx.UpsertKeyDefault(ctx, gen.UpsertKeyDefaultParams{
			AccountID: aid, Feature: FeatureCoach, KeyID: r.ID, Model: r.DefaultModel,
		})
	} else {
		// A replaced key that already backs the coach default carries its (possibly new)
		// model over. An interview default keeps its own chosen brain.
		err = qtx.SyncKeyDefaultModel(ctx, gen.SyncKeyDefaultModelParams{KeyID: r.ID, Model: r.DefaultModel})
	}
	if err != nil {
		return KeyConfig{}, fmt.Errorf("write key default: %w", err)
	}
	def, err := coachDefaultKeyID(ctx, qtx, aid)
	if err != nil {
		return KeyConfig{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return KeyConfig{}, fmt.Errorf("commit tx: %w", err)
	}
	return keyConfigFrom(keyRow{
		ID: r.ID, AccountID: r.AccountID, Provider: r.Provider,
		EncKey: r.EncKey, EncDataKey: r.EncDataKey,
		EncKeyAd: r.EncKeyAd, EncDataKeyAd: r.EncDataKeyAd, KekID: r.KekID, AdSrcDigest: r.AdSrcDigest,
		MaskedKey: r.MaskedKey, DefaultModel: r.DefaultModel, Name: r.Name, Enabled: r.Enabled,
	}, def), nil
}

// UpdateKeyMeta changes a provider's model + name without re-sealing the key (and the
// coach default's model with it when this key backs it).
func (s *PgStore) UpdateKeyMeta(ctx context.Context, accountID, provider, model, name string) (KeyConfig, error) {
	aid, err := parseUUID(accountID)
	if err != nil {
		return KeyConfig{}, ErrNotFound
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return KeyConfig{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := s.q.WithTx(tx)

	r, err := qtx.UpdateApiKeyMeta(ctx, gen.UpdateApiKeyMetaParams{AccountID: aid, Provider: provider, DefaultModel: model, Name: name})
	if errors.Is(err, pgx.ErrNoRows) {
		return KeyConfig{}, ErrNotFound
	}
	if err != nil {
		return KeyConfig{}, fmt.Errorf("update api key meta: %w", err)
	}
	if err := qtx.SyncKeyDefaultModel(ctx, gen.SyncKeyDefaultModelParams{KeyID: r.ID, Model: r.DefaultModel}); err != nil {
		return KeyConfig{}, fmt.Errorf("sync key default model: %w", err)
	}
	def, err := coachDefaultKeyID(ctx, qtx, aid)
	if err != nil {
		return KeyConfig{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return KeyConfig{}, fmt.Errorf("commit tx: %w", err)
	}
	return keyConfigFrom(keyRow{
		ID: r.ID, AccountID: r.AccountID, Provider: r.Provider,
		EncKey: r.EncKey, EncDataKey: r.EncDataKey,
		EncKeyAd: r.EncKeyAd, EncDataKeyAd: r.EncDataKeyAd, KekID: r.KekID, AdSrcDigest: r.AdSrcDigest,
		MaskedKey: r.MaskedKey, DefaultModel: r.DefaultModel, Name: r.Name, Enabled: r.Enabled,
	}, def), nil
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

// SetDefault points one feature's default at a provider's key. model is the model the
// feature should use; empty takes the key's own default_model (what the coach feature
// always wants). One transaction: the target is verified connected first, so an unknown
// provider can never blank a default.
func (s *PgStore) SetDefault(ctx context.Context, accountID, provider, feature, model string) (KeyConfig, error) {
	aid, err := parseUUID(accountID)
	if err != nil {
		return KeyConfig{}, ErrNotFound
	}
	if !ValidFeature(feature) {
		return KeyConfig{}, ErrNotFound
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return KeyConfig{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := s.q.WithTx(tx)

	target, err := qtx.GetApiKeyConfig(ctx, gen.GetApiKeyConfigParams{AccountID: aid, Provider: provider})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return KeyConfig{}, ErrNotFound
		}
		return KeyConfig{}, fmt.Errorf("get api key config: %w", err)
	}
	// A blank model still falls back to the key's own, but the HANDLER resolves and
	// validates the effective model before calling here, so this is a safety net for a
	// direct store caller rather than the path a request takes. It is deliberately NOT
	// the place to validate: the store cannot import coach (a cycle), so it has no view
	// of the catalog or of what a feature requires.
	if model == "" {
		model = target.DefaultModel
	}
	if err := qtx.UpsertKeyDefault(ctx, gen.UpsertKeyDefaultParams{
		AccountID: aid, Feature: feature, KeyID: target.ID, Model: model,
	}); err != nil {
		return KeyConfig{}, fmt.Errorf("upsert key default: %w", err)
	}
	out, err := getKey(ctx, qtx, aid, provider)
	if err != nil {
		return KeyConfig{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return KeyConfig{}, fmt.Errorf("commit tx: %w", err)
	}
	return out, nil
}

// DeleteKey removes a provider's key. One transaction: the delete (whose ON DELETE
// CASCADE takes every key_default row pointing at the key) and the coach promotion commit
// together, so the account never observes a key with no coach default.
//
// Only FeatureCoach is promoted. FeatureInterview is deliberately left unset if its key
// was the one deleted: an interview brain is an explicit choice, and silently moving it to
// a model the learner never picked (possibly one without the interview_brain capability)
// would be worse than showing "Not set".
func (s *PgStore) DeleteKey(ctx context.Context, accountID, provider string) error {
	aid, err := parseUUID(accountID)
	if err != nil {
		return ErrNotFound
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := s.q.WithTx(tx)

	n, err := qtx.DeleteApiKeyConfig(ctx, gen.DeleteApiKeyConfigParams{AccountID: aid, Provider: provider})
	if err != nil {
		return fmt.Errorf("delete api key config: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	if err := qtx.PromoteEarliestKeyDefault(ctx, gen.PromoteEarliestKeyDefaultParams{
		AccountID: aid, Feature: FeatureCoach,
	}); err != nil {
		return fmt.Errorf("promote key default: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

// UsageMonth returns the account's month-to-date usage on its own keys.
func (s *PgStore) UsageMonth(ctx context.Context, accountID string) (Usage, error) {
	aid, err := parseUUID(accountID)
	if err != nil {
		return Usage{}, ErrNotFound
	}
	r, err := s.q.UsageMonthForAccount(ctx, aid)
	if err != nil {
		return Usage{}, fmt.Errorf("usage month: %w", err)
	}
	return Usage{
		Messages:       r.Messages,
		InputTokens:    r.InputTokens,
		OutputTokens:   r.OutputTokens,
		EstCostMicros:  r.EstCostMicros,
		HasUnknownCost: r.HasUnknownCost,
	}, nil
}

// EnsureThread get-or-creates the (account, page context) thread, labelled with pathSlug
// ("" = NULL), and returns its id.
func (s *PgStore) EnsureThread(ctx context.Context, accountID, pageContext, pathSlug string) (string, error) {
	aid, err := parseUUID(accountID)
	if err != nil {
		return "", fmt.Errorf("parse account id: %w", err)
	}
	id, err := s.q.UpsertThread(ctx, gen.UpsertThreadParams{
		AccountID:   aid,
		PageContext: pageContext,
		PathSlug:    pgtype.Text{String: pathSlug, Valid: pathSlug != ""},
	})
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

// AppendMessage appends a message to a thread; the message takes the thread's path_slug.
// ErrNotFound when the thread doesn't exist.
func (s *PgStore) AppendMessage(ctx context.Context, threadID, role, content string) error {
	tid, err := parseUUID(threadID)
	if err != nil {
		return fmt.Errorf("parse thread id: %w", err)
	}
	_, err = s.q.InsertMessage(ctx, gen.InsertMessageParams{ThreadID: tid, Role: role, Content: content})
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("insert message: %w", err)
	}
	return nil
}

// AppendAssistantMessage appends the coach's reply with the usage of the turn that
// produced it. Unknown fields store NULL.
func (s *PgStore) AppendAssistantMessage(ctx context.Context, threadID, content string, u MessageUsage) error {
	tid, err := parseUUID(threadID)
	if err != nil {
		return fmt.Errorf("parse thread id: %w", err)
	}
	arg := gen.InsertAssistantMessageParams{
		ThreadID:   tid,
		Role:       RoleAssistant,
		Content:    content,
		Provider:   pgtype.Text{String: u.Provider, Valid: u.Provider != ""},
		Model:      pgtype.Text{String: u.Model, Valid: u.Model != ""},
		StopReason: pgtype.Text{String: u.StopReason, Valid: u.StopReason != ""},
	}
	if u.HasTokens {
		arg.InputTokens = pgtype.Int4{Int32: int32(u.InputTokens), Valid: true}
		arg.OutputTokens = pgtype.Int4{Int32: int32(u.OutputTokens), Valid: true}
	}
	if u.EstCostMicros != nil {
		arg.EstCostMicros = pgtype.Int8{Int64: *u.EstCostMicros, Valid: true}
	}
	_, err = s.q.InsertAssistantMessage(ctx, arg)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("insert assistant message: %w", err)
	}
	return nil
}

// --- row mapping ---

// keyRow is the common shape of every api_key_config row sqlc generates (each query gets
// its own named struct because the columns are listed explicitly). Mapping through one
// intermediate keeps a single definition of how a row becomes a KeyConfig, so a new query
// cannot quietly map a column differently.
type keyRow struct {
	ID           pgtype.UUID
	AccountID    pgtype.UUID
	Provider     string
	EncKey       []byte
	EncDataKey   []byte
	EncKeyAd     []byte
	EncDataKeyAd []byte
	KekID        pgtype.Text
	AdSrcDigest  []byte
	MaskedKey    string
	DefaultModel string
	Name         string
	Enabled      bool
}

// keyConfigFrom maps a row to the plain-scalar domain type. IsDefault compares the row to
// the account's effective COACH default id, which comes from key_default alone — never
// from the dead is_default column.
func keyConfigFrom(r keyRow, coachDefaultID pgtype.UUID) KeyConfig {
	return KeyConfig{
		ID:           uuidString(r.ID),
		AccountID:    uuidString(r.AccountID),
		Provider:     r.Provider,
		EncKey:       r.EncKey,
		EncDataKey:   r.EncDataKey,
		EncKeyAD:     r.EncKeyAd,
		EncDataKeyAD: r.EncDataKeyAd,
		KEKID:        r.KekID.String,
		ADSrcDigest:  r.AdSrcDigest,
		Masked:       r.MaskedKey,
		DefaultModel: r.DefaultModel,
		Name:         r.Name,
		Enabled:      r.Enabled,
		IsDefault:    coachDefaultID.Valid && r.ID.Valid && coachDefaultID.Bytes == r.ID.Bytes,
	}
}

// coachDefaultKeyID returns the id of the key backing the account's coach default, or an
// invalid UUID when it has none.
func coachDefaultKeyID(ctx context.Context, q *gen.Queries, aid pgtype.UUID) (pgtype.UUID, error) {
	id, err := q.GetKeyDefaultKeyID(ctx, gen.GetKeyDefaultKeyIDParams{AccountID: aid, Feature: FeatureCoach})
	if errors.Is(err, pgx.ErrNoRows) {
		return pgtype.UUID{}, nil
	}
	if err != nil {
		return pgtype.UUID{}, fmt.Errorf("get coach default key id: %w", err)
	}
	return id, nil
}

// Package store is the identity service's persistence layer: pgx/pgxpool over the
// sqlc-generated queries in ./gen, goose migrations in ./migrations, and the
// transactional outbox (ADR-0004/0005). It exposes small domain types with plain
// string ids and time.Time so the HTTP layer never touches pgtype, and keeps the
// xlearn_identity role scoped to schema identity (all SQL is schema-qualified).
package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sujaykumarsuman/xlearn/internal/identity/store/gen"
)

// Event subject + version for the account-created fact (events.md). The relay
// publishes these to the XLEARN_IDENTITY JetStream stream.
const (
	SubjectAccountCreated = "xlearn.identity.account_created"
	accountCreatedVersion = 1
)

// ErrNotFound is returned when a lookup matches no row (mapped to 404/401 above).
var ErrNotFound = errors.New("identity: not found")

// Account is an xLearn user (the parts the HTTP layer needs this sprint).
type Account struct {
	ID          string
	DisplayName string
	Email       string // "" when the provider gave no email
	Timezone    string
	// StudyBudget / Reminders are the raw jsonb blobs, surfaced only on the internal
	// service-to-service endpoint the review workers call (ADR-0016); never on /me.
	StudyBudget []byte
	Reminders   []byte
	CreatedAt   time.Time
}

// Onboarding is the 3-step first-run state for an account.
type Onboarding struct {
	AccountID   string
	PathChosen  string    // "" until step 1 is completed
	BudgetSet   bool      // step 2 (deferred to S10)
	KeyAdded    bool      // step 3 (deferred to S11)
	CompletedAt time.Time // zero until onboarding completes
}

// Session is a server-side session behind the opaque HttpOnly cookie.
type Session struct {
	ID        string
	AccountID string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// OutboxRow is one unsent domain event awaiting relay to NATS.
type OutboxRow struct {
	EventID string
	Subject string
	Payload []byte
}

// OAuthUpsert carries the provider profile used to find-or-create an account.
type OAuthUpsert struct {
	Provider       string // "github" | "google"
	ProviderUserID string
	DisplayName    string
	Email          string // "" when unavailable
}

// AccountUpdate is a partial update to an account (PATCH /me, S10). A nil field
// leaves that column unchanged. DisplayName/Timezone use pointers so an intentional
// value is distinguishable from "not provided"; the jsonb blobs are validated +
// canonicalised by the handler before they reach the store.
type AccountUpdate struct {
	DisplayName *string
	Timezone    *string
	StudyBudget []byte // canonical study_budget_json; nil = leave unchanged
	Reminders   []byte // canonical reminders_json; nil = leave unchanged
}

// Store is the identity persistence seam. The HTTP handlers depend on this
// interface so they can be unit-tested against an in-memory fake.
type Store interface {
	// FindOrCreateAccount matches an existing account by (provider, providerUserID)
	// or, on first sign-in, creates account+oauth_identity+onboarding and writes the
	// account_created outbox row in one transaction. created reports first sign-in.
	FindOrCreateAccount(ctx context.Context, in OAuthUpsert) (acct Account, created bool, err error)
	GetAccount(ctx context.Context, id string) (Account, error)
	// UpdateAccount applies a partial profile/budget/timezone/reminders update to the
	// caller's own account and returns the updated row (PATCH /me).
	UpdateAccount(ctx context.Context, id string, in AccountUpdate) (Account, error)
	GetOnboarding(ctx context.Context, accountID string) (Onboarding, error)
	SetOnboardingPath(ctx context.Context, accountID, path string) (Onboarding, error)
	// SetOnboardingBudget writes the study budget to the account and sets
	// onboarding.budget_set in one transaction (onboarding step 2).
	SetOnboardingBudget(ctx context.Context, accountID string, budgetJSON []byte) (Onboarding, error)
	// CompleteOnboarding stamps onboarding.completed_at (idempotent) — onboarding
	// step 3 (Finish / Skip). key_added is NOT set here (deferred to S11).
	CompleteOnboarding(ctx context.Context, accountID string) (Onboarding, error)
	CreateSession(ctx context.Context, id, accountID string, expiresAt time.Time) (Session, error)
	GetValidSession(ctx context.Context, id string) (Session, error)
	RevokeSession(ctx context.Context, id string) (revoked bool, err error)
	ListUnsentOutbox(ctx context.Context, limit int32) ([]OutboxRow, error)
	MarkOutboxSent(ctx context.Context, eventID string) error
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

// FindOrCreateAccount implements the OAuth upsert with the transactional outbox.
func (s *PgStore) FindOrCreateAccount(ctx context.Context, in OAuthUpsert) (Account, bool, error) {
	if acct, err := s.byProvider(ctx, in); err == nil {
		return acct, false, nil
	} else if !errors.Is(err, ErrNotFound) {
		return Account{}, false, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Account{}, false, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := s.q.WithTx(tx)

	acctRow, err := qtx.CreateAccount(ctx, gen.CreateAccountParams{
		DisplayName: in.DisplayName,
		Email:       textOrNull(in.Email),
	})
	if err != nil {
		return Account{}, false, fmt.Errorf("create account: %w", err)
	}
	if _, err := qtx.CreateOauthIdentity(ctx, gen.CreateOauthIdentityParams{
		AccountID:      acctRow.ID,
		Provider:       in.Provider,
		ProviderUserID: in.ProviderUserID,
	}); err != nil {
		// A concurrent first sign-in for the same identity lost the race on the
		// unique(provider, provider_user_id) constraint — fall back to the winner.
		if isUniqueViolation(err) {
			_ = tx.Rollback(ctx)
			acct, ferr := s.byProvider(ctx, in)
			return acct, false, ferr
		}
		return Account{}, false, fmt.Errorf("create oauth_identity: %w", err)
	}
	if _, err := qtx.CreateOnboarding(ctx, acctRow.ID); err != nil {
		return Account{}, false, fmt.Errorf("create onboarding: %w", err)
	}

	eventID := newUUIDv4()
	payload, err := marshalAccountCreated(eventID, uuidString(acctRow.ID), in.Provider, in.DisplayName, acctRow.CreatedAt.Time)
	if err != nil {
		return Account{}, false, fmt.Errorf("marshal account_created: %w", err)
	}
	if err := qtx.InsertOutbox(ctx, gen.InsertOutboxParams{
		EventID:     mustUUID(eventID),
		Subject:     SubjectAccountCreated,
		PayloadJson: payload,
	}); err != nil {
		return Account{}, false, fmt.Errorf("insert outbox: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return Account{}, false, fmt.Errorf("commit tx: %w", err)
	}
	return toAccount(acctRow), true, nil
}

func (s *PgStore) byProvider(ctx context.Context, in OAuthUpsert) (Account, error) {
	row, err := s.q.GetAccountByProviderIdentity(ctx, gen.GetAccountByProviderIdentityParams{
		Provider:       in.Provider,
		ProviderUserID: in.ProviderUserID,
	})
	if err != nil {
		return Account{}, mapErr(err)
	}
	return toAccount(row), nil
}

// GetAccount returns an account by id.
func (s *PgStore) GetAccount(ctx context.Context, id string) (Account, error) {
	uid, err := parseUUID(id)
	if err != nil {
		return Account{}, ErrNotFound
	}
	row, err := s.q.GetAccount(ctx, uid)
	if err != nil {
		return Account{}, mapErr(err)
	}
	return toAccount(row), nil
}

// GetOnboarding returns an account's onboarding state.
func (s *PgStore) GetOnboarding(ctx context.Context, accountID string) (Onboarding, error) {
	uid, err := parseUUID(accountID)
	if err != nil {
		return Onboarding{}, ErrNotFound
	}
	row, err := s.q.GetOnboarding(ctx, uid)
	if err != nil {
		return Onboarding{}, mapErr(err)
	}
	return toOnboarding(row), nil
}

// SetOnboardingPath persists the chosen path (onboarding step 1).
func (s *PgStore) SetOnboardingPath(ctx context.Context, accountID, path string) (Onboarding, error) {
	uid, err := parseUUID(accountID)
	if err != nil {
		return Onboarding{}, ErrNotFound
	}
	row, err := s.q.SetOnboardingPath(ctx, gen.SetOnboardingPathParams{
		AccountID:  uid,
		PathChosen: pgtype.Text{String: path, Valid: path != ""},
	})
	if err != nil {
		return Onboarding{}, mapErr(err)
	}
	return toOnboarding(row), nil
}

// UpdateAccount applies a partial update (PATCH /me) and returns the updated row.
func (s *PgStore) UpdateAccount(ctx context.Context, id string, in AccountUpdate) (Account, error) {
	uid, err := parseUUID(id)
	if err != nil {
		return Account{}, ErrNotFound
	}
	row, err := s.q.UpdateAccount(ctx, gen.UpdateAccountParams{
		ID:              uid,
		DisplayName:     textPtr(in.DisplayName),
		Timezone:        textPtr(in.Timezone),
		StudyBudgetJson: in.StudyBudget,
		RemindersJson:   in.Reminders,
	})
	if err != nil {
		return Account{}, mapErr(err)
	}
	return toAccount(row), nil
}

// SetOnboardingBudget writes the study budget to the account and flips
// onboarding.budget_set in one transaction (onboarding step 2). Both rows are keyed
// on the same account, so a missing account/onboarding row maps to ErrNotFound.
func (s *PgStore) SetOnboardingBudget(ctx context.Context, accountID string, budgetJSON []byte) (Onboarding, error) {
	uid, err := parseUUID(accountID)
	if err != nil {
		return Onboarding{}, ErrNotFound
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Onboarding{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := s.q.WithTx(tx)

	if _, err := qtx.UpdateAccount(ctx, gen.UpdateAccountParams{ID: uid, StudyBudgetJson: budgetJSON}); err != nil {
		return Onboarding{}, mapErr(err)
	}
	row, err := qtx.SetOnboardingBudgetSet(ctx, uid)
	if err != nil {
		return Onboarding{}, mapErr(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Onboarding{}, fmt.Errorf("commit tx: %w", err)
	}
	return toOnboarding(row), nil
}

// CompleteOnboarding stamps onboarding.completed_at (idempotent) and returns the row.
func (s *PgStore) CompleteOnboarding(ctx context.Context, accountID string) (Onboarding, error) {
	uid, err := parseUUID(accountID)
	if err != nil {
		return Onboarding{}, ErrNotFound
	}
	row, err := s.q.CompleteOnboarding(ctx, uid)
	if err != nil {
		return Onboarding{}, mapErr(err)
	}
	return toOnboarding(row), nil
}

// CreateSession inserts a session row with the opaque id and expiry.
func (s *PgStore) CreateSession(ctx context.Context, id, accountID string, expiresAt time.Time) (Session, error) {
	uid, err := parseUUID(accountID)
	if err != nil {
		return Session{}, ErrNotFound
	}
	row, err := s.q.CreateSession(ctx, gen.CreateSessionParams{
		ID:        id,
		AccountID: uid,
		ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
	})
	if err != nil {
		return Session{}, fmt.Errorf("create session: %w", err)
	}
	return toSession(row), nil
}

// GetValidSession returns a non-revoked, non-expired session by id.
func (s *PgStore) GetValidSession(ctx context.Context, id string) (Session, error) {
	row, err := s.q.GetValidSession(ctx, id)
	if err != nil {
		return Session{}, mapErr(err)
	}
	return toSession(row), nil
}

// RevokeSession marks a session revoked; revoked reports whether a row changed.
func (s *PgStore) RevokeSession(ctx context.Context, id string) (bool, error) {
	n, err := s.q.RevokeSession(ctx, id)
	if err != nil {
		return false, fmt.Errorf("revoke session: %w", err)
	}
	return n > 0, nil
}

// ListUnsentOutbox returns up to limit unsent outbox rows for the relay.
func (s *PgStore) ListUnsentOutbox(ctx context.Context, limit int32) ([]OutboxRow, error) {
	rows, err := s.q.ListUnsentOutbox(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("list unsent outbox: %w", err)
	}
	out := make([]OutboxRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, OutboxRow{
			EventID: uuidString(r.EventID),
			Subject: r.Subject,
			Payload: r.PayloadJson,
		})
	}
	return out, nil
}

// MarkOutboxSent stamps sent_at on a relayed outbox row.
func (s *PgStore) MarkOutboxSent(ctx context.Context, eventID string) error {
	uid, err := parseUUID(eventID)
	if err != nil {
		return fmt.Errorf("parse event id: %w", err)
	}
	if err := s.q.MarkOutboxSent(ctx, uid); err != nil {
		return fmt.Errorf("mark outbox sent: %w", err)
	}
	return nil
}

// --- envelope + conversions ---

// marshalAccountCreated builds the events.md envelope for account_created.
func marshalAccountCreated(eventID, accountID, provider, displayName string, occurredAt time.Time) ([]byte, error) {
	return json.Marshal(map[string]any{
		"event_id":    eventID,
		"subject":     SubjectAccountCreated,
		"occurred_at": occurredAt.UTC().Format(time.RFC3339Nano),
		"version":     accountCreatedVersion,
		"account_id":  accountID,
		"data": map[string]any{
			"provider":     provider,
			"display_name": displayName,
		},
	})
}

func toAccount(a gen.IdentityAccount) Account {
	return Account{
		ID:          uuidString(a.ID),
		DisplayName: a.DisplayName,
		Email:       a.Email.String,
		Timezone:    a.Timezone,
		StudyBudget: a.StudyBudgetJson,
		Reminders:   a.RemindersJson,
		CreatedAt:   a.CreatedAt.Time,
	}
}

func toOnboarding(o gen.IdentityOnboarding) Onboarding {
	return Onboarding{
		AccountID:   uuidString(o.AccountID),
		PathChosen:  o.PathChosen.String,
		BudgetSet:   o.BudgetSet,
		KeyAdded:    o.KeyAdded,
		CompletedAt: o.CompletedAt.Time,
	}
}

func toSession(s gen.IdentitySession) Session {
	return Session{
		ID:        s.ID,
		AccountID: uuidString(s.AccountID),
		CreatedAt: s.CreatedAt.Time,
		ExpiresAt: s.ExpiresAt.Time,
	}
}

func textOrNull(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: s != ""}
}

// textPtr maps an optional string to a nullable pgtype.Text: nil → NULL (COALESCE
// leaves the column unchanged), non-nil → the value (even if empty — the caller
// validates non-emptiness before it reaches here).
func textPtr(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func mapErr(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

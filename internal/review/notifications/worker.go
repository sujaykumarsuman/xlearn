package notifications

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/platform/events"
)

// ReminderStore persists reminders idempotently (the review store satisfies it).
type ReminderStore interface {
	// HandleRevisionDue dedupes on eventID and writes one reminder at dueAt in one
	// transaction, returning whether a row was newly written.
	HandleRevisionDue(ctx context.Context, eventID, accountID, kind string, dueAt time.Time) (bool, error)
}

// AccountResolver resolves an account's timezone + study budget + reminder prefs (the
// identity client) so reminders can be localized and gated (S10).
type AccountResolver interface {
	ResolveAccount(ctx context.Context, accountID string) (timezone string, studyBudget, reminders []byte, err error)
}

// Handler consumes xlearn.review.revision_due and writes an in-app reminder scheduled
// inside the account's study-budget window. It is idempotent (the store dedupes on
// event_id): a redelivered event is a safe no-op. A nil return acks; a non-nil return
// naks for redelivery (at-least-once).
type Handler struct {
	store    ReminderStore
	accounts AccountResolver
	log      *slog.Logger
	now      func() time.Time // injectable for tests; nil → time.Now
}

var _ events.Handler = (*Handler)(nil)

// NewHandler builds the notifications handler. accounts may be nil (local dev / no
// identity URL) — reminders then use the default prefs (a daily nudge at 20:00 UTC).
func NewHandler(store ReminderStore, accounts AccountResolver, log *slog.Logger) *Handler {
	return &Handler{store: store, accounts: accounts, log: log}
}

// envelope is the events.md envelope as the worker reads it off XLEARN_REVIEW.
type envelope struct {
	EventID   string          `json:"event_id"`
	Subject   string          `json:"subject"`
	AccountID string          `json:"account_id"`
	Data      json.RawMessage `json:"data"`
}

// Handle processes one revision_due event.
func (h *Handler) Handle(ctx context.Context, e events.Event) error {
	// Belt-and-braces: the consumer is filtered to revision_due, but ignore anything
	// else that reaches this handler.
	if e.Subject != "" && e.Subject != "xlearn.review.revision_due" {
		return nil
	}
	var env envelope
	if err := json.Unmarshal(e.Data, &env); err != nil {
		h.log.Error("notifications: undecodable envelope; dropping", "err", err)
		return nil
	}
	eventID := env.EventID
	if eventID == "" {
		eventID = e.ID // Nats-Msg-Id fallback
	}
	if eventID == "" || env.AccountID == "" {
		h.log.Error("notifications: missing event_id/account_id; dropping", "subject", e.Subject)
		return nil
	}

	tz := "UTC"
	var budget, reminders []byte
	if h.accounts != nil {
		resolvedTz, resolvedBudget, resolvedReminders, err := h.accounts.ResolveAccount(ctx, env.AccountID)
		if err != nil {
			// Don't decide send-vs-suppress without the learner's real prefs: a resolve
			// failure naks for redelivery (at-least-once, escalating backoff) so a transient
			// identity blip can't override an explicit reminders opt-out with the ON defaults.
			// A nil client (local dev / no identity URL) returns no error and uses defaults.
			return fmt.Errorf("resolve account prefs: %w", err)
		}
		if resolvedTz != "" {
			tz = resolvedTz
		}
		budget = resolvedBudget
		reminders = resolvedReminders
	}

	write, dueAt := computeReminder(h.clock(), tz, budget, reminders)
	if !write {
		// The learner has turned off the applicable channel (daily nudge / due alerts).
		h.log.Info("notifications: reminder suppressed by prefs", "account_id", env.AccountID)
		return nil
	}
	wrote, err := h.store.HandleRevisionDue(ctx, eventID, env.AccountID, ReminderKind, dueAt)
	if err != nil {
		return fmt.Errorf("write reminder: %w", err)
	}
	if wrote {
		h.log.Info("in-app reminder scheduled", "account_id", env.AccountID, "due_at", dueAt.UTC().Format(time.RFC3339))
	}
	return nil
}

func (h *Handler) clock() time.Time {
	if h.now != nil {
		return h.now()
	}
	return time.Now()
}

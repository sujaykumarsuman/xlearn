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

// AccountResolver resolves an account's timezone + study budget (the identity client).
type AccountResolver interface {
	ResolveAccount(ctx context.Context, accountID string) (timezone string, studyBudget []byte, err error)
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
// identity URL) — reminders then schedule immediately (UTC, no budget window).
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
	var budget []byte
	if h.accounts != nil {
		if resolvedTz, resolvedBudget, err := h.accounts.ResolveAccount(ctx, env.AccountID); err != nil {
			// A resolve failure must not drop the reminder — schedule it immediately (UTC).
			h.log.Warn("notifications: account resolve failed; scheduling immediately", "account_id", env.AccountID, "err", err)
		} else {
			if resolvedTz != "" {
				tz = resolvedTz
			}
			budget = resolvedBudget
		}
	}

	dueAt := computeReminderTime(h.clock(), tz, budget)
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

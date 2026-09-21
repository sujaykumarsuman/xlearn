package review

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/platform/events"
	"github.com/sujaykumarsuman/xlearn/internal/review/store"
)

// envelope is the events.md envelope as review reads it off XLEARN_PRACTICE.
// account_id is envelope-level; the per-event fields live in data.
type envelope struct {
	EventID    string          `json:"event_id"`
	Subject    string          `json:"subject"`
	OccurredAt string          `json:"occurred_at"`
	AccountID  string          `json:"account_id"`
	Data       json.RawMessage `json:"data"`
}

// practiceHandler routes a delivered practice event to the right store method. It is
// the events.Handler bound to the durable consumer; the store dedupes on event_id via
// the inbox, so a redelivered message is a safe no-op (at-least-once → effectively
// once). A non-nil return triggers redelivery (nak); a nil return acks.
type practiceHandler struct {
	store store.Store
	log   *slog.Logger
}

var _ events.Handler = (*practiceHandler)(nil)

func (h *practiceHandler) Handle(ctx context.Context, e events.Event) error {
	var env envelope
	if err := json.Unmarshal(e.Data, &env); err != nil {
		// A malformed payload can never succeed on redelivery — log and ack (return
		// nil) so it does not wedge the consumer. The raw message stays on the stream.
		h.log.Error("review consumer: undecodable envelope; dropping", "subject", e.Subject, "err", err)
		return nil
	}
	eventID := env.EventID
	if eventID == "" {
		eventID = e.ID // fall back to the id the consumer resolved (Nats-Msg-Id)
	}
	if eventID == "" || env.AccountID == "" {
		h.log.Error("review consumer: missing event_id/account_id; dropping", "subject", e.Subject)
		return nil
	}
	occurredAt := parseOccurredAt(env.OccurredAt)

	switch e.Subject {
	case store.SubjectProblemSolved:
		return h.handleProblemSolved(ctx, eventID, env.AccountID, occurredAt, env.Data)
	case store.SubjectSolutionRevealedEarly:
		return h.handleSolutionRevealedEarly(ctx, eventID, env.AccountID, occurredAt, env.Data)
	default:
		// e.g. xlearn.practice.attempt_logged — captured by the xlearn.practice.*
		// filter but not acted on by review (consumed by assessment). Ack + move on.
		return nil
	}
}

func (h *practiceHandler) handleProblemSolved(ctx context.Context, eventID, accountID string, occurredAt time.Time, data []byte) error {
	var d struct {
		ProblemID  string `json:"problem_id"`
		Outcome    string `json:"outcome"`
		FirstSolve bool   `json:"first_solve"`
	}
	if err := json.Unmarshal(data, &d); err != nil {
		h.log.Error("review consumer: bad problem_solved data; dropping", "err", err)
		return nil
	}
	if d.ProblemID == "" {
		h.log.Error("review consumer: problem_solved missing problem_id; dropping")
		return nil
	}
	n, err := h.store.HandleProblemSolved(ctx, eventID, accountID, d.ProblemID, d.Outcome, d.FirstSolve, occurredAt)
	if err != nil {
		return fmt.Errorf("handle problem_solved: %w", err)
	}
	if n > 0 {
		h.log.Info("scheduled five-touch revision", "problem_id", d.ProblemID, "touches", n)
	}
	return nil
}

func (h *practiceHandler) handleSolutionRevealedEarly(ctx context.Context, eventID, accountID string, occurredAt time.Time, data []byte) error {
	var d struct {
		ProblemID string `json:"problem_id"`
	}
	if err := json.Unmarshal(data, &d); err != nil {
		h.log.Error("review consumer: bad solution_revealed_early data; dropping", "err", err)
		return nil
	}
	if d.ProblemID == "" {
		h.log.Error("review consumer: solution_revealed_early missing problem_id; dropping")
		return nil
	}
	n, err := h.store.HandleSolutionRevealedEarly(ctx, eventID, accountID, d.ProblemID, occurredAt)
	if err != nil {
		return fmt.Errorf("handle solution_revealed_early: %w", err)
	}
	if n > 0 {
		h.log.Info("scheduled owed 3-day re-solve", "problem_id", d.ProblemID)
	}
	return nil
}

// parseOccurredAt parses the envelope timestamp (RFC3339 with optional nanos),
// falling back to now if absent/unparseable so scheduling always has an anchor.
func parseOccurredAt(s string) time.Time {
	if s == "" {
		return time.Now()
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}
	return time.Now()
}

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

// practiceHandler routes a delivered practice event to the right store method. It is
// the events.Handler bound to the durable consumer; the store dedupes on event_id via
// the inbox, so a redelivered message is a safe no-op (at-least-once → effectively
// once). A non-nil return triggers redelivery (nak); a nil return acks.
//
// It decodes v1 AND v2 envelopes (events.DecodeEnvelope, m1-02): a v1 event is DSA, a
// v2 one carries its course in path_slug, and the course is written on every row the
// store creates. A v2 practice event without path_slug is an events.ErrInvalidEnvelope,
// returned so the consumer dead-letters it at once.
type practiceHandler struct {
	store store.Store
	log   *slog.Logger
}

var _ events.Handler = (*practiceHandler)(nil)

func (h *practiceHandler) Handle(ctx context.Context, e events.Event) error {
	switch e.Subject {
	case store.SubjectProblemSolved, store.SubjectSolutionRevealedEarly:
	default:
		// A subject review deliberately doesn't act on (xlearn.practice.attempt_logged,
		// assessment's) is listed under Ignores in topology.go: ack quietly. Anything
		// else is a subject published before review learned it — never ack that
		// silently (ADR-0035 §1.1): log ERROR, then ack so it can't wedge the durable.
		if !events.Ignored(DurableName, e.Subject) {
			h.log.Error("review consumer: unlisted subject; acked without handling",
				"subject", e.Subject, "event_id", e.ID, "durable", DurableName)
		}
		return nil
	}

	env, err := events.DecodeEnvelope(e.Data)
	if err != nil {
		// Malformed JSON or a v2 practice event without path_slug can never succeed
		// on redelivery: the consumer dead-letters it on this delivery (a row + ERROR
		// log). The raw message stays on the stream.
		return fmt.Errorf("review consumer: %s: %w", e.Subject, err)
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

	if e.Subject == store.SubjectProblemSolved {
		return h.handleProblemSolved(ctx, eventID, env.AccountID, env.PathSlug, occurredAt, env.Data)
	}
	return h.handleSolutionRevealedEarly(ctx, eventID, env.AccountID, env.PathSlug, occurredAt, env.Data)
}

func (h *practiceHandler) handleProblemSolved(ctx context.Context, eventID, accountID, pathSlug string, occurredAt time.Time, data []byte) error {
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
	n, err := h.store.HandleProblemSolved(ctx, eventID, accountID, pathSlug, d.ProblemID, d.Outcome, d.FirstSolve, occurredAt)
	if err != nil {
		return fmt.Errorf("handle problem_solved: %w", err)
	}
	if n > 0 {
		h.log.Info("scheduled five-touch revision", "problem_id", d.ProblemID, "touches", n)
	}
	return nil
}

func (h *practiceHandler) handleSolutionRevealedEarly(ctx context.Context, eventID, accountID, pathSlug string, occurredAt time.Time, data []byte) error {
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
	n, err := h.store.HandleSolutionRevealedEarly(ctx, eventID, accountID, pathSlug, d.ProblemID, occurredAt)
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

package assessment

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/assessment/store"
	"github.com/sujaykumarsuman/xlearn/internal/platform/events"
)

// eventData is the union of the per-subject fields the projections read: problem_id +
// outcome + first_solve (practice solves) and touch_level (review schedules). Unknown
// fields are ignored (events are additive-only).
type eventData struct {
	ProblemID  string `json:"problem_id"`
	Outcome    string `json:"outcome"`
	FirstSolve bool   `json:"first_solve"`
	TouchLevel int    `json:"touch_level"`
}

// projectionHandler is the S09 progress-projection consumer bound to the durable
// consumers on XLEARN_PRACTICE / XLEARN_REVIEW. It decodes the envelope, dedupes on
// event_id and applies the coverage / mastery / heatmap / outcome-mix upserts — all in
// one transaction via store.ApplyProjection, so inbox <-> projected stays atomic. It is
// the events.Handler bound to the durable consumer: a non-nil return triggers
// redelivery (nak); nil acks. Handlers are a pure function of the event log so a
// drop-and-replay rebuild is deterministic (ADR-0017/0018).
//
// It decodes v1 AND v2 envelopes (events.DecodeEnvelope, m1-02): a v1 event is DSA, a
// v2 one carries its course in path_slug (carried on ProjectionEvent). A v2 practice or
// review event without path_slug is an events.ErrInvalidEnvelope, returned so the
// consumer dead-letters it at once.
type projectionHandler struct {
	store store.Store
	log   *slog.Logger
}

var _ events.Handler = (*projectionHandler)(nil)

func (h *projectionHandler) Handle(ctx context.Context, e events.Event) error {
	// Subject registry (ADR-0035 §1.1, topology.go): assessment handles every practice
	// and review subject v1 emits. A subject listed under Ignores is acked quietly; an
	// unlisted one (published before assessment learned it) is never acked silently.
	switch {
	case events.Handled(DurableName, e.Subject):
	case events.Ignored(DurableName, e.Subject):
		return nil
	default:
		h.log.Error("assessment consumer: unlisted subject; acked without handling",
			"subject", e.Subject, "event_id", e.ID, "durable", DurableName)
		return nil
	}

	env, err := events.DecodeEnvelope(e.Data)
	if err != nil {
		// Malformed JSON or a v2 event without path_slug can never succeed on
		// redelivery: the consumer dead-letters it on this delivery (a row + ERROR log).
		// The raw message stays on the stream.
		return fmt.Errorf("assessment consumer: %s: %w", e.Subject, err)
	}
	eventID := env.EventID
	if eventID == "" {
		eventID = e.ID // fall back to the id the consumer resolved (Nats-Msg-Id)
	}
	if eventID == "" || env.AccountID == "" {
		h.log.Error("assessment consumer: missing event_id/account_id; dropping", "subject", e.Subject)
		return nil
	}

	// Decode the per-event fields (best effort — an undecodable data blob still dedupes
	// and no-ops rather than wedging the consumer, since it can't succeed on redelivery).
	var d eventData
	if len(env.Data) > 0 {
		if err := json.Unmarshal(env.Data, &d); err != nil {
			h.log.Error("assessment consumer: undecodable data; recording + dropping", "subject", e.Subject, "err", err)
			d = eventData{}
		}
	}

	ev := store.ProjectionEvent{
		EventID:    eventID,
		Subject:    e.Subject,
		AccountID:  env.AccountID,
		PathSlug:   env.PathSlug,
		ProblemID:  d.ProblemID,
		Outcome:    d.Outcome,
		FirstSolve: d.FirstSolve,
		TouchLevel: d.TouchLevel,
		OccurredAt: parseOccurredAt(env.OccurredAt),
	}

	fresh, err := h.store.ApplyProjection(ctx, ev)
	if err != nil {
		return fmt.Errorf("apply projection %s: %w", e.Subject, err)
	}
	if fresh {
		h.log.Debug("assessment projection applied", "subject", e.Subject, "event_id", eventID)
	}
	return nil
}

// parseOccurredAt parses the envelope timestamp (RFC3339 with optional nanos), falling
// back to now if absent/unparseable so the heatmap day always has an anchor (our
// producers always set occurred_at, so the fallback is defensive only).
func parseOccurredAt(s string) time.Time {
	if s == "" {
		return time.Now().UTC()
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t.UTC()
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC()
	}
	return time.Now().UTC()
}

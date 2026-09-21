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

// envelope is the events.md envelope as assessment reads it off XLEARN_PRACTICE /
// XLEARN_REVIEW. account_id is envelope-level; the per-event fields live in data.
type envelope struct {
	EventID    string          `json:"event_id"`
	Subject    string          `json:"subject"`
	OccurredAt string          `json:"occurred_at"`
	AccountID  string          `json:"account_id"`
	Data       json.RawMessage `json:"data"`
}

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
type projectionHandler struct {
	store store.Store
	log   *slog.Logger
}

var _ events.Handler = (*projectionHandler)(nil)

func (h *projectionHandler) Handle(ctx context.Context, e events.Event) error {
	var env envelope
	if err := json.Unmarshal(e.Data, &env); err != nil {
		// A malformed payload can never succeed on redelivery — log and ack (return
		// nil) so it does not wedge the consumer. The raw message stays on the stream.
		h.log.Error("assessment consumer: undecodable envelope; dropping", "subject", e.Subject, "err", err)
		return nil
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

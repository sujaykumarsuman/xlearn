package assessment

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

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

// projectionHandler is the S09 progress-projection consume seam bound to the durable
// consumers on XLEARN_PRACTICE / XLEARN_REVIEW. This sprint it decodes the envelope,
// dedupes on event_id via the inbox, and no-ops the projection body
// (store.RecordProjectionEvent); S09 fills the proj_* upserts into the same
// transaction so inbox <-> projected stays atomic. It is the events.Handler bound to
// the durable consumer: a non-nil return triggers redelivery (nak); nil acks.
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

	fresh, err := h.store.RecordProjectionEvent(ctx, eventID, e.Subject, env.AccountID, env.Data)
	if err != nil {
		return fmt.Errorf("record projection event %s: %w", e.Subject, err)
	}
	if fresh {
		h.log.Debug("assessment projection stub: event consumed (no-op; S09 builds projections)",
			"subject", e.Subject, "event_id", eventID)
	}
	return nil
}

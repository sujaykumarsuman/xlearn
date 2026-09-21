package events

import (
	"context"
	"log/slog"
	"time"
)

// OutboxSource is a service's transactional outbox as seen by the relay: it lists
// unsent events and marks one sent. Each owning service's store implements it.
type OutboxSource interface {
	// ListUnsent returns up to limit unsent outbox events, oldest first.
	ListUnsent(ctx context.Context, limit int32) ([]Event, error)
	// MarkSent stamps the event delivered (by Event.ID) so it is not re-relayed.
	MarkSent(ctx context.Context, eventID string) error
}

// Relay drains a service's outbox to a Publisher on an interval (ADR-0004): it
// lists unsent rows, publishes each, and marks it sent. A crash between publish
// and mark simply re-relays (at-least-once; consumers dedupe on Event.ID).
type Relay struct {
	src      OutboxSource
	pub      Publisher
	log      *slog.Logger
	interval time.Duration
	batch    int32
}

// RelayOption configures a Relay.
type RelayOption func(*Relay)

// WithInterval sets the poll interval (default 5s).
func WithInterval(d time.Duration) RelayOption { return func(r *Relay) { r.interval = d } }

// WithBatch sets the max rows drained per tick (default 100).
func WithBatch(n int32) RelayOption { return func(r *Relay) { r.batch = n } }

// NewRelay builds a Relay over src publishing to pub.
func NewRelay(src OutboxSource, pub Publisher, log *slog.Logger, opts ...RelayOption) *Relay {
	r := &Relay{src: src, pub: pub, log: log, interval: 5 * time.Second, batch: 100}
	for _, o := range opts {
		o(r)
	}
	return r
}

// Run drains the outbox until ctx is cancelled. It drains once immediately, then
// on each tick.
func (r *Relay) Run(ctx context.Context) {
	t := time.NewTicker(r.interval)
	defer t.Stop()
	r.drain(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			r.drain(ctx)
		}
	}
}

func (r *Relay) drain(ctx context.Context) {
	rows, err := r.src.ListUnsent(ctx, r.batch)
	if err != nil {
		r.log.Error("outbox relay: list unsent", "err", err)
		return
	}
	for _, e := range rows {
		if err := r.pub.Publish(ctx, e); err != nil {
			// Leave the row unsent; the next tick retries (backoff via interval).
			r.log.Warn("outbox relay: publish failed; will retry", "subject", e.Subject, "event_id", e.ID, "err", err)
			return
		}
		if err := r.src.MarkSent(ctx, e.ID); err != nil {
			r.log.Error("outbox relay: mark sent", "event_id", e.ID, "err", err)
			return
		}
	}
}

// LogPublisher is the no-broker fallback Publisher: it logs each event and reports
// success so the outbox drains rather than growing unbounded. It is used in local dev
// (no NATS_URL) and by producers whose JetStream stream isn't provisioned yet (e.g.
// identity's account_created — events.md: reserved, no consumer). Production practice
// uses NatsPublisher (S05); swap other producers to it as their streams land.
type LogPublisher struct {
	log    *slog.Logger
	stream string
}

// NewLogPublisher builds a LogPublisher tagging events with their JetStream stream.
func NewLogPublisher(log *slog.Logger, stream string) *LogPublisher {
	return &LogPublisher{log: log, stream: stream}
}

// Publish logs the event and returns nil (fallback — events are not delivered to a
// broker).
func (p *LogPublisher) Publish(_ context.Context, e Event) error {
	p.log.Info("outbox event published (log fallback: no NATS configured)",
		"stream", p.stream, "subject", e.Subject, "event_id", e.ID)
	return nil
}

var _ Publisher = (*LogPublisher)(nil)

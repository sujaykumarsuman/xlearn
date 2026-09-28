package events

import (
	"context"
	"errors"
	"fmt"
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

	// oversize remembers the event ids already logged as over MaxEnvelopeBytes, so
	// each is logged once per process rather than on every tick. drain runs on the
	// Run goroutine only, so it needs no lock.
	oversize map[string]struct{}
}

// MaxEnvelopeBytes caps one event envelope (L20, ADR-0035 §4): the relay refuses to
// publish a larger outbox row. v1 events are ~0.4 KiB (prod /jsz, 2026-09-24), far
// below; NATS' own max_payload (1 MiB) stays as the outer bound.
const MaxEnvelopeBytes = 16 << 10

// ErrEnvelopeTooLarge is returned by CheckEnvelope for an envelope over the cap.
var ErrEnvelopeTooLarge = errors.New("events: envelope exceeds MaxEnvelopeBytes")

// CheckEnvelope reports ErrEnvelopeTooLarge when the encoded envelope b is over
// MaxEnvelopeBytes. Producers call it before writing an outbox row (m1-02's v2
// envelope type does); the relay applies it again as the backstop.
func CheckEnvelope(b []byte) error {
	if len(b) > MaxEnvelopeBytes {
		return fmt.Errorf("%w: %d > %d bytes", ErrEnvelopeTooLarge, len(b), MaxEnvelopeBytes)
	}
	return nil
}

// RelayOption configures a Relay.
type RelayOption func(*Relay)

// WithInterval sets the poll interval (default 5s).
func WithInterval(d time.Duration) RelayOption { return func(r *Relay) { r.interval = d } }

// WithBatch sets the max rows drained per tick (default 100).
func WithBatch(n int32) RelayOption { return func(r *Relay) { r.batch = n } }

// NewRelay builds a Relay over src publishing to pub.
func NewRelay(src OutboxSource, pub Publisher, log *slog.Logger, opts ...RelayOption) *Relay {
	r := &Relay{src: src, pub: pub, log: log, interval: 5 * time.Second, batch: 100, oversize: map[string]struct{}{}}
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
		if err := CheckEnvelope(e.Data); err != nil {
			// Over the L20 cap: never published, left unsent (a Postgres row, so
			// nothing is lost), and the batch carries on — one oversize row must not
			// stall the rows behind it. Logged once per event id per process.
			if _, seen := r.oversize[e.ID]; !seen {
				r.oversize[e.ID] = struct{}{}
				r.log.Error("outbox relay: envelope over the cap; left unsent",
					"event_id", e.ID, "subject", e.Subject, "bytes", len(e.Data), "max_bytes", MaxEnvelopeBytes)
			}
			continue
		}
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
// success so the outbox drains rather than growing unbounded. It is used ONLY when
// NATS_URL is unset (local dev, and identity in prod until mi-06's N2 PR gives it
// NATS_URL and its seed together). With NATS_URL set, a publisher init error is fatal
// (fail closed, mi-05): falling back here would mark rows sent without delivering them.
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

package events

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// publishTimeout bounds a single JetStream publish+ack so the relay never blocks
// indefinitely when NATS is unreachable — the row stays unsent and the next tick
// retries (ADR-0004: at-least-once via the outbox).
const publishTimeout = 5 * time.Second

// dedupeWindow is how long JetStream remembers a message id (Nats-Msg-Id) to drop
// duplicates. The relay is at-least-once (it may re-publish a row after a crash
// between publish and mark-sent), so publishing with the event id as the message id
// makes redelivery within this window a no-op at the broker.
const dedupeWindow = 5 * time.Minute

// NatsPublisher publishes outbox events to a JetStream stream, deduplicating on the
// event id. It implements Publisher; the relay drains outbox rows through it.
//
// The NATS connection auto-reconnects, so a brief broker outage is transparent:
// Publish fails, the relay leaves the row unsent, and a later tick re-publishes.
type NatsPublisher struct {
	nc       *nats.Conn
	js       jetstream.JetStream
	log      *slog.Logger
	stream   string
	subjects []string

	mu      sync.Mutex
	ensured bool
}

// NewNatsPublisher connects to NATS and prepares a JetStream publisher for stream.
// The stream is created (or updated) with the given subjects; if NATS is not yet
// reachable the stream is ensured lazily on the first successful publish.
func NewNatsPublisher(ctx context.Context, url, stream string, subjects []string, log *slog.Logger) (*NatsPublisher, error) {
	nc, err := nats.Connect(url,
		nats.Name("xlearn-"+stream),
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("nats connect %q: %w", url, err)
	}
	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("jetstream: %w", err)
	}
	p := &NatsPublisher{nc: nc, js: js, log: log, stream: stream, subjects: subjects}
	// Best-effort at startup; NATS may still be coming up (the relay retries).
	if err := p.ensureStream(ctx); err != nil {
		log.Warn("jetstream stream not ensured at startup; will retry on publish",
			"stream", stream, "err", err)
	}
	return p, nil
}

// ensureStream creates or updates the stream once. It is safe to call on every
// publish: it no-ops after the first success.
func (p *NatsPublisher) ensureStream(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.ensured {
		return nil
	}
	cctx, cancel := context.WithTimeout(ctx, publishTimeout)
	defer cancel()
	if _, err := p.js.CreateOrUpdateStream(cctx, jetstream.StreamConfig{
		Name:       p.stream,
		Subjects:   p.subjects,
		Storage:    jetstream.FileStorage,
		Duplicates: dedupeWindow,
	}); err != nil {
		return err
	}
	p.ensured = true
	p.log.Info("jetstream stream ready", "stream", p.stream, "subjects", p.subjects)
	return nil
}

// Publish sends e to its subject with the event id as the JetStream message id, so
// an at-least-once redelivery is deduplicated at the broker.
func (p *NatsPublisher) Publish(ctx context.Context, e Event) error {
	if err := p.ensureStream(ctx); err != nil {
		return fmt.Errorf("ensure stream: %w", err)
	}
	cctx, cancel := context.WithTimeout(ctx, publishTimeout)
	defer cancel()
	if _, err := p.js.Publish(cctx, e.Subject, e.Data, jetstream.WithMsgID(e.ID)); err != nil {
		return fmt.Errorf("js publish %s: %w", e.Subject, err)
	}
	return nil
}

// Close drains and closes the NATS connection.
func (p *NatsPublisher) Close() {
	if p.nc != nil {
		p.nc.Close()
	}
}

var _ Publisher = (*NatsPublisher)(nil)

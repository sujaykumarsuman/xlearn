// Package events is the asynchronous-messaging seam for xLearn: a transactional
// outbox feeding NATS JetStream, with idempotent consumers (ADR-0004). Producers
// write a domain change and an outbox row in one Postgres transaction; the Relay
// publishes unacked rows and marks them sent; consumers dedupe on Event.ID.
//
// This package ships the event envelope, the Publisher/Consumer/Handler seams, the
// Relay (relay.go), the JetStream NatsPublisher (nats.go, S05), the durable pull
// NatsConsumer (consumer.go, S06 — review is the first consumer) and the no-broker
// LogPublisher fallback.
package events

import "context"

// Event is a domain fact. Subjects follow xlearn.<context>.<event>
// (e.g. xlearn.practice.problem_solved). Events are versioned and additive-only.
type Event struct {
	// ID is the globally-unique event id; consumers dedupe on it (at-least-once).
	ID string
	// Subject is the NATS subject the event is published on.
	Subject string
	// Data is the JSON-encoded event payload.
	Data []byte
}

// Publisher publishes an event. In production this writes to the outbox inside
// the caller's transaction; the relay does the actual NATS publish.
type Publisher interface {
	// Publish enqueues e for at-least-once delivery.
	Publish(ctx context.Context, e Event) error
}

// Handler processes a delivered event. It must be idempotent: the same Event.ID
// may be delivered more than once.
type Handler interface {
	// Handle processes e, returning an error to trigger redelivery.
	Handle(ctx context.Context, e Event) error
}

// Consumer binds a durable handler to a subject filter on a JetStream stream.
type Consumer interface {
	// Subscribe registers h under durable-consumer name `durable` for events
	// matching filterSubject, delivering until Stop or ctx cancellation. The
	// durable name makes delivery resumable (offline catch-up) and at-least-once.
	Subscribe(ctx context.Context, durable, filterSubject string, h Handler) (Subscription, error)
}

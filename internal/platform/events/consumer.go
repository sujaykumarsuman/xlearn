package events

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// ackWait bounds how long the server waits for an ack before redelivering a
// message (at-least-once). A handler that panics or blocks is redelivered after
// this window rather than being lost.
const ackWait = 30 * time.Second

// handleTimeout bounds a single handler invocation. It runs on a context derived
// from Background (not the subscription's ctx), so an in-flight handler can commit
// and ack during shutdown drain rather than failing on a cancelled context and
// naking (which would waste redelivery budget on every rolling deploy). It stays
// below the last delivery's ack wait, so the final failing delivery returns its
// error (and dead-letters) before the server could redeliver.
const handleTimeout = 25 * time.Second

// maxDeliver caps redelivery attempts for a persistently-failing message so a
// poison event can't be retried forever. Combined with redeliveryBackoff it spans
// hours (not milliseconds), so a transient DB/broker outage is a bounded, recoverable
// retry — never an instant budget burn — while a genuinely un-processable event
// eventually stops: on its last failing delivery it is dead-lettered (a Postgres row
// via WithDeadLetter), terminated and logged at ERROR (ADR-0035 §1.2). The raw
// message stays on the stream for inspection/replay.
const maxDeliver = 100

// redeliveryBackoff is the escalating delay between redeliveries. A handler error
// naks WITH this delay (NakWithDelay), and it is also the consumer's server-side
// BackOff for the ack-timeout path. Delays beyond the slice repeat the last entry,
// so maxDeliver=100 attempts span ~8 hours rather than being exhausted in under a
// second by a hot nak loop.
var redeliveryBackoff = []time.Duration{
	1 * time.Second, 5 * time.Second, 15 * time.Second, 30 * time.Second,
	1 * time.Minute, 2 * time.Minute, 5 * time.Minute,
}

// backoffFor picks the redelivery delay for the n-th delivery (n starts at 1),
// clamping past the end of the schedule to its last (longest) entry.
func backoffFor(numDelivered uint64) time.Duration {
	return backoffIn(redeliveryBackoff, numDelivered)
}

func backoffIn(schedule []time.Duration, numDelivered uint64) time.Duration {
	if numDelivered == 0 {
		numDelivered = 1
	}
	i := int(numDelivered - 1)
	if i >= len(schedule) {
		i = len(schedule) - 1
	}
	return schedule[i]
}

// Subscription is a running durable subscription. Stop halts delivery (the
// durable consumer and its offset survive on the server for the next start).
type Subscription interface {
	// Stop ends delivery to the handler. It does not delete the durable consumer.
	Stop()
}

// NatsConsumer binds durable pull consumers on a JetStream stream and delivers
// each message to a Handler, acking on success and nacking (for redelivery) on
// error (ADR-0004: at-least-once; handlers dedupe on Event.ID for effectively
// once). It reads the stream a producing service owns; it never creates it.
type NatsConsumer struct {
	nc     *nats.Conn
	js     jetstream.JetStream
	log    *slog.Logger
	svc    string
	stream string
}

// NewNatsConsumer connects svc to NATS (Dial: seed, inbox prefix, ErrorHandler) and
// prepares durable pull subscriptions on stream, which must be declared in
// topology.go. The connection auto-reconnects, so a brief broker outage is
// transparent — a redelivered message is deduped by the handler's inbox.
func NewNatsConsumer(ctx context.Context, svc, url, stream string, log *slog.Logger) (*NatsConsumer, error) {
	if _, ok := LookupStream(stream); !ok {
		return nil, fmt.Errorf("events: stream %q is not declared in topology.go", stream)
	}
	nc, err := Dial(ctx, svc, url, log, nats.Name(connName(svc, stream, "cons")))
	if err != nil {
		return nil, err
	}
	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("jetstream: %w", err)
	}
	return &NatsConsumer{nc: nc, js: js, log: log, svc: svc, stream: stream}, nil
}

// subscription is one bound durable: its name, delivery budget, backoff schedule and
// dead-letter sink.
type subscription struct {
	durable    string
	maxDeliver uint64
	backoff    []time.Duration
	sink       DeadLetterSink
}

// Subscribe creates (idempotently) a durable pull consumer named durable on the
// configured stream, filtered to filterSubject, and delivers messages to h until
// Stop is called or ctx is cancelled. The durable name + explicit ack make this a
// resumable, at-least-once subscription: on restart it continues from its last
// acked offset, so events that arrived while the service was down are still
// processed (offline catch-up).
//
// The (stream, durable, filter) must be declared in topology.go for this service;
// anything else is refused, so the rendered ACL and the running code can't drift.
//
// The consumer is created lazily with retry because the producer that owns the
// stream may still be provisioning it on a fresh cluster; Subscribe blocks (with
// backoff, honouring ctx) until the stream exists.
func (c *NatsConsumer) Subscribe(ctx context.Context, durable, filterSubject string, h Handler, opts ...SubscribeOption) (Subscription, error) {
	d, ok := LookupDurable(c.stream, durable)
	if !ok {
		return nil, fmt.Errorf("events: durable %s/%s is not declared in topology.go", c.stream, durable)
	}
	if d.Filter != filterSubject {
		return nil, fmt.Errorf("events: durable %s/%s filter %q, topology.go declares %q", c.stream, durable, filterSubject, d.Filter)
	}
	if d.Service != c.svc {
		return nil, fmt.Errorf("events: durable %s/%s belongs to %s, not %s", c.stream, durable, d.Service, c.svc)
	}

	var sc SubscribeConfig
	for _, opt := range opts {
		opt(&sc)
	}
	sub := subscription{durable: durable, maxDeliver: maxDeliver, backoff: redeliveryBackoff, sink: sc.DeadLetter}
	if sc.MaxDeliver > 0 {
		sub.maxDeliver = uint64(sc.MaxDeliver)
		sub.backoff = []time.Duration{testBackoff}
	}

	// A brand-new durable defaults to DeliverAll (replay history — correct for the
	// first-ever consumer of a fresh stream). DeliverNew starts at the stream head
	// instead, for a consumer attached to an already-populated stream.
	deliver := jetstream.DeliverAllPolicy
	if sc.DeliverNew {
		deliver = jetstream.DeliverNewPolicy
	}
	cfg := jetstream.ConsumerConfig{
		Durable:       durable,
		FilterSubject: filterSubject,
		AckPolicy:     jetstream.AckExplicitPolicy,
		AckWait:       ackWait,
		MaxDeliver:    int(sub.maxDeliver),
		BackOff:       sub.backoff,
		DeliverPolicy: deliver,
	}

	cons, err := c.createConsumer(ctx, cfg)
	if err != nil {
		return nil, err
	}

	// The handler runs on a context derived from Background (see dispatch), not the
	// subscription ctx, so shutdown stops delivery via Stop() rather than by
	// cancelling in-flight handlers.
	cctx, err := cons.Consume(func(msg jetstream.Msg) {
		c.dispatch(msg, h, sub)
	})
	if err != nil {
		return nil, fmt.Errorf("consume %s/%s: %w", c.stream, durable, err)
	}
	c.log.Info("jetstream durable consumer subscribed",
		"stream", c.stream, "durable", durable, "filter", filterSubject)
	return consumeSub{cctx}, nil
}

// createConsumer provisions the durable consumer, retrying until the stream the
// producer owns exists (or ctx is cancelled).
func (c *NatsConsumer) createConsumer(ctx context.Context, cfg jetstream.ConsumerConfig) (jetstream.Consumer, error) {
	backoff := 500 * time.Millisecond
	const maxBackoff = 10 * time.Second
	for {
		cctx, cancel := context.WithTimeout(ctx, publishTimeout)
		cons, err := c.js.CreateOrUpdateConsumer(cctx, c.stream, cfg)
		cancel()
		if err == nil {
			return cons, nil
		}
		c.log.Warn("jetstream consumer not ready; retrying",
			"stream", c.stream, "durable", cfg.Durable, "err", err)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
		}
		if backoff < maxBackoff {
			backoff *= 2
		}
	}
}

// dispatch delivers one message to the handler and acks/naks accordingly. The
// Event id is the envelope event_id (falling back to the Nats-Msg-Id header the
// publisher set), so the handler's inbox dedupe keys on a stable id. The handler
// runs on a fresh, bounded context (not the subscription ctx) so it can finish and
// ack during shutdown drain; on failure it naks WITH an escalating delay so a
// transient outage never burns the redelivery budget in a hot loop. On the LAST
// failing delivery it dead-letters instead (deadLetter) — and at once, on the first
// delivery, for an ErrInvalidEnvelope (m1-02: a v2 course-scoped event without
// path_slug or undecodable JSON, which no redelivery can fix).
func (c *NatsConsumer) dispatch(msg jetstream.Msg, h Handler, sub subscription) {
	ctx, cancel := context.WithTimeout(context.Background(), handleTimeout)
	defer cancel()

	e := Event{ID: eventID(msg), Subject: msg.Subject(), Data: msg.Data()}
	if err := h.Handle(ctx, e); err != nil {
		if errors.Is(err, ErrInvalidEnvelope) {
			c.deadLetter(msg, e, sub, err, "event dead-lettered: invalid envelope")
			return
		}
		n := deliveryCount(msg)
		if n >= sub.maxDeliver {
			c.deadLetter(msg, e, sub, err, "event dead-lettered after max deliveries")
			return
		}
		delay := backoffIn(sub.backoff, n)
		c.log.Warn("event handler failed; will redeliver after backoff",
			"subject", e.Subject, "event_id", e.ID, "retry_in", delay.String(), "err", err)
		if nerr := msg.NakWithDelay(delay); nerr != nil {
			c.log.Error("nak failed", "event_id", e.ID, "err", nerr)
		}
		return
	}
	if aerr := msg.Ack(); aerr != nil {
		c.log.Error("ack failed", "event_id", e.ID, "err", aerr)
	}
}

// deadLetter handles a handler failure on the last delivery (NumDelivered >=
// MaxDeliver), replacing v1's silent skip: record the dead letter through the sink,
// then Term() so the server stops tracking it, then log ERROR with ids only
// (event_id, subject, durable, err_class, stream_seq — no payload, no error text). A
// sink error is logged and the message is still terminated: the server won't
// redeliver past MaxDeliver anyway.
func (c *NatsConsumer) deadLetter(msg jetstream.Msg, e Event, sub subscription, herr error, reason string) {
	dl := DeadLetter{
		EventID:   e.ID,
		Subject:   e.Subject,
		Durable:   sub.durable,
		ErrClass:  ErrClass(herr),
		StreamSeq: streamSeq(msg),
		At:        time.Now().UTC(),
	}
	ids := []any{
		"event_id", dl.EventID, "subject", dl.Subject, "durable", dl.Durable,
		"err_class", dl.ErrClass, "stream_seq", dl.StreamSeq,
	}
	if sub.sink != nil {
		sctx, cancel := context.WithTimeout(context.Background(), deadLetterTimeout)
		serr := sub.sink.RecordDeadLetter(sctx, dl)
		cancel()
		if serr != nil {
			c.log.Error("dead letter not recorded; terminating anyway", ids...)
		}
	}
	if terr := msg.Term(); terr != nil {
		c.log.Error("term failed", ids...)
	}
	c.log.Error(reason, ids...)
}

// deliveryCount returns how many times this message has been delivered (1 on first
// delivery), or 1 when the metadata can't be read.
func deliveryCount(msg jetstream.Msg) uint64 {
	if md, err := msg.Metadata(); err == nil && md.NumDelivered > 0 {
		return md.NumDelivered
	}
	return 1
}

// streamSeq returns the message's stream sequence, or 0 when the metadata can't be
// read.
func streamSeq(msg jetstream.Msg) uint64 {
	if md, err := msg.Metadata(); err == nil {
		return md.Sequence.Stream
	}
	return 0
}

// Conn exposes the underlying connection (the NATS-auth integration test drives
// denied operations through the same client options).
func (c *NatsConsumer) Conn() *nats.Conn { return c.nc }

// Close drains and closes the NATS connection (stops all subscriptions).
func (c *NatsConsumer) Close() {
	if c.nc != nil {
		c.nc.Close()
	}
}

// consumeSub adapts a jetstream.ConsumeContext to Subscription.
type consumeSub struct{ cctx jetstream.ConsumeContext }

func (s consumeSub) Stop() {
	if s.cctx != nil {
		s.cctx.Stop()
	}
}

// eventID resolves an event's id: the envelope's event_id if the payload parses,
// else the Nats-Msg-Id header the publisher stamped (WithMsgID), else "".
func eventID(msg jetstream.Msg) string {
	var env struct {
		EventID string `json:"event_id"`
	}
	if err := json.Unmarshal(msg.Data(), &env); err == nil && env.EventID != "" {
		return env.EventID
	}
	if h := msg.Headers(); h != nil {
		if id := h.Get(nats.MsgIdHdr); id != "" {
			return id
		}
	}
	return ""
}

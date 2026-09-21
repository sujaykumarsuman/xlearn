package events

import (
	"context"
	"encoding/json"
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
// naking (which would waste redelivery budget on every rolling deploy).
const handleTimeout = 25 * time.Second

// maxDeliver caps redelivery attempts for a persistently-failing message so a
// poison event can't be retried forever. Combined with redeliveryBackoff it spans
// hours (not milliseconds), so a transient DB/broker outage is a bounded, recoverable
// retry — never an instant budget burn — while a genuinely un-processable event
// eventually stops (it stays on the stream for inspection/replay).
const maxDeliver = 100

// redeliveryBackoff is the escalating delay between redeliveries. A handler error
// naks WITH this delay (NakWithDelay), and it is also the consumer's server-side
// BackOff for the ack-timeout path. Delays beyond the slice repeat the last entry,
// so maxDeliver=100 attempts span ~8 hours rather than being exhausted in under a
// second by a hot nak loop (which would silently drop the event).
var redeliveryBackoff = []time.Duration{
	1 * time.Second, 5 * time.Second, 15 * time.Second, 30 * time.Second,
	1 * time.Minute, 2 * time.Minute, 5 * time.Minute,
}

// backoffFor picks the redelivery delay for the n-th delivery (n starts at 1),
// clamping past the end of the schedule to its last (longest) entry.
func backoffFor(numDelivered uint64) time.Duration {
	if numDelivered == 0 {
		numDelivered = 1
	}
	i := int(numDelivered - 1)
	if i >= len(redeliveryBackoff) {
		i = len(redeliveryBackoff) - 1
	}
	return redeliveryBackoff[i]
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
	stream string
}

// NewNatsConsumer connects to NATS and prepares durable pull subscriptions on
// stream. The connection auto-reconnects, so a brief broker outage is transparent
// — a redelivered message is deduped by the handler's inbox.
func NewNatsConsumer(ctx context.Context, url, stream string, log *slog.Logger) (*NatsConsumer, error) {
	nc, err := nats.Connect(url,
		nats.Name("xlearn-"+stream+"-consumer"),
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
	return &NatsConsumer{nc: nc, js: js, log: log, stream: stream}, nil
}

// Subscribe creates (idempotently) a durable pull consumer named durable on the
// configured stream, filtered to filterSubject, and delivers messages to h until
// Stop is called or ctx is cancelled. The durable name + explicit ack make this a
// resumable, at-least-once subscription: on restart it continues from its last
// acked offset, so events that arrived while the service was down are still
// processed (offline catch-up).
//
// The consumer is created lazily with retry because the producer that owns the
// stream may still be provisioning it on a fresh cluster; Subscribe blocks (with
// backoff, honouring ctx) until the stream exists.
func (c *NatsConsumer) Subscribe(ctx context.Context, durable, filterSubject string, h Handler, opts ...SubscribeOption) (Subscription, error) {
	var sc SubscribeConfig
	for _, opt := range opts {
		opt(&sc)
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
		MaxDeliver:    maxDeliver,
		BackOff:       redeliveryBackoff,
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
		c.dispatch(msg, h)
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
// transient outage never burns the redelivery budget in a hot loop.
func (c *NatsConsumer) dispatch(msg jetstream.Msg, h Handler) {
	ctx, cancel := context.WithTimeout(context.Background(), handleTimeout)
	defer cancel()

	e := Event{ID: eventID(msg), Subject: msg.Subject(), Data: msg.Data()}
	if err := h.Handle(ctx, e); err != nil {
		delay := backoffFor(deliveryCount(msg))
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

// deliveryCount returns how many times this message has been delivered (1 on first
// delivery), or 1 when the metadata can't be read.
func deliveryCount(msg jetstream.Msg) uint64 {
	if md, err := msg.Metadata(); err == nil && md.NumDelivered > 0 {
		return md.NumDelivered
	}
	return 1
}

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

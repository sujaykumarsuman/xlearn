package events

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

// DeadLetter is the record of an event whose handler failed on its LAST delivery
// (NumDelivered >= MaxDeliver). It holds ids only — no payload, no error text — so it
// is erase-safe (ADR-0035 §1.2). Services persist it as a <svc>.event_dead_letter row,
// read on demand (D34: no alerting; anything needed after the next rollout is a row).
type DeadLetter struct {
	EventID, Subject, Durable, ErrClass string
	StreamSeq                           uint64
	At                                  time.Time
}

// DeadLetterSink records a dead letter. The consuming service's store implements it
// (an idempotent insert: ON CONFLICT (durable, event_id) DO NOTHING).
type DeadLetterSink interface {
	RecordDeadLetter(ctx context.Context, dl DeadLetter) error
}

// WithDeadLetter makes the subscription record a dead letter through sink on the last
// failing delivery, before it terminates the message. Without a sink the hook still
// terminates and logs ERROR.
func WithDeadLetter(sink DeadLetterSink) SubscribeOption {
	return func(c *SubscribeConfig) { c.DeadLetter = sink }
}

// WithMaxDeliver overrides the redelivery budget (default maxDeliver = 100, ~8 h) and
// shortens the backoff to testBackoff, so a test reaches the dead-letter path in
// milliseconds. NATS rejects a BackOff list longer than MaxDeliver, so the backoff is a
// single short entry. Test-only: services never pass it.
func WithMaxDeliver(n int) SubscribeOption {
	return func(c *SubscribeConfig) { c.MaxDeliver = n }
}

// testBackoff is the redelivery delay under WithMaxDeliver. It is also the effective
// AckWait of the first delivery (the server uses BackOff[0]), so it stays well above a
// fast test handler's run time.
const testBackoff = 250 * time.Millisecond

// deadLetterTimeout bounds the sink insert. It runs on a fresh context: the handler's
// own context may already have expired (a timeout is one of the failure classes).
const deadLetterTimeout = 5 * time.Second

// Error classes recorded in DeadLetter.ErrClass.
const (
	ErrClassTimeout = "timeout"
	ErrClassDB      = "db"
	ErrClassDecode  = "decode"
	ErrClassOther   = "other"
)

// ErrClass maps a handler error to a coarse, data-free class for the dead-letter row:
// timeout (a deadline or network timeout), db (a Postgres error or connect failure),
// decode (malformed JSON) or other.
func ErrClass(err error) string {
	if err == nil {
		return ErrClassOther
	}
	var netErr net.Error
	if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &netErr) && netErr.Timeout()) {
		return ErrClassTimeout
	}
	var pgErr *pgconn.PgError
	var connErr *pgconn.ConnectError
	if errors.As(err, &pgErr) || errors.As(err, &connErr) {
		return ErrClassDB
	}
	var syntaxErr *json.SyntaxError
	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &syntaxErr) || errors.As(err, &typeErr) {
		return ErrClassDecode
	}
	return ErrClassOther
}

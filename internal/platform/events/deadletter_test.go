package events

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// fakeMsg is a jetstream.Msg recording what dispatch did to it.
type fakeMsg struct {
	subject   string
	data      []byte
	delivered uint64
	streamSeq uint64

	acked, termed, naked bool
	nakDelay             time.Duration
}

func (m *fakeMsg) Metadata() (*jetstream.MsgMetadata, error) {
	return &jetstream.MsgMetadata{
		NumDelivered: m.delivered,
		Sequence:     jetstream.SequencePair{Stream: m.streamSeq, Consumer: m.delivered},
	}, nil
}
func (m *fakeMsg) Data() []byte                       { return m.data }
func (m *fakeMsg) Headers() nats.Header               { return nil }
func (m *fakeMsg) Subject() string                    { return m.subject }
func (m *fakeMsg) Reply() string                      { return "" }
func (m *fakeMsg) Ack() error                         { m.acked = true; return nil }
func (m *fakeMsg) DoubleAck(context.Context) error    { m.acked = true; return nil }
func (m *fakeMsg) Nak() error                         { m.naked = true; return nil }
func (m *fakeMsg) NakWithDelay(d time.Duration) error { m.naked, m.nakDelay = true, d; return nil }
func (m *fakeMsg) InProgress() error                  { return nil }
func (m *fakeMsg) Term() error                        { m.termed = true; return nil }
func (m *fakeMsg) TermWithReason(string) error        { m.termed = true; return nil }

var _ jetstream.Msg = (*fakeMsg)(nil)

type handlerFunc func(context.Context, Event) error

func (f handlerFunc) Handle(ctx context.Context, e Event) error { return f(ctx, e) }

type fakeSink struct {
	mu   sync.Mutex
	got  []DeadLetter
	fail error
}

func (s *fakeSink) RecordDeadLetter(_ context.Context, dl DeadLetter) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.got = append(s.got, dl)
	return s.fail
}

func (s *fakeSink) letters() []DeadLetter {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]DeadLetter(nil), s.got...)
}

// lockedBuffer is a goroutine-safe log sink (NATS async callbacks log from their own
// goroutine while the test reads).
type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func bufLogger() (*slog.Logger, *lockedBuffer) {
	buf := &lockedBuffer{}
	return slog.New(slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug})), buf
}

func envelopeMsg(subject, eventID string, delivered uint64) *fakeMsg {
	data, _ := json.Marshal(map[string]any{"event_id": eventID, "subject": subject, "account_id": "secret-account"})
	return &fakeMsg{subject: subject, data: data, delivered: delivered, streamSeq: 42}
}

// The last failing delivery records the dead letter, terminates (no nak) and logs
// ERROR with ids only.
func TestDispatchDeadLettersOnLastDelivery(t *testing.T) {
	log, buf := bufLogger()
	c := &NatsConsumer{log: log, stream: StreamPractice}
	sink := &fakeSink{}
	sub := subscription{durable: "review", maxDeliver: maxDeliver, backoff: redeliveryBackoff, sink: sink}
	msg := envelopeMsg("xlearn.practice.problem_solved", "evt-9", maxDeliver)

	c.dispatch(msg, handlerFunc(func(context.Context, Event) error {
		return fmt.Errorf("apply: %w", &pgconn.PgError{Code: "23505", Message: "duplicate key value (account_id)=(secret-account)"})
	}), sub)

	if !msg.termed || msg.naked || msg.acked {
		t.Fatalf("termed=%v naked=%v acked=%v, want term only", msg.termed, msg.naked, msg.acked)
	}
	got := sink.letters()
	if len(got) != 1 {
		t.Fatalf("%d dead letters, want 1", len(got))
	}
	dl := got[0]
	if dl.EventID != "evt-9" || dl.Subject != "xlearn.practice.problem_solved" || dl.Durable != "review" ||
		dl.ErrClass != ErrClassDB || dl.StreamSeq != 42 || dl.At.IsZero() {
		t.Fatalf("dead letter = %+v", dl)
	}
	out := buf.String()
	if !strings.Contains(out, `"level":"ERROR"`) || !strings.Contains(out, "event dead-lettered after max deliveries") {
		t.Fatalf("no ERROR dead-letter log: %s", out)
	}
	for _, leak := range []string{"secret-account", "duplicate key"} {
		// Only the WARN line of earlier deliveries carries err text; the terminal
		// ERROR line must carry ids only.
		for _, line := range strings.Split(out, "\n") {
			if strings.Contains(line, "dead-lettered") && strings.Contains(line, leak) {
				t.Errorf("dead-letter log leaks %q: %s", leak, line)
			}
		}
	}
}

// An earlier failing delivery naks with the escalating backoff and records nothing.
func TestDispatchNaksBeforeLastDelivery(t *testing.T) {
	log, _ := bufLogger()
	c := &NatsConsumer{log: log, stream: StreamPractice}
	sink := &fakeSink{}
	sub := subscription{durable: "review", maxDeliver: maxDeliver, backoff: redeliveryBackoff, sink: sink}
	msg := envelopeMsg("xlearn.practice.problem_solved", "evt-1", 2)

	c.dispatch(msg, handlerFunc(func(context.Context, Event) error { return errors.New("transient") }), sub)

	if !msg.naked || msg.termed || msg.acked {
		t.Fatalf("naked=%v termed=%v acked=%v, want nak only", msg.naked, msg.termed, msg.acked)
	}
	if msg.nakDelay != redeliveryBackoff[1] {
		t.Fatalf("nak delay %s, want %s", msg.nakDelay, redeliveryBackoff[1])
	}
	if n := len(sink.letters()); n != 0 {
		t.Fatalf("%d dead letters before the last delivery", n)
	}
}

// A sink error still ends in Term() (the server won't redeliver past MaxDeliver).
func TestDispatchSinkErrorStillTerms(t *testing.T) {
	log, buf := bufLogger()
	c := &NatsConsumer{log: log, stream: StreamReview}
	sink := &fakeSink{fail: errors.New("db down")}
	sub := subscription{durable: "notifications", maxDeliver: 3, backoff: []time.Duration{time.Millisecond}, sink: sink}
	msg := envelopeMsg("xlearn.review.revision_due", "evt-2", 3)

	c.dispatch(msg, handlerFunc(func(context.Context, Event) error { return context.DeadlineExceeded }), sub)

	if !msg.termed || msg.naked {
		t.Fatalf("termed=%v naked=%v, want term", msg.termed, msg.naked)
	}
	if got := sink.letters(); len(got) != 1 || got[0].ErrClass != ErrClassTimeout {
		t.Fatalf("sink got %+v", got)
	}
	if !strings.Contains(buf.String(), "dead letter not recorded; terminating anyway") {
		t.Fatalf("sink failure not logged: %s", buf.String())
	}
}

// With no sink the hook still terminates and logs ERROR; success still acks.
func TestDispatchNoSinkAndSuccess(t *testing.T) {
	log, buf := bufLogger()
	c := &NatsConsumer{log: log, stream: StreamPractice}
	sub := subscription{durable: "assessment", maxDeliver: maxDeliver, backoff: redeliveryBackoff}

	last := envelopeMsg("xlearn.practice.attempt_logged", "evt-3", maxDeliver+5)
	c.dispatch(last, handlerFunc(func(context.Context, Event) error { return errors.New("poison") }), sub)
	if !last.termed || !strings.Contains(buf.String(), "event dead-lettered") {
		t.Fatalf("no-sink last delivery: termed=%v log=%s", last.termed, buf.String())
	}

	ok := envelopeMsg("xlearn.practice.attempt_logged", "evt-4", maxDeliver)
	c.dispatch(ok, handlerFunc(func(context.Context, Event) error { return nil }), sub)
	if !ok.acked || ok.termed || ok.naked {
		t.Fatalf("success: acked=%v termed=%v naked=%v", ok.acked, ok.termed, ok.naked)
	}
}

func TestErrClass(t *testing.T) {
	var jsonErr error
	var v struct{ N int }
	jsonErr = json.Unmarshal([]byte(`{"N":"x"}`), &v)
	cases := []struct {
		err  error
		want string
	}{
		{nil, ErrClassOther},
		{context.DeadlineExceeded, ErrClassTimeout},
		{fmt.Errorf("wrap: %w", context.DeadlineExceeded), ErrClassTimeout},
		{fmt.Errorf("q: %w", &pgconn.PgError{Code: "40001"}), ErrClassDB},
		{fmt.Errorf("decode: %w", jsonErr), ErrClassDecode},
		{json.Unmarshal([]byte(`{`), &v), ErrClassDecode},
		{errors.New("boom"), ErrClassOther},
	}
	for _, c := range cases {
		if got := ErrClass(c.err); got != c.want {
			t.Errorf("ErrClass(%v) = %s, want %s", c.err, got, c.want)
		}
	}
}

func TestWithMaxDeliverShortensBackoff(t *testing.T) {
	var sc SubscribeConfig
	WithMaxDeliver(3)(&sc)
	WithDeadLetter(&fakeSink{})(&sc)
	if sc.MaxDeliver != 3 || sc.DeadLetter == nil {
		t.Fatalf("config = %+v", sc)
	}
	if testBackoff >= redeliveryBackoff[0] {
		t.Fatalf("testBackoff %s must be shorter than the production first backoff", testBackoff)
	}
}

// An ErrInvalidEnvelope (m1-02: a v2 course-scoped event without path_slug, or
// undecodable JSON) dead-letters on its FIRST delivery: no redelivery can fix it.
func TestDispatchDeadLettersInvalidEnvelopeAtOnce(t *testing.T) {
	log, buf := bufLogger()
	c := &NatsConsumer{log: log, stream: StreamPractice}
	sink := &fakeSink{}
	sub := subscription{durable: "review", maxDeliver: maxDeliver, backoff: redeliveryBackoff, sink: sink}
	msg := envelopeMsg("xlearn.practice.problem_solved", "evt-v2", 1)

	c.dispatch(msg, handlerFunc(func(context.Context, Event) error {
		_, err := DecodeEnvelope([]byte(`{"event_id":"evt-v2","subject":"xlearn.practice.problem_solved","version":2,"data":{}}`))
		return fmt.Errorf("review consumer: %w", err)
	}), sub)

	if !msg.termed || msg.naked || msg.acked {
		t.Fatalf("termed=%v naked=%v acked=%v, want term on the first delivery", msg.termed, msg.naked, msg.acked)
	}
	got := sink.letters()
	if len(got) != 1 || got[0].EventID != "evt-v2" || got[0].ErrClass != ErrClassDecode || got[0].Durable != "review" {
		t.Fatalf("dead letters = %+v", got)
	}
	if !strings.Contains(buf.String(), "event dead-lettered: invalid envelope") {
		t.Fatalf("no ERROR dead-letter log: %s", buf.String())
	}
}

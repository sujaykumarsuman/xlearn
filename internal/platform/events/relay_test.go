package events

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

type memOutbox struct {
	rows []Event
	sent map[string]bool
}

func (o *memOutbox) ListUnsent(_ context.Context, limit int32) ([]Event, error) {
	var out []Event
	for _, e := range o.rows {
		if !o.sent[e.ID] && int32(len(out)) < limit {
			out = append(out, e)
		}
	}
	return out, nil
}

func (o *memOutbox) MarkSent(_ context.Context, id string) error {
	o.sent[id] = true
	return nil
}

type memPublisher struct{ got []string }

func (p *memPublisher) Publish(_ context.Context, e Event) error {
	p.got = append(p.got, e.ID)
	return nil
}

func TestCheckEnvelope(t *testing.T) {
	if err := CheckEnvelope(make([]byte, MaxEnvelopeBytes)); err != nil {
		t.Fatalf("exactly the cap must pass: %v", err)
	}
	if err := CheckEnvelope(make([]byte, MaxEnvelopeBytes+1)); !errors.Is(err, ErrEnvelopeTooLarge) {
		t.Fatalf("over the cap: %v, want ErrEnvelopeTooLarge", err)
	}
	if MaxEnvelopeBytes != 16*1024 {
		t.Fatalf("MaxEnvelopeBytes = %d, L20 says 16 KiB", MaxEnvelopeBytes)
	}
}

// An oversize row is skipped (left unsent, logged ERROR once per id) and the rest of
// the batch still drains — the relay never stalls on it.
func TestRelaySkipsOversizeWithoutStalling(t *testing.T) {
	src := &memOutbox{sent: map[string]bool{}, rows: []Event{
		{ID: "a", Subject: "xlearn.practice.problem_solved", Data: []byte(`{}`)},
		{ID: "big", Subject: "xlearn.practice.attempt_logged", Data: bytes.Repeat([]byte("x"), MaxEnvelopeBytes+1)},
		{ID: "b", Subject: "xlearn.practice.problem_solved", Data: []byte(`{}`)},
	}}
	pub := &memPublisher{}
	log, buf := bufLogger()
	r := NewRelay(src, pub, log)

	r.drain(context.Background())
	r.drain(context.Background())

	if strings.Join(pub.got, ",") != "a,b" {
		t.Fatalf("published %v, want a,b", pub.got)
	}
	if src.sent["big"] || !src.sent["a"] || !src.sent["b"] {
		t.Fatalf("sent = %v; the oversize row must stay unsent", src.sent)
	}
	if n := strings.Count(buf.String(), "envelope over the cap"); n != 1 {
		t.Fatalf("oversize logged %d times over two ticks, want once:\n%s", n, buf.String())
	}
	if !strings.Contains(buf.String(), `"level":"ERROR"`) || !strings.Contains(buf.String(), `"event_id":"big"`) {
		t.Fatalf("oversize log lacks ERROR / event id: %s", buf.String())
	}
}

package assessment

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"testing"

	"github.com/sujaykumarsuman/xlearn/internal/assessment/store"
	"github.com/sujaykumarsuman/xlearn/internal/platform/events"
)

// assessment's stream and durable constants resolve in topology.go with the filters
// and owner the code uses.
func TestTopologyDeclaresAssessment(t *testing.T) {
	s := events.MustStream(StreamAssessment)
	if s.Owner != ServiceName || !slices.Equal(StreamSubjects, s.Subjects) {
		t.Fatalf("XLEARN_ASSESSMENT owner %q subjects %v; code says %q %v", s.Owner, s.Subjects, ServiceName, StreamSubjects)
	}
	for _, c := range []struct{ stream, filter string }{
		{StreamPractice, PracticeSubjectFilter},
		{StreamReview, ReviewSubjectFilter},
	} {
		d, ok := events.LookupDurable(c.stream, DurableName)
		if !ok || d.Filter != c.filter || d.Service != ServiceName {
			t.Errorf("durable %s/%s: declared=%v filter=%q service=%q; code filter %q", c.stream, DurableName, ok, d.Filter, d.Service, c.filter)
		}
	}
}

// Every subject constant assessment produces is in its stream's Emits; every one it
// consumes is emitted by its producer.
func TestAssessmentSubjectsInRegistry(t *testing.T) {
	if !slices.Contains(events.MustStream(StreamAssessment).Emits, store.SubjectMockCompleted) {
		t.Errorf("XLEARN_ASSESSMENT Emits lacks %s", store.SubjectMockCompleted)
	}
	for stream, subjects := range map[string][]string{
		StreamPractice: {store.SubjectProblemSolved, store.SubjectAttemptLogged, store.SubjectSolutionRevealedEarly},
		StreamReview:   {store.SubjectRevisionScheduled, store.SubjectRevisionDue, store.SubjectMistakeOpened, store.SubjectMistakeClosed},
	} {
		emits := events.MustStream(stream).Emits
		for _, s := range subjects {
			if !slices.Contains(emits, s) {
				t.Errorf("assessment consumes %s, which %s doesn't emit (%v)", s, stream, emits)
			}
		}
	}
}

// Both projection durables: every Handles subject reaches ApplyProjection (never the
// default branch); an unlisted subject is acked with an ERROR, never silently.
func TestProjectionConsumerSubjectRegistry(t *testing.T) {
	for _, stream := range []string{StreamPractice, StreamReview} {
		d, _ := events.LookupDurable(stream, DurableName)
		var applied []string
		fs := &fakeStore{applyProjection: func(_ context.Context, ev store.ProjectionEvent) (bool, error) {
			applied = append(applied, ev.Subject)
			return true, nil
		}}
		var logs bytes.Buffer
		h := &projectionHandler{store: fs, log: slog.New(slog.NewJSONHandler(&logs, nil))}
		for _, s := range d.Handles {
			env := fmt.Sprintf(`{"event_id":"e-%s","account_id":"acct-1","occurred_at":"2026-09-20T10:00:00Z","data":{"problem_id":"3"}}`, s)
			if err := h.Handle(context.Background(), events.Event{ID: "e-" + s, Subject: s, Data: []byte(env)}); err != nil {
				t.Fatalf("%s: %v", s, err)
			}
		}
		if !slices.Equal(applied, d.Handles) || strings.Contains(logs.String(), "unlisted subject") {
			t.Fatalf("%s: applied %v, want every Handles subject %v (logs %s)", stream, applied, d.Handles, logs.String())
		}

		unlisted := strings.TrimSuffix(d.Filter, "*") + "brand_new"
		env := `{"event_id":"e-new","account_id":"acct-1","data":{}}`
		if err := h.Handle(context.Background(), events.Event{ID: "e-new", Subject: unlisted, Data: []byte(env)}); err != nil {
			t.Fatalf("unlisted subject must ack: %v", err)
		}
		if len(applied) != len(d.Handles) {
			t.Fatalf("%s: unlisted subject reached ApplyProjection", stream)
		}
		if !strings.Contains(logs.String(), `"level":"ERROR"`) || !strings.Contains(logs.String(), unlisted) {
			t.Fatalf("%s: unlisted subject must log ERROR: %s", stream, logs.String())
		}
	}
}

// Both projection subscriptions pass the dead-letter sink.
func TestProjectionSubscriptionsPassDeadLetterSink(t *testing.T) {
	fs := &fakeStore{}
	svc := NewService(fs, fakeVerifier{}, testLogger())
	rec := &recordingConsumer{}
	for _, f := range []string{PracticeSubjectFilter, ReviewSubjectFilter} {
		if _, err := svc.StartProjectionConsumer(context.Background(), rec, f); err != nil {
			t.Fatal(err)
		}
	}
	if len(rec.configs) != 2 {
		t.Fatalf("%d subscriptions", len(rec.configs))
	}
	for i, sc := range rec.configs {
		if sc.DeadLetter != fs || !sc.DeliverNew {
			t.Errorf("subscription %d: dead-letter sink %v, deliverNew %v", i, sc.DeadLetter, sc.DeliverNew)
		}
	}
}

type recordingConsumer struct{ configs []events.SubscribeConfig }

func (r *recordingConsumer) Subscribe(_ context.Context, _, _ string, _ events.Handler, opts ...events.SubscribeOption) (events.Subscription, error) {
	var sc events.SubscribeConfig
	for _, o := range opts {
		o(&sc)
	}
	r.configs = append(r.configs, sc)
	return nopSub{}, nil
}

type nopSub struct{}

func (nopSub) Stop() {}

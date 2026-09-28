package assessment

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/sujaykumarsuman/xlearn/internal/assessment/store"
	"github.com/sujaykumarsuman/xlearn/internal/platform/events"
)

// The envelope fixtures live beside the decoder (internal/platform/events).
func envelopeFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "platform", "events", "testdata", "envelope", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return b
}

// projectionTwins are the fixture stems (and subjects) the projection durables handle.
var projectionTwins = map[string]string{
	"problem_solved":          "xlearn.practice.problem_solved",
	"attempt_logged":          "xlearn.practice.attempt_logged",
	"solution_revealed_early": "xlearn.practice.solution_revealed_early",
	"revision_scheduled":      "xlearn.review.revision_scheduled",
	"revision_due":            "xlearn.review.revision_due",
	"mistake_opened":          "xlearn.review.mistake_opened",
	"mistake_closed":          "xlearn.review.mistake_closed",
}

func handleFixture(t *testing.T, subject, name string) (store.ProjectionEvent, bool, error) {
	t.Helper()
	var got store.ProjectionEvent
	called := false
	fs := &fakeStore{applyProjection: func(_ context.Context, ev store.ProjectionEvent) (bool, error) {
		got, called = ev, true
		return true, nil
	}}
	h := &projectionHandler{store: fs, log: testLogger()}
	err := h.Handle(context.Background(), events.Event{ID: "nats-id", Subject: subject, Data: envelopeFixture(t, name)})
	return got, called, err
}

// m1-02 (consumers before producers): the projection consumer accepts the v2 envelope
// with the same result as its v1 twin — the same ProjectionEvent, course "dsa" — and
// dead-letters a v2 event without path_slug.
func TestProjectionHandlerV1V2Twins(t *testing.T) {
	for stem, subject := range projectionTwins {
		t.Run(stem, func(t *testing.T) {
			v1, ok1, err := handleFixture(t, subject, stem+".v1.json")
			if err != nil || !ok1 {
				t.Fatalf("v1: applied=%v err=%v", ok1, err)
			}
			v2, ok2, err := handleFixture(t, subject, stem+".v2.json")
			if err != nil || !ok2 {
				t.Fatalf("v2: applied=%v err=%v", ok2, err)
			}
			if !reflect.DeepEqual(v1, v2) {
				t.Fatalf("v1 %+v\nv2 %+v", v1, v2)
			}
			if v1.PathSlug != "dsa" {
				t.Fatalf("path slug = %q, want dsa", v1.PathSlug)
			}

			_, applied, err := handleFixture(t, subject, stem+".v2-nopath.json")
			if !errors.Is(err, events.ErrInvalidEnvelope) || applied {
				t.Fatalf("v2 without path_slug: applied=%v err=%v, want ErrInvalidEnvelope (dead-letter)", applied, err)
			}
		})
	}
}

// An unknown future version with extra fields is still processed.
func TestProjectionHandlerUnknownVersion(t *testing.T) {
	v3, applied, err := handleFixture(t, "xlearn.practice.problem_solved", "problem_solved.v3-extra.json")
	if err != nil || !applied {
		t.Fatalf("v3: applied=%v err=%v", applied, err)
	}
	v1, _, _ := handleFixture(t, "xlearn.practice.problem_solved", "problem_solved.v1.json")
	if !reflect.DeepEqual(v1, v3) {
		t.Fatalf("v1 %+v\nv3 %+v", v1, v3)
	}
}

package practice

import (
	"slices"
	"testing"

	"github.com/sujaykumarsuman/xlearn/internal/platform/events"
	"github.com/sujaykumarsuman/xlearn/internal/practice/store"
)

// practice's stream resolves in topology.go with practice as its owner, and every
// subject constant practice produces is in the stream's Emits (the producer side of
// the ADR-0035 §1.1 subject registry) — and vice versa, so the registry can't list a
// subject nothing publishes.
func TestPracticeTopologyAndSubjects(t *testing.T) {
	s := events.MustStream(StreamPractice)
	if s.Owner != ServiceName || !slices.Equal(StreamSubjects, s.Subjects) {
		t.Fatalf("XLEARN_PRACTICE owner %q subjects %v; code says %q %v", s.Owner, s.Subjects, ServiceName, StreamSubjects)
	}
	produced := []string{store.SubjectProblemSolved, store.SubjectAttemptLogged, store.SubjectSolutionRevealedEarly}
	for _, p := range produced {
		if !slices.Contains(s.Emits, p) {
			t.Errorf("practice emits %s, but XLEARN_PRACTICE's Emits %v lacks it", p, s.Emits)
		}
	}
	for _, e := range s.Emits {
		if !slices.Contains(produced, e) {
			t.Errorf("XLEARN_PRACTICE's Emits lists %s, which practice has no Subject constant for", e)
		}
	}
}

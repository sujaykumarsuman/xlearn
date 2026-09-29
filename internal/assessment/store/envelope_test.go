package store

import (
	"reflect"
	"testing"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/platform/events"
)

// m1-03 (M1b): mock_completed is a course-scoped v2 envelope, and its payload is
// append-only — v1's mock_id / total_35 / rubric stay beside the new rubric_id / total /
// max_total / scored_by, with total_35 kept only for a 35-point rubric.
func TestMockCompletedEnvelope(t *testing.T) {
	if eventVersion != events.EnvelopeV2 || !events.CourseScoped(SubjectMockCompleted) {
		t.Fatalf("eventVersion %d, course-scoped %v; want a course-scoped v2 subject", eventVersion, events.CourseScoped(SubjectMockCompleted))
	}
	if _, err := events.NewEnvelope(eventVersion, "evt-1", SubjectMockCompleted, "acct-1", "", time.Now(), map[string]any{}); err == nil {
		t.Fatalf("a v2 mock_completed without path_slug was accepted")
	}

	rubric := map[string]int{"communication": 4}
	got := mockCompletedData("m-1", RubricID, 28, MaxTotal, rubric)
	want := map[string]any{
		"mock_id": "m-1", "rubric": rubric, "total_35": 28,
		"rubric_id": RubricID, "total": 28, "max_total": MaxTotal, "scored_by": ScoredBySelf,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("35-point payload = %v\nwant %v", got, want)
	}
	// Another rubric width: no total_35 (it would not be out of 35).
	other := mockCompletedData("m-2", "other@1", 40, 50, rubric)
	if _, ok := other["total_35"]; ok || other["total"] != 40 || other["max_total"] != 50 || other["rubric_id"] != "other@1" {
		t.Fatalf("50-point payload = %v", other)
	}
}

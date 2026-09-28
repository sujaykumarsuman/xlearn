package coach

import (
	"strings"
	"testing"

	"github.com/sujaykumarsuman/xlearn/internal/course/coursetest"
)

// TestManifestGoldenMirror is the coach side of the DSA manifest golden test (sprint
// m1-01; side one is internal/course/golden_test.go): the manifest's coach.persona is
// v1's course-specific opening (prompt.go:56-57) verbatim, for every mode. m1-07 moves
// the persona into the prompt from the manifest; until then the two must not drift.
func TestManifestGoldenMirror(t *testing.T) {
	persona := coursetest.DSA(t).Coach.Persona
	for _, mode := range []string{ModeAttempt, ModeReview, ModeGeneral} {
		if got := systemPrompt(mode, pageContext{}); !strings.HasPrefix(got, persona+"\n\n") {
			t.Errorf("mode %s: system prompt does not open with the manifest persona.\n got: %.260q\nwant: %q", mode, got, persona)
		}
	}
}

package coach

import (
	"strings"
	"testing"

	"github.com/sujaykumarsuman/xlearn/internal/course"
	"github.com/sujaykumarsuman/xlearn/internal/course/coursetest"
)

// TestManifestGoldenMirror is the coach side of the DSA manifest golden test (sprint
// m1-01; side one is internal/course/golden_test.go, which pins the manifest's
// coach.persona to v1's course-specific opening, prompt.go:56-57, verbatim). Since m1-07
// the prompt TAKES the persona from the manifest: for every mode, a chat on a DSA page
// (X-Coach-Course: dsa) and on an account-wide page (X-Coach-Course: "") gets the frame
// followed by exactly that persona — so the DSA prompt keeps v1's voice word for word.
func TestManifestGoldenMirror(t *testing.T) {
	persona := coursetest.DSA(t).Coach.Persona
	reg := coursetest.Registry(t)
	for _, slug := range []string{course.DefaultSlug, ""} {
		for _, mode := range []string{ModeAttempt, ModeReview, ModeGeneral} {
			got := systemPrompt(coursePersona(reg, slug), mode, pageContext{})
			if want := promptFrame + "\n\n" + persona + "\n\n"; !strings.HasPrefix(got, want) {
				t.Errorf("slug %q mode %s: system prompt does not open with the frame then the manifest persona.\n got: %.400q\nwant: %q", slug, mode, got, want)
			}
		}
	}
}

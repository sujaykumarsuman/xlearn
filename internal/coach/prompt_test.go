package coach

import (
	"strings"
	"testing"

	"github.com/sujaykumarsuman/xlearn/internal/coach/store"
)

func TestNormalizeModeSafeDefaults(t *testing.T) {
	// A problem context with an unknown/missing mode falls back to the SPOILER-FREE
	// attempt mode — never review (a client must not be able to unlock reviewer mode by
	// omitting or corrupting the gate).
	if got := normalizeMode("", pageContext{ProblemID: "16"}); got != ModeAttempt {
		t.Fatalf("missing mode on a problem = %q, want attempt", got)
	}
	if got := normalizeMode("bogus", pageContext{ProblemID: "16"}); got != ModeAttempt {
		t.Fatalf("bogus mode on a problem = %q, want attempt", got)
	}
	// A non-problem context with no mode is the general tutor.
	if got := normalizeMode("", pageContext{Kind: "dashboard"}); got != ModeGeneral {
		t.Fatalf("missing mode off a problem = %q, want general", got)
	}
	// Valid modes pass through.
	for _, m := range []string{ModeAttempt, ModeReview, ModeGeneral} {
		if got := normalizeMode(m, pageContext{ProblemID: "16"}); got != m {
			t.Fatalf("mode %q did not pass through (got %q)", m, got)
		}
	}
}

func TestSystemPromptAttemptIsSpoilerFree(t *testing.T) {
	p := systemPrompt(ModeAttempt, pageContext{ProblemTitle: "3Sum", Pattern: "Two Pointers", Stage: "attempt"})
	for _, want := range []string{"ACTIVE ATTEMPT", "spoiler-free", "Do NOT reveal the full solution", "3Sum", "Two Pointers"} {
		if !strings.Contains(p, want) {
			t.Fatalf("attempt prompt missing %q:\n%s", want, p)
		}
	}
	// It must NOT invite the model to discuss the optimal solution (that's review mode).
	if strings.Contains(p, "Discuss the optimal solution") {
		t.Fatalf("attempt prompt leaked reviewer guidance:\n%s", p)
	}
}

func TestSystemPromptReviewIsReviewer(t *testing.T) {
	p := systemPrompt(ModeReview, pageContext{ProblemTitle: "3Sum"})
	if !strings.Contains(p, "POST-SOLVE REVIEW") || !strings.Contains(p, "Discuss the optimal solution") {
		t.Fatalf("review prompt missing reviewer guidance:\n%s", p)
	}
}

func TestSystemPromptGeneralTutor(t *testing.T) {
	p := systemPrompt(ModeGeneral, pageContext{Label: "Dashboard"})
	if !strings.Contains(p, "MODE: TUTOR") {
		t.Fatalf("general prompt missing tutor mode:\n%s", p)
	}
}

func TestBuildTurnsAlignsToUserFirst(t *testing.T) {
	// An odd-length alternating history ending on a user turn; a fixed 20-tail begins on
	// an assistant turn, which Anthropic rejects (first message must be user). buildTurns
	// must drop the leading assistant so the window starts on a user turn.
	var history []store.Message
	for i := 0; i < 23; i++ {
		role := store.RoleUser
		if i%2 == 1 {
			role = store.RoleAssistant
		}
		history = append(history, store.Message{Role: role, Content: itoaLocal(i)})
	}
	turns := buildTurns(history)
	if len(turns) == 0 || turns[0].Role != store.RoleUser {
		t.Fatalf("buildTurns did not start on a user turn: %+v", turns)
	}
	if len(turns) > historyLimit {
		t.Fatalf("buildTurns returned %d turns, want <= %d", len(turns), historyLimit)
	}
	// The last turn is preserved (the just-appended user message).
	if turns[len(turns)-1].Content != "22" {
		t.Fatalf("buildTurns dropped the newest turn: last = %q", turns[len(turns)-1].Content)
	}
}

func itoaLocal(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

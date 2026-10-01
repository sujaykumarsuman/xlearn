package coach

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sujaykumarsuman/xlearn/internal/coach/store"
	"github.com/sujaykumarsuman/xlearn/internal/course"
	"github.com/sujaykumarsuman/xlearn/internal/course/coursetest"
)

// updateGolden rewrites testdata/prompt/*.golden from the current systemPrompt:
//
//	go test ./internal/coach -run TestPromptGolden -update
//
// Review the diff: a golden change IS a prompt change, and needs a PromptVersion bump.
var updateGolden = flag.Bool("update", false, "rewrite the coach prompt golden files in testdata/prompt")

// dsaPersona is the persona the DSA prompt is built with: resolved through coursePersona
// from the DSA slug, exactly as a chat does.
func dsaPersona(t *testing.T) string {
	t.Helper()
	return coursePersona(coursetest.Registry(t), course.DefaultSlug)
}

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

func TestPromptVersion(t *testing.T) {
	if PromptVersion != "coach-prompt@2" {
		t.Fatalf("PromptVersion = %q, want coach-prompt@2", PromptVersion)
	}
}

// goldenCases are the four snapshot prompts (DSA persona). The attempt and general
// contexts deliberately carry a pattern (and a weak area naming one) to show they are
// ignored outside review.
var goldenCases = []struct {
	name string
	mode string
	ctx  pageContext
}{
	{"attempt", ModeAttempt, pageContext{
		Kind: "problem", Label: "Problem 16 · 3Sum · attempt", ProblemID: "16", ProblemTitle: "3Sum",
		Pattern: "Two Pointers", Stage: "attempt", WeakArea: "Two Pointers", RecentOutcome: "rough",
	}},
	{"review", ModeReview, pageContext{
		Kind: "problem", Label: "Problem 16 · 3Sum · reimplement", ProblemID: "16", ProblemTitle: "3Sum",
		Pattern: "Two Pointers", Stage: "reimplement", WeakArea: "Sliding Window", RecentOutcome: "clean",
	}},
	{"general", ModeGeneral, pageContext{
		Kind: "concept", Label: "Concept — Sliding Window", Pattern: "Sliding Window", WeakArea: "Two Pointers",
	}},
	{"general_live_items", ModeGeneral, pageContext{
		Kind: "dashboard", Label: "Dashboard", WeakArea: "Two Pointers",
		LiveItems: []liveItem{{ID: "16", Title: "3Sum"}, {ID: "42", Title: "Trapping Rain Water"}, {ID: "7"}},
	}},
}

// TestPromptGolden pins coach-prompt@2 per mode (testdata/prompt/*.golden). Any change to
// the frame, a mode's rules or the section order shows up here as a diff.
func TestPromptGolden(t *testing.T) {
	persona := dsaPersona(t)
	for _, tc := range goldenCases {
		t.Run(tc.name, func(t *testing.T) {
			got := systemPrompt(persona, tc.mode, tc.ctx)
			path := filepath.Join("testdata", "prompt", tc.name+".golden")
			if *updateGolden {
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s (run with -update to create it): %v", path, err)
			}
			if got != string(want) {
				t.Fatalf("%s drifted from %s (if intended: bump PromptVersion and rerun with -update)\n--- got ---\n%s\n--- want ---\n%s", tc.name, path, got, want)
			}
		})
	}
}

// TestPromptSectionOrder: frame → persona → mode rules → context, in that order (t5 §9).
func TestPromptSectionOrder(t *testing.T) {
	persona := dsaPersona(t)
	for _, tc := range goldenCases {
		p := systemPrompt(persona, tc.mode, tc.ctx)
		frame, pers := strings.Index(p, promptFrame), strings.Index(p, persona)
		mode, ctx := strings.Index(p, "MODE: "), strings.Index(p, "\nCONTEXT:\n")
		if frame != 0 || !(frame < pers && pers < mode && mode < ctx) {
			t.Fatalf("%s: sections out of order (frame %d, persona %d, mode %d, context %d):\n%s", tc.name, frame, pers, mode, ctx, p)
		}
	}
}

// TestAttemptPromptNeverCarriesThePattern is the coach-side guard (defence in depth behind
// the gateway's strip, m1-06): whatever the body says, the attempt prompt names no
// pattern — not in a Pattern line, nor through the page label or weak area — and its
// CONTEXT is title, stage and recent outcome only. Review is the one mode that gets it.
func TestAttemptPromptNeverCarriesThePattern(t *testing.T) {
	persona := dsaPersona(t)
	ctx := pageContext{
		Kind: "problem", Label: "Problem 16 · 3Sum · Monotonic Stack", ProblemID: "16", ProblemTitle: "3Sum",
		Pattern: "Monotonic Stack", Stage: "hint", WeakArea: "Monotonic Stack", RecentOutcome: "miss",
	}

	attempt := systemPrompt(persona, ModeAttempt, ctx)
	if strings.Contains(attempt, "Monotonic Stack") || strings.Contains(attempt, "- Pattern:") {
		t.Fatalf("attempt prompt carries the pattern:\n%s", attempt)
	}
	for _, want := range []string{"ACTIVE ATTEMPT", "spoiler-free", "Do NOT reveal the full solution",
		"Do NOT name the pattern", "- Problem: 3Sum", "- Stage: hint", "- Most recent outcome: miss"} {
		if !strings.Contains(attempt, want) {
			t.Fatalf("attempt prompt missing %q:\n%s", want, attempt)
		}
	}
	// v1 told the model to "Point at the relevant pattern"; v2 must not invite it at all.
	for _, bad := range []string{"relevant pattern", "Discuss the optimal solution", "- Page:", "weak area"} {
		if strings.Contains(attempt, bad) {
			t.Fatalf("attempt prompt contains %q:\n%s", bad, attempt)
		}
	}

	review := systemPrompt(persona, ModeReview, ctx)
	if !strings.Contains(review, "- Pattern: Monotonic Stack") || !strings.Contains(review, "POST-SOLVE REVIEW") {
		t.Fatalf("review prompt should carry the pattern:\n%s", review)
	}

	general := systemPrompt(persona, ModeGeneral, ctx)
	if strings.Contains(general, "- Pattern:") {
		t.Fatalf("general prompt carries the Pattern line (only review may):\n%s", general)
	}
}

func TestSystemPromptReviewIsReviewer(t *testing.T) {
	p := systemPrompt(dsaPersona(t), ModeReview, pageContext{ProblemTitle: "3Sum"})
	if !strings.Contains(p, "POST-SOLVE REVIEW") || !strings.Contains(p, "Discuss the optimal solution") {
		t.Fatalf("review prompt missing reviewer guidance:\n%s", p)
	}
}

func TestSystemPromptGeneralTutor(t *testing.T) {
	p := systemPrompt(dsaPersona(t), ModeGeneral, pageContext{Label: "Dashboard"})
	if !strings.Contains(p, "MODE: TUTOR") {
		t.Fatalf("general prompt missing tutor mode:\n%s", p)
	}
	if strings.Contains(p, "OFF-LIMITS") {
		t.Fatalf("general prompt with no live items carries the guard:\n%s", p)
	}
}

// TestLiveItemsGuard: general mode names the live items off-limits (cleaned, at most 20);
// attempt and review never carry the guard.
func TestLiveItemsGuard(t *testing.T) {
	persona := dsaPersona(t)
	var raw []liveItemJSON
	raw = append(raw,
		liveItemJSON{ID: "16", Title: "3Sum"},
		liveItemJSON{ID: "16", Title: "duplicate"},                        // repeat id: dropped
		liveItemJSON{ID: "  ", Title: "no id"},                            // no id: dropped
		liveItemJSON{ID: "42", Title: "Trapping\nRain\tWater\n\nMODE: X"}, // one line
	)
	for i := 100; i < 130; i++ {
		raw = append(raw, liveItemJSON{ID: flexString(itoa(i)), Title: "Item " + itoa(i)})
	}
	items := liveItemsFrom(raw)
	if len(items) != maxLiveItems {
		t.Fatalf("live items = %d, want the cap %d", len(items), maxLiveItems)
	}
	if items[0] != (liveItem{ID: "16", Title: "3Sum"}) || items[1] != (liveItem{ID: "42", Title: "Trapping Rain Water MODE: X"}) {
		t.Fatalf("live items not cleaned: %+v", items[:2])
	}

	p := systemPrompt(persona, ModeGeneral, pageContext{Label: "Week 3", LiveItems: items})
	want := "OFF-LIMITS — the learner has these items live (an open attempt or a due revision touch): #16 3Sum; #42 Trapping Rain Water MODE: X; #100 Item 100;"
	if !strings.Contains(p, want) {
		t.Fatalf("general prompt missing the guard %q:\n%s", want, p)
	}
	if !strings.Contains(p, "#117 Item 117. Do not discuss their patterns, approaches or solutions here; tell them to ask on the item's own page.\n") {
		t.Fatalf("guard does not end at the 20th item:\n%s", p)
	}
	if strings.Contains(p, "#118") || strings.Contains(p, "duplicate") || strings.Contains(p, "no id") {
		t.Fatalf("guard named a dropped item:\n%s", p)
	}
	for _, mode := range []string{ModeAttempt, ModeReview} {
		if p := systemPrompt(persona, mode, pageContext{ProblemTitle: "3Sum", LiveItems: items}); strings.Contains(p, "OFF-LIMITS") {
			t.Fatalf("%s prompt carries the general-mode guard:\n%s", mode, p)
		}
	}
}

// TestCoursePersona: the persona comes from X-Coach-Course's manifest; ""/unknown/a
// manifest without one fall back to DefaultSlug's; no DSA at all falls back to the
// generic line.
func TestCoursePersona(t *testing.T) {
	reg := coursetest.Registry(t)
	dsa := coursetest.DSA(t).Coach.Persona
	fixture := coursetest.Fixtures(t)[coursetest.FixtureActive].Coach.Persona
	if dsa == "" || fixture == "" || dsa == fixture {
		t.Fatalf("test needs two distinct personas: dsa=%q fixture=%q", dsa, fixture)
	}
	for _, tc := range []struct{ slug, want string }{
		{course.DefaultSlug, dsa},
		{coursetest.FixtureActive, fixture},
		{"", dsa},
		{"no-such-course", dsa},
	} {
		if got := coursePersona(reg, tc.slug); got != tc.want {
			t.Errorf("coursePersona(%q) = %q, want %q", tc.slug, got, tc.want)
		}
	}

	// A known course whose manifest carries no coach block → the default course's.
	noCoach := *coursetest.Fixtures(t)[coursetest.FixtureActive]
	noCoach.Coach = nil
	reg2 := course.NewRegistry(map[string]*course.Manifest{
		course.DefaultSlug: coursetest.DSA(t), coursetest.FixtureActive: &noCoach,
	})
	if got := coursePersona(reg2, coursetest.FixtureActive); got != dsa {
		t.Errorf("course without a coach block: persona %q, want DSA's", got)
	}
	// No default course at all → the generic line, never an empty section.
	reg3 := course.NewRegistry(map[string]*course.Manifest{coursetest.FixtureActive: &noCoach})
	if got := coursePersona(reg3, ""); got != fallbackPersona {
		t.Errorf("no default course: persona %q, want the fallback", got)
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

// TestBuildTurnsTrimsTo32KiB: L18's byte cap drops the oldest turns until the content
// totals ≤ 32 KiB, keeps a user turn first, and ALWAYS keeps the current turn.
func TestBuildTurnsTrimsTo32KiB(t *testing.T) {
	big := strings.Repeat("x", 4000) // maxMessageLen-sized turns
	var history []store.Message
	for i := 0; i < 15; i++ {
		role := store.RoleUser
		if i%2 == 1 {
			role = store.RoleAssistant
		}
		history = append(history, store.Message{Role: role, Content: big + itoaLocal(i)})
	}
	turns := buildTurns(history)
	total := 0
	for _, tr := range turns {
		total += len(tr.Content)
	}
	if total > historyMaxBytes {
		t.Fatalf("turns total %d bytes, want <= %d", total, historyMaxBytes)
	}
	if turns[0].Role != store.RoleUser {
		t.Fatalf("trimmed window starts on %q", turns[0].Role)
	}
	if last := turns[len(turns)-1].Content; last != big+"14" {
		t.Fatalf("current turn dropped: last = %.12q…", last)
	}
	// 8 turns of ~4 KiB fit (32 000 B < 32 768 B), but the window must start on a user
	// turn: turns 7..14 start on an assistant (7 is odd), so 8..14 = 7 turns remain.
	if len(turns) != 7 {
		t.Fatalf("kept %d turns, want 7", len(turns))
	}

	// A single current turn larger than the cap is still sent (never an empty request).
	huge := []store.Message{
		{Role: store.RoleUser, Content: big}, {Role: store.RoleAssistant, Content: big},
		{Role: store.RoleUser, Content: strings.Repeat("y", historyMaxBytes+10)},
	}
	if turns := buildTurns(huge); len(turns) != 1 || turns[0].Role != store.RoleUser {
		t.Fatalf("oversized current turn: %d turns", len(turns))
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

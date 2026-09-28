package course_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/sujaykumarsuman/xlearn/curriculum"
	"github.com/sujaykumarsuman/xlearn/internal/course"
)

// The DSA manifest golden test, side one of two (golden = v1).
//
// goldenDSA is a LITERAL table of v1's behaviour, each value citing the v1 source it
// pins. The loaded curriculum/courses/dsa/course.json must equal it exactly. Side two
// is the mirror tests inside each owning package (internal/practice/store,
// internal/review/store, internal/assessment, internal/assessment/store,
// internal/coach, internal/gateway): they compare each service's OWN constants and
// migration CHECK lists to the loaded manifest, so drift on either side fails CI.
//
// DORMANT values are pinned too, so changing them is always deliberate: nothing in M1
// reads grading.params["verdict_timer@1"] (the owner's D18 timing; the selected
// strategy is self@1, v1's free pick), plan.est_minutes (v1 plans by a count of 3,
// internal/gateway/dashboard.go:23), mistakes.prefill, mock.pools, mock.evidence_hints
// or coach (m1-07 moves the persona into the prompt). A later sprint that changes a
// DSA method value on purpose (m2-05 D2, m3-08 D18, m3-10 prefill, m2-04 minutes)
// updates this table in the same PR and cites the decision.
func goldenDSA() *course.Manifest {
	visible := true
	return &course.Manifest{
		Format: 1,
		// curriculum/paths.json:3-5 (slug, title, status); DSA ids stay bare (m1-09 id guard).
		Slug:     "dsa",
		Title:    "Data Structures & Algorithms",
		Status:   "active",
		IDPrefix: "dsa",

		// web/src/nav.ts:42-73 navForPath: Today · Roadmap · Problems · Progress, then the
		// "Practice loop" cap: Revision · Mistakes · Mock interview.
		Nav: &course.Nav{
			ItemNoun: "problem",
			Groups: []course.NavGroup{
				{Items: []course.NavItem{
					{Screen: "today", Label: "Today"},
					{Screen: "roadmap", Label: "Roadmap"},
					{Screen: "problems", Label: "Problems"},
					{Screen: "progress", Label: "Progress"},
				}},
				{Cap: "Practice loop", Items: []course.NavItem{
					{Screen: "revision", Label: "Revision"},
					{Screen: "mistakes", Label: "Mistakes"},
					{Screen: "mock", Label: "Mock interview"},
				}},
			},
		},

		Stages: &course.Stages{
			// internal/practice/store/store.go:39 AttemptTimer = 15 * time.Minute
			// (mirrored at web/src/screens/Problem.tsx:19, attempt: 900).
			Attempt: course.TimedStage{DurationS: 900, Label: "Attempt"},
			// internal/practice/store/store.go:40 HintTimer = 10 * time.Minute (Problem.tsx:19, hint: 600).
			Hint: course.TimedStage{DurationS: 600, Label: "Hint"},
			// web/src/screens/Problem.tsx:358-361 stage tabs: Attempt · Hint · Solution.
			Solution: course.SolutionStage{Label: "Solution"},
			// R-PF4: re-implement from memory after the solution (Problem.tsx:586).
			Reimplement: "required",
			// internal/practice/store/store.go:44 EarlyRevealPenaltyDays = 3
			// (internal/review/store/store.go:40 OwedAttemptDays = 3).
			EarlyRevealPenaltyDays: 3,
		},

		Grading: &course.Grading{
			// v1's pick: POST /problems/{id}/outcome, any of the four grades (cap none).
			Strategy:   "self@1",
			SelfReport: course.SelfReport{Outcome: "allowed", Touch: "allowed", Mock: "allowed"},
			Params: course.StrategyParams{
				Self: &course.SelfParams{Cap: "none"},
				// DORMANT until m3-08 selects verdict_timer@1: the owner's D18 defaults
				// (ADR-0029 §3) — 45:00 hard limit, hint at 15:00, Clean ≤ 20:00 and
				// ≤ 3 failed submits, CE and REJECTED free.
				VerdictTimer: &course.VerdictTimerParams{
					TimeLimitS:        2700,
					HintAtS:           900,
					CleanWithinS:      1200,
					MaxFailedForClean: 3,
					FreeClasses:       []string{"CE", "REJECTED"},
				},
			},
		},

		// web/src/screens/Problem.tsx:21-26 OUTCOMES (label + hint copy);
		// internal/practice/store/migrations/00001_init.sql:26 last_outcome CHECK.
		Grades: []course.Grade{
			{ID: "clean", Label: "Clean", Hint: "no help, in time"},
			{ID: "rough", Label: "Rough", Hint: "solved, ugly"},
			{ID: "assisted", Label: "Assisted", Hint: "needed a hint"},
			{ID: "miss", Label: "Miss", Hint: "didn’t get it"},
		},

		Revision: &course.Revision{
			// internal/review/store/store.go:27 touchDays = {1, 3, 7, 21, 45}.
			LadderDays: []int{1, 3, 7, 21, 45},
			Bands: []course.Band{
				{
					// L1–3: a re-solve from blank on a 20-minute timer
					// (web/src/screens/Revision.tsx:15 REVISION_TIMER_SECONDS = 20 * 60);
					// AutoPass (internal/review/store/store.go:706-708): pattern named in
					// < 120 s (store.go:35 NamePatternMaxSecs; Revision.tsx:17), solved in
					// the timer, complexity stated.
					Levels: []int{1, 2, 3}, Format: "resolve", Label: "Re-solve", TimerS: 1200,
					Parts: []string{"solution"},
					Criteria: []course.Criterion{
						{Key: "pattern_named_fast", ThresholdS: 120},
						{Key: "correct_in_timer"},
						{Key: "complexity_stated"},
					},
					MockMode: false,
				},
				{
					// L4–5: the same, under mock conditions
					// (internal/review/store/store.go:712 isMockTouch: level 4 or 5).
					Levels: []int{4, 5}, Format: "resolve", Label: "Re-solve", TimerS: 1200,
					Parts: []string{"solution"},
					Criteria: []course.Criterion{
						{Key: "pattern_named_fast", ThresholdS: 120},
						{Key: "correct_in_timer"},
						{Key: "complexity_stated"},
					},
					MockMode: true,
				},
			},
			// v1 has no drills (no role=drill items).
			Drills: false,
		},

		// internal/review/store/migrations/00002_mistakes_notifications.sql:28-30 category
		// CHECK (internal/review/store/store.go:86-95 MistakeCategories); labels from
		// web/src/lib/mistakes.ts:10-19 MISTAKE_CATEGORIES.
		Mistakes: &course.Mistakes{
			Categories: []course.Category{
				{ID: "misread", Label: "Misread"},
				{ID: "wrong_pattern", Label: "Wrong pattern"},
				{ID: "right_pattern_wrong_state", Label: "Right pattern, wrong state"},
				{ID: "off_by_one", Label: "Off-by-one / boundary"},
				{ID: "language_bug", Label: "Language bug"},
				{ID: "complexity_misjudged", Label: "Complexity misjudged"},
				{ID: "communication", Label: "Communication"},
				{ID: "time_management", Label: "Time management"},
			},
			// DORMANT: empty until m3-10 fills it (t4 §6.3).
			Prefill: []course.PrefillRule{},
		},

		Mock: &course.Mock{
			// internal/assessment/store/store.go:46 MockDuration = 45 * time.Minute.
			DurationS: 2700,
			// internal/assessment/mock.go:30-43 phases (the fixed six-phase rail).
			Rail: []course.RailPhase{
				{Label: "Clarify", StartMin: 0, EndMin: 5,
					Prompt: "Restate the problem and pin the ambiguities out loud — input ranges, duplicates, output order, edge inputs. State your assumptions before you write anything."},
				{Label: "Brute force", StartMin: 5, EndMin: 10,
					Prompt: "Describe the naive solution and its cost out loud, and say why it won't pass at the largest input. Naming the brute-force bound first earns real rubric points."},
				{Label: "Observation → plan", StartMin: 10, EndMin: 18,
					Prompt: "Find and say the key insight, then name the pattern out loud before you code. Commit to the optimal approach decisively instead of re-litigating brute force."},
				{Label: "Code", StartMin: 18, EndMin: 33,
					Prompt: "Implement steadily and keep narrating what each block does. Watch the parts your journal flags as weak spots for this pattern."},
				{Label: "Trace + edges", StartMin: 33, EndMin: 40,
					Prompt: "Dry-run your code on a normal case and the tricky edges (empty, single, duplicates, extremes). Talk through each step and fix what the trace reveals."},
				{Label: "Complexity + follow-ups", StartMin: 40, EndMin: 45,
					Prompt: "State the final time and space complexity, and offer a follow-up or optimisation before you're asked."},
			},
			// internal/assessment/store/store.go:28-36 Dimensions (canonical order) and
			// :42-43 MinScore/MaxScore → 7 × 1–5 = /35 (00001_init.sql:39, :65-67);
			// labels from internal/assessment/mock.go:120-128 dimensionLabels.
			Rubric: course.Rubric{
				ID: "dsa-mock@1",
				Dims: []course.Dimension{
					{ID: "communication", Label: "Communication"},
					{ID: "problem_understanding", Label: "Problem understanding"},
					{ID: "brute_force", Label: "Brute force"},
					{ID: "optimisation", Label: "Optimisation"},
					{ID: "code_quality", Label: "Code quality"},
					{ID: "edge_cases", Label: "Edge cases"},
					{ID: "complexity", Label: "Complexity"},
				},
				Scale: []int{1, 5},
			},
			// internal/assessment/mock.go:147 readinessTargets = {W13: 24, W15: 28, Pre: 30}
			// (internal/gateway/dashboard.go:26 mockW13Target = 24).
			Targets: course.Targets{W13: 24, W15: 28, Pre: 30},
			// DORMANT: v1 has no pools (the learner picks) and no evidence hints (M3+).
			Pools:         []string{},
			EvidenceHints: []course.EvidenceHint{},
		},

		// DORMANT proposal (D4; first reader m2-04). v1 plans by a count of 3
		// (internal/gateway/dashboard.go:23 newWorkLimit). touch_recall is unused until a
		// DSA band uses the recall format.
		Plan: &course.Plan{EstMinutes: map[string]int{
			"course_attempt": 45,
			"touch_resolve":  20,
			"touch_recall":   5,
			"mock":           45,
		}},

		// D7: DSA is public by default. Metrics = v1's public tiles
		// (internal/gateway/public.go:44-76) minus mock best/average (D31, m1-05).
		PublicStats: course.PublicStats{
			DefaultVisible: &visible,
			Metrics: []string{
				"solved", "streak", "mock_count", "heatmap",
				"course_completion", "phase_completion", "pattern_mastery",
			},
		},

		// internal/coach/prompt.go:56-57, the course-specific persona lines (verbatim;
		// m1-07 moves them into the prompt from here). Coach off during touches and mocks.
		Coach: &course.Coach{
			Persona: "You are the xLearn coach, an AI mentor inside a guided DSA interview-prep course. " +
				"You are concise, encouraging, and technically precise. Prefer short paragraphs and, " +
				"when it helps, small illustrative snippets. Go is the course's primary language.",
			PrimaryLanguage: "go",
			OffDuring:       []string{"touch", "mock"},
		},
	}
}

func loadAll(t *testing.T) map[string]*course.Manifest {
	t.Helper()
	ms, err := course.Load(curriculum.FS)
	if err != nil {
		t.Fatalf("course.Load(curriculum.FS): %v", err)
	}
	return ms
}

func TestDSAManifestGolden(t *testing.T) {
	got := loadAll(t)["dsa"]
	if got == nil {
		t.Fatal("no dsa manifest")
	}
	want := goldenDSA()
	if !reflect.DeepEqual(got, want) {
		gj, _ := json.MarshalIndent(got, "", "  ")
		wj, _ := json.MarshalIndent(want, "", "  ")
		t.Fatalf("curriculum/courses/dsa/course.json drifted from the v1 golden table.\n--- got\n%s\n--- want\n%s", gj, wj)
	}
}

// TestComingSoonStubs pins the five catalog stubs: coming_soon, the proposed prefixes
// (m1-09 locks them via path.id_prefix), behavioral hidden by default (D7), and no
// method blocks yet.
func TestComingSoonStubs(t *testing.T) {
	ms := loadAll(t)
	want := map[string]struct {
		prefix  string
		visible bool
	}{
		"system-design":  {"sd", true},
		"go-concurrency": {"gc", true},
		"lld-ood":        {"lld", true},
		"sql":            {"sql", true},
		"behavioral":     {"beh", false},
	}
	if len(ms) != len(want)+1 {
		t.Fatalf("got %d manifests, want %d (dsa + %d stubs)", len(ms), len(want)+1, len(want))
	}
	for slug, w := range want {
		m := ms[slug]
		if m == nil {
			t.Errorf("%s: missing", slug)
			continue
		}
		if m.Status != course.StatusComingSoon || m.IDPrefix != w.prefix || m.PublicStats.Visible() != w.visible {
			t.Errorf("%s: status %q prefix %q visible %v, want coming_soon %q %v",
				slug, m.Status, m.IDPrefix, m.PublicStats.Visible(), w.prefix, w.visible)
		}
		if m.Nav != nil || m.Stages != nil || m.Grading != nil || m.Grades != nil || m.Revision != nil ||
			m.Mistakes != nil || m.Mock != nil || m.Plan != nil || m.Coach != nil || m.PublicStats.Metrics != nil {
			t.Errorf("%s: a coming_soon stub carries only identity + public_stats.default_visible", slug)
		}
	}
}

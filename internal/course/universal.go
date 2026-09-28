// Package course holds the typed course manifest (ADR-0026 §1) and the frozen public
// item schema (ADR-0027 §1, ADR-0029): Go types, strict loading and validation.
//
// It is a shared, stdlib-only library of types and constants that every service may
// import (the manifest is data compiled into every image, never fetched at runtime).
// It has no database access and imports no service package, so it can never cause an
// import cycle. In M1a nothing reads a manifest at runtime; the first readers arrive in
// M1b and M2 (see docs/v2/sprints/sprint-m1-01.md, Scope → Out).
package course

// This file holds the UNIVERSAL method (D3, ADR-0026 §4): the parts of the learning
// method every course shares. A manifest may not change them; Validate checks that it
// restates them exactly where it restates them at all (the ladder, the grades).

// LadderDays is the universal five-touch revision ladder: touch level n (1-based) is due
// LadderDays[n-1] days after its anchor. A fail resets to Day 1.
var LadderDays = []int{1, 3, 7, 21, 45}

// Grade ids of the universal four-grade outcome scale, best first.
const (
	GradeClean    = "clean"
	GradeRough    = "rough"
	GradeAssisted = "assisted"
	GradeMiss     = "miss"
)

// Grades is the universal outcome scale in canonical order.
var Grades = []string{GradeClean, GradeRough, GradeAssisted, GradeMiss}

// CoreMistakeCategories every course's taxonomy must include (ADR-0026 §4). A course
// adds its own categories after these; category ids are append-only.
var CoreMistakeCategories = []string{"misread", "time_management", "communication"}

// StageRoles are the universal content stages of an attempt, in order. Durations and
// labels are per course; the roles are not.
var StageRoles = []string{"attempt", "hint", "solution"}

// Nav screen keys: the closed set of course-scoped screens a manifest's nav may list.
// A new screen is SPA code plus a release, then a new key here.
const (
	ScreenToday    = "today"
	ScreenRoadmap  = "roadmap"
	ScreenProblems = "problems"
	ScreenProgress = "progress"
	ScreenRevision = "revision"
	ScreenMistakes = "mistakes"
	ScreenMock     = "mock"
)

// NavScreens is the closed nav screen set.
var NavScreens = []string{
	ScreenToday, ScreenRoadmap, ScreenProblems, ScreenProgress,
	ScreenRevision, ScreenMistakes, ScreenMock,
}

// Grading strategy ids: the closed Go set a manifest picks from (ADR-0029 §3). A
// strategy is code; a manifest only selects one and sets its thresholds.
const (
	StrategySelf         = "self@1"
	StrategyVerdictTimer = "verdict_timer@1"
	StrategyRubricPct    = "rubric_pct@1"
	StrategyWeightedGate = "weighted_gate@1"
)

// Strategies is the closed strategy set.
var Strategies = []string{StrategySelf, StrategyVerdictTimer, StrategyRubricPct, StrategyWeightedGate}

// VerdictClasses is the closed code-verdict class set (ADR-0029 §2). A
// verdict_timer@1 `free_classes` entry must be one of these.
var VerdictClasses = []string{
	"AC", "WA", "TLE", "MLE", "OLE", "RE", "CE", "REJECTED", "RACE", "DEADLOCK", "LEAK",
}

// PublicMetrics is the closed set of public-profile metrics a course may expose
// (manifest public_stats.metrics; D7). Mock best and average are deliberately absent:
// the public profile shows the mock count only (D31).
var PublicMetrics = []string{
	"solved", "streak", "mock_count", "heatmap",
	"course_completion", "phase_completion", "pattern_mastery",
}

// Manifest status values.
const (
	StatusActive     = "active"
	StatusPreview    = "preview"
	StatusComingSoon = "coming_soon"
	StatusRetired    = "retired"
)

// ManifestStatuses is the closed manifest status set.
var ManifestStatuses = []string{StatusActive, StatusPreview, StatusComingSoon, StatusRetired}

// Re-implement policies (stages.reimplement).
var ReimplementPolicies = []string{"required", "optional", "off"}

// Self-report policies per signal (grading.self_report.{outcome,touch,mock}).
var SelfReportPolicies = []string{"allowed", "evaluator_only"}

// self@1 caps: "none" is v1's free pick; "stage" caps the pick at the stage ceiling.
var SelfCaps = []string{"none", "stage"}

// Prefill strengths (mistakes.prefill[].strength; ADR-0029 §4 precedence).
var PrefillStrengths = []string{"strong", "weak"}

// CoachOffDuring is the closed set of contexts in which a course may switch the coach off.
var CoachOffDuring = []string{"touch", "mock"}

// Plan minute keys (plan.est_minutes): course_attempt, mock, and touch_<band format>.
const (
	EstCourseAttempt = "course_attempt"
	EstMock          = "mock"
	EstTouchPrefix   = "touch_"
)

// MaxPersonaChars bounds coach.persona (t5 §9).
const MaxPersonaChars = 600

func contains(set []string, v string) bool {
	for _, s := range set {
		if s == v {
			return true
		}
	}
	return false
}

package assessment

import (
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/assessment/store"
)

// This file holds the mock-interview domain model that is NOT persistence: the fixed
// six-phase rail (R-MK1), the server-authoritative timer/phase computation, the
// seven rubric dimensions' display names (R-MK2), and the readiness targets (R-MK3).
// The client HUD is a mirror of what computeRail returns — the phase index and the
// remaining time are never trusted from a client clock.

// Phase is one interview phase in the 45-minute rail (R-MK1 / PRD 6.5). Grow is the
// segment's flex weight — the phase's minute span — so the rail is time-proportional.
type Phase struct {
	Index    int    `json:"index"`
	Label    string `json:"label"`
	Range    string `json:"range"`
	StartMin int    `json:"startMin"`
	EndMin   int    `json:"endMin"`
	Grow     int    `json:"grow"`
	Prompt   string `json:"prompt"`
}

// phases is the fixed six-phase rail: 0–5 clarify · 5–10 brute force · 10–18
// observation→plan · 18–33 code · 33–40 trace+edges · 40–45 complexity+follow-ups
// (R-MK1). Prompts are problem-agnostic so they apply to any mock set.
var phases = []Phase{
	{Index: 0, Label: "Clarify", Range: "0–5", StartMin: 0, EndMin: 5, Grow: 5,
		Prompt: "Restate the problem and pin the ambiguities out loud — input ranges, duplicates, output order, edge inputs. State your assumptions before you write anything."},
	{Index: 1, Label: "Brute force", Range: "5–10", StartMin: 5, EndMin: 10, Grow: 5,
		Prompt: "Describe the naive solution and its cost out loud, and say why it won't pass at the largest input. Naming the brute-force bound first earns real rubric points."},
	{Index: 2, Label: "Observation → plan", Range: "10–18", StartMin: 10, EndMin: 18, Grow: 8,
		Prompt: "Find and say the key insight, then name the pattern out loud before you code. Commit to the optimal approach decisively instead of re-litigating brute force."},
	{Index: 3, Label: "Code", Range: "18–33", StartMin: 18, EndMin: 33, Grow: 15,
		Prompt: "Implement steadily and keep narrating what each block does. Watch the parts your journal flags as weak spots for this pattern."},
	{Index: 4, Label: "Trace + edges", Range: "33–40", StartMin: 33, EndMin: 40, Grow: 7,
		Prompt: "Dry-run your code on a normal case and the tricky edges (empty, single, duplicates, extremes). Talk through each step and fix what the trace reveals."},
	{Index: 5, Label: "Complexity + follow-ups", Range: "40–45", StartMin: 40, EndMin: 45, Grow: 5,
		Prompt: "State the final time and space complexity, and offer a follow-up or optimisation before you're asked."},
}

// Phases returns a copy of the fixed rail (the setup reference + client mirror).
func Phases() []Phase {
	out := make([]Phase, len(phases))
	copy(out, phases)
	return out
}

// PhaseState is one rail segment plus whether it is past / current / upcoming.
type PhaseState struct {
	Phase
	State string `json:"state"` // "past" | "current" | "upcoming"
}

// RailState is the server-authoritative timer + phase state of a session.
type RailState struct {
	ElapsedSeconds   int          `json:"elapsedSeconds"`
	RemainingSeconds int          `json:"remainingSeconds"`
	Overtime         bool         `json:"overtime"` // elapsed clamped at 45:00 (past deadline)
	PhaseIndex       int          `json:"phaseIndex"`
	PhaseLabel       string       `json:"phaseLabel"`
	Phases           []PhaseState `json:"phases"`
}

// mockDurationSecs is the 45-minute window in seconds (R-MK1).
var mockDurationSecs = int(store.MockDuration / time.Second)

// computeRail derives the rail state from the server clock: elapsed = now - startedAt,
// clamped to [0, 45:00]. The current phase is the one whose [start,end) minute window
// contains the elapsed minutes; a session past its deadline reports elapsed clamped at
// 45:00 with the final phase current (overtime=true).
func computeRail(startedAt, now time.Time) RailState {
	elapsed := int(now.Sub(startedAt) / time.Second)
	overtime := false
	if elapsed < 0 {
		elapsed = 0
	}
	if elapsed >= mockDurationSecs {
		elapsed = mockDurationSecs
		overtime = true
	}
	idx := phaseIndexFor(elapsed)
	states := make([]PhaseState, len(phases))
	for i, p := range phases {
		st := "upcoming"
		switch {
		case i < idx:
			st = "past"
		case i == idx:
			st = "current"
		}
		states[i] = PhaseState{Phase: p, State: st}
	}
	return RailState{
		ElapsedSeconds:   elapsed,
		RemainingSeconds: mockDurationSecs - elapsed,
		Overtime:         overtime,
		PhaseIndex:       idx,
		PhaseLabel:       phases[idx].Label,
		Phases:           states,
	}
}

// phaseIndexFor returns the current phase for an elapsed-seconds value (clamped to
// [0, 45:00]): the first phase whose end boundary the elapsed time has not yet
// reached, or the final phase once the window is exhausted.
func phaseIndexFor(elapsedSecs int) int {
	for _, p := range phases {
		if elapsedSecs < p.EndMin*60 {
			return p.Index
		}
	}
	return len(phases) - 1
}

// dimensionLabels maps each rubric key to its display name (R-MK2).
var dimensionLabels = map[string]string{
	"communication":         "Communication",
	"problem_understanding": "Problem understanding",
	"brute_force":           "Brute force",
	"optimisation":          "Optimisation",
	"code_quality":          "Code quality",
	"edge_cases":            "Edge cases",
	"complexity":            "Complexity",
}

// dimensionLabel returns the display name for a rubric key (the key itself as a
// fallback for an unknown key, which validation should already have rejected).
func dimensionLabel(key string) string {
	if l, ok := dimensionLabels[key]; ok {
		return l
	}
	return key
}

// Targets are the R-MK3 readiness thresholds the trend chart draws as dashed lines.
type Targets struct {
	W13 int `json:"w13"` // >= 24 by Week 13
	W15 int `json:"w15"` // >= 28 by Week 15
	Pre int `json:"pre"` // >= 30 pre-interview
}

// readinessTargets are the fixed R-MK3 targets (from the PRD, not UI copy).
var readinessTargets = Targets{W13: 24, W15: 28, Pre: 30}

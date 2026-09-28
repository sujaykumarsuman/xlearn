package curriculum

import (
	"sort"

	"github.com/sujaykumarsuman/xlearn/internal/course"
)

// GradingSummary is problem.grading_summary (sprint m3-01): how an item is graded,
// derived at seed from its public parts and grader steps. It is answer-free and public
// (never stage-gated): GET /problems/{id} serves it, and the Problems list markers
// (m3-09, m3-12) will.
//
// Mode: "self" when no part is graded (the v1 self path; the summary is then just
// {"mode": "self"}), "auto" when every part is graded (by judge: auto or ai), "mixed"
// when some parts are graded and some are not.
type GradingSummary struct {
	Mode        string        `json:"mode"`
	Parts       []PartSummary `json:"parts,omitempty"`
	GraderKinds []string      `json:"grader_kinds,omitempty"`
	Languages   []string      `json:"languages,omitempty"`
}

// PartSummary is one part's public grading shape.
type PartSummary struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Grading string `json:"grading"`
	Cadence string `json:"cadence"`
}

// Grading modes.
const (
	ModeSelf  = "self"
	ModeAuto  = "auto"
	ModeMixed = "mixed"
)

// SummarizeGrading derives an item's grading summary.
func SummarizeGrading(it *course.Item) GradingSummary {
	graded := 0
	for _, p := range it.Parts {
		if p.Grading != "none" {
			graded++
		}
	}
	if graded == 0 {
		return GradingSummary{Mode: ModeSelf}
	}
	s := GradingSummary{Mode: ModeAuto}
	if graded < len(it.Parts) {
		s.Mode = ModeMixed
	}
	langs := map[string]bool{}
	for _, p := range it.Parts {
		s.Parts = append(s.Parts, PartSummary{ID: p.ID, Type: p.Type, Grading: p.Grading, Cadence: p.Cadence})
		if p.Type == "code" {
			for _, l := range p.Config.Languages {
				langs[l] = true
			}
		}
	}
	kinds := map[string]bool{}
	for _, st := range it.Grader {
		kinds[st.Kind] = true
	}
	s.GraderKinds = sortedSet(kinds)
	s.Languages = sortedSet(langs)
	return s
}

func sortedSet(m map[string]bool) []string {
	if len(m) == 0 {
		return nil
	}
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

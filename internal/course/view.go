package course

import "strings"

// View is the learner-safe view of a manifest (sprint m1-03; t0 §8: curriculum serves a
// learner-safe manifest view). curriculum adds it as the `course` block on GET /paths and
// GET /paths/{slug}; the SPA renders the course nav, labels and short code from it.
//
// It carries presentation data only: nothing answer-bearing and nothing a learner could
// game. Grading thresholds, revision criteria, prefill rules, mock pools and evidence
// hints, the rubric and the coach persona stay server-side. The JSON keys are snake_case
// like the rest of curriculum's API; the shape is append-only.
type View struct {
	Slug   string `json:"slug"`
	Title  string `json:"title"`
	Status string `json:"status"`
	// ShortCode is the mono badge in the course selector (DSA, …): the id prefix,
	// upper-cased.
	ShortCode string `json:"short_code"`
	// Nav is the course-scoped left nav (absent for a coming_soon or retired course).
	Nav *ViewNav `json:"nav,omitempty"`
	// Stages are the stage labels and timers (absent without a stages block).
	Stages *ViewStages `json:"stages,omitempty"`
	// EstMinutes are the planning estimates (D4), keyed course_attempt, mock, touch_<format>.
	EstMinutes map[string]int `json:"est_minutes,omitempty"`
	// MockRail is the mock's phase rail, labels and minute ranges only (no coaching prompt).
	// Empty when the course has no mock.
	MockRail []ViewRailPhase `json:"mock_rail,omitempty"`
	// PrimaryLanguage is the course's primary language (the "Go-first" note), "" if none.
	PrimaryLanguage string `json:"primary_language,omitempty"`
}

// ViewNav is the nav block: the item noun and the titled groups of screens.
type ViewNav struct {
	ItemNoun string         `json:"item_noun"`
	Groups   []ViewNavGroup `json:"groups"`
}

// ViewNavGroup is one titled nav section.
type ViewNavGroup struct {
	Cap   string        `json:"cap,omitempty"`
	Items []ViewNavItem `json:"items"`
}

// ViewNavItem is one nav entry: a screen from the closed NavScreens set and its label.
type ViewNavItem struct {
	Screen string `json:"screen"`
	Label  string `json:"label"`
}

// ViewStages are the stage labels and timers.
type ViewStages struct {
	Attempt     ViewTimedStage `json:"attempt"`
	Hint        ViewTimedStage `json:"hint"`
	Solution    ViewStage      `json:"solution"`
	Reimplement string         `json:"reimplement"`
}

// ViewTimedStage is a stage with a countdown.
type ViewTimedStage struct {
	DurationS int    `json:"duration_s"`
	Label     string `json:"label"`
}

// ViewStage is an untimed stage's label.
type ViewStage struct {
	Label string `json:"label"`
}

// ViewRailPhase is one mock rail phase, [StartMin, EndMin) minutes.
type ViewRailPhase struct {
	Label    string `json:"label"`
	StartMin int    `json:"start_min"`
	EndMin   int    `json:"end_min"`
}

// LearnerView derives the learner-safe view of m.
func (m *Manifest) LearnerView() View {
	v := View{
		Slug:      m.Slug,
		Title:     m.Title,
		Status:    m.Status,
		ShortCode: strings.ToUpper(m.IDPrefix),
	}
	if m.Nav != nil {
		nav := &ViewNav{ItemNoun: m.Nav.ItemNoun, Groups: make([]ViewNavGroup, 0, len(m.Nav.Groups))}
		for _, g := range m.Nav.Groups {
			vg := ViewNavGroup{Cap: g.Cap, Items: make([]ViewNavItem, 0, len(g.Items))}
			for _, it := range g.Items {
				vg.Items = append(vg.Items, ViewNavItem{Screen: it.Screen, Label: it.Label})
			}
			nav.Groups = append(nav.Groups, vg)
		}
		v.Nav = nav
	}
	if m.Stages != nil {
		v.Stages = &ViewStages{
			Attempt:     ViewTimedStage{DurationS: m.Stages.Attempt.DurationS, Label: m.Stages.Attempt.Label},
			Hint:        ViewTimedStage{DurationS: m.Stages.Hint.DurationS, Label: m.Stages.Hint.Label},
			Solution:    ViewStage{Label: m.Stages.Solution.Label},
			Reimplement: m.Stages.Reimplement,
		}
	}
	if m.Plan != nil && len(m.Plan.EstMinutes) > 0 {
		v.EstMinutes = make(map[string]int, len(m.Plan.EstMinutes))
		for k, n := range m.Plan.EstMinutes {
			v.EstMinutes[k] = n
		}
	}
	if m.Mock != nil {
		v.MockRail = make([]ViewRailPhase, 0, len(m.Mock.Rail))
		for _, p := range m.Mock.Rail {
			v.MockRail = append(v.MockRail, ViewRailPhase{Label: p.Label, StartMin: p.StartMin, EndMin: p.EndMin})
		}
	}
	if m.Coach != nil {
		v.PrimaryLanguage = m.Coach.PrimaryLanguage
	}
	return v
}

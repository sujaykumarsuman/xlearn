package course

// ManifestFormat is the manifest format version this package reads (`format`).
const ManifestFormat = 1

// Manifest is one course's settings-only manifest, `curriculum/courses/<slug>/course.json`
// (ADR-0026 §1, ADR-0027 §2). It carries data only, no logic. The JSON shape is
// `curriculum/_schema/course.schema.json`; schema_test.go keeps the two in lockstep.
//
// A `coming_soon` (or `retired`) manifest may omit the method blocks. An `active` or
// `preview` manifest must carry all of them except `mock` (a course may have no mock).
type Manifest struct {
	Format   int    `json:"format"`
	Slug     string `json:"slug"`
	Title    string `json:"title"`
	Status   string `json:"status"`
	IDPrefix string `json:"id_prefix"`

	Nav         *Nav        `json:"nav,omitempty"`
	Stages      *Stages     `json:"stages,omitempty"`
	Grading     *Grading    `json:"grading,omitempty"`
	Grades      []Grade     `json:"grades,omitempty"`
	Revision    *Revision   `json:"revision,omitempty"`
	Mistakes    *Mistakes   `json:"mistakes,omitempty"`
	Mock        *Mock       `json:"mock,omitempty"`
	Plan        *Plan       `json:"plan,omitempty"`
	PublicStats PublicStats `json:"public_stats"`
	Coach       *Coach      `json:"coach,omitempty"`
}

// Nav is the course-scoped left nav: an item noun and titled groups of screens.
type Nav struct {
	ItemNoun string     `json:"item_noun"`
	Groups   []NavGroup `json:"groups"`
}

// NavGroup is one titled nav section. Cap is the optional section caption.
type NavGroup struct {
	Cap   string    `json:"cap,omitempty"`
	Items []NavItem `json:"items"`
}

// NavItem is one nav entry: a screen from the closed NavScreens set and its label.
type NavItem struct {
	Screen string `json:"screen"`
	Label  string `json:"label"`
}

// Stages configures the universal stage roles (attempt → hint → solution →
// re-implement → conclude) for this course.
type Stages struct {
	Attempt  TimedStage    `json:"attempt"`
	Hint     TimedStage    `json:"hint"`
	Solution SolutionStage `json:"solution"`
	// Reimplement is required | optional | off.
	Reimplement string `json:"reimplement"`
	// EarlyRevealPenaltyDays is how far out the owed re-attempt is queued when the
	// solution is revealed before the attempt timer elapses (R-PF2).
	EarlyRevealPenaltyDays int `json:"early_reveal_penalty_days"`
}

// TimedStage is a stage with a server-authoritative countdown.
type TimedStage struct {
	DurationS int    `json:"duration_s"`
	Label     string `json:"label"`
}

// SolutionStage is the solution stage's CONFIG (its label only). It never carries
// solution content; schema_test.go asserts it has no other property.
type SolutionStage struct {
	Label string `json:"label"`
}

// Grading selects the course's grading strategy and its thresholds.
type Grading struct {
	// Strategy is one of the closed Strategies.
	Strategy string `json:"strategy"`
	// SelfReport says, per signal, whether a learner self-report is allowed.
	SelfReport SelfReport `json:"self_report"`
	// Params holds thresholds per strategy id. Params for a strategy other than the
	// selected one are DORMANT: nothing reads them until a later sprint selects it.
	Params StrategyParams `json:"params"`
}

// SelfReport is the per-signal self-report policy: allowed | evaluator_only.
type SelfReport struct {
	Outcome string `json:"outcome"`
	Touch   string `json:"touch"`
	Mock    string `json:"mock"`
}

// StrategyParams holds each strategy's thresholds, keyed by strategy id. The key set is
// the closed Strategies set, so an unknown strategy id fails strict decoding.
type StrategyParams struct {
	Self         *SelfParams         `json:"self@1,omitempty"`
	VerdictTimer *VerdictTimerParams `json:"verdict_timer@1,omitempty"`
	RubricPct    *PctParams          `json:"rubric_pct@1,omitempty"`
	WeightedGate *PctParams          `json:"weighted_gate@1,omitempty"`
}

// SelfParams are self@1's thresholds. Cap "none" is v1's free pick
// (POST /problems/{id}/outcome); "stage" caps the pick at the stage ceiling.
type SelfParams struct {
	Cap string `json:"cap"`
}

// VerdictTimerParams are verdict_timer@1's thresholds (ADR-0029 §3, D18).
type VerdictTimerParams struct {
	// TimeLimitS is the hard limit per attempt: no pass by then concludes as Miss.
	TimeLimitS int `json:"time_limit_s"`
	// HintAtS is when the hint unlocks.
	HintAtS int `json:"hint_at_s"`
	// CleanWithinS is the latest pass that can still be Clean.
	CleanWithinS int `json:"clean_within_s"`
	// MaxFailedForClean is the most failed counted submits a Clean allows.
	MaxFailedForClean int `json:"max_failed_for_clean"`
	// FreeClasses are verdict classes that never count as a failed submit.
	FreeClasses []string `json:"free_classes"`
}

// PctParams are the score-band thresholds of rubric_pct@1 and weighted_gate@1, as
// fractions in (0, 1] with PassPct <= CleanPct.
type PctParams struct {
	CleanPct float64 `json:"clean_pct"`
	PassPct  float64 `json:"pass_pct"`
}

// Grade is one of the universal four grades with this course's copy.
type Grade struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Hint  string `json:"hint"`
}

// Revision configures the touch format per level band. The ladder itself is universal
// and restated here only so the manifest is self-describing (Validate checks it).
type Revision struct {
	LadderDays []int  `json:"ladder_days"`
	Bands      []Band `json:"bands"`
	// Drills says whether role=drill items enter the revision ladder.
	Drills bool `json:"drills"`
}

// Band is the touch format for a set of ladder levels. Minutes live only in
// plan.est_minutes["touch_" + Format], never on the band.
type Band struct {
	Levels []int  `json:"levels"`
	Format string `json:"format"`
	Label  string `json:"label"`
	TimerS int    `json:"timer_s"`
	// Parts lists the item part ids whose grader steps a touch in this band runs
	// (t4 §4.4), on top of the probe steps its criteria map to.
	Parts []string `json:"parts"`
	// Criteria are the pass criteria; a touch passes when every one is met.
	Criteria []Criterion `json:"criteria"`
	// MockMode runs the touch under mock conditions (coach off, statement only).
	MockMode bool `json:"mock_mode"`
}

// Criterion is one touch pass criterion. Key is a criterion IDENTIFIER (a check name,
// e.g. pattern_named_fast), never an expected value.
type Criterion struct {
	Key        string `json:"key"`
	ThresholdS int    `json:"threshold_s,omitempty"`
}

// Mistakes is the course's mistake taxonomy and its data-only pre-fill rules.
type Mistakes struct {
	// Categories must include CoreMistakeCategories; ids are append-only.
	Categories []Category `json:"categories"`
	// Prefill maps judge/practice signals to categories, first match wins (ADR-0029 §4).
	Prefill []PrefillRule `json:"prefill"`
}

// Category is one mistake category.
type Category struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// PrefillRule maps one signal to a category with a strength (strong | weak).
type PrefillRule struct {
	Signal   string `json:"signal"`
	Category string `json:"category"`
	Strength string `json:"strength"`
}

// Mock configures the course's mock interview. The lifecycle is universal.
type Mock struct {
	DurationS int         `json:"duration_s"`
	Rail      []RailPhase `json:"rail"`
	Rubric    Rubric      `json:"rubric"`
	Targets   Targets     `json:"targets"`
	// Pools are item-pool refs the server draws each session's ordered item list from.
	// Empty means v1's behaviour (the learner picks the problem).
	Pools []string `json:"pools"`
	// EvidenceHints map signals to SUGGESTED upper bounds on a dimension (t4 §6.7).
	EvidenceHints []EvidenceHint `json:"evidence_hints"`
}

// RailPhase is one phase of the timed rail, [StartMin, EndMin) minutes.
type RailPhase struct {
	Label    string `json:"label"`
	StartMin int    `json:"start_min"`
	EndMin   int    `json:"end_min"`
	Prompt   string `json:"prompt"`
}

// Rubric is the mock rubric: N dimensions each scored on Scale; max = Scale[1] × N.
type Rubric struct {
	ID    string      `json:"id"`
	Dims  []Dimension `json:"dims"`
	Scale []int       `json:"scale"`
}

// Dimension is one rubric dimension.
type Dimension struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// MaxTotal is the rubric's maximum total (Scale max × number of dimensions).
func (r Rubric) MaxTotal() int {
	if len(r.Scale) != 2 {
		return 0
	}
	return r.Scale[1] * len(r.Dims)
}

// Targets are the readiness targets on the rubric total.
type Targets struct {
	W13 int `json:"w13"`
	W15 int `json:"w15"`
	Pre int `json:"pre"`
}

// EvidenceHint suggests an upper bound for a dimension when a signal is present.
type EvidenceHint struct {
	Signal string `json:"signal"`
	Dim    string `json:"dim"`
	Max    int    `json:"max"`
}

// Plan holds the planning estimates (D4). EstMinutes is the ONLY place minutes live,
// keyed course_attempt, mock and touch_<band format>.
type Plan struct {
	EstMinutes map[string]int `json:"est_minutes"`
}

// TouchMinutes returns the estimated minutes for a touch in band b.
func (p Plan) TouchMinutes(b Band) (int, bool) {
	m, ok := p.EstMinutes[EstTouchPrefix+b.Format]
	return m, ok
}

// PublicStats is the course's public-profile policy (D7).
type PublicStats struct {
	// DefaultVisible is required (a pointer so a missing value is an error, not false).
	DefaultVisible *bool `json:"default_visible"`
	// Metrics are picked from the closed PublicMetrics set.
	Metrics []string `json:"metrics,omitempty"`
}

// Visible reports the course's default public visibility.
func (p PublicStats) Visible() bool { return p.DefaultVisible != nil && *p.DefaultVisible }

// Coach is the course's coach persona and policy (t5 §9).
type Coach struct {
	// Persona is the course-specific persona text (≤ MaxPersonaChars).
	Persona         string `json:"persona"`
	PrimaryLanguage string `json:"primary_language"`
	// OffDuring lists contexts (touch | mock) in which the coach is off.
	OffDuring []string `json:"off_during"`
}

// BandForLevel returns the band covering ladder level (1-based), if any.
func (r Revision) BandForLevel(level int) (Band, bool) {
	for _, b := range r.Bands {
		for _, l := range b.Levels {
			if l == level {
				return b, true
			}
		}
	}
	return Band{}, false
}

// Criterion returns the band's criterion with the given key, if any.
func (b Band) Criterion(key string) (Criterion, bool) {
	for _, c := range b.Criteria {
		if c.Key == key {
			return c, true
		}
	}
	return Criterion{}, false
}

// GradeIDs returns the manifest's grade ids in order.
func (m *Manifest) GradeIDs() []string {
	out := make([]string, len(m.Grades))
	for i, g := range m.Grades {
		out[i] = g.ID
	}
	return out
}

// CategoryIDs returns the manifest's mistake category ids in order (nil without a
// mistakes block).
func (m *Manifest) CategoryIDs() []string {
	if m.Mistakes == nil {
		return nil
	}
	out := make([]string, len(m.Mistakes.Categories))
	for i, c := range m.Mistakes.Categories {
		out[i] = c.ID
	}
	return out
}

// DimensionIDs returns the mock rubric's dimension ids in order (nil without a mock).
func (m *Manifest) DimensionIDs() []string {
	if m.Mock == nil {
		return nil
	}
	out := make([]string, len(m.Mock.Rubric.Dims))
	for i, d := range m.Mock.Rubric.Dims {
		out[i] = d.ID
	}
	return out
}

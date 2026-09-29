package course

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

var (
	// slugRe is the course slug shape (m1-09 adds the reserved-segment list).
	slugRe = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	// idPrefixRe is a course's item-id prefix (DSA's prefix is recorded but its ids stay bare).
	idPrefixRe = regexp.MustCompile(`^[a-z]{2,4}$`)
	// identRe is a snake_case identifier: categories, criteria keys, band formats, dims.
	identRe = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	// signalRe is a judge/practice signal, optionally with one qualifier (dim_low:<dim>).
	signalRe = regexp.MustCompile(`^[a-z][a-z0-9_]*(:[a-z0-9_]+)?$`)
	// versionedRefRe is an `<id>@<v>` ref (rubrics, harnesses, assets, pools).
	versionedRefRe = regexp.MustCompile(`^[a-z0-9][a-z0-9/_-]*@[1-9][0-9]*$`)
	// partIDRe is an item part id (also used by band.parts).
	partIDRe = regexp.MustCompile(`^[a-z][a-z0-9_-]*$`)
)

// verr accumulates validation errors with a field path.
type verr struct{ errs []error }

func (v *verr) add(path, format string, args ...any) {
	v.errs = append(v.errs, fmt.Errorf("%s: %s", path, fmt.Sprintf(format, args...)))
}

func (v *verr) err() error { return errors.Join(v.errs...) }

// Validate checks one manifest against the rules in docs/v2/sprints/sprint-m1-01.md
// (task 2, Validation). Load runs it; rules spanning manifests (unique id_prefix, the
// paths.json consistency check) run in Load.
func (m *Manifest) Validate() error {
	v := &verr{}
	if m.Format != ManifestFormat {
		v.add("format", "must be %d, got %d", ManifestFormat, m.Format)
	}
	if !slugRe.MatchString(m.Slug) {
		v.add("slug", "%q must match %s", m.Slug, slugRe)
	}
	if strings.TrimSpace(m.Title) == "" {
		v.add("title", "is required")
	}
	if !contains(ManifestStatuses, m.Status) {
		v.add("status", "%q is not one of %v", m.Status, ManifestStatuses)
	}
	if !idPrefixRe.MatchString(m.IDPrefix) {
		v.add("id_prefix", "%q must match %s", m.IDPrefix, idPrefixRe)
	}
	if m.PublicStats.DefaultVisible == nil {
		v.add("public_stats.default_visible", "is required")
	}
	m.validateMetrics(v)

	// active and preview courses carry the whole method; coming_soon and retired may
	// omit it (but whatever they do carry is still validated).
	if m.Status == StatusActive || m.Status == StatusPreview {
		for _, req := range []struct {
			name    string
			missing bool
		}{
			{"nav", m.Nav == nil},
			{"stages", m.Stages == nil},
			{"grading", m.Grading == nil},
			{"grades", len(m.Grades) == 0},
			{"revision", m.Revision == nil},
			{"mistakes", m.Mistakes == nil},
			{"plan", m.Plan == nil},
			{"coach", m.Coach == nil},
			{"public_stats.metrics", len(m.PublicStats.Metrics) == 0},
		} {
			if req.missing {
				v.add(req.name, "is required for a %s course", m.Status)
			}
		}
	}

	if m.Nav != nil {
		m.validateNav(v)
	}
	if m.Stages != nil {
		m.validateStages(v)
	}
	if m.Grading != nil {
		m.validateGrading(v)
	}
	if len(m.Grades) > 0 {
		m.validateGrades(v)
	}
	if m.Revision != nil {
		m.validateRevision(v)
	}
	if m.Mistakes != nil {
		m.validateMistakes(v)
	}
	if m.Mock != nil {
		m.validateMock(v)
	}
	if m.Plan != nil {
		m.validatePlan(v)
	}
	if m.Coach != nil {
		m.validateCoach(v)
	}
	return v.err()
}

func (m *Manifest) validateMetrics(v *verr) {
	seen := map[string]bool{}
	for i, mt := range m.PublicStats.Metrics {
		p := fmt.Sprintf("public_stats.metrics[%d]", i)
		if !contains(PublicMetrics, mt) {
			v.add(p, "%q is not one of %v", mt, PublicMetrics)
		}
		if seen[mt] {
			v.add(p, "duplicate metric %q", mt)
		}
		seen[mt] = true
	}
}

func (m *Manifest) validateNav(v *verr) {
	n := m.Nav
	if strings.TrimSpace(n.ItemNoun) == "" {
		v.add("nav.item_noun", "is required")
	}
	if len(n.Groups) == 0 {
		v.add("nav.groups", "needs at least one group")
	}
	seen := map[string]bool{}
	for gi, g := range n.Groups {
		if len(g.Items) == 0 {
			v.add(fmt.Sprintf("nav.groups[%d].items", gi), "needs at least one item")
		}
		for ii, it := range g.Items {
			p := fmt.Sprintf("nav.groups[%d].items[%d]", gi, ii)
			if !contains(NavScreens, it.Screen) {
				v.add(p+".screen", "%q is not one of %v", it.Screen, NavScreens)
			}
			if seen[it.Screen] {
				v.add(p+".screen", "duplicate screen %q", it.Screen)
			}
			seen[it.Screen] = true
			if strings.TrimSpace(it.Label) == "" {
				v.add(p+".label", "is required")
			}
		}
	}
}

func (m *Manifest) validateStages(v *verr) {
	s := m.Stages
	for _, st := range []struct {
		path string
		ts   TimedStage
	}{{"stages.attempt", s.Attempt}, {"stages.hint", s.Hint}} {
		if st.ts.DurationS <= 0 {
			v.add(st.path+".duration_s", "must be > 0")
		}
		if strings.TrimSpace(st.ts.Label) == "" {
			v.add(st.path+".label", "is required")
		}
	}
	if strings.TrimSpace(s.Solution.Label) == "" {
		v.add("stages.solution.label", "is required")
	}
	if !contains(ReimplementPolicies, s.Reimplement) {
		v.add("stages.reimplement", "%q is not one of %v", s.Reimplement, ReimplementPolicies)
	}
	if s.EarlyRevealPenaltyDays <= 0 {
		v.add("stages.early_reveal_penalty_days", "must be > 0")
	}
}

func (m *Manifest) validateGrading(v *verr) {
	g := m.Grading
	if !contains(Strategies, g.Strategy) {
		v.add("grading.strategy", "%q is not one of %v", g.Strategy, Strategies)
	}
	for _, sr := range []struct{ path, val string }{
		{"grading.self_report.outcome", g.SelfReport.Outcome},
		{"grading.self_report.touch", g.SelfReport.Touch},
		{"grading.self_report.mock", g.SelfReport.Mock},
	} {
		if !contains(SelfReportPolicies, sr.val) {
			v.add(sr.path, "%q is not one of %v", sr.val, SelfReportPolicies)
		}
	}

	p := g.Params
	// The selected strategy must carry its thresholds. Params for any other strategy are
	// dormant but still validated, so a later sprint that selects them finds them sound.
	selected := map[string]bool{
		StrategySelf:         p.Self != nil,
		StrategyVerdictTimer: p.VerdictTimer != nil,
		StrategyRubricPct:    p.RubricPct != nil,
		StrategyWeightedGate: p.WeightedGate != nil,
	}
	if has, known := selected[g.Strategy]; known && !has {
		v.add("grading.params", "missing params for the selected strategy %q", g.Strategy)
	}
	if p.Self != nil && !contains(SelfCaps, p.Self.Cap) {
		v.add(`grading.params["self@1"].cap`, "%q is not one of %v", p.Self.Cap, SelfCaps)
	}
	if vt := p.VerdictTimer; vt != nil {
		base := `grading.params["verdict_timer@1"]`
		for _, d := range []struct {
			name string
			val  int
		}{{"time_limit_s", vt.TimeLimitS}, {"hint_at_s", vt.HintAtS}, {"clean_within_s", vt.CleanWithinS}} {
			if d.val <= 0 {
				v.add(base+"."+d.name, "must be > 0")
			}
		}
		if vt.HintAtS >= vt.TimeLimitS || vt.CleanWithinS > vt.TimeLimitS {
			v.add(base, "hint_at_s and clean_within_s must fall inside time_limit_s")
		}
		if vt.MaxFailedForClean < 0 {
			v.add(base+".max_failed_for_clean", "must be >= 0")
		}
		for i, c := range vt.FreeClasses {
			if !contains(VerdictClasses, c) {
				v.add(fmt.Sprintf("%s.free_classes[%d]", base, i), "%q is not one of %v", c, VerdictClasses)
			}
		}
	}
	for _, pp := range []struct {
		path string
		p    *PctParams
	}{{`grading.params["rubric_pct@1"]`, p.RubricPct}, {`grading.params["weighted_gate@1"]`, p.WeightedGate}} {
		if pp.p == nil {
			continue
		}
		if pp.p.CleanPct <= 0 || pp.p.CleanPct > 1 || pp.p.PassPct <= 0 || pp.p.PassPct > 1 {
			v.add(pp.path, "clean_pct and pass_pct must be in (0, 1]")
		}
		if pp.p.PassPct > pp.p.CleanPct {
			v.add(pp.path, "pass_pct must be <= clean_pct")
		}
	}
}

func (m *Manifest) validateGrades(v *verr) {
	if got := m.GradeIDs(); !equalStrings(got, Grades) {
		v.add("grades", "ids %v must equal the universal grades %v, in order", got, Grades)
	}
	for i, g := range m.Grades {
		if strings.TrimSpace(g.Label) == "" {
			v.add(fmt.Sprintf("grades[%d].label", i), "is required")
		}
		if strings.TrimSpace(g.Hint) == "" {
			v.add(fmt.Sprintf("grades[%d].hint", i), "is required")
		}
	}
}

func (m *Manifest) validateRevision(v *verr) {
	r := m.Revision
	if !equalInts(r.LadderDays, LadderDays) {
		v.add("revision.ladder_days", "%v must equal the universal ladder %v", r.LadderDays, LadderDays)
	}
	if len(r.Bands) == 0 {
		v.add("revision.bands", "needs at least one band")
	}
	covered := map[int]int{}
	formats := map[string]bool{}
	for bi, b := range r.Bands {
		p := fmt.Sprintf("revision.bands[%d]", bi)
		if len(b.Levels) == 0 {
			v.add(p+".levels", "needs at least one level")
		}
		for _, l := range b.Levels {
			if l < 1 || l > len(LadderDays) {
				v.add(p+".levels", "level %d is outside 1..%d", l, len(LadderDays))
				continue
			}
			covered[l]++
		}
		if !identRe.MatchString(b.Format) {
			v.add(p+".format", "%q must match %s", b.Format, identRe)
		}
		formats[b.Format] = true
		if strings.TrimSpace(b.Label) == "" {
			v.add(p+".label", "is required")
		}
		if b.TimerS <= 0 {
			v.add(p+".timer_s", "must be > 0")
		}
		for pi, part := range b.Parts {
			if !partIDRe.MatchString(part) {
				v.add(fmt.Sprintf("%s.parts[%d]", p, pi), "%q must match %s", part, partIDRe)
			}
		}
		if len(b.Criteria) == 0 {
			v.add(p+".criteria", "needs at least one criterion")
		}
		keys := map[string]bool{}
		for ci, c := range b.Criteria {
			cp := fmt.Sprintf("%s.criteria[%d]", p, ci)
			if !identRe.MatchString(c.Key) {
				v.add(cp+".key", "%q must match %s (a criterion identifier, never a value)", c.Key, identRe)
			}
			if keys[c.Key] {
				v.add(cp+".key", "duplicate criterion %q", c.Key)
			}
			keys[c.Key] = true
			if c.ThresholdS < 0 {
				v.add(cp+".threshold_s", "must be > 0 when set")
			}
		}
	}
	for l := 1; l <= len(LadderDays); l++ {
		if covered[l] != 1 {
			v.add("revision.bands", "ladder level %d must be covered by exactly one band (got %d)", l, covered[l])
		}
	}
	// Every band format needs its minutes in plan.est_minutes (the single source).
	if m.Plan != nil {
		for f := range formats {
			if _, ok := m.Plan.EstMinutes[EstTouchPrefix+f]; !ok {
				v.add("plan.est_minutes", "missing %q for band format %q", EstTouchPrefix+f, f)
			}
		}
	}
}

func (m *Manifest) validateMistakes(v *verr) {
	mk := m.Mistakes
	ids := map[string]bool{}
	for i, c := range mk.Categories {
		p := fmt.Sprintf("mistakes.categories[%d]", i)
		if !identRe.MatchString(c.ID) {
			v.add(p+".id", "%q must match %s", c.ID, identRe)
		}
		if ids[c.ID] {
			v.add(p+".id", "duplicate category %q", c.ID)
		}
		ids[c.ID] = true
		if strings.TrimSpace(c.Label) == "" {
			v.add(p+".label", "is required")
		}
	}
	for _, core := range CoreMistakeCategories {
		if !ids[core] {
			v.add("mistakes.categories", "missing core category %q", core)
		}
	}
	for i, r := range mk.Prefill {
		p := fmt.Sprintf("mistakes.prefill[%d]", i)
		if !signalRe.MatchString(r.Signal) {
			v.add(p+".signal", "%q must match %s", r.Signal, signalRe)
		}
		if !ids[r.Category] {
			v.add(p+".category", "%q is not a declared category", r.Category)
		}
		if !contains(PrefillStrengths, r.Strength) {
			v.add(p+".strength", "%q is not one of %v", r.Strength, PrefillStrengths)
		}
	}
}

func (m *Manifest) validateMock(v *verr) {
	mk := m.Mock
	if mk.DurationS <= 0 {
		v.add("mock.duration_s", "must be > 0")
	}
	// The rail tiles [0, duration] with contiguous phases.
	if len(mk.Rail) == 0 {
		v.add("mock.rail", "needs at least one phase")
	}
	prevEnd := 0
	for i, ph := range mk.Rail {
		p := fmt.Sprintf("mock.rail[%d]", i)
		if strings.TrimSpace(ph.Label) == "" {
			v.add(p+".label", "is required")
		}
		if strings.TrimSpace(ph.Prompt) == "" {
			v.add(p+".prompt", "is required")
		}
		if ph.StartMin != prevEnd {
			v.add(p+".start_min", "%d must equal the previous phase's end (%d)", ph.StartMin, prevEnd)
		}
		if ph.EndMin <= ph.StartMin {
			v.add(p+".end_min", "must be > start_min")
		}
		prevEnd = ph.EndMin
	}
	if len(mk.Rail) > 0 && prevEnd*60 != mk.DurationS {
		v.add("mock.rail", "the last phase must end at duration_s (%d min), got %d min", mk.DurationS/60, prevEnd)
	}

	rb := mk.Rubric
	if !versionedRefRe.MatchString(rb.ID) {
		v.add("mock.rubric.id", "%q must match %s", rb.ID, versionedRefRe)
	}
	if n := len(rb.Dims); n < 1 || n > 10 {
		v.add("mock.rubric.dims", "must have 1..10 dimensions, got %d", n)
	}
	dims := map[string]bool{}
	for i, d := range rb.Dims {
		p := fmt.Sprintf("mock.rubric.dims[%d]", i)
		if !identRe.MatchString(d.ID) {
			v.add(p+".id", "%q must match %s", d.ID, identRe)
		}
		if dims[d.ID] {
			v.add(p+".id", "duplicate dimension %q", d.ID)
		}
		dims[d.ID] = true
		if strings.TrimSpace(d.Label) == "" {
			v.add(p+".label", "is required")
		}
	}
	if !equalInts(rb.Scale, []int{1, 5}) {
		v.add("mock.rubric.scale", "%v must be [1, 5]", rb.Scale)
	}
	maxTotal := rb.MaxTotal()
	for _, t := range []struct {
		name string
		val  int
	}{{"w13", mk.Targets.W13}, {"w15", mk.Targets.W15}, {"pre", mk.Targets.Pre}} {
		if t.val <= 0 || (maxTotal > 0 && t.val > maxTotal) {
			v.add("mock.targets."+t.name, "%d must be in (0, %d]", t.val, maxTotal)
		}
	}
	for i, pool := range mk.Pools {
		if !versionedRefRe.MatchString(pool) {
			v.add(fmt.Sprintf("mock.pools[%d]", i), "%q must match %s", pool, versionedRefRe)
		}
	}
	for i, h := range mk.EvidenceHints {
		p := fmt.Sprintf("mock.evidence_hints[%d]", i)
		if !signalRe.MatchString(h.Signal) {
			v.add(p+".signal", "%q must match %s", h.Signal, signalRe)
		}
		if !dims[h.Dim] {
			v.add(p+".dim", "%q is not a rubric dimension", h.Dim)
		}
		if len(rb.Scale) == 2 && (h.Max < rb.Scale[0] || h.Max > rb.Scale[1]) {
			v.add(p+".max", "%d is outside the rubric scale %v", h.Max, rb.Scale)
		}
	}
	if m.Plan != nil {
		if _, ok := m.Plan.EstMinutes[EstMock]; !ok {
			v.add("plan.est_minutes", "missing %q (the course has a mock)", EstMock)
		}
	}
}

func (m *Manifest) validatePlan(v *verr) {
	est := m.Plan.EstMinutes
	if _, ok := est[EstCourseAttempt]; !ok {
		v.add("plan.est_minutes", "missing %q", EstCourseAttempt)
	}
	for k, minutes := range est {
		ok := k == EstCourseAttempt || k == EstMock ||
			(strings.HasPrefix(k, EstTouchPrefix) && identRe.MatchString(strings.TrimPrefix(k, EstTouchPrefix)))
		if !ok {
			v.add("plan.est_minutes", "key %q must be %q, %q or %q<band format>", k, EstCourseAttempt, EstMock, EstTouchPrefix)
		}
		if minutes <= 0 {
			v.add(fmt.Sprintf("plan.est_minutes[%q]", k), "must be > 0")
		}
	}
}

func (m *Manifest) validateCoach(v *verr) {
	c := m.Coach
	if strings.TrimSpace(c.Persona) == "" {
		v.add("coach.persona", "is required")
	}
	if n := utf8.RuneCountInString(c.Persona); n > MaxPersonaChars {
		v.add("coach.persona", "is %d chars, max %d", n, MaxPersonaChars)
	}
	if !identRe.MatchString(c.PrimaryLanguage) {
		v.add("coach.primary_language", "%q must match %s", c.PrimaryLanguage, identRe)
	}
	seen := map[string]bool{}
	for i, o := range c.OffDuring {
		p := fmt.Sprintf("coach.off_during[%d]", i)
		if !contains(CoachOffDuring, o) {
			v.add(p, "%q is not one of %v", o, CoachOffDuring)
		}
		if seen[o] {
			v.add(p, "duplicate %q", o)
		}
		seen[o] = true
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

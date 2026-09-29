package course

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

// Item is one public item, `curriculum/courses/<slug>/items/<id>/item.json` — item
// schema v1, FROZEN at ev-schema-freeze (curriculum/_schema/item.schema.json;
// internal/course/testdata/item.schema.v1.frozen.json). After the freeze, changes are
// additive and optional only (schema_test.go's freeze guard enforces it), so authored
// items never have to be re-stamped.
//
// Answer secrecy (ADR-0027 §1): no field can hold an answer. Hidden cases, keys that
// cannot be derived from public text, rubric anchors and must-cover lists live in the
// private eval pack, which only judge reads. schema_test.go walks these types and the
// schema against an answer-name denylist.
//
// Sections (statement, hints, editorial) are Markdown sidecar files next to item.json,
// not fields: `sections/<stage>/<NN>-<kind>.md` and code under `_code/` (m1-09).
// An item with no parts is valid: that is the v1 self path.
type Item struct {
	ID         string `json:"id"`
	Course     string `json:"course"`
	WeekN      int    `json:"week_n"`
	SortOrder  int    `json:"sort_order"`
	Title      string `json:"title"`
	Difficulty string `json:"difficulty"`
	// Pattern is the item's pattern/topic (public, withheld while the item is live).
	Pattern string `json:"pattern"`
	Role    string `json:"role"`
	Status  string `json:"status"`

	Provenance Provenance   `json:"provenance"`
	Links      []Link       `json:"links,omitempty"`
	Review     *ReviewStamp `json:"review,omitempty"`
	// Concepts are ≤ 3 refs `<course>:concept:<slug>`, primary first (ADR-0029 §4,
	// "concepts to revise").
	Concepts []string `json:"concepts,omitempty"`

	Parts  []Part       `json:"parts,omitempty"`
	Grader []GraderStep `json:"grader,omitempty"`
	// SolutionFacts are public facts served only with the solution stage.
	SolutionFacts *SolutionFacts `json:"solution_facts,omitempty"`
	Revision      *ItemRevision  `json:"revision,omitempty"`
	// Assets are `id@v` refs to course assets.
	Assets []string `json:"assets,omitempty"`
}

// Provenance is required on every item (t1 §8). Origin `copied` is rejected.
type Provenance struct {
	Origin      string   `json:"origin"`
	InspiredBy  []string `json:"inspired_by,omitempty"`
	License     string   `json:"license,omitempty"`
	Attribution string   `json:"attribution,omitempty"`
	AuthoredBy  string   `json:"authored_by"`
}

// Link is an outbound reference link (https only).
type Link struct {
	Kind string `json:"kind"`
	URL  string `json:"url"`
}

// ReviewStamp records when the owner reviewed each public section kind (ISO dates).
// Hints and editorial are served only once stamped.
type ReviewStamp struct {
	Statement string `json:"statement,omitempty"`
	Hints     string `json:"hints,omitempty"`
	Editorial string `json:"editorial,omitempty"`
}

// Part is a typed answer slot (ADR-0026 §2): it selects a widget, a grader step and a
// cadence.
type Part struct {
	ID string `json:"id"`
	// Type is code | text | choice | blank | canvas (audio is reserved: it is added,
	// additively, when its widget ships).
	Type string `json:"type"`
	// Cadence is iterate (run/submit many times) | final (evaluated once).
	Cadence string `json:"cadence"`
	// Grading is auto | ai | none (none = ungraded, the self path).
	Grading  string     `json:"grading"`
	Required bool       `json:"required"`
	Config   PartConfig `json:"config"`
}

// PartConfig is the part's configuration. It is one shape for every part type; Validate
// allows only the fields that belong to the part's type. No field marks a correct
// option or holds an expected output beyond a public sample.
type PartConfig struct {
	// code
	Signature   *Signature   `json:"signature,omitempty"`
	Languages   []string     `json:"languages,omitempty"`
	Harness     string       `json:"harness,omitempty"`
	Checker     *Checker     `json:"checker,omitempty"`
	Constraints []Constraint `json:"constraints,omitempty"`
	Samples     []Sample     `json:"samples,omitempty"`
	Limits      *Limits      `json:"limits,omitempty"`
	// choice
	Mode    string   `json:"mode,omitempty"`
	Options []Option `json:"options,omitempty"`
	// blank and text
	Fields []Field `json:"fields,omitempty"`
	// text
	MaxChars int `json:"max_chars,omitempty"`
	// canvas
	Palette  string `json:"palette,omitempty"`
	MaxNodes int    `json:"max_nodes,omitempty"`
}

// Signature is a code part's entry point: a function, or a class driven by ops.
type Signature struct {
	// Mode is function | class.
	Mode    string  `json:"mode"`
	Name    string  `json:"name"`
	Params  []Param `json:"params,omitempty"`
	Returns string  `json:"returns,omitempty"`
	// Ops are a class's methods (class mode only).
	Ops []Op `json:"ops,omitempty"`
}

// Param is a named, typed parameter. Types come from the closed ValueTypes registry.
type Param struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// Op is one class method.
type Op struct {
	Name    string  `json:"name"`
	Params  []Param `json:"params,omitempty"`
	Returns string  `json:"returns,omitempty"`
}

// Checker names a checker from the closed checker registry, with its parameters.
type Checker struct {
	Name   string         `json:"name"`
	Params *CheckerParams `json:"params,omitempty"`
}

// CheckerParams are checker parameters (float_abs / float_rel take eps).
type CheckerParams struct {
	Eps float64 `json:"eps,omitempty"`
}

// Constraint is one public input constraint; the generic validator derives from these.
type Constraint struct {
	// Arg names an argument, or its elements with `[*]` (e.g. nums[*]).
	Arg   string    `json:"arg"`
	Len   []int     `json:"len,omitempty"`
	Range []float64 `json:"range,omitempty"`
}

// Sample is a PUBLIC example shown with the statement. Its Expected output is public
// by definition (the statement shows it); hidden cases live in the private pack.
type Sample struct {
	ID string `json:"id"`
	// Ops is the op sequence for class mode (first op constructs).
	Ops      []string          `json:"ops,omitempty"`
	Args     []json.RawMessage `json:"args"`
	Expected json.RawMessage   `json:"expected"`
}

// Limits are the per-case resource limits.
type Limits struct {
	TimeMS   int `json:"time_ms"`
	MemoryMB int `json:"memory_mb"`
}

// Option is one choice option. It has no correctness flag: keys live in the pack.
type Option struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// Field is one named input of a blank or text part.
type Field struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	// Kind is a blank field's kind: text | numeric | big_o.
	Kind string `json:"kind,omitempty"`
	// Units a numeric blank field accepts.
	Units []string `json:"units,omitempty"`
	// MaxChars bounds a text field.
	MaxChars int `json:"max_chars,omitempty"`
}

// GraderStep is one step of the item's single, non-nested composite@1 root.
type GraderStep struct {
	Step string `json:"step"`
	// Kind is code | key | ai_rubric (the analyzer is never authored).
	Kind     string   `json:"kind"`
	Inputs   []string `json:"inputs"`
	Required bool     `json:"required"`
	Weight   float64  `json:"weight,omitempty"`
	// Rubric is the public rubric ref (`id@v`) of an ai_rubric step.
	Rubric string `json:"rubric,omitempty"`
}

// SolutionFacts are public facts about the reference, served only with the solution
// stage (and withheld while the item is live).
type SolutionFacts struct {
	Complexity Complexity `json:"complexity"`
}

// Complexity lists every accepted time and space complexity.
type Complexity struct {
	Time  []string `json:"time"`
	Space []string `json:"space"`
}

// ItemRevision holds the item's revision probes.
type ItemRevision struct {
	Probes []Probe `json:"probes"`
}

// Probe is a short keyed check a touch runs for one criterion. KeySource names WHERE
// the key lives — `public:<field>` (honor-grade, derived from public text) or `pack`
// (a private key only judge holds) — and is never the key itself.
type Probe struct {
	ID string `json:"id"`
	// Type is text | blank | choice.
	Type string `json:"type"`
	// Band is the manifest band format the probe belongs to.
	Band string `json:"band"`
	// Grading is always "key".
	Grading   string `json:"grading"`
	KeySource string `json:"key_source"`
	// Criterion is the band criterion this probe decides.
	Criterion string       `json:"criterion"`
	TimerS    int          `json:"timer_s,omitempty"`
	PromptMD  string       `json:"prompt_md"`
	Config    *ProbeConfig `json:"config,omitempty"`
}

// ProbeConfig is a probe's configuration: the choice, blank and text fields of
// PartConfig only (a probe never carries code, samples or limits).
type ProbeConfig struct {
	Mode     string   `json:"mode,omitempty"`
	Options  []Option `json:"options,omitempty"`
	Fields   []Field  `json:"fields,omitempty"`
	MaxChars int      `json:"max_chars,omitempty"`
}

func (c ProbeConfig) asPart() PartConfig {
	return PartConfig{Mode: c.Mode, Options: c.Options, Fields: c.Fields, MaxChars: c.MaxChars}
}

// Item enums.
var (
	Difficulties    = []string{"easy", "med", "hard"}
	Roles           = []string{"core", "reinforcement", "drill"}
	ItemStatuses    = []string{"live", "retired", "withdrawn"}
	Origins         = []string{"original", "adapted", "licensed"}
	LinkKinds       = []string{"leetcode", "neetcode", "other"}
	PartTypes       = []string{"code", "text", "choice", "blank", "canvas"}
	Cadences        = []string{"iterate", "final"}
	PartGradings    = []string{"auto", "ai", "none"}
	SignatureModes  = []string{"function", "class"}
	ChoiceModes     = []string{"single", "multi", "ordering"}
	BlankKinds      = []string{"text", "numeric", "big_o"}
	GraderKinds     = []string{"code", "key", "ai_rubric"}
	ProbeTypes      = []string{"text", "blank", "choice"}
	ProbeGrading    = "key"
	PublicKeyFields = []string{
		"pattern", "solution_facts.complexity", "solution_facts.complexity.time", "solution_facts.complexity.space",
	}
)

// ValueTypes is the closed base-type registry for signatures (t1 §7.1). Any base type
// may carry `[]` suffixes (T[], T[][]). A new type is code plus a release.
var ValueTypes = []string{"int", "int64", "float64", "bool", "string", "ListNode", "TreeNode", "GraphNode"}

var (
	// dsaIDRe: DSA ids stay bare (m1-09's id guard).
	dsaIDRe = regexp.MustCompile(`^[1-9][0-9]{0,2}$`)
	// prefixedIDRe: every other course uses `<id_prefix>-NNN`.
	prefixedIDRe = regexp.MustCompile(`^([a-z]{2,4})-[0-9]{3}$`)
	// conceptRefRe: `<course>:concept:<slug>`.
	conceptRefRe  = regexp.MustCompile(`^([a-z0-9]+(?:-[a-z0-9]+)*):concept:[a-z0-9]+(?:-[a-z0-9]+)*$`)
	inspiredByRe  = regexp.MustCompile(`^[a-z][a-z0-9-]*:[^\s]+$`)
	dateRe        = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}$`)
	keySourceRe   = regexp.MustCompile(`^(public:[a-z_.]+|pack)$`)
	goIdentRe     = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	languageRe    = regexp.MustCompile(`^[a-z][a-z0-9]*$`)
	checkerNameRe = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	valueTypeRe   = regexp.MustCompile(`^([A-Za-z][A-Za-z0-9]*)((?:\[\])*)$`)
)

// MaxConcepts bounds item.concepts.
const MaxConcepts = 3

// Validate checks an item on its own. ValidateFor adds the checks against its course's
// manifest (id prefix, probe bands and criteria).
func (it *Item) Validate() error {
	v := &verr{}
	it.validateIdentity(v)
	it.validateProvenance(v)
	for i, l := range it.Links {
		p := fmt.Sprintf("links[%d]", i)
		if !contains(LinkKinds, l.Kind) {
			v.add(p+".kind", "%q is not one of %v", l.Kind, LinkKinds)
		}
		if !isHTTPS(l.URL) {
			v.add(p+".url", "%q must be an https:// URL", l.URL)
		}
	}
	if r := it.Review; r != nil {
		for _, s := range []struct{ name, val string }{
			{"statement", r.Statement}, {"hints", r.Hints}, {"editorial", r.Editorial},
		} {
			if s.val != "" && !isDate(s.val) {
				v.add("review."+s.name, "%q must be an ISO date (YYYY-MM-DD)", s.val)
			}
		}
	}
	it.validateConcepts(v)
	parts := it.validateParts(v)
	it.validateGrader(v, parts)
	if sf := it.SolutionFacts; sf != nil {
		if len(sf.Complexity.Time) == 0 || len(sf.Complexity.Space) == 0 {
			v.add("solution_facts.complexity", "needs at least one time and one space complexity")
		}
		for i, c := range append(append([]string{}, sf.Complexity.Time...), sf.Complexity.Space...) {
			if strings.TrimSpace(c) == "" || utf8.RuneCountInString(c) > 40 {
				v.add("solution_facts.complexity", "entry %d must be 1..40 chars", i)
			}
		}
	}
	it.validateProbes(v)
	seen := map[string]bool{}
	for i, a := range it.Assets {
		if !versionedRefRe.MatchString(a) {
			v.add(fmt.Sprintf("assets[%d]", i), "%q must be an id@v ref", a)
		}
		if seen[a] {
			v.add(fmt.Sprintf("assets[%d]", i), "duplicate asset %q", a)
		}
		seen[a] = true
	}
	return v.err()
}

// ValidateFor validates the item and checks it against its course's manifest.
func (it *Item) ValidateFor(m *Manifest) error {
	v := &verr{}
	if err := it.Validate(); err != nil {
		v.errs = append(v.errs, err)
	}
	if it.Course != m.Slug {
		v.add("course", "%q does not match the manifest %q", it.Course, m.Slug)
	}
	if m.Slug != DSASlug {
		if sm := prefixedIDRe.FindStringSubmatch(it.ID); sm != nil && sm[1] != m.IDPrefix {
			v.add("id", "%q must use the course prefix %q", it.ID, m.IDPrefix)
		}
	}
	if it.Revision != nil {
		for i, pr := range it.Revision.Probes {
			p := fmt.Sprintf("revision.probes[%d]", i)
			var band *Band
			if m.Revision != nil {
				for bi := range m.Revision.Bands {
					if m.Revision.Bands[bi].Format == pr.Band {
						band = &m.Revision.Bands[bi]
						break
					}
				}
			}
			if band == nil {
				v.add(p+".band", "%q is not a band format of course %q", pr.Band, m.Slug)
				continue
			}
			if _, ok := band.Criterion(pr.Criterion); !ok {
				v.add(p+".criterion", "%q is not a criterion of band %q", pr.Criterion, pr.Band)
			}
		}
	}
	return v.err()
}

func (it *Item) validateIdentity(v *verr) {
	if !slugRe.MatchString(it.Course) {
		v.add("course", "%q must match %s", it.Course, slugRe)
	}
	switch {
	case it.Course == DSASlug && !dsaIDRe.MatchString(it.ID):
		v.add("id", "%q: DSA ids are bare numbers matching %s", it.ID, dsaIDRe)
	case it.Course != DSASlug && !prefixedIDRe.MatchString(it.ID):
		v.add("id", "%q must match <id_prefix>-NNN (%s)", it.ID, prefixedIDRe)
	}
	if it.WeekN < 1 {
		v.add("week_n", "must be >= 1")
	}
	if it.SortOrder < 0 {
		v.add("sort_order", "must be >= 0")
	}
	if strings.TrimSpace(it.Title) == "" || utf8.RuneCountInString(it.Title) > 200 {
		v.add("title", "must be 1..200 chars")
	}
	if !contains(Difficulties, it.Difficulty) {
		v.add("difficulty", "%q is not one of %v", it.Difficulty, Difficulties)
	}
	if strings.TrimSpace(it.Pattern) == "" || utf8.RuneCountInString(it.Pattern) > 100 {
		v.add("pattern", "must be 1..100 chars")
	}
	if !contains(Roles, it.Role) {
		v.add("role", "%q is not one of %v", it.Role, Roles)
	}
	if !contains(ItemStatuses, it.Status) {
		v.add("status", "%q is not one of %v", it.Status, ItemStatuses)
	}
}

func (it *Item) validateProvenance(v *verr) {
	pv := it.Provenance
	switch {
	case pv.Origin == "copied":
		v.add("provenance.origin", `"copied" is rejected: write original expression (t1 §8)`)
	case !contains(Origins, pv.Origin):
		v.add("provenance.origin", "%q is not one of %v", pv.Origin, Origins)
	}
	if pv.Origin == "licensed" && strings.TrimSpace(pv.License) == "" {
		v.add("provenance.license", "is required when origin is licensed")
	}
	if strings.TrimSpace(pv.AuthoredBy) == "" {
		v.add("provenance.authored_by", "is required")
	}
	for i, s := range pv.InspiredBy {
		if !inspiredByRe.MatchString(s) {
			v.add(fmt.Sprintf("provenance.inspired_by[%d]", i), "%q must look like <source>:<ref>", s)
		}
	}
}

func (it *Item) validateConcepts(v *verr) {
	if len(it.Concepts) > MaxConcepts {
		v.add("concepts", "at most %d concepts, got %d", MaxConcepts, len(it.Concepts))
	}
	seen := map[string]bool{}
	for i, c := range it.Concepts {
		p := fmt.Sprintf("concepts[%d]", i)
		sm := conceptRefRe.FindStringSubmatch(c)
		if sm == nil {
			v.add(p, "%q must be <course>:concept:<slug>", c)
			continue
		}
		if sm[1] != it.Course {
			v.add(p, "%q must reference the item's own course %q", c, it.Course)
		}
		if seen[c] {
			v.add(p, "duplicate concept %q", c)
		}
		seen[c] = true
	}
}

// validateParts checks every part and returns the parts by id.
func (it *Item) validateParts(v *verr) map[string]Part {
	byID := make(map[string]Part, len(it.Parts))
	for i, pt := range it.Parts {
		p := fmt.Sprintf("parts[%d]", i)
		if !partIDRe.MatchString(pt.ID) {
			v.add(p+".id", "%q must match %s", pt.ID, partIDRe)
		}
		if _, dup := byID[pt.ID]; dup {
			v.add(p+".id", "duplicate part %q", pt.ID)
		}
		byID[pt.ID] = pt
		if !contains(PartTypes, pt.Type) {
			v.add(p+".type", "%q is not one of %v", pt.Type, PartTypes)
		}
		if !contains(Cadences, pt.Cadence) {
			v.add(p+".cadence", "%q is not one of %v", pt.Cadence, Cadences)
		}
		if !contains(PartGradings, pt.Grading) {
			v.add(p+".grading", "%q is not one of %v", pt.Grading, PartGradings)
		}
		validateConfig(v, p+".config", pt.Type, pt.Config)
	}
	return byID
}

// validateConfig allows only the config fields of the given type and checks them.
func validateConfig(v *verr, p, typ string, c PartConfig) {
	present := map[string]bool{
		"signature":   c.Signature != nil,
		"languages":   len(c.Languages) > 0,
		"harness":     c.Harness != "",
		"checker":     c.Checker != nil,
		"constraints": len(c.Constraints) > 0,
		"samples":     len(c.Samples) > 0,
		"limits":      c.Limits != nil,
		"mode":        c.Mode != "",
		"options":     len(c.Options) > 0,
		"fields":      len(c.Fields) > 0,
		"max_chars":   c.MaxChars != 0,
		"palette":     c.Palette != "",
		"max_nodes":   c.MaxNodes != 0,
	}
	allowed := map[string][]string{
		"code":   {"signature", "languages", "harness", "checker", "constraints", "samples", "limits"},
		"choice": {"mode", "options"},
		"blank":  {"fields"},
		"text":   {"fields", "max_chars"},
		"canvas": {"palette", "max_nodes"},
	}[typ]
	for _, name := range []string{
		"signature", "languages", "harness", "checker", "constraints", "samples", "limits",
		"mode", "options", "fields", "max_chars", "palette", "max_nodes",
	} {
		if present[name] && !contains(allowed, name) {
			v.add(p+"."+name, "is not allowed on a %q part", typ)
		}
	}

	switch typ {
	case "code":
		validateCodeConfig(v, p, c)
	case "choice":
		if c.Mode != "" && !contains(ChoiceModes, c.Mode) {
			v.add(p+".mode", "%q is not one of %v", c.Mode, ChoiceModes)
		}
		if len(c.Options) < 2 {
			v.add(p+".options", "needs at least 2 options")
		}
		ids := map[string]bool{}
		for i, o := range c.Options {
			op := fmt.Sprintf("%s.options[%d]", p, i)
			if !partIDRe.MatchString(o.ID) {
				v.add(op+".id", "%q must match %s", o.ID, partIDRe)
			}
			if ids[o.ID] {
				v.add(op+".id", "duplicate option %q", o.ID)
			}
			ids[o.ID] = true
			if strings.TrimSpace(o.Label) == "" {
				v.add(op+".label", "is required")
			}
		}
	case "blank", "text":
		if typ == "blank" && len(c.Fields) == 0 {
			v.add(p+".fields", "a blank part needs at least one field")
		}
		if c.MaxChars < 0 {
			v.add(p+".max_chars", "must be > 0 when set")
		}
		ids := map[string]bool{}
		for i, f := range c.Fields {
			fp := fmt.Sprintf("%s.fields[%d]", p, i)
			if !partIDRe.MatchString(f.ID) {
				v.add(fp+".id", "%q must match %s", f.ID, partIDRe)
			}
			if ids[f.ID] {
				v.add(fp+".id", "duplicate field %q", f.ID)
			}
			ids[f.ID] = true
			if strings.TrimSpace(f.Label) == "" {
				v.add(fp+".label", "is required")
			}
			if typ == "text" && (f.Kind != "" || len(f.Units) > 0) {
				v.add(fp, "kind and units belong to blank fields")
			}
			if typ == "blank" {
				if f.MaxChars != 0 {
					v.add(fp+".max_chars", "belongs to text fields")
				}
				if f.Kind != "" && !contains(BlankKinds, f.Kind) {
					v.add(fp+".kind", "%q is not one of %v", f.Kind, BlankKinds)
				}
				if len(f.Units) > 0 && f.Kind != "numeric" {
					v.add(fp+".units", "only a numeric field takes units")
				}
			}
			if f.MaxChars < 0 {
				v.add(fp+".max_chars", "must be > 0 when set")
			}
		}
	case "canvas":
		if !versionedRefRe.MatchString(c.Palette) {
			v.add(p+".palette", "%q must be an id@v ref", c.Palette)
		}
		if c.MaxNodes < 0 {
			v.add(p+".max_nodes", "must be > 0 when set")
		}
	}
}

func validateCodeConfig(v *verr, p string, c PartConfig) {
	if c.Signature == nil {
		v.add(p+".signature", "is required on a code part")
	} else {
		s := c.Signature
		if !contains(SignatureModes, s.Mode) {
			v.add(p+".signature.mode", "%q is not one of %v", s.Mode, SignatureModes)
		}
		if !goIdentRe.MatchString(s.Name) {
			v.add(p+".signature.name", "%q must be an identifier", s.Name)
		}
		validateParams(v, p+".signature.params", s.Params)
		if s.Returns != "" && !isValueType(s.Returns) {
			v.add(p+".signature.returns", "%q is not a registered type (%v, with [] suffixes)", s.Returns, ValueTypes)
		}
		switch s.Mode {
		case "function":
			if len(s.Ops) > 0 {
				v.add(p+".signature.ops", "belong to class mode")
			}
		case "class":
			if len(s.Ops) == 0 {
				v.add(p+".signature.ops", "class mode needs at least one op")
			}
		}
		for i, op := range s.Ops {
			opp := fmt.Sprintf("%s.signature.ops[%d]", p, i)
			if !goIdentRe.MatchString(op.Name) {
				v.add(opp+".name", "%q must be an identifier", op.Name)
			}
			validateParams(v, opp+".params", op.Params)
			if op.Returns != "" && !isValueType(op.Returns) {
				v.add(opp+".returns", "%q is not a registered type", op.Returns)
			}
		}
	}
	if len(c.Languages) == 0 {
		v.add(p+".languages", "needs at least one language")
	}
	for i, l := range c.Languages {
		if !languageRe.MatchString(l) {
			v.add(fmt.Sprintf("%s.languages[%d]", p, i), "%q must match %s", l, languageRe)
		}
	}
	if !versionedRefRe.MatchString(c.Harness) {
		v.add(p+".harness", "%q must be a name@v ref", c.Harness)
	}
	if c.Checker == nil || !checkerNameRe.MatchString(c.Checker.Name) {
		v.add(p+".checker.name", "is required and must match %s", checkerNameRe)
	} else if c.Checker.Params != nil && c.Checker.Params.Eps < 0 {
		v.add(p+".checker.params.eps", "must be > 0 when set")
	}
	for i, cn := range c.Constraints {
		cp := fmt.Sprintf("%s.constraints[%d]", p, i)
		if strings.TrimSpace(cn.Arg) == "" {
			v.add(cp+".arg", "is required")
		}
		if cn.Len != nil && (len(cn.Len) != 2 || cn.Len[0] < 0 || cn.Len[0] > cn.Len[1]) {
			v.add(cp+".len", "must be [min, max] with 0 <= min <= max")
		}
		if cn.Range != nil && (len(cn.Range) != 2 || cn.Range[0] > cn.Range[1]) {
			v.add(cp+".range", "must be [min, max] with min <= max")
		}
		if cn.Len == nil && cn.Range == nil {
			v.add(cp, "needs len or range")
		}
	}
	ids := map[string]bool{}
	for i, s := range c.Samples {
		sp := fmt.Sprintf("%s.samples[%d]", p, i)
		if !partIDRe.MatchString(s.ID) {
			v.add(sp+".id", "%q must match %s", s.ID, partIDRe)
		}
		if ids[s.ID] {
			v.add(sp+".id", "duplicate sample %q", s.ID)
		}
		ids[s.ID] = true
		if s.Args == nil {
			v.add(sp+".args", "is required")
		}
		if len(s.Expected) == 0 {
			v.add(sp+".expected", "is required")
		}
		if len(s.Ops) > 0 && (c.Signature == nil || c.Signature.Mode != "class") {
			v.add(sp+".ops", "belong to class mode")
		}
	}
	if c.Limits != nil && (c.Limits.TimeMS <= 0 || c.Limits.MemoryMB <= 0) {
		v.add(p+".limits", "time_ms and memory_mb must be > 0")
	}
}

func validateParams(v *verr, p string, ps []Param) {
	names := map[string]bool{}
	for i, pr := range ps {
		pp := fmt.Sprintf("%s[%d]", p, i)
		if !goIdentRe.MatchString(pr.Name) {
			v.add(pp+".name", "%q must be an identifier", pr.Name)
		}
		if names[pr.Name] {
			v.add(pp+".name", "duplicate param %q", pr.Name)
		}
		names[pr.Name] = true
		if !isValueType(pr.Type) {
			v.add(pp+".type", "%q is not a registered type (%v, with [] suffixes)", pr.Type, ValueTypes)
		}
	}
}

// validateGrader checks the composite@1 steps against the parts (t4 §5.6 lints 2–7).
func (it *Item) validateGrader(v *verr, parts map[string]Part) {
	if len(it.Grader) > 0 {
		anyRequired := false
		for _, s := range it.Grader {
			anyRequired = anyRequired || s.Required
		}
		if !anyRequired {
			v.add("grader", "at least one step must be required")
		}
	}
	steps := map[string]bool{}
	fed := map[string]bool{}
	aiRubrics := 0
	for i, s := range it.Grader {
		p := fmt.Sprintf("grader[%d]", i)
		if !partIDRe.MatchString(s.Step) {
			v.add(p+".step", "%q must match %s", s.Step, partIDRe)
		}
		if steps[s.Step] {
			v.add(p+".step", "duplicate step %q", s.Step)
		}
		steps[s.Step] = true
		if !contains(GraderKinds, s.Kind) {
			v.add(p+".kind", "%q is not one of %v", s.Kind, GraderKinds)
		}
		if s.Kind == "ai_rubric" {
			aiRubrics++
			if !versionedRefRe.MatchString(s.Rubric) {
				v.add(p+".rubric", "an ai_rubric step needs a rubric id@v ref")
			}
		} else if s.Rubric != "" {
			v.add(p+".rubric", "belongs to ai_rubric steps")
		}
		if s.Weight < 0 || s.Weight > 1 {
			v.add(p+".weight", "must be in (0, 1] when set")
		}
		if len(s.Inputs) == 0 {
			v.add(p+".inputs", "needs at least one part id")
		}
		for j, in := range s.Inputs {
			ip := fmt.Sprintf("%s.inputs[%d]", p, j)
			pt, ok := parts[in]
			if !ok {
				v.add(ip, "%q is not a part id", in)
				continue
			}
			fed[in] = true
			switch s.Kind {
			case "code":
				if pt.Type != "code" || pt.Cadence != "iterate" {
					v.add(ip, "a code step reads iterate code parts; %q is %s/%s", in, pt.Type, pt.Cadence)
				}
			case "key", "ai_rubric":
				if pt.Cadence != "final" {
					v.add(ip, "a %s step reads final parts; %q is %s", s.Kind, in, pt.Cadence)
				}
			}
		}
	}
	if aiRubrics > 1 {
		v.add("grader", "at most one ai_rubric step per item, got %d", aiRubrics)
	}
	for _, pt := range it.Parts {
		if pt.Grading != "none" && !fed[pt.ID] {
			v.add("parts", "part %q is graded %q but feeds no grader step (set grading none)", pt.ID, pt.Grading)
		}
	}
}

func (it *Item) validateProbes(v *verr) {
	if it.Revision == nil {
		return
	}
	ids := map[string]bool{}
	for i, pr := range it.Revision.Probes {
		p := fmt.Sprintf("revision.probes[%d]", i)
		if !partIDRe.MatchString(pr.ID) {
			v.add(p+".id", "%q must match %s", pr.ID, partIDRe)
		}
		if ids[pr.ID] {
			v.add(p+".id", "duplicate probe %q", pr.ID)
		}
		ids[pr.ID] = true
		if !contains(ProbeTypes, pr.Type) {
			v.add(p+".type", "%q is not one of %v", pr.Type, ProbeTypes)
		}
		if !identRe.MatchString(pr.Band) {
			v.add(p+".band", "%q must match %s", pr.Band, identRe)
		}
		if pr.Grading != ProbeGrading {
			v.add(p+".grading", "must be %q", ProbeGrading)
		}
		if !keySourceRe.MatchString(pr.KeySource) {
			v.add(p+".key_source", "%q must match %s: it names where a key lives, never the key", pr.KeySource, keySourceRe)
		} else if f, ok := strings.CutPrefix(pr.KeySource, "public:"); ok {
			if !it.resolvesPublic(f) {
				v.add(p+".key_source", "%q does not resolve to a public field of this item", pr.KeySource)
			}
		}
		if !identRe.MatchString(pr.Criterion) {
			v.add(p+".criterion", "%q must match %s", pr.Criterion, identRe)
		}
		if pr.TimerS < 0 {
			v.add(p+".timer_s", "must be > 0 when set")
		}
		if strings.TrimSpace(pr.PromptMD) == "" || utf8.RuneCountInString(pr.PromptMD) > 2000 {
			v.add(p+".prompt_md", "must be 1..2000 chars")
		}
		if pr.Config != nil && contains(ProbeTypes, pr.Type) {
			validateConfig(v, p+".config", pr.Type, pr.Config.asPart())
		}
	}
}

// resolvesPublic reports whether a `public:<field>` key source names a public field this
// item actually carries (t4 §5.6 lint 8).
func (it *Item) resolvesPublic(field string) bool {
	if !contains(PublicKeyFields, field) {
		return false
	}
	switch field {
	case "pattern":
		return strings.TrimSpace(it.Pattern) != ""
	default: // solution_facts.complexity[.time|.space]
		return it.SolutionFacts != nil
	}
}

func isValueType(t string) bool {
	sm := valueTypeRe.FindStringSubmatch(t)
	return sm != nil && contains(ValueTypes, sm[1])
}

func isHTTPS(s string) bool {
	u, err := url.Parse(s)
	return err == nil && u.Scheme == "https" && u.Host != "" && strings.HasPrefix(s, "https://")
}

func isDate(s string) bool {
	if !dateRe.MatchString(s) {
		return false
	}
	_, err := time.Parse("2006-01-02", s)
	return err == nil
}

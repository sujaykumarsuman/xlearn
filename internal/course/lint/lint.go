// Package lint holds the public content rules that more than one tool runs: the
// public-registry half of the t4 §5.6 structure lints (#1–#8, #12–#14), the hints and
// editorial stamp gate, the label-edit flag on key-graded parts, and packlint's advisory
// "content-only edit" diff. cmd/contentlint runs them in the public `content` CI job,
// cmd/packlint reuses them against the private pack, and judge re-runs the structure
// lints at start (t4 §5.6 "again at judge start", m3-05): one definition, never
// re-implemented.
//
// It is stdlib-only and imports internal/course and internal/course/canon and nothing
// else (no database, no service package, no git): callers hand it decoded items and
// manifests.
//
// Deferred: the Caps type match of #3 (judge's grader registry, m3-06), compiled
// starters and "the reference passes the samples" (m3-04), and #10, #11, #15 (m3-06,
// m3-09, p-01).
package lint

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/sujaykumarsuman/xlearn/internal/course"
	"github.com/sujaykumarsuman/xlearn/internal/course/canon"
)

// Finding is one lint result. Rule names the t4 §5.6 lint ("5.6#3").
type Finding struct {
	Rule string
	Item string // item id; "" for a manifest finding
	Path string // JSON field path
	Msg  string
}

func (f Finding) String() string {
	who := f.Item
	if who == "" {
		who = "manifest"
	}
	return fmt.Sprintf("[%s] %s %s: %s", f.Rule, who, f.Path, f.Msg)
}

type findings struct {
	item string
	out  []Finding
}

func (fs *findings) add(rule, path, format string, args ...any) {
	fs.out = append(fs.out, Finding{Rule: rule, Item: fs.item, Path: path, Msg: fmt.Sprintf(format, args...)})
}

// StepChecks are the band criteria a grader step decides by itself (t4 §6.6), mapped to
// the step kind that decides them: a touch in a band whose parts feed such a step meets
// the criterion from the step's verdict (e.g. correct_in_timer: a passing code submit
// within the band's timer). Every other criterion needs a probe (lint #14). A new step
// check is judge code plus an entry here.
var StepChecks = map[string]string{
	"correct_in_timer": "code",
}

// partConfigFields are the PartConfig fields each part type may set (lint #1: configs
// decode strictly for their type; mirrors course.Item.Validate).
var partConfigFields = map[string][]string{
	"code":   {"signature", "languages", "harness", "checker", "constraints", "samples", "limits"},
	"choice": {"mode", "options"},
	"blank":  {"fields"},
	"text":   {"fields", "max_chars"},
	"canvas": {"palette", "max_nodes"},
}

func presentConfigFields(c course.PartConfig) []string {
	var out []string
	for _, f := range []struct {
		name string
		set  bool
	}{
		{"signature", c.Signature != nil}, {"languages", len(c.Languages) > 0}, {"harness", c.Harness != ""},
		{"checker", c.Checker != nil}, {"constraints", len(c.Constraints) > 0}, {"samples", len(c.Samples) > 0},
		{"limits", c.Limits != nil}, {"mode", c.Mode != ""}, {"options", len(c.Options) > 0},
		{"fields", len(c.Fields) > 0}, {"max_chars", c.MaxChars != 0}, {"palette", c.Palette != ""},
		{"max_nodes", c.MaxNodes != 0},
	} {
		if f.set {
			out = append(out, f.name)
		}
	}
	return out
}

var (
	conceptRefRe = regexp.MustCompile(`^([a-z0-9]+(?:-[a-z0-9]+)*):concept:([a-z0-9]+(?:-[a-z0-9]+)*)$`)
	signalRe     = regexp.MustCompile(`^[a-z][a-z0-9_]*(:[a-z0-9_]+)?$`)
)

// Item runs the public-registry structure lints over one item: #1–#8 against the closed
// registries in internal/course, #12 against concepts (the item's course concept slugs;
// nil skips #12) and #14 against its manifest's revision bands (nil manifest skips #14).
func Item(it *course.Item, m *course.Manifest, concepts map[string]bool) []Finding {
	fs := &findings{item: it.ID}
	parts := map[string]course.Part{}
	for i, p := range it.Parts {
		path := fmt.Sprintf("parts[%d]", i)
		parts[p.ID] = p
		// #1 part types are registered, and their configs decode strictly for the type.
		if !contains(course.PartTypes, p.Type) {
			fs.add("5.6#1", path+".type", "%q is not a registered part type %v", p.Type, course.PartTypes)
			continue
		}
		for _, f := range presentConfigFields(p.Config) {
			if !contains(partConfigFields[p.Type], f) {
				fs.add("5.6#1", path+".config."+f, "is not a %s part's config field", p.Type)
			}
		}
	}
	if it.Revision != nil {
		for i, pr := range it.Revision.Probes {
			path := fmt.Sprintf("revision.probes[%d]", i)
			if !contains(course.ProbeTypes, pr.Type) {
				fs.add("5.6#1", path+".type", "%q is not a registered probe type %v", pr.Type, course.ProbeTypes)
				continue
			}
			if pr.Config != nil {
				c := course.PartConfig{Mode: pr.Config.Mode, Options: pr.Config.Options, Fields: pr.Config.Fields, MaxChars: pr.Config.MaxChars}
				for _, f := range presentConfigFields(c) {
					if !contains(partConfigFields[pr.Type], f) {
						fs.add("5.6#1", path+".config."+f, "is not a %s probe's config field", pr.Type)
					}
				}
			}
			// #8 public:* key sources resolve.
			if f, ok := strings.CutPrefix(pr.KeySource, "public:"); ok && !resolvesPublic(it, f) {
				fs.add("5.6#8", path+".key_source", "%q does not resolve to a public field this item carries", pr.KeySource)
			}
		}
	}

	graded := false
	for _, p := range it.Parts {
		graded = graded || p.Grading != "none"
	}
	required, aiRubrics := false, 0
	fed := map[string]bool{}
	for i, s := range it.Grader {
		path := fmt.Sprintf("grader[%d]", i)
		required = required || s.Required
		// #2 step kinds are in the closed set.
		if !contains(course.GraderKinds, s.Kind) {
			fs.add("5.6#2", path+".kind", "%q is not a registered step kind %v", s.Kind, course.GraderKinds)
		}
		if s.Kind == "ai_rubric" {
			aiRubrics++
		}
		for j, in := range s.Inputs {
			ip := fmt.Sprintf("%s.inputs[%d]", path, j)
			// #3 inputs ⊆ part ids (the Caps type match waits for judge's registry, m3-06).
			p, ok := parts[in]
			if !ok {
				fs.add("5.6#3", ip, "%q is not a part id of this item", in)
				continue
			}
			fed[in] = true
			// #4 key/ai_rubric inputs are final; code inputs are iterate code parts.
			switch s.Kind {
			case "key", "ai_rubric":
				if p.Cadence != "final" {
					fs.add("5.6#4", ip, "a %s step reads final parts; %q is %s", s.Kind, in, p.Cadence)
				}
			case "code":
				if p.Type != "code" || p.Cadence != "iterate" {
					fs.add("5.6#4", ip, "a code step reads iterate code parts; %q is %s/%s", in, p.Type, p.Cadence)
				}
			}
		}
	}
	// #5 at most one ai_rubric step (its pass_rule is the manifest strategy's, t4 §5.3).
	if aiRubrics > 1 {
		fs.add("5.6#5", "grader", "at most one ai_rubric step per item, got %d", aiRubrics)
	}
	// #6 at least one required step when any part is graded.
	if graded && !required {
		fs.add("5.6#6", "grader", "a graded item needs at least one required step")
	}
	// #7 every part feeds a step or is ungraded.
	for i, p := range it.Parts {
		if p.Grading != "none" && !fed[p.ID] {
			fs.add("5.6#7", fmt.Sprintf("parts[%d]", i), "part %q is graded %q but feeds no step (set grading none)", p.ID, p.Grading)
		}
	}
	// #12 concepts[] resolve to an existing <course>:concept:<slug>.
	if concepts != nil {
		for i, c := range it.Concepts {
			sm := conceptRefRe.FindStringSubmatch(c)
			switch {
			case sm == nil:
				fs.add("5.6#12", fmt.Sprintf("concepts[%d]", i), "%q is not <course>:concept:<slug>", c)
			case sm[1] != it.Course:
				fs.add("5.6#12", fmt.Sprintf("concepts[%d]", i), "%q is not a concept of course %q", c, it.Course)
			case !concepts[sm[2]]:
				fs.add("5.6#12", fmt.Sprintf("concepts[%d]", i), "concept %q does not exist in course %q (concepts.json)", sm[2], it.Course)
			}
		}
	}
	// #14 every band criterion maps to a probe or a step check (items with parts only).
	if m != nil && m.Revision != nil && len(it.Parts) > 0 {
		for bi, b := range m.Revision.Bands {
			for _, c := range b.Criteria {
				if !criterionMet(it, b, c.Key) {
					fs.add("5.6#14", fmt.Sprintf("manifest revision.bands[%d]", bi),
						"criterion %q (band %q) maps to no probe of this item and no step check over the band's parts %v", c.Key, b.Format, b.Parts)
				}
			}
		}
	}
	return fs.out
}

func criterionMet(it *course.Item, b course.Band, key string) bool {
	if it.Revision != nil {
		for _, pr := range it.Revision.Probes {
			if pr.Band == b.Format && pr.Criterion == key {
				return true
			}
		}
	}
	kind, ok := StepChecks[key]
	if !ok {
		return false
	}
	for _, s := range it.Grader {
		if s.Kind != kind {
			continue
		}
		for _, in := range s.Inputs {
			if contains(b.Parts, in) {
				return true
			}
		}
	}
	return false
}

// resolvesPublic mirrors course.Item's own check (t4 §5.6 #8).
func resolvesPublic(it *course.Item, field string) bool {
	if !contains(course.PublicKeyFields, field) {
		return false
	}
	if field == "pattern" {
		return strings.TrimSpace(it.Pattern) != ""
	}
	return it.SolutionFacts != nil
}

// Manifest runs #13 over a course manifest: every mistakes.prefill rule names a
// well-formed signal and a declared category, with a known strength.
func Manifest(m *course.Manifest) []Finding {
	fs := &findings{}
	if m.Mistakes == nil {
		return nil
	}
	cats := map[string]bool{}
	for _, c := range m.Mistakes.Categories {
		cats[c.ID] = true
	}
	for i, r := range m.Mistakes.Prefill {
		p := fmt.Sprintf("%s mistakes.prefill[%d]", m.Slug, i)
		if !signalRe.MatchString(r.Signal) {
			fs.add("5.6#13", p+".signal", "%q is not a signal (%s)", r.Signal, signalRe)
		}
		if !cats[r.Category] {
			fs.add("5.6#13", p+".category", "%q is not a declared category", r.Category)
		}
		if !contains(course.PrefillStrengths, r.Strength) {
			fs.add("5.6#13", p+".strength", "%q is not one of %v", r.Strength, course.PrefillStrengths)
		}
	}
	return fs.out
}

// --- key-graded parts, label edits ---------------------------------------------------

// KeyGradedParts returns the ids of the parts a `key` step reads.
func KeyGradedParts(it *course.Item) map[string]bool {
	out := map[string]bool{}
	for _, s := range it.Grader {
		if s.Kind == "key" {
			for _, in := range s.Inputs {
				out[in] = true
			}
		}
	}
	return out
}

// HasAutoCodePart reports whether the item has a code part graded auto (judge-graded
// against the pack's cases).
func HasAutoCodePart(it *course.Item) bool {
	for _, p := range it.Parts {
		if p.Type == "code" && p.Grading == "auto" {
			return true
		}
	}
	return false
}

// LabelEdit is one changed option/field label, under the same id, on a key-graded part
// or a probe (whose keys live beside the id: t1 §3.4 rule 4).
type LabelEdit struct {
	Item, Part, ID string
	Old, New       string
}

// Token is the PR-body confirmation that accepts this edit as a typo:
// "label-edit-ok: <item>/<part>/<id>".
func (e LabelEdit) Token() string { return LabelEditToken + " " + e.Item + "/" + e.Part + "/" + e.ID }

// LabelEditToken prefixes a label-edit confirmation in a PR body.
const LabelEditToken = "label-edit-ok:"

var tokenRe = regexp.MustCompile(`(?m)` + regexp.QuoteMeta(LabelEditToken) + `[ \t]*([^\s/]+/[^\s/]+/[^\s/]+)`)

// Approved parses a PR body's confirmations into "<item>/<part>/<id>" keys.
func Approved(body string) map[string]bool {
	out := map[string]bool{}
	for _, m := range tokenRe.FindAllStringSubmatch(body, -1) {
		out[m[1]] = true
	}
	return out
}

// Key is "<item>/<part>/<id>", the form Approved returns.
func (e LabelEdit) Key() string { return e.Item + "/" + e.Part + "/" + e.ID }

// LabelEdits lists the labels that changed under an unchanged id between base and head,
// on every key-graded part and every probe present in both. A new or removed id is a
// contract change (contract_hash moves), not a label edit.
func LabelEdits(base, head *course.Item) []LabelEdit {
	if base == nil || head == nil {
		return nil
	}
	var out []LabelEdit
	keyed := KeyGradedParts(head)
	baseParts := map[string]course.Part{}
	for _, p := range base.Parts {
		baseParts[p.ID] = p
	}
	for _, p := range head.Parts {
		bp, ok := baseParts[p.ID]
		if !ok || !keyed[p.ID] {
			continue
		}
		out = append(out, labelDiff(head.ID, p.ID, bp.Config.Options, p.Config.Options, bp.Config.Fields, p.Config.Fields)...)
	}
	baseProbes := map[string]course.Probe{}
	if base.Revision != nil {
		for _, pr := range base.Revision.Probes {
			baseProbes[pr.ID] = pr
		}
	}
	if head.Revision != nil {
		for _, pr := range head.Revision.Probes {
			bp, ok := baseProbes[pr.ID]
			if !ok || bp.Config == nil || pr.Config == nil {
				continue
			}
			out = append(out, labelDiff(head.ID, pr.ID, bp.Config.Options, pr.Config.Options, bp.Config.Fields, pr.Config.Fields)...)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Key() < out[j].Key() })
	return out
}

func labelDiff(item, part string, bo, ho []course.Option, bf, hf []course.Field) []LabelEdit {
	var out []LabelEdit
	old := map[string]string{}
	for _, o := range bo {
		old["o:"+o.ID] = o.Label
	}
	for _, f := range bf {
		old["f:"+f.ID] = f.Label
	}
	for _, o := range ho {
		if l, ok := old["o:"+o.ID]; ok && l != o.Label {
			out = append(out, LabelEdit{Item: item, Part: part, ID: o.ID, Old: l, New: o.Label})
		}
	}
	for _, f := range hf {
		if l, ok := old["f:"+f.ID]; ok && l != f.Label {
			out = append(out, LabelEdit{Item: item, Part: part, ID: f.ID, Old: l, New: f.Label})
		}
	}
	return out
}

// --- stamp gate ----------------------------------------------------------------------

// Review stamp names (course.ReviewStamp).
const (
	StampHints     = "hints"
	StampEditorial = "editorial"
)

var stampCodeRe = regexp.MustCompile(`^_code/(hint|solution)-[0-9]{2}\.[a-z][a-z0-9]*\.snip$`)

// StampFor returns the review stamp an item-relative sidecar needs before it may be added
// or changed (t1 §7.2): hints for sections/hint/* and _code/hint-NN.*.snip, editorial
// for sections/solution/* and _code/solution-NN.*.snip. Statements (attempt), reference
// files (_code/solution.<lang>, machine-verified) and item.json need none ("").
func StampFor(rel string) string {
	switch {
	case strings.HasPrefix(rel, "sections/hint/"):
		return StampHints
	case strings.HasPrefix(rel, "sections/solution/"):
		return StampEditorial
	}
	if sm := stampCodeRe.FindStringSubmatch(rel); sm != nil {
		if sm[1] == "hint" {
			return StampHints
		}
		return StampEditorial
	}
	return ""
}

// Stamped reports whether the item carries the named review stamp.
func Stamped(it *course.Item, stamp string) bool {
	if it == nil || it.Review == nil {
		return false
	}
	switch stamp {
	case StampHints:
		return it.Review.Hints != ""
	case StampEditorial:
		return it.Review.Editorial != ""
	}
	return false
}

// --- packlint rule 8: content-only edits that invalidate pack expectations ------------

// AdvisoryEdits lists the content-only fields that differ between base and head and that
// the pack's expectations depend on (t1 §3.4 rule 3): a code part's limits, samples,
// languages and constraints, and probe prompts. They move content_hash only; packlint
// warns "re-run make packcheck". Statement (attempt section) edits are a file diff, not
// an item field: packlint checks them itself.
func AdvisoryEdits(base, head *course.Item) []string {
	if base == nil || head == nil {
		return nil
	}
	var out []string
	bp := map[string]course.Part{}
	for _, p := range base.Parts {
		bp[p.ID] = p
	}
	for _, p := range head.Parts {
		b, ok := bp[p.ID]
		if !ok {
			continue
		}
		pre := "parts." + p.ID + ".config."
		if !equalJSON(b.Config.Limits, p.Config.Limits) {
			out = append(out, pre+"limits")
		}
		if !equalJSON(b.Config.Samples, p.Config.Samples) {
			out = append(out, pre+"samples")
		}
		if !equalJSON(b.Config.Languages, p.Config.Languages) {
			out = append(out, pre+"languages")
		}
		if !equalJSON(b.Config.Constraints, p.Config.Constraints) {
			out = append(out, pre+"constraints")
		}
	}
	bpr := map[string]string{}
	if base.Revision != nil {
		for _, pr := range base.Revision.Probes {
			bpr[pr.ID] = pr.PromptMD
		}
	}
	if head.Revision != nil {
		for _, pr := range head.Revision.Probes {
			if old, ok := bpr[pr.ID]; ok && old != pr.PromptMD {
				out = append(out, "revision.probes."+pr.ID+".prompt_md")
			}
		}
	}
	return out
}

// equalJSON compares two typed values by their canonical bytes; open sample values are
// normalized first, so a respelled number is not an edit.
func equalJSON(a, b any) bool {
	na, err1 := normalizedSamples(a)
	nb, err2 := normalizedSamples(b)
	if err1 != nil || err2 != nil {
		return false
	}
	ba, err1 := canon.Bytes(na)
	bb, err2 := canon.Bytes(nb)
	return err1 == nil && err2 == nil && bytes.Equal(ba, bb)
}

func normalizedSamples(v any) (any, error) {
	ss, ok := v.([]course.Sample)
	if !ok {
		return v, nil
	}
	out := make([]course.Sample, len(ss))
	for i, s := range ss {
		args := make([]json.RawMessage, len(s.Args))
		for j, a := range s.Args {
			n, err := canon.NormalizeJSON(a)
			if err != nil {
				return nil, err
			}
			args[j] = n
		}
		exp, err := canon.NormalizeJSON(s.Expected)
		if err != nil {
			return nil, err
		}
		s.Args, s.Expected = args, exp
		out[i] = s
	}
	return out, nil
}

func contains(set []string, v string) bool {
	for _, s := range set {
		if s == v {
			return true
		}
	}
	return false
}

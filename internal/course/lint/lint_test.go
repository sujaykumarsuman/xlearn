package lint_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sujaykumarsuman/xlearn/internal/course"
	"github.com/sujaykumarsuman/xlearn/internal/course/lint"
)

func item(t *testing.T, name string) *course.Item {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "testdata", "items", name))
	if err != nil {
		t.Fatal(err)
	}
	it, err := course.DecodeItem(b)
	if err != nil {
		t.Fatal(err)
	}
	return it
}

func manifest(t *testing.T, slug string) *course.Manifest {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "..", "curriculum", "courses", slug, "course.json"))
	if err != nil {
		t.Fatal(err)
	}
	m, err := course.DecodeManifest(b)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func rules(fs []lint.Finding) string {
	var out []string
	for _, f := range fs {
		out = append(out, f.String())
	}
	return strings.Join(out, "\n")
}

var dsaConcepts = map[string]bool{"hashing": true, "two-pointers": true, "linked-list-design": true}

// The valid fixtures pass #1–#8 and #12 (#14 is exercised below).
func TestFixturesPassStructureLints(t *testing.T) {
	for _, name := range []string{"valid-self.json", "valid-code.json", "valid-probes.json", "valid-structured.json", "valid-class.json"} {
		concepts := map[string]bool{"hashing": true, "two-pointers": true, "linked-list-design": true, "caching": true, "sharding": true}
		if fs := lint.Item(item(t, name), nil, concepts); len(fs) > 0 {
			t.Errorf("%s: unexpected findings:\n%s", name, rules(fs))
		}
	}
}

func TestStructureLints(t *testing.T) {
	for name, tc := range map[string]struct {
		fixture string
		edit    func(it *course.Item)
		want    string
	}{
		"#1 unknown part type": {"valid-code.json", func(it *course.Item) { it.Parts[0].Type = "audio" }, "[5.6#1]"},
		"#1 config field of another type": {"valid-code.json", func(it *course.Item) {
			it.Parts[0].Config.Options = []course.Option{{ID: "a", Label: "A"}}
		}, "is not a code part's config field"},
		"#1 probe config field": {"valid-probes.json", func(it *course.Item) { it.Revision.Probes[0].Config.Mode = "single" }, "is not a text probe's config field"},
		"#2 step kind":          {"valid-code.json", func(it *course.Item) { it.Grader[0].Kind = "oracle" }, "[5.6#2]"},
		"#3 unknown input":      {"valid-code.json", func(it *course.Item) { it.Grader[0].Inputs = []string{"solution", "extra"} }, `"extra" is not a part id`},
		"#4 key step on iterate": {"valid-structured.json", func(it *course.Item) {
			it.Parts[1].Cadence = "iterate"
		}, "[5.6#4]"},
		"#4 code step on final": {"valid-code.json", func(it *course.Item) { it.Parts[0].Cadence = "final" }, "a code step reads iterate code parts"},
		"#5 two ai_rubric": {"valid-structured.json", func(it *course.Item) {
			it.Grader[0].Kind = "ai_rubric"
			it.Grader[0].Rubric = "system-design/hld@1"
		}, "[5.6#5]"},
		"#6 no required step": {"valid-code.json", func(it *course.Item) { it.Grader[0].Required = false }, "[5.6#6]"},
		"#7 graded part feeds nothing": {"valid-structured.json", func(it *course.Item) {
			it.Parts[3].Grading = "auto"
		}, `part "pick" is graded "auto" but feeds no step`},
		"#8 public key source unresolved": {"valid-probes.json", func(it *course.Item) { it.SolutionFacts = nil }, "[5.6#8]"},
		"#12 unknown concept": {"valid-code.json", func(it *course.Item) {
			it.Concepts = []string{"dsa:concept:nope"}
		}, `concept "nope" does not exist`},
		"#12 other course": {"valid-code.json", func(it *course.Item) {
			it.Concepts = []string{"sql:concept:joins"}
		}, "is not a concept of course"},
	} {
		t.Run(name, func(t *testing.T) {
			it := item(t, tc.fixture)
			tc.edit(it)
			got := rules(lint.Item(it, nil, map[string]bool{"hashing": true, "two-pointers": true, "caching": true, "sharding": true}))
			if !strings.Contains(got, tc.want) {
				t.Fatalf("findings %q, want one containing %q", got, tc.want)
			}
		})
	}
}

// #14: every band criterion of the course maps to a probe or a step check, for items
// with parts only.
func TestBandCriteriaLint(t *testing.T) {
	m := manifest(t, "dsa")
	// A DSA code item with no probes: correct_in_timer is met by its code step over the
	// band part "solution"; the pattern and complexity criteria need probes.
	got := rules(lint.Item(item(t, "valid-code.json"), m, dsaConcepts))
	for _, want := range []string{`"pattern_named_fast"`, `"complexity_stated"`} {
		if !strings.Contains(got, want) {
			t.Errorf("missing a #14 finding for %s in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "correct_in_timer") {
		t.Errorf("correct_in_timer is a step check of the code step:\n%s", got)
	}
	// With the two templated probes it is clean.
	it := item(t, "valid-code.json")
	probes := item(t, "valid-probes.json").Revision
	it.Revision = &course.ItemRevision{Probes: probes.Probes[:2]}
	if got := lint.Item(it, m, dsaConcepts); len(got) > 0 {
		t.Errorf("unexpected findings:\n%s", rules(got))
	}
	// An item with no parts is out of scope (the self path).
	if got := lint.Item(item(t, "valid-self.json"), m, dsaConcepts); len(got) > 0 {
		t.Errorf("a self-path item got #14 findings:\n%s", rules(got))
	}
}

func TestManifestLint(t *testing.T) {
	m := manifest(t, "dsa")
	if got := lint.Manifest(m); len(got) > 0 {
		t.Fatalf("the DSA manifest has findings:\n%s", rules(got))
	}
	m.Mistakes.Prefill = []course.PrefillRule{
		{Signal: "verdict:WA", Category: "off_by_one", Strength: "weak"},
		{Signal: "tle", Category: "nope", Strength: "strong"},
		{Signal: "ok", Category: "misread", Strength: "maybe"},
	}
	got := rules(lint.Manifest(m))
	for _, want := range []string{`"verdict:WA" is not a signal`, `"nope" is not a declared category`, `"maybe" is not one of`} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

func TestLabelEdits(t *testing.T) {
	base := item(t, "valid-structured.json")
	head := item(t, "valid-structured.json")
	// est is key-graded (the estimates step); pick is ungraded; reqs is ai-graded.
	head.Parts[1].Config.Fields[0].Label = "Peak uploads / s"
	head.Parts[3].Config.Options[0].Label = "A CDN"
	head.Parts[0].Config.Fields[0].Label = "Features"
	edits := lint.LabelEdits(base, head)
	if len(edits) != 1 || edits[0].Key() != "sd-001/est/write_qps" || edits[0].Old != "Peak uploads per second" {
		t.Fatalf("LabelEdits = %+v, want only sd-001/est/write_qps", edits)
	}
	if edits[0].Token() != "label-edit-ok: sd-001/est/write_qps" {
		t.Fatalf("Token = %q", edits[0].Token())
	}
	// A new id is a contract change, not a label edit.
	head2 := item(t, "valid-structured.json")
	head2.Parts[1].Config.Fields[0] = course.Field{ID: "peak_qps", Label: "Peak", Kind: "numeric"}
	if e := lint.LabelEdits(base, head2); len(e) != 0 {
		t.Fatalf("a renamed id was reported as a label edit: %+v", e)
	}
	// Probes are key-graded: their option labels are flagged.
	pb, ph := item(t, "valid-probes.json"), item(t, "valid-probes.json")
	ph.Revision.Probes[2].Config.Options[1].Label = "The pointer sweep"
	if e := lint.LabelEdits(pb, ph); len(e) != 1 || e[0].Key() != "901/p-structure/b" {
		t.Fatalf("probe label edit = %+v", e)
	}
}

func TestApproved(t *testing.T) {
	body := "Fix typos.\n\nlabel-edit-ok: sd-001/est/write_qps\nlabel-edit-ok:901/p-structure/b\nnot label-edit-ok"
	got := lint.Approved(body)
	if !got["sd-001/est/write_qps"] || !got["901/p-structure/b"] || len(got) != 2 {
		t.Fatalf("Approved = %v", got)
	}
}

func TestStampFor(t *testing.T) {
	for rel, want := range map[string]string{
		"sections/hint/01-key_observation.md": lint.StampHints,
		"sections/solution/01-approach.md":    lint.StampEditorial,
		"_code/solution-02.go.snip":           lint.StampEditorial,
		"_code/hint-01.py.snip":               lint.StampHints,
		"sections/attempt/01-summary.md":      "",
		"_code/attempt-01.go.snip":            "",
		"_code/solution.go":                   "",
		"item.json":                           "",
	} {
		if got := lint.StampFor(rel); got != want {
			t.Errorf("StampFor(%s) = %q, want %q", rel, got, want)
		}
	}
	it := item(t, "valid-code.json")
	if !lint.Stamped(it, lint.StampHints) || !lint.Stamped(it, lint.StampEditorial) {
		t.Fatal("valid-code.json carries both stamps")
	}
	if lint.Stamped(item(t, "valid-self.json"), lint.StampEditorial) {
		t.Fatal("valid-self.json has no editorial stamp")
	}
}

func TestAdvisoryEdits(t *testing.T) {
	base := item(t, "valid-code.json")
	head := item(t, "valid-code.json")
	if e := lint.AdvisoryEdits(base, head); len(e) != 0 {
		t.Fatalf("no edit reported %v", e)
	}
	head.Parts[0].Config.Samples[0].Args = []json.RawMessage{json.RawMessage(`[ "kiwi","fig" , "kiwi","plum" ]`)}
	if e := lint.AdvisoryEdits(base, head); len(e) != 0 {
		t.Fatalf("reformatting a sample was reported: %v", e)
	}
	head.Parts[0].Config.Limits.TimeMS = 500
	head.Parts[0].Config.Constraints[0].Len = []int{1, 10}
	got := strings.Join(lint.AdvisoryEdits(base, head), ",")
	if got != "parts.solution.config.limits,parts.solution.config.constraints" {
		t.Fatalf("AdvisoryEdits = %s", got)
	}
	pb, ph := item(t, "valid-probes.json"), item(t, "valid-probes.json")
	ph.Revision.Probes[0].PromptMD = "Name the pattern, fast."
	if got := lint.AdvisoryEdits(pb, ph); len(got) != 1 || got[0] != "revision.probes.p-pattern.prompt_md" {
		t.Fatalf("probe prompt edit = %v", got)
	}
}

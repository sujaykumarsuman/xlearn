package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/sujaykumarsuman/xlearn/internal/course"
	"github.com/sujaykumarsuman/xlearn/internal/curriculum"
)

// The repository's own content passes every check (the same run as `make contentlint`).
func TestRepoContentPasses(t *testing.T) {
	var out bytes.Buffer
	l := &linter{root: filepath.Join("..", "..", "curriculum"), out: &out}
	if !l.run() {
		t.Fatalf("contentlint failed on the repository's curriculum/:\n%s", out.String())
	}
}

func TestMarkdownProfile(t *testing.T) {
	for _, tc := range []struct {
		name, src string
		want      string // "" = no problem
	}{
		{"plain prose", "A **bold** claim with `code` and a list:\n\n- one\n- two\n", ""},
		{"gfm table", "| a | b |\n|---|---|\n| 1 | 2 |\n", ""},
		{"fenced code with markup inside", "```go\nx := \"<div>\"\n```\n", ""},
		{"comparison operators", "if a < b and b > c then a < c.", ""},
		{"https link", "See [the docs](https://go.dev/doc/).", ""},
		{"asset image", "![graph](asset:dsa/graph@1)", ""},
		{"raw HTML block", "<div>\nhi\n</div>\n", "raw HTML block"},
		{"inline raw HTML", "Press <kbd>Enter</kbd> now.", "raw HTML"},
		{"script tag", "<script>alert(1)</script>\n", "raw HTML"},
		{"http link", "See [x](http://example.com).", `link "http://example.com" must be https://`},
		{"relative link", "See [x](../other.md).", "must be https://"},
		{"javascript link", "See [x](javascript:alert(1)).", "must be https://"},
		{"remote image", "![x](https://example.com/x.png)", "must be an asset: ref"},
		{"bare www autolink", "Visit www.example.com today.", "autolink"},
		{"angle autolink http", "<http://example.com>", "autolink"},
		{"angle autolink https", "<https://example.com>", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := strings.Join(markdownProblems([]byte(tc.src)), "; ")
			switch {
			case tc.want == "" && got != "":
				t.Errorf("unexpected problem: %s", got)
			case tc.want != "" && !strings.Contains(got, tc.want):
				t.Errorf("problems %q, want one containing %q", got, tc.want)
			}
		})
	}
}

func TestSVGLint(t *testing.T) {
	for src, want := range map[string]string{
		`<svg><rect width="1"/></svg>`:                          "",
		`<svg><script>x()</script></svg>`:                       "script",
		`<svg><rect onclick="x()"/></svg>`:                      "event handler",
		`<svg><image href="https://evil.example/x.png"/></svg>`: "external",
		`<svg><use xlink:href="//evil.example/s.svg#a"/></svg>`: "external",
		`<svg><rect style="fill:url(http://x/y)"/></svg>`:       "external",
		`<svg><use href="#local"/></svg>`:                       "",
	} {
		got := strings.Join(svgProblems([]byte(src)), "; ")
		if (want == "") != (got == "") || !strings.Contains(got, want) {
			t.Errorf("svgProblems(%s) = %q, want %q", src, got, want)
		}
	}
}

func TestGoFileProblem(t *testing.T) {
	for name, tc := range map[string]struct{ src, want string }{
		"v1 fragment":      {"func twoSum(nums []int) []int {\n    return nil\n}", "not a complete Go file"},
		"unformatted file": {"package x\nfunc  F() {}\n", "not gofmt-clean"},
		"complete file":    {"package x\n\nfunc F() {}\n", ""},
	} {
		if got := goFileProblem(name+".go", []byte(tc.src)); (tc.want == "") != (got == "") || !strings.Contains(got, tc.want) {
			t.Errorf("%s: goFileProblem = %q, want %q", name, got, tc.want)
		}
	}
}

func TestFilenameRules(t *testing.T) {
	fsys := fstest.MapFS{
		"courses/dsa/items/1/item.json":                 {Data: []byte("{}")},
		"courses/dsa/items/1/_code/solution-02.go.snip": {Data: []byte("func f() {}")},
		"courses/dsa/items/1/hidden_cases.json":         {Data: []byte("{}")},
		"courses/dsa/items/1/Expected-Output.txt":       {Data: []byte("x")},
		"courses/dsa/items/1/case.ans":                  {Data: []byte("x")},
		"courses/dsa/items/1/submissions/a.go.snip":     {Data: []byte("x")},
		"courses/dsa/wrong/w.snip":                      {Data: []byte("x")},
		"courses/dsa/items/1/_code/frag.go":             {Data: []byte("func f() {}")},
		"courses/dsa/rubrics/anchors@1.json":            {Data: []byte("{}")},
		// m3-01: the pack's own names
		"courses/dsa/items/1/pack.json":             {Data: []byte("{}")},
		"courses/dsa/items/1/tests/edge.jsonl":      {Data: []byte("{}")},
		"courses/dsa/items/1/tests.lock":            {Data: []byte("{}")},
		"courses/dsa/items/1/cases.jsonl.zst":       {Data: []byte("x")},
		"courses/dsa/items/1/invalid/a.json":        {Data: []byte("{}")},
		"courses/dsa/assets/q@1/gen-hidden.sql":     {Data: []byte("x")},
		"courses/dsa/assets/q@1/instances.json":     {Data: []byte("{}")},
		"courses/dsa/items/1/timing.json":           {Data: []byte("{}")},
		"courses/dsa/items/1/data.jsonl.zst":        {Data: []byte("x")},
		"courses/dsa/keys/big-o-aliases.json":       {Data: []byte("{}")},
		"courses/dsa/items/1/_code/solution.go":     {Data: []byte("package solution\n")},
		"courses/dsa/items/1/sections/hint/01-a.md": {Data: []byte("x")},
	}
	l := &linter{stats: map[string]int{}}
	l.checkFilenames(fsys)
	got := strings.Join(l.problems, "\n")
	for _, want := range []string{"hidden_cases.json", "Expected-Output.txt", "case.ans", "submissions/", "wrong/", "frag.go: not a complete Go file", "anchors@1.json",
		"items/1/pack.json", "edge.jsonl", "items/1/tests.lock", "cases.jsonl.zst", "invalid/", "gen-hidden.sql", "instances.json", "timing.json", "data.jsonl.zst"} {
		if !strings.Contains(got, want) {
			t.Errorf("no filename problem for %s in:\n%s", want, got)
		}
	}
	for _, ok := range []string{"solution-02.go.snip", "item.json", "big-o-aliases.json", "_code/solution.go", "01-a.md"} {
		if strings.Contains(got, ok) {
			t.Errorf("a legitimate file (%s) was flagged:\n%s", ok, got)
		}
	}
}

// The repo-wide pack-artefact pass honours only SYNTHETIC.md-marked internal/**/testdata/
// trees.
func TestPackArtefactPass(t *testing.T) {
	markers := map[string]string{
		"internal/judge/testdata/SYNTHETIC.md":          SyntheticMarker,
		"internal/runner/testdata/SYNTHETIC.md":         "synthetic, probably",
		"internal/judge/pack/testdata/SYNTHETIC.md":     SyntheticMarker,
		"internal/judge/pack/testdata/sub/SYNTHETIC.md": SyntheticMarker,
	}
	files := []string{
		// allowed: inside a marked tree
		"internal/judge/testdata/packsrc/tests.lock",
		"internal/judge/testdata/pack/courses/dsa/items/1/cases.jsonl.zst",
		"internal/judge/pack/testdata/cases.jsonl",
		// failing
		"tests.lock",
		"curriculum/courses/dsa/items/1/cases.jsonl",
		"cmd/packlint/testdata/tests.lock",
		"internal/judge/data/cases.jsonl",
		"internal/runner/testdata/items/1/cases.jsonl",
		"internal/other/testdata/x.jsonl.zst",
		// not artefacts
		"internal/judge/testdata/notes.md",
		"docs/cases.md",
	}
	got := packArtefactProblems(files, func(p string) (string, bool) { m, ok := markers[p]; return m, ok })
	joined := strings.Join(got, "\n")
	for _, bad := range []string{"tests.lock: a private", "curriculum/courses/dsa/items/1/cases.jsonl", "cmd/packlint/testdata/tests.lock",
		"internal/judge/data/cases.jsonl: a private eval-pack artefact name outside a testdata/ tree",
		"internal/runner/testdata/items/1/cases.jsonl: pack artefact in internal/runner/testdata/ without its SYNTHETIC.md marker",
		"internal/other/testdata/x.jsonl.zst: pack artefact in internal/other/testdata/ without"} {
		if !strings.Contains(joined, bad) {
			t.Errorf("missing a problem for %q in:\n%s", bad, joined)
		}
	}
	if len(got) != 6 {
		t.Errorf("%d problems, want 6:\n%s", len(got), joined)
	}
	for _, ok := range files[:3] {
		if strings.Contains(joined, ok+":") {
			t.Errorf("%s is in a marked tree but was flagged", ok)
		}
	}
}

// The stamp gate and the label-edit flag, against a base commit in a scratch clone of
// the curriculum.
func TestDiffGates(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "curriculum")
	if err := os.CopyFS(root, os.DirFS(filepath.Join("..", "..", "curriculum"))); err != nil {
		t.Fatal(err)
	}
	gitT := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	gitT("init", "-q", "-b", "main")
	gitT("config", "user.email", "contentlint-test")
	gitT("config", "user.name", "test")
	gitT("config", "commit.gpgsign", "false")
	// Item 1 gets a pack-keyed choice probe in the base.
	itemPath := filepath.Join(root, "courses/dsa/items/1/item.json")
	editItem := func(fn func(*course.Item)) {
		t.Helper()
		b, err := os.ReadFile(itemPath)
		if err != nil {
			t.Fatal(err)
		}
		it, err := course.DecodeItem(b)
		if err != nil {
			t.Fatal(err)
		}
		fn(it)
		out, _ := json.MarshalIndent(it, "", "  ")
		if err := os.WriteFile(itemPath, append(out, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	editItem(func(it *course.Item) {
		it.Revision = &course.ItemRevision{Probes: []course.Probe{{
			ID: "p-structure", Type: "choice", Band: "resolve", Grading: "key", KeySource: "pack",
			Criterion: "complexity_stated", PromptMD: "Which step dominates?",
			Config: &course.ProbeConfig{Mode: "single", Options: []course.Option{{ID: "a", Label: "The set"}, {ID: "b", Label: "The loop"}}},
		}}}
	})
	gitT("add", "-A")
	gitT("commit", "-q", "-m", "base")
	base := gitT("rev-parse", "HEAD")

	gates := func(body string) string {
		t.Helper()
		content, err := curriculum.LoadContent(os.DirFS(root))
		if err != nil {
			t.Fatal(err)
		}
		l := &linter{root: root, base: base, prBody: body, stats: map[string]int{}}
		l.checkDiffGates(content)
		return strings.Join(append(l.problems, l.notes...), "\n")
	}
	if got := gates(""); got != "" {
		t.Fatalf("no change, but: %s", got)
	}

	// Stamp gate: a new hint and an edited editorial section on an unstamped item fail;
	// an edited statement and unchanged v1 sections do not.
	write := func(rel, body string) {
		t.Helper()
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("courses/dsa/items/1/sections/hint/01-key_observation.md", "A new hint.")
	write("courses/dsa/items/1/sections/solution/01-approach.md", "An edited approach.")
	write("courses/dsa/items/1/sections/attempt/01-summary.md", "An edited statement.")
	got := gates("")
	for _, want := range []string{
		"stamp: courses/dsa/items/1/sections/hint/01-key_observation.md: added or changed, but item 1 has no review.hints stamp",
		"stamp: courses/dsa/items/1/sections/solution/01-approach.md: added or changed, but item 1 has no review.editorial stamp",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "attempt") || strings.Count(got, "stamp:") != 2 {
		t.Errorf("only the two new/changed hint/editorial files may fail:\n%s", got)
	}
	editItem(func(it *course.Item) { it.Review = &course.ReviewStamp{Hints: "2026-10-07", Editorial: "2026-10-07"} })
	if got := gates(""); got != "" {
		t.Fatalf("stamped item still fails: %s", got)
	}

	// Label-edit flag: a changed label under the same id fails without the PR-body token.
	editItem(func(it *course.Item) { it.Revision.Probes[0].Config.Options[0].Label = "The hash set" })
	got = gates("")
	if !strings.Contains(got, `label-edit: courses/dsa/items/1/item.json: the label of key-graded p-structure/a changed ("The set" -> "The hash set")`) ||
		!strings.Contains(got, "label-edit-ok: 1/p-structure/a") {
		t.Fatalf("label edit not flagged:\n%s", got)
	}
	got = gates("Typo fix.\n\nlabel-edit-ok: 1/p-structure/a\n")
	if strings.Contains(got, "label-edit:") || !strings.Contains(got, "confirmed as a typo") {
		t.Fatalf("the PR-body token did not confirm the edit:\n%s", got)
	}
	// A new option id is a contract change, not a label edit.
	editItem(func(it *course.Item) {
		it.Revision.Probes[0].Config.Options[0] = course.Option{ID: "c", Label: "The hash set"}
	})
	if got := gates(""); strings.Contains(got, "label-edit:") {
		t.Fatalf("a new id was flagged as a label edit:\n%s", got)
	}

	// An unresolvable explicit base fails loudly.
	l := &linter{root: root, base: "0123456789abcdef0123456789abcdef01234567", stats: map[string]int{}}
	content, _ := curriculum.LoadContent(os.DirFS(root))
	l.checkDiffGates(content)
	if !strings.Contains(strings.Join(l.problems, "\n"), "is not a commit here") {
		t.Fatalf("unknown base: %v", l.problems)
	}
}

// The repository's v1 hint/editorial sections are all grandfathered (unstamped, listed).
func TestUnstampedReport(t *testing.T) {
	content, err := curriculum.LoadContent(os.DirFS(filepath.Join("..", "..", "curriculum")))
	if err != nil {
		t.Fatal(err)
	}
	list := unstamped(content)
	if len(list) == 0 {
		t.Fatal("expected grandfathered v1 sections")
	}
	for _, line := range list {
		if !strings.HasPrefix(line, "courses/dsa/items/") || !strings.Contains(line, "section(s)") {
			t.Errorf("unexpected line %q", line)
		}
	}
}

func TestCompareLocks(t *testing.T) {
	prev := &course.IDsLock{
		Items:  map[string]course.LockEntry{"1": {Course: "dsa", Status: "live"}, "2": {Course: "dsa", Status: "live"}, "3": {Course: "dsa", Status: "live"}},
		Assets: map[string]string{"dsa/graph@1": "sha256:aa", "dsa/tree@1": "sha256:bb"},
	}
	cur := &course.IDsLock{
		Items:  map[string]course.LockEntry{"1": {Course: "dsa", Status: "retired"}, "3": {Course: "sql", Status: "live"}, "4": {Course: "dsa", Status: "live"}},
		Assets: map[string]string{"dsa/graph@1": "sha256:cc"},
	}
	l := &linter{stats: map[string]int{}}
	l.compareLocks("v1.6.0", prev, cur)
	got := strings.Join(l.problems, "\n")
	for _, want := range []string{`id "2"`, `id "3" moved`, `asset "dsa/graph@1" changed bytes`, `asset "dsa/tree@1"`} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, `id "1"`) || strings.Contains(got, `id "4"`) {
		t.Errorf("a retire or an addition was flagged:\n%s", got)
	}
}

func TestDiffSets(t *testing.T) {
	extra, missing := diffSets([]string{"a", "b", ".x"}, []string{"a", "b", "c"})
	if strings.Join(extra, ",") != ".x" || strings.Join(missing, ",") != "c" {
		t.Fatalf("diffSets = %v, %v", extra, missing)
	}
}

// contentlint's dependencies (the Markdown parser, the JSON Schema validator) and the
// tool itself never reach a service binary.
func TestServiceBinariesExcludeContentlintDeps(t *testing.T) {
	if testing.Short() {
		t.Skip("shells out to go list")
	}
	entries, err := os.ReadDir("..")
	if err != nil {
		t.Fatal(err)
	}
	var svcs []string
	for _, e := range entries {
		if e.IsDir() && !devTools[e.Name()] {
			svcs = append(svcs, "../"+e.Name())
		}
	}
	if len(svcs) < 7 {
		t.Fatalf("found %d service commands under cmd/, want the 7 services", len(svcs))
	}
	out, err := exec.Command("go", append([]string{"list", "-deps"}, svcs...)...).Output()
	if err != nil {
		t.Fatalf("go list -deps: %v", err)
	}
	for _, dep := range strings.Fields(string(out)) {
		for _, banned := range []string{"github.com/yuin/goldmark", "github.com/santhosh-tekuri/jsonschema", "cmd/contentlint", "cmd/packlint"} {
			if strings.Contains(dep, banned) {
				t.Errorf("a service binary depends on %s (via %s)", banned, dep)
			}
		}
	}
}

// devTools are the cmd/ directories that are tools, not services.
var devTools = map[string]bool{"contentlint": true, "packlint": true}

// The dev tools and the hook never enter an image: no Dockerfile names them.
func TestDockerfilesExcludeDevTools(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("..", "..", "deploy", "*.Dockerfile"))
	if err != nil || len(files) < 7 {
		t.Fatalf("found %d Dockerfiles (%v), want the 7 services", len(files), err)
	}
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, tool := range []string{"contentlint", "packlint", "git-hooks"} {
			if strings.Contains(string(b), tool) {
				t.Errorf("%s mentions %s: dev tools never enter an image", f, tool)
			}
		}
	}
}

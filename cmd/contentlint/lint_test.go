package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/sujaykumarsuman/xlearn/internal/course"
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
	}
	l := &linter{stats: map[string]int{}}
	l.checkFilenames(fsys)
	got := strings.Join(l.problems, "\n")
	for _, want := range []string{"hidden_cases.json", "Expected-Output.txt", "case.ans", "submissions/", "wrong/", "frag.go: not a complete Go file", "anchors@1.json"} {
		if !strings.Contains(got, want) {
			t.Errorf("no filename problem for %s in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "solution-02.go.snip") || strings.Contains(got, "item.json") {
		t.Errorf("a legitimate file was flagged:\n%s", got)
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
		if e.IsDir() && e.Name() != "contentlint" {
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
		for _, banned := range []string{"github.com/yuin/goldmark", "github.com/santhosh-tekuri/jsonschema", "cmd/contentlint"} {
			if strings.Contains(dep, banned) {
				t.Errorf("a service binary depends on %s (via %s)", banned, dep)
			}
		}
	}
}

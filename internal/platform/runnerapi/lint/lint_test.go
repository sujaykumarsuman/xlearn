package lint

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/sujaykumarsuman/xlearn/internal/platform/runnerapi"
)

var profiles = map[string]string{"go": "go@1.26", "cpp": "cpp@g++14", "python": "python@3.13"}

// harnessStub is the language's judge-generated file set with a learner file.
func jobFiles(lang string, learner []byte) []runnerapi.File {
	files := []runnerapi.File{{Path: learnerFile[lang], Data: learner}}
	for _, h := range harnessFiles[lang] {
		files = append(files, runnerapi.File{Path: h, Data: []byte("// judge's own\n")})
	}
	return files
}

// TestGoldenFixtures: every testdata/<lang>/<name> file names the rules it trips on its first
// line (`want:`); Check must report exactly those, and every rule has a fixture.
func TestGoldenFixtures(t *testing.T) {
	seen := map[string]bool{}
	for lang, prof := range profiles {
		paths, err := filepath.Glob(filepath.Join("testdata", lang, "*"))
		if err != nil || len(paths) == 0 {
			t.Fatalf("%s: no fixtures (%v)", lang, err)
		}
		for _, p := range paths {
			src, err := os.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}
			first, _, _ := strings.Cut(string(src), "\n")
			_, wantText, ok := strings.Cut(first, "want:")
			if !ok {
				t.Fatalf("%s: first line must be a want: comment", p)
			}
			want := strings.Fields(wantText)
			var got []string
			for _, v := range Check(prof, jobFiles(lang, src)) {
				if v.File != learnerFile[lang] {
					t.Errorf("%s: violation on %s", p, v.File)
				}
				got = append(got, v.Rule)
				seen[v.Rule] = true
			}
			slices.Sort(got)
			slices.Sort(want)
			if !slices.Equal(got, want) {
				t.Errorf("%s: rules %v, want %v\n%v", p, got, want, Check(prof, jobFiles(lang, src)))
			}
		}
	}
	for _, r := range []string{RuleGoImport, RuleGoDirective, RuleCppInclude, RuleCppDirective, RuleCppAsm, RuleCppMain, RulePyImport} {
		if !seen[r] {
			t.Errorf("rule %s has no golden fixture", r)
		}
	}
}

// TestEvasionFixturesPassTheLint: each language's evasion fixture is lint-clean on purpose
// (C++, Python) — the jail is what stops it — except Go's, which has no lint-clean path to a
// syscall and is posted past the lint by refs_it_test.
func TestEvasionFixturesPassTheLint(t *testing.T) {
	for lang, prof := range profiles {
		ext := map[string]string{"go": ".go", "cpp": ".cpp", "python": ".py"}[lang]
		src, err := os.ReadFile(filepath.Join("testdata", lang, "evasion"+ext))
		if err != nil {
			t.Fatal(err)
		}
		vs := Check(prof, jobFiles(lang, src))
		if lang == "go" {
			if len(vs) == 0 {
				t.Error("go evasion: the lint must reject a syscall import")
			}
			continue
		}
		if len(vs) != 0 {
			t.Errorf("%s evasion: the lint should miss it (the jail stops it): %v", lang, vs)
		}
	}
}

func TestCheckNames(t *testing.T) {
	ok := jobFiles("cpp", []byte("class Solution {};\n"))
	if vs := CheckNames("cpp@g++14", ok, nil); len(vs) != 0 {
		t.Fatalf("clean job: %v", vs)
	}
	cases := []struct {
		name    string
		profile string
		files   []runnerapi.File
		hidden  []runnerapi.File
		want    string
	}{
		{"unknown profile", "rust@1", ok, nil, RuleProfile},
		{"test profile", "testgo@0", ok, nil, RuleProfile},
		{"an extra file", "go@1.26", append(jobFiles("go", nil), runnerapi.File{Path: "asm_amd64.s"}), nil, RuleFileName},
		{"a cgo file", "go@1.26", append(jobFiles("go", nil), runnerapi.File{Path: "x.c"}), nil, RuleFileName},
		{"go.mod", "go@1.26", append(jobFiles("go", nil), runnerapi.File{Path: "go.mod"}), nil, RuleFileName},
		{"a path", "python@3.13", []runnerapi.File{{Path: "../solution.py"}}, nil, RuleFileName},
		{"a header", "cpp@g++14", append(jobFiles("cpp", nil), runnerapi.File{Path: "evil.h"}), nil, RuleFileName},
		{"another language's file", "python@3.13", []runnerapi.File{{Path: "solution.go"}}, nil, RuleFileName},
		{"no learner file", "go@1.26", []runnerapi.File{{Path: "zz_xl_harness.go"}}, nil, RuleNoLearner},
		{"hidden files", "go@1.26", jobFiles("go", nil), []runnerapi.File{{Path: "hidden_test.go"}}, RuleHidden},
	}
	for _, c := range cases {
		vs := CheckNames(c.profile, c.files, c.hidden)
		found := false
		for _, v := range vs {
			found = found || v.Rule == c.want
		}
		if !found {
			t.Errorf("%s: want a %s violation, got %v", c.name, c.want, vs)
		}
	}
	if Language("python@3.13") != "python" || Language("go") != "go" || Language("go-race@1.26") != "" {
		t.Error("Language mapping")
	}
}

// TestLintIsALeaf: judge and the runner both link this package, so it imports the stdlib
// and runnerapi only.
func TestLintIsALeaf(t *testing.T) {
	files, _ := filepath.Glob("*.go")
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		af, err := parser.ParseFile(token.NewFileSet(), f, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range af.Imports {
			p, _ := strconv.Unquote(imp.Path.Value)
			first, _, _ := strings.Cut(p, "/")
			if strings.Contains(first, ".") && p != "github.com/sujaykumarsuman/xlearn/internal/platform/runnerapi" {
				t.Errorf("%s imports %s: lint is a stdlib + runnerapi leaf", f, p)
			}
		}
	}
}

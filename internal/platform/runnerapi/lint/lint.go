// Package lint is the per-profile source lint for the launch languages (t3 §2.4 A5 and A18,
// t4 §5.6 #15; m3-04 task 4): which files a job may carry, and which directives, includes and
// imports the learner's file may use.
//
// The lint is attack-surface reduction, NOT the security boundary — the jail and seccomp are.
// Only a rejection decided before any learner code runs is uncounted (REJECTED): judge runs
// Check before enqueue; the runner's front re-runs CheckNames (names and types only) and
// answers a violation with 400, never a verdict. Evasions (a macro-built `asm`, `__import__`)
// are expected; the jail stops them.
//
// A stdlib-only leaf library (plus runnerapi's File type): judge and the runner both link it.
package lint

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/sujaykumarsuman/xlearn/internal/platform/runnerapi"
)

// Violation is one lint failure. Judge maps any violation to REJECTED with a generic message;
// the detail is for logs and authors, never for the learner.
type Violation struct {
	File string `json:"file"`
	Line int    `json:"line,omitempty"`
	Rule string `json:"rule"`
	Msg  string `json:"msg"`
}

func (v Violation) Error() string {
	if v.Line > 0 {
		return fmt.Sprintf("%s:%d: %s (%s)", v.File, v.Line, v.Msg, v.Rule)
	}
	return fmt.Sprintf("%s: %s (%s)", v.File, v.Msg, v.Rule)
}

// Rules (stable ids; golden fixtures pin one per rule).
const (
	RuleProfile   = "profile"         // not a launch-language profile
	RuleFileName  = "file_name"       // a file the profile doesn't take
	RuleNoLearner = "no_learner_file" // the learner's file is missing
	RuleHidden    = "hidden_file"     // hidden files are for honor profiles only

	RuleGoImport    = "go_import"    // an import outside the allowlist (incl. "C")
	RuleGoDirective = "go_directive" // //go:embed, //line, /*line*/, //export, //go:* but //go:build

	RuleCppInclude   = "cpp_include"   // #include "…", a non-standard <header>, or a computed include
	RuleCppDirective = "cpp_directive" // #include_next, #embed, #line, #pragma but GCC optimize|target, _Pragma
	RuleCppAsm       = "cpp_asm"       // asm, __asm, __asm__
	RuleCppMain      = "cpp_main"      // a learner-defined main

	RulePyImport = "py_import" // an import outside the allowlist, or a relative import
)

// Language maps a profile ("go@1.26", "cpp@g++14", "python@3.13") or a language id ("go",
// "cpp", "python") to the language id; "" if it isn't a launch language.
func Language(profile string) string {
	name, _, _ := strings.Cut(profile, "@")
	switch name {
	case "go", "cpp", "python":
		return name
	}
	return ""
}

// learnerFile and harnessFiles are the only names each language's job may carry
// (internal/platform/harness generates the harness files; m3-04).
var (
	learnerFile  = map[string]string{"go": "solution.go", "cpp": "solution.cpp", "python": "solution.py"}
	harnessFiles = map[string][]string{
		"go":     {"zz_xl_harness.go"},
		"cpp":    {"zz_xl_harness.cpp", "xl_prelude.hpp"},
		"python": {"__main__.py", "xl_prelude.py"},
	}
)

// CheckNames re-checks a job's file names and types for its profile (the runner front, before
// it writes a byte; a violation is 400): every file is the learner's file or one of the
// harness files the profile's generator writes, the learner's file is present, and there are
// no hidden files. Content is judge's (Check).
func CheckNames(profile string, files, hidden []runnerapi.File) []Violation {
	lang := Language(profile)
	if lang == "" {
		return []Violation{{Rule: RuleProfile, Msg: fmt.Sprintf("profile %q is not go, cpp or python", profile)}}
	}
	var out []Violation
	allowed := map[string]bool{learnerFile[lang]: true}
	for _, h := range harnessFiles[lang] {
		allowed[h] = true
	}
	seenLearner := false
	for _, f := range files {
		if err := runnerapi.ValidFileName(f.Path); err != nil {
			out = append(out, Violation{File: f.Path, Rule: RuleFileName, Msg: err.Error()})
			continue
		}
		if !allowed[f.Path] {
			out = append(out, Violation{File: f.Path, Rule: RuleFileName, Msg: fmt.Sprintf("%s jobs carry only %s", lang, strings.Join(sortedKeys(allowed), ", "))})
		}
		if f.Path == learnerFile[lang] {
			seenLearner = true
		}
	}
	if !seenLearner {
		out = append(out, Violation{File: learnerFile[lang], Rule: RuleNoLearner, Msg: "the learner's file is missing"})
	}
	for _, f := range hidden {
		out = append(out, Violation{File: f.Path, Rule: RuleHidden, Msg: "hidden files belong to honor profiles"})
	}
	return out
}

// Check is judge's pre-enqueue lint (a violation is REJECTED, uncounted): CheckNames, then the
// language's content rules over the learner's file. Harness files are judge's own and are
// not linted.
func Check(profile string, files []runnerapi.File) []Violation {
	out := CheckNames(profile, files, nil)
	lang := Language(profile)
	if lang == "" {
		return out
	}
	for _, f := range files {
		if f.Path != learnerFile[lang] {
			continue
		}
		switch lang {
		case "go":
			out = append(out, checkGo(f)...)
		case "cpp":
			out = append(out, checkCpp(f)...)
		case "python":
			out = append(out, checkPython(f)...)
		}
	}
	return out
}

// ---- allowlists (public; t3 §6.2) ----

// GoImports is the go@1.26 import allowlist (path.Match patterns). `unicode` itself rides with
// `unicode/*`.
var GoImports = []string{
	"fmt", "sort", "slices", "maps", "strings", "strconv", "math", "math/bits", "math/rand/v2",
	"container/*", "unicode", "unicode/*", "bytes", "errors", "cmp", "iter",
}

// PythonImports is the python@3.13 import allowlist (top-level modules). `__future__` is a
// compile-time switch, not a module the program can use.
var PythonImports = []string{
	"__future__", "abc", "array", "bisect", "cmath", "collections", "copy", "dataclasses", "decimal", "enum",
	"fractions", "functools", "heapq", "itertools", "math", "numbers", "operator", "random", "re", "statistics",
	"string", "sys", "typing",
}

// CppHeaders are the standard-library headers a learner's solution.cpp may include (C++23
// library headers, the C compatibility headers in both spellings, and bits/stdc++.h).
var CppHeaders = func() map[string]bool {
	m := map[string]bool{"bits/stdc++.h": true}
	for _, h := range strings.Fields(`algorithm any array atomic barrier bit bitset charconv chrono codecvt compare
		complex concepts condition_variable coroutine deque exception execution expected filesystem flat_map flat_set
		format forward_list fstream functional future generator initializer_list iomanip ios iosfwd iostream istream
		iterator latch limits list locale map mdspan memory memory_resource mutex new numbers numeric optional ostream
		print queue random ranges ratio regex scoped_allocator semaphore set shared_mutex source_location span
		spanstream sstream stack stacktrace stdexcept stdfloat stop_token streambuf string string_view strstream
		syncstream system_error thread tuple type_traits typeindex typeinfo unordered_map unordered_set utility
		valarray variant vector version`) {
		m[h] = true
	}
	for _, c := range strings.Fields(`assert ctype errno fenv float inttypes limits locale math setjmp signal stdarg
		stddef stdint stdio stdlib string time uchar wchar wctype`) {
		m["c"+c] = true
		m[c+".h"] = true
	}
	return m
}()

func goImportAllowed(p string) bool {
	for _, pat := range GoImports {
		if ok, _ := path.Match(pat, p); ok {
			return true
		}
	}
	return false
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

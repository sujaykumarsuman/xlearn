package harness

import (
	"fmt"
	"strings"

	"github.com/sujaykumarsuman/xlearn/internal/platform/runnerapi"
)

// Starter returns the learner-file skeleton for a language, a harness and a signature: the
// signature with a zero-value body (func-json@1), or the class with its constructor and empty
// methods (class-ops@1). It is used when an item ships no `_starter/` file (t1 §7.1); it
// compiles with the generated harness and fails the samples (the public content check).
func Starter(lang, harness string, s *Sig) (runnerapi.File, error) {
	if s.Harness != harness {
		return runnerapi.File{}, fmt.Errorf("harness: signature parsed for %s, asked for %s", s.Harness, harness)
	}
	if err := checkNames(lang, s); err != nil {
		return runnerapi.File{}, err
	}
	var b strings.Builder
	switch lang {
	case "go":
		goStarter(&b, s)
	case "cpp":
		cppStarter(&b, s)
	case "python":
		pyStarter(&b, s)
	default:
		return runnerapi.File{}, fmt.Errorf("harness: unknown language %q (have %v)", lang, Languages)
	}
	return runnerapi.File{Path: LearnerFile(lang), Data: []byte(b.String())}, nil
}

// ---- Go ----

func goZero(t Type) string {
	if t.Dims > 0 {
		return "nil"
	}
	switch t.Base {
	case BaseInt, BaseInt64, BaseFloat64:
		return "0"
	case BaseBool:
		return "false"
	case BaseString:
		return `""`
	}
	return "nil"
}

func goParams(ps []Param) string {
	out := make([]string, 0, len(ps))
	for _, p := range ps {
		out = append(out, p.Name+" "+GoType(p.Type))
	}
	return strings.Join(out, ", ")
}

func goStarter(b *strings.Builder, s *Sig) {
	b.WriteString("package main\n")
	if !s.Class() {
		fmt.Fprintf(b, "\nfunc %s(%s) %s {\n\treturn %s\n}\n", s.Name, goParams(s.Params), GoType(s.Returns), goZero(s.Returns))
		return
	}
	fmt.Fprintf(b, "\ntype %s struct {\n}\n", s.Name)
	fmt.Fprintf(b, "\nfunc Constructor(%s) %s {\n\treturn %s{}\n}\n", goParams(s.Params), s.Name, s.Name)
	for _, o := range s.Ops {
		if o.Returns.Base == BaseVoid {
			fmt.Fprintf(b, "\nfunc (this *%s) %s(%s) {\n}\n", s.Name, o.Name, goParams(o.Params))
			continue
		}
		fmt.Fprintf(b, "\nfunc (this *%s) %s(%s) %s {\n\treturn %s\n}\n", s.Name, o.Name, goParams(o.Params), GoType(o.Returns), goZero(o.Returns))
	}
}

// ---- C++ ----

func cppParams(ps []Param) string {
	out := make([]string, 0, len(ps))
	for _, p := range ps {
		ty := CppType(p.Type)
		if p.Type.Dims > 0 {
			ty += "&"
		}
		out = append(out, ty+" "+p.Name)
	}
	return strings.Join(out, ", ")
}

func cppStarter(b *strings.Builder, s *Sig) {
	if !s.Class() {
		fmt.Fprintf(b, "class Solution {\npublic:\n    %s %s(%s) {\n        return {};\n    }\n};\n",
			CppType(s.Returns), s.Name, cppParams(s.Params))
		return
	}
	fmt.Fprintf(b, "class %s {\npublic:\n    %s(%s) {\n    }\n", s.Name, s.Name, cppParams(s.Params))
	for _, o := range s.Ops {
		if o.Returns.Base == BaseVoid {
			fmt.Fprintf(b, "\n    void %s(%s) {\n    }\n", o.Name, cppParams(o.Params))
			continue
		}
		fmt.Fprintf(b, "\n    %s %s(%s) {\n        return {};\n    }\n", CppType(o.Returns), o.Name, cppParams(o.Params))
	}
	b.WriteString("};\n")
}

// ---- Python ----

func pyZero(t Type) string {
	if t.Dims > 0 {
		return "[]"
	}
	switch t.Base {
	case BaseInt, BaseInt64:
		return "0"
	case BaseFloat64:
		return "0.0"
	case BaseBool:
		return "False"
	case BaseString:
		return `""`
	}
	return "None"
}

func pyParams(ps []Param) string {
	out := []string{"self"}
	for _, p := range ps {
		out = append(out, p.Name+": "+PythonType(p.Type))
	}
	return strings.Join(out, ", ")
}

func pyStarter(b *strings.Builder, s *Sig) {
	var all []Type
	for _, p := range s.Params {
		all = append(all, p.Type)
	}
	all = append(all, s.Returns)
	for _, o := range s.Ops {
		all = append(all, o.Returns)
		for _, p := range o.Params {
			all = append(all, p.Type)
		}
	}
	var imports []string
	for _, name := range []string{"List", "Optional"} {
		for _, t := range all {
			if t.Base != 0 && strings.Contains(PythonType(t), name+"[") {
				imports = append(imports, name)
				break
			}
		}
	}
	if len(imports) > 0 {
		fmt.Fprintf(b, "from typing import %s\n\n\n", strings.Join(imports, ", "))
	}
	if !s.Class() {
		fmt.Fprintf(b, "class Solution:\n    def %s(%s) -> %s:\n        return %s\n", s.Name, pyParams(s.Params), PythonType(s.Returns), pyZero(s.Returns))
		return
	}
	fmt.Fprintf(b, "class %s:\n    def __init__(%s):\n        pass\n", s.Name, pyParams(s.Params))
	for _, o := range s.Ops {
		if o.Returns.Base == BaseVoid {
			fmt.Fprintf(b, "\n    def %s(%s) -> None:\n        pass\n", o.Name, pyParams(o.Params))
			continue
		}
		fmt.Fprintf(b, "\n    def %s(%s) -> %s:\n        return %s\n", o.Name, pyParams(o.Params), PythonType(o.Returns), pyZero(o.Returns))
	}
}

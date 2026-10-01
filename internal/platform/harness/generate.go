package harness

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"text/template"

	"github.com/sujaykumarsuman/xlearn/internal/platform/runnerapi"
)

// The harness files are generated per signature from public text/template sources, with no
// reflection in the generated code (t3 §6.2). Go (m3-02):
//
//	learner file   solution.go       package main: the function (func-json@1), or the class
//	                                 with `func Constructor(<params>) <Class>` (or *<Class>)
//	                                 and one method per op (class-ops@1)
//	harness file   zz_xl_harness.go  package main: func main, the node types the signature
//	                                 uses, a tiny JSON reader/writer, recover() → panic class
//
// Both are built together as one ad-hoc package (`go build solution.go zz_xl_harness.go`).
// m3-04 adds the C++ (zz_xl_harness.cpp + xl_prelude.hpp) and Python (__main__.py +
// xl_prelude.py) generators beside these, keeping every byte on fd 3 and fd 4.

// Generated file names.
const (
	GoLearnerFile = "solution.go"
	GoHarnessFile = "zz_xl_harness.go"
)

// Languages with a generator today. C++ and Python arrive in m3-04.
var Languages = []string{"go"}

// ErrLanguagePending is returned for a language whose harness m3-04 adds (cpp, python).
var ErrLanguagePending = errors.New("harness: no generator for this language yet (m3-04 adds C++ and Python)")

//go:embed templates/go/*.tmpl
var templateFS embed.FS

var goTemplates = template.Must(template.New("go").ParseFS(templateFS, "templates/go/*.tmpl"))

// Generate returns the harness files for a language, a harness and a signature. The
// learner's own file is not among them.
func Generate(lang, harness string, s *Sig) ([]runnerapi.File, error) {
	if s.Harness != harness {
		return nil, fmt.Errorf("harness: signature parsed for %s, asked for %s", s.Harness, harness)
	}
	switch lang {
	case "go":
		b, err := generateGo(s)
		if err != nil {
			return nil, err
		}
		return []runnerapi.File{{Path: GoHarnessFile, Data: b}}, nil
	case "cpp", "py", "python":
		return nil, ErrLanguagePending
	}
	return nil, fmt.Errorf("harness: unknown language %q", lang)
}

type goArg struct{ Dec string }

type goOp struct {
	Name   string
	Params []goArg
	Void   bool
	Enc    string
}

type goData struct {
	Harness                       string
	Name                          string
	Params                        []goArg
	Enc                           string
	Ops                           []goOp
	UsesList, UsesTree, UsesGraph bool
}

func generateGo(s *Sig) ([]byte, error) {
	d := goData{Harness: s.Harness, Name: s.Name}
	use := func(t Type) {
		switch t.Base {
		case BaseListNode:
			d.UsesList = true
		case BaseTreeNode:
			d.UsesTree = true
		case BaseGraphNode:
			d.UsesGraph = true
		}
	}
	args := func(ps []Param) []goArg {
		out := make([]goArg, 0, len(ps))
		for _, p := range ps {
			use(p.Type)
			out = append(out, goArg{Dec: goDec(p.Type)})
		}
		return out
	}
	d.Params = args(s.Params)
	name := "func-json.tmpl"
	if s.Class() {
		name = "class-ops.tmpl"
		for _, o := range s.Ops {
			op := goOp{Name: o.Name, Params: args(o.Params), Void: o.Returns.Base == BaseVoid}
			if !op.Void {
				use(o.Returns)
				op.Enc = goEnc(o.Returns)
			}
			d.Ops = append(d.Ops, op)
		}
	} else {
		use(s.Returns)
		d.Enc = goEnc(s.Returns)
	}
	var buf bytes.Buffer
	if err := goTemplates.ExecuteTemplate(&buf, name, d); err != nil {
		return nil, fmt.Errorf("harness: go template: %w", err)
	}
	return buf.Bytes(), nil
}

var goScalarDec = map[Base]string{
	BaseInt: "xlInt", BaseInt64: "xlInt64", BaseFloat64: "xlFloat", BaseBool: "xlBool", BaseString: "xlString",
	BaseListNode: "xlListNode", BaseTreeNode: "xlTreeNode", BaseGraphNode: "xlGraphNode",
}

var goScalarEnc = map[Base]string{
	BaseInt: "xlEncInt", BaseInt64: "xlEncInt64", BaseFloat64: "xlEncFloat", BaseBool: "xlEncBool", BaseString: "xlEncString",
	BaseListNode: "xlEncListNode", BaseTreeNode: "xlEncTreeNode", BaseGraphNode: "xlEncGraphNode",
}

// goDec is a Go expression of type func(*xlParser) T.
func goDec(t Type) string {
	if t.Dims == 0 {
		return goScalarDec[t.Base]
	}
	return fmt.Sprintf("func(p *xlParser) %s { return xlSlice(p, %s) }", GoType(t), goDec(t.Elem()))
}

// goEnc is a Go expression of type func(*xlWriter, T).
func goEnc(t Type) string {
	if t.Dims == 0 {
		return goScalarEnc[t.Base]
	}
	return fmt.Sprintf("func(w *xlWriter, v %s) { xlEncSlice(w, v, %s) }", GoType(t), goEnc(t.Elem()))
}

// GoBuildEnv is the compile environment every Go harness build uses (m3-04's compile env,
// as packlint's executor applies it in a container): no cgo, no network, no toolchain
// switch, no telemetry, a read-only module mode.
var GoBuildEnv = []string{
	"CGO_ENABLED=0", "GOTOOLCHAIN=local", "GOPROXY=off", "GOFLAGS=-mod=readonly", "GOTELEMETRY=off", "GOENV=off",
}

// GoBuildFlags are the `go build` flags (t3 §6.2; the std-cache seed only hits with the
// same flags).
var GoBuildFlags = []string{"-trimpath", "-buildvcs=false"}

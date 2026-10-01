package harness

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"text/template"

	"github.com/sujaykumarsuman/xlearn/internal/platform/runnerapi"
)

// The harness files are generated per signature from public text/template sources, with no
// reflection in the generated code (t3 §6.2):
//
//	go      learner  solution.go        package main: the function (func-json@1), or the class
//	                                    with `func Constructor(<params>) <Class>` (or *<Class>)
//	                                    and one method per op (class-ops@1)
//	        harness  zz_xl_harness.go   package main: func main, the node types the signature
//	                                    uses, a tiny JSON reader/writer, recover() → panic class
//	cpp     learner  solution.cpp       class Solution with the method (func-json@1), or the
//	                                    class with its constructor and one method per op
//	        harness  zz_xl_harness.cpp  includes xl_prelude.hpp, defines the node types the
//	                                    signature uses, includes solution.cpp, then main
//	                 xl_prelude.hpp     bits/stdc++.h, using namespace std, the JSON reader and
//	                                    writer, exception → panic class (one static file)
//	python  learner  solution.py        class Solution with the method, or the class
//	        harness  __main__.py        installs the node classes, imports solution, one case
//	                 xl_prelude.py      node classes, the JSON reader and writer (one static file)
//
// Go builds learner + harness as one ad-hoc package (`go build solution.go zz_xl_harness.go`),
// C++ compiles zz_xl_harness.cpp alone (it includes the learner file, so diagnostics keep
// solution.cpp:line positions), and Python packs the three files into a zipapp. Every
// generator writes the same fd-4 bytes (README.md is the @1 spec).

// Generated file names.
const (
	GoLearnerFile     = "solution.go"
	GoHarnessFile     = "zz_xl_harness.go"
	CppLearnerFile    = "solution.cpp"
	CppHarnessFile    = "zz_xl_harness.cpp"
	CppPreludeFile    = "xl_prelude.hpp"
	PythonLearnerFile = "solution.py"
	PythonMainFile    = "__main__.py"
	PythonPreludeFile = "xl_prelude.py"
)

// Languages with a generator: the item's `languages[]` values (D20).
var Languages = []string{"go", "cpp", "python"}

// LearnerFile is the learner's file name for a language ("" for an unknown language).
func LearnerFile(lang string) string {
	switch lang {
	case "go":
		return GoLearnerFile
	case "cpp":
		return CppLearnerFile
	case "python":
		return PythonLearnerFile
	}
	return ""
}

//go:embed templates/go/*.tmpl templates/cpp/*.tmpl templates/cpp/xl_prelude.hpp templates/python/*.tmpl templates/python/xl_prelude.py
var templateFS embed.FS

var funcs = template.FuncMap{"cq": strconv.Quote, "q": strconv.Quote}

var (
	goTemplates  = template.Must(template.New("go").Funcs(funcs).ParseFS(templateFS, "templates/go/*.tmpl"))
	cppTemplates = template.Must(template.New("cpp").Funcs(funcs).ParseFS(templateFS, "templates/cpp/*.tmpl"))
	pyTemplates  = template.Must(template.New("python").Funcs(funcs).ParseFS(templateFS, "templates/python/*.tmpl"))
	cppPrelude   = mustRead("templates/cpp/" + CppPreludeFile)
	pyPrelude    = mustRead("templates/python/" + PythonPreludeFile)
)

func mustRead(name string) []byte {
	b, err := templateFS.ReadFile(name)
	if err != nil {
		panic(err)
	}
	return b
}

// Sources returns every public template and prelude by its path under templates/ (the
// runner hashes them into ProfileSHA, so a one-byte change moves it).
func Sources(lang string) map[string][]byte {
	out := map[string][]byte{}
	_ = fs.WalkDir(templateFS, "templates/"+lang, func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			b, _ := templateFS.ReadFile(p)
			out[strings.TrimPrefix(p, "templates/")] = b
		}
		return err
	})
	return out
}

// Generate returns the harness files for a language, a harness and a signature. The
// learner's own file is not among them.
func Generate(lang, harness string, s *Sig) ([]runnerapi.File, error) {
	if s.Harness != harness {
		return nil, fmt.Errorf("harness: signature parsed for %s, asked for %s", s.Harness, harness)
	}
	if err := checkNames(lang, s); err != nil {
		return nil, err
	}
	switch lang {
	case "go":
		b, err := generateGo(s)
		if err != nil {
			return nil, err
		}
		return []runnerapi.File{{Path: GoHarnessFile, Data: b}}, nil
	case "cpp":
		b, err := generateCpp(s)
		if err != nil {
			return nil, err
		}
		return []runnerapi.File{{Path: CppHarnessFile, Data: b}, {Path: CppPreludeFile, Data: bytes.Clone(cppPrelude)}}, nil
	case "python":
		b, err := generatePython(s)
		if err != nil {
			return nil, err
		}
		return []runnerapi.File{{Path: PythonMainFile, Data: b}, {Path: PythonPreludeFile, Data: bytes.Clone(pyPrelude)}}, nil
	}
	return nil, fmt.Errorf("harness: unknown language %q (have %v)", lang, Languages)
}

var identRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// Keywords a generated name must not be, per language (the schema only checks Go identifiers).
var keywords = map[string][]string{
	"go": {"break", "case", "chan", "const", "continue", "default", "defer", "else", "fallthrough", "for", "func",
		"go", "goto", "if", "import", "interface", "map", "package", "range", "return", "select", "struct", "switch",
		"type", "var"},
	"cpp": {"alignas", "alignof", "and", "and_eq", "asm", "auto", "bitand", "bitor", "bool", "break", "case", "catch",
		"char", "char8_t", "char16_t", "char32_t", "class", "co_await", "co_return", "co_yield", "compl", "concept",
		"const", "consteval", "constexpr", "constinit", "const_cast", "continue", "decltype", "default", "delete",
		"do", "double", "dynamic_cast", "else", "enum", "explicit", "export", "extern", "false", "float", "for",
		"friend", "goto", "if", "inline", "int", "long", "main", "mutable", "namespace", "new", "noexcept", "not",
		"not_eq", "nullptr", "operator", "or", "or_eq", "private", "protected", "public", "register",
		"reinterpret_cast", "requires", "return", "short", "signed", "sizeof", "static", "static_assert",
		"static_cast", "struct", "switch", "template", "this", "thread_local", "throw", "true", "try", "typedef",
		"typeid", "typename", "union", "unsigned", "using", "virtual", "void", "volatile", "wchar_t", "while", "xor",
		"xor_eq"},
	"python": {"False", "None", "True", "and", "as", "assert", "async", "await", "break", "class", "continue", "def",
		"del", "elif", "else", "except", "finally", "for", "from", "global", "if", "import", "in", "is", "lambda",
		"nonlocal", "not", "or", "pass", "raise", "return", "try", "while", "with", "yield"},
}

// reservedNames are names the generated code already uses where the learner's names live:
// the entry point (main), the node types, and Python's `self`.
var reservedNames = map[string][]string{
	"go":     {"main", "init", "ListNode", "TreeNode", "Node"},
	"cpp":    {"ListNode", "TreeNode", "Node"},
	"python": {"self", "ListNode", "TreeNode", "Node"},
}

// checkNames refuses a function, class, op or parameter name the target language can't take:
// a keyword or a name the generated code uses; the function, class and op names (which the
// generated code calls) also may not start with the harness's xl prefix.
func checkNames(lang string, s *Sig) error {
	check := func(what, n string, called bool) error {
		switch {
		case !identRe.MatchString(n):
			return fmt.Errorf("harness: %s %q is not an identifier", what, n)
		case called && strings.HasPrefix(n, "xl"):
			return fmt.Errorf("harness: %s %q: names starting with xl are reserved for the harness", what, n)
		case slices.Contains(keywords[lang], n), slices.Contains(reservedNames[lang], n):
			return fmt.Errorf("harness: %s %q is a reserved word in %s", what, n, lang)
		}
		return nil
	}
	if err := check("name", s.Name, true); err != nil {
		return err
	}
	for _, p := range s.Params {
		if err := check("param", p.Name, false); err != nil {
			return err
		}
	}
	for _, o := range s.Ops {
		if err := check("op", o.Name, true); err != nil {
			return err
		}
		for _, p := range o.Params {
			if err := check("param", p.Name, false); err != nil {
				return err
			}
		}
	}
	return nil
}

// uses records which node types a signature touches.
type uses struct{ List, Tree, Graph bool }

func (u *uses) add(t Type) {
	switch t.Base {
	case BaseListNode:
		u.List = true
	case BaseTreeNode:
		u.Tree = true
	case BaseGraphNode:
		u.Graph = true
	}
}

func sigUses(s *Sig) uses {
	var u uses
	for _, p := range s.Params {
		u.add(p.Type)
	}
	u.add(s.Returns)
	for _, o := range s.Ops {
		for _, p := range o.Params {
			u.add(p.Type)
		}
		u.add(o.Returns)
	}
	return u
}

// ---- Go (m3-02; its bytes are the @1 reference) ----

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

// ---- C++ (m3-04) ----

type cppArg struct{ Type string }

type cppOp struct {
	Name   string
	Params []cppArg
	Void   bool
	Ret    string
}

type cppData struct {
	Harness                       string
	Name                          string
	Params                        []cppArg
	Ret                           string
	Ops                           []cppOp
	UsesList, UsesTree, UsesGraph bool
}

func generateCpp(s *Sig) ([]byte, error) {
	u := sigUses(s)
	d := cppData{Harness: s.Harness, Name: s.Name, UsesList: u.List, UsesTree: u.Tree, UsesGraph: u.Graph}
	args := func(ps []Param) []cppArg {
		out := make([]cppArg, 0, len(ps))
		for _, p := range ps {
			out = append(out, cppArg{Type: cppHarnessType(p.Type)})
		}
		return out
	}
	d.Params = args(s.Params)
	name := "func-json.tmpl"
	if s.Class() {
		name = "class-ops.tmpl"
		for _, o := range s.Ops {
			op := cppOp{Name: o.Name, Params: args(o.Params), Void: o.Returns.Base == BaseVoid}
			if !op.Void {
				op.Ret = cppHarnessType(o.Returns)
			}
			d.Ops = append(d.Ops, op)
		}
	} else {
		d.Ret = cppHarnessType(s.Returns)
	}
	var buf bytes.Buffer
	if err := cppTemplates.ExecuteTemplate(&buf, name, d); err != nil {
		return nil, fmt.Errorf("harness: cpp template: %w", err)
	}
	return buf.Bytes(), nil
}

// cppHarnessType is a type's fully qualified C++ spelling in the generated harness.
func cppHarnessType(t Type) string {
	var base string
	switch t.Base {
	case BaseInt:
		base = "int"
	case BaseInt64:
		base = "long long"
	case BaseFloat64:
		base = "double"
	case BaseBool:
		base = "bool"
	case BaseString:
		base = "std::string"
	case BaseListNode:
		base = "ListNode*"
	case BaseTreeNode:
		base = "TreeNode*"
	case BaseGraphNode:
		base = "Node*"
	case BaseVoid:
		return "void"
	}
	for i := 0; i < t.Dims; i++ {
		base = "std::vector<" + base + ">"
	}
	return base
}

// CppType is the learner-facing C++ spelling of a type (the C++ column of the type mapping,
// with `using namespace std`): int, long long, double, bool, string, vector<T>, ListNode*,
// TreeNode*, Node*.
func CppType(t Type) string {
	return strings.ReplaceAll(cppHarnessType(t), "std::", "")
}

// ---- Python (m3-04) ----

type pyArg struct{ Dec string }

type pyOp struct {
	Name   string
	Params []pyArg
	Void   bool
	Enc    string
}

type pyRequire struct {
	Name  string
	Arity int
}

type pyData struct {
	Harness  string
	Name     string
	Class    string
	Requires []pyRequire
	Nodes    []string
	Params   []pyArg
	Enc      string
	Ops      []pyOp
}

func generatePython(s *Sig) ([]byte, error) {
	u := sigUses(s)
	d := pyData{Harness: s.Harness, Name: s.Name}
	if u.List {
		d.Nodes = append(d.Nodes, "ListNode")
	}
	if u.Tree {
		d.Nodes = append(d.Nodes, "TreeNode")
	}
	if u.Graph {
		d.Nodes = append(d.Nodes, "Node")
	}
	args := func(ps []Param) []pyArg {
		out := make([]pyArg, 0, len(ps))
		for _, p := range ps {
			out = append(out, pyArg{Dec: "dec_" + pyCodecName(p.Type)})
		}
		return out
	}
	d.Params = args(s.Params)
	name := "func-json.tmpl"
	if s.Class() {
		name = "class-ops.tmpl"
		d.Class = s.Name
		d.Requires = append(d.Requires, pyRequire{Name: "__init__", Arity: len(s.Params)})
		for _, o := range s.Ops {
			op := pyOp{Name: o.Name, Params: args(o.Params), Void: o.Returns.Base == BaseVoid}
			if !op.Void {
				op.Enc = "enc_" + pyCodecName(o.Returns)
			}
			d.Ops = append(d.Ops, op)
			d.Requires = append(d.Requires, pyRequire{Name: o.Name, Arity: len(o.Params)})
		}
	} else {
		d.Class = "Solution"
		d.Requires = []pyRequire{{Name: s.Name, Arity: len(s.Params)}}
		d.Enc = "enc_" + pyCodecName(s.Returns)
	}
	var buf bytes.Buffer
	if err := pyTemplates.ExecuteTemplate(&buf, name, d); err != nil {
		return nil, fmt.Errorf("harness: python template: %w", err)
	}
	return buf.Bytes(), nil
}

// pyCodecName names a type's xl_prelude codec pair (dec_<n> / enc_<n>): the registry base,
// then _1 or _2 for one or two [] suffixes.
func pyCodecName(t Type) string {
	n := baseNames[t.Base]
	if t.Dims > 0 {
		n += "_" + strconv.Itoa(t.Dims)
	}
	return n
}

// PythonType is the learner-facing Python annotation of a type (the Python column of the type
// mapping, typing.List / typing.Optional spelling).
func PythonType(t Type) string {
	var base string
	switch t.Base {
	case BaseInt, BaseInt64:
		base = "int"
	case BaseFloat64:
		base = "float"
	case BaseBool:
		base = "bool"
	case BaseString:
		base = "str"
	case BaseListNode:
		base = "Optional[ListNode]"
	case BaseTreeNode:
		base = "Optional[TreeNode]"
	case BaseGraphNode:
		base = "Optional['Node']"
	case BaseVoid:
		return "None"
	}
	for i := 0; i < t.Dims; i++ {
		base = "List[" + base + "]"
	}
	return base
}

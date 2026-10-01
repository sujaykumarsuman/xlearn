package harness

import (
	"errors"
	"strings"
	"testing"

	"github.com/sujaykumarsuman/xlearn/internal/course"
)

// Every value here is synthetic and hand-made.

func TestParseTypeClosedSet(t *testing.T) {
	ok := []string{
		"int", "int64", "float64", "bool", "string",
		"int[]", "int64[]", "float64[]", "bool[]", "string[]",
		"int[][]", "int64[][]", "float64[][]", "bool[][]", "string[][]",
		"ListNode", "ListNode[]", "TreeNode", "TreeNode[]", "GraphNode", "void",
	}
	for _, s := range ok {
		ty, err := ParseType(s)
		if err != nil {
			t.Errorf("ParseType(%q): %v", s, err)
			continue
		}
		if ty.String() != s {
			t.Errorf("ParseType(%q).String() = %q", s, ty.String())
		}
	}
	bad := []string{"", "int[][][]", "ListNode[][]", "TreeNode[][]", "GraphNode[]", "uint", "Int", "map", "int[", "void[]", "char"}
	for _, s := range bad {
		if _, err := ParseType(s); !errors.Is(err, ErrType) {
			t.Errorf("ParseType(%q) = %v, want ErrType", s, err)
		}
	}
}

func TestGoTypeMapping(t *testing.T) {
	cases := map[string]string{
		"int": "int", "int64": "int64", "float64": "float64", "bool": "bool", "string": "string",
		"int[][]": "[][]int", "ListNode": "*ListNode", "TreeNode[]": "[]*TreeNode", "GraphNode": "*Node", "void": "",
	}
	for in, want := range cases {
		ty, err := ParseType(in)
		if err != nil {
			t.Fatal(err)
		}
		if got := GoType(ty); got != want {
			t.Errorf("GoType(%s) = %q, want %q", in, got, want)
		}
	}
}

// canonGolden: (type, input JSON) -> canonical bytes.
var canonGolden = []struct{ typ, in, want string }{
	{"int", "  42 ", "42"},
	{"int", "-2147483648", "-2147483648"},
	{"int64", "9223372036854775807", "9223372036854775807"},
	{"bool", "true", "true"},
	{"string", `"a\"b\\c"`, `"a\"b\\c"`},
	{"string", `"tab\there\u0001"`, `"tab\u0009here\u0001"`},
	{"string", `"é /\/"`, "\"é //\""},
	{"string", `"😀"`, "\"\U0001F600\""},
	{"int[]", "[ 3, -1 ,0 ]", "[3,-1,0]"},
	{"int[][]", "[[],[1],[2,3]]", "[[],[1],[2,3]]"},
	{"string[]", `["x","y"]`, `["x","y"]`},
	{"ListNode", "[1,2,3]", "[1,2,3]"},
	{"ListNode", "[]", "[]"},
	{"ListNode[]", "[[1],[],[2,3]]", "[[1],[],[2,3]]"},
	{"TreeNode", "[]", "[]"},
	{"TreeNode", "[1,null,2]", "[1,null,2]"},
	{"TreeNode", "[1,null,2,null,null]", "[1,null,2]"},
	{"TreeNode", "[1,null,2,null,3]", "[1,null,2,null,3]"},
	{"TreeNode", "[5,3,8,1,4,null,9]", "[5,3,8,1,4,null,9]"},
	{"TreeNode[]", "[[1],[],[2,null,3]]", "[[1],[],[2,null,3]]"},
	{"GraphNode", "[]", "[]"},
	{"GraphNode", "[[2,4],[1,3],[2,4],[1,3]]", "[[2,4],[1,3],[2,4],[1,3]]"},
	{"GraphNode", "[[1],[]]", "[[1],[]]"}, // a self-loop and an isolated node
}

func TestCanonicalGolden(t *testing.T) {
	for _, c := range canonGolden {
		ty, err := ParseType(c.typ)
		if err != nil {
			t.Fatal(err)
		}
		v, err := DecodeValue(ty, []byte(c.in))
		if err != nil {
			t.Errorf("%s %s: %v", c.typ, c.in, err)
			continue
		}
		got, err := Canonical(v)
		if err != nil {
			t.Errorf("%s %s: Canonical: %v", c.typ, c.in, err)
			continue
		}
		if string(got) != c.want {
			t.Errorf("%s %s: canonical %s, want %s", c.typ, c.in, got, c.want)
		}
		// Round trip: the canonical bytes decode to the same value.
		v2, err := DecodeValue(ty, got)
		if err != nil || !Equal(v, v2) {
			t.Errorf("%s %s: round trip %v %v", c.typ, c.in, v2, err)
		}
	}
}

func TestDecodeValueRejects(t *testing.T) {
	bad := []struct{ typ, in string }{
		{"int", "2147483648"}, {"int", "-2147483649"}, {"int", "1.0"}, {"int", "1e3"}, {"int", "-0"}, {"int", `"1"`},
		{"int", "01"}, {"int", "1 2"}, {"int64", "9223372036854775808"},
		{"float64", "1e400"}, {"float64", `"NaN"`}, {"bool", "1"}, {"string", "1"},
		{"string", "\"\xff\""}, {"string", "\"raw\ncontrol\""}, {"string", `"\ud800"`}, {"string", `"\x"`},
		{"int[]", "[1,]"}, {"int[]", "[1"}, {"int[]", "{}"}, {"int[][]", "[1]"},
		{"ListNode", "[1,null]"}, {"TreeNode", "[null,1]"}, {"TreeNode", "[1,null,null,2]"},
		{"TreeNode", "[1,2,3,null,null,null,null,4]"},
		{"GraphNode", "[[0]]"}, {"GraphNode", "[[3],[1]]"}, {"GraphNode", "[1]"},
		{"void", "1"},
	}
	for _, c := range bad {
		ty, err := ParseType(c.typ)
		if err != nil {
			t.Fatal(err)
		}
		if v, err := DecodeValue(ty, []byte(c.in)); err == nil {
			t.Errorf("DecodeValue(%s, %q) = %v, want an error", c.typ, c.in, v)
		}
	}
	// Depth.
	deep := strings.Repeat("[", MaxDepth+2) + strings.Repeat("]", MaxDepth+2)
	if _, err := DecodeValue(Type{Base: BaseInt, Dims: 2}, []byte(deep)); err == nil {
		t.Error("nesting past MaxDepth decoded")
	}
	// Duplicate keys and trailing data in the generic reader.
	for _, s := range []string{`{"a":1,"a":2}`, `{"a":1} x`, `[1] [2]`} {
		if _, err := CanonicalJSON([]byte(s)); !errors.Is(err, ErrJSON) {
			t.Errorf("CanonicalJSON(%q) = %v, want ErrJSON", s, err)
		}
	}
}

func TestFloatsAreNotCanonical(t *testing.T) {
	v, err := DecodeValue(Type{Base: BaseFloat64, Dims: 1}, []byte("[0.1, 3, -0.0, 1e21, 1e-7]"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Canonical(v); !errors.Is(err, ErrNotCanonical) {
		t.Fatalf("Canonical(float[]) = %v, want ErrNotCanonical", err)
	}
	if got := string(AppendJSON(nil, v)); got != "[0.1,3,0,1e+21,1e-07]" {
		t.Errorf("bytes-mode floats %s", got)
	}
}

func TestCanonicalJSONSortsKeys(t *testing.T) {
	got, err := CanonicalJSON([]byte(` {"b": [1, 2.50, 3e2], "a": {"y": "A", "x": null}, "c": 18446744073709551615} `))
	if err != nil {
		t.Fatal(err)
	}
	want := `{"a":{"x":null,"y":"A"},"b":[1,2.5,300],"c":18446744073709551615}`
	if string(got) != want {
		t.Errorf("CanonicalJSON = %s, want %s", got, want)
	}
}

func sig(t *testing.T, harness string, s course.Signature) *Sig {
	t.Helper()
	out, err := ParseSig(harness, &s)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func funcSig(t *testing.T) *Sig {
	return sig(t, FuncJSON, course.Signature{
		Mode: "function", Name: "pairSum",
		Params:  []course.Param{{Name: "nums", Type: "int[]"}, {Name: "target", Type: "int"}},
		Returns: "int[]",
	})
}

func classSig(t *testing.T) *Sig {
	return sig(t, ClassOps, course.Signature{
		Mode: "class", Name: "Shelf",
		Params: []course.Param{{Name: "capacity", Type: "int"}},
		Ops: []course.Op{
			{Name: "Put", Params: []course.Param{{Name: "key", Type: "int"}, {Name: "label", Type: "string"}}},
			{Name: "Get", Params: []course.Param{{Name: "key", Type: "int"}}, Returns: "string"},
		},
	})
}

func TestParseSigRules(t *testing.T) {
	bad := []struct {
		harness string
		s       course.Signature
	}{
		{ClassOps, course.Signature{Mode: "function", Name: "f", Returns: "int"}},
		{FuncJSON, course.Signature{Mode: "class", Name: "C", Ops: []course.Op{{Name: "x"}}}},
		{"func-json@2", course.Signature{Mode: "function", Name: "f", Returns: "int"}},
		{FuncJSON, course.Signature{Mode: "function", Name: "f"}}, // void function
		{FuncJSON, course.Signature{Mode: "function", Name: "f", Returns: "int[][][]"}},
		{FuncJSON, course.Signature{Mode: "function", Name: "f", Params: []course.Param{{Name: "a", Type: "void"}}, Returns: "int"}},
		{ClassOps, course.Signature{Mode: "class", Name: "C"}},
		{ClassOps, course.Signature{Mode: "class", Name: "C", Ops: []course.Op{{Name: "C"}}}},
		{ClassOps, course.Signature{Mode: "class", Name: "C", Ops: []course.Op{{Name: "x"}, {Name: "x"}}}},
	}
	for i, c := range bad {
		if _, err := ParseSig(c.harness, &c.s); err == nil {
			t.Errorf("case %d: ParseSig accepted %+v", i, c.s)
		}
	}
}

func TestCanonicalInput(t *testing.T) {
	f := funcSig(t)
	got, err := CanonicalInput(f, []byte(` { "args" : [ [1, 2, 3], 4 ] } `))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != `{"args":[[1,2,3],4]}` {
		t.Errorf("func input %s", got)
	}
	frame, _ := EncodeInput(f, got)
	if p, err := SplitFrame(frame); err != nil || string(p) != string(got) || len(frame) != len(got)+4 {
		t.Errorf("EncodeInput frame %q %v", frame, err)
	}
	for _, s := range []string{`{"args":[[1],2,3]}`, `{"args":[[1]]}`, `{"args":[[1],2],"x":1}`, `{"args":[["1"],2]}`, `[]`} {
		if _, err := CanonicalInput(f, []byte(s)); err == nil {
			t.Errorf("func input %s accepted", s)
		}
	}

	c := classSig(t)
	got, err = CanonicalInput(c, []byte(`{"ops":["Shelf","Put","Get"],"args":[[2],[1,"a"],[1]]}`))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != `{"args":[[2],[1,"a"],[1]],"ops":["Shelf","Put","Get"]}` {
		t.Errorf("class input %s", got)
	}
	for _, s := range []string{
		`{"ops":["Put"],"args":[[1,"a"]]}`,           // no constructor
		`{"ops":["Shelf","Nope"],"args":[[2],[]]}`,   // unknown op
		`{"ops":["Shelf","Get"],"args":[[2]]}`,       // arity of lists
		`{"ops":["Shelf","Get"],"args":[[2],["k"]]}`, // arg type
		`{"ops":["Shelf","Shelf"],"args":[[2],[2]]}`, // constructor twice
		`{"ops":["Shelf"],"args":[[2]],"ctor":[2]}`,  // unknown key
	} {
		if _, err := CanonicalInput(c, []byte(s)); err == nil {
			t.Errorf("class input %s accepted", s)
		}
	}
}

func TestDecodeOutput(t *testing.T) {
	f := funcSig(t)
	v, err := DecodeOutput(f, AppendFrame(nil, []byte(`{"ok":[0,2]}`)))
	if err != nil || !Equal(v, ListV(IntV(0), IntV(2))) {
		t.Fatalf("ok frame: %v %v", v, err)
	}
	var pe *PanicError
	if _, err := DecodeOutput(f, AppendFrame(nil, []byte(`{"panic":"index_out_of_range"}`))); !errors.As(err, &pe) || pe.Class != "index_out_of_range" {
		t.Errorf("panic frame: %v", err)
	}
	var he *HarnessError
	if _, err := DecodeOutput(f, AppendFrame(nil, []byte(`{"error":"cycle"}`))); !errors.As(err, &he) || he.Code != "cycle" {
		t.Errorf("error frame: %v", err)
	}
	for _, b := range [][]byte{
		nil, {0, 0, 0}, AppendFrame(nil, []byte(`{"ok":[0,2]}`))[:8], append(AppendFrame(nil, []byte(`{"ok":[]}`)), 'x'),
		AppendFrame(nil, []byte(`{"ok":[],"panic":"other"}`)), AppendFrame(nil, []byte(`{"panic":"segfault"}`)),
		AppendFrame(nil, []byte(`{"error":"oops"}`)), AppendFrame(nil, []byte(`{"result":1}`)),
	} {
		if _, err := DecodeOutput(f, b); !errors.Is(err, ErrFrame) {
			t.Errorf("DecodeOutput(%q) = %v, want ErrFrame", b, err)
		}
	}
	if _, err := DecodeOutput(f, AppendFrame(nil, []byte(`{"ok":["x"]}`))); !errors.Is(err, ErrValue) {
		t.Errorf("ill-typed ok = %v, want ErrValue", err)
	}

	c := classSig(t)
	in, err := DecodeInput(c, []byte(`{"ops":["Shelf","Put","Get"],"args":[[2],[1,"a"],[1]]}`))
	if err != nil {
		t.Fatal(err)
	}
	out := AppendFrame(nil, []byte(`{"ok":[null,null,"a"]}`))
	v, err = DecodeOutputFor(c, in, out)
	if err != nil || !Equal(v, ListV(Null(), Null(), StringV("a"))) {
		t.Fatalf("class ok: %v %v", v, err)
	}
	if _, err := DecodeOutputFor(c, in, AppendFrame(nil, []byte(`{"ok":[null,"a","a"]}`))); err == nil {
		t.Error("a void op's non-null result decoded")
	}
	if _, err := DecodeOutputFor(c, in, AppendFrame(nil, []byte(`{"ok":[null,null]}`))); err == nil {
		t.Error("a short class result decoded")
	}
	if got := OKFrame(v); string(got) != string(out) {
		t.Errorf("OKFrame %q, want %q", got, out)
	}
}

func TestCanonicalizableOutput(t *testing.T) {
	if !CanonicalizableOutput(funcSig(t)) || !CanonicalizableOutput(classSig(t)) {
		t.Error("int/string outputs are canonicalizable")
	}
	fl := sig(t, FuncJSON, course.Signature{Mode: "function", Name: "f", Params: []course.Param{{Name: "a", Type: "int[]"}}, Returns: "float64[]"})
	if CanonicalizableOutput(fl) {
		t.Error("a float output is not canonicalizable")
	}
}

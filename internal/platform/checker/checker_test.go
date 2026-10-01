package checker

import (
	"errors"
	"testing"

	"github.com/sujaykumarsuman/xlearn/internal/course"
	"github.com/sujaykumarsuman/xlearn/internal/platform/harness"
)

// Every value here is synthetic and hand-made.

func val(t *testing.T, typ, raw string) harness.Value {
	t.Helper()
	ty, err := harness.ParseType(typ)
	if err != nil {
		t.Fatal(err)
	}
	v, err := harness.DecodeValue(ty, []byte(raw))
	if err != nil {
		t.Fatalf("%s %s: %v", typ, raw, err)
	}
	return v
}

func lookup(t *testing.T, name string, eps float64) Checker {
	t.Helper()
	c := &course.Checker{Name: name}
	if eps != 0 {
		c.Params = &course.CheckerParams{Eps: eps}
	}
	ch, err := Lookup(c)
	if err != nil {
		t.Fatal(err)
	}
	return ch
}

func TestCheckers(t *testing.T) {
	cases := []struct {
		checker   string
		eps       float64
		typ       string
		got, want string
		ok        bool
		reason    Reason
	}{
		{Exact, 0, "bool", "true", "true", true, Match},
		{Exact, 0, "bool", "false", "true", false, ValueMismatch},
		{Exact, 0, "int[]", "[1,2]", "[1,2]", true, Match},
		{Exact, 0, "int[]", "[2,1]", "[1,2]", false, ValueMismatch},
		{Exact, 0, "int[]", "[1]", "[1,2]", false, LengthMismatch},
		{Exact, 0, "TreeNode", "[1,null,2]", "[1,null,2]", true, Match},
		{Exact, 0, "string", `"a"`, `"b"`, false, ValueMismatch},
		{Unordered, 0, "int[]", "[3,1,2]", "[1,2,3]", true, Match},
		{Unordered, 0, "int[]", "[1,1,2]", "[1,2,2]", false, ValueMismatch},
		{Unordered, 0, "int[][]", "[[2,1],[3]]", "[[3],[2,1]]", true, Match},
		{Unordered, 0, "int[][]", "[[1,2],[3]]", "[[3],[2,1]]", false, ValueMismatch},
		{UnorderedDeep, 0, "int[][]", "[[1,2],[3]]", "[[3],[2,1]]", true, Match},
		{UnorderedDeep, 0, "int[][]", "[[0,-1,1],[-2,0,2]]", "[[-2,2,0],[1,0,-1]]", true, Match},
		{UnorderedDeep, 0, "int[][]", "[[0,-1,1],[0,-1,1]]", "[[-1,0,1]]", false, LengthMismatch},
		{UnorderedDeep, 0, "string[][]", `[["b","a"]]`, `[["a","b"]]`, true, Match},
		{SetEqual, 0, "int[]", "[1,1,2]", "[2,1]", true, Match},
		{SetEqual, 0, "int[]", "[1,3]", "[2,1]", false, ValueMismatch},
		{SetEqual, 0, "int[]", "[1]", "[2,1]", false, LengthMismatch},
		{FloatAbs, 1e-6, "float64", "0.1000004", "0.1", true, Match},
		{FloatAbs, 1e-6, "float64", "0.100002", "0.1", false, FloatTolerance},
		{FloatAbs, 1e-6, "float64[]", "[1, 2.0000001]", "[1, 2]", true, Match},
		{FloatAbs, 1e-6, "float64[]", "[1]", "[1, 2]", false, LengthMismatch},
		{FloatRel, 1e-6, "float64", "1000000.5", "1000000", true, Match},
		{FloatRel, 1e-6, "float64", "1000002", "1000000", false, FloatTolerance},
		{FloatRel, 1e-6, "float64", "0.0000005", "0", true, Match},
	}
	for i, c := range cases {
		ch := lookup(t, c.checker, c.eps)
		v := ch.Check(val(t, c.typ, c.got), val(t, c.typ, c.want))
		if v.OK != c.ok || v.Reason != c.reason {
			t.Errorf("case %d %s %s vs %s: %+v, want ok=%v %s", i, c.checker, c.got, c.want, v, c.ok, c.reason)
		}
	}
}

func TestAnyOf(t *testing.T) {
	ch := lookup(t, AnyOf, 0)
	want := harness.ListV(val(t, "int[]", "[0,1]"), val(t, "int[]", "[1,0]"))
	if v := ch.Check(val(t, "int[]", "[1,0]"), want); !v.OK {
		t.Errorf("any_of second alternative: %+v", v)
	}
	if v := ch.Check(val(t, "int[]", "[1,1]"), want); v.OK || v.Reason != NoAlternative {
		t.Errorf("any_of miss: %+v", v)
	}
	if v := ch.Check(harness.IntV(1), harness.IntV(1)); v.OK || v.Reason != ShapeMismatch {
		t.Errorf("any_of needs an alternatives list: %+v", v)
	}
}

func TestShapeMismatch(t *testing.T) {
	for _, name := range []string{Exact, Unordered, UnorderedDeep, SetEqual} {
		v := lookup(t, name, 0).Check(harness.IntV(1), harness.ListV(harness.IntV(1)))
		if v.OK || v.Reason != ShapeMismatch {
			t.Errorf("%s: %+v, want shape_mismatch", name, v)
		}
	}
}

func TestLookupAndSupports(t *testing.T) {
	if _, err := Lookup(&course.Checker{Name: "regex"}); !errors.Is(err, ErrUnknown) {
		t.Errorf("unknown checker: %v", err)
	}
	if _, err := Lookup(nil); !errors.Is(err, ErrUnknown) {
		t.Errorf("nil checker: %v", err)
	}
	if _, err := Lookup(&course.Checker{Name: Exact, Params: &course.CheckerParams{Eps: 0.1}}); err == nil {
		t.Error("exact with eps accepted")
	}
	if _, err := Lookup(&course.Checker{Name: FloatAbs, Params: &course.CheckerParams{Eps: -1}}); err == nil {
		t.Error("negative eps accepted")
	}
	if c := lookup(t, FloatRel, 0); c.(floatChecker).eps != DefaultEps {
		t.Error("default eps")
	}
	for _, name := range Names {
		c := lookup(t, name, 0)
		if c.Name() != name {
			t.Errorf("%s: Name() = %s", name, c.Name())
		}
		if c.Canonical() != (name == Exact) {
			t.Errorf("%s: Canonical() = %v", name, c.Canonical())
		}
	}
	intArr, _ := harness.ParseType("int[]")
	flt, _ := harness.ParseType("float64")
	for _, c := range []struct {
		name string
		t    harness.Type
		ok   bool
	}{
		{Unordered, intArr, true}, {Unordered, flt, false}, {UnorderedDeep, intArr, true},
		{SetEqual, flt, false}, {FloatAbs, flt, true}, {FloatRel, intArr, false}, {Exact, flt, true}, {AnyOf, intArr, true},
		{"nope", intArr, false},
	} {
		if err := Supports(c.name, c.t); (err == nil) != c.ok {
			t.Errorf("Supports(%s, %s) = %v", c.name, c.t, err)
		}
	}
}

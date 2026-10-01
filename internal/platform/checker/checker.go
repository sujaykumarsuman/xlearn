// Package checker is the ONE closed checker registry (t1 §7.1, t4 §4.1): `exact`,
// `unordered`, `unordered_deep`, `float_abs`, `float_rel`, `set_equal`, `any_of`. A checker
// compares a decoded result with an expected value, both harness.Values, and returns a typed
// Verdict — never formatted values, so nothing it returns can leak a hidden case.
//
// m3-02 created it for packlint's oracle and wrong-solution gates; m3-06's `code@1` grader
// imports it (no checker is ever defined inline in judge), and m3-04's jail tests use it. A
// library: the stdlib, internal/course and internal/platform/harness only. A new checker is
// code plus a release; per-item checker code never runs.
package checker

import (
	"errors"
	"fmt"
	"math"
	"sort"

	"github.com/sujaykumarsuman/xlearn/internal/course"
	"github.com/sujaykumarsuman/xlearn/internal/platform/harness"
)

// Checker names (the item's `config.checker.name`).
const (
	Exact         = "exact"
	Unordered     = "unordered"
	UnorderedDeep = "unordered_deep"
	FloatAbs      = "float_abs"
	FloatRel      = "float_rel"
	SetEqual      = "set_equal"
	AnyOf         = "any_of"
)

// Names is the closed registry, sorted.
var Names = []string{AnyOf, Exact, FloatAbs, FloatRel, SetEqual, Unordered, UnorderedDeep}

// Reason is a verdict's typed reason (an enum; it never carries a value).
type Reason string

// Reasons.
const (
	Match          Reason = "match"
	ValueMismatch  Reason = "value_mismatch"
	LengthMismatch Reason = "length_mismatch"
	ShapeMismatch  Reason = "shape_mismatch"
	FloatTolerance Reason = "outside_tolerance"
	NoAlternative  Reason = "no_alternative_matched"
)

// Verdict is a checker's typed result.
type Verdict struct {
	OK     bool
	Reason Reason
}

func ok() Verdict                { return Verdict{OK: true, Reason: Match} }
func fail(r Reason) Verdict      { return Verdict{Reason: r} }
func (v Verdict) String() string { return string(v.Reason) }

// Checker compares a result (got) with the expected value (want).
type Checker interface {
	Name() string
	Check(got, want harness.Value) Verdict
	// Canonical reports whether two results agree exactly when their canonical bytes do,
	// so judge may compare a perf case by its fd-4 sha256 (`exact` only).
	Canonical() bool
}

// ErrUnknown is returned for a name outside the registry.
var ErrUnknown = errors.New("checker: not in the closed registry")

// DefaultEps is the tolerance of a float checker declared without `eps`.
const DefaultEps = 1e-6

// Lookup returns the named checker with its parameters (float_abs and float_rel take eps;
// the others take none).
func Lookup(c *course.Checker) (Checker, error) {
	if c == nil {
		return nil, fmt.Errorf("%w: no checker", ErrUnknown)
	}
	eps := DefaultEps
	if c.Params != nil && c.Params.Eps != 0 {
		eps = c.Params.Eps
	}
	switch c.Name {
	case FloatAbs, FloatRel:
		if !(eps > 0) || math.IsInf(eps, 0) {
			return nil, fmt.Errorf("checker %s: eps must be > 0", c.Name)
		}
		return floatChecker{name: c.Name, eps: eps, rel: c.Name == FloatRel}, nil
	}
	if c.Params != nil && c.Params.Eps != 0 {
		return nil, fmt.Errorf("checker %s takes no eps", c.Name)
	}
	switch c.Name {
	case Exact:
		return exact{}, nil
	case Unordered:
		return unordered{}, nil
	case UnorderedDeep:
		return unorderedDeep{}, nil
	case SetEqual:
		return setEqual{}, nil
	case AnyOf:
		return anyOf{}, nil
	}
	return nil, fmt.Errorf("%w: %q (want one of %v)", ErrUnknown, c.Name, Names)
}

// Supports reports whether a checker can compare a result of the given type: the
// collection checkers need an array result, the float checkers a float somewhere.
func Supports(name string, t harness.Type) error {
	switch name {
	case Unordered, UnorderedDeep, SetEqual:
		if t.Dims == 0 {
			return fmt.Errorf("checker %s needs an array result, not %s", name, t)
		}
	case FloatAbs, FloatRel:
		if !t.IsFloat() {
			return fmt.Errorf("checker %s needs a float64 result, not %s", name, t)
		}
	case Exact, AnyOf:
	default:
		return fmt.Errorf("%w: %q", ErrUnknown, name)
	}
	return nil
}

// --- exact ---------------------------------------------------------------------------

type exact struct{}

func (exact) Name() string    { return Exact }
func (exact) Canonical() bool { return true }
func (exact) Check(got, want harness.Value) Verdict {
	return compare(got, want, nil)
}

// compare is deep equality with an optional float comparison (nil = exact ==).
func compare(got, want harness.Value, eqf func(a, b float64) bool) Verdict {
	if got.Kind != want.Kind {
		return fail(ShapeMismatch)
	}
	switch got.Kind {
	case harness.KindList:
		if len(got.List) != len(want.List) {
			return fail(LengthMismatch)
		}
		for i := range got.List {
			if v := compare(got.List[i], want.List[i], eqf); !v.OK {
				return v
			}
		}
		return ok()
	case harness.KindFloat:
		if eqf != nil {
			if eqf(got.Float, want.Float) {
				return ok()
			}
			return fail(FloatTolerance)
		}
	}
	if harness.Equal(got, want) {
		return ok()
	}
	return fail(ValueMismatch)
}

// --- unordered: the top-level array as a multiset -------------------------------------

type unordered struct{}

func (unordered) Name() string    { return Unordered }
func (unordered) Canonical() bool { return false }
func (unordered) Check(got, want harness.Value) Verdict {
	if got.Kind != harness.KindList || want.Kind != harness.KindList {
		return fail(ShapeMismatch)
	}
	if len(got.List) != len(want.List) {
		return fail(LengthMismatch)
	}
	return compare(sorted(got.List), sorted(want.List), nil)
}

// --- unordered_deep: every array at every level as a multiset --------------------------

type unorderedDeep struct{}

func (unorderedDeep) Name() string    { return UnorderedDeep }
func (unorderedDeep) Canonical() bool { return false }
func (unorderedDeep) Check(got, want harness.Value) Verdict {
	if got.Kind != harness.KindList || want.Kind != harness.KindList {
		return fail(ShapeMismatch)
	}
	return compare(deepSorted(got), deepSorted(want), nil)
}

// --- set_equal: the top-level array as a set (duplicates ignored) ----------------------

type setEqual struct{}

func (setEqual) Name() string    { return SetEqual }
func (setEqual) Canonical() bool { return false }
func (setEqual) Check(got, want harness.Value) Verdict {
	if got.Kind != harness.KindList || want.Kind != harness.KindList {
		return fail(ShapeMismatch)
	}
	g, w := dedupe(sorted(got.List)), dedupe(sorted(want.List))
	if len(g.List) != len(w.List) {
		return fail(LengthMismatch)
	}
	return compare(g, w, nil)
}

// --- any_of: want is the array of every accepted result ---------------------------------

type anyOf struct{}

func (anyOf) Name() string    { return AnyOf }
func (anyOf) Canonical() bool { return false }
func (anyOf) Check(got, want harness.Value) Verdict {
	if want.Kind != harness.KindList || len(want.List) == 0 {
		return fail(ShapeMismatch)
	}
	for _, alt := range want.List {
		if compare(got, alt, nil).OK {
			return ok()
		}
	}
	return fail(NoAlternative)
}

// --- float_abs / float_rel ------------------------------------------------------------

type floatChecker struct {
	name string
	eps  float64
	rel  bool
}

func (f floatChecker) Name() string    { return f.name }
func (f floatChecker) Canonical() bool { return false }

// Check compares structurally; floats agree when |got − want| ≤ eps (float_abs), or
// ≤ eps · max(1, |want|) (float_rel: relative, with an absolute floor near zero). Every
// non-float value must match exactly.
func (f floatChecker) Check(got, want harness.Value) Verdict {
	return compare(got, want, func(a, b float64) bool {
		if math.IsNaN(a) || math.IsInf(a, 0) {
			return false
		}
		d := math.Abs(a - b)
		if f.rel {
			return d <= f.eps*math.Max(1, math.Abs(b))
		}
		return d <= f.eps
	})
}

// --- ordering helpers -----------------------------------------------------------------

// order is a total order on Values: kind first, then value; lists lexicographically.
func order(a, b harness.Value) int {
	if a.Kind != b.Kind {
		return int(a.Kind) - int(b.Kind)
	}
	switch a.Kind {
	case harness.KindInt:
		return cmp3(a.Int < b.Int, a.Int > b.Int)
	case harness.KindFloat:
		return cmp3(a.Float < b.Float, a.Float > b.Float)
	case harness.KindBool:
		return cmp3(!a.Bool && b.Bool, a.Bool && !b.Bool)
	case harness.KindString:
		return cmp3(a.Str < b.Str, a.Str > b.Str)
	case harness.KindList:
		for i := 0; i < len(a.List) && i < len(b.List); i++ {
			if c := order(a.List[i], b.List[i]); c != 0 {
				return c
			}
		}
		return len(a.List) - len(b.List)
	}
	return 0
}

func cmp3(less, more bool) int {
	switch {
	case less:
		return -1
	case more:
		return 1
	}
	return 0
}

func sorted(vs []harness.Value) harness.Value {
	out := append([]harness.Value{}, vs...)
	sort.SliceStable(out, func(i, j int) bool { return order(out[i], out[j]) < 0 })
	return harness.Value{Kind: harness.KindList, List: out}
}

func deepSorted(v harness.Value) harness.Value {
	if v.Kind != harness.KindList {
		return v
	}
	out := make([]harness.Value, len(v.List))
	for i, e := range v.List {
		out[i] = deepSorted(e)
	}
	return sorted(out)
}

func dedupe(v harness.Value) harness.Value {
	out := harness.Value{Kind: harness.KindList}
	for i, e := range v.List {
		if i == 0 || order(e, v.List[i-1]) != 0 {
			out.List = append(out.List, e)
		}
	}
	return out
}

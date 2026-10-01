package packspec

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/sujaykumarsuman/xlearn/internal/course"
	"github.com/sujaykumarsuman/xlearn/internal/platform/harness"
)

// The ONE generic validator (t1 §7.2): derived from the public item's constraints[] — the
// grammar m1-01 froze — and applied to every edge, random and materialized perf input.
//
//	{"arg": "<param>", "len": [lo, hi]}          the param's length: an array's elements, a
//	                                             string's characters (runes), a ListNode's or
//	                                             TreeNode's nodes, a GraphNode's nodes
//	{"arg": "<param>[*]…", "range": [lo, hi]}    every number reached through the [*] steps
//	{"arg": "<param>[*]", "len": [lo, hi]}       the length of every element (e.g. string[])
//
// `[*]` steps into an array's elements, a ListNode's or TreeNode's values, or a GraphNode's
// adjacency lists. In class mode a name matches the constructor's and every op's params of
// that name. Anything else — an unknown param, a range on a list, a len on a number, a
// constraint with neither — is an error "extend the validator", never silently ignored.
// Custom structural invariants (BST, connected graph …) are programs under validate/.

// ErrConstraint marks a constraint the validator cannot apply (the item's fault, not the
// input's).
var ErrConstraint = errors.New("constraint not supported by the generic validator (extend the validator)")

var argPathRe = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)((?:\[\*\])*)$`)

// Validator checks inputs against one code part's constraints.
type Validator struct {
	sig  *harness.Sig
	cons []constraint
}

type constraint struct {
	src   course.Constraint
	param string
	steps int
}

// NewValidator compiles the constraints against the signature; a constraint it cannot
// apply is an ErrConstraint.
func NewValidator(sig *harness.Sig, cons []course.Constraint) (*Validator, error) {
	v := &Validator{sig: sig}
	var errs []error
	for i, c := range cons {
		m := argPathRe.FindStringSubmatch(c.Arg)
		if m == nil {
			errs = append(errs, fmt.Errorf("constraints[%d] arg %q: %w", i, c.Arg, ErrConstraint))
			continue
		}
		cc := constraint{src: c, param: m[1], steps: strings.Count(m[2], "[*]")}
		types := v.paramTypes(cc.param)
		if len(types) == 0 {
			errs = append(errs, fmt.Errorf("constraints[%d] arg %q names no parameter: %w", i, c.Arg, ErrConstraint))
			continue
		}
		if c.Len == nil && c.Range == nil {
			errs = append(errs, fmt.Errorf("constraints[%d] has neither len nor range: %w", i, ErrConstraint))
			continue
		}
		for _, t := range types {
			if err := cc.check(t); err != nil {
				errs = append(errs, fmt.Errorf("constraints[%d] arg %q (%s): %v: %w", i, c.Arg, t, err, ErrConstraint))
			}
		}
		v.cons = append(v.cons, cc)
	}
	return v, errors.Join(errs...)
}

// paramTypes lists the types of every param of that name (class mode: constructor + ops).
func (v *Validator) paramTypes(name string) []harness.Type {
	var out []harness.Type
	add := func(ps []harness.Param) {
		for _, p := range ps {
			if p.Name == name {
				out = append(out, p.Type)
			}
		}
	}
	add(v.sig.Params)
	for _, o := range v.sig.Ops {
		add(o.Params)
	}
	return out
}

// step is the type reached by one [*] (ok=false when it cannot step).
func step(t harness.Type) (harness.Type, bool) {
	switch {
	case t.Dims > 0:
		return t.Elem(), true
	case t.Base == harness.BaseListNode || t.Base == harness.BaseTreeNode:
		return harness.Type{Base: harness.BaseInt}, true
	case t.Base == harness.BaseGraphNode:
		return harness.Type{Base: harness.BaseInt, Dims: 1}, true
	}
	return harness.Type{}, false
}

// check is the static check of a constraint against one param type.
func (c constraint) check(t harness.Type) error {
	for i := 0; i < c.steps; i++ {
		var ok bool
		if t, ok = step(t); !ok {
			return fmt.Errorf("[*] does not apply to %s", t)
		}
	}
	if c.src.Len != nil && (len(c.src.Len) != 2 || c.src.Len[0] > c.src.Len[1] || !hasLen(t)) {
		return fmt.Errorf("len [min, max] needs an array, string or node type, not %s", t)
	}
	if c.src.Range != nil {
		if len(c.src.Range) != 2 || c.src.Range[0] > c.src.Range[1] {
			return errors.New("range must be [min, max]")
		}
		if t.Dims > 0 || (t.Base != harness.BaseInt && t.Base != harness.BaseInt64 && t.Base != harness.BaseFloat64) {
			return fmt.Errorf("range needs numbers, not %s (step in with [*])", t)
		}
	}
	return nil
}

func hasLen(t harness.Type) bool {
	return t.Dims > 0 || t.Base == harness.BaseString || t.Base == harness.BaseListNode ||
		t.Base == harness.BaseTreeNode || t.Base == harness.BaseGraphNode
}

// ErrInvalidInput marks an input that breaks a constraint.
var ErrInvalidInput = errors.New("input breaks a public constraint")

// Check applies every constraint to a decoded input. The error names the constraint
// (arg and bound), never the value.
func (v *Validator) Check(in *harness.Input) error {
	for _, c := range v.cons {
		for _, val := range v.values(in, c.param) {
			if err := c.apply(val.v, val.t, c.steps); err != nil {
				return err
			}
		}
	}
	return nil
}

type typedValue struct {
	v harness.Value
	t harness.Type
}

// values collects every argument bound to a param name.
func (v *Validator) values(in *harness.Input, name string) []typedValue {
	var out []typedValue
	if !v.sig.Class() {
		for i, p := range v.sig.Params {
			if p.Name == name {
				out = append(out, typedValue{in.Args[i], p.Type})
			}
		}
		return out
	}
	for i, op := range in.Ops {
		ps := v.sig.Params
		if i > 0 {
			for _, o := range v.sig.Ops {
				if o.Name == op {
					ps = o.Params
				}
			}
		}
		for k, p := range ps {
			if p.Name == name {
				out = append(out, typedValue{in.Args[i].List[k], p.Type})
			}
		}
	}
	return out
}

func (c constraint) apply(val harness.Value, t harness.Type, steps int) error {
	if steps > 0 {
		next, _ := step(t)
		elems := val.List
		if t.Dims == 0 && t.Base == harness.BaseTreeNode {
			elems = nil
			for _, e := range val.List {
				if !e.IsNull() {
					elems = append(elems, e)
				}
			}
		}
		for _, e := range elems {
			if err := c.apply(e, next, steps-1); err != nil {
				return err
			}
		}
		return nil
	}
	if c.src.Len != nil {
		n := length(val, t)
		if n < c.src.Len[0] || n > c.src.Len[1] {
			return fmt.Errorf("%w: %s len [%d, %d]", ErrInvalidInput, c.src.Arg, c.src.Len[0], c.src.Len[1])
		}
	}
	if c.src.Range != nil {
		var x float64
		switch val.Kind {
		case harness.KindInt:
			x = float64(val.Int)
		case harness.KindFloat:
			x = val.Float
		default:
			return fmt.Errorf("%w: %s is not a number", ErrInvalidInput, c.src.Arg)
		}
		if x < c.src.Range[0] || x > c.src.Range[1] {
			return fmt.Errorf("%w: %s range [%g, %g]", ErrInvalidInput, c.src.Arg, c.src.Range[0], c.src.Range[1])
		}
	}
	return nil
}

func length(val harness.Value, t harness.Type) int {
	switch {
	case val.Kind == harness.KindString:
		return utf8.RuneCountInString(val.Str)
	case t.Dims == 0 && t.Base == harness.BaseTreeNode:
		n := 0
		for _, e := range val.List {
			if !e.IsNull() {
				n++
			}
		}
		return n
	}
	return len(val.List)
}

package harness

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Kind is a Value's JSON shape.
type Kind uint8

// Value kinds.
const (
	KindNull Kind = iota
	KindInt
	KindFloat
	KindBool
	KindString
	KindList
)

// Value is a typed case value (an input argument, an expected output, a decoded fd-4
// result): the JSON tree a registry type maps to. Arrays, ListNode (its values in order),
// TreeNode (level order with null holes) and GraphNode (1-indexed adjacency lists) are all
// KindList. Checkers compare Values, never formatted text.
type Value struct {
	Kind  Kind
	Int   int64
	Float float64
	Bool  bool
	Str   string
	List  []Value
}

// Convenience constructors (tests, generators).
func Null() Value              { return Value{Kind: KindNull} }
func IntV(n int64) Value       { return Value{Kind: KindInt, Int: n} }
func FloatV(f float64) Value   { return Value{Kind: KindFloat, Float: f} }
func BoolV(b bool) Value       { return Value{Kind: KindBool, Bool: b} }
func StringV(s string) Value   { return Value{Kind: KindString, Str: s} }
func ListV(vs ...Value) Value  { return Value{Kind: KindList, List: append([]Value{}, vs...)} }
func (v Value) IsNull() bool   { return v.Kind == KindNull }
func (v Value) Len() int       { return len(v.List) }
func (v Value) At(i int) Value { return v.List[i] }
func (v Value) String() string { b, _ := appendValue(nil, v, true); return string(b) }
func (k Kind) String() string {
	return [...]string{"null", "int", "float", "bool", "string", "list"}[k]
}
func (v Value) Equal(w Value) bool { return Equal(v, w) }

// Equal is deep, exact equality (floats compare with ==; an Int never equals a Float).
func Equal(a, b Value) bool {
	if a.Kind != b.Kind {
		return false
	}
	switch a.Kind {
	case KindNull:
		return true
	case KindInt:
		return a.Int == b.Int
	case KindFloat:
		return a.Float == b.Float
	case KindBool:
		return a.Bool == b.Bool
	case KindString:
		return a.Str == b.Str
	default:
		if len(a.List) != len(b.List) {
			return false
		}
		for i := range a.List {
			if !Equal(a.List[i], b.List[i]) {
				return false
			}
		}
		return true
	}
}

// ErrValue is the class of every typed-decode failure (a value that does not fit its type).
var ErrValue = errors.New("harness: value does not fit its type")

// DecodeValue strictly decodes one JSON value against a registry type: integer ranges
// (int is 32-bit), finite floats, valid UTF-8, depth ≤ MaxDepth, ≤ MaxNodes nodes, a
// well-formed level order / adjacency. A void type accepts only null. The result is in
// canonical form (a TreeNode's trailing nulls trimmed).
func DecodeValue(t Type, raw []byte) (Value, error) {
	j, err := parseJSON(raw)
	if err != nil {
		return Value{}, err
	}
	d := &typed{}
	return d.value(t, &j)
}

// typed decodes parsed JSON against types, counting nodes.
type typed struct {
	nodes int
}

func (d *typed) errf(t Type, format string, args ...any) error {
	return fmt.Errorf("%w %s: %s", ErrValue, t, fmt.Sprintf(format, args...))
}

func (d *typed) addNodes(t Type, n int) error {
	d.nodes += n
	if d.nodes > MaxNodes {
		return d.errf(t, "more than %d nodes", MaxNodes)
	}
	return nil
}

func (d *typed) value(t Type, j *jv) (Value, error) {
	if t.Base == BaseVoid {
		if j.kind != jNull {
			return Value{}, d.errf(t, "a void op returns null")
		}
		return Null(), nil
	}
	if t.Dims > 0 {
		if j.kind != jArr {
			return Value{}, d.errf(t, "want an array")
		}
		out := Value{Kind: KindList, List: make([]Value, 0, len(j.arr))}
		for i := range j.arr {
			e, err := d.value(t.Elem(), &j.arr[i])
			if err != nil {
				return Value{}, err
			}
			out.List = append(out.List, e)
		}
		return out, nil
	}
	switch t.Base {
	case BaseInt:
		return d.integer(t, j, math.MinInt32, math.MaxInt32)
	case BaseInt64:
		return d.integer(t, j, math.MinInt64, math.MaxInt64)
	case BaseFloat64:
		if j.kind != jNum {
			return Value{}, d.errf(t, "want a number")
		}
		f, err := strconv.ParseFloat(j.s, 64)
		if err != nil || math.IsInf(f, 0) || math.IsNaN(f) {
			return Value{}, d.errf(t, "not a finite float64")
		}
		return FloatV(f), nil
	case BaseBool:
		if j.kind != jBool {
			return Value{}, d.errf(t, "want true or false")
		}
		return BoolV(j.b), nil
	case BaseString:
		if j.kind != jStr {
			return Value{}, d.errf(t, "want a string")
		}
		return StringV(j.s), nil
	case BaseListNode:
		if j.kind != jArr {
			return Value{}, d.errf(t, "want an array of node values")
		}
		if err := d.addNodes(t, len(j.arr)); err != nil {
			return Value{}, err
		}
		out := Value{Kind: KindList, List: make([]Value, 0, len(j.arr))}
		for i := range j.arr {
			e, err := d.integer(t, &j.arr[i], math.MinInt32, math.MaxInt32)
			if err != nil {
				return Value{}, err
			}
			out.List = append(out.List, e)
		}
		return out, nil
	case BaseTreeNode:
		return d.tree(t, j)
	case BaseGraphNode:
		return d.graph(t, j)
	}
	return Value{}, d.errf(t, "unknown type")
}

func (d *typed) integer(t Type, j *jv, lo, hi int64) (Value, error) {
	if j.kind != jNum {
		return Value{}, d.errf(t, "want an integer")
	}
	if strings.ContainsAny(j.s, ".eE") {
		return Value{}, d.errf(t, "want an integer, not %s", j.s)
	}
	n, err := strconv.ParseInt(j.s, 10, 64)
	if err != nil || n < lo || n > hi {
		return Value{}, d.errf(t, "integer out of range [%d, %d]", lo, hi)
	}
	if j.s == "-0" {
		return Value{}, d.errf(t, "-0 is not an integer spelling")
	}
	return IntV(n), nil
}

// tree checks a LeetCode level order: the root is not null, every non-null node takes the
// next two slots as its children, and nothing is left over. Trailing nulls are trimmed.
func (d *typed) tree(t Type, j *jv) (Value, error) {
	if j.kind != jArr {
		return Value{}, d.errf(t, "want a level-order array")
	}
	a := j.arr
	end := len(a)
	for end > 0 && a[end-1].kind == jNull {
		end--
	}
	a = a[:end]
	out := Value{Kind: KindList, List: make([]Value, 0, len(a))}
	if len(a) == 0 {
		return out, nil
	}
	if a[0].kind == jNull {
		return Value{}, d.errf(t, "the root of a non-empty level order is null")
	}
	nonNull := 0
	for i := range a {
		if a[i].kind == jNull {
			out.List = append(out.List, Null())
			continue
		}
		e, err := d.integer(t, &a[i], math.MinInt32, math.MaxInt32)
		if err != nil {
			return Value{}, err
		}
		out.List = append(out.List, e)
		nonNull++
	}
	if err := d.addNodes(t, nonNull); err != nil {
		return Value{}, err
	}
	// Slot accounting: the root takes slot 0 and every non-null node, in order, adds two
	// child slots; an entry past the available slots has no parent.
	avail := 1
	for i := range a {
		if i >= avail {
			return Value{}, d.errf(t, "level order lists a node with no parent slot")
		}
		if a[i].kind != jNull {
			avail += 2
		}
	}
	return out, nil
}

// graph checks 1-indexed adjacency lists: node i+1's neighbours are in 1..n.
func (d *typed) graph(t Type, j *jv) (Value, error) {
	if j.kind != jArr {
		return Value{}, d.errf(t, "want adjacency lists")
	}
	n := len(j.arr)
	if err := d.addNodes(t, n); err != nil {
		return Value{}, err
	}
	out := Value{Kind: KindList, List: make([]Value, 0, n)}
	for i := range j.arr {
		row := &j.arr[i]
		if row.kind != jArr {
			return Value{}, d.errf(t, "node %d: want a neighbour list", i+1)
		}
		r := Value{Kind: KindList, List: make([]Value, 0, len(row.arr))}
		for k := range row.arr {
			e, err := d.integer(t, &row.arr[k], 1, int64(n))
			if err != nil {
				return Value{}, d.errf(t, "node %d: neighbours are node numbers 1..%d", i+1, n)
			}
			r.List = append(r.List, e)
		}
		out.List = append(out.List, r)
	}
	return out, nil
}

// RawOf returns a Value's canonical JSON as a json.RawMessage (floats included, in the
// non-canonical bytes-mode spelling); for display and tests only.
func RawOf(v Value) json.RawMessage {
	b, _ := appendValue(nil, v, true)
	return b
}

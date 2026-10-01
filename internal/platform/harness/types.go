// Package harness is the public, versioned harness codec shared by packlint (m3-02), the
// runner profiles (m3-04) and judge (m3-06): the closed type registry, the canonical JSON,
// the fd-3/fd-4 frame protocol and the per-language harness generators (README.md is the
// wire-format spec, `func-json@1` and `class-ops@1`).
//
// m3-02 created it with the Go half (this package's types, codec and the Go templates);
// m3-04 added the C++ and Python templates and preludes without changing a byte on fd 3 or
// fd 4. harness@v is part of contract_hash, so the wire format is fixed once: a change that
// alters frame bytes is a new major (`@2`), never an edit of `@1`.
//
// A leaf library: the stdlib, internal/course and internal/platform/runnerapi only (no
// runner internals, no service imports), so judge and packlint can both link it.
package harness

import (
	"errors"
	"fmt"
	"strings"

	"github.com/sujaykumarsuman/xlearn/internal/course"
)

// Harness names (the item's `config.harness` value).
const (
	FuncJSON = "func-json@1"
	ClassOps = "class-ops@1"
)

// Harnesses are the harness codecs this package generates and decodes.
var Harnesses = []string{FuncJSON, ClassOps}

// Base is a base type of the closed registry (t1 §7.1).
type Base uint8

// Base types.
const (
	BaseInt Base = iota + 1
	BaseInt64
	BaseFloat64
	BaseBool
	BaseString
	BaseListNode
	BaseTreeNode
	BaseGraphNode
	// BaseVoid is a class op's "no return" (an op with no `returns`); never a parameter.
	BaseVoid
)

var baseNames = map[Base]string{
	BaseInt: "int", BaseInt64: "int64", BaseFloat64: "float64", BaseBool: "bool", BaseString: "string",
	BaseListNode: "ListNode", BaseTreeNode: "TreeNode", BaseGraphNode: "GraphNode", BaseVoid: "void",
}

// maxDims is the deepest `[]` suffix each base type accepts (the closed set).
var maxDims = map[Base]int{
	BaseInt: 2, BaseInt64: 2, BaseFloat64: 2, BaseBool: 2, BaseString: 2,
	BaseListNode: 1, BaseTreeNode: 1, BaseGraphNode: 0, BaseVoid: 0,
}

// Type is one registry type: a base and its array depth (T, T[], T[][]).
type Type struct {
	Base Base
	Dims int
}

// Void is the class-op "no return" type.
var Void = Type{Base: BaseVoid}

// String returns the type's registry spelling ("int[][]", "TreeNode", "void").
func (t Type) String() string {
	return baseNames[t.Base] + strings.Repeat("[]", t.Dims)
}

// Elem is the element type of an array type.
func (t Type) Elem() Type { return Type{Base: t.Base, Dims: t.Dims - 1} }

// IsFloat reports whether the type holds a float anywhere (floats are never canonical).
func (t Type) IsFloat() bool { return t.Base == BaseFloat64 }

// ErrType is the class of every ParseType failure.
var ErrType = errors.New("harness: unsupported type")

// ParseType parses a registry type over the CLOSED set: int (32-bit range), int64, float64,
// bool, string with up to two `[]`; ListNode and TreeNode (level order) with up to one;
// GraphNode; and "void" (class-op returns only — callers check the context). Anything else
// is an error: a new type is code plus a release (t1 §7.1).
func ParseType(s string) (Type, error) {
	base, dims := s, 0
	for strings.HasSuffix(base, "[]") {
		base, dims = strings.TrimSuffix(base, "[]"), dims+1
	}
	for b, name := range baseNames {
		if name != base {
			continue
		}
		if dims > maxDims[b] {
			return Type{}, fmt.Errorf("%w %q: %s takes at most %d [] suffix(es)", ErrType, s, name, maxDims[b])
		}
		return Type{Base: b, Dims: dims}, nil
	}
	return Type{}, fmt.Errorf("%w %q (closed set: int int64 float64 bool string [, []..], ListNode, TreeNode [, []], GraphNode, void)", ErrType, s)
}

// returnType parses a signature or op return; "" is void.
func returnType(s string) (Type, error) {
	if s == "" {
		return Void, nil
	}
	return ParseType(s)
}

// paramType parses a parameter type (never void).
func paramType(s string) (Type, error) {
	t, err := ParseType(s)
	if err != nil {
		return Type{}, err
	}
	if t.Base == BaseVoid {
		return Type{}, fmt.Errorf("%w: void is not a parameter type", ErrType)
	}
	return t, nil
}

// GoType is the Go spelling of a type (the Go column of the type mapping, m3-04 task 1).
func GoType(t Type) string {
	var base string
	switch t.Base {
	case BaseInt:
		base = "int"
	case BaseInt64:
		base = "int64"
	case BaseFloat64:
		base = "float64"
	case BaseBool:
		base = "bool"
	case BaseString:
		base = "string"
	case BaseListNode:
		base = "*ListNode"
	case BaseTreeNode:
		base = "*TreeNode"
	case BaseGraphNode:
		base = "*Node"
	case BaseVoid:
		return ""
	}
	return strings.Repeat("[]", t.Dims) + base
}

// Sig is a parsed signature: the code part's entry point with every type resolved.
type Sig struct {
	// Harness is func-json@1 (function mode) or class-ops@1 (class mode).
	Harness string
	// Name is the function, or the class.
	Name string
	// Params are the function's parameters, or the class constructor's.
	Params []Param
	// Returns is the function's return type (function mode).
	Returns Type
	// Ops are the class's methods, by name (class mode).
	Ops []Op
}

// Param is a named, typed parameter.
type Param struct {
	Name string
	Type Type
}

// Op is a class method.
type Op struct {
	Name    string
	Params  []Param
	Returns Type
}

// op returns the op by name.
func (s *Sig) op(name string) (*Op, bool) {
	for i := range s.Ops {
		if s.Ops[i].Name == name {
			return &s.Ops[i], true
		}
	}
	return nil, false
}

// Class reports class mode.
func (s *Sig) Class() bool { return s.Harness == ClassOps }

// ParseSig resolves a code part's signature against its harness: function mode needs
// func-json@1, class mode class-ops@1 (the constructor is the class name, its params are
// the signature's params, and ops are the methods). Every type must be in the closed set.
func ParseSig(harness string, sig *course.Signature) (*Sig, error) {
	if sig == nil {
		return nil, errors.New("harness: no signature")
	}
	out := &Sig{Harness: harness, Name: sig.Name}
	switch {
	case harness == FuncJSON && sig.Mode == "function":
	case harness == ClassOps && sig.Mode == "class":
	case harness != FuncJSON && harness != ClassOps:
		return nil, fmt.Errorf("harness: %q is not a harness this package speaks (%v)", harness, Harnesses)
	default:
		return nil, fmt.Errorf("harness: %s does not drive %s mode", harness, sig.Mode)
	}
	var err error
	if out.Params, err = params(sig.Params); err != nil {
		return nil, err
	}
	if out.Class() {
		if sig.Returns != "" {
			return nil, errors.New("harness: a class signature has no returns (each op has its own)")
		}
		if len(sig.Ops) == 0 {
			return nil, errors.New("harness: class mode needs at least one op")
		}
		seen := map[string]bool{sig.Name: true}
		for _, o := range sig.Ops {
			if seen[o.Name] {
				return nil, fmt.Errorf("harness: op %q is duplicated or shadows the class name", o.Name)
			}
			seen[o.Name] = true
			op := Op{Name: o.Name}
			if op.Params, err = params(o.Params); err != nil {
				return nil, fmt.Errorf("op %s: %w", o.Name, err)
			}
			if op.Returns, err = returnType(o.Returns); err != nil {
				return nil, fmt.Errorf("op %s returns: %w", o.Name, err)
			}
			out.Ops = append(out.Ops, op)
		}
		return out, nil
	}
	if len(sig.Ops) > 0 {
		return nil, errors.New("harness: ops belong to class mode")
	}
	if out.Returns, err = returnType(sig.Returns); err != nil {
		return nil, fmt.Errorf("returns: %w", err)
	}
	if out.Returns.Base == BaseVoid {
		return nil, errors.New("harness: a func-json@1 function must return a value (in-place signatures are a later @2)")
	}
	return out, nil
}

func params(ps []course.Param) ([]Param, error) {
	out := make([]Param, 0, len(ps))
	for _, p := range ps {
		t, err := paramType(p.Type)
		if err != nil {
			return nil, fmt.Errorf("param %s: %w", p.Name, err)
		}
		out = append(out, Param{Name: p.Name, Type: t})
	}
	return out, nil
}

// CanonicalizableOutput reports whether every output of the signature has canonical bytes
// (no float64 anywhere): only then may judge compare fd 4 by sha256 (OutputMode sha256);
// a float output is compared with a float checker in bytes mode.
func CanonicalizableOutput(s *Sig) bool {
	if !s.Class() {
		return !s.Returns.IsFloat()
	}
	for _, o := range s.Ops {
		if o.Returns.IsFloat() {
			return false
		}
	}
	return true
}

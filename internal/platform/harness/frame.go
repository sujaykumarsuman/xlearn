package harness

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"slices"
)

// The frame protocol (both harnesses, every language; t3 §6.2, m3-04 task 2): a u32
// big-endian length, then that many bytes of canonical JSON. One process per case reads
// exactly one frame on fd 3 and writes exactly one frame on fd 4:
//
//	fd 3  {"args":[…]}                         func-json@1
//	      {"args":[[ctor…],[…],…],"ops":["Class","op",…]}   class-ops@1 (constructor first)
//	fd 4  {"ok":<value>}                       the result (class-ops@1: the per-op results, null for void)
//	      {"panic":"<class>"}                  a caught runtime error (PanicClasses)
//	      {"error":"<code>"}                   the result cannot be encoded (HarnessErrors)
//
// stdout and stderr stay the program's.

// HeaderBytes is the frame's length prefix.
const HeaderBytes = 4

// MaxFrame bounds a frame's payload (a streamed input is ≤ 8 MiB, t4 §9.4; the runner's
// output cap is far below this).
const MaxFrame = 64 << 20

// ErrFrame is the class of every framing error.
var ErrFrame = errors.New("harness: bad frame")

// AppendFrame appends the frame for payload.
func AppendFrame(dst, payload []byte) []byte {
	dst = binary.BigEndian.AppendUint32(dst, uint32(len(payload)))
	return append(dst, payload...)
}

// SplitFrame returns the payload of exactly one frame (no trailing bytes).
func SplitFrame(b []byte) ([]byte, error) {
	if len(b) < HeaderBytes {
		return nil, fmt.Errorf("%w: %d bytes, no length prefix", ErrFrame, len(b))
	}
	n := binary.BigEndian.Uint32(b)
	if n > MaxFrame {
		return nil, fmt.Errorf("%w: length %d over %d", ErrFrame, n, MaxFrame)
	}
	if uint64(len(b)-HeaderBytes) != uint64(n) {
		return nil, fmt.Errorf("%w: length %d, %d payload bytes", ErrFrame, n, len(b)-HeaderBytes)
	}
	return b[HeaderBytes:], nil
}

// ReadFrame reads one frame from r.
func ReadFrame(r io.Reader) ([]byte, error) {
	var h [HeaderBytes]byte
	if _, err := io.ReadFull(r, h[:]); err != nil {
		return nil, fmt.Errorf("%w: header: %v", ErrFrame, err)
	}
	n := binary.BigEndian.Uint32(h[:])
	if n > MaxFrame {
		return nil, fmt.Errorf("%w: length %d over %d", ErrFrame, n, MaxFrame)
	}
	b := make([]byte, n)
	if _, err := io.ReadFull(r, b); err != nil {
		return nil, fmt.Errorf("%w: payload: %v", ErrFrame, err)
	}
	return b, nil
}

// PanicClasses are the closed per-language panic classes a `{"panic"}` frame may carry.
// Each language's generated harness (Go, C++, Python) emits only its own set.
var PanicClasses = map[string][]string{
	"go":     {"index_out_of_range", "nil_dereference", "divide_by_zero", "slice_bounds", "nil_map_write", "other"},
	"cpp":    {"out_of_range", "bad_alloc", "length_error", "logic_error", "runtime_error", "other"},
	"python": {"IndexError", "KeyError", "ValueError", "TypeError", "ZeroDivisionError", "RecursionError", "AttributeError", "AssertionError", "other"},
}

// HarnessErrors are the closed `{"error"}` codes. bad_input is a judge-side fault (the
// input frame did not decode); every other code means the learner's result could not be
// encoded, which judge treats as WA.
var HarnessErrors = []string{"bad_input", "cycle", "too_many_nodes", "invalid_utf8", "non_finite", "invalid_graph"}

// PanicError is a `{"panic":"<class>"}` result.
type PanicError struct{ Class string }

func (e *PanicError) Error() string { return "harness: panic " + e.Class }

// HarnessError is an `{"error":"<code>"}` result.
type HarnessError struct{ Code string }

func (e *HarnessError) Error() string { return "harness: error " + e.Code }

// CanonicalInput validates a case input against the signature and returns its canonical
// JSON (the fd-3 payload). raw is the input object: {"args":[…]} in function mode,
// {"ops":[…],"args":[…]} in class mode (ops[0] is the class name, args[0] the
// constructor's arguments; key order is free, the output's keys are sorted).
func CanonicalInput(s *Sig, raw []byte) ([]byte, error) {
	j, err := parseJSON(raw)
	if err != nil {
		return nil, err
	}
	in, err := decodeInput(s, &j)
	if err != nil {
		return nil, err
	}
	return in.canonical()
}

// EncodeInput is CanonicalInput as the fd-3 frame.
func EncodeInput(s *Sig, raw []byte) ([]byte, error) {
	p, err := CanonicalInput(s, raw)
	if err != nil {
		return nil, err
	}
	return AppendFrame(nil, p), nil
}

// Input is a decoded case input.
type Input struct {
	// Args are the function's arguments (function mode), or one argument list per op
	// (class mode: Args[i] is a KindList of op i's arguments).
	Args []Value
	// Ops are the class-mode op names, the class name first.
	Ops []string
}

// DecodeInput validates a case input (see CanonicalInput) and returns it typed.
func DecodeInput(s *Sig, raw []byte) (*Input, error) {
	j, err := parseJSON(raw)
	if err != nil {
		return nil, err
	}
	return decodeInput(s, &j)
}

func (in *Input) canonical() ([]byte, error) {
	b := []byte(`{"args":[`)
	for i, a := range in.Args {
		if i > 0 {
			b = append(b, ',')
		}
		b = AppendJSON(b, a)
	}
	b = append(b, ']')
	if in.Ops != nil {
		b = append(b, `,"ops":[`...)
		for i, o := range in.Ops {
			if i > 0 {
				b = append(b, ',')
			}
			b = AppendString(b, o)
		}
		b = append(b, ']')
	}
	return append(b, '}'), nil
}

// ErrInput is the class of every input that does not fit the signature.
var ErrInput = errors.New("harness: input does not fit the signature")

func decodeInput(s *Sig, j *jv) (*Input, error) {
	if j.kind != jObj {
		return nil, fmt.Errorf("%w: want an object", ErrInput)
	}
	want := []string{"args"}
	if s.Class() {
		want = []string{"args", "ops"}
	}
	for _, k := range j.keys {
		if !slices.Contains(want, k) {
			return nil, fmt.Errorf("%w: unexpected key %q (want %v)", ErrInput, k, want)
		}
	}
	args, ok := j.field("args")
	if !ok || args.kind != jArr {
		return nil, fmt.Errorf("%w: args must be an array", ErrInput)
	}
	d := &typed{}
	in := &Input{}
	if !s.Class() {
		if len(args.arr) != len(s.Params) {
			return nil, fmt.Errorf("%w: %d args for %d params", ErrInput, len(args.arr), len(s.Params))
		}
		for i, p := range s.Params {
			v, err := d.value(p.Type, &args.arr[i])
			if err != nil {
				return nil, fmt.Errorf("%w: arg %s: %v", ErrInput, p.Name, err)
			}
			in.Args = append(in.Args, v)
		}
		return in, nil
	}
	ops, ok := j.field("ops")
	if !ok || ops.kind != jArr {
		return nil, fmt.Errorf("%w: ops must be an array", ErrInput)
	}
	if len(ops.arr) == 0 || len(ops.arr) != len(args.arr) {
		return nil, fmt.Errorf("%w: %d ops and %d arg lists (want equal, at least the constructor)", ErrInput, len(ops.arr), len(args.arr))
	}
	for i := range ops.arr {
		o := &ops.arr[i]
		if o.kind != jStr {
			return nil, fmt.Errorf("%w: ops[%d] is not a string", ErrInput, i)
		}
		var ps []Param
		switch op, isOp := s.op(o.s); {
		case i == 0 && o.s == s.Name:
			ps = s.Params
		case i == 0:
			return nil, fmt.Errorf("%w: ops[0] must be the constructor %q", ErrInput, s.Name)
		case isOp:
			ps = op.Params
		default:
			return nil, fmt.Errorf("%w: ops[%d] %q is not an op of %s", ErrInput, i, o.s, s.Name)
		}
		a := &args.arr[i]
		if a.kind != jArr || len(a.arr) != len(ps) {
			return nil, fmt.Errorf("%w: args[%d] must list %d argument(s) for %s", ErrInput, i, len(ps), o.s)
		}
		list := Value{Kind: KindList, List: make([]Value, 0, len(ps))}
		for k, p := range ps {
			v, err := d.value(p.Type, &a.arr[k])
			if err != nil {
				return nil, fmt.Errorf("%w: args[%d] %s.%s: %v", ErrInput, i, o.s, p.Name, err)
			}
			list.List = append(list.List, v)
		}
		in.Ops = append(in.Ops, o.s)
		in.Args = append(in.Args, list)
	}
	return in, nil
}

// OutputType is a function signature's result type. Class mode has no single output type:
// use DecodeOutputFor or ExpectedFor with the input.
func (s *Sig) OutputType() (Type, bool) {
	if s.Class() {
		return Type{}, false
	}
	return s.Returns, true
}

// DecodeOutput decodes an fd-4 frame. `{"ok"}` gives the typed Value; `{"panic"}` a
// *PanicError; `{"error"}` a *HarnessError; anything else an ErrFrame error. In class mode
// each element is checked against the union of the op return types; DecodeOutputFor checks
// it op by op.
func DecodeOutput(s *Sig, frame []byte) (Value, error) {
	return decodeOutput(s, nil, frame)
}

// DecodeOutputFor is DecodeOutput with the case input, so a class-mode result is typed op
// by op and must have one entry per op (null for the constructor and void ops).
func DecodeOutputFor(s *Sig, in *Input, frame []byte) (Value, error) {
	return decodeOutput(s, in, frame)
}

func decodeOutput(s *Sig, in *Input, frame []byte) (Value, error) {
	payload, err := SplitFrame(frame)
	if err != nil {
		return Value{}, err
	}
	j, err := parseJSON(payload)
	if err != nil {
		return Value{}, fmt.Errorf("%w: %v", ErrFrame, err)
	}
	if j.kind != jObj || len(j.keys) != 1 {
		return Value{}, fmt.Errorf("%w: want exactly one of ok, panic, error", ErrFrame)
	}
	v := &j.vals[0]
	switch j.keys[0] {
	case "ok":
		return decodeResult(s, in, v)
	case "panic":
		if v.kind != jStr || !knownPanic(v.s) {
			return Value{}, fmt.Errorf("%w: unknown panic class", ErrFrame)
		}
		return Value{}, &PanicError{Class: v.s}
	case "error":
		if v.kind != jStr || !slices.Contains(HarnessErrors, v.s) {
			return Value{}, fmt.Errorf("%w: unknown harness error", ErrFrame)
		}
		return Value{}, &HarnessError{Code: v.s}
	}
	return Value{}, fmt.Errorf("%w: unexpected key %q", ErrFrame, j.keys[0])
}

func knownPanic(c string) bool {
	for _, set := range PanicClasses {
		if slices.Contains(set, c) {
			return true
		}
	}
	return false
}

// decodeResult types an `ok` value (also a case line's `expected`).
func decodeResult(s *Sig, in *Input, v *jv) (Value, error) {
	d := &typed{}
	if !s.Class() {
		return d.value(s.Returns, v)
	}
	if v.kind != jArr {
		return Value{}, fmt.Errorf("%w: a class-ops@1 result is an array", ErrValue)
	}
	if in != nil && len(v.arr) != len(in.Ops) {
		return Value{}, fmt.Errorf("%w: %d results for %d ops", ErrValue, len(v.arr), len(in.Ops))
	}
	out := Value{Kind: KindList, List: make([]Value, 0, len(v.arr))}
	for i := range v.arr {
		e := &v.arr[i]
		var types []Type
		switch {
		case i == 0:
			types = []Type{Void}
		case in != nil:
			op, _ := s.op(in.Ops[i])
			types = []Type{op.Returns}
		default:
			types = append(types, Void)
			for _, o := range s.Ops {
				types = append(types, o.Returns)
			}
		}
		var val Value
		var err error
		for _, t := range types {
			if val, err = d.value(t, e); err == nil {
				break
			}
		}
		if err != nil {
			return Value{}, fmt.Errorf("result %d: %w", i, err)
		}
		out.List = append(out.List, val)
	}
	return out, nil
}

// DecodeExpected types a case line's `expected` (raw JSON) for a signature and input.
func DecodeExpected(s *Sig, in *Input, raw []byte) (Value, error) {
	j, err := parseJSON(raw)
	if err != nil {
		return Value{}, err
	}
	return decodeResult(s, in, &j)
}

// OKFrame is the fd-4 frame `{"ok":<v>}` (canonical for non-float values): the bytes a
// correct harness writes, whose sha256 a perf case's expected digest holds.
func OKFrame(v Value) []byte {
	b := append([]byte(`{"ok":`), AppendJSON(nil, v)...)
	return AppendFrame(nil, append(b, '}'))
}

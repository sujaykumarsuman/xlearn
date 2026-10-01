package gen

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/bits"
	"math/rand/v2"
	"slices"
	"strconv"
	"strings"

	"github.com/sujaykumarsuman/xlearn/internal/platform/harness"
)

// MaxInput is the per-case streamed input cap (t4 §9.4): WriteInput fails past it.
const MaxInput = 8 << 20

var (
	// ErrSpec is the class of every Check failure (an invalid spec for the signature).
	ErrSpec = errors.New("gen: invalid perf spec")
	// ErrTooLarge is the class of a WriteInput failure for an input over MaxInput.
	ErrTooLarge = errors.New("gen: input over the 8 MiB per-case cap")
)

// Spec is a seeded perf-case spec, an item pack.json gen[] entry's "spec":
// {"gen": "int_array@1", "params": {...}}.
type Spec struct {
	Gen    string          `json:"gen"`
	Params json.RawMessage `json:"params"`
}

// Output is a function-mode generator's named output. Type is a registry type spelling
// ("int[]", "int", "int[][]", "string", "TreeNode").
type Output struct {
	Name string
	Type string
}

// Generator is one registry entry.
type Generator struct {
	name    string
	stream  uint64 // the PCG stream constant (part of name@v: changing it changes the bytes)
	outputs []Output
	types   []harness.Type // outputs' parsed types
	// fn compiles a function-mode spec's own params; types[i] is the signature param type
	// output i fills (the output's own type when it is unbound).
	fn func(f *fields, types []harness.Type) (funcPlan, error)
	// cls compiles a class-mode spec against the signature.
	cls func(f *fields, sig *harness.Sig) (classPlan, error)
}

// funcPlan is a compiled function-mode spec; generate draws one case from src.
type funcPlan interface {
	generate(src *rand.PCG) funcCase
}

// funcCase is one generated function-mode case; write streams output k's value.
type funcCase interface {
	write(o *out, k int)
}

// classPlan is a compiled class-mode spec; write streams the whole input object.
type classPlan interface {
	write(o *out, src *rand.PCG)
}

// Name is the generator's name@v ("int_array@1").
func (g *Generator) Name() string { return g.name }

// Outputs are the generator's named outputs (function mode; nil for op_sequence@1).
func (g *Generator) Outputs() []Output { return slices.Clone(g.outputs) }

func newFuncGen(name string, stream uint64, outputs []Output, fn func(*fields, []harness.Type) (funcPlan, error)) *Generator {
	g := &Generator{name: name, stream: stream, outputs: outputs, fn: fn}
	for _, o := range outputs {
		t, err := harness.ParseType(o.Type)
		if err != nil {
			panic(fmt.Sprintf("gen: %s output %s: %v", name, o.Name, err))
		}
		g.types = append(g.types, t)
	}
	return g
}

func newClassGen(name string, stream uint64, cls func(*fields, *harness.Sig) (classPlan, error)) *Generator {
	return &Generator{name: name, stream: stream, cls: cls}
}

// registry is THE closed list, sorted by name. A new generator is a new name@1 here.
var registry = []*Generator{
	graphGen,
	intArrayGen,
	opSequenceGen,
	permutationGen,
	stringGen,
	treeGen,
}

// Registry returns the closed generator list, sorted by name.
func Registry() []*Generator { return slices.Clone(registry) }

// Lookup returns the generator called name ("int_array@1").
func Lookup(name string) (*Generator, error) {
	for _, g := range registry {
		if g.name == name {
			return g, nil
		}
	}
	return nil, fmt.Errorf("%w: unknown generator %q (registry: %s)", ErrSpec, name, Version())
}

// Version is the registry version recorded in tests.lock's header: the sorted generator
// names joined by ",".
func Version() string {
	names := make([]string, len(registry))
	for i, g := range registry {
		names[i] = g.name
	}
	slices.Sort(names)
	return strings.Join(names, ",")
}

// Check validates a spec against a parsed signature without generating: the generator
// exists; its params decode strictly (unknown keys are errors) and are sane (n ≥ 0,
// min ≤ max, the caps); every signature param is filled exactly once (function mode);
// types are compatible; op_sequence@1 drives class mode only and the others function mode.
func Check(sig *harness.Sig, s Spec) error {
	_, err := compile(sig, s)
	return err
}

// WriteInput streams the canonical input object for (spec, seed) to w: {"args":[…]} in
// function mode, {"args":[…],"ops":[…]} in class mode, the exact bytes
// harness.CanonicalInput returns for them. It is deterministic from (spec, seed) only and
// calls Check first. It fails with ErrTooLarge once the input passes MaxInput; on any
// error w may already hold a prefix of the input, which the caller discards.
func WriteInput(w io.Writer, sig *harness.Sig, s Spec, seed uint64) error {
	c, err := compile(sig, s)
	if err != nil {
		return err
	}
	src := rand.NewPCG(seed, c.g.stream)
	o := newOut(w)
	if c.cplan != nil {
		c.cplan.write(o, src)
	} else {
		cs := c.fplan.generate(src)
		o.lit(`{"args":[`)
		for i, sl := range c.slots {
			if i > 0 {
				o.ch(',')
			}
			if sl.out < 0 {
				o.raw(sl.lit)
			} else {
				cs.write(o, sl.out)
			}
		}
		o.lit(`]}`)
	}
	if err := o.flush(); err != nil {
		if errors.Is(err, ErrTooLarge) {
			return fmt.Errorf("%w: %s stopped at %d bytes", err, c.g.name, MaxInput)
		}
		return fmt.Errorf("gen: %s: write: %w", c.g.name, err)
	}
	return nil
}

// compiled is a checked spec, ready to generate.
type compiled struct {
	g     *Generator
	fplan funcPlan
	slots []slot // function mode: what fills each signature param, in order
	cplan classPlan
}

// slot fills one signature param: generator output `out`, or the literal `lit` (out < 0).
type slot struct {
	out int
	lit []byte
}

func compile(sig *harness.Sig, s Spec) (*compiled, error) {
	if sig == nil {
		return nil, fmt.Errorf("%w: no signature", ErrSpec)
	}
	if sig.Harness != harness.FuncJSON && sig.Harness != harness.ClassOps {
		return nil, fmt.Errorf("%w: harness %q (want %v)", ErrSpec, sig.Harness, harness.Harnesses)
	}
	g, err := Lookup(s.Gen)
	if err != nil {
		return nil, err
	}
	switch class := g.cls != nil; {
	case class && !sig.Class():
		return nil, fmt.Errorf("%w: %s generates a class-ops@1 op sequence; %s is a function (%s)", ErrSpec, g.name, sig.Name, sig.Harness)
	case !class && sig.Class():
		return nil, fmt.Errorf("%w: %s generates function arguments; %s is a class (%s): use op_sequence@1", ErrSpec, g.name, sig.Name, sig.Harness)
	}
	raw := bytes.TrimSpace(s.Params)
	if len(raw) == 0 {
		raw = []byte("{}")
	}
	canon, err := harness.CanonicalJSON(raw)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: params: %v", ErrSpec, g.name, err)
	}
	f, err := newFields(g.name, canon)
	if err != nil {
		return nil, err
	}
	c := &compiled{g: g}
	if g.cls != nil {
		if c.cplan, err = g.cls(f, sig); err != nil {
			return nil, err
		}
		return c, nil
	}
	var types []harness.Type
	if c.slots, types, err = g.bind(f, sig); err != nil {
		return nil, err
	}
	if c.fplan, err = g.fn(f, types); err != nil {
		return nil, err
	}
	return c, nil
}

// bind resolves "bind" and "with" (function mode): what fills each signature param, and
// the param type each output fills.
func (g *Generator) bind(f *fields, sig *harness.Sig) ([]slot, []harness.Type, error) {
	bindRaw, hasBind := f.get("bind")
	withRaw, hasWith := f.get("with")
	if f.err != nil {
		return nil, nil, f.err
	}
	params := sig.Params
	index := make(map[string]int, len(params))
	names := make([]string, len(params))
	for i, p := range params {
		index[p.Name] = i
		names[i] = p.Name
	}
	slots := make([]slot, len(params))
	by := make([]string, len(params)) // what fills each param so far
	fill := func(i int, what string) error {
		if by[i] != "" {
			return f.errf("param %s is bound twice (%s and %s)", params[i].Name, by[i], what)
		}
		by[i] = what
		return nil
	}
	if hasWith {
		var with map[string]json.RawMessage
		if withRaw[0] != '{' || json.Unmarshal(withRaw, &with) != nil {
			return nil, nil, f.errf("with must be an object of signature param → literal value")
		}
		for _, name := range sortedKeys(with) {
			i, ok := index[name]
			if !ok {
				return nil, nil, f.errf("with names %q, not a param of %s (%s)", name, sig.Name, strings.Join(names, ", "))
			}
			v, err := harness.DecodeValue(params[i].Type, with[name])
			if err != nil {
				return nil, nil, f.errf("with %s: %v", name, err)
			}
			if err := fill(i, "with"); err != nil {
				return nil, nil, err
			}
			slots[i] = slot{out: -1, lit: harness.AppendJSON(nil, v)}
		}
	}
	bound := make([]int, len(g.outputs)) // output → param index, -1 unbound
	for k := range bound {
		bound[k] = -1
	}
	switch {
	case hasBind:
		var b map[string]string
		if bindRaw[0] != '{' || json.Unmarshal(bindRaw, &b) != nil {
			return nil, nil, f.errf("bind must be an object of output name → signature param name")
		}
		for _, out := range sortedKeys(b) {
			k := g.output(out)
			if k < 0 {
				return nil, nil, f.errf("bind names output %q; %s outputs %s", out, g.name, g.outputNames())
			}
			i, ok := index[b[out]]
			if !ok {
				return nil, nil, f.errf("bind %s → %q: not a param of %s (%s)", out, b[out], sig.Name, strings.Join(names, ", "))
			}
			bound[k] = i
		}
	case len(g.outputs) == 1 && len(params) == 1:
		bound[0] = 0
	default:
		for k, o := range g.outputs {
			if i, ok := index[o.Name]; ok {
				bound[k] = i
			}
		}
	}
	types := slices.Clone(g.types)
	anyBound := false
	for k, i := range bound {
		if i < 0 {
			continue
		}
		anyBound = true
		o, p := g.outputs[k], params[i]
		if err := fill(i, "output "+o.Name); err != nil {
			return nil, nil, err
		}
		if !compatible(g.types[k], p.Type) {
			return nil, nil, f.errf("output %s (%s) cannot fill param %s (%s)", o.Name, o.Type, p.Name, p.Type)
		}
		slots[i] = slot{out: k}
		types[k] = p.Type
	}
	if !anyBound {
		return nil, nil, f.errf("no output fills a param of %s (outputs %s; params %s): use bind", sig.Name, g.outputNames(), strings.Join(names, ", "))
	}
	for i, p := range params {
		if by[i] == "" {
			return nil, nil, f.errf("param %s is not bound: bind an output to it or give it in with", p.Name)
		}
	}
	return slots, types, nil
}

func (g *Generator) output(name string) int {
	for k, o := range g.outputs {
		if o.Name == name {
			return k
		}
	}
	return -1
}

func (g *Generator) outputNames() string {
	var b strings.Builder
	for k, o := range g.outputs {
		if k > 0 {
			b.WriteString(", ")
		}
		b.WriteString(o.Name + " " + o.Type)
	}
	return b.String()
}

// compatible: identical, or an int output filling the int64 param of the same depth.
func compatible(out, param harness.Type) bool {
	return out == param || (out.Base == harness.BaseInt && param.Base == harness.BaseInt64 && out.Dims == param.Dims)
}

// intBounds is the integer range of a type's base: int64 is 64-bit, everything else that
// holds integers (int, ListNode, TreeNode values) is 32-bit.
func intBounds(t harness.Type) (int64, int64) {
	if t.Base == harness.BaseInt64 {
		return math.MinInt64, math.MaxInt64
	}
	return math.MinInt32, math.MaxInt32
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

// fields reads one params object strictly: exact key names, no nulls, typed values, and
// done() rejects every key no accessor asked for.
type fields struct {
	where string
	m     map[string]json.RawMessage
	known []string
	err   error
}

// newFields reads a canonical JSON object (duplicate keys were already rejected by
// harness.CanonicalJSON).
func newFields(where string, raw json.RawMessage) (*fields, error) {
	f := &fields{where: where}
	if len(raw) == 0 || raw[0] != '{' {
		return nil, f.errf("want a JSON object, got %s", clip(raw))
	}
	if err := json.Unmarshal(raw, &f.m); err != nil {
		return nil, f.errf("%v", err)
	}
	return f, nil
}

func (f *fields) errf(format string, args ...any) error {
	return fmt.Errorf("%w: %s: %s", ErrSpec, f.where, fmt.Sprintf(format, args...))
}

func (f *fields) fail(format string, args ...any) {
	if f.err == nil {
		f.err = f.errf(format, args...)
	}
}

func (f *fields) has(key string) bool {
	_, ok := f.m[key]
	return ok
}

// get returns key's raw value; a null is an error.
func (f *fields) get(key string) (json.RawMessage, bool) {
	f.known = append(f.known, key)
	r, ok := f.m[key]
	if ok && string(r) == "null" {
		f.fail("%s must not be null", key)
		return nil, false
	}
	return r, ok
}

func (f *fields) integer(key string, required bool, def int64) int64 {
	r, ok := f.get(key)
	if !ok {
		if required {
			f.fail("%s is required", key)
		}
		return def
	}
	var v int64
	if err := json.Unmarshal(r, &v); err != nil {
		f.fail("%s must be a 64-bit integer, got %s", key, clip(r))
		return def
	}
	return v
}

func (f *fields) boolean(key string) bool {
	r, ok := f.get(key)
	if !ok {
		return false
	}
	var v bool
	if err := json.Unmarshal(r, &v); err != nil {
		f.fail("%s must be true or false, got %s", key, clip(r))
	}
	return v
}

func (f *fields) str(key string, required bool, def string) string {
	r, ok := f.get(key)
	if !ok {
		if required {
			f.fail("%s is required", key)
		}
		return def
	}
	var v string
	if err := json.Unmarshal(r, &v); err != nil {
		f.fail("%s must be a string, got %s", key, clip(r))
		return def
	}
	return v
}

// list reads an array value's elements.
func (f *fields) list(key string) ([]json.RawMessage, bool) {
	r, ok := f.get(key)
	if !ok {
		return nil, false
	}
	var v []json.RawMessage
	if r[0] != '[' || json.Unmarshal(r, &v) != nil {
		f.fail("%s must be an array, got %s", key, clip(r))
		return nil, false
	}
	return v, true
}

// done returns the first accessor error, else rejects keys no accessor asked for.
func (f *fields) done() error {
	if f.err != nil {
		return f.err
	}
	var unknown []string
	for _, k := range sortedKeys(f.m) {
		if !slices.Contains(f.known, k) {
			unknown = append(unknown, strconv.Quote(k))
		}
	}
	if len(unknown) > 0 {
		return f.errf("unknown param %s (known: %s)", strings.Join(unknown, ", "), strings.Join(f.known, ", "))
	}
	return nil
}

// within checks a range of integers for the type it fills.
func (f *fields) within(what string, lo, hi int64, t harness.Type) error {
	t = harness.Type{Base: t.Base} // the element type
	tlo, thi := intBounds(t)
	if lo < tlo || hi > thi {
		return f.errf("%s [%d, %d] leaves the %s range [%d, %d]", what, lo, hi, t, tlo, thi)
	}
	return nil
}

func clip(b []byte) string {
	if len(b) > 40 {
		return string(b[:40]) + "…"
	}
	return string(b)
}

// out is WriteInput's buffered, byte-counting stream: the first error (a write error, or
// ErrTooLarge once MaxInput is passed) sticks and later writes are dropped.
type out struct {
	bw  *bufio.Writer
	n   int
	err error
	tmp []byte
}

func newOut(w io.Writer) *out {
	return &out{bw: bufio.NewWriterSize(w, 64<<10), tmp: make([]byte, 0, 32)}
}

func (o *out) ok() bool { return o.err == nil }

func (o *out) grow(k int) bool {
	if o.err != nil {
		return false
	}
	if o.n += k; o.n > MaxInput {
		o.err = ErrTooLarge
		return false
	}
	return true
}

func (o *out) raw(b []byte) {
	if o.grow(len(b)) {
		_, o.err = o.bw.Write(b)
	}
}

func (o *out) lit(s string) {
	if o.grow(len(s)) {
		_, o.err = o.bw.WriteString(s)
	}
}

func (o *out) ch(c byte) {
	if o.grow(1) {
		o.err = o.bw.WriteByte(c)
	}
}

func (o *out) num(v int64) {
	o.tmp = strconv.AppendInt(o.tmp[:0], v, 10)
	o.raw(o.tmp)
}

func (o *out) flush() error {
	if o.err != nil {
		return o.err
	}
	return o.bw.Flush()
}

// below returns a uniform integer in [0, n), n ≥ 1: Lemire's multiply-shift with
// rejection over src.Uint64 alone, so the draws never depend on math/rand/v2's
// higher-level methods.
func below(src *rand.PCG, n uint64) uint64 {
	hi, lo := bits.Mul64(src.Uint64(), n)
	if lo < n {
		thresh := -n % n
		for lo < thresh {
			hi, lo = bits.Mul64(src.Uint64(), n)
		}
	}
	return hi
}

// uniform returns a uniform integer in [lo, hi] (lo ≤ hi), the full int64 range included.
func uniform(src *rand.PCG, lo, hi int64) int64 {
	span := uint64(hi) - uint64(lo)
	if span == math.MaxUint64 {
		return int64(src.Uint64())
	}
	return int64(uint64(lo) + below(src, span+1))
}

func coin(src *rand.PCG) bool { return src.Uint64()>>63 == 1 }

// shuffle is Fisher–Yates over below.
func shuffle[T any](src *rand.PCG, a []T) {
	for i := len(a) - 1; i > 0; i-- {
		j := below(src, uint64(i)+1)
		a[i], a[j] = a[j], a[i]
	}
}

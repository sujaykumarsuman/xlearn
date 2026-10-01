package gen

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"

	"github.com/sujaykumarsuman/xlearn/internal/platform/harness"
)

// Per-case size caps, enforced by Check (the streamed input is also capped at MaxInput).
const (
	capIntArray    = 1_000_000
	capString      = 4_000_000
	capPermutation = 1_000_000
	capTree        = 200_000
	capGraphNodes  = 200_000
	capGraphEdges  = 500_000
	capOps         = 200_000
	capOpWeight    = 1_000_000
)

// The PCG stream constants: fixed per generator, part of its name@v (ASCII of a short tag).
const (
	streamGraph       = 0x67726170685f3031 // "graph_01"
	streamIntArray    = 0x696e745f61727231 // "int_arr1"
	streamOpSequence  = 0x6f705f7365713031 // "op_seq01"
	streamPermutation = 0x7065726d5f5f3031 // "perm__01"
	streamString      = 0x737472696e673031 // "string01"
	streamTree        = 0x747265655f5f3031 // "tree__01"
)

var (
	graphGen       = newFuncGen("graph@1", streamGraph, []Output{{"n", "int"}, {"edges", "int[][]"}}, compileGraph)
	intArrayGen    = newFuncGen("int_array@1", streamIntArray, []Output{{"array", "int[]"}}, compileIntArray)
	opSequenceGen  = newClassGen("op_sequence@1", streamOpSequence, compileOpSequence)
	permutationGen = newFuncGen("permutation@1", streamPermutation, []Output{{"array", "int[]"}}, compilePermutation)
	stringGen      = newFuncGen("string@1", streamString, []Output{{"string", "string"}}, compileString)
	treeGen        = newFuncGen("tree@1", streamTree, []Output{{"tree", "TreeNode"}}, compileTree)
)

// count checks a size param against [lo, cap].
func (f *fields) count(key string, v, lo, hi int64) (int, error) {
	if v < lo || v > hi {
		return 0, f.errf("%s %d out of [%d, %d]", key, v, lo, hi)
	}
	return int(v), nil
}

// ---- int_array@1 ----------------------------------------------------------------------

type intArrayPlan struct {
	n        int
	min, max int64
	distinct bool
	sorted   string
}

func compileIntArray(f *fields, types []harness.Type) (funcPlan, error) {
	n := f.integer("n", true, 0)
	p := &intArrayPlan{
		min:      f.integer("min", true, 0),
		max:      f.integer("max", true, 0),
		distinct: f.boolean("distinct"),
		sorted:   f.str("sorted", false, ""),
	}
	if err := f.done(); err != nil {
		return nil, err
	}
	var err error
	if p.n, err = f.count("n", n, 0, capIntArray); err != nil {
		return nil, err
	}
	if p.min > p.max {
		return nil, f.errf("min %d > max %d", p.min, p.max)
	}
	if err := f.within("min..max", p.min, p.max, types[0]); err != nil {
		return nil, err
	}
	if p.distinct && !fits(p.n, p.min, p.max) {
		return nil, f.errf("distinct: [%d, %d] holds fewer than n=%d values", p.min, p.max, p.n)
	}
	if p.sorted != "" && p.sorted != "asc" && p.sorted != "desc" {
		return nil, f.errf("sorted %q (want \"\", \"asc\" or \"desc\")", p.sorted)
	}
	return p, nil
}

func (p *intArrayPlan) generate(src *rand.PCG) funcCase {
	var a []int64
	if p.distinct {
		a = distinct(src, p.n, p.min, p.max)
	} else {
		a = make([]int64, p.n)
		for i := range a {
			a[i] = uniform(src, p.min, p.max)
		}
	}
	switch p.sorted {
	case "asc":
		slices.Sort(a)
	case "desc":
		slices.Sort(a)
		slices.Reverse(a)
	}
	return ints(a)
}

// fits reports whether [lo, hi] holds at least n distinct values.
func fits(n int, lo, hi int64) bool {
	return n == 0 || uint64(hi)-uint64(lo) >= uint64(n-1)
}

// distinct draws n distinct values of [lo, hi] (fits(n, lo, hi)) in random order with
// O(n) memory: a partial Fisher–Yates over the whole range when it holds ≤ 2n values,
// else rejection against a set (each draw then succeeds with probability ≥ 1/2).
func distinct(src *rand.PCG, n int, lo, hi int64) []int64 {
	span := uint64(hi) - uint64(lo) // the range holds span+1 values
	if n == 0 {
		return []int64{}
	}
	if span < 2*uint64(n) {
		size := int(span) + 1
		pool := make([]int64, size)
		for i := range pool {
			pool[i] = lo + int64(i)
		}
		for i := range n {
			j := i + int(below(src, uint64(size-i)))
			pool[i], pool[j] = pool[j], pool[i]
		}
		return pool[:n:n]
	}
	out := make([]int64, 0, n)
	seen := make(map[int64]struct{}, n)
	for len(out) < n {
		v := uniform(src, lo, hi)
		if _, dup := seen[v]; dup {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

// ints is a generated int[] output.
type ints []int64

func (a ints) write(o *out, _ int) {
	o.ch('[')
	for i, v := range a {
		if !o.ok() {
			return
		}
		if i > 0 {
			o.ch(',')
		}
		o.num(v)
	}
	o.ch(']')
}

// ---- string@1 -------------------------------------------------------------------------

const defaultAlphabet = "abcdefghijklmnopqrstuvwxyz"

type stringPlan struct {
	n     int
	alpha string
}

func compileString(f *fields, _ []harness.Type) (funcPlan, error) {
	n := f.integer("n", true, 0)
	p := &stringPlan{alpha: f.str("alphabet", false, defaultAlphabet)}
	if err := f.done(); err != nil {
		return nil, err
	}
	var err error
	if p.n, err = f.count("n", n, 0, capString); err != nil {
		return nil, err
	}
	if p.alpha == "" {
		return nil, f.errf("alphabet is empty")
	}
	var seen [128]bool
	for i := 0; i < len(p.alpha); i++ {
		c := p.alpha[i]
		if c < 0x20 || c > 0x7e {
			return nil, f.errf("alphabet byte %#02x is not printable ASCII (0x20..0x7e)", c)
		}
		if seen[c] {
			return nil, f.errf("alphabet repeats %q", c)
		}
		seen[c] = true
	}
	return p, nil
}

func (p *stringPlan) generate(src *rand.PCG) funcCase { return &stringCase{p: p, src: src} }

// stringCase draws its characters while it writes them (one output, written once).
type stringCase struct {
	p   *stringPlan
	src *rand.PCG
}

func (s *stringCase) write(o *out, _ int) {
	k := uint64(len(s.p.alpha))
	o.ch('"')
	for i := 0; i < s.p.n && o.ok(); i++ {
		c := s.p.alpha[below(s.src, k)]
		if c == '"' || c == '\\' {
			o.ch('\\')
		}
		o.ch(c)
	}
	o.ch('"')
}

// ---- permutation@1 --------------------------------------------------------------------

type permutationPlan struct {
	n    int
	base int64
}

func compilePermutation(f *fields, _ []harness.Type) (funcPlan, error) {
	n := f.integer("n", true, 0)
	p := &permutationPlan{base: f.integer("base", false, 1)}
	if err := f.done(); err != nil {
		return nil, err
	}
	var err error
	if p.n, err = f.count("n", n, 0, capPermutation); err != nil {
		return nil, err
	}
	if p.base != 0 && p.base != 1 {
		return nil, f.errf("base %d (want 0 or 1)", p.base)
	}
	return p, nil
}

func (p *permutationPlan) generate(src *rand.PCG) funcCase {
	a := make([]int64, p.n)
	for i := range a {
		a[i] = p.base + int64(i)
	}
	shuffle(src, a)
	return ints(a)
}

// ---- tree@1 ---------------------------------------------------------------------------

var treeShapes = []string{"random", "left", "right", "complete", "bst"}

type treePlan struct {
	n        int
	min, max int64
	shape    string
}

func compileTree(f *fields, types []harness.Type) (funcPlan, error) {
	n := f.integer("n", true, 0)
	p := &treePlan{
		min:   f.integer("min", true, 0),
		max:   f.integer("max", true, 0),
		shape: f.str("shape", false, "random"),
	}
	if err := f.done(); err != nil {
		return nil, err
	}
	var err error
	if p.n, err = f.count("n", n, 0, capTree); err != nil {
		return nil, err
	}
	if p.min > p.max {
		return nil, f.errf("min %d > max %d", p.min, p.max)
	}
	if err := f.within("min..max", p.min, p.max, types[0]); err != nil {
		return nil, err
	}
	if !slices.Contains(treeShapes, p.shape) {
		return nil, f.errf("shape %q (want one of %s)", p.shape, strings.Join(treeShapes, ", "))
	}
	if p.shape == "bst" && !fits(p.n, p.min, p.max) {
		return nil, f.errf("bst: [%d, %d] holds fewer than n=%d distinct values", p.min, p.max, p.n)
	}
	return p, nil
}

// treeCase is a binary tree of n nodes, node 0 the root; children are node indices, -1 none.
type treeCase struct {
	val         []int64
	left, right []int32
}

func (p *treePlan) generate(src *rand.PCG) funcCase {
	n := p.n
	t := &treeCase{val: make([]int64, n), left: make([]int32, n), right: make([]int32, n)}
	for i := range n {
		t.left[i], t.right[i] = -1, -1
	}
	if n == 0 {
		return t
	}
	if p.shape == "bst" {
		// Distinct values inserted in their (random) draw order: a random BST.
		copy(t.val, distinct(src, n, p.min, p.max))
		for i := 1; i < n; i++ {
			cur := int32(0)
			for {
				child := &t.right[cur]
				if t.val[i] < t.val[cur] {
					child = &t.left[cur]
				}
				if *child < 0 {
					*child = int32(i)
					break
				}
				cur = *child
			}
		}
		return t
	}
	for i := range t.val {
		t.val[i] = uniform(src, p.min, p.max)
	}
	switch p.shape {
	case "left":
		for i := 1; i < n; i++ {
			t.left[i-1] = int32(i)
		}
	case "right":
		for i := 1; i < n; i++ {
			t.right[i-1] = int32(i)
		}
	case "complete":
		for i := 1; i < n; i++ {
			if parent := (i - 1) / 2; i%2 == 1 {
				t.left[parent] = int32(i)
			} else {
				t.right[parent] = int32(i)
			}
		}
	default: // random: each new node takes a uniformly random free child slot
		slots := make([]int32, 0, n+1) // node*2 + side (0 left, 1 right)
		slots = append(slots, 0, 1)
		for i := 1; i < n; i++ {
			k := below(src, uint64(len(slots)))
			s := slots[k]
			slots[k] = slots[len(slots)-1]
			slots = slots[:len(slots)-1]
			if s%2 == 0 {
				t.left[s/2] = int32(i)
			} else {
				t.right[s/2] = int32(i)
			}
			slots = append(slots, int32(2*i), int32(2*i+1))
		}
	}
	return t
}

// write streams the LeetCode level order: the root, then each node's two child slots in
// BFS order, null for an empty slot, trailing nulls trimmed (harness's canonical TreeNode).
func (t *treeCase) write(o *out, _ int) {
	o.ch('[')
	if len(t.val) > 0 {
		o.num(t.val[0])
		queue := make([]int32, 1, len(t.val))
		nulls := 0
		for h := 0; h < len(queue) && o.ok(); h++ {
			x := queue[h]
			for _, c := range [2]int32{t.left[x], t.right[x]} {
				if c < 0 {
					nulls++
					continue
				}
				for ; nulls > 0; nulls-- {
					o.lit(",null")
				}
				o.ch(',')
				o.num(t.val[c])
				queue = append(queue, c)
			}
		}
	}
	o.ch(']')
}

// ---- graph@1 --------------------------------------------------------------------------

type graphPlan struct {
	n, m                                            int
	directed, weighted, connected, selfLoops, multi bool
	wmin, wmax, base                                int64
}

func compileGraph(f *fields, types []harness.Type) (funcPlan, error) {
	n := f.integer("n", true, 0)
	m := f.integer("m", true, 0)
	hasW := f.has("wmin") || f.has("wmax")
	p := &graphPlan{
		directed:  f.boolean("directed"),
		weighted:  f.boolean("weighted"),
		connected: f.boolean("connected"),
		selfLoops: f.boolean("self_loops"),
		multi:     f.boolean("multi"),
		wmin:      f.integer("wmin", false, 1),
		wmax:      f.integer("wmax", false, 1),
		base:      f.integer("base", false, 0),
	}
	if err := f.done(); err != nil {
		return nil, err
	}
	var err error
	if p.n, err = f.count("n", n, 1, capGraphNodes); err != nil {
		return nil, err
	}
	if p.m, err = f.count("m", m, 0, capGraphEdges); err != nil {
		return nil, err
	}
	if p.base != 0 && p.base != 1 {
		return nil, f.errf("base %d (want 0 or 1)", p.base)
	}
	if hasW && !p.weighted {
		return nil, f.errf("wmin/wmax need weighted: true")
	}
	if p.wmin > p.wmax {
		return nil, f.errf("wmin %d > wmax %d", p.wmin, p.wmax)
	}
	if p.weighted {
		if err := f.within("wmin..wmax", p.wmin, p.wmax, types[1]); err != nil {
			return nil, err
		}
	}
	if p.m > 0 && p.n == 1 && !p.selfLoops {
		return nil, f.errf("m=%d edges on one node need self_loops", p.m)
	}
	if p.connected && p.m < p.n-1 {
		return nil, f.errf("connected needs m ≥ n-1 = %d, got m=%d", p.n-1, p.m)
	}
	if lim := p.maxEdges(); !p.multi && uint64(p.m) > lim {
		return nil, f.errf("m=%d exceeds the %d distinct edges n=%d allows (set multi for repeats)", p.m, lim, p.n)
	}
	return p, nil
}

// maxEdges is the number of distinct edges (no multi).
func (p *graphPlan) maxEdges() uint64 {
	n := uint64(p.n)
	e := n * (n - 1)
	if !p.directed {
		e /= 2
	}
	if p.selfLoops {
		e += n
	}
	return e
}

type edge struct {
	u, v int32
	w    int64
}

type graphCase struct {
	n        int
	base     int64
	weighted bool
	edges    []edge
}

func (p *graphPlan) generate(src *rand.PCG) funcCase {
	n := p.n
	g := &graphCase{n: n, base: p.base, weighted: p.weighted, edges: make([]edge, 0, p.m)}
	var seen map[uint64]struct{} // distinct edges (no multi)
	key := func(u, v int32) uint64 {
		if !p.directed && u > v {
			u, v = v, u
		}
		return uint64(u)*uint64(n) + uint64(v)
	}
	add := func(u, v int32) {
		e := edge{u: u, v: v}
		if p.weighted {
			e.w = uniform(src, p.wmin, p.wmax)
		}
		g.edges = append(g.edges, e)
		if seen != nil {
			seen[key(u, v)] = struct{}{}
		}
	}
	dense := !p.multi && p.maxEdges() <= 2*uint64(p.m)
	if !p.multi {
		size := p.m
		if dense {
			size = n
		}
		seen = make(map[uint64]struct{}, size)
	}
	if p.connected && n > 1 {
		// A random spanning tree: each node of a random order joins an earlier one.
		order := make([]int32, n)
		for i := range order {
			order[i] = int32(i)
		}
		shuffle(src, order)
		for i := 1; i < n; i++ {
			u, v := order[below(src, uint64(i))], order[i]
			if coin(src) {
				u, v = v, u
			}
			add(u, v)
		}
	}
	if dense {
		// Few free edges left: enumerate them and pick by a partial Fisher–Yates.
		var cand [][2]int32
		for u := range int32(n) {
			for v := range int32(n) {
				if (u == v && !p.selfLoops) || (!p.directed && v < u) {
					continue
				}
				if _, dup := seen[key(u, v)]; !dup {
					cand = append(cand, [2]int32{u, v})
				}
			}
		}
		for i := 0; len(g.edges) < p.m; i++ {
			j := i + int(below(src, uint64(len(cand)-i)))
			cand[i], cand[j] = cand[j], cand[i]
			u, v := cand[i][0], cand[i][1]
			if !p.directed && coin(src) {
				u, v = v, u
			}
			add(u, v)
		}
	} else {
		for len(g.edges) < p.m {
			u, v := int32(below(src, uint64(n))), int32(below(src, uint64(n)))
			if u == v && !p.selfLoops {
				continue
			}
			if seen != nil {
				if _, dup := seen[key(u, v)]; dup {
					continue
				}
			}
			add(u, v)
		}
	}
	shuffle(src, g.edges)
	return g
}

// write streams output 0 (n) or 1 (edges: [[u,v(,w)],…], node numbers from base).
func (g *graphCase) write(o *out, k int) {
	if k == 0 {
		o.num(int64(g.n))
		return
	}
	o.ch('[')
	for i, e := range g.edges {
		if !o.ok() {
			return
		}
		if i > 0 {
			o.ch(',')
		}
		o.ch('[')
		o.num(int64(e.u) + g.base)
		o.ch(',')
		o.num(int64(e.v) + g.base)
		if g.weighted {
			o.ch(',')
			o.num(e.w)
		}
		o.ch(']')
	}
	o.ch(']')
}

// ---- op_sequence@1 --------------------------------------------------------------------

type opArg struct{ min, max int64 }

type opChoice struct {
	quoted []byte // the op name as a canonical JSON string
	weight uint64
	args   []opArg
}

type opPlan struct {
	n     int
	class []byte // the class name as a canonical JSON string
	ctor  []byte // the constructor's canonical argument list
	ops   []opChoice
	total uint64
}

func compileOpSequence(f *fields, sig *harness.Sig) (classPlan, error) {
	n := f.integer("n", true, 0)
	ctorRaw, hasCtor := f.list("ctor")
	opsRaw, hasOps := f.list("ops")
	if err := f.done(); err != nil {
		return nil, err
	}
	p := &opPlan{class: harness.AppendString(nil, sig.Name)}
	var err error
	if p.n, err = f.count("n", n, 0, capOps); err != nil {
		return nil, err
	}
	if !hasCtor && len(sig.Params) > 0 {
		return nil, f.errf("ctor is required: %s's constructor takes %d argument(s)", sig.Name, len(sig.Params))
	}
	if len(ctorRaw) != len(sig.Params) {
		return nil, f.errf("ctor lists %d argument(s); %s's constructor takes %d", len(ctorRaw), sig.Name, len(sig.Params))
	}
	p.ctor = append(p.ctor, '[')
	for i, prm := range sig.Params {
		v, err := harness.DecodeValue(prm.Type, ctorRaw[i])
		if err != nil {
			return nil, f.errf("ctor %s: %v", prm.Name, err)
		}
		if i > 0 {
			p.ctor = append(p.ctor, ',')
		}
		p.ctor = harness.AppendJSON(p.ctor, v)
	}
	p.ctor = append(p.ctor, ']')
	if !hasOps || len(opsRaw) == 0 {
		return nil, f.errf("ops must list at least one op of %s", sig.Name)
	}
	seen := map[string]bool{}
	for i, raw := range opsRaw {
		c, err := compileOp(fmt.Sprintf("%s: ops[%d]", f.where, i), raw, sig, seen)
		if err != nil {
			return nil, err
		}
		p.ops = append(p.ops, c)
		p.total += c.weight
	}
	return p, nil
}

func compileOp(where string, raw json.RawMessage, sig *harness.Sig, seen map[string]bool) (opChoice, error) {
	of, err := newFields(where, raw)
	if err != nil {
		return opChoice{}, err
	}
	name := of.str("name", true, "")
	weight := of.integer("weight", false, 1)
	argsRaw, _ := of.list("args")
	if err := of.done(); err != nil {
		return opChoice{}, err
	}
	var op *harness.Op
	for i := range sig.Ops {
		if sig.Ops[i].Name == name {
			op = &sig.Ops[i]
		}
	}
	switch {
	case name == sig.Name:
		return opChoice{}, of.errf("%s is the constructor (every case starts with it; set ctor), not an op", name)
	case op == nil:
		return opChoice{}, of.errf("%q is not an op of %s", name, sig.Name)
	case seen[name]:
		return opChoice{}, of.errf("op %s is listed twice", name)
	}
	seen[name] = true
	if weight < 1 || weight > capOpWeight {
		return opChoice{}, of.errf("weight %d out of [1, %d]", weight, capOpWeight)
	}
	if len(argsRaw) != len(op.Params) {
		return opChoice{}, of.errf("args lists %d range(s); %s takes %d argument(s)", len(argsRaw), name, len(op.Params))
	}
	c := opChoice{quoted: harness.AppendString(nil, name), weight: uint64(weight)}
	for k, prm := range op.Params {
		if prm.Type.Dims != 0 || (prm.Type.Base != harness.BaseInt && prm.Type.Base != harness.BaseInt64) {
			return opChoice{}, of.errf("%s.%s is %s; op_sequence@1 generates int and int64 arguments only", name, prm.Name, prm.Type)
		}
		af, err := newFields(fmt.Sprintf("%s.args[%d]", where, k), argsRaw[k])
		if err != nil {
			return opChoice{}, err
		}
		a := opArg{min: af.integer("min", true, 0), max: af.integer("max", true, 0)}
		if err := af.done(); err != nil {
			return opChoice{}, err
		}
		if a.min > a.max {
			return opChoice{}, af.errf("min %d > max %d", a.min, a.max)
		}
		if err := af.within(name+"."+prm.Name, a.min, a.max, prm.Type); err != nil {
			return opChoice{}, err
		}
		c.args = append(c.args, a)
	}
	return c, nil
}

// pick draws an op index by weight.
func (p *opPlan) pick(src *rand.PCG) int {
	r := below(src, p.total)
	for k := range p.ops {
		if r < p.ops[k].weight {
			return k
		}
		r -= p.ops[k].weight
	}
	return len(p.ops) - 1 // unreachable: r < total
}

// write streams {"args":[ctor,…],"ops":["Class",…]}: "args" sorts first, so the op
// choices are drawn (with their arguments) while the args stream and kept as indices for
// the ops array.
func (p *opPlan) write(o *out, src *rand.PCG) {
	o.lit(`{"args":[`)
	o.raw(p.ctor)
	chosen := make([]int32, p.n)
	for i := 0; i < p.n && o.ok(); i++ {
		k := p.pick(src)
		chosen[i] = int32(k)
		o.lit(",[")
		for a, r := range p.ops[k].args {
			if a > 0 {
				o.ch(',')
			}
			o.num(uniform(src, r.min, r.max))
		}
		o.ch(']')
	}
	o.lit(`],"ops":[`)
	o.raw(p.class)
	for _, k := range chosen {
		if !o.ok() {
			return
		}
		o.ch(',')
		o.raw(p.ops[k].quoted)
	}
	o.lit(`]}`)
}

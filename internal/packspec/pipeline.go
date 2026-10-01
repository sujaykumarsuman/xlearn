package packspec

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/sujaykumarsuman/xlearn/internal/course"
	"github.com/sujaykumarsuman/xlearn/internal/packspec/gen"
	"github.com/sujaykumarsuman/xlearn/internal/platform/checker"
	"github.com/sujaykumarsuman/xlearn/internal/platform/harness"
)

// The source → cases pipeline. Messages carry ids, paths, groups and counts only — never
// an input, an output or an expected value.

// Generous per-case CPU limits for programs that are not under test.
const (
	referenceCPUms = 20_000
	toolCPUms      = 20_000
)

// ItemSource is one pack item with its public half resolved.
type ItemSource struct {
	Course, ID string
	// Dir is the item directory on disk.
	Dir       string
	Pack      *Item
	Public    *course.ResolvedItem
	Part      *course.Part
	Sig       *harness.Sig
	Checker   checker.Checker
	Validator *Validator
	// Reference is the public Go reference, _code/solution.go (the ONLY source of expected
	// outputs: never AI-written, t1 §7.3).
	Reference []byte
}

// LoadItemSource resolves a pack item against its public item: the auto-graded code part,
// its signature (closed types, a known harness), checker, constraints, limits and the Go
// reference.
func LoadItemSource(packRoot, courseSlug, id string, pub *course.ResolvedItem) (*ItemSource, error) {
	dir := filepath.Join(packRoot, "courses", courseSlug, "items", id)
	b, err := os.ReadFile(filepath.Join(dir, ItemFile))
	if err != nil {
		return nil, err
	}
	pk, err := DecodeItem(b)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", ItemFile, err)
	}
	if err := pk.Validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", ItemFile, err)
	}
	if pub == nil {
		return nil, fmt.Errorf("no public item %s", id)
	}
	s := &ItemSource{Course: courseSlug, ID: id, Dir: dir, Pack: pk, Public: pub}
	for i := range pub.Item.Parts {
		p := &pub.Item.Parts[i]
		if p.Type == "code" && p.Grading == "auto" {
			if s.Part != nil {
				return nil, errors.New("more than one auto code part (one per pack item until a later format)")
			}
			s.Part = p
		}
	}
	if s.Part == nil {
		return nil, errors.New("the public item has no auto-graded code part (nothing to materialize)")
	}
	cfg := s.Part.Config
	if s.Sig, err = harness.ParseSig(cfg.Harness, cfg.Signature); err != nil {
		return nil, err
	}
	if s.Checker, err = checker.Lookup(cfg.Checker); err != nil {
		return nil, err
	}
	if t, ok := s.Sig.OutputType(); ok {
		if err := checker.Supports(s.Checker.Name(), t); err != nil {
			return nil, err
		}
	}
	if s.Validator, err = NewValidator(s.Sig, cfg.Constraints); err != nil {
		return nil, err
	}
	if cfg.Limits == nil {
		return nil, errors.New("the code part needs limits {time_ms, memory_mb} (the TL gate checks them)")
	}
	ref, ok := pub.References["go"]
	if !ok {
		return nil, fmt.Errorf("no public Go reference (%s): expected outputs are computed from it only", course.ReferenceFile("go"))
	}
	s.Reference = []byte(ref)
	return s, nil
}

// Key is the item's lock key.
func (s *ItemSource) Key() string { return ItemKey(s.Course, s.ID) }

func (s *ItemSource) label(what string) string { return s.Key() + " " + what }

// harnessProgram wraps a Go learner-shaped file with the generated harness.
func (s *ItemSource) harnessProgram(label string, learner []byte) (Program, error) {
	files, err := harness.Generate("go", s.Sig.Harness, s.Sig)
	if err != nil {
		return Program{}, err
	}
	p := Program{Label: label, Lang: "go", Mode: ModeHarness, Files: []SourceFile{{Name: harness.GoLearnerFile, Data: learner}}}
	for _, f := range files {
		p.Files = append(p.Files, SourceFile{Name: f.Path, Data: f.Data})
	}
	return p, nil
}

// readInputLine turns an edge / invalid / generator line ({args | ops+args[, tags]}) into a
// canonical input and its tags.
func (s *ItemSource) readInputLine(line []byte) ([]byte, []string, error) {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(line, &m); err != nil {
		return nil, nil, errors.New("not a JSON object")
	}
	var tags []string
	if t, ok := m["tags"]; ok {
		if err := json.Unmarshal(t, &tags); err != nil {
			return nil, nil, errors.New("tags must be a string array")
		}
		delete(m, "tags")
	}
	for k := range m {
		if k != "args" && k != "ops" {
			return nil, nil, fmt.Errorf("unexpected key %q (inputs only: args, ops, tags; expected outputs are computed)", k)
		}
	}
	raw, _ := json.Marshal(m)
	in, err := harness.CanonicalInput(s.Sig, raw)
	return in, tags, err
}

func normTags(tags []string, group string) ([]string, error) {
	set := map[string]bool{group: true}
	for _, t := range tags {
		if !tagRe.MatchString(t) {
			return nil, fmt.Errorf("tag %q must match %s", t, tagRe)
		}
		set[t] = true
	}
	out := make([]string, 0, len(set))
	for t := range set {
		out = append(out, t)
	}
	sort.Strings(out)
	n := 0
	for _, g := range []string{GroupEdge, GroupRandom, GroupPerf} {
		if set[g] {
			n++
		}
	}
	if n != 1 {
		return nil, fmt.Errorf("a case belongs to exactly one group (edge, random or perf), got %v", out)
	}
	return out, nil
}

// Inputs collects the item's cases without expected outputs, ordered edge → random → perf:
// tests/edge.jsonl, every gen[] cmd entry (run through the executor), every spec entry.
func (s *ItemSource) Inputs(ctx context.Context, ex Executor) ([]*Case, error) {
	var out []*Case
	limit := s.Pack.LiteralLimit()
	addLiteral := func(in []byte, tags []string, where string) error {
		if len(in) > limit {
			return fmt.Errorf("%s: a literal case is %d bytes, over the %d cap (perf inputs belong in a spec; a larger literal needs a stamped large_case_exception)", where, len(in), limit)
		}
		out = append(out, &Case{ID: CaseID(in), Input: in, Tags: tags})
		return nil
	}

	// Edge cases.
	if b, err := os.ReadFile(filepath.Join(s.Dir, filepath.FromSlash(EdgeFile))); err == nil {
		n := 0
		err := ScanLines(bytes.NewReader(b), func(line []byte) error {
			n++
			where := fmt.Sprintf("%s line %d", EdgeFile, n)
			in, tags, err := s.readInputLine(line)
			if err != nil {
				return fmt.Errorf("%s: %w", where, err)
			}
			if slices.Contains(tags, GroupPerf) || slices.Contains(tags, GroupRandom) {
				return fmt.Errorf("%s: an edge case cannot be tagged perf or random", where)
			}
			if tags, err = normTags(tags, GroupEdge); err != nil {
				return fmt.Errorf("%s: %w", where, err)
			}
			return addLiteral(in, tags, where)
		})
		if err != nil {
			return nil, err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	// Generator programs: one build per program, one process per case.
	type pending struct {
		entry int
		index int
		tags  []string
	}
	byCmd := map[string][]pending{}
	var cmds []string
	var runs = map[string][]Run{}
	for ei, g := range s.Pack.Gen {
		if g.Cmd == "" {
			continue
		}
		group := GroupRandom
		if slices.Contains(g.Tags, GroupPerf) {
			group = GroupPerf
		}
		tags, err := normTags(g.Tags, group)
		if err != nil {
			return nil, fmt.Errorf("gen[%d]: %w", ei, err)
		}
		if _, ok := byCmd[g.Cmd]; !ok {
			cmds = append(cmds, g.Cmd)
		}
		args := argsJSON(g.Args)
		for i := 0; i < g.Count; i++ {
			seed := Seed(s.ID, g.Cmd, args, i)
			byCmd[g.Cmd] = append(byCmd[g.Cmd], pending{ei, i, tags})
			runs[g.Cmd] = append(runs[g.Cmd], Run{Args: append([]string{"--seed", strconv.FormatUint(seed, 10)}, g.Args...)})
		}
	}
	for _, cmd := range cmds {
		src, err := os.ReadFile(filepath.Join(s.Dir, filepath.FromSlash(cmd)))
		if err != nil {
			return nil, err
		}
		p := Program{Label: s.label("generator " + cmd), Lang: "go", Mode: ModeStdio, Files: []SourceFile{{Name: "main.go", Data: src}}}
		res, err := ex.Execute(ctx, p, runs[cmd], Limits{CPUms: toolCPUms})
		if err != nil {
			return nil, err
		}
		if !res.CompileOK {
			return nil, fmt.Errorf("generator %s does not compile", cmd)
		}
		for k, r := range res.Runs {
			pd := byCmd[cmd][k]
			where := fmt.Sprintf("generator %s (gen[%d] case %d)", cmd, pd.entry, pd.index)
			if r.Killed || r.Exit != 0 || r.Signal != "" {
				return nil, fmt.Errorf("%s failed (exit %d%s)", where, r.Exit, killedNote(r))
			}
			line := bytes.TrimSpace(r.Output)
			if len(line) == 0 || bytes.ContainsRune(line, '\n') {
				return nil, fmt.Errorf("%s: must print exactly one input line", where)
			}
			in, tags, err := s.readInputLine(line)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", where, err)
			}
			if len(tags) > 0 {
				return nil, fmt.Errorf("%s: tags come from pack.json, not the generator", where)
			}
			if err := addLiteral(in, pd.tags, where); err != nil {
				return nil, err
			}
		}
	}

	// Seeded perf specs.
	for ei, g := range s.Pack.Gen {
		if g.Spec == nil {
			continue
		}
		if err := gen.Check(s.Sig, *g.Spec); err != nil {
			return nil, fmt.Errorf("gen[%d].spec: %w", ei, err)
		}
		cmd, params, err := specJSON(g.Spec)
		if err != nil {
			return nil, fmt.Errorf("gen[%d]: %w", ei, err)
		}
		tags, err := normTags(g.Tags, GroupPerf)
		if err != nil {
			return nil, fmt.Errorf("gen[%d]: %w", ei, err)
		}
		for i := 0; i < g.Count; i++ {
			c := &Case{Gen: g.Spec.Gen, Params: params, Seed: Seed(s.ID, cmd, "", i), Tags: tags}
			c.ID = CaseID(c.CanonicalInput())
			out = append(out, c)
		}
	}

	SortCases(out)
	seen := map[string]bool{}
	for _, c := range out {
		if seen[c.ID] {
			return nil, fmt.Errorf("duplicate case %s (the same input twice)", c.ID)
		}
		seen[c.ID] = true
	}
	return out, nil
}

func killedNote(r RunResult) string {
	switch {
	case r.Killed:
		return ", killed by the time limit"
	case r.Signal != "":
		return ", signal " + r.Signal
	}
	return ""
}

// InputPayload returns a case's canonical input (a perf case's is generated).
func (s *ItemSource) InputPayload(c *Case) ([]byte, error) {
	if !c.IsPerfSpec() {
		return c.Input, nil
	}
	var buf bytes.Buffer
	if err := gen.WriteInput(&buf, s.Sig, gen.Spec{Gen: c.Gen, Params: c.Params}, c.Seed); err != nil {
		return nil, fmt.Errorf("case %s: %w", c.ID, err)
	}
	return buf.Bytes(), nil
}

// decodedInput decodes a case's input.
func (s *ItemSource) decodedInput(c *Case) ([]byte, *harness.Input, error) {
	p, err := s.InputPayload(c)
	if err != nil {
		return nil, nil, err
	}
	in, err := harness.DecodeInput(s.Sig, p)
	if err != nil {
		return nil, nil, fmt.Errorf("case %s (%s): %w", c.ID, c.Group(), err)
	}
	return p, in, nil
}

// --- validators ---------------------------------------------------------------------

// ValidateReport counts what the validate gate checked.
type ValidateReport struct {
	Inputs, Invalid, Custom int
}

// Validate is the validators gate: the generic validator and every custom validator
// (validate/*.go, through the executor: stdin = the canonical input, exit 0 = valid) accept
// every edge, random and materialized perf input; every invalid/*.jsonl line is rejected by
// the codec, the generic validator or at least one custom validator.
func (s *ItemSource) Validate(ctx context.Context, ex Executor, cases []*Case) (ValidateReport, error) {
	rep := ValidateReport{Inputs: len(cases)}
	payloads := make([][]byte, len(cases))
	for i, c := range cases {
		p, in, err := s.decodedInput(c)
		if err != nil {
			return rep, err
		}
		if err := s.Validator.Check(in); err != nil {
			return rep, fmt.Errorf("case %s (%s): %w", c.ID, c.Group(), err)
		}
		payloads[i] = p
	}

	// invalid/*.jsonl: the codec or the generic validator may already reject a line.
	type invalidLine struct {
		where   string
		payload []byte
		done    bool
	}
	var invalid []*invalidLine
	files, _ := filepath.Glob(filepath.Join(s.Dir, "invalid", "*.jsonl"))
	sort.Strings(files)
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			return rep, err
		}
		n := 0
		err = ScanLines(bytes.NewReader(b), func(line []byte) error {
			n++
			il := &invalidLine{where: fmt.Sprintf("invalid/%s line %d", filepath.Base(f), n)}
			in, _, err := s.readInputLine(line)
			if err != nil {
				il.done = true // rejected by the codec
			} else if dec, err := harness.DecodeInput(s.Sig, in); err != nil || s.Validator.Check(dec) != nil {
				il.done = true
			}
			il.payload = in
			invalid = append(invalid, il)
			return nil
		})
		if err != nil {
			return rep, err
		}
	}
	rep.Invalid = len(invalid)

	customs, _ := filepath.Glob(filepath.Join(s.Dir, "validate", "*.go"))
	sort.Strings(customs)
	rep.Custom = len(customs)
	for _, f := range customs {
		src, err := os.ReadFile(f)
		if err != nil {
			return rep, err
		}
		name := "validate/" + filepath.Base(f)
		runs := make([]Run, 0, len(payloads)+len(invalid))
		for _, p := range payloads {
			runs = append(runs, Run{Input: p})
		}
		var pendingIdx []int
		for i, il := range invalid {
			if !il.done && il.payload != nil {
				runs = append(runs, Run{Input: il.payload})
				pendingIdx = append(pendingIdx, i)
			}
		}
		res, err := ex.Execute(ctx, Program{Label: s.label("validator " + name), Lang: "go", Mode: ModeStdio,
			Files: []SourceFile{{Name: "main.go", Data: src}}}, runs, Limits{CPUms: toolCPUms})
		if err != nil {
			return rep, err
		}
		if !res.CompileOK {
			return rep, fmt.Errorf("validator %s does not compile", name)
		}
		for i := range payloads {
			r := res.Runs[i]
			if r.Killed || r.Signal != "" {
				return rep, fmt.Errorf("validator %s crashed on case %s%s", name, cases[i].ID, killedNote(r))
			}
			if r.Exit != 0 {
				return rep, fmt.Errorf("case %s (%s): rejected by %s", cases[i].ID, cases[i].Group(), name)
			}
		}
		for k, i := range pendingIdx {
			r := res.Runs[len(payloads)+k]
			if r.Exit != 0 && !r.Killed && r.Signal == "" {
				invalid[i].done = true
			}
		}
	}
	for _, il := range invalid {
		if !il.done {
			return rep, fmt.Errorf("%s: accepted by every validator (an invalid input must be rejected)", il.where)
		}
	}
	return rep, nil
}

// --- expected outputs -----------------------------------------------------------------

// Materialized is an item's cases with expected outputs, and the reference's measurements.
type Materialized struct {
	Src        *ItemSource
	Cases      []*Case
	Ref        []RunResult
	BaselineKB int64
}

// Entries are the item's tests.lock entries.
func (m *Materialized) Entries() (map[string]string, error) { return Entries(m.Cases) }

// Materialize collects the inputs and computes every expected output by running the public
// Go reference through the executor (never written by hand or by AI).
func (s *ItemSource) Materialize(ctx context.Context, ex Executor) (*Materialized, error) {
	cases, err := s.Inputs(ctx, ex)
	if err != nil {
		return nil, err
	}
	prog, err := s.harnessProgram(s.label("reference "+course.ReferenceFile("go")), s.Reference)
	if err != nil {
		return nil, err
	}
	runs := make([]Run, len(cases))
	inputs := make([]*harness.Input, len(cases))
	for i, c := range cases {
		p, in, err := s.decodedInput(c)
		if err != nil {
			return nil, err
		}
		runs[i], inputs[i] = Run{Input: harness.AppendFrame(nil, p)}, in
	}
	res, err := ex.Execute(ctx, prog, runs, Limits{CPUms: referenceCPUms})
	if err != nil {
		return nil, err
	}
	if !res.CompileOK {
		return nil, fmt.Errorf("the Go reference %s does not compile with the %s harness", course.ReferenceFile("go"), s.Sig.Harness)
	}
	digest := s.Checker.Canonical() && harness.CanonicalizableOutput(s.Sig)
	for i, c := range cases {
		r := res.Runs[i]
		if r.Killed || r.Exit != 0 || r.Signal != "" {
			return nil, fmt.Errorf("the reference failed on case %s (%s): exit %d%s", c.ID, c.Group(), r.Exit, killedNote(r))
		}
		v, err := harness.DecodeOutputFor(s.Sig, inputs[i], r.Output)
		if err != nil {
			return nil, fmt.Errorf("the reference's result on case %s (%s): %w", c.ID, c.Group(), err)
		}
		switch {
		case c.IsPerfSpec() && digest:
			if !bytes.Equal(r.Output, harness.OKFrame(v)) {
				return nil, fmt.Errorf("case %s: the reference's fd-4 bytes are not canonical", c.ID)
			}
			sum := sha256.Sum256(r.Output)
			c.Expected = []byte(`{"bytes":` + strconv.Itoa(len(r.Output)) + `,"sha256":"` + hex.EncodeToString(sum[:]) + `"}`)
		case s.Checker.Name() == checker.AnyOf:
			c.Expected = append(append([]byte{'['}, harness.AppendJSON(nil, v)...), ']')
		default:
			c.Expected = harness.AppendJSON(nil, v)
		}
	}
	return &Materialized{Src: s, Cases: cases, Ref: res.Runs, BaselineKB: res.BaselineKB}, nil
}

// --- verdicts -----------------------------------------------------------------------

// Verdict classes (t4 §4.1), with "AC" for a full pass.
const (
	AC  = "AC"
	WA  = "WA"
	TLE = "TLE"
	MLE = "MLE"
	RE  = "RE"
	CE  = "CE"
)

// Outcome is a program's verdict over the case set: the class of the first failing case
// in order (within a case: TLE > MLE > RE > WA).
type Outcome struct {
	Verdict string
	// FirstFail is the index of the first failing case (-1 when AC).
	FirstFail int
	// PerfTLE reports a TLE on at least one perf case.
	PerfTLE bool
	// Passed counts passing cases.
	Passed int
}

// expectedValue decodes a literal expected (any_of: the alternatives list).
func (s *ItemSource) expectedValue(in *harness.Input, raw []byte) (harness.Value, error) {
	if s.Checker.Name() != checker.AnyOf {
		return harness.DecodeExpected(s.Sig, in, raw)
	}
	var alts []json.RawMessage
	if err := json.Unmarshal(raw, &alts); err != nil {
		return harness.Value{}, err
	}
	out := harness.Value{Kind: harness.KindList}
	for _, a := range alts {
		v, err := harness.DecodeExpected(s.Sig, in, a)
		if err != nil {
			return harness.Value{}, err
		}
		out.List = append(out.List, v)
	}
	return out, nil
}

// classify judges one run of a case.
func (m *Materialized) classify(i int, r RunResult, tlMS, memKB int64, baseline int64) (string, error) {
	c := m.Cases[i]
	switch {
	case r.Killed || r.CPUms > tlMS:
		return TLE, nil
	case memKB > 0 && r.PeakKB-baseline > memKB:
		return MLE, nil
	case r.Exit != 0 || r.Signal != "":
		return RE, nil
	}
	var digest struct {
		Bytes  int    `json:"bytes"`
		SHA256 string `json:"sha256"`
	}
	if c.IsPerfSpec() && json.Unmarshal(c.Expected, &digest) == nil && digest.SHA256 != "" {
		sum := sha256.Sum256(r.Output)
		if len(r.Output) == digest.Bytes && hex.EncodeToString(sum[:]) == digest.SHA256 {
			return AC, nil
		}
	}
	_, in, err := m.Src.decodedInput(c)
	if err != nil {
		return "", err
	}
	got, err := harness.DecodeOutputFor(m.Src.Sig, in, r.Output)
	var pe *harness.PanicError
	var he *harness.HarnessError
	switch {
	case errors.As(err, &pe):
		return RE, nil
	case errors.As(err, &he) && he.Code == "bad_input":
		return "", fmt.Errorf("case %s: the harness rejected its own input (a packlint bug)", c.ID)
	case errors.As(err, &he), errors.Is(err, harness.ErrValue):
		return WA, nil
	case err != nil:
		return RE, nil // no result frame
	}
	if digest.SHA256 != "" {
		return WA, nil
	}
	want, err := m.Src.expectedValue(in, c.Expected)
	if err != nil {
		return "", fmt.Errorf("case %s: expected: %w", c.ID, err)
	}
	if !m.Src.Checker.Check(got, want).OK {
		return WA, nil
	}
	return AC, nil
}

// judge runs a Go learner-shaped program over cases (indices into m.Cases) and judges it.
func (m *Materialized) judge(ctx context.Context, ex Executor, label string, src []byte, idx []int, tlMS, memKB int64, stopAfterKill bool) (*Outcome, error) {
	prog, err := m.Src.harnessProgram(label, src)
	if err != nil {
		return nil, err
	}
	runs := make([]Run, len(idx))
	for k, i := range idx {
		p, err := m.Src.InputPayload(m.Cases[i])
		if err != nil {
			return nil, err
		}
		runs[k] = Run{Input: harness.AppendFrame(nil, p)}
	}
	res, err := ex.Execute(ctx, prog, runs, Limits{CPUms: tlMS, StopAfterKill: stopAfterKill})
	if err != nil {
		return nil, err
	}
	out := &Outcome{Verdict: AC, FirstFail: -1}
	if !res.CompileOK {
		out.Verdict, out.FirstFail = CE, 0
		return out, nil
	}
	for k, i := range idx {
		r := res.Runs[k]
		if r.Skipped {
			continue
		}
		v, err := m.classify(i, r, tlMS, memKB, res.BaselineKB)
		if err != nil {
			return nil, err
		}
		if v == TLE && m.Cases[i].Group() == GroupPerf {
			out.PerfTLE = true
		}
		if v == AC {
			out.Passed++
			continue
		}
		if out.FirstFail < 0 {
			out.Verdict, out.FirstFail = v, k
		}
	}
	return out, nil
}

// --- gates --------------------------------------------------------------------------

// Gate statuses.
const (
	Pass    = "pass"
	Fail    = "fail"
	Pending = "pending"
)

// GateResult is one gate line: ids, paths and counts only.
type GateResult struct {
	Item   string `json:"item"`
	Gate   string `json:"gate"`
	Target string `json:"target,omitempty"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}

func (g GateResult) String() string {
	s := fmt.Sprintf("%s %s", g.Item, g.Gate)
	if g.Target != "" {
		s += " " + g.Target
	}
	s += ": " + g.Status
	if g.Detail != "" {
		s += " (" + g.Detail + ")"
	}
	return s
}

func (m *Materialized) gate(gate, target, status, format string, args ...any) GateResult {
	return GateResult{Item: m.Src.Key(), Gate: gate, Target: target, Status: status, Detail: fmt.Sprintf(format, args...)}
}

// limits returns the public TL (ms) and memory (KiB).
func (m *Materialized) limits() (int64, int64) {
	l := m.Src.Part.Config.Limits
	return int64(l.TimeMS), int64(l.MemoryMB) * 1024
}

// smallCases are the correctness cases (edge and random).
func (m *Materialized) smallCases() []int {
	var out []int
	for i, c := range m.Cases {
		if c.Group() != GroupPerf {
			out = append(out, i)
		}
	}
	return out
}

func (m *Materialized) allCases() []int {
	out := make([]int, len(m.Cases))
	for i := range out {
		out[i] = i
	}
	return out
}

// langOf maps a file extension to an executor language.
func langOf(name string) string {
	switch path.Ext(name) {
	case ".go":
		return "go"
	case ".cpp", ".cc":
		return "cpp"
	case ".py":
		return "python"
	}
	return ""
}

// Oracle is the correctness gate: the reference agrees with submissions/brute.* (a
// different algorithm) on the small cases, checker-aware. A C++/Python oracle is
// pending(harness) until m3-04.
func (m *Materialized) Oracle(ctx context.Context, ex Executor) []GateResult {
	brutes, _ := filepath.Glob(filepath.Join(m.Src.Dir, filepath.FromSlash(BruteBase)+".*"))
	sort.Strings(brutes)
	if len(brutes) == 0 {
		return []GateResult{m.gate("oracle", "", Fail, "no %s.<ext> oracle", BruteBase)}
	}
	var out []GateResult
	for _, f := range brutes {
		rel := "submissions/" + filepath.Base(f)
		if langOf(rel) != "go" {
			out = append(out, m.gate("oracle", rel, Pending, "harness: %s runs from m3-04", langOf(rel)))
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			out = append(out, m.gate("oracle", rel, Fail, "%v", err))
			continue
		}
		idx := m.smallCases()
		o, err := m.judge(ctx, ex, m.Src.label("oracle "+rel), src, idx, referenceCPUms, 0, false)
		switch {
		case err != nil:
			out = append(out, m.gate("oracle", rel, Fail, "%v", err))
		case o.Verdict != AC:
			out = append(out, m.gate("oracle", rel, Fail, "disagrees: %s on small case %d of %d", o.Verdict, o.FirstFail+1, len(idx)))
		default:
			out = append(out, m.gate("oracle", rel, Pass, "%d small cases agree", len(idx)))
		}
	}
	return out
}

// WrongResult is one wrong solution's gate outcome.
type WrongResult struct {
	Gate    GateResult
	Wrong   Wrong
	Outcome *Outcome
}

// WrongSolutions is the wrong-solutions gate: each declared wrong solution gets exactly its
// declared verdict on the full case set (CE via the compile, WA via the checker, TLE via the
// public TL, RE via the exit or a panic frame, MLE via the memory limit). REJECTED needs
// m3-04's source lint and C++/Python need its harnesses: pending.
func (m *Materialized) WrongSolutions(ctx context.Context, ex Executor) []WrongResult {
	tl, mem := m.limits()
	var out []WrongResult
	for _, w := range m.Src.Pack.Wrong {
		wr := WrongResult{Wrong: w}
		switch {
		case w.Expect == "REJECTED":
			wr.Gate = m.gate("wrong", w.File, Pending, "lint: REJECTED is decided by m3-04's source lint")
		case langOf(w.File) != "go":
			wr.Gate = m.gate("wrong", w.File, Pending, "harness: %s runs from m3-04", langOf(w.File))
		default:
			src, err := os.ReadFile(filepath.Join(m.Src.Dir, filepath.FromSlash(w.File)))
			if err != nil {
				wr.Gate = m.gate("wrong", w.File, Fail, "%v", err)
				break
			}
			o, err := m.judge(ctx, ex, m.Src.label("wrong "+w.File), src, m.allCases(), tl, mem, true)
			switch {
			case err != nil:
				wr.Gate = m.gate("wrong", w.File, Fail, "%v", err)
			case o.Verdict == w.Expect:
				wr.Outcome = o
				wr.Gate = m.gate("wrong", w.File, Pass, "%s as declared, first at case %d of %d", o.Verdict, o.FirstFail+1, len(m.Cases))
			default:
				wr.Outcome = o
				wr.Gate = m.gate("wrong", w.File, Fail, "got %s, declared %s", o.Verdict, w.Expect)
			}
		}
		out = append(out, wr)
	}
	return out
}

// Timing is timing.json, written by the TL gate.
type Timing struct {
	BaselineKB        int64   `json:"baseline_kb"`
	Cases             int     `json:"cases"`
	Image             string  `json:"image"`
	MemoryFloorMB     int64   `json:"memory_floor_mb"`
	Provisional       bool    `json:"provisional"`
	ReferenceMaxCPUms int64   `json:"reference_max_cpu_ms"`
	ReferencePeakKB   int64   `json:"reference_peak_kb"`
	Scale             float64 `json:"scale"`
	SumTLms           int64   `json:"sum_tl_ms"`
	TimeLimitMS       int64   `json:"time_limit_ms"`
	TLFloorMS         int64   `json:"tl_floor_ms"`
}

// TL budget rules (t3 §7.2).
const (
	TLFactor    = 3
	TLFloorMS   = 1000
	SumTLCapMS  = 40_000
	ProvisScale = 1.0
)

// TimeLimits is the time-limit gate (t3 §7.2): time_ms ≥ max(3 × reference max CPU ×
// scale, 1 s) with scale 1.0 on the pinned image (provisional until m3-13 re-gates with the
// runner baselines × production multipliers); Σ TL over the hidden cases ≤ 40 CPU-s;
// memory_mb ≥ 2 × the reference's peak + baseline; every `expect: TLE` wrong solution
// exceeds the TL on at least one perf case. It writes timing.json beside pack.json.
func (m *Materialized) TimeLimits(wrong []WrongResult, image string) (GateResult, *Timing) {
	tl, memKB := m.limits()
	t := &Timing{Cases: len(m.Cases), Image: image, Provisional: true, Scale: ProvisScale, TimeLimitMS: tl, BaselineKB: m.BaselineKB}
	for _, r := range m.Ref {
		t.ReferenceMaxCPUms = max(t.ReferenceMaxCPUms, r.CPUms)
		t.ReferencePeakKB = max(t.ReferencePeakKB, r.PeakKB)
	}
	t.TLFloorMS = max(int64(float64(TLFactor*t.ReferenceMaxCPUms)*t.Scale), TLFloorMS)
	t.SumTLms = tl * int64(len(m.Cases))
	net := max(t.ReferencePeakKB-t.BaselineKB, 0)
	t.MemoryFloorMB = (2*net + t.BaselineKB + 1023) / 1024
	var fails []string
	if tl < t.TLFloorMS {
		fails = append(fails, fmt.Sprintf("time_ms %d < floor %d", tl, t.TLFloorMS))
	}
	if t.SumTLms > SumTLCapMS {
		fails = append(fails, fmt.Sprintf("sum TL %d ms over %d cases > %d", t.SumTLms, len(m.Cases), SumTLCapMS))
	}
	if memKB < 2*net+t.BaselineKB {
		fails = append(fails, fmt.Sprintf("memory_mb %d < floor %d", memKB/1024, t.MemoryFloorMB))
	}
	for _, w := range wrong {
		if w.Wrong.Expect != TLE || w.Gate.Status == Pending {
			continue
		}
		if w.Outcome == nil || !w.Outcome.PerfTLE {
			fails = append(fails, fmt.Sprintf("%s (expect TLE) never exceeds the TL on a perf case", w.Wrong.File))
		}
	}
	detail := fmt.Sprintf("provisional; time_ms %d ≥ floor %d; sum %d ms; memory floor %d MB", tl, t.TLFloorMS, t.SumTLms, t.MemoryFloorMB)
	if len(fails) > 0 {
		return m.gate("tl", "", Fail, "%s", strings.Join(fails, "; ")), t
	}
	return m.gate("tl", "", Pass, "%s", detail), t
}

// WriteTiming writes timing.json beside the item's pack.json.
func (m *Materialized) WriteTiming(t *Timing) error {
	b, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(m.Src.Dir, TimingFile), append(b, '\n'), 0o644)
}

// ReferenceSyntax syntax-checks the public C++ and Python references (their execution
// gates are pending(harness) until m3-04 runs all three through the jail).
func (s *ItemSource) ReferenceSyntax(ctx context.Context, ex Executor) []GateResult {
	var out []GateResult
	for _, l := range []struct{ ext, lang string }{{"cpp", "cpp"}, {"py", "python"}} {
		src, ok := s.Public.References[l.ext]
		if !ok {
			continue
		}
		name := course.ReferenceFile(l.ext)
		files := []SourceFile{{Name: path.Base(name), Data: []byte(src)}}
		g := GateResult{Item: s.Key(), Gate: "syntax", Target: name}
		ok2, _, err := ex.SyntaxCheck(ctx, l.lang, files)
		switch {
		case err != nil:
			g.Status, g.Detail = Fail, err.Error()
		case !ok2:
			g.Status, g.Detail = Fail, "does not compile"
		default:
			g.Status, g.Detail = Pass, "execution gates pending(harness) until m3-04"
		}
		out = append(out, g)
	}
	return out
}

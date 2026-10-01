package packspec

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"

	"github.com/klauspost/compress/zstd"

	"github.com/sujaykumarsuman/xlearn/internal/platform/harness"
)

// A materialized case (t1 §7.1, t4 §2.6, §4.1) is one canonical JSON line of
// cases.jsonl.zst:
//
//	literal  {"args":[…],"expected":…,"id":"sha256:…","tags":[…]}                 func-json@1
//	         {"args":[[…],…],"expected":[…],"id":…,"ops":["Class",…],"tags":[…]}   class-ops@1
//	perf     {"expected":…,"gen":"int_array@1","id":…,"params":{…},"seed":N,"tags":["perf"]}
//
// Keys are sorted, values canonical (internal/platform/harness). The id is a hash of the
// INPUT only, sha256("xlearn.case@1\n" + canonical input), so fixing a case makes a new
// case; the canonical input of a literal case is its fd-3 payload ({"args"…[,"ops"…]}) and
// of a perf case {"gen","params","seed"}. A perf case's expected is {"bytes","sha256"} of
// the reference's fd-4 frame when the checker is canonical (exact) and the output has no
// float; otherwise the literal result. Cases are ordered edge → random → perf.

// Case groups, in evaluation order.
const (
	GroupEdge   = "edge"
	GroupRandom = "random"
	GroupPerf   = "perf"
)

var groupRank = map[string]int{GroupEdge: 0, GroupRandom: 1, GroupPerf: 2}

// CaseIDPrefix is the case id's domain separation.
const CaseIDPrefix = "xlearn.case@1\n"

// Case is one materialized case.
type Case struct {
	// ID is sha256 of the canonical input ("sha256:<64 hex>").
	ID string
	// Input is the canonical input payload of a literal case (the fd-3 frame's JSON).
	Input []byte
	// Gen, Params and Seed are a perf case's spec (Params canonical).
	Gen    string
	Params []byte
	Seed   uint64
	// Expected is the canonical expected value: the result, or a perf digest object.
	Expected []byte
	// Tags are sorted and unique, and hold exactly one group tag.
	Tags []string
}

// IsPerfSpec reports a generated perf case (no literal input).
func (c *Case) IsPerfSpec() bool { return c.Gen != "" }

// Group is the case's group (edge, random or perf).
func (c *Case) Group() string {
	for _, g := range []string{GroupPerf, GroupEdge} {
		if contains(c.Tags, g) {
			return g
		}
	}
	return GroupRandom
}

// CanonicalInput is what the case id hashes.
func (c *Case) CanonicalInput() []byte {
	if !c.IsPerfSpec() {
		return c.Input
	}
	b := []byte(`{"gen":`)
	b = harness.AppendString(b, c.Gen)
	b = append(b, `,"params":`...)
	b = append(b, c.Params...)
	b = append(b, `,"seed":`...)
	b = strconv.AppendUint(b, c.Seed, 10)
	return append(b, '}')
}

// CaseID is sha256("xlearn.case@1\n" + canonical input).
func CaseID(canonicalInput []byte) string {
	h := sha256.New()
	h.Write([]byte(CaseIDPrefix))
	h.Write(canonicalInput)
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}

// Line returns the case's canonical JSON line (no newline).
func (c *Case) Line() ([]byte, error) {
	members := map[string][]byte{"expected": c.Expected, "id": harness.AppendString(nil, c.ID)}
	tags := []byte{'['}
	for i, t := range c.Tags {
		if i > 0 {
			tags = append(tags, ',')
		}
		tags = harness.AppendString(tags, t)
	}
	members["tags"] = append(tags, ']')
	if c.IsPerfSpec() {
		members["gen"] = harness.AppendString(nil, c.Gen)
		members["params"] = c.Params
		members["seed"] = strconv.AppendUint(nil, c.Seed, 10)
	} else {
		var in map[string]json.RawMessage
		if err := json.Unmarshal(c.Input, &in); err != nil {
			return nil, fmt.Errorf("case %s: input: %w", c.ID, err)
		}
		for k, v := range in {
			members[k] = v
		}
	}
	keys := make([]string, 0, len(members))
	for k := range members {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	b := []byte{'{'}
	for i, k := range keys {
		if i > 0 {
			b = append(b, ',')
		}
		b = harness.AppendString(b, k)
		b = append(b, ':')
		b = append(b, members[k]...)
	}
	return append(b, '}'), nil
}

// LineHash is a tests.lock entry: sha256 of the canonical line.
func LineHash(line []byte) string {
	s := sha256.Sum256(line)
	return "sha256:" + hex.EncodeToString(s[:])
}

// ParseLine decodes a canonical case line and checks it is canonical: re-encoding gives the
// same bytes and the id matches the input. The input itself is checked against a signature
// by the caller (harness.CanonicalInput).
func ParseLine(line []byte) (*Case, error) {
	var raw map[string]json.RawMessage
	dec := json.NewDecoder(bytes.NewReader(line))
	dec.UseNumber()
	if err := dec.Decode(&raw); err != nil {
		return nil, fmt.Errorf("case line: %w", err)
	}
	c := &Case{}
	if err := json.Unmarshal(raw["id"], &c.ID); err != nil {
		return nil, errors.New("case line: id must be a string")
	}
	if err := json.Unmarshal(raw["tags"], &c.Tags); err != nil {
		return nil, errors.New("case line: tags must be a string array")
	}
	c.Expected = raw["expected"]
	if len(c.Expected) == 0 {
		return nil, errors.New("case line: no expected")
	}
	if g, ok := raw["gen"]; ok {
		if err := json.Unmarshal(g, &c.Gen); err != nil {
			return nil, errors.New("case line: gen must be a string")
		}
		c.Params = raw["params"]
		var err error
		if c.Seed, err = strconv.ParseUint(string(raw["seed"]), 10, 64); err != nil {
			return nil, errors.New("case line: seed must be a uint64")
		}
	} else {
		in := []byte(`{"args":`)
		in = append(in, raw["args"]...)
		if ops, ok := raw["ops"]; ok {
			in = append(in, `,"ops":`...)
			in = append(in, ops...)
		}
		c.Input = append(in, '}')
	}
	again, err := c.Line()
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(again, line) {
		return nil, errors.New("case line: not canonical")
	}
	if CaseID(c.CanonicalInput()) != c.ID {
		return nil, fmt.Errorf("case line %s: the id does not hash its input", c.ID)
	}
	return c, nil
}

// SortCases orders cases edge → random → perf, stably.
func SortCases(cs []*Case) {
	sort.SliceStable(cs, func(i, j int) bool { return groupRank[cs[i].Group()] < groupRank[cs[j].Group()] })
}

// --- cases.jsonl.zst ------------------------------------------------------------------

// The zstd encoding is deterministic: one frame (EncodeAll), single-threaded, a fixed
// level, a content checksum and an explicit frame for empty input — so the same lines give
// the same bytes, and the same image digest.
var (
	zenc, _ = zstd.NewWriter(nil,
		zstd.WithEncoderConcurrency(1),
		zstd.WithEncoderLevel(zstd.SpeedDefault),
		zstd.WithEncoderCRC(true),
		zstd.WithZeroFrames(true),
	)
	zdec, _ = zstd.NewReader(nil, zstd.WithDecoderConcurrency(1), zstd.WithDecoderMaxMemory(256<<20))
)

// MaxCasesFile bounds a decompressed cases file.
const MaxCasesFile = 256 << 20

// EncodeCases joins the cases' canonical lines (each ending in "\n") and compresses them.
func EncodeCases(cs []*Case) ([]byte, error) {
	var raw []byte
	for _, c := range cs {
		l, err := c.Line()
		if err != nil {
			return nil, err
		}
		raw = append(append(raw, l...), '\n')
	}
	return zenc.EncodeAll(raw, nil), nil
}

// Compress is the deterministic zstd encoding of raw bytes.
func Compress(raw []byte) []byte { return zenc.EncodeAll(raw, nil) }

// Decompress reverses Compress.
func Decompress(b []byte) ([]byte, error) {
	out, err := zdec.DecodeAll(b, nil)
	if err != nil {
		return nil, fmt.Errorf("cases: zstd: %w", err)
	}
	if len(out) > MaxCasesFile {
		return nil, errors.New("cases: decompressed file too large")
	}
	return out, nil
}

// DecodeCases decompresses a cases file and parses every line (canonical and id-checked).
func DecodeCases(b []byte) ([]*Case, error) {
	raw, err := Decompress(b)
	if err != nil {
		return nil, err
	}
	var out []*Case
	err = ScanLines(bytes.NewReader(raw), func(line []byte) error {
		c, err := ParseLine(line)
		if err != nil {
			return err
		}
		out = append(out, c)
		return nil
	})
	return out, err
}

// ScanLines calls fn for every non-empty line of r (up to LargeCaseCap plus slack each).
func ScanLines(r io.Reader, fn func(line []byte) error) error {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64<<10), 16<<20)
	for sc.Scan() {
		line := bytes.TrimRight(sc.Bytes(), "\r")
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		if err := fn(line); err != nil {
			return err
		}
	}
	return sc.Err()
}

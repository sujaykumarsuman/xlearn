package packspec

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/sujaykumarsuman/xlearn/internal/packspec/gen"
	"github.com/sujaykumarsuman/xlearn/internal/platform/harness"
)

// tests.lock (t1 §3.3, §7.2) pins every materialized case: per item, case id → sha256 of
// its canonical line (expected included). `packlint lock --write` rewrites it after an
// intended change; `--verify` regenerates every case — generators run again with the same
// seeds, expected outputs computed again from the public Go reference — and requires the
// same bytes. Its header records the tool versions a regeneration depends on; a change there
// is what triggers the private CI's full run.
//
//	{"cases": {"<course>/<id>": {"sha256:<case id>": "sha256:<line>"}},
//	 "format": 1,
//	 "tools": {"gen": "<registry>", "harness": "class-ops@1,func-json@1",
//	           "images": {"cpp": "…@sha256:…", "go": "…@sha256:…", "python": "…@sha256:…"},
//	           "packlint": "lock@1"}}
//
// Encoded as indented JSON with sorted keys and a trailing newline, so it is byte-stable.

// LockVersion is the materialization algorithm's version (the case line, the seed rule,
// the expected rule). A change is a full re-lock.
const LockVersion = "lock@1"

// Lock is tests.lock.
type Lock struct {
	Cases  map[string]map[string]string `json:"cases"`
	Format int                          `json:"format"`
	Tools  Tools                        `json:"tools"`
}

// Tools are the versions a regeneration depends on.
type Tools struct {
	Gen      string            `json:"gen"`
	Harness  string            `json:"harness"`
	Images   map[string]string `json:"images"`
	Packlint string            `json:"packlint"`
}

// CurrentTools returns this packlint's tool versions with the executor's images.
func CurrentTools(images map[string]string) Tools {
	hs := append([]string{}, harness.Harnesses...)
	sort.Strings(hs)
	img := map[string]string{}
	for k, v := range images {
		img[k] = v
	}
	return Tools{Gen: gen.Version(), Harness: strings.Join(hs, ","), Images: img, Packlint: LockVersion}
}

// Equal reports identical tool versions.
func (t Tools) Equal(o Tools) bool {
	a, _ := json.Marshal(t)
	b, _ := json.Marshal(o)
	return string(a) == string(b)
}

// ItemKey is a lock key: "<course>/<id>".
func ItemKey(course, id string) string { return course + "/" + id }

// NewLock is an empty format-1 lock.
func NewLock(t Tools) *Lock { return &Lock{Cases: map[string]map[string]string{}, Format: 1, Tools: t} }

// DecodeLock strictly decodes tests.lock. mi-07's scaffold lock `{}` decodes as an empty
// lock with format 0.
func DecodeLock(b []byte) (*Lock, error) {
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(b, &probe); err != nil {
		return nil, fmt.Errorf("%s: %w", LockFile, err)
	}
	if len(probe) == 0 {
		return &Lock{Cases: map[string]map[string]string{}}, nil
	}
	var l Lock
	if err := decodeStrict(b, &l); err != nil {
		return nil, fmt.Errorf("%s: %w", LockFile, err)
	}
	if l.Format != 1 {
		return nil, fmt.Errorf("%s: format %d is not 1", LockFile, l.Format)
	}
	if l.Cases == nil {
		l.Cases = map[string]map[string]string{}
	}
	for k, cs := range l.Cases {
		for id, h := range cs {
			if !hashRe.MatchString(id) || !hashRe.MatchString(h) {
				return nil, fmt.Errorf("%s: %s: malformed entry", LockFile, k)
			}
		}
	}
	return &l, nil
}

// Encode returns the lock's bytes.
func (l *Lock) Encode() ([]byte, error) {
	if l.Cases == nil {
		l.Cases = map[string]map[string]string{}
	}
	b, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

// Entries returns an item's lock entries for its cases.
func Entries(cs []*Case) (map[string]string, error) {
	out := make(map[string]string, len(cs))
	for _, c := range cs {
		line, err := c.Line()
		if err != nil {
			return nil, err
		}
		if _, dup := out[c.ID]; dup {
			return nil, fmt.Errorf("duplicate case %s (the same input twice)", c.ID)
		}
		out[c.ID] = LineHash(line)
	}
	return out, nil
}

// LockDiff compares an item's regenerated entries with the lock: counts only (ids are
// internal; the diff never prints a payload).
type LockDiff struct {
	Missing, Extra, Changed int
}

// Empty reports no difference.
func (d LockDiff) Empty() bool { return d.Missing == 0 && d.Extra == 0 && d.Changed == 0 }

func (d LockDiff) String() string {
	return fmt.Sprintf("%d missing, %d new, %d changed", d.Missing, d.Extra, d.Changed)
}

// Diff compares locked entries (want) with regenerated ones (got).
func Diff(want, got map[string]string) LockDiff {
	var d LockDiff
	for id, h := range want {
		switch g, ok := got[id]; {
		case !ok:
			d.Missing++
		case g != h:
			d.Changed++
		}
	}
	for id := range got {
		if _, ok := want[id]; !ok {
			d.Extra++
		}
	}
	return d
}

// Seed derives a generated case's seed (t1 §7.2: seed = hash(item_id, cmd)): the first 8
// bytes, big-endian, of sha256(item ‖ 0x00 ‖ cmd ‖ 0x00 ‖ args ‖ 0x00 ‖ index). args is the
// canonical JSON array of the entry's args; for a spec entry, cmd is the spec's canonical
// JSON and args is empty. index is the case's 0-based decimal index within the entry.
func Seed(item, cmd, args string, index int) uint64 {
	h := sha256.New()
	h.Write([]byte(item))
	h.Write([]byte{0})
	h.Write([]byte(cmd))
	h.Write([]byte{0})
	h.Write([]byte(args))
	h.Write([]byte{0})
	h.Write([]byte(strconv.Itoa(index)))
	return binary.BigEndian.Uint64(h.Sum(nil)[:8])
}

// argsJSON is the canonical JSON array of a cmd entry's args.
func argsJSON(args []string) string {
	b := []byte{'['}
	for i, a := range args {
		if i > 0 {
			b = append(b, ',')
		}
		b = harness.AppendString(b, a)
	}
	return string(append(b, ']'))
}

// specJSON is a spec's canonical JSON {"gen","params"}.
func specJSON(s *gen.Spec) (string, []byte, error) {
	params := []byte("{}")
	if len(s.Params) > 0 {
		var err error
		if params, err = harness.CanonicalJSON(s.Params); err != nil {
			return "", nil, fmt.Errorf("spec params: %w", err)
		}
		if params[0] != '{' {
			return "", nil, errors.New("spec params must be an object")
		}
	}
	b := []byte(`{"gen":`)
	b = harness.AppendString(b, s.Gen)
	b = append(b, `,"params":`...)
	b = append(b, params...)
	return string(append(b, '}')), params, nil
}

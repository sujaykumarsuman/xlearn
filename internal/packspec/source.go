// Package packspec is the private eval pack's FORMAT (t1 §3.3, §7.1; ADR-0027 §2): the Go
// types of the root and per-item pack.json in the xlearn-evalpack repo, strict decoding and
// structural validation; and, from m3-02, the pipeline that turns a pack's source into the
// built pack — canonical cases and their ids, tests.lock, the generic validator derived from
// the public constraints, the built /manifest.json (ReadManifest + VerifyFiles for judge),
// the builder and the image listing check. It holds the format and the code only, never
// data: every pack file lives in the private repo, and tests here use hand-made synthetic
// values (the public fixture pack is internal/judge/testdata, marked SYNTHETIC.md).
//
// Named packspec, never evalpack: the repo's .gitignore ignores any directory called
// evalpack/ (a nested private clone must never be committed).
//
// cmd/packlint drives it. A library: no database, no service imports.
package packspec

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/packspec/gen"
)

// Pack layout names.
const (
	// RootFile is the pack root's pack.json.
	RootFile = "pack.json"
	// ItemFile is an item directory's pack.json.
	ItemFile = "pack.json"
	// ItemsGlob matches the item directories: courses/<slug>/items/<id>.
	ItemsGlob = "courses/*/items/*"
)

// Item-relative and root names of format 1 (m3-02).
const (
	// EdgeFile holds the hand-picked inputs, one JSON object per line: {args | ops+args, tags}.
	EdgeFile = "tests/edge.jsonl"
	// TimingFile is written by `packlint exec --gate tl` (provisional until m3-13).
	TimingFile = "timing.json"
	// BruteBase is the oracle's path without its extension (submissions/brute.<ext>).
	BruteBase = "submissions/brute"
	// LockFile is the pack root's tests.lock.
	LockFile = "tests.lock"
	// DraftsDir holds unstamped prose at the pack root; it is never built.
	DraftsDir = "drafts"
)

// FormatMajors are the root format_major values this package reads: 0 is mi-07's
// scaffold (items may be authored against m3-01's item format), 1 is the format m3-02
// completes (cases, tests.lock, generators).
var FormatMajors = []int{0, 1}

// FormatMajor is the format this package builds.
const FormatMajor = 1

// ItemDirs are the only subdirectories an item directory may hold (t1 §3.3). Anything
// else beside ItemFiles is an undeclared file.
var ItemDirs = []string{"tests", "gen", "validate", "invalid", "submissions", "keys", "anchors", "exemplars"}

// ItemFiles are the only files directly in an item directory: pack.json and the TL gate's
// timing.json.
var ItemFiles = []string{ItemFile, TimingFile}

// Expects are the verdicts a wrong solution may declare.
var Expects = []string{"WA", "TLE", "RE", "MLE", "CE", "REJECTED"}

// MaxAcceptedHashes bounds accepts_contract_hashes (ADR-0027 §2: ≤ 2, the old and the new
// contract while either repo ships first).
const MaxAcceptedHashes = 2

// LiteralCap is the canonical-input cap of a literal case (t4 §9.4); a stamped
// large_case_exception raises it to at most LargeCaseCap.
const (
	LiteralCap   = 256 << 10
	LargeCaseCap = 2 << 20
)

// MaxGenCount bounds one gen[] entry's case count.
const MaxGenCount = 1000

// Root is the pack's root pack.json.
type Root struct {
	FormatMajor int    `json:"format_major"`
	Version     string `json:"version"`
}

// Item is courses/<slug>/items/<id>/pack.json.
type Item struct {
	// Item is the public item id; it equals the directory name.
	Item string `json:"item"`
	// AcceptsContractHashes lists the public contract_hash values this pack was written
	// against (1..2, "sha256:<64 hex>"): the live one must be among them.
	AcceptsContractHashes []string `json:"accepts_contract_hashes"`
	// Gen are the generated cases (m3-02): a CI-only generator program under gen/, or a
	// seeded perf spec of the public registry (internal/packspec/gen).
	Gen []GenEntry `json:"gen,omitempty"`
	// Wrong are the wrong solutions with the verdict each must get.
	Wrong []Wrong `json:"wrong,omitempty"`
	// Keys maps a key-graded part id or a `key_source: pack` probe id to its key file,
	// relative to the item directory, under keys/.
	Keys map[string]string `json:"keys,omitempty"`
	// Review holds the pack's review stamp.
	Review *Review `json:"review,omitempty"`
	// LargeCaseException raises the literal-case cap for this item (owner-stamped, ≤ 2 MiB).
	LargeCaseException *LargeCaseException `json:"large_case_exception,omitempty"`
}

// GenEntry is one gen[] entry: exactly one of Cmd (with Args) or Spec, Count cases, Tags.
// Seeds are derived (Seed), never written by hand.
type GenEntry struct {
	// Cmd is a generator program, gen/<name>.go: run through the executor once per case
	// with `--seed <u64>` then Args; it prints exactly one input line ({args | ops+args}).
	Cmd  string   `json:"cmd,omitempty"`
	Args []string `json:"args,omitempty"`
	// Spec is a seeded perf spec of the public registry: the pack stores the spec and the
	// seed, never generator code (t4 §4.1). Spec cases are always group perf.
	Spec  *gen.Spec `json:"spec,omitempty"`
	Count int       `json:"count"`
	Tags  []string  `json:"tags,omitempty"`
}

// LargeCaseException is the owner's stamp for literal cases above LiteralCap.
type LargeCaseException struct {
	MaxBytes int    `json:"max_bytes"`
	Stamped  string `json:"stamped,omitempty"`
}

// Wrong is one wrong solution: a file under submissions/ and its expected verdict.
type Wrong struct {
	File   string `json:"file"`
	Expect string `json:"expect"`
	// Category is the mistake category it models (the course manifest's taxonomy).
	Category string `json:"category,omitempty"`
}

// Review is the pack's review stamp: when the owner reviewed the hidden tests.
type Review struct {
	Tests string `json:"tests,omitempty"`
}

// Stamped reports whether the pack's tests are review-stamped.
func (it *Item) Stamped() bool { return it.Review != nil && it.Review.Tests != "" }

// LiteralLimit is the item's literal-case cap: LiteralCap, or a stamped exception's.
func (it *Item) LiteralLimit() int {
	if e := it.LargeCaseException; e != nil && e.Stamped != "" && e.MaxBytes > LiteralCap && e.MaxBytes <= LargeCaseCap {
		return e.MaxBytes
	}
	return LiteralCap
}

// DecodeRoot strictly decodes the root pack.json.
func DecodeRoot(b []byte) (*Root, error) {
	var r Root
	if err := decodeStrict(b, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// DecodeItem strictly decodes an item pack.json.
func DecodeItem(b []byte) (*Item, error) {
	var it Item
	if err := decodeStrict(b, &it); err != nil {
		return nil, err
	}
	return &it, nil
}

// SupportedFormat reports whether this package reads the root's format_major.
func (r *Root) SupportedFormat() bool {
	for _, m := range FormatMajors {
		if r.FormatMajor == m {
			return true
		}
	}
	return false
}

var (
	hashRe   = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
	semverRe = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)
	dateRe   = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}$`)
	keyIDRe  = regexp.MustCompile(`^[a-z][a-z0-9_-]*$`)
	tagRe    = regexp.MustCompile(`^[a-z][a-z0-9_-]*$`)
)

// IsSemver reports MAJOR.MINOR.PATCH.
func IsSemver(s string) bool { return semverRe.MatchString(s) }

// Validate checks the root's shape (not whether the format is supported).
func (r *Root) Validate() error {
	if !semverRe.MatchString(r.Version) {
		return fmt.Errorf("version %q is not MAJOR.MINOR.PATCH", r.Version)
	}
	return nil
}

// Validate checks an item pack.json on its own: the hash list, the declared files and
// verdicts, the generators, the key map and the stamps. packlint checks it against the
// public item and the files on disk.
func (it *Item) Validate() error {
	var errs []error
	add := func(format string, args ...any) { errs = append(errs, fmt.Errorf(format, args...)) }
	if strings.TrimSpace(it.Item) == "" {
		add("item is required")
	}
	switch n := len(it.AcceptsContractHashes); {
	case n == 0:
		add("accepts_contract_hashes needs the live contract hash (packlint hash --item %s)", it.Item)
	case n > MaxAcceptedHashes:
		add("accepts_contract_hashes lists %d hashes; at most %d (the old and the new contract)", n, MaxAcceptedHashes)
	}
	seen := map[string]bool{}
	for i, h := range it.AcceptsContractHashes {
		if !hashRe.MatchString(h) {
			add("accepts_contract_hashes[%d] is not sha256:<64 hex>", i)
		}
		if seen[h] {
			add("accepts_contract_hashes[%d] is a duplicate", i)
		}
		seen[h] = true
	}
	files := map[string]bool{}
	for i, w := range it.Wrong {
		if err := checkRel(w.File, "submissions"); err != nil {
			add("wrong[%d].file: %v", i, err)
		}
		if files[w.File] {
			add("wrong[%d].file %q is declared twice", i, w.File)
		}
		files[w.File] = true
		if !contains(Expects, w.Expect) {
			add("wrong[%d].expect %q is not one of %v", i, w.Expect, Expects)
		}
	}
	for id, f := range it.Keys {
		if !keyIDRe.MatchString(id) {
			add("keys: %q is not a part or probe id", id)
		}
		if err := checkRel(f, "keys"); err != nil {
			add("keys[%s]: %v", id, err)
		}
	}
	if it.Review != nil && it.Review.Tests != "" && !isDate(it.Review.Tests) {
		add("review.tests %q must be an ISO date (YYYY-MM-DD)", it.Review.Tests)
	}
	for i, g := range it.Gen {
		p := fmt.Sprintf("gen[%d]", i)
		switch {
		case (g.Cmd == "") == (g.Spec == nil):
			add("%s needs exactly one of cmd or spec", p)
		case g.Cmd != "":
			if err := checkRel(g.Cmd, "gen"); err != nil {
				add("%s.cmd: %v", p, err)
			} else if !strings.HasSuffix(g.Cmd, ".go") {
				add("%s.cmd %q: generators are Go programs (gen/<name>.go)", p, g.Cmd)
			}
		default:
			if len(g.Args) > 0 {
				add("%s: args belong to a cmd entry (a spec's params are in the spec)", p)
			}
			if _, err := gen.Lookup(g.Spec.Gen); err != nil {
				add("%s.spec.gen: %v", p, err)
			}
			for _, tg := range g.Tags {
				if tg == GroupRandom {
					add("%s: a spec entry is always group perf, not %q", p, tg)
				}
			}
		}
		if g.Count < 1 || g.Count > MaxGenCount {
			add("%s.count must be 1..%d", p, MaxGenCount)
		}
		for _, tg := range g.Tags {
			if !tagRe.MatchString(tg) {
				add("%s.tags: %q must match %s", p, tg, tagRe)
			}
			if tg == GroupEdge {
				add("%s.tags: edge cases live in %s", p, EdgeFile)
			}
		}
	}
	if e := it.LargeCaseException; e != nil {
		if e.MaxBytes <= LiteralCap || e.MaxBytes > LargeCaseCap {
			add("large_case_exception.max_bytes must be in (%d, %d]", LiteralCap, LargeCaseCap)
		}
		if e.Stamped != "" && !isDate(e.Stamped) {
			add("large_case_exception.stamped %q must be an ISO date (YYYY-MM-DD)", e.Stamped)
		}
	}
	return errors.Join(errs...)
}

func isDate(s string) bool {
	_, err := time.Parse("2006-01-02", s)
	return err == nil && dateRe.MatchString(s)
}

// checkRel requires a clean, relative, slash-separated path under dir/.
func checkRel(p, dir string) error {
	switch {
	case p == "":
		return errors.New("is required")
	case strings.Contains(p, `\`) || path.IsAbs(p) || path.Clean(p) != p || strings.HasPrefix(p, "../"):
		return fmt.Errorf("%q must be a clean relative path", p)
	case !strings.HasPrefix(p, dir+"/"):
		return fmt.Errorf("%q must be under %s/", p, dir)
	}
	for _, seg := range strings.Split(p, "/") {
		if strings.HasPrefix(seg, ".") {
			return fmt.Errorf("%q has a dotfile segment", p)
		}
	}
	return nil
}

// Files returns every file the pack.json declares (wrong solutions, keys and generator
// programs), sorted and unique.
func (it *Item) Files() []string {
	seen := map[string]bool{}
	var out []string
	add := func(f string) {
		if !seen[f] {
			seen[f] = true
			out = append(out, f)
		}
	}
	for _, w := range it.Wrong {
		add(w.File)
	}
	for _, f := range it.Keys {
		add(f)
	}
	for _, g := range it.Gen {
		if g.Cmd != "" {
			add(g.Cmd)
		}
	}
	sort.Strings(out)
	return out
}

func decodeStrict(b []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("decode: %w", err)
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return errors.New("decode: trailing data after the JSON value")
	}
	return nil
}

func contains[T comparable](set []T, v T) bool {
	for _, s := range set {
		if s == v {
			return true
		}
	}
	return false
}

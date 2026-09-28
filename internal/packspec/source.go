// Package packspec is the private eval pack's SOURCE FORMAT (t1 §3.3, §7.1; ADR-0027
// §2): the Go types of the root and per-item pack.json in the xlearn-evalpack repo, strict
// decoding and structural validation. It holds the format only, never data: every pack
// file lives in the private repo, and tests here use hand-made synthetic values.
//
// Named packspec, never evalpack: the repo's .gitignore ignores any directory called
// evalpack/ (a nested private clone must never be committed).
//
// cmd/packlint (m3-01) reads it; m3-02 extends it (gen[], large_case_exception, cases,
// tests.lock, the built manifest). Stdlib-only; no database, no service imports.
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

// FormatMajors are the root format_major values this package reads: 0 is mi-07's
// scaffold (items may be authored against m3-01's item format), 1 is the format m3-02
// completes (cases, tests.lock, generators).
var FormatMajors = []int{0, 1}

// ItemDirs are the only subdirectories an item directory may hold (t1 §3.3). Anything
// else beside pack.json is an undeclared file.
var ItemDirs = []string{"tests", "gen", "validate", "invalid", "submissions", "keys", "anchors", "exemplars"}

// Expects are the verdicts a wrong solution may declare.
var Expects = []string{"WA", "TLE", "RE", "MLE", "CE", "REJECTED"}

// MaxAcceptedHashes bounds accepts_contract_hashes (ADR-0027 §2: ≤ 2, the old and the new
// contract while either repo ships first).
const MaxAcceptedHashes = 2

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
	// Wrong are the wrong solutions with the verdict each must get.
	Wrong []Wrong `json:"wrong,omitempty"`
	// Keys maps a key-graded part id or a `key_source: pack` probe id to its key file,
	// relative to the item directory, under keys/.
	Keys map[string]string `json:"keys,omitempty"`
	// Review holds the pack's review stamp.
	Review *Review `json:"review,omitempty"`
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
)

// Validate checks the root's shape (not whether the format is supported).
func (r *Root) Validate() error {
	if !semverRe.MatchString(r.Version) {
		return fmt.Errorf("version %q is not MAJOR.MINOR.PATCH", r.Version)
	}
	return nil
}

// Validate checks an item pack.json on its own: the hash list, the declared files and
// verdicts, the key map and the stamp date. packlint checks it against the public item
// and the files on disk.
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
	if it.Review != nil && it.Review.Tests != "" {
		if _, err := time.Parse("2006-01-02", it.Review.Tests); err != nil || !dateRe.MatchString(it.Review.Tests) {
			add("review.tests %q must be an ISO date (YYYY-MM-DD)", it.Review.Tests)
		}
	}
	return errors.Join(errs...)
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

// Files returns every file the pack.json declares (wrong solutions and keys), sorted.
func (it *Item) Files() []string {
	var out []string
	for _, w := range it.Wrong {
		out = append(out, w.File)
	}
	for _, f := range it.Keys {
		out = append(out, f)
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

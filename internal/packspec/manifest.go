package packspec

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"regexp"
	"sort"
	"strings"
)

// The built pack's /manifest.json (t1 §3.3), EXACTLY this shape — judge may decode it
// strictly, so there are no extra fields (the course is already in every files path):
//
//	{"format_major": 1, "version": "X.Y.Z", "validated_against": "<public commit sha>",
//	 "items": {"<id>": {"item_hash": "sha256:…", "accepts_contract_hashes": ["sha256:…"],
//	                    "grader_kinds": ["code"], "files": {"courses/<slug>/items/<id>/cases.jsonl.zst": "sha256:…"}}}}
//
// item_hash = sha256("xlearn.item@1\n" + the sorted lines "<path> <sha256>\n"). ReadManifest
// (strict) and VerifyFiles (streaming sha256, never a whole file in memory — judge runs in
// 256 Mi) are the pure reader and verifier m3-05's loader uses; its statuses and index stay
// m3-05's. The writer is the same type.

// ManifestFile is the built pack's manifest path.
const ManifestFile = "manifest.json"

// ItemHashPrefix is item_hash's domain separation.
const ItemHashPrefix = "xlearn.item@1\n"

// Manifest is /manifest.json.
type Manifest struct {
	FormatMajor      int                     `json:"format_major"`
	Version          string                  `json:"version"`
	ValidatedAgainst string                  `json:"validated_against"`
	Items            map[string]ManifestItem `json:"items"`
}

// ManifestItem is one built item.
type ManifestItem struct {
	ItemHash              string            `json:"item_hash"`
	AcceptsContractHashes []string          `json:"accepts_contract_hashes"`
	GraderKinds           []string          `json:"grader_kinds"`
	Files                 map[string]string `json:"files"`
}

var (
	shaRe       = regexp.MustCompile(`^[0-9a-f]{40}$`)
	itemPathRe  = regexp.MustCompile(`^courses/([a-z0-9]+(?:-[a-z0-9]+)*)/items/([A-Za-z0-9-]+)/(.+)$`)
	graderKinds = []string{"code", "key", "ai_rubric"}
)

// ItemHash is the item_hash of a files map.
func ItemHash(files map[string]string) string {
	paths := make([]string, 0, len(files))
	for p := range files {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	h := sha256.New()
	h.Write([]byte(ItemHashPrefix))
	for _, p := range paths {
		fmt.Fprintf(h, "%s %s\n", p, files[p])
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}

// FileHash is a manifest file entry: "sha256:<hex>".
func FileHash(b []byte) string {
	s := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(s[:])
}

// Encode returns the manifest's bytes: indented JSON with sorted keys and a trailing
// newline (deterministic).
func (m *Manifest) Encode() ([]byte, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

// ReadManifest strictly decodes and validates a manifest (unknown fields, trailing data, a
// bad hash or path are errors).
func ReadManifest(r io.Reader) (*Manifest, error) {
	b, err := io.ReadAll(io.LimitReader(r, 64<<20))
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := decodeStrict(b, &m); err != nil {
		return nil, fmt.Errorf("manifest: %w", err)
	}
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return &m, nil
}

// Validate checks the manifest's shape: the version, the public sha, each item's hashes,
// grader kinds and file paths (under courses/<slug>/items/<id>/, one course per item).
func (m *Manifest) Validate() error {
	var errs []error
	add := func(format string, args ...any) { errs = append(errs, fmt.Errorf("manifest: "+format, args...)) }
	if m.FormatMajor != FormatMajor {
		add("format_major %d is not %d", m.FormatMajor, FormatMajor)
	}
	if !semverRe.MatchString(m.Version) {
		add("version %q is not MAJOR.MINOR.PATCH", m.Version)
	}
	if !shaRe.MatchString(m.ValidatedAgainst) {
		add("validated_against %q is not a 40-hex commit sha", m.ValidatedAgainst)
	}
	if m.Items == nil {
		add("items is required")
	}
	for id, it := range m.Items {
		if it.ItemHash != ItemHash(it.Files) {
			add("item %s: item_hash does not match its files", id)
		}
		if len(it.AcceptsContractHashes) == 0 || len(it.AcceptsContractHashes) > MaxAcceptedHashes {
			add("item %s: accepts_contract_hashes needs 1..%d hashes", id, MaxAcceptedHashes)
		}
		for _, h := range it.AcceptsContractHashes {
			if !hashRe.MatchString(h) {
				add("item %s: accepts_contract_hashes holds a malformed hash", id)
			}
		}
		if len(it.GraderKinds) == 0 {
			add("item %s: grader_kinds is empty", id)
		}
		for _, k := range it.GraderKinds {
			if !contains(graderKinds, k) {
				add("item %s: grader kind %q is not one of %v", id, k, graderKinds)
			}
		}
		if len(it.Files) == 0 {
			add("item %s: no files", id)
		}
		course := ""
		for p, h := range it.Files {
			sm := itemPathRe.FindStringSubmatch(p)
			switch {
			case sm == nil || sm[2] != id || path.Clean(p) != p:
				add("item %s: file %q is not under courses/<slug>/items/%s/", id, p, id)
			case course != "" && sm[1] != course:
				add("item %s: files in two courses", id)
			case !AllowedItemFile(sm[3]):
				add("item %s: file %q is not an allowed pack file", id, p)
			}
			if sm != nil {
				course = sm[1]
			}
			if !hashRe.MatchString(h) {
				add("item %s: file %q has a malformed hash", id, p)
			}
		}
	}
	return errors.Join(errs...)
}

// ItemCourse returns the course slug of an item's files.
func (it *ManifestItem) ItemCourse() string {
	for p := range it.Files {
		if sm := itemPathRe.FindStringSubmatch(p); sm != nil {
			return sm[1]
		}
	}
	return ""
}

// VerifyFiles streams every listed file of every item from fsys (the mounted image root)
// through sha256 and compares it with the manifest. It returns the items that failed (id →
// error); an item absent from the result verified.
func (m *Manifest) VerifyFiles(fsys fs.FS) map[string]error {
	bad := map[string]error{}
	for id, it := range m.Items {
		for _, p := range sortedKeys(it.Files) {
			got, err := streamHash(fsys, p)
			if err != nil {
				bad[id] = fmt.Errorf("%s: %w", p, err)
				break
			}
			if got != it.Files[p] {
				bad[id] = fmt.Errorf("%s: sha256 mismatch", p)
				break
			}
		}
	}
	return bad
}

func streamHash(fsys fs.FS, p string) (string, error) {
	f, err := fsys.Open(p)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil)), nil
}

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// AllowedItemFile reports whether an item-relative path may ship in the built pack:
// cases.jsonl.zst, or a file under keys/, anchors/ or exemplars/ (no dotfiles).
func AllowedItemFile(rel string) bool {
	if rel == CasesFile {
		return true
	}
	top, rest, ok := strings.Cut(rel, "/")
	if !ok || rest == "" || !contains(builtDirs, top) {
		return false
	}
	for _, seg := range strings.Split(rel, "/") {
		if seg == "" || strings.HasPrefix(seg, ".") {
			return false
		}
	}
	return true
}

// CasesFile is a built item's cases.
const CasesFile = "cases.jsonl.zst"

// builtDirs are the item subdirectories copied into the built pack.
var builtDirs = []string{"keys", "anchors", "exemplars"}

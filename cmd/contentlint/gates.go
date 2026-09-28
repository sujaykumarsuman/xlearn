package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/sujaykumarsuman/xlearn/internal/course"
	"github.com/sujaykumarsuman/xlearn/internal/course/lint"
	"github.com/sujaykumarsuman/xlearn/internal/curriculum"
)

// The public content gates m1-09 deferred to m3-01: the t4 §5.6 structure lints, the
// hints/editorial stamp gate, the label-edit flag on key-graded parts, and the repo-wide
// pack-artefact pass. The rules themselves live in internal/course/lint (packlint and
// judge import the same ones).

// --- t4 §5.6 structure lints (public-registry half) -----------------------------------

func (l *linter) checkStructure(content *curriculum.Content) {
	if content == nil {
		return
	}
	for _, slug := range sortedKeys(content.Manifests) {
		for _, f := range lint.Manifest(content.Manifests[slug]) {
			l.fail("structure", "courses/%s/course.json: %s", slug, f)
		}
	}
	for _, cc := range content.Courses {
		concepts := map[string]bool{}
		for _, c := range cc.Concepts {
			concepts[c.Slug] = true
		}
		for i := range cc.Items {
			it := &cc.Items[i].Item
			for _, f := range lint.Item(it, content.Manifests[cc.Slug], concepts) {
				l.fail("structure", "courses/%s/items/%s/item.json: %s", cc.Slug, it.ID, f)
			}
		}
	}
}

// --- the diff gates: stamp gate + label-edit flag --------------------------------------

// itemFileRe splits a content path into course, item id and the item-relative path.
var itemFileRe = regexp.MustCompile(`^courses/([^/]+)/items/([^/]+)/(.+)$`)

// resolveBase returns the PR base the diff gates compare against: -base /
// CONTENTLINT_BASE when set (CI: the PR's base sha, or the push's previous head), else
// the merge base of HEAD and origin/main. "" skips the gates with a note.
func (l *linter) resolveBase() (string, bool) {
	if b := strings.TrimSpace(l.base); b != "" && strings.Trim(b, "0") != "" {
		out, err := l.git("rev-parse", "--verify", "--quiet", b+"^{commit}")
		if err != nil {
			l.fail("gates", "base %s is not a commit here (a shallow clone needs fetch-depth: 0): %v", b, err)
			return "", false
		}
		return strings.TrimSpace(out), true
	}
	out, err := l.git("merge-base", "HEAD", "origin/main")
	if err != nil {
		l.note("stamp gate and label-edit flag skipped: no base (set CONTENTLINT_BASE or -base, or fetch origin/main)")
		return "", false
	}
	return strings.TrimSpace(out), true
}

// changedFiles lists content files added or modified since base (working tree, including
// untracked files), relative to the curriculum root, with their git status letter.
func (l *linter) changedFiles(base string) (map[string]string, error) {
	out, err := l.git("diff", "--name-status", "--no-renames", "--relative", base, "--", ".")
	if err != nil {
		return nil, err
	}
	changed := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		st, p, ok := strings.Cut(line, "\t")
		if ok {
			changed[p] = st
		}
	}
	untracked, err := l.git("ls-files", "--others", "--exclude-standard", "--", ".")
	if err != nil {
		return nil, err
	}
	for _, p := range strings.Fields(untracked) {
		changed[p] = "A"
	}
	return changed, nil
}

func (l *linter) checkDiffGates(content *curriculum.Content) {
	if content == nil {
		return
	}
	base, ok := l.resolveBase()
	if !ok {
		return
	}
	changed, err := l.changedFiles(base)
	if err != nil {
		l.fail("gates", "diff against %s: %v", base, err)
		return
	}
	items := map[string]*course.Item{}
	for _, cc := range content.Courses {
		for i := range cc.Items {
			items[cc.Items[i].Item.ID] = &cc.Items[i].Item
		}
	}
	approved := lint.Approved(l.prBody)
	for _, p := range sortedKeys(changed) {
		st := changed[p]
		m := itemFileRe.FindStringSubmatch(p)
		if m == nil || strings.HasPrefix(st, "D") {
			continue
		}
		it := items[m[2]]
		if it == nil {
			continue // not a loadable item (the loader reports it)
		}
		// Stamp gate (t1 §7.2): an added or changed hint or editorial file needs its stamp.
		if stamp := lint.StampFor(m[3]); stamp != "" && !lint.Stamped(it, stamp) {
			l.fail("stamp", "%s: added or changed, but item %s has no review.%s stamp (stamp it after your review; keep unreviewed drafts local or in the private repo's drafts/)", p, it.ID, stamp)
		}
		// Label-edit flag (t1 §3.4 rule 4) on a changed item.json.
		if m[3] == course.ItemFile && st == "M" {
			old, err := l.itemAt(base, p)
			if err != nil {
				l.fail("label-edit", "%s at %s: %v", p, base, err)
				continue
			}
			for _, e := range lint.LabelEdits(old, it) {
				if approved[e.Key()] {
					l.note("label edit %s confirmed as a typo by the PR body", e.Key())
					continue
				}
				l.fail("label-edit", "%s: the label of key-graded %s/%s changed (%q -> %q) under the same id, so the pack's key may no longer match. Mint a new id (a contract change: the pack lists the new contract hash), or confirm a typo with `%s` in the PR body",
					p, e.Part, e.ID, e.Old, e.New, e.Token())
			}
		}
	}
}

// itemAt decodes a content-relative item.json at a git ref.
func (l *linter) itemAt(ref, rel string) (*course.Item, error) {
	prefix, err := l.git("rev-parse", "--show-prefix")
	if err != nil {
		return nil, err
	}
	out, err := l.git("show", ref+":"+strings.TrimSpace(prefix)+rel)
	if err != nil {
		return nil, err
	}
	return course.DecodeItem([]byte(out))
}

// unstamped lists, per item, the stamps its existing hint/editorial sections still lack
// (v1's converted sections are grandfathered until they change).
func unstamped(content *curriculum.Content) []string {
	if content == nil {
		return nil
	}
	var out []string
	for _, cc := range content.Courses {
		for _, ri := range cc.Items {
			need := map[string]int{}
			for _, s := range ri.Sections {
				var stamp string
				switch s.Stage {
				case "hint":
					stamp = lint.StampHints
				case "solution":
					stamp = lint.StampEditorial
				}
				if stamp != "" && !lint.Stamped(&ri.Item, stamp) {
					need[stamp]++
				}
			}
			if len(need) == 0 {
				continue
			}
			var parts []string
			for _, s := range []string{lint.StampHints, lint.StampEditorial} {
				if n := need[s]; n > 0 {
					parts = append(parts, fmt.Sprintf("%s (%d section(s))", s, n))
				}
			}
			out = append(out, fmt.Sprintf("courses/%s/items/%s: %s", cc.Slug, ri.Item.ID, strings.Join(parts, ", ")))
		}
	}
	return out
}

// --- the repo-wide pack-artefact pass ------------------------------------------------

// SyntheticMarker is the first line of a SYNTHETIC.md: the only thing that lets a pack
// artefact name live in the public repo, inside an internal/**/testdata/ tree.
const SyntheticMarker = "SYNTHETIC — hand-made test data, never derived from xlearn-evalpack."

// isPackArtefact matches the private pack's artefact names (t1 §3.3).
func isPackArtefact(name string) bool {
	name = strings.ToLower(name)
	return name == "tests.lock" || strings.HasPrefix(name, "cases.jsonl") || strings.HasSuffix(name, ".jsonl.zst")
}

// packArtefactProblems applies the repo-wide rule to tracked files: a pack artefact name
// fails anywhere except inside an internal/**/testdata/ tree whose nearest testdata/
// directory holds a SYNTHETIC.md whose first line is SyntheticMarker. marker returns a
// file's first line (ok=false when it does not exist).
func packArtefactProblems(files []string, marker func(p string) (string, bool)) []string {
	var out []string
	for _, f := range files {
		if !isPackArtefact(path.Base(f)) {
			continue
		}
		if !strings.HasPrefix(f, "internal/") {
			out = append(out, fmt.Sprintf("%s: a private eval-pack artefact name outside internal/**/testdata/ (never commit pack data; a hand-made fixture goes under internal/**/testdata/ with a SYNTHETIC.md)", f))
			continue
		}
		dir, found := path.Dir(f), ""
		for d := dir; d != "." && d != "/"; d = path.Dir(d) {
			if path.Base(d) == "testdata" {
				found = d
				break
			}
		}
		if found == "" {
			out = append(out, fmt.Sprintf("%s: a private eval-pack artefact name outside a testdata/ tree", f))
			continue
		}
		first, ok := marker(found + "/SYNTHETIC.md")
		if !ok || first != SyntheticMarker {
			out = append(out, fmt.Sprintf("%s: pack artefact in %s/ without its SYNTHETIC.md marker (first line exactly %q)", f, found, SyntheticMarker))
		}
	}
	return out
}

func (l *linter) checkPackArtefacts() {
	top, err := l.git("rev-parse", "--show-toplevel")
	if err != nil {
		l.note("repo-wide pack-artefact pass skipped: not a git checkout (%v)", err)
		return
	}
	top = strings.TrimSpace(top)
	out, err := l.gitIn(top, "ls-files", "-z")
	if err != nil {
		l.fail("artefacts", "git ls-files: %v", err)
		return
	}
	files := strings.Split(strings.TrimRight(out, "\x00"), "\x00")
	sort.Strings(files)
	for _, p := range packArtefactProblems(files, func(p string) (string, bool) {
		b, err := os.ReadFile(filepath.Join(top, filepath.FromSlash(p)))
		if err != nil {
			return "", false
		}
		first, _, _ := strings.Cut(string(b), "\n")
		return strings.TrimRight(first, "\r"), true
	}) {
		l.fail("artefacts", "%s", p)
	}
}

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sujaykumarsuman/xlearn/internal/course"
	"github.com/sujaykumarsuman/xlearn/internal/course/canon"
	"github.com/sujaykumarsuman/xlearn/internal/course/lint"
	"github.com/sujaykumarsuman/xlearn/internal/packspec"
)

// Finding levels.
const (
	levelError = "ERROR"
	levelWarn  = "WARN"
	levelInfo  = "INFO"
)

// finding is one check result. It never carries pack payloads: ids, paths, hashes only.
type finding struct {
	Level string `json:"level"`
	Rule  int    `json:"rule"`
	Item  string `json:"item,omitempty"`
	Path  string `json:"path,omitempty"`
	Msg   string `json:"message"`
}

type checker struct {
	pub    *publicContent
	pack   string
	only   map[string]bool
	since  string
	found  []finding
	packed map[string]bool
}

func (c *checker) add(level string, rule int, item, p, format string, args ...any) {
	c.found = append(c.found, finding{Level: level, Rule: rule, Item: item, Path: p, Msg: fmt.Sprintf(format, args...)})
}

func runCheck(args []string, stdout, stderr io.Writer) int {
	fl := flag.NewFlagSet("packlint check", flag.ContinueOnError)
	fl.SetOutput(stderr)
	public := fl.String("public", ".", "the public repo root, or a content root holding courses/")
	pack := fl.String("pack", defaultPack(), "the private eval pack checkout (default $XLEARN_EVALPACK_DIR or ../xlearn-evalpack)")
	since := fl.String("since", "", "public git ref: warn about label edits (rule 7) and content-only edits (rule 8) since it")
	asJSON := fl.Bool("json", false, "print the findings as JSON")
	strict := fl.Bool("strict", false, "also fail on WARN")
	var items multiFlag
	fl.Var(&items, "item", "check only this item id (repeatable)")
	if err := fl.Parse(args); err != nil {
		return exitUsage
	}
	if fl.NArg() > 0 {
		fmt.Fprintf(stderr, "packlint check: unexpected arguments %v\n", fl.Args())
		return exitUsage
	}
	pub, err := loadPublic(*public)
	if err != nil {
		fmt.Fprintln(stderr, "packlint check:", err)
		return exitUsage
	}
	if fi, err := os.Stat(*pack); err != nil || !fi.IsDir() {
		fmt.Fprintf(stderr, "packlint check: --pack %s is not a directory\n", *pack)
		return exitUsage
	}
	c := &checker{pub: pub, pack: *pack, since: *since, packed: map[string]bool{}}
	if len(items) > 0 {
		c.only = map[string]bool{}
		for _, id := range items {
			c.only[id] = true
		}
	}
	if err := c.run(); err != nil {
		fmt.Fprintln(stderr, "packlint check:", err)
		return exitUsage
	}
	return report(c.found, *asJSON, *strict, stdout)
}

// run applies the nine rules. An error is a usage/environment failure (exit 2).
func (c *checker) run() error {
	// Rule 1: the root pack.json decodes and its format_major is supported.
	b, err := os.ReadFile(filepath.Join(c.pack, packspec.RootFile))
	if err != nil {
		c.add(levelError, 1, "", packspec.RootFile, "cannot read the root pack.json: %v", err)
		return nil
	}
	root, err := packspec.DecodeRoot(b)
	switch {
	case err != nil:
		c.add(levelError, 1, "", packspec.RootFile, "%v", err)
		return nil
	case !root.SupportedFormat():
		c.add(levelError, 1, "", packspec.RootFile, "format_major %d is not supported by this packlint (supports %v)", root.FormatMajor, packspec.FormatMajors)
		return nil
	}
	if err := root.Validate(); err != nil {
		c.add(levelError, 1, "", packspec.RootFile, "%v", err)
	}

	dirs, err := fs.Glob(os.DirFS(c.pack), packspec.ItemsGlob)
	if err != nil {
		return err
	}
	sort.Strings(dirs)
	for _, d := range dirs {
		fi, err := os.Stat(filepath.Join(c.pack, d))
		if err != nil || !fi.IsDir() {
			c.add(levelError, 4, "", d, "not a directory (items/<id>/)")
			continue
		}
		id := path.Base(d)
		if strings.HasPrefix(id, ".") {
			c.add(levelError, 4, "", d, "dotfiles are not allowed in a pack")
			continue
		}
		if c.only != nil && !c.only[id] {
			continue
		}
		c.packed[id] = true
		if err := c.checkItem(d, id); err != nil {
			return err
		}
	}

	// Rule 9: auto-graded public items with no pack yet (self-graded until packed).
	for _, id := range c.pub.ids {
		if c.only != nil && !c.only[id] {
			continue
		}
		pi := c.pub.items[id]
		if !c.packed[id] && lint.HasAutoCodePart(&pi.resolved.Item) && pi.resolved.Item.Status == "live" {
			c.add(levelInfo, 9, id, pi.dir, "has an auto code part and no pack yet (self-graded until packed)")
		}
	}
	return nil
}

func (c *checker) checkItem(dir, id string) error {
	pf := dir + "/" + packspec.ItemFile
	b, err := os.ReadFile(filepath.Join(c.pack, pf))
	if err != nil {
		c.add(levelError, 2, id, pf, "cannot read the item pack.json: %v", err)
		return nil
	}
	pk, err := packspec.DecodeItem(b)
	if err != nil {
		c.add(levelError, 2, id, pf, "%v", err)
		return nil
	}
	if err := pk.Validate(); err != nil {
		for _, line := range strings.Split(err.Error(), "\n") {
			rule := 2
			if strings.HasPrefix(line, "accepts_contract_hashes") {
				rule = 3
			}
			c.add(levelError, rule, id, pf, "%s", line)
		}
	}
	if pk.Item != id {
		c.add(levelError, 2, id, pf, "item %q does not match its directory %q", pk.Item, id)
	}
	slug := strings.Split(dir, "/")[1]
	pi := c.pub.items[id]
	switch {
	case pi == nil:
		c.add(levelError, 2, id, pf, "no public item %q (courses/%s/items/%s/item.json)", id, slug, id)
		c.checkFiles(dir, id, pk)
		return nil
	case pi.course != slug:
		c.add(levelError, 2, id, pf, "public item %q is in course %q, not %q", id, pi.course, slug)
	case pi.resolved.Item.Status != "live":
		c.add(levelWarn, 2, id, pf, "packed item is not live (%s)", pi.resolved.Item.Status)
	}
	it := &pi.resolved.Item

	// Rule 3: the pack accepts the live contract hash.
	live, err := canon.ContractHash(it)
	if err != nil {
		return fmt.Errorf("item %s: %w", id, err)
	}
	switch {
	case live == "":
		c.add(levelError, 3, id, pf, "public item %s is self-path (no graded part, no pack-keyed probe): it has no contract_hash, so nothing reads this pack", id)
	case !containsStr(pk.AcceptsContractHashes, live):
		var acc []string
		for _, h := range pk.AcceptsContractHashes {
			acc = append(acc, "sha256:"+canon.Prefix(h))
		}
		c.add(levelError, 3, id, pf, "stale contract hash for %s: live sha256:%s, pack accepts [%s] (packlint hash --item %s prints the live hash)",
			id, canon.Prefix(live), strings.Join(acc, ", "), id)
	}

	// Rule 4: files on disk <-> declarations.
	c.checkFiles(dir, id, pk)

	// Wrong-solution categories come from the course's mistake taxonomy.
	if m := c.pub.manifests[pi.course]; m != nil {
		cats := map[string]bool{}
		for _, cid := range m.CategoryIDs() {
			cats[cid] = true
		}
		for i, w := range pk.Wrong {
			if w.Category != "" && !cats[w.Category] {
				c.add(levelError, 2, id, pf, "wrong[%d].category %q is not a mistake category of course %q", i, w.Category, pi.course)
			}
		}
	}

	// Rule 5: a stamped full pack for an auto code part declares >= 2 wrong solutions.
	if lint.HasAutoCodePart(it) {
		n := 0
		for _, w := range pk.Wrong {
			if w.Expect != "" {
				n++
			}
		}
		if n < 2 {
			if pk.Stamped() {
				c.add(levelError, 5, id, pf, "review.tests is stamped but only %d wrong solution(s) declare an expect (the full-pack tier needs >= 2)", n)
			} else {
				c.add(levelWarn, 5, id, pf, "only %d wrong solution(s) declare an expect; the full-pack tier needs >= 2 before review.tests is stamped", n)
			}
		}
	}

	// Rule 6: key-graded parts are final and every pack key exists.
	keyed := lint.KeyGradedParts(it)
	need := map[string]string{}
	for _, p := range it.Parts {
		if keyed[p.ID] {
			if p.Cadence != "final" {
				c.add(levelError, 6, id, pf, "part %q is key-graded but cadence %q (key parts are final, t4 §5.6 #4)", p.ID, p.Cadence)
			}
			need[p.ID] = "key-graded part"
		}
	}
	if it.Revision != nil {
		for _, pr := range it.Revision.Probes {
			if pr.KeySource == "pack" {
				need[pr.ID] = "key_source: pack probe"
			}
		}
	}
	for _, kid := range sortedMapKeys(need) {
		if _, ok := pk.Keys[kid]; !ok {
			c.add(levelError, 6, id, pf, "%s %q has no key file (keys[%q])", need[kid], kid, kid)
		}
	}
	for _, kid := range sortedMapKeys(pk.Keys) {
		if _, ok := need[kid]; !ok {
			c.add(levelError, 6, id, pf, "keys[%q] names no key-graded part or pack-keyed probe of item %s", kid, id)
		}
	}

	// Rules 7 and 8: edits since --since (advisory).
	if c.since != "" {
		base, err := c.pub.itemAt(c.since, pi)
		if err != nil {
			return fmt.Errorf("--since %s: %w", c.since, err)
		}
		for _, e := range lint.LabelEdits(base, it) {
			c.add(levelWarn, 7, id, pi.dir+"/"+course.ItemFile, "label edit on key-graded %s/%s since %s: the pack's key may no longer match (%s)", e.Part, e.ID, c.since, e.Token())
		}
		edits := lint.AdvisoryEdits(base, it)
		if changed, err := c.pub.changedSince(c.since, pi, "sections/attempt"); err != nil {
			return fmt.Errorf("--since %s: %w", c.since, err)
		} else if changed {
			edits = append(edits, "sections/attempt (statement)")
		}
		if len(edits) > 0 {
			c.add(levelWarn, 8, id, pi.dir, "content-only edits since %s (%s): re-run `make packcheck ITEM=%s` (time limits and TLE expectations, the validator)",
				c.since, strings.Join(edits, ", "), id)
		}
	}
	return nil
}

// checkFiles is rule 4: every declared file exists; nothing undeclared sits outside the
// allowed subdirectories; no dotfiles.
func (c *checker) checkFiles(dir, id string, pk *packspec.Item) {
	for _, f := range pk.Files() {
		fi, err := os.Stat(filepath.Join(c.pack, dir, filepath.FromSlash(f)))
		if err != nil || !fi.Mode().IsRegular() {
			c.add(levelError, 4, id, dir+"/"+f, "declared in pack.json but missing")
		}
	}
	allowed := map[string]bool{}
	for _, d := range packspec.ItemDirs {
		allowed[d] = true
	}
	root := filepath.Join(c.pack, dir)
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			c.add(levelError, 4, id, dir, "%v", err)
			return nil
		}
		if p == root {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		if strings.HasPrefix(d.Name(), ".") {
			c.add(levelError, 4, id, dir+"/"+rel, "dotfiles are not allowed in a pack")
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		top := strings.Split(rel, "/")[0]
		switch {
		case rel == packspec.ItemFile:
		case !strings.Contains(rel, "/") && d.IsDir() && !allowed[top]:
			c.add(levelError, 4, id, dir+"/"+rel+"/", "undeclared directory (allowed: %s)", strings.Join(packspec.ItemDirs, ", "))
			return fs.SkipDir
		case !strings.Contains(rel, "/") && !d.IsDir():
			c.add(levelError, 4, id, dir+"/"+rel, "undeclared file (only pack.json and the allowed subdirectories)")
		}
		return nil
	})
}

// report prints the findings and returns the exit code.
func report(found []finding, asJSON, strict bool, w io.Writer) int {
	sort.SliceStable(found, func(i, j int) bool {
		if found[i].Item != found[j].Item {
			return lessID(found[i].Item, found[j].Item)
		}
		return found[i].Rule < found[j].Rule
	})
	counts := map[string]int{}
	for _, f := range found {
		counts[f.Level]++
	}
	if asJSON {
		out := struct {
			Findings []finding `json:"findings"`
			Errors   int       `json:"errors"`
			Warnings int       `json:"warnings"`
			Infos    int       `json:"infos"`
		}{found, counts[levelError], counts[levelWarn], counts[levelInfo]}
		if out.Findings == nil {
			out.Findings = []finding{}
		}
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		_ = enc.Encode(out)
	} else {
		for _, f := range found {
			fmt.Fprintf(w, "packlint: %s [rule %d] %s: %s\n", f.Level, f.Rule, f.Path, f.Msg)
		}
		fmt.Fprintf(w, "packlint check: %d error(s), %d warning(s), %d info\n", counts[levelError], counts[levelWarn], counts[levelInfo])
	}
	if counts[levelError] > 0 || (strict && counts[levelWarn] > 0) {
		return exitFail
	}
	return exitOK
}

func containsStr(set []string, v string) bool {
	for _, s := range set {
		if s == v {
			return true
		}
	}
	return false
}

func sortedMapKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

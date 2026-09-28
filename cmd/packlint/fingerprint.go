package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"hash/fnv"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/sujaykumarsuman/xlearn/internal/course/canon"
	"github.com/sujaykumarsuman/xlearn/internal/packspec"
)

// packlint fingerprint — the pre-push leak scan (t1 §7.2, §10): the only check that sees
// both repos BEFORE anything is public (a slip into the public repo is permanent).
//
// Corpus (read locally, held in memory, never written, printed, logged or uploaded):
//   - every value of every hand-written case in the pack (tests/*.jsonl, e.g. edge.jsonl:
//     args, each arg, ctor, ops, expected), every key-file value (keys/), and any
//     materialized case under build/ (plain *.jsonl; m3-02): each whitespace-stripped
//     canonical JSON value of >= --min-len bytes is a payload, with the element list of an
//     array and the text of a string as variants;
//   - anchors/ and exemplars/ files become normalized 8-word shingles.
//
// Scan targets: the added lines of every pushed commit (per commit, against its first
// parent, so an addition later removed inside the push is still caught) and the pushed
// commits' messages. A payload substring, or >= 3 consecutive shingles of one anchor or
// exemplar, is a hit: `BLOCKED: <file>:<line> contains private eval-pack data (matches
// <pack-relative path>)` — never the payload — and exit 1.

const (
	defaultMinLen = 24
	shingleWords  = 8
	shingleRun    = 3
	maxLine       = 64 << 20
)

// corpus is the in-memory fingerprint of the pack.
type corpus struct {
	minLen   int
	payloads map[string]string   // stripped payload -> pack source (path[:line])
	prefixes map[string][]string // payload[:minLen] -> payloads
	shingles map[uint64][]int32  // shingle hash -> source indices
	sources  []string            // shingle sources (pack-relative paths)
}

func newCorpus(minLen int) *corpus {
	return &corpus{minLen: minLen, payloads: map[string]string{}, prefixes: map[string][]string{}, shingles: map[uint64][]int32{}}
}

func (c *corpus) empty() bool { return len(c.payloads) == 0 && len(c.shingles) == 0 }

// addPayload records a stripped payload of at least minLen bytes.
func (c *corpus) addPayload(s, src string) {
	s = stripSpace(s)
	if len(s) < c.minLen {
		return
	}
	if _, ok := c.payloads[s]; ok {
		return
	}
	c.payloads[s] = src
	k := s[:c.minLen]
	c.prefixes[k] = append(c.prefixes[k], s)
}

// addValue records a JSON value (canonical), its array element list and its string text.
func (c *corpus) addValue(raw json.RawMessage, src string) error {
	n, err := canon.NormalizeJSON(raw)
	if err != nil || n == nil {
		return err
	}
	c.addPayload(string(n), src)
	switch n[0] {
	case '[':
		c.addPayload(string(n[1:len(n)-1]), src)
	case '"':
		var s string
		if json.Unmarshal(n, &s) == nil {
			c.addPayload(s, src)
		}
	}
	return nil
}

// addCase records one case line: args (whole and each element), ctor, ops, expected.
func (c *corpus) addCase(line []byte, src string) error {
	var cs map[string]json.RawMessage
	if err := json.Unmarshal(line, &cs); err != nil {
		return fmt.Errorf("%s: not a JSON object: %w", src, err)
	}
	for _, k := range []string{"args", "ctor", "ops", "expected"} {
		v, ok := cs[k]
		if !ok {
			continue
		}
		if err := c.addValue(v, src); err != nil {
			return fmt.Errorf("%s: %s: %w", src, k, err)
		}
		if k == "args" {
			var elems []json.RawMessage
			if json.Unmarshal(v, &elems) == nil {
				for _, e := range elems {
					if err := c.addValue(e, src); err != nil {
						return fmt.Errorf("%s: args: %w", src, err)
					}
				}
			}
		}
	}
	return nil
}

// addKeyFile records a key file: a JSON file's value and its top-level members or
// elements; any other file line by line.
func (c *corpus) addKeyFile(b []byte, src string) error {
	var v json.RawMessage
	if json.Unmarshal(b, &v) == nil {
		if err := c.addValue(v, src); err != nil {
			return fmt.Errorf("%s: %w", src, err)
		}
		var obj map[string]json.RawMessage
		var arr []json.RawMessage
		switch {
		case json.Unmarshal(v, &obj) == nil:
			for _, m := range obj {
				if err := c.addValue(m, src); err != nil {
					return fmt.Errorf("%s: %w", src, err)
				}
			}
		case json.Unmarshal(v, &arr) == nil:
			for _, e := range arr {
				if err := c.addValue(e, src); err != nil {
					return fmt.Errorf("%s: %w", src, err)
				}
			}
		}
		return nil
	}
	for i, line := range strings.Split(string(b), "\n") {
		c.addPayload(line, fmt.Sprintf("%s:%d", src, i+1))
	}
	return nil
}

// addProse records the 8-word shingles of an anchor or exemplar.
func (c *corpus) addProse(b []byte, src string) {
	words := normWords(string(b))
	if len(words) < shingleWords {
		return
	}
	idx := int32(len(c.sources))
	c.sources = append(c.sources, src)
	for i := 0; i+shingleWords <= len(words); i++ {
		h := shingleHash(words[i : i+shingleWords])
		list := c.shingles[h]
		if len(list) == 0 || list[len(list)-1] != idx {
			c.shingles[h] = append(list, idx)
		}
	}
}

// buildCorpus reads the pack. It reads only; nothing is written anywhere.
func buildCorpus(pack string, minLen int) (*corpus, error) {
	c := newCorpus(minLen)
	dirs, err := fs.Glob(os.DirFS(pack), packspec.ItemsGlob)
	if err != nil {
		return nil, err
	}
	sort.Strings(dirs)
	for _, d := range dirs {
		root := filepath.Join(pack, filepath.FromSlash(d))
		err := filepath.WalkDir(root, func(p string, e fs.DirEntry, err error) error {
			if err != nil || e.IsDir() {
				return err
			}
			rel, _ := filepath.Rel(pack, p)
			rel = filepath.ToSlash(rel)
			sub := strings.TrimPrefix(rel, d+"/")
			top, _, _ := strings.Cut(sub, "/")
			switch {
			case top == "tests" && strings.HasSuffix(sub, ".jsonl"):
				return c.addJSONL(p, rel)
			case top == "keys":
				b, err := os.ReadFile(p)
				if err != nil {
					return err
				}
				return c.addKeyFile(b, rel)
			case top == "anchors" || top == "exemplars":
				b, err := os.ReadFile(p)
				if err != nil {
					return err
				}
				c.addProse(b, rel)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	// Materialized cases (m3-02's build/); compressed ones are left to the private CI.
	build := filepath.Join(pack, "build")
	if fi, err := os.Stat(build); err == nil && fi.IsDir() {
		err := filepath.WalkDir(build, func(p string, e fs.DirEntry, err error) error {
			if err != nil || e.IsDir() || !strings.HasSuffix(p, ".jsonl") {
				return err
			}
			rel, _ := filepath.Rel(pack, p)
			return c.addJSONL(p, filepath.ToSlash(rel))
		})
		if err != nil {
			return nil, err
		}
	}
	return c, nil
}

func (c *corpus) addJSONL(p, rel string) error {
	f, err := os.Open(p)
	if err != nil {
		return err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1<<20), maxLine)
	for n := 1; sc.Scan(); n++ {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		if err := c.addCase(line, fmt.Sprintf("%s:%d", rel, n)); err != nil {
			return err
		}
	}
	return sc.Err()
}

// --- scan ---------------------------------------------------------------------------

// target is one scanned text: a file's added lines, or a commit message.
type target struct {
	name  string // "path" or "commit <sha> message"
	lines []textLine
}

type textLine struct {
	n    int
	text string
}

type hit struct {
	where string
	line  int
	src   string
}

// scan returns the target's hits (deduplicated).
func (c *corpus) scan(t target) []hit {
	seen := map[hit]bool{}
	var out []hit
	add := func(h hit) {
		if !seen[h] {
			seen[h] = true
			out = append(out, h)
		}
	}
	// Payloads: over the whitespace-stripped concatenation, so a value split across
	// lines still matches.
	if len(c.payloads) > 0 {
		var buf []byte
		var lineOf []int
		for _, l := range t.lines {
			s := stripSpace(l.text)
			buf = append(buf, s...)
			for range len(s) {
				lineOf = append(lineOf, l.n)
			}
		}
		for i := 0; i+c.minLen <= len(buf); i++ {
			cands, ok := c.prefixes[string(buf[i:i+c.minLen])]
			if !ok {
				continue
			}
			for _, p := range cands {
				if bytes.HasPrefix(buf[i:], []byte(p)) {
					add(hit{where: t.name, line: lineOf[i], src: c.payloads[p]})
				}
			}
		}
	}
	// Shingles: >= shingleRun consecutive windows found in one source.
	if len(c.shingles) > 0 {
		type word struct {
			w string
			n int
		}
		var words []word
		for _, l := range t.lines {
			for _, w := range normWords(l.text) {
				words = append(words, word{w, l.n})
			}
		}
		run := map[int32]int{}
		ws := make([]string, shingleWords)
		for i := 0; i+shingleWords <= len(words); i++ {
			for j := range ws {
				ws[j] = words[i+j].w
			}
			next := map[int32]int{}
			for _, src := range c.shingles[shingleHash(ws)] {
				next[src] = run[src] + 1
				if next[src] >= shingleRun {
					add(hit{where: t.name, line: words[i-shingleRun+1].n, src: c.sources[src]})
				}
			}
			run = next
		}
	}
	return out
}

// --- git targets --------------------------------------------------------------------

var hunkRe = regexp.MustCompile(`^@@ -[0-9,]+ \+([0-9]+)(?:,[0-9]+)? @@`)

// parseDiff turns a unified diff (-U0) into one target per file with its added lines.
func parseDiff(diff string) []target {
	var out []target
	var cur *target
	n := 0
	for _, line := range strings.Split(diff, "\n") {
		switch {
		case strings.HasPrefix(line, "diff --git "):
			cur = nil
		case strings.HasPrefix(line, "+++ "):
			name := strings.TrimPrefix(line, "+++ ")
			if name == "/dev/null" {
				cur = nil
				continue
			}
			if uq, err := strconv.Unquote(name); err == nil {
				name = uq
			}
			name = strings.TrimPrefix(name, "b/")
			out = append(out, target{name: name})
			cur = &out[len(out)-1]
		case strings.HasPrefix(line, "@@"):
			if m := hunkRe.FindStringSubmatch(line); m != nil {
				n, _ = strconv.Atoi(m[1])
			}
		case strings.HasPrefix(line, "+") && cur != nil:
			cur.lines = append(cur.lines, textLine{n: n, text: line[1:]})
			n++
		}
	}
	return out
}

// commitTargets returns every commit's added lines and message in a rev-list range.
func commitTargets(repo string, revs []string) ([]target, error) {
	out, err := gitOut(repo, append([]string{"rev-list"}, revs...)...)
	if err != nil {
		return nil, err
	}
	var targets []target
	for _, sha := range strings.Fields(out) {
		msg, err := gitOut(repo, "log", "-1", "--format=%B", sha)
		if err != nil {
			return nil, err
		}
		mt := target{name: "commit " + short(sha) + " message"}
		for i, l := range strings.Split(msg, "\n") {
			mt.lines = append(mt.lines, textLine{n: i + 1, text: l})
		}
		targets = append(targets, mt)

		parents, err := gitOut(repo, "rev-list", "--parents", "-n", "1", sha)
		if err != nil {
			return nil, err
		}
		var diff string
		if ps := strings.Fields(parents); len(ps) > 1 {
			diff, err = gitOut(repo, "-c", "core.quotePath=false", "diff", "-U0", "--no-color", "--no-ext-diff", "--no-renames", ps[1], sha)
		} else {
			diff, err = gitOut(repo, "-c", "core.quotePath=false", "diff-tree", "-p", "-U0", "--no-color", "--no-ext-diff", "--no-renames", "--root", sha)
		}
		if err != nil {
			return nil, err
		}
		targets = append(targets, parseDiff(diff)...)
	}
	return targets, nil
}

// prePushRevs maps the hook's stdin (`<local ref> <local sha> <remote ref> <remote sha>`)
// to rev-list ranges: an existing remote ref pushes remote..local; a new ref pushes what
// is not on origin/main (merge-base), else not on any remote-tracking ref; a deleted ref
// pushes nothing.
func prePushRevs(repo string, stdin io.Reader) ([][]string, error) {
	var out [][]string
	sc := bufio.NewScanner(stdin)
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		if len(f) == 0 {
			continue
		}
		if len(f) != 4 {
			return nil, fmt.Errorf("pre-push stdin: want <local ref> <local sha> <remote ref> <remote sha>, got %q", sc.Text())
		}
		local, remote := f[1], f[3]
		switch {
		case isZero(local):
			continue // a deleted ref
		case !isZero(remote) && objectExists(repo, remote):
			out = append(out, []string{remote + ".." + local})
		default:
			if mb, err := gitOut(repo, "merge-base", local, "origin/main"); err == nil && strings.TrimSpace(mb) != "" {
				out = append(out, []string{strings.TrimSpace(mb) + ".." + local})
			} else {
				out = append(out, []string{local, "--not", "--remotes"})
			}
		}
	}
	return out, sc.Err()
}

func isZero(sha string) bool { return strings.Trim(sha, "0") == "" }

func objectExists(repo, sha string) bool {
	_, err := gitOut(repo, "cat-file", "-e", sha+"^{commit}")
	return err == nil
}

func short(sha string) string {
	if len(sha) > 12 {
		return sha[:12]
	}
	return sha
}

// treeTargets returns every text file under dir as a target (m3-02's private-CI backstop
// over a public checkout).
func treeTargets(dir string) ([]target, error) {
	var out []target
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if p != dir && (d.Name() == ".git" || d.Name() == "node_modules") {
				return fs.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if bytes.IndexByte(b[:min(len(b), 8000)], 0) >= 0 {
			return nil // binary
		}
		rel, _ := filepath.Rel(dir, p)
		t := target{name: filepath.ToSlash(rel)}
		for i, l := range strings.Split(string(b), "\n") {
			t.lines = append(t.lines, textLine{n: i + 1, text: l})
		}
		out = append(out, t)
		return nil
	})
	return out, err
}

// --- command ------------------------------------------------------------------------

func runFingerprint(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fl := flag.NewFlagSet("packlint fingerprint", flag.ContinueOnError)
	fl.SetOutput(stderr)
	pack := fl.String("pack", defaultPack(), "the private eval pack checkout")
	repo := fl.String("repo", ".", "the public git checkout (for --pre-push and --diff)")
	prePush := fl.Bool("pre-push", false, "scan what a push sends (the pre-push hook's stdin)")
	diffRange := fl.String("diff", "", "scan every commit in a rev-list range, e.g. origin/main..HEAD")
	tree := fl.String("tree", "", "scan every text file under a directory (a public checkout)")
	minLen := fl.Int("min-len", defaultMinLen, "the shortest payload, in bytes after whitespace stripping")
	if err := fl.Parse(args); err != nil {
		return exitUsage
	}
	modes := 0
	for _, on := range []bool{*prePush, *diffRange != "", *tree != ""} {
		if on {
			modes++
		}
	}
	if modes != 1 || fl.NArg() > 0 || *minLen < 8 {
		fmt.Fprintln(stderr, "packlint fingerprint: pick exactly one of --pre-push, --diff <range>, --tree <dir> (and --min-len >= 8)")
		return exitUsage
	}
	if fi, err := os.Stat(*pack); err != nil || !fi.IsDir() {
		fmt.Fprintf(stderr, "packlint fingerprint: --pack %s is not a directory\n", *pack)
		return exitUsage
	}
	c, err := buildCorpus(*pack, *minLen)
	if err != nil {
		fmt.Fprintln(stderr, "packlint fingerprint: reading the pack:", err)
		return exitUsage
	}

	var targets []target
	switch {
	case *tree != "":
		targets, err = treeTargets(*tree)
	case *diffRange != "":
		targets, err = commitTargets(*repo, []string{*diffRange})
	default:
		var ranges [][]string
		ranges, err = prePushRevs(*repo, stdin)
		for _, r := range ranges {
			if err != nil {
				break
			}
			var ts []target
			ts, err = commitTargets(*repo, r)
			targets = append(targets, ts...)
		}
	}
	if err != nil {
		fmt.Fprintln(stderr, "packlint fingerprint:", err)
		return exitUsage
	}

	var hits []hit
	seen := map[hit]bool{}
	for _, t := range targets {
		for _, h := range c.scan(t) {
			if !seen[h] {
				seen[h] = true
				hits = append(hits, h)
			}
		}
	}
	if len(hits) == 0 {
		fmt.Fprintf(stdout, "packlint fingerprint: clean (%d targets scanned)\n", len(targets))
		return exitOK
	}
	for _, h := range hits {
		fmt.Fprintf(stdout, "BLOCKED: %s:%d contains private eval-pack data (matches %s)\n", h.where, h.line, h.src)
	}
	fmt.Fprintf(stdout, `packlint fingerprint: %d hit(s); nothing was pushed.
  Remove the data from every outgoing commit (amend or rewrite the local branch), then push again.
  If any of it already reached a remote, treat it as disclosed and author new cases or keys
  (leak runbook: docs/v2/research/t1-content-data-model.md §10).
  Never bypass this with git push --no-verify without the owner's explicit decision.
`, len(hits))
	return exitFail
}

// --- normalization ------------------------------------------------------------------

// stripSpace removes every whitespace character.
func stripSpace(s string) string {
	if !strings.ContainsFunc(s, unicode.IsSpace) {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if !unicode.IsSpace(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// normWords lowercases and splits text into letter/digit words.
func normWords(s string) []string {
	var out []string
	start := -1
	for i, r := range s {
		ok := unicode.IsLetter(r) || unicode.IsDigit(r)
		switch {
		case ok && start < 0:
			start = i
		case !ok && start >= 0:
			out = append(out, strings.ToLower(s[start:i]))
			start = -1
		}
	}
	if start >= 0 && utf8.ValidString(s[start:]) {
		out = append(out, strings.ToLower(s[start:]))
	}
	return out
}

func shingleHash(words []string) uint64 {
	h := fnv.New64a()
	for i, w := range words {
		if i > 0 {
			h.Write([]byte{' '})
		}
		h.Write([]byte(w))
	}
	return h.Sum64()
}

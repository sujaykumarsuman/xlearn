package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"

	"github.com/sujaykumarsuman/xlearn/internal/course"
	"github.com/sujaykumarsuman/xlearn/internal/curriculum"
)

// linter runs the checks over one curriculum/ directory and collects every problem.
type linter struct {
	root    string // the curriculum/ directory (the embed package)
	prevTag string // "" = the latest v* tag reachable from HEAD
	out     io.Writer
	// base is the PR base the stamp gate and the label-edit flag diff against ("" = the
	// merge base with origin/main); prBody carries the label-edit-ok confirmations.
	base   string
	prBody string
	// reportUnstamped lists the grandfathered unstamped hint/editorial sections.
	reportUnstamped bool

	problems []string
	notes    []string
	stats    map[string]int
}

func (l *linter) fail(check, format string, args ...any) {
	l.problems = append(l.problems, fmt.Sprintf("%s: %s", check, fmt.Sprintf(format, args...)))
}

func (l *linter) note(format string, args ...any) {
	l.notes = append(l.notes, fmt.Sprintf(format, args...))
}

func (l *linter) count(what string) { l.stats[what]++ }

// run runs every check, prints the result and reports success.
func (l *linter) run() bool {
	l.stats = map[string]int{}
	fsys := os.DirFS(l.root)
	l.checkSchemas(fsys)
	content := l.checkLoad(fsys)
	l.checkLock(content)
	l.checkStructure(content)
	l.checkDiffGates(content)
	l.checkMarkdown(fsys)
	l.checkFilenames(fsys)
	l.checkPackArtefacts()
	l.checkEmbed()

	if list := unstamped(content); len(list) > 0 {
		if l.reportUnstamped {
			for _, u := range list {
				fmt.Fprintln(l.out, "contentlint: unstamped:", u)
			}
		}
		l.note("stamp gate: %d item(s) carry unstamped hint/editorial sections, grandfathered until they change (-report-unstamped lists them)", len(list))
	}
	for _, n := range l.notes {
		fmt.Fprintln(l.out, "contentlint: note:", n)
	}
	for _, p := range l.problems {
		fmt.Fprintln(l.out, "contentlint: FAIL", p)
	}
	if len(l.problems) > 0 {
		fmt.Fprintf(l.out, "contentlint: %d problem(s)\n", len(l.problems))
		return false
	}
	fmt.Fprintf(l.out, "contentlint: ok (%d JSON files, %d Markdown files, %d items, %d embedded files)\n",
		l.stats["json"], l.stats["markdown"], l.stats["items"], l.stats["embedded"])
	return true
}

// --- 1. schema + strict decode --------------------------------------------------------

// schemaFor maps each JSON file (by its path in the embed package) to its schema
// (_schema/<name>.schema.json). A JSON file matching none of them is an error.
var schemaFor = []struct{ glob, schema string }{
	{"paths.json", "paths"},
	{"ids.lock.json", "ids-lock"},
	{"courses/*/course.json", "course"},
	{"courses/*/phases.json", "phases"},
	{"courses/*/weeks.json", "weeks"},
	{"courses/*/concepts.json", "concepts"},
	{"courses/*/items/*/item.json", "item"},
}

const schemaDir = "_schema"

func (l *linter) checkSchemas(fsys fs.FS) {
	schemas, err := compileSchemas(fsys)
	if err != nil {
		l.fail("schema", "%v", err)
		return
	}
	_ = fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			l.fail("schema", "%s: %v", p, err)
			return nil
		}
		if d.IsDir() || !strings.HasSuffix(p, ".json") || strings.HasPrefix(p, schemaDir+"/") {
			return nil
		}
		l.count("json")
		name := ""
		for _, sf := range schemaFor {
			if ok, _ := path.Match(sf.glob, p); ok {
				name = sf.schema
				break
			}
		}
		if name == "" {
			l.fail("schema", "%s: no schema covers this JSON file (see curriculum/README.md)", p)
			return nil
		}
		sch := schemas[name]
		if sch == nil {
			l.fail("schema", "%s: schema %s/%s.schema.json is missing", p, schemaDir, name)
			return nil
		}
		b, err := fs.ReadFile(fsys, p)
		if err != nil {
			l.fail("schema", "%s: %v", p, err)
			return nil
		}
		inst, err := jsonschema.UnmarshalJSON(bytes.NewReader(b))
		if err != nil {
			l.fail("schema", "%s: %v", p, err)
			return nil
		}
		if err := sch.Validate(inst); err != nil {
			l.fail("schema", "%s: %s", p, oneLine(err.Error()))
		}
		return nil
	})
}

// compileSchemas compiles every _schema/<name>.schema.json (validating each against the
// 2020-12 meta-schema) and returns them by name.
func compileSchemas(fsys fs.FS) (map[string]*jsonschema.Schema, error) {
	names, err := fs.Glob(fsys, schemaDir+"/*.schema.json")
	if err != nil || len(names) == 0 {
		return nil, fmt.Errorf("no schemas under %s/ (%v)", schemaDir, err)
	}
	c := jsonschema.NewCompiler()
	c.AssertFormat()
	ids := map[string]string{}
	for _, n := range names {
		b, err := fs.ReadFile(fsys, n)
		if err != nil {
			return nil, err
		}
		doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(b))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", n, err)
		}
		m, _ := doc.(map[string]any)
		id, _ := m["$id"].(string)
		if id == "" {
			return nil, fmt.Errorf("%s: no $id", n)
		}
		if err := c.AddResource(id, doc); err != nil {
			return nil, fmt.Errorf("%s: %w", n, err)
		}
		ids[strings.TrimSuffix(path.Base(n), ".schema.json")] = id
	}
	out := map[string]*jsonschema.Schema{}
	for name, id := range ids {
		sch, err := c.Compile(id)
		if err != nil {
			return nil, fmt.Errorf("%s/%s.schema.json: %w", schemaDir, name, err)
		}
		out[name] = sch
	}
	return out, nil
}

// checkLoad runs the service's own loader: strict decoding into the Go types, item
// validation, the id and course-slug guards, the sidecar layout.
func (l *linter) checkLoad(fsys fs.FS) *curriculum.Content {
	content, err := curriculum.LoadContent(fsys)
	if err != nil {
		for _, line := range strings.Split(err.Error(), "\n") {
			l.fail("load", "%s", line)
		}
		return nil
	}
	for _, cc := range content.Courses {
		l.stats["items"] += len(cc.Items)
	}
	return content
}

// --- 2. id / slug guards vs the lock and the previous tag ----------------------------

func (l *linter) checkLock(content *curriculum.Content) {
	if content == nil {
		return // the loader already failed; its errors are reported
	}
	items := map[string]course.LockEntry{}
	for _, cc := range content.Courses {
		for _, it := range cc.Items {
			items[it.Item.ID] = course.LockEntry{Course: cc.Slug, Status: it.Item.Status}
		}
	}
	for _, id := range sortedKeys(content.Lock.Items) {
		le := content.Lock.Items[id]
		it, ok := items[id]
		switch {
		case !ok:
			l.fail("ids", "%s: id %q (course %q) has no item directory: an id never disappears (retire it instead)", course.LockFile, id, le.Course)
		case it.Status != le.Status:
			l.fail("ids", "%s: id %q is %q in the lock but %q in its item.json", course.LockFile, id, le.Status, it.Status)
		}
	}

	prev, tag, err := l.previousLock()
	switch {
	case err != nil:
		l.fail("ids", "previous release lock: %v", err)
	case prev == nil:
		l.note("id guard vs the previous release skipped: %s", tag)
	default:
		l.compareLocks(tag, prev, &content.Lock)
	}
}

// compareLocks enforces append-only against the previous release's lock.
func (l *linter) compareLocks(tag string, prev, cur *course.IDsLock) {
	for _, id := range sortedKeys(prev.Items) {
		was := prev.Items[id]
		now, ok := cur.Items[id]
		switch {
		case !ok:
			l.fail("ids", "id %q (course %q at %s) disappeared from %s", id, was.Course, tag, course.LockFile)
		case now.Course != was.Course:
			l.fail("ids", "id %q moved from course %q (at %s) to %q: ids are never re-parented", id, was.Course, tag, now.Course)
		}
	}
	for _, ref := range sortedKeys(prev.Assets) {
		now, ok := cur.Assets[ref]
		switch {
		case !ok:
			l.fail("ids", "asset %q (at %s) disappeared from %s", ref, tag, course.LockFile)
		case now != prev.Assets[ref]:
			l.fail("ids", "asset %q changed bytes since %s (%s -> %s): publish a new @v instead", ref, tag, prev.Assets[ref], now)
		}
	}
}

// previousLock reads ids.lock.json at the previous release tag. It returns (nil, why,
// nil) when there is nothing to compare yet (no tag, or the tag predates the lock).
func (l *linter) previousLock() (*course.IDsLock, string, error) {
	tag := l.prevTag
	if tag == "" {
		out, err := l.git("describe", "--tags", "--abbrev=0", "--match", "v[0-9]*")
		if err != nil {
			return nil, "no release tag reachable from HEAD (a shallow clone needs fetch-depth: 0)", nil
		}
		tag = strings.TrimSpace(out)
	}
	prefix, err := l.git("rev-parse", "--show-prefix")
	if err != nil {
		return nil, "", fmt.Errorf("git rev-parse: %w", err)
	}
	spec := tag + ":" + strings.TrimSpace(prefix) + course.LockFile
	out, err := l.git("show", spec)
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") || strings.Contains(err.Error(), "exists on disk, but not in") {
			return nil, fmt.Sprintf("%s has no %s yet", tag, course.LockFile), nil
		}
		return nil, "", fmt.Errorf("git show %s: %w", spec, err)
	}
	var lock course.IDsLock
	if err := json.Unmarshal([]byte(out), &lock); err != nil {
		return nil, "", fmt.Errorf("%s: %w", spec, err)
	}
	return &lock, tag, nil
}

func (l *linter) git(args ...string) (string, error) { return l.gitIn(l.root, args...) }

func (l *linter) gitIn(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return string(out), nil
}

// --- 3. Markdown profile + SVG lint --------------------------------------------------

var markdown = goldmark.New(goldmark.WithExtensions(extension.GFM))

func (l *linter) checkMarkdown(fsys fs.FS) {
	_ = fs.WalkDir(fsys, "courses", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				l.fail("markdown", "%s: %v", p, err)
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		switch {
		case strings.HasSuffix(p, ".md"):
			l.count("markdown")
			b, err := fs.ReadFile(fsys, p)
			if err != nil {
				l.fail("markdown", "%s: %v", p, err)
				return nil
			}
			for _, msg := range markdownProblems(b) {
				l.fail("markdown", "%s: %s", p, msg)
			}
		case strings.HasSuffix(strings.ToLower(p), ".svg"):
			b, err := fs.ReadFile(fsys, p)
			if err != nil {
				l.fail("svg", "%s: %v", p, err)
				return nil
			}
			for _, msg := range svgProblems(b) {
				l.fail("svg", "%s: %s", p, msg)
			}
		}
		return nil
	})
}

// markdownProblems applies the public Markdown profile: CommonMark + GFM (tables, fenced
// code), no raw HTML, images only as asset: refs, links https:// only.
func markdownProblems(src []byte) []string {
	var out []string
	doc := markdown.Parser().Parse(text.NewReader(src))
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch v := n.(type) {
		case *ast.HTMLBlock:
			out = append(out, fmt.Sprintf("raw HTML block %q is not allowed", snippet(blockText(v, src))))
		case *ast.RawHTML:
			out = append(out, fmt.Sprintf("raw HTML %q is not allowed", snippet(rawHTMLText(v, src))))
		case *ast.Image:
			if !strings.HasPrefix(string(v.Destination), "asset:") {
				out = append(out, fmt.Sprintf("image %q must be an asset: ref", v.Destination))
			}
		case *ast.Link:
			if !strings.HasPrefix(string(v.Destination), "https://") {
				out = append(out, fmt.Sprintf("link %q must be https://", v.Destination))
			}
		case *ast.AutoLink:
			if u := string(v.URL(src)); !strings.HasPrefix(u, "https://") {
				out = append(out, fmt.Sprintf("autolink %q must be https://", u))
			}
		}
		return ast.WalkContinue, nil
	})
	return out
}

func blockText(n *ast.HTMLBlock, src []byte) string {
	var b strings.Builder
	for i := 0; i < n.Lines().Len(); i++ {
		seg := n.Lines().At(i)
		b.Write(seg.Value(src))
	}
	return b.String()
}

func rawHTMLText(n *ast.RawHTML, src []byte) string {
	var b strings.Builder
	for i := 0; i < n.Segments.Len(); i++ {
		seg := n.Segments.At(i)
		b.Write(seg.Value(src))
	}
	return b.String()
}

func snippet(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 60 {
		return s[:60] + "…"
	}
	return s
}

var (
	svgScriptRe   = regexp.MustCompile(`(?i)<\s*script`)
	svgHandlerRe  = regexp.MustCompile(`(?i)\son[a-z]+\s*=`)
	svgExternalRe = regexp.MustCompile(`(?i)(href\s*=\s*["']?\s*(https?:|//|data:|javascript:)|url\(\s*["']?\s*(https?:|//))`)
)

// svgProblems is the SVG lint: no scripts, no event handlers, no external references.
func svgProblems(b []byte) []string {
	var out []string
	if svgScriptRe.Match(b) {
		out = append(out, "contains <script>")
	}
	if svgHandlerRe.Match(b) {
		out = append(out, "contains an on*= event handler")
	}
	if svgExternalRe.Match(b) {
		out = append(out, "references an external resource")
	}
	return out
}

// --- 4. filename rules ---------------------------------------------------------------

// deniedNames look like private eval-pack material (ADR-0027 §1, t1 §3.3); matched on
// every path segment, case-insensitively. Public keys/*.json alias tables stay allowed.
var (
	deniedFileGlobs = []string{
		"*.ans", "hidden*", "secret*", "expected*", "anchors*", "exemplar*", "calibration*",
		// m3-01: the pack's own file names
		"pack.json", "tests.lock", "cases.jsonl*", "*.jsonl.zst", "edge.jsonl", "gen-hidden*",
		"instances.json", "timing.json",
	}
	deniedDirs = []string{"submissions", "wrong", "invalid"}
)

func (l *linter) checkFilenames(fsys fs.FS) {
	_ = fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || p == "." {
			return nil
		}
		name := strings.ToLower(d.Name())
		for _, g := range deniedFileGlobs {
			if ok, _ := path.Match(g, name); ok {
				l.fail("filename", "%s: matches the private-name denylist %q", p, g)
			}
		}
		if d.IsDir() {
			for _, dn := range deniedDirs {
				if name == dn {
					l.fail("filename", "%s/: a %s/ directory is private-only", p, dn)
				}
			}
			return nil
		}
		if strings.HasSuffix(name, ".go") {
			b, err := fs.ReadFile(fsys, p)
			if err != nil {
				l.fail("filename", "%s: %v", p, err)
				return nil
			}
			if msg := goFileProblem(p, b); msg != "" {
				l.fail("filename", "%s: %s", p, msg)
			}
		}
		return nil
	})
}

// goFileProblem: every .go file under curriculum/ is a complete, gofmt-clean Go file.
// A display fragment (no package clause) belongs in *.snip.
func goFileProblem(name string, src []byte) string {
	if _, err := parser.ParseFile(token.NewFileSet(), name, src, parser.ParseComments); err != nil {
		return fmt.Sprintf("not a complete Go file (%s); save a code fragment as *.<lang>.snip", oneLine(err.Error()))
	}
	formatted, err := format.Source(src)
	if err != nil || !bytes.Equal(formatted, src) {
		return "not gofmt-clean (run gofmt -w)"
	}
	return ""
}

// --- 5. embedded-file allowlist ------------------------------------------------------

const allowlistFile = "embed.allowlist"

func (l *linter) allowlistPath() string { return filepath.Join(l.root, allowlistFile) }

// embedFiles lists the files the curriculum package embeds (go list's EmbedFiles).
func (l *linter) embedFiles() ([]string, error) {
	cmd := exec.Command("go", "list", "-json", ".")
	cmd.Dir = l.root
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("go list -json %s: %w: %s", l.root, err, strings.TrimSpace(stderr.String()))
	}
	var pkg struct{ EmbedFiles []string }
	if err := json.Unmarshal(out, &pkg); err != nil {
		return nil, fmt.Errorf("go list -json %s: %w", l.root, err)
	}
	sort.Strings(pkg.EmbedFiles)
	return pkg.EmbedFiles, nil
}

// embedPkg is one module package that embeds files.
type embedPkg struct {
	ImportPath string
	Dir        string
	EmbedFiles []string
}

// embedPackages lists every package of the module with a //go:embed (go list -json ./...
// -> EmbedFiles), not only ./curriculum (m3-01).
func (l *linter) embedPackages() ([]embedPkg, error) {
	// The module is the curriculum root's, not the caller's directory's.
	gomod, err := exec.Command("go", "-C", l.root, "env", "GOMOD").Output()
	if err != nil {
		return nil, fmt.Errorf("go env GOMOD: %w", err)
	}
	cmd := exec.Command("go", "list", "-json", "./...")
	cmd.Dir = filepath.Dir(strings.TrimSpace(string(gomod)))
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("go list -json ./...: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	var pkgs []embedPkg
	dec := json.NewDecoder(bytes.NewReader(out))
	for dec.More() {
		var p embedPkg
		if err := dec.Decode(&p); err != nil {
			return nil, fmt.Errorf("go list -json ./...: %w", err)
		}
		if len(p.EmbedFiles) > 0 {
			sort.Strings(p.EmbedFiles)
			pkgs = append(pkgs, p)
		}
	}
	return pkgs, nil
}

// checkEmbed: every package with a //go:embed has an embed.allowlist beside it, and
// every embedded file matches one of its lines (path.Match patterns), so nothing
// unexpected — a dotfile, a stray, anything private — ever ships in an image.
// curriculum/embed.allowlist is the exact, generated list and is also checked the other
// way (every line is embedded).
func (l *linter) checkEmbed() {
	pkgs, err := l.embedPackages()
	if err != nil {
		l.fail("embed", "%v", err)
		return
	}
	curDir := realDir(l.root)
	seenCurriculum := false
	for _, p := range pkgs {
		exact := realDir(p.Dir) == curDir
		allowFile := filepath.Join(p.Dir, allowlistFile)
		allowed, err := readAllowlist(allowFile)
		if err != nil {
			l.fail("embed", "%s embeds %d file(s) but has no %s beside it (list the allowed path.Match patterns, one per line): %v", p.ImportPath, len(p.EmbedFiles), allowlistFile, err)
			continue
		}
		if exact {
			seenCurriculum = true
			l.stats["embedded"] = len(p.EmbedFiles)
			extra, missing := diffSets(p.EmbedFiles, allowed)
			for _, f := range extra {
				l.fail("embed", "%s is embedded but not in %s (a dotfile or stray? if intended: go run ./cmd/contentlint -write-allowlist)", f, allowlistFile)
			}
			for _, f := range missing {
				l.fail("embed", "%s is in %s but not embedded (refresh it: go run ./cmd/contentlint -write-allowlist)", f, allowlistFile)
			}
			continue
		}
		for _, f := range p.EmbedFiles {
			if !matchesAny(allowed, f) {
				l.fail("embed", "%s: %s is embedded but matches no line of %s", p.ImportPath, f, allowFile)
			}
		}
	}
	if !seenCurriculum {
		l.fail("embed", "the curriculum package (%s) embeds nothing", l.root)
	}
}

// realDir is a directory's absolute, symlink-free path (best effort).
func realDir(d string) string {
	if abs, err := filepath.Abs(d); err == nil {
		d = abs
	}
	if r, err := filepath.EvalSymlinks(d); err == nil {
		d = r
	}
	return d
}

func matchesAny(patterns []string, f string) bool {
	for _, pat := range patterns {
		if ok, _ := path.Match(pat, f); ok {
			return true
		}
	}
	return false
}

const allowlistHeader = `# The exact list of files embedded by curriculum/embed.go (go list -json ./curriculum
# -> EmbedFiles). cmd/contentlint fails CI on any difference, so nothing unexpected (a
# dotfile, a stray, anything private) ever ships in the curriculum image. After adding
# or removing content, regenerate with: go run ./cmd/contentlint -write-allowlist
# and review the diff.
`

func (l *linter) writeAllowlist() (int, error) {
	embedded, err := l.embedFiles()
	if err != nil {
		return 0, err
	}
	var b strings.Builder
	b.WriteString(allowlistHeader)
	for _, f := range embedded {
		b.WriteString(f)
		b.WriteByte('\n')
	}
	return len(embedded), os.WriteFile(l.allowlistPath(), []byte(b.String()), 0o644)
}

func readAllowlist(name string) ([]string, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", name, err)
	}
	defer f.Close()
	var out []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, line)
	}
	return out, sc.Err()
}

// diffSets returns the elements only in a (extra) and only in b (missing).
func diffSets(a, b []string) (extra, missing []string) {
	inA, inB := map[string]bool{}, map[string]bool{}
	for _, s := range a {
		inA[s] = true
	}
	for _, s := range b {
		inB[s] = true
		if !inA[s] {
			missing = append(missing, s)
		}
	}
	for _, s := range a {
		if !inB[s] {
			extra = append(extra, s)
		}
	}
	return extra, missing
}

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func oneLine(s string) string { return strings.Join(strings.Fields(s), " ") }

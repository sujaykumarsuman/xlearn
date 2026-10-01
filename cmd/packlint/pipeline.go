package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sujaykumarsuman/xlearn/internal/packspec"
	"github.com/sujaykumarsuman/xlearn/internal/packspec/executor"
)

// The m3-02 pipeline subcommands (t1 §7.2 gate table):
//
//	packlint lock      --public . --pack … [--item <id>]… (--write | --verify)
//	packlint lock      --tools
//	packlint validate  --public . --pack … [--item <id>]…
//	packlint exec      --public . --pack … [--item <id>]… --gate oracle|wrong|tl|syntax|all [--provisional]
//	packlint exec      --warm-cache [--cache-dir <dir>]
//	packlint build     --public . --pack … --out build/ --version <semver> --validated-against <sha> [--dockerfile=false]
//	packlint listing   <image ref | image archive .tar | build directory>
//
// Every program runs through the executor (docker, --network none). Output carries ids,
// paths, groups and counts only.

// newExecutor is the executor factory (tests swap it for a fake).
var newExecutor = func(stderr io.Writer, verbose bool) packspec.Executor {
	return &executor.Docker{Stderr: stderr, Verbose: verbose}
}

// pipeFlags are the flags every pipeline command shares.
type pipeFlags struct {
	public, pack string
	items        multiFlag
	verbose      bool
}

func (p *pipeFlags) register(fl *flag.FlagSet) {
	fl.StringVar(&p.public, "public", ".", "the public repo root, or a content root holding courses/")
	fl.StringVar(&p.pack, "pack", defaultPack(), "the eval pack source (default $XLEARN_EVALPACK_DIR or ../xlearn-evalpack)")
	fl.Var(&p.items, "item", "only this item id (repeatable)")
	fl.BoolVar(&p.verbose, "verbose", false, "print compiler output of failing builds to stderr (local debugging; may quote pack source)")
}

// packRun is a loaded pack with its selected items.
type packRun struct {
	pub     *publicContent
	pack    string
	root    *packspec.Root
	sources []*packspec.ItemSource
	all     bool // no --item filter
	ex      packspec.Executor
	stdout  io.Writer
}

// loadPack reads the pack root and resolves every (selected) item against the public half.
// A scaffold (format_major 0) has nothing to materialize: ok is false.
func loadPack(p *pipeFlags, stdout, stderr io.Writer, cmd string) (*packRun, bool, int) {
	pub, err := loadPublic(p.public)
	if err != nil {
		fmt.Fprintf(stderr, "packlint %s: %v\n", cmd, err)
		return nil, false, exitUsage
	}
	b, err := os.ReadFile(filepath.Join(p.pack, packspec.RootFile))
	if err != nil {
		fmt.Fprintf(stderr, "packlint %s: --pack %s: %v\n", cmd, p.pack, err)
		return nil, false, exitUsage
	}
	root, err := packspec.DecodeRoot(b)
	if err == nil {
		err = root.Validate()
	}
	if err != nil {
		fmt.Fprintf(stderr, "packlint %s: %s: %v\n", cmd, packspec.RootFile, err)
		return nil, false, exitFail
	}
	if root.FormatMajor != packspec.FormatMajor {
		fmt.Fprintf(stdout, "packlint %s: format_major %d is mi-07's scaffold; nothing to materialize\n", cmd, root.FormatMajor)
		return nil, false, exitOK
	}
	r := &packRun{pub: pub, pack: p.pack, root: root, all: len(p.items) == 0, stdout: stdout,
		ex: newExecutor(stderr, p.verbose)}
	only := map[string]bool{}
	for _, id := range p.items {
		only[id] = true
	}
	dirs, err := fs.Glob(os.DirFS(p.pack), packspec.ItemsGlob)
	if err != nil {
		fmt.Fprintf(stderr, "packlint %s: %v\n", cmd, err)
		return nil, false, exitUsage
	}
	sort.Strings(dirs)
	failed := false
	for _, d := range dirs {
		parts := strings.Split(d, "/")
		slug, id := parts[1], parts[3]
		if strings.HasPrefix(id, ".") || (len(only) > 0 && !only[id]) {
			continue
		}
		delete(only, id)
		pi := pub.items[id]
		if pi == nil || pi.course != slug {
			fmt.Fprintf(stdout, "packlint %s: %s/%s: no public item in course %s\n", cmd, slug, id, slug)
			failed = true
			continue
		}
		src, err := packspec.LoadItemSource(p.pack, slug, id, pi.resolved)
		if err != nil {
			fmt.Fprintf(stdout, "packlint %s: %s/%s: %v\n", cmd, slug, id, err)
			failed = true
			continue
		}
		r.sources = append(r.sources, src)
	}
	for id := range only {
		fmt.Fprintf(stderr, "packlint %s: no pack item %q\n", cmd, id)
		return nil, false, exitUsage
	}
	if failed {
		return r, false, exitFail
	}
	return r, true, exitOK
}

func (r *packRun) say(format string, args ...any) {
	fmt.Fprintf(r.stdout, format+"\n", args...)
}

// materialize runs one item's materialization, reporting a failure.
func (r *packRun) materialize(ctx context.Context, cmd string, s *packspec.ItemSource) (*packspec.Materialized, bool) {
	m, err := s.Materialize(ctx, r.ex)
	if err != nil {
		r.say("packlint %s: %s: %v", cmd, s.Key(), err)
		return nil, false
	}
	return m, true
}

// --- lock ---------------------------------------------------------------------------

func runLock(args []string, stdout, stderr io.Writer) int {
	fl := flag.NewFlagSet("packlint lock", flag.ContinueOnError)
	fl.SetOutput(stderr)
	var p pipeFlags
	p.register(fl)
	write := fl.Bool("write", false, "rewrite tests.lock for the selected items")
	verify := fl.Bool("verify", false, "regenerate every selected item and require tests.lock's exact entries")
	tools := fl.Bool("tools", false, "print this packlint's tests.lock tool header as JSON and exit (the private CI's full-run trigger)")
	if err := fl.Parse(args); err != nil {
		return exitUsage
	}
	if *tools {
		b, _ := json.Marshal(packspec.CurrentTools(newExecutor(stderr, false).Images()))
		fmt.Fprintln(stdout, string(b))
		return exitOK
	}
	if fl.NArg() > 0 || *write == *verify {
		fmt.Fprintln(stderr, "packlint lock: pick exactly one of --write or --verify")
		return exitUsage
	}
	r, ok, code := loadPack(&p, stdout, stderr, "lock")
	if !ok {
		return code
	}
	lockPath := filepath.Join(r.pack, packspec.LockFile)
	lb, err := os.ReadFile(lockPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		fmt.Fprintln(stderr, "packlint lock:", err)
		return exitUsage
	}
	lock := packspec.NewLock(packspec.CurrentTools(r.ex.Images()))
	if len(lb) > 0 {
		old, err := packspec.DecodeLock(lb)
		if err != nil {
			if *verify {
				r.say("packlint lock: %v", err)
				return exitFail
			}
		} else {
			lock.Cases = old.Cases
			if *verify && old.Format == 1 && !old.Tools.Equal(lock.Tools) {
				r.say("packlint lock: the tests.lock header records other tool versions (re-lock with --write: a full run)")
				return exitFail
			}
		}
	} else if *verify {
		r.say("packlint lock: no %s", packspec.LockFile)
		return exitFail
	}
	ctx := context.Background()
	failed := false
	seen := map[string]bool{}
	for _, s := range r.sources {
		seen[s.Key()] = true
		m, ok := r.materialize(ctx, "lock", s)
		if !ok {
			failed = true
			continue
		}
		got, err := m.Entries()
		if err != nil {
			r.say("packlint lock: %s: %v", s.Key(), err)
			failed = true
			continue
		}
		if *write {
			lock.Cases[s.Key()] = got
			r.say("packlint lock: %s: %d cases locked", s.Key(), len(got))
			continue
		}
		if d := packspec.Diff(lock.Cases[s.Key()], got); !d.Empty() {
			r.say("packlint lock: %s: regenerated cases differ from tests.lock (%s)", s.Key(), d)
			failed = true
		} else {
			r.say("packlint lock: %s: %d cases reproduce tests.lock", s.Key(), len(got))
		}
	}
	if r.all {
		for k := range lock.Cases {
			if !seen[k] {
				if *write {
					delete(lock.Cases, k)
					r.say("packlint lock: %s: no longer in the pack; entry removed", k)
				} else {
					r.say("packlint lock: %s: in tests.lock but not in the pack", k)
					failed = true
				}
			}
		}
	}
	enc, err := lock.Encode()
	if err != nil {
		fmt.Fprintln(stderr, "packlint lock:", err)
		return exitFail
	}
	if *write {
		if failed {
			r.say("packlint lock: not written (fix the errors above)")
			return exitFail
		}
		if err := os.WriteFile(lockPath, enc, 0o644); err != nil {
			fmt.Fprintln(stderr, "packlint lock:", err)
			return exitFail
		}
		r.say("packlint lock: wrote %s (%d items)", packspec.LockFile, len(lock.Cases))
		return exitOK
	}
	if r.all && !failed && !bytes.Equal(enc, lb) {
		r.say("packlint lock: %s is not byte-identical to its regeneration (re-lock with --write)", packspec.LockFile)
		failed = true
	}
	if failed {
		return exitFail
	}
	r.say("packlint lock --verify: ok (%d items)", len(r.sources))
	return exitOK
}

// --- validate -----------------------------------------------------------------------

func runValidate(args []string, stdout, stderr io.Writer) int {
	fl := flag.NewFlagSet("packlint validate", flag.ContinueOnError)
	fl.SetOutput(stderr)
	var p pipeFlags
	p.register(fl)
	if err := fl.Parse(args); err != nil {
		return exitUsage
	}
	if fl.NArg() > 0 {
		fmt.Fprintf(stderr, "packlint validate: unexpected arguments %v\n", fl.Args())
		return exitUsage
	}
	r, ok, code := loadPack(&p, stdout, stderr, "validate")
	if !ok {
		return code
	}
	ctx := context.Background()
	failed := false
	for _, s := range r.sources {
		cases, err := s.Inputs(ctx, r.ex)
		if err == nil {
			var rep packspec.ValidateReport
			rep, err = s.Validate(ctx, r.ex, cases)
			if err == nil {
				r.say("packlint validate: %s: ✓ %d inputs valid, %d invalid inputs rejected, %d custom validator(s)", s.Key(), rep.Inputs, rep.Invalid, rep.Custom)
				continue
			}
		}
		r.say("packlint validate: %s: ✗ %v", s.Key(), err)
		failed = true
	}
	if failed {
		return exitFail
	}
	return exitOK
}

// --- exec ---------------------------------------------------------------------------

var gates = []string{"oracle", "wrong", "tl", "syntax", "all"}

func runExec(args []string, stdout, stderr io.Writer) int {
	fl := flag.NewFlagSet("packlint exec", flag.ContinueOnError)
	fl.SetOutput(stderr)
	var p pipeFlags
	p.register(fl)
	gate := fl.String("gate", "", "oracle | wrong | tl | syntax | all")
	provisional := fl.Bool("provisional", false, "the TL gate on the pinned image with scale 1.0, marked provisional (until m3-13)")
	warm := fl.Bool("warm-cache", false, "build (or restore) the read-only Go std-cache volume, then exit")
	cacheDir := fl.String("cache-dir", "", "with --warm-cache: restore/save the volume as <dir>/<volume>.tar (CI's actions/cache)")
	if err := fl.Parse(args); err != nil {
		return exitUsage
	}
	if fl.NArg() > 0 {
		fmt.Fprintf(stderr, "packlint exec: unexpected arguments %v\n", fl.Args())
		return exitUsage
	}
	ctx := context.Background()
	if *warm {
		d, ok := newExecutor(stderr, p.verbose).(interface {
			WarmCache(context.Context, string, io.Writer) error
		})
		if !ok {
			fmt.Fprintln(stderr, "packlint exec --warm-cache: this executor has no cache")
			return exitUsage
		}
		if err := d.WarmCache(ctx, *cacheDir, stdout); err != nil {
			fmt.Fprintln(stderr, "packlint exec --warm-cache:", err)
			return exitFail
		}
		return exitOK
	}
	if !containsStr(gates, *gate) {
		fmt.Fprintf(stderr, "packlint exec: --gate must be one of %v\n", gates)
		return exitUsage
	}
	if (*gate == "tl" || *gate == "all") && !*provisional {
		fmt.Fprintln(stderr, "packlint exec: the TL gate is provisional until m3-13 re-gates with the runner baselines: pass --provisional")
		return exitUsage
	}
	r, ok, code := loadPack(&p, stdout, stderr, "exec")
	if !ok {
		return code
	}
	want := func(g string) bool { return *gate == "all" || *gate == g }
	var results []packspec.GateResult
	for _, s := range r.sources {
		if want("syntax") {
			results = append(results, s.ReferenceSyntax(ctx, r.ex)...)
		}
		if *gate == "syntax" {
			continue
		}
		m, ok := r.materialize(ctx, "exec", s)
		if !ok {
			results = append(results, packspec.GateResult{Item: s.Key(), Gate: "materialize", Status: packspec.Fail, Detail: "see above"})
			continue
		}
		if want("oracle") {
			results = append(results, m.Oracle(ctx, r.ex)...)
		}
		var wrong []packspec.WrongResult
		if want("wrong") || want("tl") {
			wrong = m.WrongSolutions(ctx, r.ex)
		}
		if want("wrong") {
			for _, w := range wrong {
				results = append(results, w.Gate)
			}
		}
		if want("tl") {
			g, t := m.TimeLimits(wrong, r.ex.Images()["go"])
			if err := m.WriteTiming(t); err != nil {
				g.Status, g.Detail = packspec.Fail, err.Error()
			}
			results = append(results, g)
		}
	}
	failed := false
	counts := map[string]int{}
	for _, g := range results {
		mark := map[string]string{packspec.Pass: "✓", packspec.Fail: "✗", packspec.Pending: "…"}[g.Status]
		r.say("packlint exec: %s %s", mark, g)
		counts[g.Status]++
		failed = failed || g.Status == packspec.Fail
	}
	r.say("packlint exec: %d pass, %d fail, %d pending (%d items)", counts[packspec.Pass], counts[packspec.Fail], counts[packspec.Pending], len(r.sources))
	if failed {
		return exitFail
	}
	return exitOK
}

// --- build --------------------------------------------------------------------------

func runBuild(args []string, stdout, stderr io.Writer) int {
	fl := flag.NewFlagSet("packlint build", flag.ContinueOnError)
	fl.SetOutput(stderr)
	var p pipeFlags
	p.register(fl)
	out := fl.String("out", "build", "the output directory (must be empty or absent)")
	version := fl.String("version", "", "the pack version (must equal pack.json's; the tag without its v)")
	against := fl.String("validated-against", "", "the public commit sha the pack was validated against")
	dockerfile := fl.Bool("dockerfile", true, "also write <out>/Dockerfile (off for the compose fixture pack)")
	if err := fl.Parse(args); err != nil {
		return exitUsage
	}
	if fl.NArg() > 0 || *version == "" || *against == "" {
		fmt.Fprintln(stderr, "packlint build: --version and --validated-against are required")
		return exitUsage
	}
	if len(p.items) > 0 {
		fmt.Fprintln(stderr, "packlint build: builds the whole pack (no --item)")
		return exitUsage
	}
	r, ok, code := loadPack(&p, stdout, stderr, "build")
	if !ok {
		if code == exitOK {
			fmt.Fprintln(stderr, "packlint build: a format_major 0 scaffold cannot be built by this packlint")
			return exitFail
		}
		return code
	}
	if r.root.Version != *version {
		r.say("packlint build: --version %s does not match pack.json's version %s", *version, r.root.Version)
		return exitFail
	}
	lb, err := os.ReadFile(filepath.Join(r.pack, packspec.LockFile))
	if err != nil {
		r.say("packlint build: %v", err)
		return exitFail
	}
	lock, err := packspec.DecodeLock(lb)
	if err != nil {
		r.say("packlint build: %v", err)
		return exitFail
	}
	ctx := context.Background()
	var built []packspec.BuiltItem
	var excluded []string
	failed := false
	for _, s := range r.sources {
		if !s.Pack.Stamped() {
			excluded = append(excluded, s.Key())
			continue
		}
		m, ok := r.materialize(ctx, "build", s)
		if !ok {
			failed = true
			continue
		}
		got, err := m.Entries()
		if err != nil {
			r.say("packlint build: %s: %v", s.Key(), err)
			failed = true
			continue
		}
		if d := packspec.Diff(lock.Cases[s.Key()], got); !d.Empty() {
			r.say("packlint build: %s: cases differ from tests.lock (%s); run lock --verify", s.Key(), d)
			failed = true
			continue
		}
		var kinds []string
		for _, g := range s.Public.Item.Grader {
			if !containsStr(kinds, g.Kind) {
				kinds = append(kinds, g.Kind)
			}
		}
		built = append(built, packspec.BuiltItem{Materialized: m, GraderKinds: kinds})
	}
	if failed {
		return exitFail
	}
	man, err := packspec.Build(built, packspec.BuildOptions{Out: *out, Version: *version, ValidatedAgainst: *against, Dockerfile: *dockerfile})
	if err != nil {
		r.say("packlint build: %v", err)
		return exitFail
	}
	for _, k := range excluded {
		r.say("packlint build: %s: unstamped (review.tests): left out, graded as self", k)
	}
	r.say("packlint build: %s built: %d items, %d unstamped left out", *out, len(man.Items), len(excluded))
	return exitOK
}

// --- listing ------------------------------------------------------------------------

func runListing(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 || strings.HasPrefix(args[0], "-") {
		fmt.Fprintln(stderr, "usage: packlint listing <image ref | image archive .tar | build directory>")
		return exitUsage
	}
	target := args[0]
	var l *packspec.Listing
	var err error
	switch fi, statErr := os.Stat(target); {
	case statErr == nil && fi.IsDir():
		l, err = packspec.ListDir(target)
	case statErr == nil:
		l, err = packspec.ListImageTar(target)
	default:
		l, err = listImageRef(target)
	}
	if err != nil {
		fmt.Fprintln(stderr, "packlint listing:", err)
		return exitUsage
	}
	problems := packspec.CheckListing(l)
	files := 0
	for _, layer := range l.Layers {
		files += len(layer)
	}
	for _, e := range problems {
		fmt.Fprintf(stdout, "packlint listing: ✗ %v\n", e)
	}
	if len(problems) > 0 {
		fmt.Fprintf(stdout, "packlint listing: %d problem(s) in %s\n", len(problems), path.Base(target))
		return exitFail
	}
	fmt.Fprintf(stdout, "packlint listing: ✓ %d files in %d layers, only allowed files, manifest verifies\n", files, len(l.Layers))
	return exitOK
}

// listImageRef saves a local image (docker save) and lists the archive.
func listImageRef(ref string) (*packspec.Listing, error) {
	tmp, err := os.MkdirTemp("", "packlint-listing-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmp)
	archive := filepath.Join(tmp, "image.tar")
	cmd := exec.Command("docker", "save", "-o", archive, ref)
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("docker save %s: %v: %s", ref, err, strings.TrimSpace(string(out)))
	}
	return packspec.ListImageTar(archive)
}

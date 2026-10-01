package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sujaykumarsuman/xlearn/internal/packspec"
)

// The m3-02 subcommands' wiring. The pipeline itself is tested in internal/packspec with a
// fake executor; the real docker executor runs in CI's pack-fixture job. Every value is
// synthetic.

const fixtureRoot = "../../internal/judge/testdata"

func TestListingCommand(t *testing.T) {
	pack := filepath.Join(fixtureRoot, "pack")
	if code, out := runPacklint(t, nil, "listing", pack); code != exitOK || !strings.Contains(out, "✓") {
		t.Fatalf("fixture pack: %d\n%s", code, out)
	}
	planted := t.TempDir()
	if err := os.CopyFS(planted, os.DirFS(pack)); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(planted, "courses/fixture/items/fx-001/gen/planted.go"), "package main\n")
	code, out := runPacklint(t, nil, "listing", planted)
	if code != exitFail || !strings.Contains(out, "gen/planted.go: not allowed in a pack image") {
		t.Fatalf("a planted gen/ file must fail: %d\n%s", code, out)
	}
	if code, _ := runPacklint(t, nil, "listing"); code != exitUsage {
		t.Fatalf("no target: %d", code)
	}
}

func TestPipelineOnScaffold(t *testing.T) {
	pub := publicRoot(t)
	pack := t.TempDir()
	write(t, filepath.Join(pack, "pack.json"), `{"format_major": 0, "version": "0.1.0"}`)
	write(t, filepath.Join(pack, "tests.lock"), "{}\n")
	for _, cmd := range [][]string{
		{"lock", "--verify"}, {"validate"}, {"exec", "--gate", "oracle"},
	} {
		code, out := runPacklint(t, nil, append(cmd, "--public", pub, "--pack", pack)...)
		if code != exitOK || !strings.Contains(out, "scaffold; nothing to materialize") {
			t.Errorf("%v on a scaffold: %d\n%s", cmd, code, out)
		}
	}
	code, out := runPacklint(t, nil, "build", "--public", pub, "--pack", pack, "--out", t.TempDir(), "--version", "0.1.0", "--validated-against", strings.Repeat("0", 40))
	if code != exitFail {
		t.Errorf("build on a scaffold: %d\n%s", code, out)
	}
}

func TestLockTools(t *testing.T) {
	code, out := runPacklint(t, nil, "lock", "--tools")
	var tools packspec.Tools
	if code != exitOK || json.Unmarshal([]byte(out), &tools) != nil || tools.Packlint != packspec.LockVersion || tools.Images["go"] == "" {
		t.Fatalf("lock --tools: %d %s", code, out)
	}
	// The committed fixture lock was written by this packlint: same header.
	b, err := os.ReadFile(filepath.Join(fixtureRoot, "packsrc", "tests.lock"))
	if err != nil {
		t.Fatal(err)
	}
	lock, err := packspec.DecodeLock(b)
	if err != nil || !lock.Tools.Equal(tools) {
		t.Fatalf("the fixture tests.lock header is stale (re-lock it): %+v vs %+v", lock.Tools, tools)
	}
}

func TestPipelineFlags(t *testing.T) {
	for _, args := range [][]string{
		{"lock"}, {"lock", "--write", "--verify"},
		{"exec", "--gate", "tl"}, {"exec", "--gate", "all"}, {"exec", "--gate", "nope"}, {"exec"},
		{"build", "--version", "0.1.0"}, {"build", "--validated-against", "x"}, {"build", "--item", "fx-001", "--version", "1.0.0", "--validated-against", "x"},
	} {
		if code, out := runPacklint(t, nil, args...); code != exitUsage {
			t.Errorf("%v: %d, want usage\n%s", args, code, out)
		}
	}
}

// The pre-push scan reads materialized cases under the pack's build/, compressed included.
func TestFingerprintBuiltCases(t *testing.T) {
	pack := t.TempDir()
	write(t, filepath.Join(pack, "pack.json"), `{"format_major": 1, "version": "0.1.0"}`)
	line := `{"args":[[903,881,577,402,219,764,318,650,991]],"expected":true,"id":"sha256:` + strings.Repeat("e", 64) + `","tags":["random"]}` + "\n"
	write(t, filepath.Join(pack, "build/courses/dsa/items/900/cases.jsonl.zst"), string(packspec.Compress([]byte(line))))
	repo := gitRepo(t)
	commitFile(t, repo, "notes.txt", "copied: 903, 881, 577, 402, 219, 764, 318, 650, 991\n", "docs: notes")
	code, out := fingerprint(t, pack, repo, "--diff", "HEAD~1..HEAD")
	if code != exitFail || !strings.Contains(out, "matches build/courses/dsa/items/900/cases.jsonl.zst:1") {
		t.Fatalf("a payload from a built cases.jsonl.zst must block: %d\n%s", code, out)
	}
	if strings.Contains(out, "903") {
		t.Fatalf("the output leaks the payload:\n%s", out)
	}
}

// Rule 4 accepts the TL gate's timing.json beside pack.json.
func TestCheckAllowsTiming(t *testing.T) {
	pub := publicRoot(t)
	pack := packRoot(t, pub)
	write(t, filepath.Join(pack, "courses/dsa/items/900/timing.json"), `{"provisional": true}`)
	write(t, filepath.Join(pack, "courses/dsa/items/900/notes.txt"), "x")
	_, out := runPacklint(t, nil, "check", "--public", pub, "--pack", pack)
	if strings.Contains(out, "900/timing.json") || !strings.Contains(out, "notes.txt: undeclared file") {
		t.Fatalf("rule 4:\n%s", out)
	}
}

// The fixture course passes check (m3-01's rules) against its own content root.
func TestCheckFixture(t *testing.T) {
	code, out := runPacklint(t, nil, "check", "--public", filepath.Join(fixtureRoot, "content"), "--pack", filepath.Join(fixtureRoot, "packsrc"))
	if code != exitOK || !strings.Contains(out, "0 error(s), 0 warning(s)") {
		t.Fatalf("fixture check: %d\n%s", code, out)
	}
}

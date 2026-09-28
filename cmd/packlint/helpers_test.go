package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sujaykumarsuman/xlearn/internal/course"
)

// Every pack in these tests is SYNTHETIC: built in a temp dir from hand-made values and
// m1-01's original fixture items, never from xlearn-evalpack.

func write(t *testing.T, name, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readRepo(t *testing.T, rel string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", rel))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// publicRoot builds a content root with the DSA manifest and m1-01's original fixtures
// valid-code.json (item 900) and valid-probes.json (item 901).
func publicRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	manifest := readRepo(t, "curriculum/courses/dsa/course.json")
	m, err := course.DecodeManifest(manifest)
	if err != nil {
		t.Fatal(err)
	}
	paths, _ := json.Marshal([]course.PathRow{{Slug: "dsa", Title: m.Title, Status: m.Status, Summary: "s", SortOrder: 1}})
	write(t, filepath.Join(root, "paths.json"), string(paths))
	write(t, filepath.Join(root, "ids.lock.json"),
		`{"items": {"900": {"course": "dsa", "status": "live"}, "901": {"course": "dsa", "status": "live"}}, "assets": {}}`)
	write(t, filepath.Join(root, "courses/dsa/course.json"), string(manifest))
	write(t, filepath.Join(root, "courses/dsa/items/900/item.json"), string(readRepo(t, "internal/course/testdata/items/valid-code.json")))
	write(t, filepath.Join(root, "courses/dsa/items/901/item.json"), string(readRepo(t, "internal/course/testdata/items/valid-probes.json")))
	return root
}

// liveHash returns packlint hash's contract hash for an item.
func liveHash(t *testing.T, pub, id string) string {
	t.Helper()
	code, out := runPacklint(t, nil, "hash", "--public", pub, "--item", id)
	if code != 0 {
		t.Fatalf("packlint hash: %d %s", code, out)
	}
	f := strings.Fields(out)
	if len(f) != 3 {
		t.Fatalf("packlint hash output %q", out)
	}
	return f[2]
}

// packRoot builds a clean synthetic pack for items 900 (code, two wrong solutions,
// stamped) and 901 (a pack-keyed probe).
func packRoot(t *testing.T, pub string) string {
	t.Helper()
	pack := t.TempDir()
	write(t, filepath.Join(pack, "pack.json"), `{"format_major": 0, "version": "0.1.0"}`)
	i900 := filepath.Join(pack, "courses/dsa/items/900")
	write(t, filepath.Join(i900, "pack.json"), `{
  "item": "900",
  "accepts_contract_hashes": ["`+liveHash(t, pub, "900")+`"],
  "wrong": [
    {"file": "submissions/wrong/first-seen.go", "expect": "WA", "category": "right_pattern_wrong_state"},
    {"file": "submissions/wrong/quadratic.go", "expect": "TLE", "category": "complexity_misjudged"}
  ],
  "review": {"tests": "2026-10-09"}
}`)
	write(t, filepath.Join(i900, "submissions/wrong/first-seen.go"), "package wrong\n")
	write(t, filepath.Join(i900, "submissions/wrong/quadratic.go"), "package wrong\n")
	write(t, filepath.Join(i900, "tests/edge.jsonl"), `{"args": [["zz", "yy", "zz"]], "expected": "zz"}`+"\n")
	i901 := filepath.Join(pack, "courses/dsa/items/901")
	write(t, filepath.Join(i901, "pack.json"), `{
  "item": "901",
  "accepts_contract_hashes": ["`+liveHash(t, pub, "901")+`"],
  "keys": {"p-structure": "keys/p-structure.json"}
}`)
	write(t, filepath.Join(i901, "keys/p-structure.json"), `{"answer": "q"}`)
	return pack
}

func runPacklint(t *testing.T, stdin []byte, args ...string) (int, string) {
	t.Helper()
	var out, errb bytes.Buffer
	code := run(args, bytes.NewReader(stdin), &out, &errb)
	return code, out.String() + errb.String()
}

// gitRepo makes a temp git repository with one base commit.
func gitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	git(t, dir, "init", "-q", "-b", "main")
	git(t, dir, "config", "user.email", "packlint-test")
	git(t, dir, "config", "user.name", "test")
	git(t, dir, "config", "commit.gpgsign", "false")
	git(t, dir, "config", "core.hooksPath", "/dev/null")
	write(t, filepath.Join(dir, "README.md"), "base\n")
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-q", "-m", "base")
	return dir
}

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

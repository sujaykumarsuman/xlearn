package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Synthetic corpus values (hand-made for this test; never derived from xlearn-evalpack).
const (
	plantArgs   = `[17, 4, 99, 23, 61, 8, 45, 12, 77, 30]` // 30 bytes stripped
	plantInner  = `17,4,99,23,61,8,45,12,77,30`
	plantKey    = `synthetic-key-value-zeta-omega-42`
	shortString = `short-token-xyz`
	exemplar    = `The candidate first restates the constraints, then names a bounded queue as the core ` +
		`structure, explains why eviction must be constant time, and finally walks two failure ` +
		`scenarios before estimating the memory ceiling for a million entries.`
)

// fpPack is a synthetic pack: one edge case, one key file, one exemplar.
func fpPack(t *testing.T) string {
	t.Helper()
	pack := t.TempDir()
	write(t, filepath.Join(pack, "pack.json"), `{"format_major": 0, "version": "0.1.0"}`)
	item := filepath.Join(pack, "courses/dsa/items/900")
	write(t, filepath.Join(item, "tests/edge.jsonl"),
		`{"args": [`+plantArgs+`], "expected": 12}`+"\n"+
			`{"args": [["`+shortString+`"]], "expected": [1, 2]}`+"\n")
	write(t, filepath.Join(item, "keys/est.json"), `{"answer": "`+plantKey+`"}`)
	write(t, filepath.Join(item, "exemplars/band-5.md"), exemplar+"\n")
	return pack
}

func commitFile(t *testing.T, repo, name, content, msg string) {
	t.Helper()
	write(t, filepath.Join(repo, name), content)
	git(t, repo, "add", "-A")
	git(t, repo, "commit", "-q", "-m", msg)
}

func fingerprint(t *testing.T, pack, repo string, extra ...string) (int, string) {
	t.Helper()
	return runPacklint(t, nil, append([]string{"fingerprint", "--pack", pack, "--repo", repo}, extra...)...)
}

func assertNoPayload(t *testing.T, out string) {
	t.Helper()
	for _, p := range []string{plantInner, "17, 4, 99", plantKey, "bounded queue", "eviction must"} {
		if strings.Contains(out, p) {
			t.Fatalf("the output leaks pack data (%q):\n%s", p, out)
		}
	}
}

// A planted payload in a commit blocks, naming the file and the pack file, never the data.
func TestFingerprintPlantedPayload(t *testing.T) {
	pack := fpPack(t)
	repo := gitRepo(t)
	commitFile(t, repo, "internal/x/x_test.go",
		"package x\n\nvar cases = []int{17, 4, 99,\n\t23, 61, 8, 45, 12, 77, 30}\n", "test: add cases")
	code, out := fingerprint(t, pack, repo, "--diff", "HEAD~1..HEAD")
	if code != exitFail || !strings.Contains(out, "BLOCKED: internal/x/x_test.go:3 contains private eval-pack data (matches courses/dsa/items/900/tests/edge.jsonl:1)") {
		t.Fatalf("exit %d, want 1 naming the file:\n%s", code, out)
	}
	assertNoPayload(t, out)

	// Removed again in a later commit: the pushed history still carries it.
	git(t, repo, "rm", "-q", "internal/x/x_test.go")
	git(t, repo, "commit", "-q", "-m", "remove")
	if code, out := fingerprint(t, pack, repo, "--diff", "HEAD~2..HEAD"); code != exitFail {
		t.Fatalf("an addition removed inside the push must still block: %d\n%s", code, out)
	}
}

// A payload only in a commit message blocks.
func TestFingerprintCommitMessage(t *testing.T) {
	pack := fpPack(t)
	repo := gitRepo(t)
	commitFile(t, repo, "notes.md", "harmless\n", "fix: the estimate is "+plantKey+"\n\nbody")
	code, out := fingerprint(t, pack, repo, "--diff", "HEAD~1..HEAD")
	if code != exitFail || !strings.Contains(out, "message:1 contains private eval-pack data (matches courses/dsa/items/900/keys/est.json)") {
		t.Fatalf("exit %d, want a commit-message hit:\n%s", code, out)
	}
	assertNoPayload(t, out)
}

// A payload shorter than --min-len passes (common short inputs are not fingerprints).
func TestFingerprintShortPayloadPasses(t *testing.T) {
	pack := fpPack(t)
	repo := gitRepo(t)
	commitFile(t, repo, "a.go", "package a\n\nvar s = \""+shortString+"\"\nvar p = []int{1, 2}\n", "add a")
	if code, out := fingerprint(t, pack, repo, "--diff", "HEAD~1..HEAD"); code != exitOK {
		t.Fatalf("a sub-min-len payload blocked: %d\n%s", code, out)
	}
	// Lowering --min-len catches it.
	if code, _ := fingerprint(t, pack, repo, "--diff", "HEAD~1..HEAD", "--min-len", "12"); code != exitFail {
		t.Fatal("--min-len 12 must catch the 15-byte string")
	}
}

// A pasted exemplar paragraph blocks, even rewrapped and recased.
func TestFingerprintExemplarParagraph(t *testing.T) {
	pack := fpPack(t)
	repo := gitRepo(t)
	pasted := strings.ToUpper(exemplar[:120]) + "\n" + exemplar[120:] + "\n"
	commitFile(t, repo, "docs/example.md", "# Example\n\n"+pasted, "docs: example answer")
	code, out := fingerprint(t, pack, repo, "--diff", "HEAD~1..HEAD")
	if code != exitFail || !strings.Contains(out, "BLOCKED: docs/example.md:3 contains private eval-pack data (matches courses/dsa/items/900/exemplars/band-5.md)") {
		t.Fatalf("exit %d, want an exemplar hit:\n%s", code, out)
	}
	assertNoPayload(t, out)
	// Two consecutive shingles (9 words) are not enough.
	repo2 := gitRepo(t)
	words := strings.Fields(exemplar)
	commitFile(t, repo2, "a.md", strings.Join(words[:9], " ")+"\n", "short quote")
	if code, out := fingerprint(t, pack, repo2, "--diff", "HEAD~1..HEAD"); code != exitOK {
		t.Fatalf("9 words blocked: %d\n%s", code, out)
	}
}

// --pre-push reads the hook's stdin: a new branch scans what is not on a remote, a deleted
// ref scans nothing.
func TestFingerprintPrePushStdin(t *testing.T) {
	pack := fpPack(t)
	repo := gitRepo(t)
	base := git(t, repo, "rev-parse", "HEAD")
	commitFile(t, repo, "leak.txt", plantArgs+"\n", "add")
	head := git(t, repo, "rev-parse", "HEAD")
	zero := strings.Repeat("0", 40)
	for name, tc := range map[string]struct {
		stdin string
		code  int
	}{
		"new branch":      {"refs/heads/x " + head + " refs/heads/x " + zero + "\n", exitFail},
		"existing branch": {"refs/heads/x " + head + " refs/heads/x " + base + "\n", exitFail},
		"already pushed":  {"refs/heads/x " + head + " refs/heads/x " + head + "\n", exitOK},
		"deleted ref":     {"(delete) " + zero + " refs/heads/x " + head + "\n", exitOK},
		"nothing to push": {"", exitOK},
		"malformed stdin": {"garbage\n", exitUsage},
		// The base commit only adds README.md.
		"clean new branch": {"refs/heads/x " + base + " refs/heads/x " + zero + "\n", exitOK},
	} {
		code, out := runPacklint(t, []byte(tc.stdin), "fingerprint", "--pre-push", "--pack", pack, "--repo", repo)
		if code != tc.code {
			t.Errorf("%s: exit %d, want %d\n%s", name, code, tc.code, out)
		}
		assertNoPayload(t, out)
	}
}

// --tree scans a checkout (m3-02's private-CI backstop).
func TestFingerprintTree(t *testing.T) {
	pack := fpPack(t)
	dir := t.TempDir()
	write(t, filepath.Join(dir, "ok.txt"), "nothing here\n")
	if code, out := fingerprint(t, pack, dir, "--tree", dir); code != exitOK {
		t.Fatalf("clean tree: %d\n%s", code, out)
	}
	write(t, filepath.Join(dir, "sub/leak.py"), "x = "+plantArgs+"\n")
	code, out := fingerprint(t, pack, dir, "--tree", dir)
	if code != exitFail || !strings.Contains(out, "BLOCKED: sub/leak.py:1") {
		t.Fatalf("tree leak: %d\n%s", code, out)
	}
	assertNoPayload(t, out)
}

// The shell hook: no pack directory -> exit 0 silently.
func TestHookNoPackIsNoop(t *testing.T) {
	hook, err := filepath.Abs("../../hack/git-hooks/pre-push")
	if err != nil {
		t.Fatal(err)
	}
	repo := gitRepo(t)
	cmd := exec.Command("sh", hook, "origin", "unused")
	cmd.Dir = repo
	cmd.Env = append(os.Environ(), "XLEARN_EVALPACK_DIR="+filepath.Join(t.TempDir(), "absent"))
	cmd.Stdin = strings.NewReader("refs/heads/main " + git(t, repo, "rev-parse", "HEAD") + " refs/heads/main " + strings.Repeat("0", 40) + "\n")
	out, err := cmd.CombinedOutput()
	if err != nil || len(out) != 0 {
		t.Fatalf("hook without a pack: err %v, output %q (want exit 0, silent)", err, out)
	}
}

// End to end: the hook installed in a clone blocks a push of a planted synthetic string
// to a local bare remote without printing it, and lets the clean push through.
func TestHookBlocksPushToBareRemote(t *testing.T) {
	if testing.Short() {
		t.Skip("builds packlint and pushes to a bare repository")
	}
	bin := filepath.Join(t.TempDir(), "packlint")
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	hooks, err := filepath.Abs("../../hack/git-hooks")
	if err != nil {
		t.Fatal(err)
	}
	pack := fpPack(t)
	remote := filepath.Join(t.TempDir(), "remote.git")
	git(t, t.TempDir(), "init", "-q", "--bare", remote)
	repo := gitRepo(t)
	git(t, repo, "config", "core.hooksPath", hooks)
	git(t, repo, "remote", "add", "scratch", remote)
	env := append(os.Environ(), "XLEARN_EVALPACK_DIR="+pack, "XLEARN_PACKLINT="+bin)

	push := func() (string, error) {
		cmd := exec.Command("git", "push", "-q", "scratch", "HEAD:refs/heads/main")
		cmd.Dir = repo
		cmd.Env = env
		out, err := cmd.CombinedOutput()
		return string(out), err
	}
	if out, err := push(); err != nil {
		t.Fatalf("clean push blocked: %v\n%s", err, out)
	}
	commitFile(t, repo, "fixture.json", `{"nums": `+plantArgs+`}`+"\n", "add fixture")
	out, err := push()
	if err == nil || !strings.Contains(out, "BLOCKED: fixture.json:1") {
		t.Fatalf("planted push: err %v, want BLOCKED:\n%s", err, out)
	}
	assertNoPayload(t, out)
	if got := git(t, remote, "rev-parse", "refs/heads/main"); got == git(t, repo, "rev-parse", "HEAD") {
		t.Fatal("the planted commit reached the remote")
	}
	// Remove the plant from history; the push passes.
	git(t, repo, "reset", "-q", "--hard", "HEAD~1")
	commitFile(t, repo, "fixture.json", `{"nums": [1, 2, 3]}`+"\n", "add fixture")
	if out, err := push(); err != nil {
		t.Fatalf("clean push after removing the plant: %v\n%s", err, out)
	}
}

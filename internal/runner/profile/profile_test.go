package profile_test

import (
	"maps"
	"os"
	"path/filepath"
	"testing"

	"github.com/sujaykumarsuman/xlearn/internal/platform/harness"
	"github.com/sujaykumarsuman/xlearn/internal/runner/profile"
	_ "github.com/sujaykumarsuman/xlearn/internal/runner/profile/cpp"
	_ "github.com/sujaykumarsuman/xlearn/internal/runner/profile/go"
	_ "github.com/sujaykumarsuman/xlearn/internal/runner/profile/python"
)

// TestForLanguage: the item's languages[] values map to the launch profiles in one table.
func TestForLanguage(t *testing.T) {
	for lang, want := range map[string]string{"go": "go@1.26", "cpp": "cpp@g++14", "python": "python@3.13"} {
		p, ok := profile.ForLanguage(lang)
		if !ok || p.Name != want {
			t.Errorf("ForLanguage(%s) = %v", lang, p)
		}
		if !p.Accepts(harness.FuncJSON) || !p.Accepts(harness.ClassOps) {
			t.Errorf("%s takes both harnesses", want)
		}
		if len(p.Assets) == 0 {
			t.Errorf("%s carries no harness assets", want)
		}
	}
	if _, ok := profile.ForLanguage("rust"); ok {
		t.Error("rust is not a launch language")
	}
}

// TestFingerprintCoversTemplates: ProfileSHA moves with a one-byte template change, a
// toolchain version change and a toolchain tree change.
func TestFingerprintCoversTemplates(t *testing.T) {
	p, _ := profile.ForLanguage("cpp")
	tree := func(string) (string, error) { return "tree", nil }
	base, err := profile.Fingerprint(p, "g++ 14.2.0", tree)
	if err != nil {
		t.Fatal(err)
	}
	q := *p
	q.Assets = maps.Clone(p.Assets)
	for k, v := range q.Assets {
		b := append([]byte{}, v...)
		b[len(b)-1] ^= 1
		q.Assets[k] = b
		break
	}
	if s, _ := profile.Fingerprint(&q, "g++ 14.2.0", tree); s == base {
		t.Error("a one-byte template change must change ProfileSHA")
	}
	if s, _ := profile.Fingerprint(p, "g++ 14.2.1", tree); s == base {
		t.Error("a toolchain version change must change ProfileSHA")
	}
	if len(p.ToolchainPaths) > 0 {
		if s, _ := profile.Fingerprint(p, "g++ 14.2.0", func(string) (string, error) { return "other", nil }); s == base {
			t.Error("a toolchain tree change must change ProfileSHA")
		}
	}
	if s, _ := profile.Fingerprint(p, "g++ 14.2.0", tree); s != base {
		t.Error("ProfileSHA is deterministic")
	}
}

func TestTreeHash(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a"), []byte("x"), 0o644)
	h1, err := profile.TreeHash(dir)
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(dir, "a"), []byte("y"), 0o644)
	if h2, _ := profile.TreeHash(dir); h2 == h1 {
		t.Error("a content change must change the tree hash")
	}
	if _, err := profile.TreeHash(filepath.Join(dir, "a")); err != nil {
		t.Errorf("a single file is a tree: %v", err)
	}
}

func TestTextDiags(t *testing.T) {
	out := []byte("./solution.go:3:2: undefined: x\n/src/solution.go:4: y\nother.go:1:1: z\n")
	d := profile.TextDiags(out, map[string]bool{"solution.go": true})
	if len(d) != 2 || d[0].Line != 3 || d[0].Col != 2 || d[1].Line != 4 || d[1].Col != 0 {
		t.Errorf("diags %+v", d)
	}
}

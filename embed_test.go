package xlearn

import "testing"

// TestDistIsReadable guards the go:embed contract: web/dist must contain at
// least .gitkeep (committed) so the binary always compiles, and the real Vite
// build overwrites it before release.
func TestDistIsReadable(t *testing.T) {
	entries, err := Dist.ReadDir("web/dist")
	if err != nil {
		t.Fatalf("ReadDir(web/dist): %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("web/dist is empty; expected at least .gitkeep")
	}
}

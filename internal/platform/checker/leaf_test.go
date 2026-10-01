package checker

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// The checker registry is linked into judge (m3-06's code@1) and packlint: the stdlib,
// internal/course and internal/platform/harness only.
func TestCheckerIsALeaf(t *testing.T) {
	allowed := map[string]bool{
		"github.com/sujaykumarsuman/xlearn/internal/course":           true,
		"github.com/sujaykumarsuman/xlearn/internal/platform/harness": true,
	}
	files, _ := filepath.Glob("*.go")
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		af, err := parser.ParseFile(token.NewFileSet(), f, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range af.Imports {
			p, _ := strconv.Unquote(imp.Path.Value)
			first, _, _ := strings.Cut(p, "/")
			if strings.Contains(first, ".") && !allowed[p] {
				t.Errorf("%s imports %s: the checker registry is a leaf library", f, p)
			}
		}
	}
}

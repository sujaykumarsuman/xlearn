package harness

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// leafImports checks that a package's non-test files import only the stdlib and the listed
// module packages: the shared harness and checker are linked into judge and packlint, so
// they must never pull in the jail, cgroups, go-sandbox or a service (m3-04 task 1).
func leafImports(t *testing.T, dir string, allowed ...string) {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	ok := map[string]bool{}
	for _, a := range allowed {
		ok[a] = true
	}
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
			if !strings.Contains(first, ".") || ok[p] {
				continue
			}
			t.Errorf("%s imports %s: a leaf library imports the stdlib and %v only", f, p, allowed)
		}
	}
}

func TestHarnessIsALeaf(t *testing.T) {
	leafImports(t, ".",
		"github.com/sujaykumarsuman/xlearn/internal/course",
		"github.com/sujaykumarsuman/xlearn/internal/platform/runnerapi")
}

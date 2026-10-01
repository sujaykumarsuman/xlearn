package lint

import (
	"fmt"
	"go/parser"
	"go/scanner"
	"go/token"
	"strconv"
	"strings"

	"github.com/sujaykumarsuman/xlearn/internal/platform/runnerapi"
)

// checkGo applies t3 A5 to solution.go: the import allowlist (go/parser, imports only) and
// the directive denylist over every comment (go/scanner, so a file that doesn't parse is
// still scanned; its syntax error is the compile's CE, not a REJECTED).
func checkGo(f runnerapi.File) []Violation {
	var out []Violation
	fset := token.NewFileSet()
	if af, err := parser.ParseFile(fset, f.Path, f.Data, parser.ImportsOnly); err == nil {
		for _, imp := range af.Imports {
			p, _ := strconv.Unquote(imp.Path.Value)
			if !goImportAllowed(p) {
				out = append(out, Violation{File: f.Path, Line: fset.Position(imp.Pos()).Line, Rule: RuleGoImport,
					Msg: fmt.Sprintf("import %q is not in the go allowlist", p)})
			}
		}
	}
	var s scanner.Scanner
	file := token.NewFileSet().AddFile(f.Path, -1, len(f.Data))
	s.Init(file, f.Data, nil, scanner.ScanComments)
	for {
		pos, tok, lit := s.Scan()
		if tok == token.EOF {
			break
		}
		if tok != token.COMMENT {
			continue
		}
		if why := goDirective(lit); why != "" {
			out = append(out, Violation{File: f.Path, Line: file.Position(pos).Line, Rule: RuleGoDirective, Msg: why})
		}
	}
	return out
}

// goDirective reports why a comment is a denied directive ("" if it isn't one): //line and
// /*line*/ (position rewriting), //export (cgo), //go:embed and every other //go: directive
// except //go:build. Directives have no space after the slashes; a plain comment may.
func goDirective(c string) string {
	switch {
	case strings.HasPrefix(c, "//line ") || strings.HasPrefix(c, "/*line "):
		return "a line directive"
	case strings.HasPrefix(c, "//export "):
		return "a cgo export directive"
	case strings.HasPrefix(c, "//go:"):
		name := strings.TrimPrefix(c, "//go:")
		if i := strings.IndexAny(name, " \t"); i >= 0 {
			name = name[:i]
		}
		if name == "build" {
			return ""
		}
		return "the //go:" + name + " directive"
	}
	return ""
}

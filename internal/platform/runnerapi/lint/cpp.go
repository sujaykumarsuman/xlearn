package lint

import (
	"fmt"
	"strings"

	"github.com/sujaykumarsuman/xlearn/internal/platform/runnerapi"
)

// checkCpp applies the C++ rules to solution.cpp over a small preprocessor-aware tokenizer:
// line splices are joined first (so `#inc\<newline>lude` is still an include), comments and
// string, character and raw-string literals are skipped, `#` and the digraph `%:` start a
// directive at the beginning of a line.
//
//   - #include <h> only for a standard-library header (CppHeaders); never #include "…" or a
//     computed include; never #include_next, #import, #embed or #line;
//   - #pragma only GCC optimize / GCC target; never the _Pragma operator;
//   - never asm / __asm / __asm__, and never a learner-defined main (any `main` identifier).
func checkCpp(f runnerapi.File) []Violation {
	t := newCppText(f.Data)
	var out []Violation
	add := func(line int, rule, format string, args ...any) {
		out = append(out, Violation{File: f.Path, Line: line, Rule: rule, Msg: fmt.Sprintf(format, args...)})
	}
	for _, tok := range t.tokens() {
		switch tok.kind {
		case cppDirective:
			name, rest := splitDirective(tok.text)
			switch name {
			case "include":
				switch {
				case strings.HasPrefix(rest, "<"):
					h, _, ok := strings.Cut(rest[1:], ">")
					if !ok || !CppHeaders[strings.TrimSpace(h)] {
						add(tok.line, RuleCppInclude, "#include <%s> is not a standard-library header", strings.TrimSpace(h))
					}
				case strings.HasPrefix(rest, `"`):
					add(tok.line, RuleCppInclude, "#include \"…\" is not allowed")
				default:
					add(tok.line, RuleCppInclude, "a computed #include is not allowed")
				}
			case "include_next", "import", "embed", "line":
				add(tok.line, RuleCppDirective, "#%s is not allowed", name)
			case "pragma":
				w := strings.Fields(rest)
				if len(w) < 2 || w[0] != "GCC" || !(strings.HasPrefix(w[1], "optimize") || strings.HasPrefix(w[1], "target")) {
					add(tok.line, RuleCppDirective, "#pragma %s is not allowed (only GCC optimize and GCC target)", rest)
				}
			}
			// Identifiers inside directives (a #define body) are checked like code.
			for _, id := range cppIdents(rest) {
				out = append(out, identViolations(f.Path, tok.line, id)...)
			}
		case cppIdent:
			out = append(out, identViolations(f.Path, tok.line, tok.text)...)
		}
	}
	return out
}

func identViolations(file string, line int, id string) []Violation {
	switch id {
	case "asm", "__asm", "__asm__":
		return []Violation{{File: file, Line: line, Rule: RuleCppAsm, Msg: id + " is not allowed"}}
	case "main":
		return []Violation{{File: file, Line: line, Rule: RuleCppMain, Msg: "the harness defines main; the learner's file may not"}}
	case "_Pragma":
		return []Violation{{File: file, Line: line, Rule: RuleCppDirective, Msg: "_Pragma is not allowed"}}
	}
	return nil
}

// splitDirective splits "include <x>" into ("include", "<x>").
func splitDirective(s string) (string, string) {
	s = strings.TrimSpace(s)
	i := 0
	for i < len(s) && isIdentByte(s[i]) {
		i++
	}
	return s[:i], strings.TrimSpace(s[i:])
}

type cppKind int

const (
	cppIdent cppKind = iota
	cppDirective
)

type cppTok struct {
	kind cppKind
	text string
	line int
}

// cppText is the source after line splicing, with each byte's original line.
type cppText struct {
	b    []byte
	line []int
}

func newCppText(src []byte) *cppText {
	t := &cppText{}
	ln := 1
	for i := 0; i < len(src); i++ {
		c := src[i]
		if c == '\\' {
			j := i + 1
			if j < len(src) && src[j] == '\r' {
				j++
			}
			if j < len(src) && src[j] == '\n' {
				ln++
				i = j
				continue
			}
		}
		t.b = append(t.b, c)
		t.line = append(t.line, ln)
		if c == '\n' {
			ln++
		}
	}
	return t
}

func isIdentByte(c byte) bool {
	return c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c >= 0x80
}

// tokens returns the identifiers outside comments and literals, and each directive (the text
// after `#`, comments removed, up to the end of the logical line).
func (t *cppText) tokens() []cppTok {
	var out []cppTok
	b := t.b
	lineStart := true // only whitespace (or comments) since the last newline
	for i := 0; i < len(b); {
		c := b[i]
		switch {
		case c == '\n':
			lineStart = true
			i++
		case c == ' ' || c == '\t' || c == '\r' || c == '\f' || c == '\v':
			i++
		case c == '/' && i+1 < len(b) && b[i+1] == '/':
			for i < len(b) && b[i] != '\n' {
				i++
			}
		case c == '/' && i+1 < len(b) && b[i+1] == '*':
			i = skipBlockComment(b, i)
		case lineStart && (c == '#' || (c == '%' && i+1 < len(b) && b[i+1] == ':')):
			line := t.line[i]
			if c == '%' {
				i++
			}
			i++
			var d strings.Builder
			for i < len(b) && b[i] != '\n' {
				switch {
				case b[i] == '/' && i+1 < len(b) && b[i+1] == '/':
					for i < len(b) && b[i] != '\n' {
						i++
					}
				case b[i] == '/' && i+1 < len(b) && b[i+1] == '*':
					i = skipBlockComment(b, i)
					d.WriteByte(' ')
				default:
					d.WriteByte(b[i])
					i++
				}
			}
			out = append(out, cppTok{kind: cppDirective, text: d.String(), line: line})
		case c == '"' || c == '\'':
			lineStart = false
			i = skipQuoted(b, i)
		case isIdentByte(c) && !(c >= '0' && c <= '9'):
			lineStart = false
			s := i
			for i < len(b) && isIdentByte(b[i]) {
				i++
			}
			id := string(b[s:i])
			// A raw string: R"delim( … )delim", with an encoding prefix.
			if i < len(b) && b[i] == '"' && (id == "R" || id == "u8R" || id == "uR" || id == "UR" || id == "LR") {
				i = skipRaw(b, i)
				continue
			}
			if i < len(b) && (b[i] == '"' || b[i] == '\'') && (id == "u8" || id == "u" || id == "U" || id == "L") {
				i = skipQuoted(b, i)
				continue
			}
			out = append(out, cppTok{kind: cppIdent, text: id, line: t.line[s]})
		case c >= '0' && c <= '9':
			lineStart = false
			// A pp-number (with digit separators): 1'000, 0x1p3, 1e+5.
			for i < len(b) && (isIdentByte(b[i]) || b[i] == '.' || b[i] == '\'' ||
				((b[i] == '+' || b[i] == '-') && (b[i-1] == 'e' || b[i-1] == 'E' || b[i-1] == 'p' || b[i-1] == 'P'))) {
				i++
			}
		default:
			lineStart = false
			i++
		}
	}
	return out
}

func skipBlockComment(b []byte, i int) int {
	i += 2
	for i+1 < len(b) && !(b[i] == '*' && b[i+1] == '/') {
		i++
	}
	return min(i+2, len(b))
}

func skipQuoted(b []byte, i int) int {
	q := b[i]
	i++
	for i < len(b) && b[i] != q && b[i] != '\n' {
		if b[i] == '\\' {
			i++
		}
		i++
	}
	return min(i+1, len(b))
}

func skipRaw(b []byte, i int) int {
	i++ // the opening quote
	s := i
	for i < len(b) && b[i] != '(' && i-s <= 16 {
		i++
	}
	if i >= len(b) || b[i] != '(' {
		return i
	}
	end := ")" + string(b[s:i]) + `"`
	k := strings.Index(string(b[i:]), end)
	if k < 0 {
		return len(b)
	}
	return i + k + len(end)
}

// cppIdents lists the identifiers in a directive's text (outside its literals).
func cppIdents(s string) []string {
	var out []string
	b := []byte(s)
	for i := 0; i < len(b); {
		switch c := b[i]; {
		case c == '"' || c == '\'':
			i = skipQuoted(b, i)
		case c == '<' && strings.HasPrefix(strings.TrimSpace(s), "include"):
			// A header name is not code.
			for i < len(b) && b[i] != '>' {
				i++
			}
		case isIdentByte(c) && !(c >= '0' && c <= '9'):
			st := i
			for i < len(b) && isIdentByte(b[i]) {
				i++
			}
			out = append(out, string(b[st:i]))
		case c >= '0' && c <= '9':
			for i < len(b) && (isIdentByte(b[i]) || b[i] == '.') {
				i++
			}
		default:
			i++
		}
	}
	return out
}

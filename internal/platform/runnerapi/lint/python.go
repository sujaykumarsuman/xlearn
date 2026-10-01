package lint

import (
	"fmt"
	"slices"
	"strings"

	"github.com/sujaykumarsuman/xlearn/internal/platform/runnerapi"
)

// checkPython applies the Python import allowlist to solution.py. Decision (m3-04, recorded in
// the decisions log): the walk runs in Go over a small tokenizer, in judge before enqueue, so a
// disallowed import is REJECTED without spending a compile; nothing runs Python for it.
// `import` is a hard keyword, so every `import` token starts or continues an import statement;
// `from` starts one only when a dotted name and `import` follow (not `yield from` or
// `raise … from`). Relative imports are refused. `__import__` and importlib are expected
// evasions the jail stops (no network, no shell, KILL-default seccomp).
func checkPython(f runnerapi.File) []Violation {
	toks := pyTokens(f.Data)
	var out []Violation
	bad := func(t pyTok, mod string) {
		out = append(out, Violation{File: f.Path, Line: t.line, Rule: RulePyImport, Msg: fmt.Sprintf("import %s is not in the python allowlist", mod)})
	}
	for i := 0; i < len(toks); i++ {
		t := toks[i]
		if t.kind != pyName {
			continue
		}
		switch t.text {
		case "from":
			j := i + 1
			dots := 0
			for j < len(toks) && toks[j].kind == pyOp && (toks[j].text == "." || toks[j].text == "...") {
				dots += len(toks[j].text)
				j++
			}
			mod, k := dottedName(toks, j)
			if k >= len(toks) || toks[k].kind != pyName || toks[k].text != "import" {
				continue // `yield from x`, `raise e from x`
			}
			switch {
			case dots > 0:
				bad(t, strings.Repeat(".", dots)+mod)
			case !pyAllowed(mod):
				bad(t, mod)
			}
			i = k // the `import` of this statement
		case "import":
			j := i + 1
			for {
				mod, k := dottedName(toks, j)
				if mod == "" {
					break
				}
				if !pyAllowed(mod) {
					bad(toks[j], mod)
				}
				j = k
				if j+1 < len(toks) && toks[j].kind == pyName && toks[j].text == "as" {
					j += 2
				}
				if j < len(toks) && toks[j].kind == pyOp && toks[j].text == "," {
					j++
					continue
				}
				break
			}
			i = j - 1
		}
	}
	return out
}

func pyAllowed(mod string) bool {
	top, _, _ := strings.Cut(mod, ".")
	return slices.Contains(PythonImports, top)
}

// dottedName reads NAME ('.' NAME)* from toks[j]; it returns the name and the next index.
func dottedName(toks []pyTok, j int) (string, int) {
	var parts []string
	for j < len(toks) && toks[j].kind == pyName {
		parts = append(parts, toks[j].text)
		j++
		if j+1 < len(toks) && toks[j].kind == pyOp && toks[j].text == "." && toks[j+1].kind == pyName {
			j++
			continue
		}
		break
	}
	return strings.Join(parts, "."), j
}

type pyKind int

const (
	pyName pyKind = iota
	pyOp
)

type pyTok struct {
	kind pyKind
	text string
	line int
}

// pyTokens returns the NAME and operator tokens of a Python source, skipping comments, string
// literals (every prefix, single and triple quoted) and line continuations.
func pyTokens(src []byte) []pyTok {
	var out []pyTok
	line := 1
	for i := 0; i < len(src); {
		c := src[i]
		switch {
		case c == '\n':
			line++
			i++
		case c == '#':
			for i < len(src) && src[i] != '\n' {
				i++
			}
		case c == '\\' && i+1 < len(src) && (src[i+1] == '\n' || src[i+1] == '\r'):
			i++
		case c == '"' || c == '\'':
			i, line = skipPyString(src, i, line)
		case c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= 0x80:
			s := i
			for i < len(src) && (isIdentByte(src[i])) {
				i++
			}
			word := string(src[s:i])
			if i < len(src) && (src[i] == '"' || src[i] == '\'') && isStringPrefix(word) {
				i, line = skipPyString(src, i, line)
				continue
			}
			out = append(out, pyTok{kind: pyName, text: word, line: line})
		case c >= '0' && c <= '9':
			for i < len(src) && (isIdentByte(src[i]) || src[i] == '.') {
				i++
			}
		case c == '.':
			if i+2 < len(src) && src[i+1] == '.' && src[i+2] == '.' {
				out = append(out, pyTok{kind: pyOp, text: "...", line: line})
				i += 3
			} else {
				out = append(out, pyTok{kind: pyOp, text: ".", line: line})
				i++
			}
		case c == ' ' || c == '\t' || c == '\r' || c == '\f':
			i++
		default:
			out = append(out, pyTok{kind: pyOp, text: string(c), line: line})
			i++
		}
	}
	return out
}

func isStringPrefix(w string) bool {
	switch strings.ToLower(w) {
	case "r", "u", "b", "f", "br", "rb", "fr", "rf", "t", "tr", "rt":
		return true
	}
	return false
}

func skipPyString(src []byte, i, line int) (int, int) {
	q := src[i]
	if i+2 < len(src) && src[i+1] == q && src[i+2] == q {
		i += 3
		for i < len(src) {
			switch {
			case src[i] == '\\':
				if i+1 < len(src) && src[i+1] == '\n' {
					line++
				}
				i += 2
				continue
			case src[i] == '\n':
				line++
			case src[i] == q && i+2 < len(src) && src[i+1] == q && src[i+2] == q:
				return i + 3, line
			}
			i++
		}
		return len(src), line
	}
	i++
	for i < len(src) && src[i] != q && src[i] != '\n' {
		if src[i] == '\\' {
			if i+1 < len(src) && src[i+1] == '\n' {
				line++
			}
			i++
		}
		i++
	}
	return min(i+1, len(src)), line
}

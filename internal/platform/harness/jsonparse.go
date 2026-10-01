package harness

import (
	"errors"
	"fmt"
	"strconv"
	"unicode/utf16"
	"unicode/utf8"
)

// A strict JSON reader (RFC 8259) for the codec: invalid UTF-8, duplicate object keys,
// trailing data and nesting deeper than MaxDepth are errors (encoding/json silently
// replaces invalid UTF-8 and keeps the last duplicate). Numbers stay as their literal text
// so integers are parsed exactly.

// MaxDepth bounds JSON nesting in any frame or case value.
const MaxDepth = 64

// MaxNodes bounds the ListNode / TreeNode / GraphNode nodes one value may hold (m3-04).
const MaxNodes = 1_000_000

type jkind uint8

const (
	jNull jkind = iota
	jBool
	jNum
	jStr
	jArr
	jObj
)

// jv is one parsed JSON value.
type jv struct {
	kind jkind
	b    bool
	s    string // number literal or string text
	arr  []jv
	keys []string // object keys in document order
	vals []jv
}

// field returns an object member.
func (v *jv) field(k string) (*jv, bool) {
	for i, kk := range v.keys {
		if kk == k {
			return &v.vals[i], true
		}
	}
	return nil, false
}

// ErrJSON is the class of every malformed-JSON error.
var ErrJSON = errors.New("harness: malformed JSON")

func parseJSON(b []byte) (jv, error) {
	p := &jparser{b: b}
	p.ws()
	v, err := p.value(0)
	if err != nil {
		return jv{}, err
	}
	p.ws()
	if p.i != len(p.b) {
		return jv{}, p.errf("trailing data")
	}
	return v, nil
}

type jparser struct {
	b []byte
	i int
}

func (p *jparser) errf(format string, args ...any) error {
	return fmt.Errorf("%w at byte %d: %s", ErrJSON, p.i, fmt.Sprintf(format, args...))
}

func (p *jparser) ws() {
	for p.i < len(p.b) {
		switch p.b[p.i] {
		case ' ', '\t', '\n', '\r':
			p.i++
		default:
			return
		}
	}
}

func (p *jparser) value(depth int) (jv, error) {
	if depth > MaxDepth {
		return jv{}, p.errf("nesting deeper than %d", MaxDepth)
	}
	if p.i >= len(p.b) {
		return jv{}, p.errf("unexpected end")
	}
	switch c := p.b[p.i]; {
	case c == '{':
		return p.object(depth)
	case c == '[':
		return p.array(depth)
	case c == '"':
		s, err := p.str()
		return jv{kind: jStr, s: s}, err
	case c == 't':
		return jv{kind: jBool, b: true}, p.lit("true")
	case c == 'f':
		return jv{kind: jBool}, p.lit("false")
	case c == 'n':
		return jv{kind: jNull}, p.lit("null")
	case c == '-' || (c >= '0' && c <= '9'):
		s, err := p.num()
		return jv{kind: jNum, s: s}, err
	default:
		return jv{}, p.errf("unexpected %q", c)
	}
}

func (p *jparser) lit(s string) error {
	if len(p.b)-p.i < len(s) || string(p.b[p.i:p.i+len(s)]) != s {
		return p.errf("invalid literal")
	}
	p.i += len(s)
	return nil
}

func (p *jparser) object(depth int) (jv, error) {
	p.i++ // {
	v := jv{kind: jObj}
	p.ws()
	if p.i < len(p.b) && p.b[p.i] == '}' {
		p.i++
		return v, nil
	}
	seen := map[string]bool{}
	for {
		p.ws()
		if p.i >= len(p.b) || p.b[p.i] != '"' {
			return jv{}, p.errf("want an object key")
		}
		k, err := p.str()
		if err != nil {
			return jv{}, err
		}
		if seen[k] {
			return jv{}, p.errf("duplicate key %q", k)
		}
		seen[k] = true
		p.ws()
		if p.i >= len(p.b) || p.b[p.i] != ':' {
			return jv{}, p.errf("want ':'")
		}
		p.i++
		p.ws()
		val, err := p.value(depth + 1)
		if err != nil {
			return jv{}, err
		}
		v.keys, v.vals = append(v.keys, k), append(v.vals, val)
		p.ws()
		if p.i >= len(p.b) {
			return jv{}, p.errf("unterminated object")
		}
		switch p.b[p.i] {
		case ',':
			p.i++
		case '}':
			p.i++
			return v, nil
		default:
			return jv{}, p.errf("want ',' or '}'")
		}
	}
}

func (p *jparser) array(depth int) (jv, error) {
	p.i++ // [
	v := jv{kind: jArr, arr: []jv{}}
	p.ws()
	if p.i < len(p.b) && p.b[p.i] == ']' {
		p.i++
		return v, nil
	}
	for {
		p.ws()
		e, err := p.value(depth + 1)
		if err != nil {
			return jv{}, err
		}
		v.arr = append(v.arr, e)
		p.ws()
		if p.i >= len(p.b) {
			return jv{}, p.errf("unterminated array")
		}
		switch p.b[p.i] {
		case ',':
			p.i++
		case ']':
			p.i++
			return v, nil
		default:
			return jv{}, p.errf("want ',' or ']'")
		}
	}
}

func (p *jparser) num() (string, error) {
	start := p.i
	if p.b[p.i] == '-' {
		p.i++
	}
	digits := func() int {
		n := 0
		for p.i < len(p.b) && p.b[p.i] >= '0' && p.b[p.i] <= '9' {
			p.i++
			n++
		}
		return n
	}
	if p.i < len(p.b) && p.b[p.i] == '0' {
		p.i++
	} else if digits() == 0 {
		return "", p.errf("invalid number")
	}
	if p.i < len(p.b) && p.b[p.i] == '.' {
		p.i++
		if digits() == 0 {
			return "", p.errf("invalid number")
		}
	}
	if p.i < len(p.b) && (p.b[p.i] == 'e' || p.b[p.i] == 'E') {
		p.i++
		if p.i < len(p.b) && (p.b[p.i] == '+' || p.b[p.i] == '-') {
			p.i++
		}
		if digits() == 0 {
			return "", p.errf("invalid number")
		}
	}
	return string(p.b[start:p.i]), nil
}

func (p *jparser) str() (string, error) {
	p.i++ // "
	var out []byte
	for {
		if p.i >= len(p.b) {
			return "", p.errf("unterminated string")
		}
		c := p.b[p.i]
		switch {
		case c == '"':
			p.i++
			return string(out), nil
		case c < 0x20:
			return "", p.errf("raw control character in a string")
		case c == '\\':
			p.i++
			if p.i >= len(p.b) {
				return "", p.errf("unterminated escape")
			}
			e := p.b[p.i]
			p.i++
			switch e {
			case '"', '\\', '/':
				out = append(out, e)
			case 'b':
				out = append(out, '\b')
			case 'f':
				out = append(out, '\f')
			case 'n':
				out = append(out, '\n')
			case 'r':
				out = append(out, '\r')
			case 't':
				out = append(out, '\t')
			case 'u':
				r, err := p.hex4()
				if err != nil {
					return "", err
				}
				if utf16.IsSurrogate(r) {
					if r >= 0xdc00 || p.i+6 > len(p.b) || p.b[p.i] != '\\' || p.b[p.i+1] != 'u' {
						return "", p.errf("unpaired surrogate")
					}
					p.i += 2
					r2, err := p.hex4()
					if err != nil {
						return "", err
					}
					r = utf16.DecodeRune(r, r2)
					if r == utf8.RuneError {
						return "", p.errf("invalid surrogate pair")
					}
				}
				out = utf8.AppendRune(out, r)
			default:
				return "", p.errf("invalid escape \\%c", e)
			}
		case c < utf8.RuneSelf:
			out = append(out, c)
			p.i++
		default:
			r, n := utf8.DecodeRune(p.b[p.i:])
			if r == utf8.RuneError && n <= 1 {
				return "", p.errf("invalid UTF-8")
			}
			out = append(out, p.b[p.i:p.i+n]...)
			p.i += n
		}
	}
}

func (p *jparser) hex4() (rune, error) {
	if p.i+4 > len(p.b) {
		return 0, p.errf("short \\u escape")
	}
	n, err := strconv.ParseUint(string(p.b[p.i:p.i+4]), 16, 32)
	if err != nil {
		return 0, p.errf("invalid \\u escape")
	}
	p.i += 4
	return rune(n), nil
}

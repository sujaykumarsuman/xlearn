package harness

import (
	"errors"
	"math"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Canonical JSON (harness@1, README.md): no whitespace; integers in decimal; strings with
// only `"`, `\` and the controls < 0x20 escaped (`\"`, `\\`, `\u00xx` with lowercase hex),
// everything else raw UTF-8; object keys sorted by their UTF-8 bytes; ListNode as an
// array, TreeNode as a LeetCode level order with null holes and trailing nulls trimmed,
// GraphNode as 1-indexed adjacency lists. Floats are never canonical: Canonical refuses
// them, and the bytes-mode spelling (FormatFloat) is for frames compared by a float checker.

// ErrNotCanonical is returned by Canonical for a value holding a float.
var ErrNotCanonical = errors.New("harness: floats have no canonical bytes (compare with a float checker)")

// Canonical returns a value's canonical JSON. It fails for a value holding a float.
func Canonical(v Value) ([]byte, error) {
	return appendValue(nil, v, false)
}

// AppendJSON appends a value's JSON: canonical for everything but floats, which use the
// bytes-mode spelling FormatFloat (the fd-4 bytes of a float output).
func AppendJSON(dst []byte, v Value) []byte {
	b, _ := appendValue(dst, v, true)
	return b
}

func appendValue(dst []byte, v Value, floats bool) ([]byte, error) {
	switch v.Kind {
	case KindNull:
		return append(dst, "null"...), nil
	case KindInt:
		return strconv.AppendInt(dst, v.Int, 10), nil
	case KindFloat:
		if !floats {
			return dst, ErrNotCanonical
		}
		return append(dst, FormatFloat(v.Float)...), nil
	case KindBool:
		return strconv.AppendBool(dst, v.Bool), nil
	case KindString:
		return AppendString(dst, v.Str), nil
	default:
		dst = append(dst, '[')
		for i, e := range v.List {
			if i > 0 {
				dst = append(dst, ',')
			}
			var err error
			if dst, err = appendValue(dst, e, floats); err != nil {
				return dst, err
			}
		}
		return append(dst, ']'), nil
	}
}

// AppendString appends s as a canonical JSON string. s must be valid UTF-8 (the typed
// decoders guarantee it; an invalid byte is written as U+FFFD's bytes so the output is
// always valid JSON).
func AppendString(dst []byte, s string) []byte {
	const hex = "0123456789abcdef"
	dst = append(dst, '"')
	for i := 0; i < len(s); {
		c := s[i]
		switch {
		case c == '"' || c == '\\':
			dst = append(dst, '\\', c)
			i++
		case c < 0x20:
			dst = append(dst, '\\', 'u', '0', '0', hex[c>>4], hex[c&0xf])
			i++
		case c < utf8.RuneSelf:
			dst = append(dst, c)
			i++
		default:
			r, n := utf8.DecodeRuneInString(s[i:])
			if r == utf8.RuneError && n <= 1 {
				dst = utf8.AppendRune(dst, utf8.RuneError)
			} else {
				dst = append(dst, s[i:i+n]...)
			}
			i += n
		}
	}
	return append(dst, '"')
}

// FormatFloat is the one float spelling of every frame (input floats, and float outputs in
// bytes mode): the shortest round-trip form, strconv 'g' with -1 precision ("0.1", "3",
// "1e+21", "1e-07"). It is not canonical in the hashing sense: two float outputs are
// compared by a float checker, never by bytes.
func FormatFloat(f float64) string {
	if f == 0 {
		return "0" // no "-0"
	}
	return strconv.FormatFloat(f, 'g', -1, 64)
}

// CanonicalJSON re-encodes any JSON value canonically without a type: objects with sorted
// keys, arrays in order, strings as AppendString, integer literals in decimal and other
// numbers through FormatFloat. Duplicate keys, invalid UTF-8, non-finite or overflowing
// numbers and nesting deeper than MaxDepth are errors. It is the canonical form of the
// untyped parts of a case line (generator params) and of tests.lock.
func CanonicalJSON(raw []byte) ([]byte, error) {
	j, err := parseJSON(raw)
	if err != nil {
		return nil, err
	}
	return appendJV(nil, &j)
}

func appendJV(dst []byte, j *jv) ([]byte, error) {
	switch j.kind {
	case jNull:
		return append(dst, "null"...), nil
	case jBool:
		return strconv.AppendBool(dst, j.b), nil
	case jNum:
		return appendNumber(dst, j.s)
	case jStr:
		return AppendString(dst, j.s), nil
	case jArr:
		dst = append(dst, '[')
		for i := range j.arr {
			if i > 0 {
				dst = append(dst, ',')
			}
			var err error
			if dst, err = appendJV(dst, &j.arr[i]); err != nil {
				return dst, err
			}
		}
		return append(dst, ']'), nil
	default:
		idx := make([]int, len(j.keys))
		for i := range idx {
			idx[i] = i
		}
		sort.Slice(idx, func(a, b int) bool { return j.keys[idx[a]] < j.keys[idx[b]] })
		dst = append(dst, '{')
		for n, i := range idx {
			if n > 0 {
				dst = append(dst, ',')
			}
			dst = AppendString(dst, j.keys[i])
			dst = append(dst, ':')
			var err error
			if dst, err = appendJV(dst, &j.vals[i]); err != nil {
				return dst, err
			}
		}
		return append(dst, '}'), nil
	}
}

// appendNumber writes an integer literal as a decimal integer (int64, or uint64 for a
// large positive seed) and any other number through FormatFloat.
func appendNumber(dst []byte, s string) ([]byte, error) {
	if !strings.ContainsAny(s, ".eE") {
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return strconv.AppendInt(dst, n, 10), nil
		}
		if u, err := strconv.ParseUint(s, 10, 64); err == nil {
			return strconv.AppendUint(dst, u, 10), nil
		}
		return dst, errors.New("harness: integer out of the 64-bit range")
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsInf(f, 0) || math.IsNaN(f) {
		return dst, errors.New("harness: number out of the float64 range")
	}
	if f == math.Trunc(f) && math.Abs(f) < 1<<53 {
		return strconv.AppendInt(dst, int64(f), 10), nil
	}
	return append(dst, FormatFloat(f)...), nil
}

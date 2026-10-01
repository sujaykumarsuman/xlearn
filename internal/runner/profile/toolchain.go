package profile

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/sujaykumarsuman/xlearn/internal/platform/runnerapi"
)

// DefaultToolchainsDir is where the runner image keeps its pinned toolchains (m3-15's
// Dockerfile): the Go tarball at /opt/xl/go and the GOCACHE seed at /opt/xl/gocache. g++ and
// CPython come from Debian trixie at their system paths.
const DefaultToolchainsDir = "/opt/xl"

// ToolchainsDir is RUNNER_TOOLCHAINS_DIR (tests point it at their own layout), else
// DefaultToolchainsDir.
func ToolchainsDir() string {
	if d := strings.TrimSpace(os.Getenv("RUNNER_TOOLCHAINS_DIR")); d != "" {
		return d
	}
	return DefaultToolchainsDir
}

// Resolve returns p with symlinks resolved when it exists (binds keep the resolved path, so a
// toolchain's own paths and the GOCACHE seed's action ids line up), else p unchanged.
func Resolve(p string) string {
	if r, err := filepath.EvalSymlinks(p); err == nil {
		return r
	}
	return p
}

// Triplet is the Debian multiarch triplet of the running arch.
func Triplet() string {
	if runtime.GOARCH == "arm64" {
		return "aarch64-linux-gnu"
	}
	return "x86_64-linux-gnu"
}

// Loader is the dynamic loader's absolute path (the ELF interpreter of the system binaries).
func Loader() string {
	if runtime.GOARCH == "arm64" {
		return "/lib/ld-linux-aarch64.so.1"
	}
	return "/lib64/ld-linux-x86-64.so.2"
}

// SystemBinds are read-only binds of /usr and the merged-/usr links (/lib, /lib64) that
// exist: a dynamically linked Debian toolchain's view (the C++ and Python compile jails).
func SystemBinds() []Bind {
	return Existing(Bind{Source: "/usr"}, Bind{Source: "/lib"}, Bind{Source: "/lib64"})
}

// Existing keeps the binds whose source exists (a missing bind source would fail the jail).
func Existing(bs ...Bind) []Bind {
	var out []Bind
	for _, b := range bs {
		if _, err := os.Stat(b.Source); err == nil {
			out = append(out, b)
		}
	}
	return out
}

// ExistingPaths keeps the paths that exist.
func ExistingPaths(ps ...string) []string {
	var out []string
	for _, p := range ps {
		if _, err := os.Stat(p); err == nil {
			out = append(out, p)
		}
	}
	return out
}

// TreeHash reads every regular file under root (so a caller's page cache charges here, not to
// a case cgroup) and returns a hash over relative paths, modes, sizes, contents and symlink
// targets. root may be a single file.
func TreeHash(root string) (string, error) {
	h := sha256.New()
	buf := make([]byte, 1<<20)
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		info, err := d.Info()
		if err != nil {
			return err
		}
		switch {
		case info.Mode()&fs.ModeSymlink != 0:
			t, err := os.Readlink(p)
			if err != nil {
				return err
			}
			fmt.Fprintf(h, "L %s %s\n", rel, t)
		case info.Mode().IsRegular():
			f, err := os.Open(p)
			if err != nil {
				return err
			}
			fh := sha256.New()
			_, err = io.CopyBuffer(fh, f, buf)
			f.Close()
			if err != nil {
				return err
			}
			fmt.Fprintf(h, "F %s %o %d %x\n", rel, info.Mode().Perm(), info.Size(), fh.Sum(nil))
		case info.IsDir():
			fmt.Fprintf(h, "D %s %o\n", rel, info.Mode().Perm())
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// ---- diagnostics ----

// textDiagRE is a "file:line[:col]: msg" diagnostic line (gc, gcc's text output, ld).
var textDiagRE = regexp.MustCompile(`(?m)^(?:\./|/src/)?([A-Za-z0-9_]+\.[A-Za-z0-9_]+):(\d+)(?::(\d+))?: (.+)$`)

// TextDiags extracts "file:line[:col]: msg" diagnostics for the job's own file names: the
// generic parser, and every profile parser's fallback for plain-text lines.
func TextDiags(out []byte, names map[string]bool) []runnerapi.Diag {
	var diags []runnerapi.Diag
	for _, m := range textDiagRE.FindAllSubmatch(out, -1) {
		line, _ := strconv.Atoi(string(m[2]))
		col, _ := strconv.Atoi(string(m[3]))
		if d, ok := NewDiag(string(m[1]), line, col, string(m[4]), names); ok {
			diags = append(diags, d)
			if len(diags) == runnerapi.MaxDiags {
				break
			}
		}
	}
	return diags
}

// NewDiag builds a diagnostic for one of the job's files (a "./" or "/src/" prefix is
// dropped) with the message capped; ok is false for any other file.
func NewDiag(file string, line, col int, msg string, names map[string]bool) (runnerapi.Diag, bool) {
	file = strings.TrimPrefix(strings.TrimPrefix(file, "./"), "/src/")
	if !names[file] {
		return runnerapi.Diag{}, false
	}
	msg = strings.TrimSpace(msg)
	if len(msg) > runnerapi.MaxDiagMsgBytes {
		msg = msg[:runnerapi.MaxDiagMsgBytes]
	}
	return runnerapi.Diag{File: file, Line: line, Col: col, Msg: msg}, true
}

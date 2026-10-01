package goprofile

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/runner/profile"
)

// SeedTime is the fixed future mtime of every seed file and the seed's trim stamp. The go
// command refreshes a cache file's mtime when it is more than an hour old and trims files
// unused for days; a future mtime means it never tries either on the read-only seed, and the
// tree hash doesn't depend on when the seed was built.
var SeedTime = time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC)

// SeedRecipe versions BuildSeed (m3-15's image build keys on it).
const SeedRecipe = "gocache-seed@1"

// BuildSeed is the GOCACHE seed recipe (t3 §16.2 block 1, §16.4): `go build std` with the
// compile's exact toolchain, flags and environment (BuildFlags, BuildEnv; the seed only hits
// when they match) into a fresh GOCACHE at out, then every file gets SeedTime as its mtime,
// the trim stamp and each index entry's time are set to SeedTime (so two builds of the same
// toolchain give the same tree), and modes become world-readable. out must be absent or empty;
// on any failure nothing is left at out. It returns the seed's tree hash (profile.TreeHash,
// the same hash ProfileSHA covers). m3-15's Dockerfile runs it as `runner seed-gocache`.
func BuildSeed(goroot, out string) (string, error) {
	if goroot == "" || out == "" {
		return "", errors.New("seed: need a GOROOT and an output directory")
	}
	if ents, err := os.ReadDir(out); err == nil && len(ents) > 0 {
		return "", fmt.Errorf("seed: %s is not empty", out)
	} else if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return "", fmt.Errorf("seed: %w", err)
	}
	out = filepath.Clean(out)
	tmp, err := os.MkdirTemp(filepath.Dir(out), "."+filepath.Base(out)+".partial-")
	if err != nil {
		return "", fmt.Errorf("seed: %w", err)
	}
	defer os.RemoveAll(tmp)
	cache, work := filepath.Join(tmp, "cache"), filepath.Join(tmp, "work")
	for _, d := range []string{cache, work} {
		if err := os.Mkdir(d, 0o755); err != nil {
			return "", fmt.Errorf("seed: %w", err)
		}
	}
	cmd := exec.Command(filepath.Join(goroot, "bin", "go"), append(append([]string{"build"}, BuildFlags...), "std")...)
	cmd.Dir = work
	cmd.Env = BuildEnv(goroot, cache, work)
	var stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stderr, &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("seed: go build std: %v\n%s", err, stderr.Bytes())
	}
	if err := fixSeed(cache); err != nil {
		return "", err
	}
	if err := os.Remove(out); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return "", fmt.Errorf("seed: %w", err)
	}
	if err := os.Rename(cache, out); err != nil {
		return "", fmt.Errorf("seed: %w", err)
	}
	return profile.TreeHash(out)
}

// entrySize is the go command's cache index entry: "v1 <actionID> <outputID> <size> <time>\n"
// (cmd/go/internal/cache).
const entrySize = 2 + 1 + 64 + 1 + 64 + 1 + 20 + 1 + 20 + 1

func fixSeed(cache string) error {
	stamp := strconv.FormatInt(SeedTime.Unix(), 10)
	if err := os.WriteFile(filepath.Join(cache, "trim.txt"), []byte(stamp), 0o644); err != nil {
		return fmt.Errorf("seed: trim stamp: %w", err)
	}
	when := fmt.Sprintf("%20d", SeedTime.UnixNano())
	return filepath.WalkDir(cache, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if err := os.Chmod(p, 0o755); err != nil {
				return err
			}
		} else {
			if b, err := os.ReadFile(p); err == nil && len(b) == entrySize && bytes.HasPrefix(b, []byte("v1 ")) && b[entrySize-1] == '\n' {
				copy(b[entrySize-21:entrySize-1], when)
				if err := os.WriteFile(p, b, 0o644); err != nil {
					return err
				}
			}
			if err := os.Chmod(p, 0o644); err != nil {
				return err
			}
		}
		return os.Chtimes(p, SeedTime, SeedTime)
	})
}

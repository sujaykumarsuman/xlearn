// Package deploy has no code of its own: it holds the guard over the image build
// recipes (deploy/*.Dockerfile) so `go test ./...` covers them.
package deploy

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// probeVersion stands in for the release tag CI passes as --build-arg VERSION.
const probeVersion = "v0.0.0-versioncheck"

var (
	ldflagsRE = regexp.MustCompile(`-ldflags="([^"]*\$\{VERSION\}[^"]*)"`)
	targetRE  = regexp.MustCompile(`-o /out/\S+ (\./cmd/\S+)`)
)

// TestDockerfilesStampVersion builds each image's Go target with that Dockerfile's own
// -ldflags (VERSION = a probe) and asserts `<svc> -version` echoes the probe. The linker
// treats -X on a symbol that does not exist as a silent no-op, so a wrong path (e.g.
// `.../cmd/practice.version` instead of `main.version`) builds cleanly and ships a
// binary that reports "dev" — this is the only thing that catches it.
func TestDockerfilesStampVersion(t *testing.T) {
	if testing.Short() {
		t.Skip("builds every service binary")
	}
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go toolchain not on PATH")
	}
	dockerfiles, err := filepath.Glob("*.Dockerfile")
	if err != nil || len(dockerfiles) == 0 {
		t.Fatalf("no deploy/*.Dockerfile found (err=%v)", err)
	}
	binDir := t.TempDir()

	for _, df := range dockerfiles {
		svc := strings.TrimSuffix(df, ".Dockerfile")
		t.Run(svc, func(t *testing.T) {
			src, err := os.ReadFile(df)
			if err != nil {
				t.Fatal(err)
			}
			ldflags := ldflagsRE.FindStringSubmatch(string(src))
			target := targetRE.FindStringSubmatch(string(src))
			if ldflags == nil || target == nil {
				t.Fatalf("%s: want a `go build ... -ldflags=\"... -X <symbol>=${VERSION}\" -o /out/<bin> ./cmd/<svc>` line", df)
			}

			bin := filepath.Join(binDir, svc)
			build := exec.Command(goBin, "build", "-trimpath",
				"-ldflags", strings.ReplaceAll(ldflags[1], "${VERSION}", probeVersion),
				"-o", bin, target[1])
			build.Dir = ".." // the Dockerfiles build from the repo root
			build.Env = append(os.Environ(), "CGO_ENABLED=0")
			if out, err := build.CombinedOutput(); err != nil {
				t.Fatalf("go build %s: %v\n%s", target[1], err, out)
			}

			out, err := exec.Command(bin, "-version").Output()
			if err != nil {
				t.Fatalf("%s -version: %v", svc, err)
			}
			if got := strings.TrimSpace(string(out)); got != probeVersion {
				t.Errorf("%s: `%s -version` = %q, want %q: -ldflags %q does not stamp the binary "+
					"(a package-main var is addressed as main.<name>)", df, svc, got, probeVersion, ldflags[1])
			}
		})
	}
}

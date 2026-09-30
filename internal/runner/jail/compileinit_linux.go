//go:build linux

package jail

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"

	"golang.org/x/sys/unix"
)

// Compile-init exit codes (the spawner reads them from the wait status; no learner byte
// crosses). 1..95 is the build tool's own non-zero exit.
const (
	CompileExitNoStart  = 96  // the build tool couldn't start
	CompileExitSignaled = 97  // the build tool died of a signal
	CompileExitMissing  = 100 // the build succeeded but left no regular file at -out
	CompileExitTooBig   = 101 // the artifact is over -max
	CompileExitExport   = 102 // writing the export pipe failed
)

// ExportFd is the compile jail's export pipe (fd 3).
const ExportFd = 3

// CompileInit is `runner compile-init -out <path> -max <bytes> -- <build argv…>`: the compile
// jail's init and in-jail exporter (t3 §5.4 step 4). It runs as the compile UID, capless,
// under the compile seccomp filter, with /src (read-only) as its working directory. It runs
// the build as a child, then opens the build output with O_NOFOLLOW and streams its raw bytes
// (≤ max) over fd 3. The spawner copies those bytes into the artifact tmpfs without parsing
// them, before any learner code runs.
func CompileInit(args []string) int {
	fs := flag.NewFlagSet("compile-init", flag.ContinueOnError)
	out := fs.String("out", "", "artifact path inside the jail")
	max := fs.Int64("max", 64<<20, "artifact size cap in bytes")
	if err := fs.Parse(args); err != nil {
		return CompileExitNoStart
	}
	argv := fs.Args()
	if *out == "" || len(argv) == 0 {
		fmt.Fprintln(os.Stderr, "compile-init: need -out and a build command")
		return CompileExitNoStart
	}
	cmd := exec.Command(argv[0], argv[1:]...)
	// The jail has no /dev: pass our own fd 0 (the spawner's /dev/null) through, or os/exec
	// would try to open /dev/null itself.
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			if ee.ProcessState != nil && ee.ProcessState.ExitCode() < 0 {
				return CompileExitSignaled
			}
			code := ee.ExitCode()
			if code < 1 || code > 95 {
				code = 95
			}
			return code
		}
		fmt.Fprintln(os.Stderr, "compile-init: start build:", err)
		return CompileExitNoStart
	}
	fd, err := unix.Open(*out, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return CompileExitMissing
	}
	f := os.NewFile(uintptr(fd), "artifact")
	defer f.Close()
	fi, err := f.Stat()
	if err != nil || !fi.Mode().IsRegular() {
		return CompileExitMissing
	}
	if fi.Size() > *max {
		return CompileExitTooBig
	}
	export := os.NewFile(ExportFd, "export")
	n, err := io.Copy(export, io.LimitReader(f, *max+1))
	if err != nil {
		return CompileExitExport
	}
	if n > *max {
		return CompileExitTooBig
	}
	if err := export.Close(); err != nil {
		return CompileExitExport
	}
	return 0
}

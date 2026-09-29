// sigsys makes one dangerous syscall named by its input (bpf, io_uring_setup, unshare with
// CLONE_NEWUSER, mount, ptrace, keyctl): the exec filter kills it with SIGSYS (RE, counted),
// and the runner rotates. "survived" on fd 4 means the filter let it through.
package main

import (
	"io"
	"os"
	"runtime"
	"strings"
	"syscall"
)

// Numbers per arch: amd64, arm64.
var calls = map[string][2]uintptr{
	"bpf":            {321, 280},
	"io_uring_setup": {425, 425},
	"unshare":        {272, 97},
	"mount":          {165, 40},
	"ptrace":         {101, 117},
	"keyctl":         {250, 219},
}

func main() {
	in, _ := io.ReadAll(os.NewFile(3, "input"))
	nrs, ok := calls[strings.TrimSpace(string(in))]
	if !ok {
		os.Exit(2)
	}
	nr := nrs[0]
	if runtime.GOARCH == "arm64" {
		nr = nrs[1]
	}
	var a1 uintptr
	if strings.TrimSpace(string(in)) == "unshare" {
		a1 = 0x10000000 // CLONE_NEWUSER
	}
	syscall.RawSyscall6(nr, a1, 0, 0, 0, 0, 0)
	os.NewFile(4, "result").Write([]byte("survived"))
}

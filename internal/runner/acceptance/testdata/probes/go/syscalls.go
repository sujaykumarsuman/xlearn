package main

// Section C (syscalls), Go: arg names one call from t3 §2.4 A1's dangerous set (or clone with a
// namespace flag). The exec filter must kill it with SIGSYS; "survived: ..." on fd 4 means the
// filter let it through (a failure, whatever the call then returned). Numbers are per arch
// (amd64, arm64): the jail has no /proc and x/sys is not on the compile path.

import (
	"fmt"
	"os"
	"runtime"
	"syscall"
	"unsafe"
)

var nrs = map[string][2]uintptr{
	"io_uring_setup":   {425, 425},
	"bpf":              {321, 280},
	"perf_event_open":  {298, 241},
	"userfaultfd":      {323, 282},
	"keyctl":           {250, 219},
	"add_key":          {248, 217},
	"ptrace":           {101, 117},
	"process_vm_readv": {310, 270},
	"mount":            {165, 40},
	"unshare_newuser":  {272, 97},
	"setns":            {308, 268},
	"clone_newns":      {56, 220},
	"socket_netlink":   {41, 198},
}

func cstr(s string) uintptr {
	b := append([]byte(s), 0)
	return uintptr(unsafe.Pointer(&b[0]))
}

func probe(arg string) string {
	nr, ok := nrs[arg]
	if !ok {
		return "bad probe " + arg
	}
	n := nr[0]
	if runtime.GOARCH == "arm64" {
		n = nr[1]
	}
	buf := make([]byte, 512)
	p := uintptr(unsafe.Pointer(&buf[0]))
	var a [6]uintptr
	switch arg {
	case "io_uring_setup":
		a = [6]uintptr{1, p}
	case "bpf":
		a = [6]uintptr{0, p, 72} // BPF_MAP_CREATE
	case "perf_event_open":
		a = [6]uintptr{p, 0, ^uintptr(0), ^uintptr(0), 0} // pid 0, cpu -1, group -1
	case "userfaultfd":
		a = [6]uintptr{0x80000} // O_CLOEXEC
	case "keyctl":
		a = [6]uintptr{0, ^uintptr(3)} // KEYCTL_GET_KEYRING_ID, KEY_SPEC_USER_KEYRING
	case "add_key":
		a = [6]uintptr{cstr("user"), cstr("xl-probe"), cstr("x"), 1, ^uintptr(1)} // KEY_SPEC_PROCESS_KEYRING
	case "ptrace":
		a = [6]uintptr{0} // PTRACE_TRACEME
	case "process_vm_readv":
		pid, _, _ := syscall.RawSyscall(syscall.SYS_GETPID, 0, 0, 0)
		iov := [2]uintptr{p, 8}
		a = [6]uintptr{pid, uintptr(unsafe.Pointer(&iov[0])), 1, uintptr(unsafe.Pointer(&iov[0])), 1, 0}
	case "mount":
		a = [6]uintptr{cstr("none"), cstr("/w"), cstr("tmpfs"), 0, 0}
	case "unshare_newuser":
		a = [6]uintptr{0x10000000} // CLONE_NEWUSER
	case "setns":
		a = [6]uintptr{^uintptr(0), 0}
	case "clone_newns":
		a = [6]uintptr{0x00020000 | 17} // CLONE_NEWNS | SIGCHLD, no new stack: EPERM without caps
	case "socket_netlink":
		a = [6]uintptr{16, 3, 0} // AF_NETLINK, SOCK_RAW, NETLINK_ROUTE
	}
	os.Stderr.WriteString("calling " + arg + "\n")
	r1, _, errno := syscall.RawSyscall6(n, a[0], a[1], a[2], a[3], a[4], a[5])
	return fmt.Sprintf("survived: %s rc=%d errno=%d", arg, int64(r1), int(errno))
}

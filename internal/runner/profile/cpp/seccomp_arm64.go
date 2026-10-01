//go:build arm64

package cpp

// The arm64 lists are for dev VMs only: NEVER in a release image (the release image is amd64,
// m3-15; a guard test fails if an amd64 build links this file). They are the amd64 names
// translated to arm64 (no arch_prctl; access, chmod, readlink and unlink become their *at
// forms) plus abort's calls (as amd64). The delta was searched on an arm64 dev VM (m3-04: the
// whole runner-it lane KILL-default, every kill read from the kernel's audit records): none.
// KILL-default, like amd64.

var execAllow = []string{
	"brk", "execve", "exit_group", "fstat", "futex", "getpid", "getrandom", "gettid",
	"lseek", "mmap", "mprotect", "munmap", "prlimit64", "read", "readlinkat", "rseq",
	"rt_sigaction", "rt_sigprocmask", "set_robust_list", "set_tid_address", "tgkill", "write", "writev",
}

var compileAllow = []string{
	"brk", "clone", "close", "dup", "execve", "exit_group", "faccessat", "faccessat2", "fchmodat",
	"fcntl", "fstat", "futex", "getcwd", "getrandom", "getrusage", "ioctl", "lseek", "mmap",
	"mprotect", "mremap", "munmap", "newfstatat", "openat", "pread64", "prlimit64", "read",
	"readlinkat", "rseq", "rt_sigaction", "rt_sigprocmask", "set_robust_list", "set_tid_address",
	"sysinfo", "umask", "unlinkat", "wait4", "write",
}

// spk02Exec and execAdditions are amd64's (the invariant test checks only amd64 against t3).
var (
	spk02Exec     []string
	execAdditions []string
)

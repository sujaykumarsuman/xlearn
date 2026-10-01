//go:build arm64

package python

// The arm64 lists are for dev VMs only: NEVER in a release image (the release image is amd64,
// m3-15; a guard test fails if an amd64 build links this file). They are the amd64 names
// translated to arm64 (no arch_prctl; access, open, readlink, mkdir and rename become their
// *at forms). The delta was searched on an arm64 dev VM (m3-04: the whole runner-it lane
// KILL-default, every kill read from the kernel's audit records): none. KILL-default, like
// amd64.

var execAllow = []string{
	"brk", "clone", "close", "execve", "exit", "exit_group", "faccessat", "fcntl",
	"fstat", "futex", "getcwd", "getdents64", "getegid", "geteuid", "getgid", "getrandom", "gettid",
	"getuid", "ioctl", "lseek", "madvise", "mmap", "mprotect", "mremap", "munmap", "newfstatat",
	"openat", "pread64", "prlimit64", "read", "readlinkat", "rseq", "rt_sigaction",
	"rt_sigprocmask", "set_robust_list", "set_tid_address", "write",
}

var compileAllow = []string{
	"brk", "close", "execve", "exit_group", "faccessat", "fcntl", "fstat", "futex",
	"getcwd", "getdents64", "getegid", "geteuid", "getgid", "getrandom", "gettid", "getuid", "ioctl",
	"lseek", "mkdirat", "mmap", "mprotect", "mremap", "munmap", "newfstatat", "openat",
	"pread64", "prlimit64", "read", "readlinkat", "renameat", "rseq", "rt_sigaction", "rt_sigprocmask",
	"set_robust_list", "set_tid_address", "write",
}

// spk02Exec and execAdditions are amd64's (the invariant test checks only amd64 against t3).
var (
	spk02Exec     []string
	execAdditions []string
)

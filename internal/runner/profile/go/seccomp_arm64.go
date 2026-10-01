//go:build arm64

package goprofile

// The arm64 lists are for dev VMs only: NEVER in a release image (the release image is amd64,
// m3-15; a guard test fails if an amd64 build links this file). They are the amd64 names that
// exist on arm64 (arm64 has no arch_prctl). The delta was searched on an arm64 dev VM (m3-04:
// the whole runner-it lane KILL-default, every kill read from the kernel's audit records):
// none. The exec filter is KILL-default, like amd64's.

var execAllow = []string{
	"clone", "close", "epoll_create1", "epoll_ctl", "epoll_pwait", "eventfd2", "execve",
	"exit_group", "fcntl", "futex", "getpid", "gettid", "madvise", "mmap", "nanosleep", "openat",
	"prctl", "prlimit64", "read", "rt_sigaction", "rt_sigprocmask", "rt_sigreturn",
	"sched_getaffinity", "sched_yield", "sigaltstack", "tgkill", "write",
}

var compileAllow = []string{
	"chdir", "clone", "close", "copy_file_range", "dup3", "epoll_create1",
	"epoll_ctl", "epoll_pwait", "eventfd2", "execve", "exit_group", "faccessat2", "fallocate",
	"fchmodat", "fcntl", "flock", "fstat", "ftruncate", "futex", "getcwd", "getdents64", "getpid",
	"gettid", "lseek", "madvise", "mkdirat", "mmap", "munmap", "nanosleep", "newfstatat",
	"openat", "pidfd_open", "pidfd_send_signal", "pipe2", "prctl", "pread64", "prlimit64",
	"pwrite64", "read", "readlinkat", "renameat", "rt_sigaction", "rt_sigprocmask",
	"rt_sigreturn", "sched_getaffinity", "sched_yield", "sigaltstack", "tgkill", "uname",
	"unlinkat", "utimensat", "waitid", "write",
}

// spk02Exec and execAdditions are amd64's (the invariant test checks only amd64 against t3).
var (
	spk02Exec     []string
	execAdditions []string
)

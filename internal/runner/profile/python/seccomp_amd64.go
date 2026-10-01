package python

// execAllow is spk-02's amd64 exec allowlist for `python3` 3.13 (dynamic), copied verbatim
// from t3 §16.2 (39 names, sorted; 0 unexpected SIGSYS under KILL), plus the justified
// additions marked below. clone is listed as a name; the filter allows it only with
// CLONE_THREAD (threads yes, os.fork() dies). getcwd (¶) was seen under `python3 -c`; ioctl
// (TCGETS on stdin) and legacy open are needed.
var execAllow = []string{
	"access", "arch_prctl", "brk", "clone", "close", "execve", "exit", "exit_group", "fcntl",
	"fstat", "futex", "getcwd", "getdents64", "getegid", "geteuid", "getgid", "getrandom", "gettid",
	"getuid", "ioctl", "lseek", "madvise", "mmap", "mprotect", "mremap", "munmap", "newfstatat",
	"open", "openat", "pread64", "prlimit64", "read", "readlink", "rseq", "rt_sigaction",
	"rt_sigprocmask", "set_robust_list", "set_tid_address", "write",
}

// compileAllow is spk-02's logged amd64 `python3 -m py_compile` compile set, copied verbatim
// from t3 §16.2 (38 names). The profile unions it with the go build set for the runner's
// compile init.
var compileAllow = []string{
	"access", "arch_prctl", "brk", "close", "execve", "exit_group", "fcntl", "fstat", "futex",
	"getcwd", "getdents64", "getegid", "geteuid", "getgid", "getrandom", "gettid", "getuid", "ioctl",
	"lseek", "mkdir", "mmap", "mprotect", "mremap", "munmap", "newfstatat", "open", "openat",
	"pread64", "prlimit64", "read", "readlink", "rename", "rseq", "rt_sigaction", "rt_sigprocmask",
	"set_robust_list", "set_tid_address", "write",
}

// spk02Exec is t3 §16.2's python exec list exactly as published.
var spk02Exec = []string{
	"access", "arch_prctl", "brk", "clone", "close", "execve", "exit", "exit_group", "fcntl",
	"fstat", "futex", "getcwd", "getdents64", "getegid", "geteuid", "getgid", "getrandom", "gettid",
	"getuid", "ioctl", "lseek", "madvise", "mmap", "mprotect", "mremap", "munmap", "newfstatat",
	"open", "openat", "pread64", "prlimit64", "read", "readlink", "rseq", "rt_sigaction",
	"rt_sigprocmask", "set_robust_list", "set_tid_address", "write",
}

// execAdditions are the justified additions to spk02Exec (decisions log, m3-04).
var execAdditions = []string{}

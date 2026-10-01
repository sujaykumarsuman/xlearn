package seccomp

import "sort"

// Dangerous is the set every runner filter kills (t3 §2.4 A1, m3-04 task 5): io_uring, bpf,
// perf, userfaultfd, the keyring, ptrace and process_vm_*, the whole mount family including the
// new mount API, unshare/setns, socket, and the pipe-splicing family (splice, vmsplice and tee:
// A1 denies splice/vmsplice, tee is the same pipe-buffer family). Namespace flags on clone are
// handled by Policy.CloneThreadOnly; clone3 returns ENOSYS. No exec allowlist may name any of
// these (each profile's invariant test), and no compile filter allows one: spk-02's compile
// sets (t3 §16.2) show no toolchain calling splice, vmsplice or tee, so none of them is
// downgraded to ENOSYS anywhere.
var Dangerous = []string{
	"add_key", "bpf", "delete_module", "finit_module", "fsconfig", "fsmount", "fsopen", "fspick",
	"init_module", "io_uring_enter", "io_uring_register", "io_uring_setup", "kexec_file_load",
	"kexec_load", "keyctl", "mount", "mount_setattr", "move_mount", "open_by_handle_at",
	"open_tree", "perf_event_open", "pivot_root", "process_vm_readv", "process_vm_writev",
	"ptrace", "reboot", "request_key", "setns", "socket", "splice", "swapoff", "swapon", "tee",
	"umount2", "unshare", "userfaultfd", "vmsplice",
}

// CompileInitExtra are the calls the runner's own in-jail compile init (a Go program that runs
// the compiler as a child, then exports the artifact) makes beyond spk-02's `go build` set,
// which every compile filter includes for the init's Go runtime.
var CompileInitExtra = []string{"exit", "getrandom", "kill", "wait4"}

// Without returns names minus drop.
func Without(names []string, drop ...string) []string {
	skip := map[string]bool{}
	for _, d := range drop {
		skip[d] = true
	}
	out := make([]string, 0, len(names))
	for _, n := range names {
		if !skip[n] {
			out = append(out, n)
		}
	}
	return out
}

// Union returns the sorted, de-duplicated union of lists.
func Union(lists ...[]string) []string {
	seen := map[string]bool{}
	var out []string
	for _, l := range lists {
		for _, n := range l {
			if !seen[n] {
				seen[n] = true
				out = append(out, n)
			}
		}
	}
	sort.Strings(out)
	return out
}

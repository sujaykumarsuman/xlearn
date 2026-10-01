// Package seccomp builds classic-BPF seccomp filters from syscall-name lists (t3 §2.4 A1,
// §16.2). The assembler is portable and table-tested with a small BPF interpreter; the
// per-arch name tables and the kernel conversion are Linux-only.
//
// Every filter starts with the fixed rules spk-02 validated (t3 §16.2 block 2):
//   - a foreign audit arch, or an x32 syscall number (nr ≥ 0x40000000 on x86_64) → KILL_PROCESS;
//   - the dangerous set → KILL_PROCESS, whatever the default;
//   - clone3 → ERRNO(ENOSYS), so glibc and Go fall back to clone;
//   - optionally clone only with CLONE_THREAD (threads yes, fork no) and no namespace flag,
//     and prctl only with PR_SET_VMA;
//
// then the allowlist, then the default action (KILL_PROCESS for exec filters, ERRNO(ENOSYS)
// for compile filters). The per-profile lists live with their profiles
// (internal/runner/profile/<p>/seccomp_<arch>.go, m3-04); this package keeps the dangerous set.
package seccomp

import (
	"fmt"
	"sort"
)

// Action is a seccomp return value.
type Action uint32

// Seccomp return actions (linux/seccomp.h).
const (
	ActKillProcess Action = 0x80000000
	ActKillThread  Action = 0x00000000
	ActTrap        Action = 0x00030000
	actErrnoBase   Action = 0x00050000
	ActLog         Action = 0x7ffc0000
	ActAllow       Action = 0x7fff0000
)

// ENOSYS is errno 38 on every Linux arch the runner builds for.
const ENOSYS = 38

// Errno returns ERRNO(e).
func Errno(e uint16) Action { return actErrnoBase | Action(e) }

// Arg-check constants (linux/sched.h, linux/prctl.h).
const (
	CloneThread = 0x00010000
	// CloneNSMask is every namespace flag clone(2) takes.
	CloneNSMask = 0x00000080 | // CLONE_NEWTIME
		0x00020000 | // CLONE_NEWNS
		0x02000000 | // CLONE_NEWCGROUP
		0x04000000 | // CLONE_NEWUTS
		0x08000000 | // CLONE_NEWIPC
		0x10000000 | // CLONE_NEWUSER
		0x20000000 | // CLONE_NEWPID
		0x40000000 // CLONE_NEWNET
	PRSetVMA = 0x53564d41
	// x32ABIBit marks an x32 syscall number on x86_64.
	x32ABIBit = 0x40000000
)

// Arch is one architecture's audit value and syscall-number table.
type Arch struct {
	Name    string
	Audit   uint32
	X32Bit  bool // reject nr ≥ 0x40000000 (x86_64's x32 ABI)
	Numbers map[string]uint32
}

// Policy is a filter described by names.
type Policy struct {
	// Default is the action for anything not listed.
	Default Action
	// Allow is the allowlist.
	Allow []string
	// Kill are always KILL_PROCESS (the dangerous set), checked before Allow.
	Kill []string
	// Clone3ENOSYS makes clone3 fail with ENOSYS (checked before Allow).
	Clone3ENOSYS bool
	// CloneThreadOnly allows clone only with CLONE_THREAD and no namespace flag; any other
	// clone gets Default. "clone" must not also be in Allow.
	CloneThreadOnly bool
	// PrctlSetVMAOnly allows prctl only with arg0 == PR_SET_VMA; any other prctl gets Default.
	// "prctl" must not also be in Allow.
	PrctlSetVMAOnly bool
}

// Instr is one classic-BPF instruction (struct sock_filter), kept portable so the assembler
// is testable everywhere.
type Instr struct {
	Code uint16
	Jt   uint8
	Jf   uint8
	K    uint32
}

// BPF opcodes used here.
const (
	opLdAbsW  = 0x20 // BPF_LD | BPF_W | BPF_ABS
	opJeqK    = 0x15 // BPF_JMP | BPF_JEQ | BPF_K
	opJgeK    = 0x35 // BPF_JMP | BPF_JGE | BPF_K
	opJsetK   = 0x45 // BPF_JMP | BPF_JSET | BPF_K
	opRetK    = 0x06 // BPF_RET | BPF_K
	offNr     = 0
	offArch   = 4
	offArg0Lo = 16 // args[0], little endian (amd64, arm64)
	offArg0Hi = 20
)

// MaxInstrs is the kernel's BPF_MAXINSNS.
const MaxInstrs = 4096

// Assemble compiles p for arch a. Unresolved lists the names a has no number for (they are
// skipped); a strict caller (amd64) treats a non-empty list as an error.
func Assemble(p Policy, a Arch) (prog []Instr, unresolved []string, err error) {
	if a.Audit == 0 || len(a.Numbers) == 0 {
		return nil, nil, fmt.Errorf("seccomp: no syscall table for arch %q", a.Name)
	}
	allow := uniq(p.Allow)
	kill := uniq(p.Kill)
	inKill := map[string]bool{}
	for _, n := range kill {
		inKill[n] = true
	}
	for _, n := range allow {
		if inKill[n] {
			return nil, nil, fmt.Errorf("seccomp: %q is in both the allow and kill lists", n)
		}
		if (n == "clone" && p.CloneThreadOnly) || (n == "prctl" && p.PrctlSetVMAOnly) || (n == "clone3" && p.Clone3ENOSYS) {
			return nil, nil, fmt.Errorf("seccomp: %q has an argument rule; don't list it in Allow", n)
		}
	}
	resolve := func(n string) (uint32, bool) {
		nr, ok := a.Numbers[n]
		if !ok {
			unresolved = append(unresolved, n)
		}
		return nr, ok
	}
	ret := func(act Action) Instr { return Instr{Code: opRetK, K: uint32(act)} }

	prog = append(prog,
		Instr{Code: opLdAbsW, K: offArch},
		Instr{Code: opJeqK, Jt: 1, Jf: 0, K: a.Audit},
		ret(ActKillProcess),
		Instr{Code: opLdAbsW, K: offNr},
	)
	if a.X32Bit {
		prog = append(prog, Instr{Code: opJgeK, Jt: 0, Jf: 1, K: x32ABIBit}, ret(ActKillProcess))
	}
	for _, n := range kill {
		if nr, ok := resolve(n); ok {
			prog = append(prog, Instr{Code: opJeqK, Jt: 0, Jf: 1, K: nr}, ret(ActKillProcess))
		}
	}
	if p.Clone3ENOSYS {
		if nr, ok := resolve("clone3"); ok {
			prog = append(prog, Instr{Code: opJeqK, Jt: 0, Jf: 1, K: nr}, ret(Errno(ENOSYS)))
		}
	}
	if p.CloneThreadOnly {
		if nr, ok := resolve("clone"); ok {
			prog = append(prog,
				Instr{Code: opJeqK, Jt: 0, Jf: 5, K: nr},
				Instr{Code: opLdAbsW, K: offArg0Lo},
				Instr{Code: opJsetK, Jt: 0, Jf: 2, K: CloneThread},
				Instr{Code: opJsetK, Jt: 1, Jf: 0, K: CloneNSMask},
				ret(ActAllow),
				ret(p.Default),
			)
		}
	}
	if p.PrctlSetVMAOnly {
		if nr, ok := resolve("prctl"); ok {
			prog = append(prog,
				Instr{Code: opJeqK, Jt: 0, Jf: 6, K: nr},
				Instr{Code: opLdAbsW, K: offArg0Hi},
				Instr{Code: opJeqK, Jt: 0, Jf: 3, K: 0},
				Instr{Code: opLdAbsW, K: offArg0Lo},
				Instr{Code: opJeqK, Jt: 0, Jf: 1, K: PRSetVMA},
				ret(ActAllow),
				ret(p.Default),
			)
		}
	}
	for _, n := range allow {
		if nr, ok := resolve(n); ok {
			prog = append(prog, Instr{Code: opJeqK, Jt: 0, Jf: 1, K: nr}, ret(ActAllow))
		}
	}
	prog = append(prog, ret(p.Default))
	if len(prog) > MaxInstrs {
		return nil, nil, fmt.Errorf("seccomp: %d instructions, kernel cap %d", len(prog), MaxInstrs)
	}
	sort.Strings(unresolved)
	return prog, unresolved, nil
}

func uniq(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, n := range in {
		if !seen[n] {
			seen[n] = true
			out = append(out, n)
		}
	}
	sort.Strings(out)
	return out
}

// ExecDefault is the default action of a per-case (exec) filter: KILL_PROCESS, on every arch
// and in every mode. m3-04 retired m3-03's dev-only arm64 LOG switch: the arm64 dev-VM lists
// (each profile's seccomp_arm64.go, never in a release image) are KILL-default like amd64's;
// LOG was only how their delta was found.
func ExecDefault() Action { return ActKillProcess }

// CompileDefault is the default action of a compile filter: ERRNO(ENOSYS) (t3 §5.4), with
// the dangerous set still KILL.
func CompileDefault() Action { return Errno(ENOSYS) }

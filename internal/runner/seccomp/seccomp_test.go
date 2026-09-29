package seccomp

import (
	"encoding/binary"
	"runtime"
	"testing"
)

// fakeAMD64 is a small x86_64-numbered table so the assembler is testable on every platform.
var fakeAMD64 = Arch{Name: "fake-amd64", Audit: 0xC000003E, X32Bit: true, Numbers: map[string]uint32{
	"read": 0, "write": 1, "openat": 257, "clone": 56, "clone3": 435, "prctl": 157, "bpf": 321,
	"socket": 41, "exit_group": 231, "mount": 165, "unshare": 272, "getpid": 39, "io_uring_setup": 425,
}}

// data is a seccomp_data: nr, arch, ip, args[6] (little endian).
func data(nr, arch uint32, args ...uint64) []byte {
	b := make([]byte, 64)
	binary.LittleEndian.PutUint32(b[0:], nr)
	binary.LittleEndian.PutUint32(b[4:], arch)
	for i, a := range args {
		binary.LittleEndian.PutUint64(b[16+8*i:], a)
	}
	return b
}

// eval is a tiny classic-BPF interpreter for the opcodes Assemble emits.
func eval(t *testing.T, prog []Instr, d []byte) Action {
	t.Helper()
	var acc uint32
	for pc := 0; pc < len(prog); {
		in := prog[pc]
		switch in.Code {
		case opLdAbsW:
			acc = binary.LittleEndian.Uint32(d[in.K:])
			pc++
		case opJeqK, opJgeK, opJsetK:
			var ok bool
			switch in.Code {
			case opJeqK:
				ok = acc == in.K
			case opJgeK:
				ok = acc >= in.K
			default:
				ok = acc&in.K != 0
			}
			if ok {
				pc += 1 + int(in.Jt)
			} else {
				pc += 1 + int(in.Jf)
			}
		case opRetK:
			return Action(in.K)
		default:
			t.Fatalf("unknown opcode %#x at %d", in.Code, pc)
		}
	}
	t.Fatal("fell off the end of the program")
	return 0
}

func execPolicy() Policy {
	return Policy{
		Default:         ActKillProcess,
		Allow:           Without([]string{"read", "write", "openat", "exit_group", "getpid", "clone", "prctl"}, "clone", "prctl"),
		Kill:            []string{"bpf", "mount", "unshare", "io_uring_setup", "socket"},
		Clone3ENOSYS:    true,
		CloneThreadOnly: true,
		PrctlSetVMAOnly: true,
	}
}

func TestAssembleSemantics(t *testing.T) {
	prog, unresolved, err := Assemble(execPolicy(), fakeAMD64)
	if err != nil {
		t.Fatal(err)
	}
	if len(unresolved) != 0 {
		t.Fatalf("unresolved %v", unresolved)
	}
	const x86 = 0xC000003E
	cases := []struct {
		name string
		d    []byte
		want Action
	}{
		{"read allowed", data(0, x86), ActAllow},
		{"getpid allowed", data(39, x86), ActAllow},
		{"unlisted → default kill", data(999, x86), ActKillProcess},
		{"bpf killed", data(321, x86), ActKillProcess},
		{"socket killed", data(41, x86), ActKillProcess},
		{"clone3 → ENOSYS", data(435, x86), Errno(ENOSYS)},
		{"thread clone allowed", data(56, x86, 0x003d0f00), ActAllow},
		{"fork-style clone killed", data(56, x86, 0x01200011), ActKillProcess},
		{"thread clone with CLONE_NEWNET killed", data(56, x86, 0x003d0f00|0x40000000), ActKillProcess},
		{"thread clone with CLONE_NEWUSER killed", data(56, x86, 0x003d0f00|0x10000000), ActKillProcess},
		{"prctl(PR_SET_VMA) allowed", data(157, x86, PRSetVMA), ActAllow},
		{"prctl(PR_SET_NO_NEW_PRIVS) killed", data(157, x86, 38), ActKillProcess},
		{"prctl with a high-word PR_SET_VMA killed", data(157, x86, 1<<32|PRSetVMA), ActKillProcess},
		{"x32 read killed", data(0x40000000, x86), ActKillProcess},
		{"ia32 arch killed", data(0, 0x40000003), ActKillProcess},
		{"aarch64 arch killed", data(0, 0xC00000B7), ActKillProcess},
	}
	for _, c := range cases {
		if got := eval(t, prog, c.d); got != c.want {
			t.Errorf("%s: got %#x, want %#x", c.name, uint32(got), uint32(c.want))
		}
	}
}

func TestCompileDefaultIsENOSYS(t *testing.T) {
	p := Policy{Default: CompileDefault(), Allow: []string{"read", "clone"}, Kill: []string{"socket", "bpf"}, Clone3ENOSYS: true}
	prog, _, err := Assemble(p, fakeAMD64)
	if err != nil {
		t.Fatal(err)
	}
	const x86 = 0xC000003E
	if got := eval(t, prog, data(999, x86)); got != Errno(ENOSYS) {
		t.Errorf("unlisted = %#x, want ENOSYS", uint32(got))
	}
	if got := eval(t, prog, data(41, x86)); got != ActKillProcess {
		t.Errorf("socket = %#x, want KILL even under an ENOSYS default", uint32(got))
	}
	if got := eval(t, prog, data(56, x86, 0x01200011)); got != ActAllow {
		t.Errorf("a compile-jail fork must be allowed (posix_spawn), got %#x", uint32(got))
	}
}

func TestAssembleRejectsContradictions(t *testing.T) {
	if _, _, err := Assemble(Policy{Default: ActKillProcess, Allow: []string{"bpf"}, Kill: []string{"bpf"}}, fakeAMD64); err == nil {
		t.Error("a name in both lists must be refused")
	}
	if _, _, err := Assemble(Policy{Default: ActKillProcess, Allow: []string{"clone"}, CloneThreadOnly: true}, fakeAMD64); err == nil {
		t.Error("clone in Allow with the thread-only rule must be refused")
	}
	if _, _, err := Assemble(Policy{Default: ActKillProcess}, Arch{Name: "none"}); err == nil {
		t.Error("an empty arch must be refused")
	}
	_, unresolved, err := Assemble(Policy{Default: ActKillProcess, Allow: []string{"read", "no_such_call"}}, fakeAMD64)
	if err != nil || len(unresolved) != 1 || unresolved[0] != "no_such_call" {
		t.Errorf("unresolved = %v, %v", unresolved, err)
	}
}

func TestExecDefault(t *testing.T) {
	if ExecDefault("amd64", true) != ActKillProcess {
		t.Error("amd64 exec filters are KILL-default in every build and mode")
	}
	if ExecDefault("arm64", false) != ActKillProcess {
		t.Error("arm64 in prod mode is KILL-default")
	}
}

func TestGoListsResolveNatively(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("the amd64 lists are authoritative only on linux/amd64")
	}
	for name, names := range map[string][]string{
		"GoExecAMD64": GoExecAMD64, "GoBuildAMD64": GoBuildAMD64, "Dangerous": Dangerous, "CompileInitExtra": CompileInitExtra,
	} {
		_, unresolved, err := Assemble(Policy{Default: ActKillProcess, Allow: names}, Native())
		if err != nil || len(unresolved) != 0 {
			t.Errorf("%s: unresolved %v (%v)", name, unresolved, err)
		}
	}
	if len(GoExecAMD64) != 27 || len(GoBuildAMD64) != 54 {
		t.Errorf("spk-02's lists are 27 and 54 names; got %d and %d", len(GoExecAMD64), len(GoBuildAMD64))
	}
}

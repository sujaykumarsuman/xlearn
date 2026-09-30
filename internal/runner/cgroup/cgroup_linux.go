//go:build linux

// Package cgroup owns the runner's cgroup v2 layout inside the container's writable cgroup
// namespace (t3 §7.1):
//
//	/sys/fs/cgroup            (container: cpu.max 2 · memory.max 3 GiB, set by the pod)
//	├─ runner/                spawner + front
//	└─ slots/                 +cpu +memory +pids
//	   ├─ s0/  cpu.max 1 CPU · memory.max 1.25 GiB · swap 0 · pids.max 512
//	   │   ├─ job/ ├─ compile/  └─ case-K/  (memory.max = mem + baseline · oom.group 1 · swap 0 · pids.max)
//	   │   └─ canary/
//	   └─ s1/  same
//
// INV-14: 2 × 1.25 GiB + runner/ (≤ ~300 MiB) < 3 GiB, so a learner OOM always lands in a case
// or compile cgroup, never in the container. No memory.high and no io.max (every writable path
// is tmpfs).
package cgroup

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sys/unix"

	"github.com/sujaykumarsuman/xlearn/internal/runner/measure"
)

// Slot limits (t3 §7.1).
const (
	SlotCPUMax    = "100000 100000" // 1 CPU
	SlotMemoryMax = 1280 << 20      // 1.25 GiB
	SlotPidsMax   = 512
	Slots         = 2
)

// Manager is the layout rooted at the container's cgroup mount.
type Manager struct {
	Root string
}

// Path joins elements under the root.
func (m *Manager) Path(elem ...string) string {
	return filepath.Join(append([]string{m.Root}, elem...)...)
}

// Slot is slots/s<i>.
func (m *Manager) Slot(i int) string { return m.Path("slots", fmt.Sprintf("s%d", i)) }

// Runner is runner/.
func (m *Manager) Runner() string { return m.Path("runner") }

// Setup builds the layout: move this process into runner/, enable the controllers, wipe and
// recreate slots/. The container's cgroup must hold no other process (the no-internal-process
// rule); in the pod the spawner is PID 1 and alone.
func (m *Manager) Setup(pid int) error {
	if err := os.MkdirAll(m.Runner(), 0o755); err != nil {
		return fmt.Errorf("cgroup: mkdir runner/: %w", err)
	}
	if err := Write(m.Runner(), "cgroup.procs", strconv.Itoa(pid)); err != nil {
		return fmt.Errorf("cgroup: move self into runner/: %w", err)
	}
	procs, err := os.ReadFile(m.Path("cgroup.procs"))
	if err != nil {
		return err
	}
	if n := len(bytes.Fields(procs)); n != 0 {
		return fmt.Errorf("cgroup: %d processes still in the container's cgroup root %s (move them to a leaf first)", n, m.Root)
	}
	if err := enable(m.Root, "+cpu +memory +pids"); err != nil {
		return err
	}
	if err := m.Wipe(m.Path("slots")); err != nil {
		return err
	}
	if err := os.Mkdir(m.Path("slots"), 0o755); err != nil {
		return fmt.Errorf("cgroup: mkdir slots/: %w", err)
	}
	if err := enable(m.Path("slots"), "+cpu +memory +pids"); err != nil {
		return err
	}
	for i := 0; i < Slots; i++ {
		s := m.Slot(i)
		if err := os.Mkdir(s, 0o755); err != nil {
			return fmt.Errorf("cgroup: mkdir %s: %w", s, err)
		}
		for _, kv := range [][2]string{
			{"cpu.max", SlotCPUMax},
			{"memory.max", strconv.Itoa(SlotMemoryMax)},
			{"pids.max", strconv.Itoa(SlotPidsMax)},
		} {
			if err := Write(s, kv[0], kv[1]); err != nil {
				return err
			}
		}
		if err := WriteOptional(s, "memory.swap.max", "0"); err != nil {
			return err
		}
		if err := enable(s, "+memory +pids"); err != nil {
			return err
		}
	}
	return nil
}

func enable(dir, controllers string) error {
	if err := Write(dir, "cgroup.subtree_control", controllers); err != nil {
		return fmt.Errorf("cgroup: enable %q in %s: %w", controllers, dir, err)
	}
	return nil
}

// Write writes a cgroup interface file.
func Write(dir, file, value string) error {
	f, err := os.OpenFile(filepath.Join(dir, file), os.O_WRONLY|unix.O_CLOEXEC, 0)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.WriteString(value); err != nil {
		return fmt.Errorf("write %s/%s: %w", dir, file, err)
	}
	return nil
}

// WriteOptional writes a file that may not exist (memory.swap.max without swap accounting).
func WriteOptional(dir, file, value string) error {
	if err := Write(dir, file, value); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

// Limits for a leaf (compile, case-K, canary).
type Limits struct {
	MemoryMax int64
	PidsMax   int64
}

// MkLeaf creates a leaf with memory.max, swap 0, memory.oom.group 1 and pids.max.
func MkLeaf(dir string, l Limits) error {
	if err := os.Mkdir(dir, 0o755); err != nil {
		return fmt.Errorf("cgroup: mkdir %s: %w", dir, err)
	}
	if err := Write(dir, "memory.max", strconv.FormatInt(l.MemoryMax, 10)); err != nil {
		return err
	}
	if err := WriteOptional(dir, "memory.swap.max", "0"); err != nil {
		return err
	}
	if err := Write(dir, "memory.oom.group", "1"); err != nil {
		return err
	}
	return Write(dir, "pids.max", strconv.FormatInt(l.PidsMax, 10))
}

// MkInner creates an inner node (job/) that delegates memory and pids to its leaves.
func MkInner(dir string) error {
	if err := os.Mkdir(dir, 0o755); err != nil {
		return fmt.Errorf("cgroup: mkdir %s: %w", dir, err)
	}
	return enable(dir, "+memory +pids")
}

// Kill writes cgroup.kill (every process in the subtree gets SIGKILL).
func Kill(dir string) error {
	err := Write(dir, "cgroup.kill", "1")
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}

// WaitEmpty waits for cgroup.events "populated 0".
func WaitEmpty(dir string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		b, err := os.ReadFile(filepath.Join(dir, "cgroup.events"))
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		ev, err := measure.ParseFlatKeyed(b)
		if err != nil {
			return err
		}
		if ev["populated"] == 0 {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("cgroup: %s still populated after %v", dir, timeout)
		}
		time.Sleep(time.Millisecond)
	}
}

// Wipe kills and removes dir and everything below it, deepest first. A missing dir is fine.
func (m *Manager) Wipe(dir string) error {
	if _, err := os.Stat(dir); errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err := Kill(dir); err != nil {
		return fmt.Errorf("cgroup: kill %s: %w", dir, err)
	}
	if err := WaitEmpty(dir, 5*time.Second); err != nil {
		return err
	}
	var dirs []string
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			dirs = append(dirs, p)
		}
		return nil
	})
	if err != nil {
		return err
	}
	sort.Slice(dirs, func(i, j int) bool { return len(dirs[i]) > len(dirs[j]) })
	for _, d := range dirs {
		if err := rmdirRetry(d); err != nil {
			return err
		}
	}
	return nil
}

// Remove kills and removes one leaf (or a subtree).
func (m *Manager) Remove(dir string) error { return m.Wipe(dir) }

func rmdirRetry(d string) error {
	var err error
	for i := 0; i < 200; i++ {
		err = unix.Rmdir(d)
		if err == nil || errors.Is(err, unix.ENOENT) {
			return nil
		}
		if !errors.Is(err, unix.EBUSY) {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	return fmt.Errorf("cgroup: rmdir %s: %w", d, err)
}

// ReadInt reads a single-value file (memory.current, memory.peak, pids.current).
func ReadInt(dir, file string) (int64, error) {
	b, err := os.ReadFile(filepath.Join(dir, file))
	if err != nil {
		return 0, err
	}
	return measure.ParseSingle(b)
}

// ReadKeyed reads a flat-keyed file.
func ReadKeyed(dir, file string) (map[string]int64, error) {
	b, err := os.ReadFile(filepath.Join(dir, file))
	if err != nil {
		return nil, err
	}
	return measure.ParseFlatKeyed(b)
}

// CPUUsage is cpu.stat usage_usec as a duration.
func CPUUsage(dir string) (time.Duration, error) {
	kv, err := ReadKeyed(dir, "cpu.stat")
	if err != nil {
		return 0, err
	}
	return time.Duration(kv["usage_usec"]) * time.Microsecond, nil
}

// Evidence is what a finished leaf says about its processes.
type Evidence struct {
	CPU       time.Duration
	PeakBytes int64
	OOMKills  int64
	PidsMax   int64
}

// ReadEvidence reads a leaf's verdict evidence (t3 §5.5).
func ReadEvidence(dir string) (Evidence, error) {
	var e Evidence
	var err error
	if e.CPU, err = CPUUsage(dir); err != nil {
		return e, err
	}
	if e.PeakBytes, err = ReadInt(dir, "memory.peak"); err != nil {
		return e, err
	}
	mem, err := ReadKeyed(dir, "memory.events")
	if err != nil {
		return e, err
	}
	e.OOMKills = mem["oom_kill"]
	pids, err := ReadKeyed(dir, "pids.events")
	if err != nil {
		return e, err
	}
	e.PidsMax = pids["max"]
	return e, nil
}

// OpenDir opens a cgroup directory for CLONE_INTO_CGROUP.
func OpenDir(dir string) (int, error) {
	return unix.Open(dir, unix.O_PATH|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
}

// Probe checks cgroupfs is writable (a readiness canary): create and remove a leaf.
func (m *Manager) Probe() error {
	d := m.Path("slots", ".probe")
	_ = unix.Rmdir(d)
	if err := os.Mkdir(d, 0o755); err != nil {
		return err
	}
	return unix.Rmdir(d)
}

// Leftovers lists cgroup directories under slots/ other than the two slots (cleanup check).
func (m *Manager) Leftovers() ([]string, error) {
	var out []string
	err := filepath.WalkDir(m.Path("slots"), func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			rel := strings.TrimPrefix(p, m.Path("slots"))
			if rel != "" && rel != "/s0" && rel != "/s1" {
				out = append(out, rel)
			}
		}
		return nil
	})
	return out, err
}

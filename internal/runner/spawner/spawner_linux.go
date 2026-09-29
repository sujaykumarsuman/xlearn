//go:build linux

// Package spawner is the runner's capability-holding half (t3 §5.2): PID 1 in the pod, UID 0
// in the pod's user namespace with the namespaced capabilities ADR-0030 lists. It creates
// cgroups, per-job tmpfs mounts and per-case jails, reads cgroup evidence and wait statuses,
// and kills with cgroup.kill. It never parses a learner byte — it only reads fixed-schema
// requests from the front — and it never listens on a socket.
//
// Rule R-FS (t3 §2.3): once learner code has run, no capability-holding process opens, chowns,
// copies or traverses a path learners can write to. The spawner creates the per-job src tmpfs
// (owned by the front's UID) and passes the front a directory fd; it never reads, chmods or
// traverses it. It writes the artifact tmpfs itself, from the compile jail's export pipe,
// before any learner code runs; learner-writable paths (/w) exist only inside jail mount
// namespaces and die with them.
package spawner

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sys/unix"

	"github.com/sujaykumarsuman/xlearn/internal/platform/slogx"
	"github.com/sujaykumarsuman/xlearn/internal/runner"
	"github.com/sujaykumarsuman/xlearn/internal/runner/cgroup"
	"github.com/sujaykumarsuman/xlearn/internal/runner/ipc"
	"github.com/sujaykumarsuman/xlearn/internal/runner/jail"
	"github.com/sujaykumarsuman/xlearn/internal/runner/measure"
	"github.com/sujaykumarsuman/xlearn/internal/runner/profile"
)

// UID layout inside the pod's 65536-UID map. Job UIDs come from a pool that is not reused
// within 1,000 jobs (t3 §2.4 A6): two per job (compile, exec) out of 50,000.
const (
	uidPoolStart = 10000
	uidPoolSize  = 50000
	canaryUID    = 65531
)

// Spawner is the running spawner.
type Spawner struct {
	cfg       runner.Config
	log       *slog.Logger
	version   string
	cg        *cgroup.Manager
	exe       string
	devnull   *os.File
	profiles  []*profile.Profile
	pstate    []*profState
	slots     [cgroup.Slots]*slot
	conn      *ipc.Conn
	bootEpoch string
	cpuModel  string

	uidMu   sync.Mutex
	uidNext int

	canaryMu sync.Mutex // one canary at a time
	medMu    sync.Mutex
	median   measure.Rolling

	drainOnce sync.Once
}

// Main runs the spawner until the front exits. It returns the process exit code.
func Main(version string) int {
	cfg, err := runner.Load()
	log := slogx.New(cfg.LogLevel).With("proc", "spawner")
	if err != nil {
		log.Error("config invalid; refusing to start", "err", err)
		return 1
	}
	s := &Spawner{cfg: cfg, log: log, version: version, cg: &cgroup.Manager{Root: cfg.CgroupRoot}}
	s.median.N = measure.CanaryWindow
	code, err := s.run()
	if err != nil {
		log.Error("spawner stopped", "err", err)
		if code == 0 {
			code = 1
		}
	}
	return code
}

func (s *Spawner) run() (int, error) {
	start := time.Now()
	exe, err := os.Executable()
	if err != nil {
		return 1, fmt.Errorf("resolve own binary: %w", err)
	}
	s.exe = exe
	if s.devnull, err = os.OpenFile("/dev/null", os.O_RDWR, 0); err != nil {
		return 1, err
	}
	token, err := readToken(s.cfg.TokenFile)
	if err != nil {
		return 1, err
	}
	s.bootEpoch = bootEpoch(start)
	s.cpuModel = cpuModel()
	s.uidNext = randomOffset()

	// Startup (t3 §5.4 step 7): wipe slots/, move into runner/, pre-read and hash toolchains,
	// measure baselines, then serve. Any failure refuses to start.
	if err := s.cg.Setup(os.Getpid()); err != nil {
		return 1, fmt.Errorf("cgroup setup: %w", err)
	}
	if err := s.setupJailDirs(); err != nil {
		return 1, err
	}
	if err := s.loadProfiles(); err != nil {
		return 1, err
	}
	if err := s.seedCanary(); err != nil {
		return 1, fmt.Errorf("canary: %w", err)
	}
	if err := s.measureBaselines(); err != nil {
		return 1, fmt.Errorf("baselines: %w", err)
	}
	for _, sl := range s.slots {
		b, err := cgroup.ReadInt(s.cg.Slot(sl.idx), "memory.current")
		if err != nil {
			return 1, err
		}
		sl.baseline = b
	}
	s.log.Info("startup complete", "boot_epoch", s.bootEpoch, "profiles", len(s.profiles),
		"canary_median_us", s.canaryMedian().Microseconds(), "spawn_path", s.cfg.SpawnPath,
		"mode", s.cfg.Mode, "took_ms", time.Since(start).Milliseconds())

	frontPid, err := s.startFront(token)
	if err != nil {
		return 1, fmt.Errorf("start front: %w", err)
	}
	frontDone := make(chan int, 1)
	go func() { frontDone <- waitPid(frontPid) }()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)
	defer signal.Stop(sigs)

	serveErr := make(chan error, 1)
	go func() { serveErr <- s.serve() }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go s.idleCanaryLoop(ctx)

	var code int
	var runErr error
	select {
	case code = <-frontDone:
		s.log.Info("front exited", "code", code)
	case sig := <-sigs:
		s.log.Info("signal; draining the front", "signal", sig.String())
		s.requestDrain()
		select {
		case code = <-frontDone:
		case <-time.After(s.cfg.DrainTimeout + 10*time.Second):
			s.log.Error("front did not exit after the drain; killing it")
			_ = unix.Kill(frontPid, unix.SIGKILL)
			code = <-frontDone
		}
	case err := <-serveErr:
		// A broken or violated IPC pair: kill the front, clean up, exit (kubelet restarts us).
		s.log.Error("ipc failed; killing the front", "err", err)
		_ = unix.Kill(frontPid, unix.SIGKILL)
		<-frontDone
		code, runErr = 1, err
	}
	cancel()
	s.conn.Close()
	s.cleanup()
	if code != 0 && runErr == nil {
		runErr = fmt.Errorf("front exited with %d", code)
	}
	return code, runErr
}

// readToken reads the bearer token once. It is never logged.
func readToken(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read the bearer token file %s: %w", path, err)
	}
	tok := strings.TrimSpace(string(b))
	if len(tok) < 16 {
		return "", fmt.Errorf("the bearer token in %s is missing or shorter than 16 bytes", path)
	}
	return tok, nil
}

// bootEpoch is "<hostname>/<container start, unix ns>/<8 random bytes hex>" (t3 §5.2).
func bootEpoch(start time.Time) string {
	host, err := os.Hostname()
	if err != nil || host == "" {
		host = "unknown"
	}
	var b [8]byte
	_, _ = rand.Read(b[:])
	return fmt.Sprintf("%s/%d/%s", host, start.UnixNano(), hex.EncodeToString(b[:]))
}

func cpuModel() string {
	b, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return "unknown"
	}
	for _, line := range strings.Split(string(b), "\n") {
		k, v, ok := strings.Cut(line, ":")
		if ok && strings.TrimSpace(k) == "model name" {
			return strings.TrimSpace(v)
		}
	}
	return "unknown"
}

func randomOffset() int {
	var b [4]byte
	_, _ = rand.Read(b[:])
	n := int(b[0])<<24 | int(b[1])<<16 | int(b[2])<<8 | int(b[3])
	if n < 0 {
		n = -n
	}
	return (n % (uidPoolSize / 2)) * 2
}

// nextUIDs returns the job's compile and exec UIDs.
func (s *Spawner) nextUIDs() (int, int) {
	s.uidMu.Lock()
	defer s.uidMu.Unlock()
	a := uidPoolStart + s.uidNext
	s.uidNext = (s.uidNext + 2) % uidPoolSize
	return a, a + 1
}

// setupJailDirs creates <jail>/s<N>/{root,src,art} and unmounts leftovers from a crashed run.
func (s *Spawner) setupJailDirs() error {
	if err := os.MkdirAll(s.cfg.JailDir, 0o755); err != nil {
		return err
	}
	for i := range s.slots {
		base := filepath.Join(s.cfg.JailDir, fmt.Sprintf("s%d", i))
		sl := &slot{idx: i, rootDir: filepath.Join(base, "root"), srcDir: filepath.Join(base, "src"), artDir: filepath.Join(base, "art")}
		for _, d := range []string{sl.srcDir, sl.artDir, sl.rootDir} {
			for unix.Unmount(d, unix.MNT_DETACH) == nil {
			}
			if err := os.MkdirAll(d, 0o755); err != nil {
				return err
			}
			if err := os.Chmod(d, 0o755); err != nil {
				return err
			}
		}
		if err := os.Chmod(base, 0o755); err != nil {
			return err
		}
		s.slots[i] = sl
	}
	return nil
}

// startFront re-execs this binary as `runner front`: a fixed non-zero UID with 0 caps, an
// empty bounding set and NO_NEW_PRIVS, set in the child before execve. fd 3 is its IPC end;
// fd 4 a pipe carrying the manifest (token included).
func (s *Spawner) startFront(token string) (int, error) {
	conn, theirs, err := ipc.Pair()
	if err != nil {
		return 0, err
	}
	s.conn = conn
	mr, mw, err := os.Pipe()
	if err != nil {
		return 0, err
	}
	env := []string{"PATH=/usr/bin:/bin"}
	for _, k := range []string{"LOG_LEVEL", "RUNNER_ADDR", "RUNNER_MODE", "RUNNER_IMAGE_DIGEST", "RUNNER_MAX_JOBS",
		"RUNNER_MAX_AGE", "RUNNER_DRAIN_TIMEOUT", "RUNNER_FRONT_UID", "RUNNER_CGROUP_ROOT", "RUNNER_JAIL_DIR",
		"RUNNER_SPAWN_PATH", "RUNNER_CANARY_EVERY", "RUNNER_TOKEN_FILE"} {
		if v, ok := os.LookupEnv(k); ok {
			env = append(env, k+"="+v)
		}
	}
	pid, err := jail.StartProcess(jail.Process{
		Argv:  []string{s.exe, "front"},
		Env:   env,
		Files: []uintptr{s.devnull.Fd(), os.Stdout.Fd(), os.Stderr.Fd(), theirs.Fd(), mr.Fd()},
		UID:   s.cfg.FrontUID,
	})
	theirs.Close()
	mr.Close()
	if err != nil {
		mw.Close()
		return 0, err
	}
	m := runner.Manifest{Token: token, Version: s.version, BootEpoch: s.bootEpoch, CPUModel: s.cpuModel}
	for i, p := range s.profiles {
		m.Profiles = append(m.Profiles, runner.ProfileManifest{
			Name: p.Name, Toolchain: s.pstate[i].toolchain, ProfileSHA256: s.pstate[i].sha, BaselineBytes: s.pstate[i].baseline,
		})
	}
	b, _ := json.Marshal(m)
	_, werr := mw.Write(b)
	mw.Close()
	if werr != nil {
		_ = unix.Kill(pid, unix.SIGKILL)
		return 0, fmt.Errorf("write the manifest: %w", werr)
	}
	s.log.Info("front started", "pid", pid, "uid", s.cfg.FrontUID)
	return pid, nil
}

func waitPid(pid int) int {
	var ws unix.WaitStatus
	for {
		_, err := unix.Wait4(pid, &ws, 0, nil)
		if errors.Is(err, unix.EINTR) {
			continue
		}
		if err != nil {
			return 1
		}
		if ws.Exited() {
			return ws.ExitStatus()
		}
		return 128 + int(ws.Signal())
	}
}

// requestDrain tells the front to drain and exit (once).
func (s *Spawner) requestDrain() {
	if s.conn == nil {
		return // startup: no front yet; the startup error stops the process
	}
	s.drainOnce.Do(func() {
		if err := s.conn.Send(&ipc.Msg{Type: ipc.TDrain, Body: &ipc.Drain{}}); err != nil {
			s.log.Error("send drain", "err", err)
		}
	})
}

// cleanup kills whatever is left in the slots and unmounts the per-job tmpfs.
func (s *Spawner) cleanup() {
	for _, sl := range s.slots {
		_ = s.cg.Wipe(filepath.Join(s.cg.Slot(sl.idx), "job"))
		_ = s.cg.Wipe(filepath.Join(s.cg.Slot(sl.idx), "canary"))
		for _, d := range []string{sl.srcDir, sl.artDir} {
			for unix.Unmount(d, unix.MNT_DETACH) == nil {
			}
		}
	}
}

// serve reads front requests until the pair breaks. A malformed message, a message in the
// wrong direction or any decode error ends it (the caller kills the front).
func (s *Spawner) serve() error {
	for {
		m, err := s.conn.Recv()
		if err != nil {
			return err
		}
		if !m.Type.FromFront() {
			return fmt.Errorf("ipc: %v from the front", m.Type)
		}
		switch m.Type {
		case ipc.TPing:
			s.reply(m, ipc.TPong, &ipc.Pong{Nonce: m.Body.(*ipc.Ping).Nonce})
		case ipc.TCaseKill:
			s.caseKill(int(m.Slot))
		case ipc.TJobBegin:
			go s.handleJobBegin(m)
		case ipc.TCompile:
			go s.handleCompile(m)
		case ipc.TCaseRun:
			go s.handleCaseRun(m)
		case ipc.TJobEnd:
			go s.handleJobEnd(m)
		case ipc.TQuietCheck:
			go s.handleQuietCheck(m)
		case ipc.TStats:
			go s.handleStats(m)
		case ipc.TProbe:
			go s.handleProbe(m)
		default:
			return fmt.Errorf("ipc: unhandled %v", m.Type)
		}
	}
}

func (s *Spawner) reply(req *ipc.Msg, t ipc.Type, body any, fds ...int) {
	if err := s.conn.Send(&ipc.Msg{Type: t, ReqID: req.ReqID, Slot: req.Slot, Body: body, Fds: fds}); err != nil {
		s.log.Error("ipc send", "type", t.String(), "err", err)
	}
}

func (s *Spawner) fail(req *ipc.Msg, code ipc.FailCode, err error) {
	var errno unix.Errno
	_ = errors.As(err, &errno)
	lvl := slog.LevelError
	if code == ipc.FailBadRequest {
		lvl = slog.LevelWarn
	}
	s.log.Log(context.Background(), lvl, "request failed", "type", req.Type.String(), "slot", req.Slot, "code", code, "err", err)
	s.reply(req, ipc.TFail, &ipc.Fail{Code: uint16(code), Errno: uint32(errno)})
}

//go:build linux

package front

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/sujaykumarsuman/xlearn/internal/platform/slogx"
	"github.com/sujaykumarsuman/xlearn/internal/runner"
	"github.com/sujaykumarsuman/xlearn/internal/runner/ipc"
)

// Inherited fds (set up by the spawner).
const (
	fdIPC      = 3
	fdManifest = 4
)

// Main is `runner front`. It refuses to run holding any privilege.
func Main() int {
	cfg, err := runner.Load()
	log := slogx.New(cfg.LogLevel).With("proc", "front")
	if err != nil {
		log.Error("config invalid", "err", err)
		return 1
	}
	if err := checkUnprivileged(); err != nil {
		log.Error("refusing to run privileged", "err", err)
		return 1
	}
	man, err := runner.ReadManifest(os.NewFile(fdManifest, "manifest"))
	if err != nil {
		log.Error("read the manifest", "err", err)
		return 1
	}
	conn, err := ipc.FromFile(os.NewFile(fdIPC, "ipc"))
	if err != nil {
		log.Error("ipc", "err", err)
		return 1
	}
	b := newIPCBackend(conn, log)
	srv := NewServer(cfg, man, b, log)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		select {
		case <-sigs:
			srv.StartDrain("signal")
		case <-b.DrainRequested():
			srv.StartDrain("spawner")
		}
	}()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.Serve(ctx) }()
	log.Info("front listening", "addr", cfg.Addr, "boot_epoch", man.BootEpoch, "mode", cfg.Mode, "profiles", len(man.Profiles))

	select {
	case <-srv.Drained():
		log.Info("front drained; exiting")
		return 0
	case <-b.Broken():
		log.Error("the spawner is gone; exiting")
		return 1
	case err := <-serveErr:
		if err != nil {
			log.Error("http server", "err", err)
			return 1
		}
		<-srv.Drained()
		return 0
	}
}

// checkUnprivileged asserts what the spawner set before execve: a non-zero UID, no
// capabilities in any set, an empty bounding set and NO_NEW_PRIVS.
func checkUnprivileged() error {
	if os.Getuid() == 0 || os.Geteuid() == 0 {
		return fmt.Errorf("uid 0")
	}
	b, err := os.ReadFile("/proc/self/status")
	if err != nil {
		return err
	}
	want := map[string]string{"CapInh": "0000000000000000", "CapPrm": "0000000000000000", "CapEff": "0000000000000000",
		"CapBnd": "0000000000000000", "CapAmb": "0000000000000000", "NoNewPrivs": "1"}
	for _, line := range strings.Split(string(b), "\n") {
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		if w, ok := want[k]; ok {
			if strings.TrimSpace(v) != w {
				return fmt.Errorf("%s is %s, want %s", k, strings.TrimSpace(v), w)
			}
			delete(want, k)
		}
	}
	if len(want) != 0 {
		return fmt.Errorf("/proc/self/status lacks %v", want)
	}
	return nil
}

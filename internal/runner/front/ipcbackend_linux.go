//go:build linux

package front

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"

	"golang.org/x/sys/unix"

	"github.com/sujaykumarsuman/xlearn/internal/runner/ipc"
	"github.com/sujaykumarsuman/xlearn/internal/runner/measure"
)

// ipcBackend is Backend over the spawner⇄front pair: requests carry a ReqID, one reader
// goroutine routes replies, and any protocol error breaks the pair (the front exits).
type ipcBackend struct {
	conn *ipc.Conn
	log  *slog.Logger

	mu      sync.Mutex
	next    uint32
	pending map[uint32]chan *ipc.Msg

	drain      chan struct{}
	drainOnce  sync.Once
	broken     chan struct{}
	brokenOnce sync.Once
}

func newIPCBackend(conn *ipc.Conn, log *slog.Logger) *ipcBackend {
	b := &ipcBackend{conn: conn, log: log, pending: map[uint32]chan *ipc.Msg{},
		drain: make(chan struct{}), broken: make(chan struct{})}
	go b.readLoop()
	return b
}

func (b *ipcBackend) DrainRequested() <-chan struct{} { return b.drain }
func (b *ipcBackend) Broken() <-chan struct{}         { return b.broken }

func (b *ipcBackend) breakPair(err error) {
	b.brokenOnce.Do(func() {
		b.log.Error("ipc pair broken", "err", err)
		b.conn.Close()
		close(b.broken)
	})
}

func (b *ipcBackend) readLoop() {
	for {
		m, err := b.conn.Recv()
		if err != nil {
			b.breakPair(err)
			return
		}
		if m.Type.FromFront() {
			closeFds(m.Fds)
			b.breakPair(fmt.Errorf("%v from the spawner", m.Type))
			return
		}
		if m.Type == ipc.TDrain {
			b.drainOnce.Do(func() { close(b.drain) })
			continue
		}
		b.mu.Lock()
		ch := b.pending[m.ReqID]
		b.mu.Unlock()
		if ch == nil {
			closeFds(m.Fds) // a late reply to an abandoned request
			continue
		}
		select {
		case ch <- m:
		default:
			closeFds(m.Fds)
			b.breakPair(fmt.Errorf("unexpected extra %v for request %d", m.Type, m.ReqID))
			return
		}
	}
}

func closeFds(fds []int) {
	for _, fd := range fds {
		unix.Close(fd)
	}
}

// file wraps a received fd as a pollable *os.File (non-blocking, so Close unblocks a Read).
func file(fd int, name string) *os.File {
	_ = unix.SetNonblock(fd, true)
	return os.NewFile(uintptr(fd), name)
}

func (b *ipcBackend) send(t ipc.Type, slot int, body any) (uint32, chan *ipc.Msg, error) {
	b.mu.Lock()
	b.next++
	id := b.next
	ch := make(chan *ipc.Msg, 2)
	b.pending[id] = ch
	b.mu.Unlock()
	if err := b.conn.Send(&ipc.Msg{Type: t, ReqID: id, Slot: uint8(slot), Body: body}); err != nil {
		b.forget(id)
		b.breakPair(err)
		return 0, nil, err
	}
	return id, ch, nil
}

func (b *ipcBackend) forget(id uint32) {
	b.mu.Lock()
	delete(b.pending, id)
	b.mu.Unlock()
}

// await waits for the next message of the request, which must be of type want.
func (b *ipcBackend) await(ctx context.Context, ch chan *ipc.Msg, want ipc.Type) (*ipc.Msg, error) {
	select {
	case m := <-ch:
		if m.Type == ipc.TFail {
			f := m.Body.(*ipc.Fail)
			return nil, &FailError{Code: ipc.FailCode(f.Code), Errno: f.Errno}
		}
		if m.Type != want {
			closeFds(m.Fds)
			err := fmt.Errorf("got %v, want %v", m.Type, want)
			b.breakPair(err)
			return nil, err
		}
		return m, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-b.broken:
		return nil, errors.New("ipc pair broken")
	}
}

func (b *ipcBackend) call(ctx context.Context, t ipc.Type, slot int, body any, want ipc.Type) (*ipc.Msg, error) {
	id, ch, err := b.send(t, slot, body)
	if err != nil {
		return nil, err
	}
	defer b.forget(id)
	return b.await(ctx, ch, want)
}

func (b *ipcBackend) Ping(ctx context.Context) error {
	_, err := b.call(ctx, ipc.TPing, 0, &ipc.Ping{Nonce: 1}, ipc.TPong)
	return err
}

func (b *ipcBackend) BeginJob(ctx context.Context, slot, profileIdx int, seq uint64, compileCPUms, compileMemMB int64) (Begun, error) {
	m, err := b.call(ctx, ipc.TJobBegin, slot, &ipc.JobBegin{
		ProfileIdx: uint16(profileIdx), JobSeq: seq, CompileCPUms: uint32(compileCPUms), CompileMemMB: uint32(compileMemMB),
	}, ipc.TJobBegun)
	if err != nil {
		return Begun{}, err
	}
	return Begun{SrcDir: os.NewFile(uintptr(m.Fds[0]), "src"), CanaryMedianUs: int64(m.Body.(*ipc.JobBegun).CanaryMedianUs)}, nil
}

func (b *ipcBackend) Compile(ctx context.Context, slot int) (*CompileRun, error) {
	id, ch, err := b.send(ipc.TCompile, slot, &ipc.Compile{})
	if err != nil {
		return nil, err
	}
	m, err := b.await(ctx, ch, ipc.TCompileStarted)
	if err != nil {
		b.forget(id)
		return nil, err
	}
	done := make(chan CompileOutcome, 1)
	go func() {
		defer b.forget(id)
		dm, err := b.await(context.Background(), ch, ipc.TCompileDone)
		if err != nil {
			done <- CompileOutcome{Err: err}
			return
		}
		done <- CompileOutcome{Done: dm.Body.(*ipc.CompileDone)}
	}()
	return &CompileRun{Stdout: file(m.Fds[0], "compile-stdout"), Stderr: file(m.Fds[1], "compile-stderr"), Done: done}, nil
}

func (b *ipcBackend) RunCase(ctx context.Context, slot int, req ipc.CaseRun) (*CaseRun, error) {
	id, ch, err := b.send(ipc.TCaseRun, slot, &req)
	if err != nil {
		return nil, err
	}
	m, err := b.await(ctx, ch, ipc.TCaseStarted)
	if err != nil {
		b.forget(id)
		return nil, err
	}
	done := make(chan CaseOutcome, 1)
	go func() {
		defer b.forget(id)
		dm, err := b.await(context.Background(), ch, ipc.TCaseDone)
		if err != nil {
			done <- CaseOutcome{Err: err}
			return
		}
		done <- CaseOutcome{Done: dm.Body.(*ipc.CaseDone)}
	}()
	kill := func() {
		if err := b.conn.Send(&ipc.Msg{Type: ipc.TCaseKill, ReqID: id, Slot: uint8(slot), Body: &ipc.CaseKill{Reason: uint8(measure.KillOLE)}}); err != nil {
			b.breakPair(err)
		}
	}
	return &CaseRun{
		Input: file(m.Fds[0], "fd3"), Stdout: file(m.Fds[1], "fd1"), Stderr: file(m.Fds[2], "fd2"), Result: file(m.Fds[3], "fd4"),
		Done: done, Kill: kill,
	}, nil
}

func (b *ipcBackend) EndJob(ctx context.Context, slot int, abort bool) (ipc.Teardown, error) {
	var a uint8
	if abort {
		a = 1
	}
	m, err := b.call(ctx, ipc.TJobEnd, slot, &ipc.JobEnd{Abort: a}, ipc.TJobEnded)
	if err != nil {
		return 0, err
	}
	return ipc.Teardown(m.Body.(*ipc.JobEnded).Outcome), nil
}

func (b *ipcBackend) QuietCheck(ctx context.Context, slot int) (QuietStatus, error) {
	m, err := b.call(ctx, ipc.TQuietCheck, slot, &ipc.QuietCheck{}, ipc.TQuietStatus)
	if err != nil {
		return QuietStatus{}, err
	}
	q := m.Body.(*ipc.QuietStatus)
	return QuietStatus{Steal: float64(q.StealPermille) / 1000, Canary: float64(q.CanaryPermille) / 1000}, nil
}

func (b *ipcBackend) Stats(ctx context.Context) (ipc.StatsReply, error) {
	m, err := b.call(ctx, ipc.TStats, 0, &ipc.Stats{}, ipc.TStatsReply)
	if err != nil {
		return ipc.StatsReply{}, err
	}
	return *m.Body.(*ipc.StatsReply), nil
}

func (b *ipcBackend) Probe(ctx context.Context) (ipc.ProbeReply, error) {
	m, err := b.call(ctx, ipc.TProbe, 0, &ipc.Probe{}, ipc.TProbeReply)
	if err != nil {
		return ipc.ProbeReply{}, err
	}
	return *m.Body.(*ipc.ProbeReply), nil
}

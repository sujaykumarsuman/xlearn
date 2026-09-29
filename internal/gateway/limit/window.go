package limit

import (
	"sync"
	"time"
)

// FailureWindow counts failures per key inside a sliding window (L1's "5 failures / 15 min
// per identifier"). At max failures the key is blocked until its oldest failure ages out;
// a success clears the key. Keys are opaque: the gateway passes a SHA-256 of the
// normalised identifier, never the identifier itself. A nil *FailureWindow never blocks.
type FailureWindow struct {
	max        int
	window     time.Duration
	now        Clock
	sweepEvery time.Duration
	maxKeys    int

	mu        sync.Mutex
	fails     map[string][]time.Time // oldest first, at most max entries
	lastSweep time.Time
}

// NewFailureWindow returns a window blocking a key after max failures within window.
func NewFailureWindow(max int, window time.Duration, opts ...Option) *FailureWindow {
	// Reuse Limiter's options (clock, key cap) so callers configure both the same way.
	cfg := &Limiter{now: time.Now, maxKeys: DefaultMaxKeys}
	for _, o := range opts {
		o(cfg)
	}
	f := &FailureWindow{
		max:        max,
		window:     window,
		now:        cfg.now,
		sweepEvery: DefaultSweepEvery,
		maxKeys:    cfg.maxKeys,
		fails:      make(map[string][]time.Time),
	}
	f.lastSweep = f.now()
	return f
}

// Blocked reports whether key has reached max failures inside the window and, if so, how
// long until its oldest failure ages out.
func (f *FailureWindow) Blocked(key string) (bool, time.Duration) {
	if f == nil {
		return false, 0
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	now := f.now()
	f.maybeSweep(now)
	live := f.prune(key, now)
	if len(live) < f.max {
		return false, 0
	}
	return true, live[0].Add(f.window).Sub(now)
}

// Fail records one failure for key.
func (f *FailureWindow) Fail(key string) {
	if f == nil {
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	now := f.now()
	f.maybeSweep(now)
	live := f.prune(key, now)
	if live == nil && len(f.fails) >= f.maxKeys {
		f.makeRoom(now)
	}
	live = append(live, now)
	if len(live) > f.max {
		live = live[len(live)-f.max:]
	}
	f.fails[key] = live
}

// Reset clears key (a successful login).
func (f *FailureWindow) Reset(key string) {
	if f == nil {
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.fails, key)
}

// Len is the number of keys held (tests and the memory bound).
func (f *FailureWindow) Len() int {
	if f == nil {
		return 0
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.fails)
}

// prune drops key's failures older than the window and returns what is left (nil when
// nothing is, in which case the key is deleted).
func (f *FailureWindow) prune(key string, now time.Time) []time.Time {
	ts, ok := f.fails[key]
	if !ok {
		return nil
	}
	i := 0
	for i < len(ts) && now.Sub(ts[i]) >= f.window {
		i++
	}
	if i == len(ts) {
		delete(f.fails, key)
		return nil
	}
	if i > 0 {
		ts = append([]time.Time(nil), ts[i:]...)
		f.fails[key] = ts
	}
	return ts
}

// maybeSweep drops keys whose every failure has aged out, at most once every sweepEvery.
func (f *FailureWindow) maybeSweep(now time.Time) {
	if now.Sub(f.lastSweep) < f.sweepEvery {
		return
	}
	f.lastSweep = now
	f.evictExpired(now)
}

func (f *FailureWindow) evictExpired(now time.Time) {
	for k, ts := range f.fails {
		if now.Sub(ts[len(ts)-1]) >= f.window {
			delete(f.fails, k)
		}
	}
}

// makeRoom runs when the map is full: drop expired keys first, else reset.
func (f *FailureWindow) makeRoom(now time.Time) {
	f.evictExpired(now)
	if len(f.fails) >= f.maxKeys {
		f.fails = make(map[string][]time.Time)
	}
}

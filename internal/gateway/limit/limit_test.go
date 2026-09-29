package limit

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// fakeClock is an injectable clock; tests advance it instead of sleeping.
type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func newFakeClock() *fakeClock { return &fakeClock{t: time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC)} }

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

func TestLimiterBurstThenRefill(t *testing.T) {
	clk := newFakeClock()
	l := New(PerMinute(60), 20, WithClock(clk.Now)) // L4's value
	for i := 1; i <= 20; i++ {
		if ok, _ := l.Allow("203.0.113.7"); !ok {
			t.Fatalf("request %d refused inside the burst", i)
		}
	}
	ok, wait := l.Allow("203.0.113.7")
	if ok {
		t.Fatal("request 21 admitted past the burst")
	}
	if wait <= 0 || wait > time.Second {
		t.Fatalf("retry after %v, want (0, 1s] at 60/min", wait)
	}
	clk.Advance(time.Second)
	if ok, _ := l.Allow("203.0.113.7"); !ok {
		t.Fatal("a token did not refill after 1s at 60/min")
	}
}

func TestLimiterKeysAreIndependent(t *testing.T) {
	clk := newFakeClock()
	l := New(PerMinute(5), 1, WithClock(clk.Now))
	if ok, _ := l.Allow("a"); !ok {
		t.Fatal("a refused")
	}
	if ok, _ := l.Allow("a"); ok {
		t.Fatal("a admitted twice with burst 1")
	}
	if ok, _ := l.Allow("b"); !ok {
		t.Fatal("b refused because of a")
	}
}

func TestLimiterRetryAfterMatchesRate(t *testing.T) {
	clk := newFakeClock()
	l := New(PerMinute(10), 5, WithClock(clk.Now)) // L1 per IP
	for i := 0; i < 5; i++ {
		l.Allow("ip")
	}
	ok, wait := l.Allow("ip")
	if ok || wait != 6*time.Second {
		t.Fatalf("got ok=%v wait=%v, want refused with 6s (10/min)", ok, wait)
	}
	if got := RetryAfterSeconds(wait); got != 6 {
		t.Fatalf("RetryAfterSeconds(%v) = %d, want 6", wait, got)
	}
}

func TestLimiterSweepEvictsIdleKeys(t *testing.T) {
	clk := newFakeClock()
	l := New(PerSecond(20), 40, WithClock(clk.Now))
	l.Allow("old")
	clk.Advance(DefaultIdle)
	l.Allow("new") // ≥ 60 s since the last sweep: the sweep runs and drops "old"
	if l.Len() != 1 {
		t.Fatalf("len = %d after sweeping an idle key, want 1", l.Len())
	}
	clk.Advance(30 * time.Second)
	l.Allow("newer") // < 60 s since the sweep: no sweep
	if l.Len() != 2 {
		t.Fatalf("len = %d, want 2 (no sweep inside 60 s)", l.Len())
	}
}

func TestLimiterCapEvictsIdleFirstThenResets(t *testing.T) {
	clk := newFakeClock()
	l := New(PerSecond(1), 1, WithClock(clk.Now), WithMaxKeys(3))
	l.Allow("idle")
	clk.Advance(DefaultIdle - time.Second) // not yet swept (the sweep needs idle ≥ 10 min)
	l.Allow("b")
	l.Allow("c")
	clk.Advance(time.Second) // "idle" is now idle ≥ 10 min; b and c are fresh
	// The sweep already ran at "b" (≥ 60 s since construction); the next is ≥ 60 s away,
	// so the cap path is what makes room here.
	l.Allow("d")
	if l.Len() != 3 {
		t.Fatalf("len = %d, want 3 (idle key evicted to make room)", l.Len())
	}
	// All three fresh and the map full: the next new key resets the map.
	l.Allow("e")
	if l.Len() != 1 {
		t.Fatalf("len = %d, want 1 after an overflow reset", l.Len())
	}
	// A reset is safe: the reset keys start with a full (cold) bucket.
	if ok, _ := l.Allow("b"); !ok {
		t.Fatal("a reset key was not admitted")
	}
}

func TestLimiterBoundedUnderFlood(t *testing.T) {
	clk := newFakeClock()
	l := New(PerMinute(60), 20, WithClock(clk.Now), WithMaxKeys(100))
	for i := 0; i < 10_000; i++ {
		l.Allow(string(rune('a'+i%26)) + time.Duration(i).String())
		if l.Len() > 100 {
			t.Fatalf("len %d exceeds the cap", l.Len())
		}
	}
}

func TestNilLimiterAndWindowAdmit(t *testing.T) {
	var l *Limiter
	if ok, _ := l.Allow("x"); !ok || l.Len() != 0 {
		t.Fatal("nil limiter must admit")
	}
	var f *FailureWindow
	f.Fail("x")
	f.Reset("x")
	if blocked, _ := f.Blocked("x"); blocked || f.Len() != 0 {
		t.Fatal("nil window must never block")
	}
}

func TestFailureWindowBlocksAtMaxUntilOldestAgesOut(t *testing.T) {
	clk := newFakeClock()
	f := NewFailureWindow(5, 15*time.Minute, WithClock(clk.Now)) // L1 per identifier
	for i := 0; i < 5; i++ {
		if blocked, _ := f.Blocked("k"); blocked {
			t.Fatalf("blocked after %d failures, want 5", i)
		}
		f.Fail("k")
		clk.Advance(time.Minute)
	}
	// Failures at t0..t0+4m; now t0+5m.
	blocked, wait := f.Blocked("k")
	if !blocked || wait != 10*time.Minute {
		t.Fatalf("blocked=%v wait=%v, want blocked for 10m (oldest at t0 + 15m)", blocked, wait)
	}
	if blocked, _ := f.Blocked("other"); blocked {
		t.Fatal("another identifier is blocked")
	}
	clk.Advance(10 * time.Minute) // the oldest failure ages out: 4 left in the window
	if blocked, _ := f.Blocked("k"); blocked {
		t.Fatal("still blocked after the oldest failure aged out")
	}
	f.Fail("k") // back to 5
	if blocked, _ := f.Blocked("k"); !blocked {
		t.Fatal("not blocked at 5 failures in the window")
	}
	f.Reset("k") // a success clears it
	if blocked, _ := f.Blocked("k"); blocked || f.Len() != 0 {
		t.Fatal("a reset did not clear the identifier")
	}
}

func TestFailureWindowSweepAndCap(t *testing.T) {
	clk := newFakeClock()
	f := NewFailureWindow(5, 15*time.Minute, WithClock(clk.Now), WithMaxKeys(2))
	f.Fail("a")
	f.Fail("b")
	clk.Advance(15 * time.Minute)
	f.Fail("c") // the sweep drops a and b (their failures aged out)
	if f.Len() != 1 {
		t.Fatalf("len = %d, want 1 after the sweep", f.Len())
	}
	f.Fail("d")
	f.Fail("e") // full with live keys: reset
	if f.Len() != 1 {
		t.Fatalf("len = %d, want 1 after an overflow reset", f.Len())
	}
}

func TestWriteTooMany(t *testing.T) {
	for _, tc := range []struct {
		code string
		wait time.Duration
		want int
	}{
		{CodeRateLimited, 5500 * time.Millisecond, 6},
		{CodeBusy, 0, 1},
		{CodeRateLimited, 10 * time.Minute, 600},
	} {
		rec := httptest.NewRecorder()
		WriteTooMany(rec, tc.code, tc.wait)
		if rec.Code != http.StatusTooManyRequests {
			t.Fatalf("status %d", rec.Code)
		}
		if got := rec.Header().Get("Retry-After"); got != itoa(tc.want) {
			t.Fatalf("Retry-After %q, want %d", got, tc.want)
		}
		var env struct {
			Error struct {
				Code       string `json:"code"`
				Message    string `json:"message"`
				RetryAfter int    `json:"retry_after"`
			} `json:"error"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
			t.Fatal(err)
		}
		if env.Error.Code != tc.code || env.Error.RetryAfter != tc.want || env.Error.Message == "" {
			t.Fatalf("envelope %+v, want code %s retry_after %d", env.Error, tc.code, tc.want)
		}
	}
}

func itoa(n int) string {
	b, _ := json.Marshal(n)
	return string(b)
}

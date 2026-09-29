// Package limit holds the gateway's in-process request limits (ADR-0035 §4 L1, L2, L4, L5):
// keyed token buckets, the login failure window, the client-IP key and the typed 429.
//
// Placement: per-IP and per-account limits live in the gateway process (Traefik can't key
// by the account behind the session cookie). The state is process-local, which is fine for
// the single gateway replica and is why L24 lists it as a scale-out blocker
// (docs/architecture/services.md). Every map here is bounded: idle keys are swept and a
// hard cap resets the map rather than letting a flood of distinct keys grow it. A cold
// bucket is always safe (it only ever admits a full burst).
//
// Limits fail loudly: a denied request gets a typed 429 with Retry-After (WriteTooMany),
// never a silent drop.
package limit

import (
	"math"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Defaults for every keyed map (the sprint plan; ADR-0035 §5's memory-sum rule).
const (
	// DefaultMaxKeys caps each limiter's map. At a few hundred bytes a bucket the worst
	// case is a few MiB, well inside the gateway's 128 Mi limit.
	DefaultMaxKeys = 16384
	// DefaultSweepEvery is how often (at most) a limiter sweeps idle keys.
	DefaultSweepEvery = 60 * time.Second
	// DefaultIdle is how long a key must be untouched before the sweep evicts it. Every
	// bucket in the gateway refills completely well inside it, so evicting is lossless.
	DefaultIdle = 10 * time.Minute
)

// Clock returns the current time. Tests inject a fake; production uses time.Now.
type Clock func() time.Time

// Limiter is a set of token buckets keyed by a string (a client IP, an account id). The
// zero value is not usable; build one with New. A nil *Limiter admits everything, so a
// caller can leave a limit unset in tests without nil checks.
type Limiter struct {
	limit      rate.Limit
	burst      int
	now        Clock
	idle       time.Duration
	sweepEvery time.Duration
	maxKeys    int

	mu        sync.Mutex
	buckets   map[string]*bucket
	lastSweep time.Time
}

type bucket struct {
	lim  *rate.Limiter
	seen time.Time
}

// Option configures a Limiter.
type Option func(*Limiter)

// WithClock injects the clock (tests).
func WithClock(now Clock) Option { return func(l *Limiter) { l.now = now } }

// WithMaxKeys overrides the key cap (tests).
func WithMaxKeys(n int) Option { return func(l *Limiter) { l.maxKeys = n } }

// PerMinute is n events per minute as a rate.Limit.
func PerMinute(n int) rate.Limit { return rate.Limit(float64(n) / 60) }

// PerSecond is n events per second as a rate.Limit.
func PerSecond(n int) rate.Limit { return rate.Limit(n) }

// New returns a Limiter admitting r events per second per key with the given burst.
func New(r rate.Limit, burst int, opts ...Option) *Limiter {
	l := &Limiter{
		limit:      r,
		burst:      burst,
		now:        time.Now,
		idle:       DefaultIdle,
		sweepEvery: DefaultSweepEvery,
		maxKeys:    DefaultMaxKeys,
		buckets:    make(map[string]*bucket),
	}
	for _, o := range opts {
		o(l)
	}
	l.lastSweep = l.now()
	return l
}

// Allow takes one token from key's bucket. When the bucket is empty it returns false and
// how long until a token is available (the Retry-After).
func (l *Limiter) Allow(key string) (bool, time.Duration) {
	if l == nil {
		return true, 0
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	l.maybeSweep(now)
	b, ok := l.buckets[key]
	if !ok {
		if len(l.buckets) >= l.maxKeys {
			l.makeRoom(now)
		}
		b = &bucket{lim: rate.NewLimiter(l.limit, l.burst)}
		l.buckets[key] = b
	}
	b.seen = now
	if b.lim.AllowN(now, 1) {
		return true, 0
	}
	return false, waitForToken(b.lim.TokensAt(now), float64(l.limit))
}

// Len is the number of keys held (tests and the memory bound).
func (l *Limiter) Len() int {
	if l == nil {
		return 0
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.buckets)
}

// maybeSweep evicts keys idle for at least l.idle, at most once every l.sweepEvery. It runs
// inline on the next call (no goroutine to manage); with no traffic nothing grows anyway.
func (l *Limiter) maybeSweep(now time.Time) {
	if now.Sub(l.lastSweep) < l.sweepEvery {
		return
	}
	l.lastSweep = now
	l.evictIdle(now)
}

func (l *Limiter) evictIdle(now time.Time) {
	for k, b := range l.buckets {
		if now.Sub(b.seen) >= l.idle {
			delete(l.buckets, k)
		}
	}
}

// makeRoom runs when the map is full: evict idle keys first, else reset the map (a cold
// bucket is always safe).
func (l *Limiter) makeRoom(now time.Time) {
	l.evictIdle(now)
	if len(l.buckets) >= l.maxKeys {
		l.buckets = make(map[string]*bucket)
	}
}

// waitForToken is how long a bucket holding `tokens` needs to reach one whole token at
// `perSecond`.
func waitForToken(tokens, perSecond float64) time.Duration {
	if perSecond <= 0 {
		return time.Hour
	}
	need := 1 - tokens
	if need <= 0 {
		return 0
	}
	return time.Duration(math.Ceil(need / perSecond * float64(time.Second)))
}

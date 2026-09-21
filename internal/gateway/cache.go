package gateway

import (
	"sync"
	"time"
)

// aggCache is a per-account, TTL-bounded cache of composed BFF aggregation responses
// (the Dashboard "Today" and the Week view). It exists because those two screens fan
// out to several services on every load; caching the composed JSON per account cuts the
// repeat-load cost toward the p95 target without a shared schema (the gateway owns none,
// ADR-0002).
//
// Safety (ADR-0006): entries are keyed by account id, so one user's cache can NEVER
// serve another. Correctness is kept two ways — (1) the account's own mutating writes
// (outcome / revision score / mock score / mistake edit) invalidate ALL of that
// account's entries eagerly, and (2) a short TTL bounds staleness from async events the
// gateway does not observe directly (the review sweep marking a touch due, the
// assessment projections updating), so "due today" / streak data can never be stale by
// more than the TTL. Only successful (200) responses are cached.
//
// The cache is process-local (each gateway replica has its own). With a small TTL and
// per-account keys, two replicas at most differ by the TTL for the same account — the
// same bound as a single replica, so it stays correct without cross-replica coordination.
type aggCache struct {
	ttl time.Duration
	now func() time.Time // injectable clock for tests

	mu  sync.Mutex
	m   map[string]aggEntry
	gen map[string]uint64 // per-account invalidation epoch (see putFresh)
}

type aggEntry struct {
	body    []byte
	expires time.Time
}

// maxCacheEntries caps the map so a burst of distinct accounts can't grow it without
// bound. On overflow the whole map is dropped (a cold cache is always correct); for a
// single-node solo app this ceiling is never realistically hit.
const maxCacheEntries = 4096

// newAggCache returns a cache with the given TTL, or nil when ttl <= 0 (caching off).
// A nil *aggCache is safe to call every method on (they no-op), so callers need no
// nil-guards.
func newAggCache(ttl time.Duration) *aggCache {
	if ttl <= 0 {
		return nil
	}
	return &aggCache{ttl: ttl, now: time.Now, m: make(map[string]aggEntry), gen: make(map[string]uint64)}
}

// key namespaces an entry by account so no two accounts ever collide. The NUL
// separator can't appear in an account id or a cache key, so the join is unambiguous.
func cacheKey(account, name string) string { return account + "\x00" + name }

// get returns a fresh cached body for (account, name), or (nil, false) on a miss or
// expiry.
func (c *aggCache) get(account, name string) ([]byte, bool) {
	if c == nil {
		return nil, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.m[cacheKey(account, name)]
	if !ok || c.now().After(e.expires) {
		return nil, false
	}
	return e.body, true
}

// put stores body unconditionally at the account's current epoch. Convenience for
// callers that don't straddle a lock-free compose (and the unit tests); the read
// handlers use epoch()+putFresh() to guard the invalidate-during-compose race.
func (c *aggCache) put(account, name string, body []byte) {
	c.putFresh(account, name, body, c.epoch(account))
}

// epoch returns the account's current invalidation epoch. A reader captures it right
// after a cache miss and hands it to putFresh, so a write that invalidates DURING the
// (lock-free) fan-out compose is not silently defeated (see putFresh).
func (c *aggCache) epoch(account string) uint64 {
	if c == nil {
		return 0
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.gen[account]
}

// putFresh stores body under (account, name) — but ONLY if the account's epoch still
// equals `seen` (the value the reader captured just after its cache miss). If a write
// invalidated the account in the meantime, the epoch has advanced and the now-stale
// snapshot is dropped instead of being cached, preserving the eager-invalidation
// guarantee without holding a lock across the multi-upstream compose. body must not be
// mutated after the call (the cache holds the same slice).
func (c *aggCache) putFresh(account, name string, body []byte, seen uint64) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.gen[account] != seen {
		return // invalidated since the read began — do not cache the stale snapshot
	}
	if len(c.m) >= maxCacheEntries {
		// Simple, always-correct overflow policy: drop everything and start cold.
		c.m = make(map[string]aggEntry)
	}
	c.m[cacheKey(account, name)] = aggEntry{body: body, expires: c.now().Add(c.ttl)}
}

// invalidate drops every cached entry for an account (all of its aggregation views) and
// advances its epoch so an in-flight read's putFresh cannot re-cache a pre-write
// snapshot. Called on the account's mutating writes so a post-write read recomputes fresh.
func (c *aggCache) invalidate(account string) {
	if c == nil {
		return
	}
	prefix := account + "\x00"
	c.mu.Lock()
	defer c.mu.Unlock()
	c.gen[account]++
	for k := range c.m {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			delete(c.m, k)
		}
	}
}

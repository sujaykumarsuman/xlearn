package gateway

import (
	"testing"
	"time"
)

// fixedClock returns a controllable clock for deterministic TTL tests.
func fixedClock(t *time.Time) func() time.Time { return func() time.Time { return *t } }

func TestAggCacheHitThenExpiry(t *testing.T) {
	clock := time.Unix(1_000, 0)
	c := newAggCache(15 * time.Second)
	c.now = fixedClock(&clock)

	c.put("acct-1", "dashboard", []byte(`{"a":1}`))

	if got, ok := c.get("acct-1", "dashboard"); !ok || string(got) != `{"a":1}` {
		t.Fatalf("expected fresh hit, got %q ok=%v", got, ok)
	}
	// Still fresh at the TTL boundary...
	clock = clock.Add(15 * time.Second)
	if _, ok := c.get("acct-1", "dashboard"); !ok {
		t.Fatalf("entry should still be valid at the TTL boundary")
	}
	// ...expired just past it.
	clock = clock.Add(time.Nanosecond)
	if _, ok := c.get("acct-1", "dashboard"); ok {
		t.Fatalf("entry should be expired past the TTL")
	}
}

func TestAggCachePerAccountIsolation(t *testing.T) {
	c := newAggCache(time.Minute)
	c.put("acct-1", "dashboard", []byte(`one`))
	c.put("acct-2", "dashboard", []byte(`two`))

	if got, _ := c.get("acct-1", "dashboard"); string(got) != `one` {
		t.Fatalf("acct-1 got %q", got)
	}
	if got, _ := c.get("acct-2", "dashboard"); string(got) != `two` {
		t.Fatalf("acct-2 got %q", got)
	}
	// Invalidating one account must not touch another's entries.
	c.invalidate("acct-1")
	if _, ok := c.get("acct-1", "dashboard"); ok {
		t.Fatalf("acct-1 entry should be invalidated")
	}
	if _, ok := c.get("acct-2", "dashboard"); !ok {
		t.Fatalf("acct-2 entry must survive acct-1 invalidation")
	}
}

func TestAggCacheInvalidateDropsAllViews(t *testing.T) {
	c := newAggCache(time.Minute)
	c.put("acct-1", "dashboard", []byte(`d`))
	c.put("acct-1", "week:dsa:3", []byte(`w`))

	c.invalidate("acct-1")
	if _, ok := c.get("acct-1", "dashboard"); ok {
		t.Fatalf("dashboard should be gone after invalidate")
	}
	if _, ok := c.get("acct-1", "week:dsa:3"); ok {
		t.Fatalf("week should be gone after invalidate")
	}
}

func TestAggCacheInvalidateDuringComposeDropsStalePut(t *testing.T) {
	c := newAggCache(time.Minute)
	// A reader captures the epoch right after its cache miss...
	seen := c.epoch("acct-1")
	// ...a mutating write invalidates the account mid-compose (advances the epoch)...
	c.invalidate("acct-1")
	// ...so the reader's putFresh must DROP the now-stale snapshot rather than cache it.
	c.putFresh("acct-1", "dashboard", []byte("stale"), seen)
	if _, ok := c.get("acct-1", "dashboard"); ok {
		t.Fatalf("putFresh after an intervening invalidate must not cache the stale snapshot")
	}
	// A putFresh at the current epoch stores normally.
	c.putFresh("acct-1", "dashboard", []byte("fresh"), c.epoch("acct-1"))
	if got, ok := c.get("acct-1", "dashboard"); !ok || string(got) != "fresh" {
		t.Fatalf("putFresh at the current epoch should store, got %q ok=%v", got, ok)
	}
}

func TestAggCacheDisabledIsNilSafe(t *testing.T) {
	c := newAggCache(0) // disabled
	if c != nil {
		t.Fatalf("zero TTL should yield a nil cache")
	}
	// Every method must be a safe no-op on nil.
	c.put("acct-1", "dashboard", []byte(`x`))
	if _, ok := c.get("acct-1", "dashboard"); ok {
		t.Fatalf("nil cache must never hit")
	}
	c.invalidate("acct-1")
}

func TestAggCacheOverflowResetsClean(t *testing.T) {
	c := newAggCache(time.Minute)
	for i := 0; i < maxCacheEntries; i++ {
		c.put("acct", cacheKey("k", string(rune(i))), []byte("v"))
	}
	// The next put trips the overflow reset; the cache stays correct (cold), and the
	// just-written entry is present.
	c.put("acct-new", "dashboard", []byte("fresh"))
	if got, ok := c.get("acct-new", "dashboard"); !ok || string(got) != "fresh" {
		t.Fatalf("post-overflow put should be retrievable, got %q ok=%v", got, ok)
	}
}

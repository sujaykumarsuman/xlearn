package events

import (
	"testing"
	"time"
)

// backoffFor spaces out redeliveries so a hot nak loop can't burn the MaxDeliver
// budget in milliseconds (dropping the event). These pure tests pin the schedule.
func TestBackoffFor(t *testing.T) {
	last := redeliveryBackoff[len(redeliveryBackoff)-1]
	cases := []struct {
		numDelivered uint64
		want         time.Duration
	}{
		{0, redeliveryBackoff[0]},                  // defensive: treat 0 as first delivery
		{1, redeliveryBackoff[0]},                  // first delivery -> first delay
		{2, redeliveryBackoff[1]},                  // second -> second delay
		{uint64(len(redeliveryBackoff)), last},     // last explicit entry
		{uint64(len(redeliveryBackoff)) + 1, last}, // beyond the slice -> clamp to last
		{100, last}, // maxDeliver-ish -> still the last (longest)
	}
	for _, c := range cases {
		if got := backoffFor(c.numDelivered); got != c.want {
			t.Errorf("backoffFor(%d) = %s, want %s", c.numDelivered, got, c.want)
		}
	}

	// The schedule must be monotonically non-decreasing so retries only slow down.
	for i := 1; i < len(redeliveryBackoff); i++ {
		if redeliveryBackoff[i] < redeliveryBackoff[i-1] {
			t.Errorf("backoff not monotonic at %d: %s < %s", i, redeliveryBackoff[i], redeliveryBackoff[i-1])
		}
	}

	// The budget must span a real outage window, not milliseconds: sum of the first
	// maxDeliver delays (repeating the last) should be comfortably over an hour.
	var total time.Duration
	for n := uint64(1); n <= maxDeliver; n++ {
		total += backoffFor(n)
	}
	if total < time.Hour {
		t.Errorf("total redelivery window = %s, want >= 1h", total)
	}
}

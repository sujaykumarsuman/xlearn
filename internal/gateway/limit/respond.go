package limit

import (
	"encoding/json"
	"math"
	"net/http"
	"strconv"
	"time"
)

// The typed 429 codes the gateway's own limits answer with. Other services' 429s keep
// their own codes (identity's too_many_requests, coach's provider_limited, …): the SPA
// routes on the code, so none of them may collapse into these.
const (
	// CodeRateLimited is a token bucket or failure window refusing (L1, L2, L4, L5).
	CodeRateLimited = "rate_limited"
	// CodeBusy is the public-profile compose semaphore refusing (L4).
	CodeBusy = "busy"
)

// RetryAfterSeconds renders d as whole seconds for Retry-After: rounded up, at least 1.
func RetryAfterSeconds(d time.Duration) int {
	s := int(math.Ceil(d.Seconds()))
	if s < 1 {
		return 1
	}
	return s
}

// WriteTooMany writes the typed 429: a Retry-After header in whole seconds and
// {"error":{"code":…,"message":…,"retry_after":N}}.
func WriteTooMany(w http.ResponseWriter, code string, retryAfter time.Duration) {
	secs := RetryAfterSeconds(retryAfter)
	msg := "Too many requests. Try again in a moment."
	if code == CodeBusy {
		msg = "The server is busy. Try again in a moment."
	}
	w.Header().Set("Retry-After", strconv.Itoa(secs))
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusTooManyRequests)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{
		"code":        code,
		"message":     msg,
		"retry_after": secs,
	}})
}

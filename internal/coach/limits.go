package coach

import (
	"context"
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/platform/auth"
)

// L18 — the BYO coach caps (sprint m1-07 task 3; ADR-0035 §4 L18, t5 §9 "Usage display
// and limits"). They bound what a stolen xLearn session can spend from a learner's OWN
// provider key. Code constants, deliberately not env flags: nothing here joins the flag
// inventory (ADR-0034 §2).
//
// Three admission checks run on every chat, in this order, after the key lookup (no_key /
// key_disabled stay first) and BEFORE the user turn is persisted or the key decrypted:
//
//  1. streams    ≤ 2 concurrent per account   in process  429 coach_busy          Retry-After: 5
//  2. per minute 20, a token bucket            in process  429 coach_rate_limited  Retry-After: to the next token
//  3. per day    300 per UTC day               Postgres    429 coach_daily_cap     Retry-After: to the next UTC midnight
//
// A request rejected at any step consumes nothing: a later rejection gives back what an
// earlier step took. GET /admission runs the same three checks without taking anything.
//
// The in-process state (streams, buckets) resets on a coach restart and is per replica,
// which is why coach is on the L24 scale-out-blocker list (services.md); the daily cap is
// durable and shared.
const (
	// maxConcurrentStreams is how many chat replies one account may stream at once.
	maxConcurrentStreams = 2
	// busyRetryAfter is coach_busy's Retry-After: roughly one short reply's length.
	busyRetryAfter = 5 * time.Second

	// rateBurst is the token bucket's capacity, and rateRefillEvery the time one token
	// takes to come back: 20 messages a minute, with a burst of 20.
	rateBurst       = 20
	rateRefillEvery = time.Minute / rateBurst // 3s

	// dailyMessageCap is the most chat turns one account may send per UTC day.
	dailyMessageCap = 300

	// historyLimit is how many of the thread's messages a turn replays to the provider
	// (the last ones), and historyMaxBytes the most their content may total: buildTurns
	// drops the oldest until it fits, always keeping the current turn. GET /threads still
	// returns the full history.
	historyLimit    = 20
	historyMaxBytes = 32 << 10
)

// The L18 error codes (the typed JSON envelope; every one carries Retry-After), plus the
// defensive locked-mode refusal. m1-10's provider_limited 429 stays distinct.
const (
	codeCoachBusy        = "coach_busy"
	codeCoachRateLimited = "coach_rate_limited"
	codeCoachDailyCap    = "coach_daily_cap"
	codeCoachPaused      = "coach_paused"
)

// Fallback copy for the L18 rejections. The SPA renders AB01's own copy from the code and
// Retry-After; this is what a client that only reads `message` shows.
const (
	msgCoachBusy        = "you already have two coach replies in progress — wait for one to finish"
	msgCoachRateLimited = "you're sending messages faster than 20 a minute — wait a few seconds"
	msgCoachDailyCap    = "you've reached today's limit of 300 coach messages — it resets at 00:00 UTC"
	msgCoachPaused      = "the coach is paused while a mock interview is live"
)

// limitRejection is one L18 refusal: the typed code, its message, and how long to wait.
type limitRejection struct {
	code       string
	message    string
	retryAfter time.Duration
}

// retryAfterSeconds is the Retry-After header value: whole seconds, rounded up, never 0.
func (r limitRejection) retryAfterSeconds() int {
	s := int(math.Ceil(r.retryAfter.Seconds()))
	if s < 1 {
		s = 1
	}
	return s
}

func busyRejection() *limitRejection {
	return &limitRejection{code: codeCoachBusy, message: msgCoachBusy, retryAfter: busyRetryAfter}
}

// dailyCapRejection waits until the next UTC midnight after now.
func dailyCapRejection(now time.Time) *limitRejection {
	u := now.UTC()
	next := time.Date(u.Year(), u.Month(), u.Day()+1, 0, 0, 0, 0, time.UTC)
	return &limitRejection{code: codeCoachDailyCap, message: msgCoachDailyCap, retryAfter: next.Sub(u)}
}

// limiter holds L18's in-process state: per-account concurrent streams and per-account
// token buckets. One mutex guards both, so admit takes a slot and a token atomically.
//
// A bucket is stored as the instant it was (or would have been) empty: it holds
// min(now - empty, rateBurst*rateRefillEvery) of refill time, i.e. that duration divided
// by rateRefillEvery tokens. Durations keep the arithmetic exact (no float drift), and an
// account with no entry has a full bucket — so full buckets can be pruned freely.
type limiter struct {
	mu        sync.Mutex
	now       func() time.Time // the clock seam; tests make it deterministic
	streams   map[string]int
	empty     map[string]time.Time
	lastPrune time.Time
}

// pruneAbove is the bucket count above which full buckets are swept (at most once a
// minute), so the map stays bounded by the accounts active in the last minute.
const pruneAbove = 1024

func newLimiter() *limiter {
	return &limiter{now: time.Now, streams: map[string]int{}, empty: map[string]time.Time{}}
}

// fullBucket is a full bucket's refill time.
const fullBucket = rateBurst * rateRefillEvery

// availLocked is how much refill time the account's bucket holds at now (callers hold mu).
func (l *limiter) availLocked(account string, now time.Time) time.Duration {
	e, ok := l.empty[account]
	if !ok {
		return fullBucket
	}
	a := now.Sub(e)
	switch {
	case a > fullBucket:
		return fullBucket
	case a < 0: // the clock stepped back: treat as empty rather than minting tokens
		return 0
	}
	return a
}

// checkLocked runs the two in-process checks without changing anything (callers hold mu).
func (l *limiter) checkLocked(account string, now time.Time) *limitRejection {
	if l.streams[account] >= maxConcurrentStreams {
		return busyRejection()
	}
	if a := l.availLocked(account, now); a < rateRefillEvery {
		return &limitRejection{code: codeCoachRateLimited, message: msgCoachRateLimited, retryAfter: rateRefillEvery - a}
	}
	return nil
}

// admit takes one stream slot and one token for account, or neither: streams first, then
// the bucket. The caller must releaseStream when the chat ends (a defer), and
// refundToken if a later check rejects the request.
func (l *limiter) admit(account string) *limitRejection {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	if rej := l.checkLocked(account, now); rej != nil {
		return rej
	}
	l.empty[account] = now.Add(-(l.availLocked(account, now) - rateRefillEvery))
	l.streams[account]++
	l.pruneLocked(now)
	return nil
}

// probe runs the same in-process checks as admit and takes nothing (GET /admission).
func (l *limiter) probe(account string) *limitRejection {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.checkLocked(account, l.now())
}

// releaseStream gives the account's stream slot back.
func (l *limiter) releaseStream(account string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.streams[account] <= 1 {
		delete(l.streams, account)
		return
	}
	l.streams[account]--
}

// refundToken gives back the token admit took. A bucket that has since refilled (or been
// pruned as full) simply stays full.
func (l *limiter) refundToken(account string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if e, ok := l.empty[account]; ok {
		l.empty[account] = e.Add(-rateRefillEvery)
	}
}

// pruneLocked drops full buckets once the map is large (callers hold mu). Dropping a full
// bucket changes nothing: no entry reads as full.
func (l *limiter) pruneLocked(now time.Time) {
	if len(l.empty) <= pruneAbove || now.Sub(l.lastPrune) < time.Minute {
		return
	}
	l.lastPrune = now
	for a, e := range l.empty {
		if now.Sub(e) >= fullBucket {
			delete(l.empty, a)
		}
	}
}

// --- admission (the Service side: the in-process checks plus the durable daily cap) ---

// chatAdmission is one admitted chat's hold on the L18 limits: a stream slot (released by
// done, which handleChat defers so it runs when the stream ends or the client goes), a
// token and one of the day's messages (both given back by refund).
type chatAdmission struct {
	s       *Service
	account string
	day     time.Time
	once    sync.Once
}

// done releases the stream slot. Safe to call more than once.
func (a *chatAdmission) done() {
	a.once.Do(func() { a.s.limits.releaseStream(a.account) })
}

// refund gives back the token and the day's message, for a turn that was admitted but
// never reached the provider (a store or key error): it spent nothing, so it counts
// nothing. Best effort on the database side.
func (a *chatAdmission) refund(ctx context.Context) {
	a.s.limits.refundToken(a.account)
	if err := a.s.store.RefundDailyMessage(ctx, a.account, a.day); err != nil {
		a.s.log.Warn("coach: refund daily message failed", "err", err)
	}
}

// admitChat runs L18 for one chat — streams → per minute → per UTC day — and writes the
// typed 429 (or a 500 on a store error) when it refuses. On success the caller owns the
// returned admission and must defer its done.
func (s *Service) admitChat(w http.ResponseWriter, r *http.Request, accountID string) (*chatAdmission, bool) {
	if rej := s.limits.admit(accountID); rej != nil {
		writeLimit(w, rej)
		return nil, false
	}
	now := s.limits.now()
	_, ok, err := s.store.TakeDailyMessage(r.Context(), accountID, now, dailyMessageCap)
	if err != nil || !ok {
		// Nothing is consumed by a refused request: give back what the in-process checks
		// took before answering.
		s.limits.refundToken(accountID)
		s.limits.releaseStream(accountID)
		if err != nil {
			s.mapErr(w, "chat: daily quota", err)
			return nil, false
		}
		writeLimit(w, dailyCapRejection(now))
		return nil, false
	}
	return &chatAdmission{s: s, account: accountID, day: now}, true
}

// handleAdmission: GET /admission — the L18 probe. The same three checks as a chat, in the
// same order, consuming nothing: 204 when a chat would pass right now, else the same typed
// 429 + Retry-After the chat would get. The gateway calls it right before recording a D27
// assist (at most once per attempt), so an exhausted learner is told so before the attempt
// is capped. It is not a reservation: the chat still runs the real checks.
func (s *Service) handleAdmission(w http.ResponseWriter, r *http.Request) {
	accountID := auth.ClaimsFrom(r.Context()).Subject
	if rej := s.limits.probe(accountID); rej != nil {
		writeLimit(w, rej)
		return
	}
	now := s.limits.now()
	n, err := s.store.DailyMessages(r.Context(), accountID, now)
	if err != nil {
		s.mapErr(w, "admission: daily quota", err)
		return
	}
	if n >= dailyMessageCap {
		writeLimit(w, dailyCapRejection(now))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// writeLimit writes an L18 refusal: 429, the Retry-After header, the typed error envelope.
func writeLimit(w http.ResponseWriter, rej *limitRejection) {
	w.Header().Set("Retry-After", strconv.Itoa(rej.retryAfterSeconds()))
	writeError(w, http.StatusTooManyRequests, rej.code, rej.message)
}

// writePaused is the defensive answer to a locked mode reaching coach (the gateway answers
// locked itself and never forwards it): 409 coach_paused, nothing persisted, no limit
// consumed. A live mock is the only lock today (a live touch joins in M2a, and the gateway
// names it then).
func writePaused(w http.ResponseWriter) {
	writeJSON(w, http.StatusConflict, map[string]any{
		"error": map[string]any{"code": codeCoachPaused, "message": msgCoachPaused, "reason": "mock"},
	})
}

# ADR-0020 — Coach service realization: envelope crypto, the SSE path, and the behaviour gate

- **Status:** Accepted
- **Date:** 2026-09-21
- **Deciders:** @sujaykumarsuman
- **Related:** [0007](0007-ai-coach-byo-key-and-secrets.md) (governing), [0006](0006-authn-authz.md), [0003](0003-service-decomposition.md), [0014](0014-nats-jetstream-topology-and-outbox-relay.md)

## Context

[ADR-0007](0007-ai-coach-byo-key-and-secrets.md) set the coach's *design* (BYO key, envelope
encryption, in-memory decrypt, server-built Socratic/reviewer prompts). S11 builds it. A few
realization choices needed pinning down; this ADR records them so ADR-0007 stays the "why" and this
is the "how".

## Decision

### Envelope cipher: XChaCha20-Poly1305 (`golang.org/x/crypto/chacha20poly1305.NewX`)

`internal/platform/secrets` seals the raw key under a fresh per-record 32-byte **data key** and wraps
that data key under the 32-byte **master key**, both with **XChaCha20-Poly1305** — an AEAD with a
24-byte random nonce prefixed to each ciphertext. ADR-0007's parenthetical "NaCl `secretbox`" named
the *family*; the concrete primitive is the modern XChaCha20-Poly1305 AEAD (NaCl `secretbox` is the
older XSalsa20-Poly1305). Persist `enc_key` + `enc_data_key` (both nonce-prefixed) + `masked_key`
(`sk-...3f2a`). Decrypt in memory only, `Zero` the buffer the moment the provider call returns, and
return the static `ErrDecrypt` on any failure (no secret bytes in an error). The master key loads from
`COACH_MASTER_KEY` (base64/hex/raw, decoded to exactly 32 bytes) or `COACH_MASTER_KEY_FILE`; the
service **refuses to serve** without a valid key (never an ephemeral fallback — that would silently
make every stored key unrecoverable).

### Behaviour gate: a gateway-set, server-authoritative `X-Coach-Mode` header

The Socratic-vs-reviewer decision is server-authoritative and must not be client-influenced (the
spoiler-control requirement, R-AC3). Rather than have coach call back through the gateway to read
practice state (a coach→gateway auth loop with no natural credential), **the gateway — the BFF that
already aggregates practice state — derives the mode and passes it to coach as the `X-Coach-Mode`
header** (`attempt` | `review` | `general`). This is the same trust direction as every other `agg`.
Rules:

- Only a **problem** context can be an attempt or a review; every other page is the general tutor.
- **The gated problem id is the one in the thread's `context` (`problem:<id>`), never a free client
  field.** The gateway parses the id from `context`, fetches practice `GET /state/{id}` (practice-scoped
  JWT), and returns `review` only when *that* problem is genuinely **solved** (`status:"solved"` or
  `firstSolvedAt` set); otherwise `attempt`. It also **overrides** the descriptive `problemId` /
  `problemTitle` / `pattern` in the forwarded body with authoritative values (title/pattern from
  curriculum, keyed by the same id) so the coach names the problem the gate is about. Without this
  binding a client could point the solved-check at a solved problem B while the coach discusses
  unsolved problem A and leak A's solution — the exact bypass the S11 review caught.
- On **any** uncertainty — practice unreachable, a non-problem page, an unknown mode — the safe default
  is `attempt` (spoiler-free). Coach re-normalizes the header the same way, so a missing/garbage gate
  never unlocks reviewer mode on an unsolved problem.

The client's remaining descriptive fields (label, stage) only *flavour* the prompt; they never gate
behaviour, so a client that lies in them still can't turn a spoiler-free attempt into a reviewer.

### SSE all the way through, no buffering, no write-deadline cut

Coach streams the provider's tokens as `text/event-stream` frames (`data:{delta}` / `data:{done}` /
`data:{error}`); the gateway **relays** that stream byte-for-byte with per-chunk `Flush`, passing
coach's status + content-type through (so a pre-stream `409 no_key` relays as JSON, while a
post-stream provider-auth failure relays as an SSE `error` frame). Two platform requirements fell out:
`httpx.statusWriter` gained **`Unwrap()`** so `http.ResponseController` reaches the base writer's
`Flush`/`SetWriteDeadline` through the access-log middleware; and both the coach handler and the
gateway relay **clear the server `WriteTimeout`** for the streaming response (a 60s deadline would
otherwise cut a long reply). A provider auth failure (`401/403`) flips `enabled=false` so the client
routes the user back to Settings.

### Data model: single key per account; thread by page-context query param

`api_key_config.account_id` is **UNIQUE** — v1 is one active provider key per account, so `PUT
/coach/key` upserts (a second key replaces the first). `coach_thread` is unique per
`(account_id, page_context)`; `coach_message` carries a monotonic `seq` identity for a stable
history order. The external thread read is `GET /coach/thread?context=` (a query param, not
services.md's `/{pageContext}` path form) because a page context can contain characters that are
awkward in a path segment. Coach **emits no events and consumes no NATS** — page context arrives on
the request — so its schema has no outbox/inbox.

## Consequences

- ✅ A DB/backup leak alone can't decrypt keys (master key is SOPS-guarded, separate from the DB).
- ✅ The spoiler gate is genuinely un-bypassable from the browser: the authority is a header the
  gateway sets from practice, defaulting safe.
- ✅ The token stream reaches the browser live through the gateway + Traefik (Traefik does not buffer
  by default; `X-Accel-Buffering: no` + explicit flush guard against a buffering proxy).
- ⚠️ The gateway makes one extra practice call per problem-context chat to resolve the mode; it is
  best-effort (a failure degrades to the safe `attempt` default), so it never blocks the reply.
- ⚠️ Single-key-per-account means a user switching providers replaces (not accumulates) keys; multi-key
  is a later change behind the same masked-array response shape.

## Alternatives considered

| Option | Why not |
|--------|---------|
| **Coach reads context by calling the gateway** (services.md's literal "reads through the gateway") | Needs a coach→gateway credential and inverts the dependency; the gateway already holds practice access, so it enriches instead — same result, no new auth path. |
| **Trust a client-sent `stage`/`mode`** | Defeats spoiler control — a client could request reviewer mode mid-attempt and extract the solution. |
| **NaCl `secretbox` (XSalsa20-Poly1305)** | Fine, but XChaCha20-Poly1305 is the named primitive and the modern default; both are in `x/crypto`. |
| **Validate the key with a live provider call on PUT** | Adds latency + a network dependency to the store path; instead the first chat surfaces a bad key and flips `enabled=false`. |
| **Buffer the coach response in the gateway** (reuse the existing `passthrough`) | Kills streaming — the whole reply would land at once after the model finishes; a dedicated flushing relay is required. |

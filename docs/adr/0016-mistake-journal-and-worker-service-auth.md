# ADR-0016 — Mistake journal, weekly weak-area & the worker service-auth path

- **Status:** Accepted. **Refined by [ADR-0033](0033-invite-only-admission-and-owner-admin.md) (v2, Accepted 2026-09-24; see [Amended 2026-09-24 by ADR-0033](#amended-2026-09-24-by-adr-0033)):** the unauthenticated internal endpoints are fenced by the `xlearn` ingress NetworkPolicy before M3; still no service tokens.
- **Date:** 2026-09-21
- **Deciders:** @sujaykumarsuman
- **Related:** [0003](0003-service-decomposition.md) (notifications lives in review for v1),
  [0004](0004-inter-service-comms-and-events.md) (outbox + idempotent consumers),
  [0005](0005-data-ownership-and-migrations.md) (schema-per-service, soft refs),
  [0006](0006-authn-authz.md) (gateway-minted JWT; cluster-internal isolation),
  [0015](0015-five-touch-scheduler-model.md) (five-touch scheduler this extends)

## Context

S07 closes the repetition + mistakes loop (milestone M4) inside the existing **review**
service: the **mistake journal** (open on a below-Clean outcome or a failed re-solve;
close after two clean revisits; re-open on a later fail), a **weekly weak-area** rollup,
and an in-app **notifications worker**. Three design questions needed a durable call.

## Decision

### 1 · The close / re-open state machine is one transaction with the scheduler

`mistake_entry` (schema `review`) carries `category` (the 8-value R-MJ2 enum, **NULL until
the learner picks** from the seeded picker), `status` (open|closed) and `revisit_count`.
A **partial unique index** `(account_id, problem_id) WHERE status = 'open'` enforces
**at most one open entry per problem** — the idempotency key for below-clean opens and the
invariant the clean-revisit increment relies on.

The state changes ride the **same transactions** the S06 scheduler already runs
(ADR-0015), so the journal and the ladder never diverge:

- **below-Clean `problem_solved`** → inside `HandleProblemSolved`'s tx: open an entry
  (dedup via the partial index) + an outbox `mistake_opened`.
- **auto-passed re-solve** (a clean revisit) → inside `Score`'s tx: increment the open
  entry; at **2** (`MistakeCloseThreshold`) close it + emit `mistake_closed`.
- **failed re-solve** → inside `Score`'s tx (which already resets the ladder to Day 1):
  re-open a closed entry (+ `mistake_opened`), or reset an open entry's streak to 0 (no
  re-emit — a fail never counts toward the 2), or open a fresh entry if none exists.

Count **clean revisits only**; a fail resets the streak. The scheduler-reset and the
journal transition share one code path so a crash can't apply one without the other.

### 2 · The week boundary is the account timezone, not UTC

`weak_area_snapshot` is recomputed on the **same ~15-min tick as the S06 due-sweep**, one
row per account, **idempotent per `week_of`** (`UNIQUE (account_id, week_of)` upsert).
`week_of` is the **Monday of the current week in the account's timezone** — Go loads the
IANA zone and computes the Monday, then range-filters the UTC-stored `created_at` against
`[Monday 00:00 local, +7d)`. Snapshots therefore never straddle the wrong day for a user
far from UTC (verified for LA/UTC-7 and Kiribati/UTC+14). `top_category` is the max
per-category count (ties break by canonical R-MJ2 order); uncategorised entries don't
count. The read (`GET /weak-area/current`) serves the latest snapshot — cheap, no identity
call on the hot path.

### 3 · Background workers reach identity over a ClusterIP internal endpoint (no JWT)

The weak-area recompute and the notifications worker run with **no user request context**,
so they can't carry a gateway-minted JWT. They resolve account **timezone + study-budget**
via a new identity endpoint **`GET /internal/accounts/{id}`** — **unauthenticated,
ClusterIP-only**, the same trust model as identity's existing `/sessions/validate`
(ADR-0006 "v1 relies on cluster-internal isolation"). It returns **only** scheduling prefs
(timezone, study budget), never OAuth identities/email/sessions. This avoids spreading the
gateway's RSA signing key into review or inventing a service-token system for one call;
when a NetworkPolicy lands (S12) it will fence these internal endpoints. The clients
degrade safely: no `CURRICULUM_BASE_URL` → empty pattern pre-fill; no `IDENTITY_BASE_URL`
→ UTC week + immediate reminders. Pattern resolution runs only when a mistake is actually
opened (never on a clean solve), bounded by a 5s client timeout.

### 4 · Notifications is a separable worker inside review (v1)

Per ADR-0003, notifications is a **package** (`internal/review/notifications/`), not a
service: a **second durable consumer** on review's own stream `XLEARN_REVIEW`
(subject `revision_due`, durable `notifications`), its own store/account seams, idempotent
on `event_id` via the shared inbox. In-app channel only (writes `reminder` rows the
Dashboard reads). The provisional study-budget window schema (`{"windows":[{start,end}]}`,
local time) clamps a reminder into the next window; an empty budget (the v1 default)
schedules immediately. A later split into a standalone service (once email/push justifies
it) is a lift-and-shift, not a rewrite.

### Amended 2026-09-24 by ADR-0033

[ADR-0033](0033-invite-only-admission-and-owner-admin.md) §12 and §14, accepted at the v2 build-plan
sign-off, change §3's "when a NetworkPolicy lands (S12)". The text above stays as the v1 record.

- **S12 never landed a NetworkPolicy. MI-5a is the fence**, and it lands **before M3**. The `xlearn`
  ingress policy admits internal routes (`/sessions/*`, `/internal/*`) only from the `xlearn`
  namespace, admits the gateway only from Traefik, and denies the runner. The unauthenticated
  `GET /internal/accounts/{id}` keeps its trust model.
- **`/internal/accounts/{id}` grows at M4.** judge reads `tier`, `status`, the consents and
  `ai_disabled` there (5-minute cache). That is allowed only because the fence is live by then.
- **Still no service tokens.** The NetworkPolicy is the only service-auth fence, and a regression fails
  open. With no alerting (D34), `host-verify --cluster` checks that the policy is present, on demand
  ([ADR-0035](0035-v2-operations-nats-auth-limits-capacity.md) §3).

## Consequences

- ✅ M4 closed: below-Clean/fail → journal + events; close-after-2 / re-open-on-fail exact
  (verified against real Postgres); weekly weak-area tz-correct; due reviews → in-app
  reminders.
- ✅ No new service / schema / HelmRelease / DB role — additive goose migrations to schema
  `review` + two env vars on the existing Deployment; Flux image-automation ships it.
- ⚠️ The internal identity endpoint trades a JWT for network isolation; it must be covered
  by the NetworkPolicy work (S12) and exposes only non-sensitive prefs until then.
- ⚠️ Pattern resolution is a curriculum call inside the mistake-open transaction (bounded,
  low-volume, mistake paths only) — acceptable on the single node; revisit if it shows up.

## Alternatives considered

| Option | Why not |
|--------|---------|
| **Mint a service JWT in review** | Spreads the gateway's RSA signing key beyond the gateway (against ADR-0006); a shared secret/token system is heavier than one ClusterIP call needs in v1. |
| **Widen `GET /accounts/{id}`** to drop the ownership check | That route is the user-facing `/me` backing; loosening it for workers is worse than a separate, minimal internal endpoint. |
| **Carry the pattern/timezone in events** | `problem_solved` doesn't carry the pattern and the timezone changes independently; a soft-ref lookup is the ADR-0005 pattern and stays correct as data changes. |
| **A standalone notifications service now** | Only one channel (in-app) in v1; a second channel is what would justify the split (ADR-0003). Keep it a separable package until then. |
| **Recompute weak-area live on read** | Adds an identity call to every Dashboard/Mistakes load; the weekly snapshot is the cheaper single signal, refreshed on the sweep tick. |

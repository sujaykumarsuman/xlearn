# ADR-0035 — v2 operations: NATS auth, limits, capacity triggers (no alerting)

- **Status:** Accepted (2026-09-24, v2 build-plan sign-off). Its §6 amendments are folded into [ADR-0014](0014-nats-jetstream-topology-and-outbox-relay.md), [ADR-0027](0027-content-evalpack-and-user-data-model.md) and [ADR-0028](0028-object-storage-and-backups.md) (2026-09-24); [ADR-0030](0030-runner-technology-and-host-hardening.md) §7 and [ADR-0031](0031-platform-ai-and-two-tier-keys.md) §8 already carry theirs, and those two ADRs stay Proposed until their spikes. **NATS auth (N1–N3) plus the MI-5 ingress policies are a hard precondition for M3 and for erase (L-E).** "No alerting in v2" is an owner decision (D34), to be revisited before the first real invite.
- **Date:** 2026-09-24
- **Deciders:** @sujaykumarsuman
- **Related:** [0014](0014-nats-jetstream-topology-and-outbox-relay.md) (the topology this bounds and secures), [0030](0030-runner-technology-and-host-hardening.md) (Track A/B, host window; amended here), [0027](0027-content-evalpack-and-user-data-model.md) (stream budget, PAT and erase; amended here), [0031](0031-platform-ai-and-two-tier-keys.md) (spend breaker; amended here), [0028](0028-object-storage-and-backups.md) (no backups; amended here), [0029](0029-judge-contract-and-learning-signal.md) (judge queue and admission control), [0032](0032-realtime-ai-mock-interviewer.md) (L19 caps), [0033](0033-invite-only-admission-and-owner-admin.md) (`SEAT_CAP`, MI-5a/MI-5b, the admin CLI), [0034](0034-v2-release-labelling-gating-and-rollback.md) (tags, gating, migration order, rollback).
- **Links:** v2 topic T7, [feasibility § T7](../v2/feasibility.md#t7--cross-cutting-and-infra-first-rollout), the [rollout plan](../v2/rollout-plan.md) (MI track, milestone map) and the [research appendix](../v2/research/t7-cross-cutting-and-rollout.md).

## Context

Measured read-only on 2026-09-24: code at `2681862` (v1.5.0 plus docs; the live tag v1.5.1, `fff530e`, adds only F010's coach and onboarding fixes, so the identity, gateway and events citations hold, with `store.go` lines shifted by +2), `../infra` at `origin/main` `37c4fe6` (infra#28, which adds `hack/host-verify.sh` and `host-bootstrap.sh`; the local checkout was then at `d16895f` and has since been synced), and the VPS (`get`/`describe`/`top`, `configz`, the NATS monitor port, `sar`).

| Area | Today | Evidence |
|---|---|---|
| **NATS** | v2.14.6, one server, `$G` account, **no auth**. 7 connections, from practice, review and assessment only. `max_file_store` 5 Gi, `max_payload` 1 MiB | `/varz`, `/connz`; `I/infrastructure/messaging/release.yaml` |
| **Streams** | 3 streams, 4 consumers, **no limits** (`max_bytes -1`, `max_age 0`); config is hard-coded | `/jsz`; `internal/platform/events/nats.go:78-83` |
| **Poison events** | `MaxDeliver` 100 with backoff ≈ 8 h of retries, then **skipped with no record, log line or counter** | `internal/platform/events/consumer.go:30-40,176-192` |
| **identity events** | Published to a `LogPublisher`, not NATS. `XLEARN_IDENTITY` doesn't exist, and erase needs it | `internal/identity/service.go:101-105` |
| **Unknown subjects** | Acked silently (review), so a subject published before its consumer code is live is **lost** to that consumer | `internal/review/consumers.go:58-61` |
| **JetStream permissions** | Reads are governed by *publish* permission on `$JS.API.CONSUMER.*`, so a coarse `$JS.API.>` grant lets any service read, UPDATE or purge any stream | [nats-server#3202](https://github.com/nats-io/nats-server/issues/3202) |
| **nkey handshake** | nats.go refuses nkey auth when the server's INFO has no nonce (`ErrNkeysNotSupported`), and the server sends one only once nkeys are configured. **Clients-first (ADR-0030 A5) can't work** | `nats.go@v1.54.0:3065-3067`; `nats-server/v2 server/nkey.go:36-37` |
| **Alerting** | **None**: no Flux Alerts, no CronJobs, no metrics or logs stack. Views only: metrics-server, landscape, kubescope, the Longhorn UI. `sar` holds 9 days of history. Pod logs are lost on every rollout (kubelet 10 Mi × 5 per container) | `kubectl get providers,alerts,cronjobs -A`; `configz` |
| **Limits** | **None**, in the app or Traefik. Every API call makes a synchronous `/sessions/validate` round trip to identity, a 250m pod that also runs bcrypt. Bodies use a silent 1 MiB `LimitReader`. `pgxpool` defaults to max(4, NumCPU) | `gateway/bff.go:811-823`; `I/apps/xlearn-identity.yaml:70-71` |
| **NetworkPolicy** | None in `xlearn`, `databases` or `messaging` | `kubectl get networkpolicy -A` |
| **Kubelet** | **No memory eviction** (`evictionHard` covers only nodefs/imagefs; no `system-reserved`). Allocatable = capacity, so the kernel OOM killer decides | `configz` |
| **Memory** | Σ limits 9,354 Mi (58%) today, ≈ 12.6 GiB of 15.6 after v2. Plus 23 containers **without** a limit (≈ 1.16 GiB working set) and a one-commit fleet rollout surge (≈ +1.1 GiB): **the margin is ≈ 0, and negative during a fleet rollout** | `describe node`, `top`, `get pods -o json` |
| **Steal** | Elevated and noisy: 9-day p95 5.2% (daily p95 2.0–6.4% on 9/16–9/23), 2026-09-24 p95 7.1% (partial day); 1-minute p95 10%, max 16% | `sar -u`; S0 vmstat |

## Decision

### 1. Event pipeline: no new component, five code changes

JetStream plus the per-service outbox plus judge's Postgres `SKIP LOCKED` queue carry v2. Volume is ≈ 4,000 events a day at 100 learners, with ≥ 20× headroom on the relay (5 s tick × 100 rows). The gaps are closed in code:

1. **`internal/platform/events/topology.go`** is the single source of truth for streams (owner, subjects, `MaxBytes`/`MaxAge`/`Discard`) and durable consumers (stream, durable, filter, service). CI tests:
   - Σ `MaxBytes` ≤ 3.75 GiB (75% of `max_file_store`);
   - every publisher and `Subscribe` call has a table entry;
   - a **golden rendered NATS `authorization` block** (§2), stale = red;
   - the **subject-registry test**: every subject a producer can emit is in each subscriber's switch or in an explicit "ignored by X" list.
2. **Dead-letter hook.** On the last delivery of a failing message: insert `<svc>.event_dead_letter(event_id, subject, durable, err_class, at)`, call `Term()`, log at ERROR. Ids only, so it is erase-safe. The rows are durable and read on demand (§3).
3. **identity publishes `XLEARN_IDENTITY`** through its outbox to NATS. A precondition for erase.
4. **Client options**, all optional env and a no-op against today's server: `NATS_NKEY_SEED_FILE`, `NATS_INBOX_PREFIX=_INBOX_<svc>`, an `ErrorHandler` that logs permission violations at ERROR, and a relay envelope cap of 16 KiB.
5. **Pin `pgxpool` `MaxConns`**: 4 per service, 8 for judge (Σ ≤ ~40 of Postgres' 100). The default would double on a KVM 8.

These ship **dark as N0 in the next M1 tag that is ready** (MI-6). A new subject's consumer ships one tag before its producer, or the producer ships dark behind a T-2 switch ([0034](0034-v2-release-labelling-gating-and-rollback.md)).

**v2 topology** (drives the ACLs):

| Stream (owner) | `MaxBytes` · discard | Durable consumers |
|---|---|---|
| XLEARN_PRACTICE (practice) | 1 GiB · New | review, assessment, judge (analyzer), identity |
| XLEARN_REVIEW (review) | 1.5 GiB · New | review/notifications, assessment, identity |
| XLEARN_ASSESSMENT (assessment) | 128 MiB · New | identity |
| XLEARN_JUDGE (judge) | 512 MiB · 14 d · Old | practice (`evaluation_completed`), review (`evaluation_analyzed`), identity |
| XLEARN_IDENTITY (identity) | 128 MiB · New | practice, review, assessment, judge, coach |
| **XLEARN_COACH (coach)**, new: coach's erase acknowledgement (chosen over an HTTP ack, for one uniform path) | 128 MiB · New | identity |

- Σ = **3.375 of the 3.75 GiB ceiling**.
- `Discard=New` makes a full replay stream stall the relay loudly; the rows stay in the outbox, so nothing is lost.
- The NATS PVC resizes on the same 60% trigger as Postgres.

### 2. NATS auth: nkey users, fine ACLs, server first

**Model.**
- nkey users in `$G`. No new account, so streams and consumer offsets are kept.
- Keys are generated offline (`nk -gen user -pubout`).
- **Public keys** go in plaintext in the `messaging` values, so **`messaging` needs no SOPS decryption**.
- **Seeds** are SOPS files in `I/apps/secrets/xlearn-nats-<svc>.enc.yaml` (the existing `apps` rule), mounted through the chart's `extraVolumes` at `/var/run/secrets/nats/seed`.
- **Rotation or revocation:** add the new public key → roll the seed with a `podAnnotations` bump → remove the old key. All three steps are reloads. Removing a key alone revokes it.

**ACLs: fine, rendered from `topology.go`, in the same step.** Once rendered, fine costs the same as coarse, and coarse lets any service read, UPDATE or purge any stream. Per service:

| Direction | Allowed |
|---|---|
| Publish, events | `xlearn.<svc>.>` |
| Publish, own stream | `$JS.API.INFO`; `$JS.API.STREAM.{CREATE,UPDATE,INFO}.<OWN_STREAM>` |
| Publish, per consumed (stream, durable) | `$JS.API.CONSUMER.CREATE.<S>.<d>` and `….<d>.>` (a filtered create carries the filter subject); `$JS.API.CONSUMER.INFO.<S>.<d>`; `$JS.API.CONSUMER.MSG.NEXT.<S>.<d>`; `$JS.ACK.<S>.<d>.>` |
| Subscribe | `_INBOX_<svc>.>` only |
| Never | `STREAM.DELETE`, `STREAM.PURGE`, `STREAM.MSG.DELETE`, any other service's stream or durable |

**Other identities.**
- **ops** (all of `$JS.API.>`, purge included) serves the re-seal runbook (ADR-0027 §6) and stream surgery. Its seed stays **offline** with the age-key copies.
  - **Break-glass** is the only sanctioned manual NATS path: `ssh sujaykumar-vps` with `k3s kubectl port-forward` to the `nats` Service, then the `nats` CLI with the offline seed on the owner's machine. Record each use in `docs/v2/status.md`.
- **No monitoring identity.** `:8222` admits no pod. `host-verify --cluster` reads `/varz`, `/jsz` and `/connz` through the API-server proxy, which is node-local (node traffic bypasses NetworkPolicy, t3 §8.4).

**Verification before any seed is mounted.** The golden file only proves the render is stable. A **compose integration test** runs NATS 2.14 with the rendered block and, per service:
- `ensureStream`, create a filtered consumer, fetch, ack and nak;
- is **denied** publishing on another service's subject, updating another service's stream, and purging.

It must pass before N2.

**Rollout, server first** (MI-7; replaces ADR-0030 A5's "coarse, clients first"):

| Step | Where | Change | NATS | Verify | Revert |
|---|---|---|---|---|---|
| N0 | X (next M1 tag) | §1 items 1–5, dark | — | unit, golden, budget, registry and integration tests | revert |
| **N1** | I | Add every nkey user with its fine ACL, plus a `legacy` user (allow `>`, random unused password) and top-level `no_auth_user: legacy`. Restart with a pod-template annotation bump | **the one restart** (~30–60 s; the outbox buffers) | `/connz?auth=true`: every connection is `legacy` | revert + restart |
| N2 | I | Per service, mount the seed and set `NATS_NKEY_SEED_FILE` and `NATS_INBOX_PREFIX`: practice, review and assessment now; identity, judge and coach as they connect | none | each connection shows its nkey; no permission-violation logs; outbox unsent = 0; consumer pending = 0 | revert that release |
| **N3** | I | `legacy` permissions → `deny ">"` | reload | no `legacy` connection right after the reload and again ≥ 24 h later (`host-verify --cluster`) | revert |
| N4 | I (Track B, MI-15) | Remove the `legacy` user and `no_auth_user` **together** | reload (allowed on 2.10.22; **re-verify on 2.14.6**, else restart in the monthly window) | `/varz` `auth_required: true` | revert |

- **Gates:** M3 and L-E need **N3 plus MI-5**. N4 is hygiene.
- **Standing rule:** a new stream, consumer or subject needs its **infra ACL PR (the re-rendered golden block) merged before the consuming service's tag**. Otherwise that service gets permission-denied, which its `ErrorHandler` logs.

**MI-5 ingress policies (`databases`, `messaging`)** list today's callers **and forward-declare the ones that join later**. A selector for a pod that doesn't exist yet is harmless, and a missing one blocks the caller silently: identity's relay would only log "not ensured" while its outbox grows.

| Port | Admits today | Forward-declared now |
|---|---|---|
| NATS 4222 | practice, review, assessment | identity (N2, after N0), judge (MI-13, M3-1), coach (L-E erase consumer) |
| PG 5432 | the 6 DB-owning services (identity, curriculum, practice, review, assessment, coach); **not the gateway** (t3 §8.4) | judge (MI-13) |
| CNPG :8000 | `cnpg-system` only | — |
| NATS 8222 | no pod | — (the host reads it node-locally) |

Selection uses `app.kubernetes.io/instance`. Any caller not in this table updates the policy in its own infra PR, merged before its tag.

### 3. No alerting in v2 (owner: D34)

**Dropped** (supersedes the T7 research recommendation):

| Item | Was | Effect of dropping |
|---|---|---|
| Push channel | healthchecks.io (or ntfy, email, Telegram) | No ping URLs or bot tokens in SOPS |
| Host dead-man | MI-1a: an hourly `host-check` systemd timer | MI-1 keeps only Hostinger 2FA, 2 offline age-key copies, and 2FA on GitHub, Anthropic and OpenAI |
| opscheck | Hourly and daily CronJobs, the `xlearn_opscheck` role, `ops_*` views, RBAC | **MI-8 becomes the `host-verify --cluster` extension** (below). opscheck's 443 egress leaves the Track B egress list. Chart 0.3.0's `workload: cronjob` loses its only v2 user; the rollout plan owns the knob list |
| Flux notifications | A generic `Provider` plus an `Alert` on every Kustomization and HelmRelease | None |
| judge pings | `judge-runner` at M3, `judge-ai` at M4 | **The M3 egress conflict disappears:** judge stays default-deny egress until M4's 443 to the provider (ADR-0031) |
| Metrics and logs stack | VictoriaMetrics and VictoriaLogs on trigger | None in v2 |

**What the owner uses:** **landscape and kubescope** (moving to `ops.sujaykumar.dev` at MI-5b, [0033](0033-invite-only-admission-and-owner-admin.md)), plus `host-verify --cluster` after every host change (reboot, k3s upgrade, the host window).

**What remains. None of it is an alert:**
- **CI (xlearn):** the ACL golden file and integration test, the stream-budget test, the subject-registry test.
- **CI (evalpack):** the anonymous-GET probe, on every pack build.
- **Durable state:** dead-letter rows in Postgres. Rule: anything needed after the next rollout is a **Postgres row** (ledger, telemetry, dead letters, erase requests), because pod logs don't survive a rollout.
- **Product:** in-app degradation badges; judge's own spend caps and breaker (L17), which degrade to deterministic pre-fill and manual entry.
- **Provider side, outside the cluster:** the Anthropic workspace's hard limit ($15 in owner-only v2, D25's dogfood default; $100 from the v3 opening) and its Console alerts at 50%/80%; OpenAI limits on learners' own keys.

**Accepted risk.** A failure is seen only when the owner looks:

| Failure | Where it shows |
|---|---|
| Node down | the site (and landscape, on the same node) is unreachable |
| Flux object stuck | landscape or kubescope; `host-verify --cluster` |
| Poison event | `event_dead_letter` rows; an ERROR log line until the next rollout |
| Eval-pack exposure (the package made public) | the evalpack CI probe on the next pack build |
| Evalpack PAT expiry | the expiry date in `docs/v2/status.md` (manual check); judge's `evaluable = 0` badge if pulls fail |
| AI spend anomaly | judge caps, breaker and badge; the provider's hard limit |
| OOMKills, restarts, PVC growth, memory sum, steal | `host-verify --cluster` |

This is bounded because v2 is owner-only use with testers (D35). Nobody else depends on the platform until the opening.

**MI-8: the `host-verify --cluster` extension** (H, in `I/hack/host-verify.sh`, run over `ssh sujaykumar-vps`). It adds cheap, read-only checks to the existing script, run on demand and in the release checklist before contract, erase or GA tags ([0034](0034-v2-release-labelling-gating-and-rollback.md)):

| Check | Reads | Flags when |
|---|---|---|
| Memory sum (§5) | `get pods -o json`, `top` | the rule is breached, or any container has no memory limit and no budget entry |
| Flux | Kustomizations, HelmReleases | any object not Ready |
| Pod health | pods in `xlearn`, `messaging`, `databases`, `xlearn-runner` | OOMKilled, or > 3 restarts in 24 h; CNPG not healthy |
| Volumes and disk | PVCs, `df` | a PG or NATS PVC ≥ 60%; node disk ≥ 70% |
| NATS | `/varz`, `/connz?auth=true` | `auth_required` false after N4; any `legacy` connection after N3 |
| NetworkPolicy | `get networkpolicy -A` | an expected policy missing in `xlearn`, `databases`, `messaging` or `xlearn-runner` |
| Steal and CPU | `sar` | 24 h steal p95 > 10%, any 1 h average > 25%, busy > 50% (the §5 triggers) |

Cheap later additions, if wanted: stream usage ≥ 70% of `MaxBytes` (`/jsz`), consumer pending, the live `SIGNUP_MODE` and seat count, and deployed tag vs ImagePolicy latest.

**Revisit before the first real invite** (an opening gate, D35). The ready option is the T7 research design in the [appendix](../v2/research/t7-cross-cutting-and-rollout.md):
- healthchecks.io Hobbyist ($0, 20 checks) as dead-man and router;
- the hourly host timer;
- opscheck hourly plus a daily digest;
- a Flux generic Provider;
- judge pings on state change (needs M4's 443).

That is 8 of the 20 checks, with only ping URLs in SOPS. A metrics or logs stack is revisited when R2 or a second node exists, when two incidents in a quarter can't be diagnosed from sar and Postgres rows, or at the opening. It needs L23 and MI-11a first (§5).

### 4. Limits inventory

**Placement:**
- the gateway, in process, for per-IP and per-account request limits (single replica);
- the owning service, in Postgres, for durable quotas, money, seats and queue caps;
- the runner, kernel and kubelet for physical caps.

Traefik `RateLimit` isn't used: it keys only by IP, a request header or the host, not by the account behind the session cookie, and per-path limits would need IngressRoute splits. Limits fail loudly: a typed 429 with `Retry-After`, a typed 413, or a 503 for a kill switch. Nothing below exists today unless noted.

| # | Limit | Where | Start value | Milestone |
|---|---|---|---|---|
| L1 | Login | gateway (`X-Real-Ip`) | 10/min per IP (burst 5) + 5 failures / 15 min per identifier | **M1b** |
| L2 | Signup, GitHub start, invite check | gateway | 5/min per IP | M1b |
| L3 | bcrypt concurrency; dummy hash for unknown identifiers | identity | ≤ 2 in flight → 429 (protects `/sessions/validate`) | M1b |
| L4 | Public profile | gateway | 60/min per IP (burst 20); 60 s negative 404 cache; ≤ 8 concurrent cold composes | M1b |
| L5 | Per-account API token bucket (inferred values) | gateway | all `/api` 20 req/s, burst 40; mutating 5/s, burst 10 | M1b |
| L6 | Body caps, typed 413 (replaces the silent `LimitReader` at 11 sites) | gateway + judge | 1 MiB default; canvas 640 KiB; interview snapshot 64 KiB; judge scene 512 KiB, export 32 KiB | M1b (≤ M3) |
| L7 | Account creation | identity | `SIGNUP_MODE` + `SEAT_CAP` 15 ([0033](0033-invite-only-admission-and-owner-admin.md)). identity honours `open` only with `DEV_AUTH` set, which replaces the dropped opscheck assertion; production at `open` is D21's open-signup trigger ✅ `closed` live in v1.5.2 (MI-2c, xlearn#52; infra#30), which honours `open` without `DEV_AUTH`; **M1b** adds the `DEV_AUTH` guard; L builds `invite` (rehearsed on prod, then back to `closed`); `invite` in prod only at the v3 opening |
| L8 | Erase | gateway + identity | session ≤ 5 min old; typed confirmation; 1 open per account | L-E |
| L9 | Judge daily quotas | judge (PG) | 200 Runs / 100 counted / 100 arena per account | M3 |
| L10 | Lane queue caps | judge | runner 32, llm 8 → 429 `queue_full` | M3 |
| L11 | Per-account pending | judge | ≤ 2 queued Runs, ≤ 4 non-Run; ≥ 2 s between submits; 1 running per lane | M3 |
| L12 | Runner budget | judge | ≤ 10 slot-min/h, ≤ 60/day per account | M3 |
| L13 | Duty-cycle breaker | judge | 40% / 60% of 2 slots over 60 min | M3 |
| L14 | Runner hard caps | runner + `sandbox-guards` | 2 slots; compile ≤ 15 s; tests ≤ 45 s wall; ≤ 8 MiB per case; pod 2 vCPU / 3 GiB; PriorityClass −1000; Quota; LimitRange | MI-4, MI-12 |
| L15 | Kill switch | judge env | 503; the queue drains | M3 |
| L16 | Classic mock | gateway / assessment | ≤ 1 live per course; ≤ 3 started per day | M3 |
| L17 | Platform AI | Anthropic workspace + judge ledger | provider $15 / app $12 in owner-only v2 (D25's dogfood defaults, D35); provider $100 / app $80 from the v3 opening; daily guard 15%; per account $6/month, $1/day; ≤ 20 analyses and ≤ 6 AI finals a day; ≤ 8 calls per evaluation; input > 16 KiB skipped | M4 |
| L18 | BYO coach | coach | 20 msg/min; 2 concurrent streams; 300/day; history 20 turns / 32 KiB | M1 |
| L19 | Interviewer | coach (+ judge) | 1 non-terminal per account; ≤ 3 live voice platform-wide; ≤ 2 starts/day; mandatory $ cap; ≤ 1 snapshot / 2 s; mock Run ≤ 1 / 10 s | M6 |
| L20 | NATS | `topology.go` + NATS | per-stream `MaxBytes` (Σ 3.375 GiB); `max_payload` 1 MiB (kept); envelope ≤ 16 KiB; per-service ACLs | N0 (M1), MI-7 |
| L21 | PG pools | `cmd/*/main.go`; later CNPG | `MaxConns` 4 per service, judge 8; later a role `connectionLimit` of 20 | M1 (pin), MI-15 |
| L22 | CNPG memory | infra | 1 Gi (today); 2 Gi only on TR-MEM | trigger |
| L23 | **Kubelet** | host k3s config (`host-bootstrap.sh`, asserted by `host-verify`) | `system-reserved=cpu=250m,memory=1Gi`; `eviction-hard=memory.available<500Mi,nodefs.available<5%,imagefs.available<5%`; pid limits. Allocatable ≈ 14.1 GiB | **MI-11** (October host window) |
| L24 | Single gateway replica | this ADR | the in-process limiter state joins the cache epoch on the scale-out-blocker list | M1b |

`X-Real-Ip` is trusted only because Traefik sets it (`externalTrafficPolicy: Local`, no CDN). An M1b test asserts that, and the MI-5a policy ([0033](0033-invite-only-admission-and-owner-admin.md)) closes in-cluster spoofing.

### 5. Capacity: the memory-sum rule, triggers and ordered responses

**Headroom (2026-09-24).**
- CPU busy: p95 9.2%, max 33% (sar, 9 days).
- Working set: 4.38 GiB (27%).
- Requests: 1,545m / 1,612 Mi. Limits: 8,850m / 9,354 Mi.
- Disk: 19 / 193 GB. PG ≈ 10.5 MB of 10 Gi; NATS 8.4 KB of 5 Gi.

**v2 resource model:**

| | CPU req | Mem req | CPU lim | Mem lim |
|---|---|---|---|---|
| Live today (all namespaces) | 1,545m | 1,612 Mi | 8,850m | 9,354 Mi |
| + judge (M3) | 50m | 128 Mi | 500m | 256 Mi |
| + runner (MI-12) | 100m | 512 Mi | 2,000m | 3,072 Mi |
| + coach P1 (M6b) | — | — | +250m | +128 Mi |
| **v2 total** | **≈ 1.7 vCPU (43%)** | **≈ 2.2 GiB** | ≈ 11.6 vCPU | **≈ 12.6 GiB (81% of 15.6)**, rounded up; the research's opscheck (+64 Mi) is gone under D34 |
| *Outside Σ limits:* 23 containers with **no memory limit** (Longhorn, Traefik, cert-manager, metrics-server, local-path, svclb, the NATS reloader) | — | — | — | **≈ 1.16 GiB** working set |
| *Transient:* one image-automation commit rolls every xlearn Deployment at maxSurge 1 | — | — | — | **+≈ 1.1 GiB** (7 × 128 Mi + judge 256 Mi) |
| *Host:* k3s-server (1.1 GiB RSS), containerd, dockerd, system | — | — | — | 1.8–2.3 GiB |
| **MI-11a:** Flux controllers 6 × 1 GiB → 6 × 512 Mi (use 70–174 Mi) | — | — | — | **−3 GiB** of limits |

**The memory-sum rule** (replaces "Σ limits ≤ ~13 GiB ⇒ no OOM", which left out the limitless containers and the surge):

> **Σ limits + Σ p95 working set of limitless containers + the largest rollout surge + host ≤ node capacity − 0.5 GiB**

| | Sum | vs 15.62 GiB |
|---|---|---|
| v2 before MI-11a, steady | 12.6 + 1.16 + 1.8–2.3 = 15.6–16.1 GiB | **≈ 0** |
| v2 before MI-11a, fleet rollout | + 1.1 | **−1.1 to −1.5 GiB** |
| v2 after MI-11a, fleet rollout | 9.6 + 1.16 + 1.1 + 2.3 ≈ 14.2 GiB | 1.4 GiB under capacity, **≈ 0.9 GiB inside the rule** |

- **MI-11a limit hygiene** (I, October, before or with MI-11) **gates MI-12**, because the runner adds 3 GiB of limits:
  - Flux controllers to 512 Mi;
  - limits on Traefik, cert-manager and metrics-server through their values;
  - Longhorn budgeted at measured p95 × 1.5, not capped blindly (instance-manager serves the PG and NATS volumes).
- **The runner stays `strategy: Recreate`**, grace 70 s. Its Quota (limits 2 vCPU / 3 GiB) would stall a surge.
- **L23 in MI-11** (the same k3s restart as the host sandbox block) turns on kubelet memory eviction. The runner, at the lowest priority, becomes the first eviction candidate instead of whatever the kernel OOM killer picks.
- **Standing rules:**
  - any new always-on pod needs L23 plus a trim, or R1, checked against the memory sum, not Σ limits alone;
  - every new container carries a memory limit or a budget entry.

**What binds first (inferred):**
- **the platform-AI budget, at 13–17 seats** at `medium` (~29 at `low`);
- **CPU** only at ≈ 95 learners active in the same hour (M/M/2 at ρ 0.4, mean wait ≈ 0.6 s);
- memory stays flat per learner;
- PG disk reaches the 60% mark after ≈ 198 learner-years.

**Triggers.** The owner measures these on demand; nothing alerts (D34).

| Trigger | Threshold | Measured by | Response |
|---|---|---|---|
| TR-CPU | user + system 60-min average > 50% on ≥ 3 of 7 days | sar (`host-verify --cluster`) | R0 → R1 |
| **TR-STEAL** | 24 h p95 > 10% on ≥ 3 of 7 days; **or** quiet re-runs > 5% a day; **or** any 1 h average > 25%, or a Hostinger CPU-limit event | sar; judge telemetry rows | R0 (lower the breaker) → **R2**. R1 doesn't help |
| TR-MEM | the memory sum > capacity − 0.5 GiB; **or** an OOMKill outside `xlearn-runner`; **or** daily peak working set > 75% | `host-verify --cluster` | R0 (trim) → R1 |
| TR-QUEUE | counted-submit wait p95 > 10 s over a day; any counted `queue_full`; breaker level 2 > 1 h a week | judge telemetry (`judge admin`) | R0 → R2 |
| TR-SEATS | seats ≥ `SEAT_CAP` with the AI forecast < 80% → raise the cap. **`SEAT_CAP` > 40** (a T7 threshold) **or `SIGNUP_MODE=open`** (D21's trigger) ⇒ R2 first | `identity admin seats`; judge ledger | raise the cap / R2 |
| TR-DISK | a PG or NATS PVC ≥ 60%; node disk ≥ 70% | `host-verify --cluster` | Longhorn expand → R1 |
| TR-D21 | open signup; an unpatched reachable LPE > 7 d; the spike forces R1b; TR-STEAL | owner; `host-verify` | **R2** |

**Ordered responses** (Hostinger prices, 24-month term, 2026-09-24):

| Response | What | Cost | Notes |
|---|---|---|---|
| **R0** | Tune: breaker thresholds, arena shedding, seat freeze, analyzer effort `low`, trim non-xlearn limits | $0 | hours |
| **R1** | KVM 4 → **KVM 8** (8 vCPU / 32 GB) | **≈ +$21/month** at renewal ($49.99 vs $28.99) | ≤ 10 min in hPanel, then re-check the pool pins and re-calibrate time limits. **Fixes neither steal nor isolation** |
| **R2** | A new **KVM 2** as its **own single-node k3s + Flux** (`clusters/runner`). The runner is reached over 443 with mTLS plus the existing bearer token; ufw admits only the main node | **$8.99 → $14.99/month** | ≈ 2–3 days (inferred). **Never a k3s agent**: flannel is unencrypted vxlan, and an agent would hold a production kubelet credential. **Never the backup VPS** (D12, D21) |
| R3 | Off single node (HA, managed PG) | ≥ $50–100/month | out of v2 scope |

**Watch:** steal is elevated and noisy. Re-read sar after MI-0 and at the 2026-09-27 collection point. **TR-STEAL could fire before M3 and pull R2 (2–3 days) ahead of it.**

### 6. Amendments

| ADR / item | Change |
|---|---|
| **ADR-0030 §5** | Track A **A5 becomes server-first nkeys with fine ACLs** rendered from `topology.go` (N0–N4); Track B's "fine NATS ACLs" is absorbed. A4 (MI-5) forward-declares identity, coach and judge as NATS callers and judge as a PG caller (§2); `:8222` admits no pod now that opscheck is dropped. **L23 kubelet args move from Track B into the October host window (MI-11)**, and **MI-11a** joins the gates before A8 (the runner, MI-12) |
| **ADR-0027** | §3: the stream budget becomes **3.375 GiB** (+ `XLEARN_COACH` 128 MiB) under the 3.75 GiB ceiling. Consequences: "opscheck alerts 14 days before PAT expiry" becomes **a manual check** (expiry date in status.md) plus the **evalpack CI anonymous-GET probe** |
| **ADR-0031 §5** | **No push channel.** The breaker opens and degrades with an in-app badge. The spend notice is the provider's own Console alerts and hard limit, plus judge's caps. MI-14 loses "judge → healthchecks.io" |
| **ADR-0028 §4** | "opscheck watches disk and volume usage" → `host-verify --cluster` reports PVC and disk usage on demand. The 60% resize trigger is unchanged |
| **ADR-0014** | Stream config moves out of `nats.go` into `topology.go`, with limits. "No auth in v1" ends at N3 |
| **Back-pointers** | 0014, 0027, 0028, 0030 and 0031 carry a status note pointing here |
| **Superseded T-items** | T1's `xlearn-opscheck` CronJob plus push channel (dependency matrix); T5's "a push channel judge can call" before M4; T6's "opscheck interview counters". The M6 release-runbook check for live interviews stays ([0034](0034-v2-release-labelling-gating-and-rollback.md)). The matrix's NATS row loses "`messaging` gains SOPS decryption" |

## Consequences

- ✅ **No new platform component, vendor or cost.** Five small code changes and one NATS restart.
- ✅ **NATS stops being an open bus.** Forging, cross-stream reads, and UPDATE or purge of another service's stream are closed before M3 and erase, with zero event loss (the outbox buffers).
- ✅ **Poison events leave a durable row** instead of vanishing after ≈ 8 h.
- ✅ **Overload fails loudly and locally** (typed 429/413/503). A login flood can no longer starve `/sessions/validate`.
- ✅ **L23 plus MI-11a turn a negative worst-case memory margin into ≈ 0.9 GiB inside the rule**, and kubelet eviction picks the runner first.
- ✅ **Capacity responses are priced and ordered**, and the first binding constraint (AI seats) is known.
- ⚠️ **No alerting (D34).** Node loss, a stuck Flux object, dead letters, an eval-pack exposure, PAT expiry or a spend anomaly are seen only when the owner looks. Accepted for owner-only v2; revisit before the first real invite.
- ⚠️ **No metrics history.** Diagnosis relies on sar, Postgres rows and pod logs that are lost on each rollout.
- ⚠️ **A standing cross-repo coupling:** every new stream, consumer or subject needs its infra ACL PR merged before the service tag.
- ⚠️ **The memory margin stays thin** after MI-11a. Every new always-on pod is checked against the sum, and a metrics stack needs a trim or R1.
- ⚠️ **Steal is elevated and noisy.** R2 may be needed before M3.
- ⚠️ **N4's reload path is verified only on nats-server 2.10.22**; 2.14.6 may need a restart in the monthly window.
- ⚠️ **The single gateway replica** now also holds limiter state. That's a scale-out blocker, recorded with the cache epoch.

## Alternatives considered

| Option | Why not |
|---|---|
| Workflow engine (Temporal, Restate) | A judge job is a one-table state machine, erase is an idempotent fan-out, the interview is coach's FSM; ≈ 1–2 GiB (inferred) |
| Kafka or Redpanda | ≈ 4 orders of magnitude more throughput than needed; ≥ 1 GiB |
| Redis for limiter counters | One gateway replica holds in-process buckets; durable quotas are Postgres counts. Revisit at gateway scale-out |
| A DLQ stream | A dead-letter row is durable, erase-safe and needs no infra |
| NATS user/password (clients first, no restart) | A weaker model; hashes or SOPS end up in `messaging` |
| A dedicated NATS account | JetStream is per account: streams are recreated and consumer offsets reset |
| Operator/JWT mode; mTLS client certs | Overkill for 6 identities; more moving parts than nkeys |
| Coarse ACLs now, fine later (ADR-0030 A5/Track B) | `$JS.API.>` lets any service read, UPDATE or purge any stream (nats-server#3202); fine costs the same once rendered |
| Clients-first nkeys (ADR-0030 A5) | nats.go fails with `ErrNkeysNotSupported` against a server with no nonce |
| MI-5 admitting the whole `xlearn` namespace | Simpler, but it admits the internet-facing gateway to PG and (before N3) to anonymous NATS; forward-declared selectors cost the same |
| A single-reconcile nkey flip | An ordering race that converges only if every reconnect path retries |
| healthchecks.io + host timer + opscheck now (the T7 research recommendation) | Owner declined for now (D34): monitoring is landscape and kubescope while v2 is owner-only. Kept ready for the revisit |
| ntfy, email or a Telegram bot as the channel | Can't report their own node down; a token or API key in the cluster |
| VictoriaMetrics + VictoriaLogs now | ≈ 0.4–0.6 GiB, which breaks the memory-sum rule before MI-11a and L23; upkeep |
| Traefik `RateLimit` | Keys by IP, header or host, not by account; would need per-path IngressRoute splits |
| Leave the kubelet as is | The kernel OOM killer picks victims, not priority |
| R1 (KVM 8) for steal or isolation | Fixes neither |
| R2 as a k3s agent | Unencrypted vxlan over the internet, and a production kubelet credential on the runner host |
| The backup VPS as the runner | Reserved for backups (D12); never runs learner code (D21) |

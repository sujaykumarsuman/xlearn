# Prompt — Sprint m6b-02 · Voice robustness: lease, re-attach, drain, cost + $ cap, PTT, rollover (+ coach-interview if M7 failed)

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-m6b-02.md`](../sprints/sprint-m6b-02.md)   ·   **Milestone:** M6b (voice, one shell)   ·   **Prereqs:** [m6b-01](../sprints/sprint-m6b-01.md)

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md) — conventions, service boundaries, GitOps, land-and-sync.
- The plan: [`../sprints/sprint-m6b-02.md`](../sprints/sprint-m6b-02.md) — the sideband-lease table, handoff protocol, drain order, cap/idle rules,
  the caps table with its typed errors and the ordered M7-failed steps are spelled out there. Follow them.
- Research [t6 §3](../research/t6-realtime-interviewer.md#3-architecture--media-path) ("Surviving deploys", caps table),
  [§4](../research/t6-realtime-interviewer.md#4-session-state-machine--the-owners-failsafes) (state machine, grace, rollover, failure table incl. idle
  and coach restart), [§5](../research/t6-realtime-interviewer.md#5-the-coding-round) (quiet during Code, hold voice),
  [§7](../research/t6-realtime-interviewer.md#7-privacy-consent-retention--accessibility) (modes), [§8](../research/t6-realtime-interviewer.md#8-cost-on-byo-keys)
  (cost and the estimate), [§10](../research/t6-realtime-interviewer.md#10-the-smallest-spike) (M3b, M5, M7), and the **S6 results** — t6 **§16**
  ("for m6b-02": rollover rule, reseed TTFA, push-to-talk mechanics, cache ratio) appended by [spk-04](../sprints/sprint-spk-04.md), the
  fixtures in `docs/v2/research/t6-s6-fixtures/`, and the spike-results row in [`../status.md`](../status.md).
- [ADR-0032](../../adr/0032-realtime-ai-mock-interviewer.md) §2 (M7 fallback), §4 (failsafes), §6 (cap, caps);
  [ADR-0035 §4](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#4-limits-inventory) (L19, L21) and
  [§5](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#5-capacity-the-memory-sum-rule-triggers-and-ordered-responses) (memory-sum rule);
  [ADR-0035 §2](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#2-nats-auth-nkey-users-fine-acls-server-first) (NetworkPolicy standing rule);
  [ADR-0034 §1.1](../../adr/0034-v2-release-labelling-gating-and-rollback.md#11-scheme) (patches after GA),
  [§2](../../adr/0034-v2-release-labelling-gating-and-rollback.md#2-feature-gating-three-tiers-no-flag-service) (dark, cohort),
  [§3](../../adr/0034-v2-release-labelling-gating-and-rollback.md#3-migration-and-event-compatibility-rules) (expand/contract, floors),
  [§6](../../adr/0034-v2-release-labelling-gating-and-rollback.md#6-release-checklist) (release checklist);
  [rollout §2.2](../rollout-plan.md#22-operating-rules-every-mi-step-and-every-tag) (image before HelmRelease or policy; infra PRs stand alone).
- Neighbours: [m6b-01](../sprints/sprint-m6b-01.md) (what exists: `VoiceShell`/`Sideband` with `Detach` ≠ `HangUp`, `voice.Segment`, the broker,
  `voiceAdmission`, the reaper job, lease-takeover supersede, `caption`/`segment` events), [m6a-01](../sprints/sprint-m6a-01.md) (FSM table with
  `held`/`bridging` already in the enum, clock, client lease, heartbeat + idle rule, sweeper under `pg_try_advisory_lock`, `AddSpend`, UTC day,
  SIGTERM checkpoint, `coach admin interviews --live`, `docs/runbooks/interviewer.md`), [m6a-02](../sprints/sprint-m6a-02.md) (`Segment`, `Prime`,
  director), [m6a-04](../sprints/sprint-m6a-04.md) (event log + bounded SSE: in-process notifier + 1 s poll), [mi-13](../sprints/sprint-mi-13.md)
  (coach grace 60 s, rollingUpdate 1/0, the memory-sum margin), [m6b-03](../sprints/sprint-m6b-03.md) (consumes the events; expects
  `coach-interview` live if M7 failed, and bumps it after its tag).
- Code: `cmd/coach/main.go` (shutdown today: 10 s), `internal/coach/interview/` (M6a + `voice/`), `internal/coach/store/`,
  `internal/coach/catalog.go`, the gateway interview proxy, `docs/architecture/{services,api,data-model,events}.md`,
  `docs/architecture/openapi.yaml`, `docker-compose.yml`, `docs/runbooks/interviewer.md`. Infra (M7-failed only; edit through PRs):
  `../infra/apps/{xlearn-coach,xlearn-gateway,image-automation}.yaml`, `../infra/charts/project/{values.yaml,templates/networkpolicy.yaml}`
  (xlearn policies are **chart-rendered per release**; `networkPolicy.enabled` defaults to `false`), the MI-15 egress values
  ([mi-11](../sprints/sprint-mi-11.md)), the MI-5 `databases` ingress policy under `../infra/infrastructure/database/cluster/` and MI-5a's
  per-release ingress ([mi-03](../sprints/sprint-mi-03.md)), `../infra/hack/{expected-netpol.tsv,host-verify.sh,host-lint.sh}`
  ([mi-02](../sprints/sprint-mi-02.md)).

## Context

[m6b-01](../sprints/sprint-m6b-01.md) gave coach voice: the SDP broker, the outbound sideband that drops audio unparsed, `voice.Segment` (so every
FSM close hangs up), captions on the event log. It has no protection against coach restarts, no voice metering and no voice caps. Flux rolls
coach on every release tag (one replica, `maxSurge 1 / maxUnavailable 0`, 60 s grace since [mi-13](../sprints/sprint-mi-13.md)); audio runs
browser ↔ OpenAI, so a restart only cuts the **sideband** — if a new pod attaches before the old one leaves, the call never notices (S6 **M7**).
This sprint builds that sideband lease, handoff and drain; prices voice usage into M6a's `AddSpend` so the **mandatory $ cap** works on voice;
enforces the **L19 voice caps** (≤ 3 live platform-wide, ≤ 75 voice-min/day per account × multiplier) with typed 429s; adds the "Still there?"
idle prompt; and adds push-to-talk, Patient, hold, rollover, AI notes checkpoints and transcript self-edit. If S6 found **M7 failed**, it also
stands up the fallback — a separately pinned `coach-interview` Deployment — live and routed before [m6b-03](../sprints/sprint-m6b-03.md), which
means a `v2.0.x` patch tag first (image before HelmRelease). Everything stays dark behind the cohort gate.

## Entry gates — verify first (stop and report if any is unmet)

- [ ] [m6b-01](../sprints/sprint-m6b-01.md) is merged (`internal/coach/interview/voice/` on `main`).
- [ ] t6 §16 gives M7 (pass/fail), M3b (push-to-talk native/emulated), M5 (duration / `expired`), `usage_ratio` at 60 min, reseed TTFA, the
      cached-token ratio and the M9 usage-event shape. **M7 decides task 7** — write it down before starting.
- [ ] `ssh vps 'sudo k3s kubectl get deploy -n xlearn xlearn-coach -o jsonpath="{.spec.strategy}{.spec.template.spec.terminationGracePeriodSeconds}"'`
      shows rollingUpdate 1/0 and 60 (read-only); status.md holds mi-13's memory-sum margin.
- [ ] You have read m6a-01's UTC day rule, idle rule, client lease, sweeper lock and SIGTERM checkpoint, and m6a-04's stream notifier.
- [ ] Parallel sessions: `gh pr list` (xlearn and `../infra`), `git ls-remote --tags origin`, `git worktree list`, ListAgents — no open PR adds a
      coach migration or edits `internal/coach/interview/`, `cmd/coach/main.go`, the gateway interview proxy, `apps/xlearn-coach*.yaml`,
      `apps/xlearn-gateway.yaml`, `apps/image-automation.yaml` or the `xlearn` NetworkPolicies; if task 7 applies, note the next free `v2.0.N`.

## Do this (in order)

1. **[X] Branch** `feat/m6b-02-voice-robustness` off an up-to-date `origin/main` (M7-failed only: `feat/xlearn-coach-interview-policies` and
   `feat/xlearn-coach-interview` in `../infra`).
2. **[X] Migration + sqlc** (expand, next free coach version): `interview_sideband_lease` (plan task 1) and `interview.voice_mode`. Nothing else —
   `usage`/`cost_micros`/`spent_micros`/`cap_micros` and the `held`/`bridging` states exist since m6a-01. `sqlc generate`; `sqlc diff` clean.
3. **[X] Sideband lease** (plan task 1): 5 s renew / 15 s TTL; the holder-only rule for all provider I/O, FSM-driven closes included; NOTIFY
   wake-ups (`snapshot`, `ptt`, `hold`, `mode`, `ask`, `close`, `handoff`, `taken` — never content); a `pg_notify` wake after each `events.Append`
   for cross-pod captions; a dedicated LISTEN `pgx.Conn` outside the pool (pool stays `MaxConns` 4); make-before-break (second sideband → first
   event → epoch CAS → `taken` → old pod `Detach()`es without hanging up and zeroes its buffer); cold re-attach as a job in m6a-01's sweeper,
   and on refusal `interrupt(provider)` → free re-prime with `segment{event:"reprime", gap_ms}`; `coach admin interviews --live` gains
   holder/epoch/segment/shell/purpose.
4. **[X] Drain** (plan task 2): `shutdownTimeout` 10 s → **50 s**; `503 draining` (`Retry-After: 2`) on new segments and
   `srv.SetKeepAlivesEnabled(false)` — **no readiness flip is relied on** (coach's HelmRelease probes `/healthz` for readiness; the chart's
   `probes.readinessPath` opt-in isn't set for coach); m6a-01's `sigterm` checkpoints
   then handoffs (≤ 30 s); expire leftovers and detach without hang-up; stop the erase consumer subscription **before** `srv.Shutdown`; zero
   buffers; exit.
5. **[X] Voice usage → `AddSpend`** (plan task 3): price sideband usage per the catalog (per-second, or tokens incl. cached) into the segment's
   `usage`/`cost_micros` and `AddSpend`; `cap_100` closes the segment → hang-up; `cap` events for AB30 F10; 409 `cap_reached` at mint below one
   minute of headroom; review + one re-propose outside the live cap but inside "plan for"; `voice.estimate{…}` table-tested against t6 §8 (±5%).
   Token-billed shells use a **speaking profile derived in coach** from the snapshot's rail and t6 §8's assumptions (named constants in
   `voice/estimate.go`) — there is no manifest `mock.voice_profile` (no sprint adds it; unknown manifest fields are rejected), so no
   curriculum or schema change; record that in the Decisions log.
6. **[X] Voice caps + idle** (plan task 4) in `voiceAdmission` and the holder's meter: ≤ 3 live voice under
   `pg_advisory_xact_lock(hashtext('coach.voice_live'))` → **429 `voice_capacity`** `Retry-After: 60`; ≤ 75 min/day × multiplier since 00:00 UTC
   → **429 `voice_daily_limit`** with `Retry-After`, and live: a 3-minute wrap cue, then `interrupt(voice_limit)` + `save_later` → `paused`;
   `minutes_left_today` and `reason ∈ {capacity, daily_limit}` in the `voice` block; speech events feed `last_input_at`; `notice{idle_check}` 60 s
   before m6a-01's `interrupt(idle)`.
7. **[X] Modes, hold, grace** (plan task 5): `voice_mode` + `POST …/voice-mode` (standard ↔ ptt mid-call) + `POST …/ptt {down|up}` (holder
   executes via NOTIFY if the request lands elsewhere; ≤ 10/s) with the winner's native or emulated settings; quiet during Code; FSM rows
   `live —hold→ held —talk→ connecting`, `POST …/hold` (hang-up, clock runs, buffer zeroed) and Talk via `/segments` from `held`; `held` also
   takes `live`'s `rail_done`/`finish`/`cap_85` → `wrapping` and `cap_100` → `finished(cut_short=cap)` rows (a debrief entered from `held` is
   text turns); **`mic`** as a new client-postable interrupt reason next to `voice_limit` (m6b-03 posts it after 30 s of mic loss; the clock
   stops at the interrupt); on `ErrQuota` the segment closes at once; `notice{probe, checked_at}` per probe. Gateway interview-proxy rows +
   `openapi.yaml`/`api.md` for every new route.
8. **[X] Rollover, checkpoints, self-edit, mirror** (plan task 6): FSM rows `live —rollover→ bridging —segment_open→ live`; Realtime rollover at the
   start of Code (> 40 min or ~24k tokens; forced for a 60-minute rail or multiplier > 1×), GPT-Live on a short S6 duration limit or
   `usage_ratio` > 0.8; close-then-open with the bridge line; `Prime` ≤ 8,192 tokens; `ErrSessionCap` → same path; a creation error on the new
   offer → a voice-only `bridging —interrupt→ interrupted` row (m6b-01's `connecting` effects); AI notes checkpoints at phase
   boundaries; `PATCH …/turns/{seq}` (candidate turns, `finished`/`proposed` only, ≤ 2 KiB, `edited` → `transcript_edited` → `self`);
   `POST …/turns/mirror` (≤ 20 × 2 KiB, only inside a re-prime gap, `source='client'`, else 409 `no_gap`).
8b. **[X] Only if S6 M7 failed** (recorded at entry; plan task 7a) — otherwise skip and mark rows 7a–7e ✅ "n/a — M7 passed". All of it goes
   in the step-11 PR:
   - `COACH_ROLE ∈ {all, core, interview}`: `interview` serves `/interviews/*` + `/internal/interviews/*` and runs the sweeper; `core` serves
     keys/chat/threads/models, the outbox relay and the erase consumer, and 404s the interview paths; the **lagging-schema test**;
   - **`interview` never connects to NATS** (no publisher, relay or consumer even if `NATS_URL` is set — a `LogPublisher` relay would mark
     `core`'s erase acks sent) + its test;
   - gateway `COACH_INTERVIEW_BASE_URL` chosen by **coach upstream path**: the `/api/interviews/*` rows **and** the gateway's internal
     `POST /internal/interviews/{id}/exposure` push, the resume/finish backstop and every `GET /interviews/active` read (exposure push,
     coach-lock derivation, submit orchestration) + a call-site test;
   - `docs/runbooks/interviewer.md` § "coach-interview (M7 fallback)" (manual bump, rollback, contract floor, the policy shape).
9. **[X] Tests** (plan task 8 — see the plan's acceptance): the **in-process two-instance restart test** (make-before-break with no provider
   hang-up; crash → cold re-attach ≤ 20 s; refused → re-prime, TTFA ≤ 3 s on the fake); the drain < 50 s with 3 sessions; cap 85/100 and
   `cap_reached`; `voice_capacity`; the **76th-minute `voice_daily_limit`** at 1× (and 112.5 min at 1.5×); idle notice + hang-up; PTT across
   instances; hold (and `held` → `wrapping` on rail end / finish / cap); `interrupt(mic)` and `interrupt(voice_limit)` hang up; rollover (and a
   failed rollover offer → `interrupted`); self-edit and mirror → no `ai-byo`; the estimate table (rail-derived profile). Then a **compose run** with two coach containers on the
   same DB and the scratchpad fake provider (never committed): start a fake call, SIGTERM the holder, confirm the fake saw a second sideband
   before the first detached and no hang-up — paste the log excerpt into the PR ([m6b-04](../sprints/sprint-m6b-04.md) relies on it).
   `go test ./...`, `go vet`, lint, `sqlc diff`, OpenAPI drift.
10. **[X] Optional live restart check** — **only with the owner's explicit go-ahead in chat** (≈ $0.20): the same compose setup on the owner's own
    key (he enters it himself), a 3-minute Chrome fake-media call, SIGTERM the holder; confirm audio continues and captions resume. Otherwise
    record "S6 M7 + fakes + compose run are the evidence".
11. **[X] Docs + status**, then PR → CI green → squash-merge.
12. **[I · X · I · X] Only if S6 M7 failed** (plan tasks 7b–7e; the role code is already merged with step 11) — otherwise stop after step 11:
    - **I — PR A (peer-side NetworkPolicies only; plan 7b)**: `apps/xlearn-gateway.yaml` egress admits `xlearn-coach-interview` :8086; the
      MI-5 `databases/projects-pgstore-ingress` instance list gains `xlearn-coach-interview` on 5432; verify (no change) that identity :8081
      and the gateway JWKS :8080 admit same-namespace pods; no messaging change. `coach-interview`'s **own** policy can't go here — xlearn
      policies are chart-rendered per release. Render diff, `--dry-run=server`; merge once CI is green; Flux reconciles; smoke.
    - **X — tag `v2.0.N`** (plan 7c) after the plan's release checklist (no live interviews; next free; major = `.release-line`); title
      `v2.0.N — v2.1 build · M6b backend (dark) + coach-interview roles`; verify healthz, images, ImagePolicies, HelmReleases, smoke.
    - **I — PR B (the Deployment; plan 7d)**, only when `coach admin interviews --live` is empty and `host-verify --cluster` shows the memory
      sum fits (+256 Mi limits): `apps/xlearn-coach-interview.yaml` copying `xlearn-coach`'s values (`replicaCount: 1`, `COACH_ROLE=interview`,
      **no `NATS_URL`/nkey**, 500m / 256 Mi, grace 60, 1/0, `tag: 2.0.N` + setter marker) **with its own NetworkPolicy** (`enabled: true`,
      `from: null`, same-namespace ingress on 8086, egress DNS / PG 5432 / identity :8081 / gateway :8080 / 443 with mi-11's except-list);
      the `hack/expected-netpol.tsv` row `xlearn<TAB>xlearn-coach-interview<TAB>m6b-02` + the embedded copy in `host-verify.sh`
      (`host-lint.sh` clean); ImagePolicy `xlearn-coach-interview` on ImageRepository `xlearn-coach` pinned to **exactly `2.0.N`**;
      `COACH_ROLE=core` on `xlearn-coach`; `COACH_INTERVIEW_BASE_URL` on `xlearn-gateway`. Merge; verify `xlearn-coach-interview` Ready on
      `2.0.N` with its NetworkPolicy present, the owner's `GET /xlearn/api/v1/interviews/active` answered, an arena page with no exposure
      WARN in the gateway log, coach chat still fine, `host-verify --cluster` green.
    - **X — records (plan 7e)**: a small `docs(status)` PR with the status.md records (step 11's PR has merged by now).

## Constraints

- **Service boundaries ([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)):** coach alone owns interviews, keys, provider sessions and the
  caps; the gateway only proxies (plus, M7-failed, chooses the upstream). No cross-schema reads; limits are **durable in Postgres**, never in
  process (two pods overlap on every rollout).
- **goose + sqlc:** expand-only migration, next free coach version at rebase; generated code committed; `sqlc diff` clean. A coach **contract**
  migration later must respect the `coach-interview` floor (M7-failed).
- **Privacy:** NOTIFY payloads, `usage`, logs and the admin CLI never carry transcript, audio, SDP or key bytes; buffers zeroed on every handoff.
- **Provider I/O only in the lease holder**, epoch-checked. No retries on session creation. Never conflate the sideband lease with m6a-01's client lease.
- **No new events on NATS:** subjects, streams and consumers unchanged → no ACL PR; consumers-before-producers not in play.
- **Dark:** cohort-gated API only; no new flag (`COACH_ROLE`, `COACH_INTERVIEW_BASE_URL` are deploy-shape config — record them in status.md's
  config notes, not the flag inventory).
- **Release discipline (M7-failed only):** PR A before the tag, PR B after it (image before HelmRelease/policy); the tag is a **patch**; **no live
  interviews** before the tag and before PR B; never move or re-push a tag; parallel-sessions check before tagging.
- **GitOps:** infra changes only via PRs reconciled by Flux; never `kubectl apply`/`scale`/`rollout` by hand; `ssh vps` for reads, `host-verify.sh`
  and the `coach admin` CLI only; the `coach-interview` ImagePolicy is pinned to an existing tag and moved only by PR.
- **Memory-sum rule ([ADR-0035 §5](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#5-capacity-the-memory-sum-rule-triggers-and-ordered-responses)):**
  every container keeps a memory limit; the `coach-interview` sum is checked before PR B merges.
- **NetworkPolicy standing rule:** every new in-cluster caller gets its policy in its own infra PR — the peers' side (gateway egress →
  `coach-interview`, the `databases` ingress for `coach-interview` → PG) in PR A, merged before the tag; `coach-interview`'s own ingress and
  egress are chart-rendered with its release, so they ride PR B and exist before its pod does (identity and the gateway JWKS already admit
  same-namespace callers).
- **D34:** no counters, alerts, opscheck or Flux `Alert`; `coach admin interviews --live` is the ops read.
- **Keys:** never type or paste an API key; live checks need the owner to enter his own.
- **Parallel sessions:** re-check peers' PRs, tags and worktrees before merging or tagging; claim an ADR number only after that check.

## Deliverables

- The sideband lease/handoff/cold re-attach, cross-pod wake-ups, the drain, voice pricing into `AddSpend`, the cap hang-up and headroom check,
  the estimate, the L19 voice caps, the idle notice, modes/PTT/hold, voice grace notices, rollover, AI notes checkpoints, transcript self-edit,
  client mirror — in `internal/coach/interview/` and `cmd/coach/main.go`.
- The coach migration + sqlc output; gateway interview-proxy rows; `openapi.yaml`/`api.md`/`services.md`/`data-model.md`/`events.md`.
- Tests listed in the plan and the compose-run log excerpt in the PR.
- **M7 failed only:** `COACH_ROLE` (no NATS in the `interview` role) + `COACH_INTERVIEW_BASE_URL` for every coach interview path + their
  tests; `../infra` PR A (peer-side NetworkPolicies) and PR B (`coach-interview` HelmRelease with its own NetworkPolicy, the
  `expected-netpol.tsv` row, pinned ImagePolicy, `COACH_ROLE=core`, gateway env); tag `v2.0.N`; the runbook section; the records PR.

## Update status

- [`../sprints/sprint-m6b-02.md`](../sprints/sprint-m6b-02.md): each task 🔄 → ✅ (⛔ with a reason; rows 7a–7e "n/a — M7 passed" when applicable);
  _Overall_ ✅ when all are.
- [`../status.md`](../status.md): Sprint board row; **Milestones** M6b stays 🔄; flags — none new; **Decisions log** — sideband lease TTL/renew and
  the handoff protocol, cross-pod wake-ups, the drain budget, "review + re-propose outside the live cap", the UTC voice-minute day and the
  live-limit → `paused` rule, the `mic` reason (clock stops at the 30 s interrupt), `held`'s rail/finish/cap rows, PTT native vs emulated,
  rollover close-then-open, the estimate's coach-derived speaking profile (no manifest `mock.voice_profile`), the drain without a readiness
  flip, the compose-run result, whether the live restart check ran.
  **M7 failed only:** milestone → tag → floor → snapshot (`v2.0.N`, floor unchanged, no snapshot), PR A/B numbers, the pinned version, the memory
  sum before/after, config notes (`COACH_ROLE`, `COACH_INTERVIEW_BASE_URL`), the `coach-interview` contract floor, and a **hand-off line for
  m6b-03** ("after your tag: bump `coach-interview` per `docs/runbooks/interviewer.md`").
- No ADR expected (ADR-0032 §2 already names the fallback). If the handoff protocol has to diverge from t6 §3, write one — its number claimed
  only after the parallel-sessions check.

## Done when (acceptance)

- [ ] Restart during a call keeps audio (make-before-break, no provider hang-up) or reseeds within TTFA ≤ 3 s (cold path) — in-process test and
      compose run green
- [ ] SIGTERM drain < 50 s, no provider hang-up, buffers zeroed
- [ ] The cap stops the session with a clear message (85% wrap, 100% hang-up + `finished(cut_short=cap)`; `cap_reached` at mint)
- [ ] The 76th voice-minute of the day → typed 429 `voice_daily_limit` (1×); a 4th live voice → 429 `voice_capacity`
- [ ] `notice{idle_check}` 60 s before `interrupt(idle)`, which hangs up
- [ ] Standard / Patient / push-to-talk per the S6 winner; hold keeps the clock and reseeds free on Talk
- [ ] Rollover primes ≤ 8,192 tokens, close-then-open at a phase boundary; self-edit and mirror turns block `ai-byo`
- [ ] (M7 failed only) PR A (peer side) → `v2.0.N` → PR B (with its own NetworkPolicy + `expected-netpol.tsv` row) in order;
      `coach-interview` Ready on `2.0.N` and serving `/api/interviews/*` and the gateway's `/internal/interviews/*` calls; no NATS in the
      `interview` role; memory sum inside the rule; runbook + hand-off recorded
- [ ] CI green; merged

Ship at session end per AGENT.md land-and-sync with **this sprint's release action — merge only (ships dark in m6b-03's `v2.0.x` patch)**:
branch → conventional commit(s) with the attribution lines → push → PR → CI green → squash-merge → no tag. **Only if S6 M7 failed:** then
merge `../infra` PR A (peer-side NetworkPolicies) → run the release checklist and **tag `v2.0.N`** → let Flux deploy and verify → merge PR B
(the `coach-interview` Deployment with its own NetworkPolicy, pinned ImagePolicy, routing) with no live interviews → verify live → the
records PR. This conditional tag deviates from the register's "merge only" on purpose (image before HelmRelease; the plan's Release says
why). Finally `git checkout main && git pull` in every repo touched.

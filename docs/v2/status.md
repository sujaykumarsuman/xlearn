# xLearn v2 — status

Cross-sprint living tracker for the v2 build. Updated per the
[status protocol](sprints/README.md#status-protocol-way-of-working) whenever a task or sprint changes state (it's
baked into every `prompt-<id>.md`). Per-task detail lives in each [`sprints/sprint-<id>.md`](sprints/); the static
plan is [`build-plan.md`](build-plan.md); the source is the [rollout plan](rollout-plan.md); the one-page run order
with workstream diagrams is [`execution-order.md`](execution-order.md). Newest decisions at the
top of the log. v1's tracker is [`../v1/status.md`](../v1/status.md).

- **Build line:** v2 · **Phase:** **build plan scaffolded** (2026-09-24): 89 sprints (86 plan + prompt pairs, 3 M6c
  outline cards), the ADR sign-off (BP2) and the calendar (BP4). **Next: the MI-0 H0 reboot, Fri 2026-09-25
  (pending, owner)**, then week 1: [mi-01](sprints/sprint-mi-01.md) (first), [ds-m1-01](sprints/sprint-ds-m1-01.md),
  [mi-02](sprints/sprint-mi-02.md), [m1-01](sprints/sprint-m1-01.md), [mi-07](sprints/sprint-mi-07.md),
  [ds-m2-01](sprints/sprint-ds-m2-01.md), [ds-l-01](sprints/sprint-ds-l-01.md).
- **Live release:** **v1.5.2** (all seven services; signup closed in production since 2026-09-24, `SIGNUP_MODE=closed`).
- **Deploy mode:** release-semver tags. The v2 build ships as **1.x minors** until `v2.0.0` (D32, ADR-0034):
  `.release-line = 1`; every `xlearn-*` ImagePolicy is `>=1.0.0 <2.0.0`. Runner and evalpack are separate streams.
- **Audience:** owner only (D35). Testers exercise the non-owner paths; real learners arrive at v3.
- **Sprint merge rule (D40, 2026-09-25):** launching a sprint prompt is the owner's approval for every change it
  makes; every prompt ends with land-and-sync, and nothing waits on the owner mid-session ([decisions log](#decisions-log)).
- **Last updated:** 2026-09-25 (spk-03 done: **WIF GO**, with `check_jti=false` on the one-rule issuer. The rule scope is `workspace:developer`, because the Console offers no `workspace:inference`, so ADR-0031 amendments are proposed for mi-12 ([t5 §15](research/t5-platform-ai.md#15-wif-spike-result-spk-03-2026-09-25)). Earlier: spk-02 done: **MI-10 ✅**, Spike P0–P3 GO and image volume GO. Block 3 narrowed the AppArmor remount rules; block 2 gave Q-C GO, the amd64 allowlists and the x86_64-only proposal; block 1 gave image volume GO and the GOCACHE seed. Earlier: spk-02 partial results, ⛔ rows open; earlier the same day, D40: the sprint merge directive applied to the plan, status and indexes; before
  that, 2026-09-24: the v2 build-plan session with the build plan, status, sprint + prompt scaffolds and ADR sign-off).

## Snapshot

| Area | State |
|------|-------|
| v2 PRD | ✅ [drafted](../prd/xlearn-v2-prd.md) |
| Feasibility (D0–D40; D36–D39 = BP1–BP4; D40 = the sprint merge directive) + research T0–T7 | ✅ [feasibility](feasibility.md), [research/](research/) |
| Rollout plan (the source) | ✅ [rollout-plan.md](rollout-plan.md) |
| ADRs 0026, 0027, 0028, 0029, 0033, 0034, 0035 | ✅ **Accepted 2026-09-24** (BP2; amendments folded into the ADRs they amend) |
| ADRs 0030, 0031, 0032 | Proposed until their spikes report (see [ADRs](#adrs)) |
| v2 build plan | ✅ [build-plan.md](build-plan.md) |
| Sprint plans + prompts | ✅ all scaffolded: 89 plans, 86 prompts (M6c = 3 outline cards) · [sprints](sprints/README.md) · [prompts](prompts/README.md) |
| Artboards (`design-system/screens/v2/`) | ⬜ 0 / 29 drafted, 0 frozen (AB01–AB30 less the dropped AB23; AB31 is an outline for v2.2) |
| Application code (v2) | ⬜ not started · live `v1.5.2` |
| `infra` | ✅ infra#28 (`host-bootstrap` + `host-verify`), ✅ infra#29 (ranges `<2.0.0`), ✅ infra#30 (`SIGNUP_MODE: closed`) · MI-2 onward ⬜ |
| Host | kernel update pending the **H0 reboot (MI-0, Fri 2026-09-25)**; pre-reboot gate GO (45/45, 2026-09-24) |
| Content | ⬜ item schema not frozen · pilot packs **0 / 14** stamped |

## Sprint board

**Pre-build stopgaps (done before this plan; not sprints):**

| Item | What | Where | State |
|------|------|-------|-------|
| MI-2a | Accidental-major guard: every `xlearn-*` ImagePolicy `>=1.0.0 <2.0.0` + the `.release-line` check in `deploy.yml` | infra#29 · xlearn#53 (`363dee9`) | ✅ Done 2026-09-24 |
| MI-2b | GitHub auto-links only into password-less accounts (pre-account hijack closed; amends ADR-0023 §3) | xlearn#51 (`e915479`) | ✅ Done 2026-09-24 · live in v1.5.2 |
| MI-2c | `SIGNUP_MODE ∈ {open, closed}`, production `closed` (D13 enforced) | xlearn#52 (`1b90d2b`) · infra#30 | ✅ Done 2026-09-24 · live in v1.5.2 |
| infra#28 | `host-bootstrap` + `host-verify` merged; local `../infra` `main` synced | infra#28 | ✅ Done 2026-09-24 |

**Sprints** (recommended order; tracks interleave, only prereqs bind; see [build-plan](build-plan.md#sprint-table-recommended-order)):

| # | Sprint | Focus | MS | Track | Release | State |
|---|--------|-------|----|-------|---------|-------|
| 1 | [mi-01](sprints/sprint-mi-01.md) | Prune guards + chart 0.3.0 knob union (MI-2, MI-3) | MI | infra | infra PRs only (MI-2 first, then MI-3) | ⬜ Not started |
| 2 | [ds-m1-01](sprints/sprint-ds-m1-01.md) | Design M1: coach states, course nav, revision v2 (AB01–AB03) | M1 | design | land-and-sync; the merge is the design freeze | ⬜ Not started |
| 3 | [mi-02](sprints/sprint-mi-02.md) | host-verify --cluster extension (MI-8) | MI | infra | infra PR only (host script) | ⬜ Not started |
| 4 | [m1-01](sprints/sprint-m1-01.md) | Curriculum spine part 1: compose parity, course manifest + golden, item schema freeze (M1a) | M1 | product | merge only (ships in v1.6.0) | ⬜ Not started |
| 5 | [mi-07](sprints/sprint-mi-07.md) | Evalpack plumbing (MI-9) | MI | infra | infra PRs only (+ evalpack repo; `v0.1.0` image, no `>=1.0.0`) | ⬜ Not started |
| 6 | [ds-m2-01](sprints/sprint-ds-m2-01.md) | Design M2: Touch ★, catalog/agenda, public profile v2, visibility (AB04 ★, AB05, AB06, AB22) | M2 | design | land-and-sync; the merge is the design freeze | ⬜ Not started |
| 7 | [ds-l-01](sprints/sprint-ds-l-01.md) | Design L: invite acceptance ★, privacy notice, erase account (AB19 ★, AB20, AB21) | L | design | land-and-sync; the merge is the design freeze | ⬜ Not started |
| 8 | [m1-09](sprints/sprint-m1-09.md) | Curriculum spine part 2: converter, loader + guards, curriculum expand migration, content CI (M1a) | M1 | product | merge only (ships in v1.6.0) | ⬜ Not started |
| 9 | [m3-01](sprints/sprint-m3-01.md) | Authoring tooling T25/T26: canonical hashes, contract_hash, packlint, pre-push fingerprint hook | M3 | content | merge only (dev tools; rides the next tag) | ⬜ Not started |
| 10 | [mi-05](sprints/sprint-mi-05.md) | N0: NATS topology, dead letters, identity on NATS, client options, pool pins (MI-6) | MI | product | merge only (ships dark in v1.6.0) | ⬜ Not started |
| 11 | [m1-02](sprints/sprint-m1-02.md) | path_slug everywhere + v2 envelope consumers + identity M1a columns → v1.6.0 | M1 | product | **tag v1.6.0** (+ infra PR before) | ⬜ Not started |
| 12 | [mi-14](sprints/sprint-mi-14.md) | sandbox-guards: empty default-deny xlearn-runner namespace + VAP (MI-4) | MI | infra | infra PRs only | ⬜ Not started |
| 13 | [mi-03](sprints/sprint-mi-03.md) | Fences: databases/messaging ingress, xlearn ingress (MI-5, MI-5a) | MI | infra | infra PRs only (MI-5 → MI-5a) | ⬜ Not started |
| 14 | [m3-02](sprints/sprint-m3-02.md) | Evalpack pipeline: private CI gates, generic validator, image build (E) + compose fixture pack | M3 | content | merge only (evalpack `main`; optional `v0.2.0`) | ⬜ Not started |
| 15 | [mi-04](sprints/sprint-mi-04.md) | Admin consoles to ops.sujaykumar.dev (MI-5b) | MI | infra | infra PRs only | ⬜ Not started |
| 16 | [spk-01](sprints/sprint-spk-01.md) | Sandbox mechanism spike P0–P2 (multipass arm64, throwaway) | MI | spike | no merge (spike, throwaway); results docs PR | ✅ Done — Q-A GO, Q-B GO ([t3 §16.1](research/t3-sandbox.md#161-p0p2-spk-01-arm64-multipass)) |
| 17 | [spk-02](sprints/sprint-spk-02.md) | P3 amd64 replay + image-volume spike (throwaway) | MI | spike | no merge (spike, throwaway); results docs PR | ✅ Done 2026-09-25 (re-run in three blocks after the interrupted first session): **Spike P0–P3 GO, image volume GO.** Block 1: image volume GO (kubelet defaults) and the GOCACHE seed. Block 2: Q-C GO (TSAN needs no ASLR policy; amd64 allowlists at 0 unexpected SIGSYS; x86_64-only pod profile proposed). Block 3: the AppArmor `remount,` finding is closed, because `ro`-only scoped rules refuse read-write remounts of `/`, `/sys` and `/jail/**`. Env A torn down ([t3 §16.2–16.4](research/t3-sandbox.md#162-p3-amd64-replay-spk-02)) |
| 18 | [spk-03](sprints/sprint-spk-03.md) | WIF spike (≤ ½ day, throwaway) | MI | spike | no merge (spike, throwaway); results docs PR | ✅ Done 2026-09-25: **WIF GO, with `check_jti=false`** on the one-rule issuer. `jti` is present, and an in-place restart re-presents it (an opaque 401 until the next rotation). Exchange p50 0.33 s; first call p50 1.61 s; workspace header 5/5. The scope is `workspace:developer`, and ADR-0031 amendments are proposed for mi-12. VM purged ([t5 §15](research/t5-platform-ai.md#15-wif-spike-result-spk-03-2026-09-25)) |
| 19 | [mi-06](sprints/sprint-mi-06.md) | NATS auth server-first: N1 → N2 → N3 (MI-7) | MI | infra | infra PRs only (N1, N2, N3) | ⬜ Not started |
| 20 | [m1-03](sprints/sprint-m1-03.md) | Course resolution: gateway, BFF, SPA /:course/*, producers emit v2 (M1b) | M1 | product | merge only (ships in v1.7.0) | ⬜ Not started |
| 21 | [m1-04](sprints/sprint-m1-04.md) | Identity security floor: roles, sessions, admin CLI, DEV_AUTH guard, L3, CSP (M1b) | M1 | product | merge only (ships in v1.7.0) | ⬜ Not started |
| 22 | [m1-10](sprints/sprint-m1-10.md) | Coach keys + provider hygiene: key_default(feature), catalog, AEAD AD, keyring re-wrap, store:false, classifier, usage | M1 | product | merge only (ships in v1.7.0) | ⬜ Not started |
| 23 | [m1-05](sprints/sprint-m1-05.md) | Gateway limits + public-dashboard floor (L1, L2, L4–L6, L24; P1, P2, P4, P10, P11) | M1 | product | merge only (ships in v1.7.0) | ⬜ Not started |
| 24 | [m1-06](sprints/sprint-m1-06.md) | withhold() on every surface + Markdown renderer + revision v2 (AB03) | M1 | product | merge only (ships in v1.7.0) | ⬜ Not started |
| 25 | [mi-08](sprints/sprint-mi-08.md) | Limit hygiene (MI-11a) + Track B slice 1: PSA labels, SA tokens off (MI-15 part) | MI | infra | infra PRs only (by the 10-24 window) | ⬜ Not started |
| 26 | [mi-09](sprints/sprint-mi-09.md) | October host-window prep: sandbox block, L23 kubelet args, pid limits, bumps (MI-11) | MI | infra | infra PRs only (scripts applied in the window) | ⬜ Not started |
| 27 | [m1-07](sprints/sprint-m1-07.md) | Coach D27 assist capture, mode gate, L18 caps + AB01 → v1.7.0 | M1 | product | **tag v1.7.0** | ⬜ Not started |
| 28 | [ds-m3-01](sprints/sprint-ds-m3-01.md) | Design M3 part 1: Workspace-Code ★, results dock, degradation badges (AB07 ★, AB08, AB11) | M3 | design | land-and-sync; the merge is the design freeze | ⬜ Not started |
| 29 | [m1-08](sprints/sprint-m1-08.md) | M1c contract → v1.8.0 | M1 | product | **tag v1.8.0** (contract; snapshot first) | ⬜ Not started |
| 30 | [m3-03](sprints/sprint-m3-03.md) | Runner core: supervisor, jail, cgroups, API | M3 | product | merge only (ships in runner-v1.0.0) | ⬜ Not started |
| 31 | [m3-04](sprints/sprint-m3-04.md) | Runner profiles Go/C++/Python + harness codecs + amd64 allowlists | M3 | product | merge only (ships in runner-v1.0.0) | ⬜ Not started |
| 32 | [m3-15](sprints/sprint-m3-15.md) | Runner release: reproducible image, runner-release.yml, acceptance suite, TL baselines → runner-v1.0.0 | M3 | product | **runner-v1.0.0** | ⬜ Not started |
| 33 | [m2-01](sprints/sprint-m2-01.md) | M2a touch-attempt engine + `touch_concluded` consumer | M2 | product | merge only (ships in v1.9.0) + own infra PR | ⬜ Not started |
| 34 | [m2-02](sprints/sprint-m2-02.md) | M2b consumers + projections v2 → v1.9.0 | M2 | product | **tag v1.9.0** (+ ACL PR before) | ⬜ Not started |
| 35 | [ds-m3-02](sprints/sprint-ds-m3-02.md) | Design M3 part 2: Problems, Arena, Week/Mistakes/Progress deltas (AB09, AB10, AB12) | M3 | design | land-and-sync; the merge is the design freeze | ⬜ Not started |
| 36 | [m2-03](sprints/sprint-m2-03.md) | `public-read`, `/public/stats`, visibility toggles, profile v2 (P3, P5, P6, P7, P9; AB06, AB22) | M2 | product | merge only (ships in v1.10.0) | ⬜ Not started |
| 37 | [m2-04](sprints/sprint-m2-04.md) | Touch UI (AB04★) + catalog/agenda + Today in minutes (D4) (AB05) | M2 | product | merge only (ships in v1.10.0) | ⬜ Not started |
| 38 | [m2-05](sprints/sprint-m2-05.md) | Producers on + `touch_scored` backfill + reader switch + replay + D2 (M2c) → v1.10.0 | M2 | product | **tag v1.10.0** | ⬜ Not started |
| 39 | [l-01](sprints/sprint-l-01.md) | L-E erase consumers (practice, review, assessment, coach) + identity ack path → v1.11.0 | L | product | **tag v1.11.0** (+ ACL/seed PRs before) | ⬜ Not started |
| 40 | [l-02](sprints/sprint-l-02.md) | L-E erase producer + `DELETE /api/me` + UI (AB21) → v1.12.0 | L | product | **tag v1.12.0** (erase; snapshot first) | ⬜ Not started |
| 41 | [mi-10](sprints/sprint-mi-10.md) | Runner dark on production (MI-12) | MI | infra | infra PRs only (runner dark) | ⬜ Not started |
| 42 | [m3-05](sprints/sprint-m3-05.md) | judge service: skeleton, schema, evalpack loader, erase consumer (M3-1) | M3 | product | merge only (ships dark in v1.13.0) | ⬜ Not started |
| 43 | [m3-06](sprints/sprint-m3-06.md) | judge queue, runner lane, graders, contexts, internal context endpoints | M3 | product | merge only (ships dark in v1.13.0) | ⬜ Not started |
| 44 | [m3-14](sprints/sprint-m3-14.md) | judge admission (L9–L15, L6), learner API + DTO allowlist, drafts, arena history/progress, telemetry, judge admin | M3 | product | merge only (ships dark in v1.13.0) | ⬜ Not started |
| 45 | [ds-p-01](sprints/sprint-ds-p-01.md) | Design P: Workspace-Quiz, go-concurrency multi-file + race, AB02/AB05 full fidelity (AB14, AB15) | P | design | land-and-sync; the merge is the design freeze | ⬜ Not started |
| 46 | [m3-07](sprints/sprint-m3-07.md) | M3-1 on prod: evalpack v1.0.0, judge ACL, tag v1.13.0, judge HelmRelease (MI-13) | M3 | product | **evalpack v1.0.0 + tag v1.13.0** (ACL PR before, HelmRelease after) | ⬜ Not started |
| 47 | [l-03](sprints/sprint-l-03.md) | L-A admission backend (inert while closed): invites, seats, redeem, public checks, `SIGNUP_MODE=invite`, erase clears invite notes | L | product | merge only (ships dark, inert, in the next tag) | ⬜ Not started |
| 48 | [m3-08](sprints/sprint-m3-08.md) | practice: judge consumer, reconciler, D15/D16/D18 grading strategy | M3 | product | merge only (ships in v1.14.0) + ACL PR | ⬜ Not started |
| 49 | [m3-09](sprints/sprint-m3-09.md) | gateway judge BFF, DTO allow/deny lists, typed 413, L16, degradation status | M3 | product | merge only (ships in v1.14.0) | ⬜ Not started |
| 50 | [m3-10](sprints/sprint-m3-10.md) | review + assessment on judge signals (mistake pre-fill, P7 checked, P8) | M3 | product | merge only (ships in v1.14.0) | ⬜ Not started |
| 51 | [ds-m4-01](sprints/sprint-ds-m4-01.md) | Design M4: AI suggestion/dispute ★, pointer notes, allowance + consents (AB16 ★, AB17, AB18) | M4 | design | land-and-sync; the merge is the design freeze | ⬜ Not started |
| 52 | [mi-11](sprints/sprint-mi-11.md) | Track B finish: xlearn egress, PG connection limits, Renovate, N4 (MI-15) | MI | infra | infra PRs only (+ xlearn PR, no tag) | ⬜ Not started |
| 53 | [m3-11](sprints/sprint-m3-11.md) | Workspace-Code UI (AB07★) + CodeMirror lazy load | M3 | product | merge only (ships in v1.14.0) | ⬜ Not started |
| 54 | [m3-12](sprints/sprint-m3-12.md) | Results dock, Problems, Arena, degradation badges (AB08–AB11) | M3 | product | merge only (ships in v1.14.0) | ⬜ Not started |
| 55 | [m3-13](sprints/sprint-m3-13.md) | Week/Mistakes/Progress deltas (AB12) + M3 exit tests → v1.14.0 | M3 | product | **tag v1.14.0** (+ `JUDGE_BASE_URL` PR after) | ⬜ Not started |
| 56 | [mi-12](sprints/sprint-mi-12.md) | M4 gates: judge 443 egress, LLM secret, provider runbook (MI-14) + ADR-0031 → Accepted | MI | infra | infra PRs only (+ ADR-0031 docs PR) | ⬜ Not started |
| 57 | [p-01](sprints/sprint-p-01.md) | go-race runner profile → runner-v1.1.0 | P | product | **runner-v1.1.0** (judge code rides v1.15.0) | ⬜ Not started |
| 58 | [p-02](sprints/sprint-p-02.md) | Multi-course: go-concurrency manifest (preview), catalog, agenda, nav | P | product | merge only (ships in v1.15.0) | ⬜ Not started |
| 59 | [p-03](sprints/sprint-p-03.md) | Quiz widget (AB14) + race verdict UI (AB15) → v1.15.0 | P | product | **tag v1.15.0** + **evalpack v1.N.0** | ⬜ Not started |
| 60 | [m4-01](sprints/sprint-m4-01.md) | platform/llm extraction + WIF auth + retention policy | M4 | product | merge only (ships in v1.16.0) | ⬜ Not started |
| 61 | [m4-02](sprints/sprint-m4-02.md) | judge ai: Scorer, ledger, llm lane, breaker, caps (L17) | M4 | product | merge only (ships in v1.16.0) | ⬜ Not started |
| 62 | [m4-03](sprints/sprint-m4-03.md) | Analyzer (D16/D26), pointer notes, `evaluation_analyzed` + acceptance-set harness | M4 | product | merge only (ships in v1.16.0) + ACL PR | ⬜ Not started |
| 63 | [m4-04](sprints/sprint-m4-04.md) | Provisional grades, dispute, re-grade, honor claims (D14) | M4 | product | merge only (ships in v1.16.0) | ⬜ Not started |
| 64 | [m4-05](sprints/sprint-m4-05.md) | AI allowance + consents (`account_consent` AI kinds) | M4 | product | merge only (ships in v1.16.0) | ⬜ Not started |
| 65 | [m4-06](sprints/sprint-m4-06.md) | M4 UI: AI suggestion/dispute (AB16★), pointer notes (AB17), allowance + consents (AB18) | M4 | product | merge only (ships in v1.16.0) | ⬜ Not started |
| 66 | [m4-07](sprints/sprint-m4-07.md) | Canary log test, caps sizing, ledger check → v1.16.0 (+ LLM_PLATFORM_ENABLED for the cohort) | M4 | product | **tag v1.16.0** (+ `LLM_PLATFORM_ENABLED` PR) | ⬜ Not started |
| 67 | [l-05](sprints/sprint-l-05.md) | L-A/L-C front door: auth-page invite state, acceptance step★, privacy notice, AI consents (AB19★, AB20) | L | product | merge only (ships in v1.17.0) | ⬜ Not started |
| 68 | [l-04](sprints/sprint-l-04.md) | Web erase for every non-owner → v1.17.0 (L-A/L-C) + L-exit rehearsal prep | L | product | **tag v1.17.0** (erase-class; + rehearsal PRs) | ⬜ Not started |
| 69 | [ga-01](sprints/sprint-ga-01.md) | GA PR: .release-line = 2, T-1 default flips, rc rehearsal | GA | product | merge only: the GA PR merges on CI green (+ optional `v2.0.0-rc.N`) | ⬜ Not started |
| 70 | [ga-02](sprints/sprint-ga-02.md) | Cut v2.0.0: pre-flip check, widen ranges, snapshot, tag, verify | GA | product | **tag v2.0.0** (widen first; GA snapshot) | ⬜ Not started |
| 71 | [spk-04](sprints/sprint-spk-04.md) | S6 voice-shell bake-off (owner present, throwaway) | M6a | spike | no merge (spike, throwaway); results docs PR | ⬜ Not started |
| 72 | [ds-m6a-01](sprints/sprint-ds-m6a-01.md) | Design M6a part 1 + ADR-0032 → Accepted: Mock-v2, setup/consent/pre-flight, live HUD text (AB13, AB24, AB25) | M6a | design | ADR-0032 docs PR, then land-and-sync; the merge is the design freeze | ⬜ Not started |
| 73 | [ds-m6a-02](sprints/sprint-ds-m6a-02.md) | Design M6a part 2: grace/paused/resume, debrief + proposal, accessibility (AB26, AB27, AB28) | M6a | design | land-and-sync; the merge is the design freeze | ⬜ Not started |
| 74 | [m6a-01](sprints/sprint-m6a-01.md) | Interview core: schema, state machine, failsafes, caps (L19) | M6a | product | merge only (ships dark in a v2.0.x patch) | ⬜ Not started |
| 75 | [m6a-02](sprints/sprint-m6a-02.md) | Text brain: loop, classifier, give_hint, resume brief, store:false, replay tests | M6a | product | merge only (ships dark in a v2.0.x patch) | ⬜ Not started |
| 76 | [m6a-03](sprints/sprint-m6a-03.md) | Assessment deltas + proposal/accept + ScoreMock once + twin fairness gate | M6a | product | merge only (ships dark in a v2.0.x patch) | ⬜ Not started |
| 77 | [m6a-04](sprints/sprint-m6a-04.md) | judge mock budgeting + CodeMirror interview mode + bounded SSE | M6a | product | merge only (ships dark in a v2.0.x patch) | ⬜ Not started |
| 78 | [m6a-05](sprints/sprint-m6a-05.md) | M6a UI part 1: Mock-v2 (AB13), setup/consent/pre-flight/$ cap (AB24), live HUD text (AB25) | M6a | product | merge only (ships dark in a v2.0.x patch) | ⬜ Not started |
| 79 | [m6a-06](sprints/sprint-m6a-06.md) | M6a UI part 2: grace/paused/resume (AB26), debrief/proposal (AB27), accessibility (AB28) → v2.0.x patch | M6a | product | **tag v2.0.x patch** (M6a dark, cohort) | ⬜ Not started |
| 80 | [ds-m6b-01](sprints/sprint-ds-m6b-01.md) | Design M6b: voice pre-flight/notices, voice live HUD ★ (AB29, AB30 ★) | M6b | design | land-and-sync; the merge is the design freeze | ⬜ Not started |
| 81 | [mi-13](sprints/sprint-mi-13.md) | M6b gates: Permissions-Policy, camera/mic deny, coach sizing, SDP route, WSS egress (MI-16) → v2.0.x patch | MI | infra | **tag v2.0.x patch** (MI-16) + infra PRs first | ⬜ Not started |
| 82 | [m6b-01](sprints/sprint-m6b-01.md) | VoiceShell adapter + SDP broker + sideband | M6b | product | merge only (ships dark in m6b-03's patch) | ⬜ Not started |
| 83 | [m6b-02](sprints/sprint-m6b-02.md) | Voice robustness: lease, re-attach, drain, cost + $ cap, PTT, rollover (+ coach-interview if M7 failed) | M6b | product | merge only (ships dark in m6b-03's patch) | ⬜ Not started |
| 84 | [m6b-03](sprints/sprint-m6b-03.md) | Voice UI: pre-flight/notices (AB29), live HUD★ (AB30), consent, EU gate, self-view → v2.0.x patch (M6b dark) | M6b | product | **tag v2.0.x patch** (M6b dark, cohort) | ⬜ Not started |
| 85 | [m6b-04](sprints/sprint-m6b-04.md) | Fake-media e2e + interviewer GA flip → v2.1.0 | M6b | product | **tag v2.1.0** (interviewer GA) | ⬜ Not started |
| 86 | [m5-01](sprints/sprint-m5-01.md) | DSA evaluator-only flip (M5) | M5 | product | rides v2.1.0 if merged before m6b-04, else **tag the next minor ≥ v2.2.0** | ⬜ Not started |
| 87 | [m6c-01](sprints/sprint-m6c-01.md) | Outline: "show your work" photo + chat read-aloud + local recording download | M6c | product | outline only (v2.2) | ⬜ outline (v2.2; no prompt until expanded) |
| 88 | [m6c-02](sprints/sprint-m6c-02.md) | Outline: multi-speaker fairness check → AI-proposed voice Communication; history-calibrated estimates | M6c | product | outline only (v2.2) | ⬜ outline (v2.2; no prompt until expanded) |
| 89 | [m6c-03](sprints/sprint-m6c-03.md) | Outline: Safari leg, voice-lite on demand, weekly canary; AB31 canvas outline (SD course, v2.2+) | M6c | product | outline only (v2.2) | ⬜ outline (v2.2; no prompt until expanded) |

Legend: ✅ done · 🔄 in progress · ⬜ planned / not started · ⛔ blocked (note why: an unmet entry gate, a missing
owner prerequisite, or "needs owner decision" after a spike). Nothing waits on the owner mid-session (D40). Update per the
[status protocol](sprints/README.md#status-protocol-way-of-working).

## Milestones

| ID | Target | Closed by | Tag(s) | State |
|----|--------|-----------|--------|-------|
| M0 namespace rule + coach P0 fixes | done | — | v1.5.0, v1.5.1 (+ v1.5.2 stopgaps) | ✅ live |
| MI cluster safe for untrusted code | Sep 25 → Nov (+ mi-13 Q1 2027) | [mi-12](sprints/sprint-mi-12.md), [mi-13](sprints/sprint-mi-13.md) | infra PRs; runner-v1.0.0 | ⬜ (MI-2a/2b/2c ✅; MI-0 pending) |
| M1 spine (M1a → M1b → M1c) | October | [m1-08](sprints/sprint-m1-08.md) | v1.6.0 → v1.7.0 → v1.8.0 | ⬜ |
| M2 attempt engine + projections | late October | [m2-05](sprints/sprint-m2-05.md) | v1.9.0 → v1.10.0 | ⬜ |
| L learner gate (L-E → L-A/L-C → exit rehearsal) | Nov → Dec | [l-04](sprints/sprint-l-04.md) + `ev-l-rehearsal` | v1.11.0 → v1.12.0 → v1.17.0 | ⬜ |
| M3 judge + code grader (M3-1 → M3-2) | November | [m3-13](sprints/sprint-m3-13.md) | runner-v1.0.0 · evalpack v1.0.0 · v1.13.0 → v1.14.0 | ⬜ (hard checklist below) |
| P pilot course (go-concurrency) | December | [p-03](sprints/sprint-p-03.md) | runner-v1.1.0 · v1.15.0 | ⬜ |
| M4 platform AI (owner cohort) | December | [m4-07](sprints/sprint-m4-07.md) | v1.16.0 | ⬜ |
| **GA v2.0.0** (owner-facing default flip) | ≈ Dec 2026 – Jan 2027 | [ga-02](sprints/sprint-ga-02.md) | v2.0.0 | ⬜ |
| M6a text interviewer | Q1 2027 | [m6a-06](sprints/sprint-m6a-06.md) | v2.0.x (dark) | ⬜ |
| M6b voice → **v2.1.0** interviewer GA | Q1 2027 | [m6b-04](sprints/sprint-m6b-04.md) | v2.0.x → v2.1.0 | ⬜ |
| M5 DSA evaluator-only | ≈ H2 2027 | [m5-01](sprints/sprint-m5-01.md) | rides v2.1.0, or next minor ≥ v2.2.0 | ⬜ |
| M6c interviewer extras | v2.2 | v2.2 planning | v2.2 | ⬜ outline |
| Opening (v3) | v3 | the owner's call ([rollout §11](rollout-plan.md#11-opening-gates-v3)) | — | not in v2 |

## MI track (rollout §2)

The **MI table**. Rollout step ids `MI-NN` are **not** sprint ids `mi-NN`; the Sprint column maps them.

| Step | What | Sprint(s) / event | Risk | State | PRs · date · notes |
|------|------|-------------------|------|-------|--------------------|
| **MI-0** | H0 reboot into kernel 6.8.0-142 (copy `/tmp/xlearn-s0-vmstat.log` off first; `host-verify --pre-reboot --cluster`; weekly image date; reboot; `host-verify --cluster`) | `ev-mi0` (recorded in [mi-02](sprints/sprint-mi-02.md)) | ◐ | ⬜ pending Fri 2026-09-25 | pre-reboot gate GO 45/45 (2026-09-24) |
| MI-1 | Owner hygiene: Hostinger 2FA, 2 offline age-key copies, 2FA on GitHub/Anthropic/OpenAI | `ev-mi1` | ○ | ⬜ | week 1 |
| ~~MI-1a~~ | host-check timer / dead-man | — | — | dropped (D34) | — |
| MI-2 | Prune guards (CNPG Cluster, `databases`, `messaging`) | [mi-01](sprints/sprint-mi-01.md) | ○ | ⬜ | week 1, first |
| MI-2a | Accidental-major guard | pre-build | ○ | ✅ | infra#29 · xlearn#53 · 2026-09-24 |
| MI-2b | Auto-link fix | pre-build | ○ | ✅ | xlearn#51 · live v1.5.2 |
| MI-2c | `SIGNUP_MODE` closed | pre-build | ◐ | ✅ | xlearn#52 · infra#30 · live v1.5.2 (the `DEV_AUTH` guard on `open` is M1b: [m1-04](sprints/sprint-m1-04.md)) |
| MI-3 | Chart 0.3.0 knob union + `hack/chart-diff.sh` | [mi-01](sprints/sprint-mi-01.md) | ○ | ⬜ | — |
| MI-4 | `sandbox-guards` (empty default-deny `xlearn-runner` + VAP) | [mi-14](sprints/sprint-mi-14.md) | ○ | ⬜ | — |
| MI-5 | NetworkPolicies `databases` + `messaging` (forward-declared callers) | [mi-03](sprints/sprint-mi-03.md) | ◐ | ⬜ | gates N3, L-E, M3 |
| MI-5a | `xlearn` ingress (gateway ← Traefik; internal routes ← `xlearn`; runner denied) | [mi-03](sprints/sprint-mi-03.md) | ◐ | ⬜ | — |
| **MI-5b** | Admin consoles → `ops.sujaykumar.dev` | [mi-04](sprints/sprint-mi-04.md) + `ev-mi5b-dns` | ◐ | ⬜ | **required before the first `tester`** |
| MI-6 | N0 code (topology, dead letters, identity on NATS, client options, pool pins) | [mi-05](sprints/sprint-mi-05.md) | ○ dark | ⬜ | rides v1.6.0 |
| MI-7 | NATS auth N1 → N2 → N3 (server first) | [mi-06](sprints/sprint-mi-06.md) (+ ≥ 24 h re-check in [l-01](sprints/sprint-l-01.md)) | ◐ | ⬜ | see [NATS auth](#nats-auth-n0n4) |
| **MI-8** | `host-verify --cluster` extension (read-only checks) | [mi-02](sprints/sprint-mi-02.md) | ○ | ⬜ | the only ops tooling (D34) |
| MI-9 | Evalpack plumbing (machine user, private repo + CI probe, GHCR, pull secret, ImageRepository) | [mi-07](sprints/sprint-mi-07.md) + `ev-machine-user` (ImagePolicy in [m3-07](sprints/sprint-m3-07.md)) | ○ | ⬜ | must land by Fri 2026-10-09 · PAT expiry → [below](#evalpack-pat-expiry-manual-d34) |
| MI-10 | Spike week: P0–P3 + image volume (+ WIF) | [spk-01](sprints/sprint-spk-01.md), [spk-02](sprints/sprint-spk-02.md), [spk-03](sprints/sprint-spk-03.md) · `ev-spike-week` | ○ | ✅ | **Spike P0–P3 GO and image-volume GO** (2026-09-25). Part 1 (spk-01): P0–P2 GO (Q-A GO, Q-B GO; mechanism go-sandbox `forkexec.Runner`, no userns). Part 2 (spk-02 re-run): Q-C GO (TSAN needs no ASLR policy; amd64 allowlists at 0 unexpected SIGSYS). The AppArmor remount finding is closed with `ro`-only scoped rules. Image volume GO with the kubelet defaults (no `AlwaysVerify`). No owner decision is needed · [spike record](#spike-record) |
| MI-11 | October host window: sandbox block, L23 kubelet args, pid limits, k3s/CNPG bumps | [mi-09](sprints/sprint-mi-09.md) (prep) + `ev-host-window` | ◐ | ⬜ | **Sat 2026-10-24** |
| **MI-11a** | Limit hygiene (Flux controllers 512 Mi; Traefik, cert-manager, metrics-server limits; Longhorn budget) | [mi-08](sprints/sprint-mi-08.md) | ◐ | ⬜ | gates MI-12 |
| MI-12 | Runner dark (`runner-v1.0.0`, `Recreate`, off `apps`' wait path, 2nd IUA) | code [m3-03](sprints/sprint-m3-03.md), [m3-04](sprints/sprint-m3-04.md), [m3-15](sprints/sprint-m3-15.md) · deploy [mi-10](sprints/sprint-mi-10.md) | ○ dark | ⬜ | image before policy |
| MI-13 | judge HelmRelease (8087, 4-step, evalpack volume, default-deny egress) | [m3-07](sprints/sprint-m3-07.md) | ○ | ⬜ | tag v1.13.0 first, then the infra PR |
| MI-14 | M4 gates: WIF spike, judge 443 egress, `xlearn-judge-llm` secret, provider runbook ($15 / $12) | [spk-03](sprints/sprint-spk-03.md) (WIF item) + [mi-12](sprints/sprint-mi-12.md) + `ev-provider-runbook` | ○ | ⬜ (WIF item ✅) | **WIF spike: GO, 2026-09-25** ([spk-03](sprints/sprint-spk-03.md), [t5 §15](research/t5-platform-ai.md#15-wif-spike-result-spk-03-2026-09-25)). Settings: `check_jti=false`, rule lifetime 1 h (set explicitly), `expirationSeconds: 3600`, scope `workspace:developer`. The org's monthly spend limit ($5) must be raised before mi-12. MI-14 stays open until mi-12 |
| MI-15 | Track B: PSA labels, SA tokens off · `xlearn` egress, PG `connectionLimit` 20, Renovate, N4 | [mi-08](sprints/sprint-mi-08.md) (PSA, tokens) + [mi-11](sprints/sprint-mi-11.md) (rest) | ◐ | ⬜ | hygiene |
| MI-16 | M6b gates: CSP + Permissions-Policy, camera/mic deny, coach sizing, 20 s SDP route, WSS egress | [mi-13](sprints/sprint-mi-13.md) | ◐ | ⬜ | before M6b (Q1 2027) |

### NATS auth (N0–N4)

[ADR-0035 §2](../adr/0035-v2-operations-nats-auth-limits-capacity.md#2-nats-auth-nkey-users-fine-acls-server-first).
**Current stage: no auth** (pre-N1). The N3 timestamp gates [l-01](sprints/sprint-l-01.md) (≥ 24 h) and M3.

| Step | Change | Sprint | State | Date / timestamp · verify |
|------|--------|--------|-------|---------------------------|
| N0 | `topology.go` + golden/budget/registry tests, dead letters, identity on NATS, nkey client options, pool pins, compose NATS-auth test | [mi-05](sprints/sprint-mi-05.md) → live in v1.6.0 ([m1-02](sprints/sprint-m1-02.md)) | ⬜ | — |
| N1 | nkey users + fine ACLs + `legacy` + `no_auth_user` (the one restart) | [mi-06](sprints/sprint-mi-06.md) | ⬜ | — |
| N2 | seeds: practice, review, assessment, identity | [mi-06](sprints/sprint-mi-06.md) | ⬜ | — |
| N2 | seed: coach (erase consumer) | [l-01](sprints/sprint-l-01.md) | ⬜ | — |
| N2 | seed: judge | [m3-07](sprints/sprint-m3-07.md) | ⬜ | — |
| **N3** | `legacy` → `deny ">"` (reload); no `legacy` connection right after and ≥ 24 h later | [mi-06](sprints/sprint-mi-06.md) (+ re-check in [l-01](sprints/sprint-l-01.md)) | ⬜ | — |
| N4 | remove `legacy` + `no_auth_user` together (reload, or the monthly window) | [mi-11](sprints/sprint-mi-11.md) | ⬜ | — |

### M3 hard entry checklist (live)

Copied verbatim from [rollout §5](rollout-plan.md#5-m3-hard-entry-checklist); ticked here as items land. A red item
blocks the M3 UI sprint ([m3-11](sprints/sprint-m3-11.md)).

- [x] Spike P0–P3 **GO** and the image-volume spike **GO** (MI-10). ✅ 2026-09-25: spk-01 + spk-02 ([t3 §16.4](research/t3-sandbox.md#164-mi-10-verdict-and-proposed-adr-0030-deltas))
- [ ] MI-4, MI-5, **MI-5a**, **MI-7 (N3)**, MI-9, MI-11, **MI-11a**, MI-12 and MI-13 done
- [ ] **MI-8: `host-verify --cluster` (extended) green.** The memory sum, *with the runner's 3 GiB counted*, is ≤ capacity − 0.5 GiB; no OOMKills; PVCs < 60%; NATS `auth_required`; the NetworkPolicies are present
- [ ] T25/T26 tooling (pre-push fingerprint hook, packlint, `contract_hash`)
- [ ] **14 pilot packs stamped** (Go, C++ and Python references; about 28–41 owner hours)
- [ ] `account.role` live (M1a), so the owner and tester cohort gates judge features
- [ ] TR-STEAL not firing (sar p95 read by `host-verify`), or R2 planned
- [ ] AB07–AB12 frozen
- [ ] The last Hostinger weekly image is ≤ 7 days old

*Key:* MI-4 = mi-14 · MI-5/5a = mi-03 · MI-7 (N3) = mi-06 (+ l-01 re-check) · MI-8 = mi-02 · MI-9 = mi-07 (+ m3-07) ·
MI-10 = spk-01 + spk-02 · MI-11 = mi-09 + the host window · MI-11a = mi-08 · MI-12 = mi-10 · MI-13 = m3-07 ·
T25/T26 = m3-01 · 14 packs = `ev-packs-14` · `account.role` = m1-02 · AB07–AB12 = ds-m3-01 + ds-m3-02 merged (the
merge is the freeze, D40).

## Releases

### Milestone → tag → rollback floor → snapshot

Indicative per [ADR-0034 §1.6](../adr/0034-v2-release-labelling-gating-and-rollback.md#16-indicative-tag-timeline):
take the **next free** version at tag time and write the real one here. **Snapshot?** = a manual Hostinger snapshot
right before the tag (contract, erase or GA), after `host-verify --cluster` is green; record its name and time, since
R-d step 2 reads this table.

| Milestone | Tag (indicative) | Sprint | Snapshot? | Gate state after | Rollback floor after | Snapshot taken | Verified live | State |
|-----------|------------------|--------|-----------|------------------|----------------------|----------------|---------------|-------|
| ADR-0033 §2 stopgap | **v1.5.2** | pre-build | no | signup closed | — | — | all seven `v1.5.2` | ✅ 2026-09-24 |
| M1a expand | v1.6.0 | [m1-02](sprints/sprint-m1-02.md) | no | no behaviour change | none (expand only) | n/a | — | ⬜ |
| M1b | v1.7.0 | [m1-07](sprints/sprint-m1-07.md) | no | golden = v1 (D27, D31, 429s aside) | **1.6.0** | n/a | — | ⬜ |
| M1c contract | v1.8.0 | [m1-08](sprints/sprint-m1-08.md) | **yes (contract)** | — | **1.7.0, hard** | — | — | ⬜ |
| NATS N1 → N3 (infra) | — | [mi-06](sprints/sprint-mi-06.md) | weekly-image check (N1 restart) | auth on | client images ≥ the N0 tag | — | — | ⬜ |
| M2 consumers | v1.9.0 | [m2-02](sprints/sprint-m2-02.md) | no | producers off | unchanged | n/a | — | ⬜ |
| M2 producers + D2 | v1.10.0 | [m2-05](sprints/sprint-m2-05.md) | no | D2 on (`REVISION_ENTRY_RULE`) | 1.9.0 | n/a | — | ⬜ |
| L-E consumers | v1.11.0 | [l-01](sprints/sprint-l-01.md) | no | — | unchanged (1.9.0) | n/a | — | ⬜ |
| L-E erase | v1.12.0 | [l-02](sprints/sprint-l-02.md) | **yes (erase)** | web erase: testers only; owner CLI only | 1.11.0 | — | — | ⬜ |
| runner dark (MI-12) | runner-v1.0.0 | [m3-15](sprints/sprint-m3-15.md) → [mi-10](sprints/sprint-mi-10.md) | no | runner reachable only from judge | — | n/a | — | ⬜ |
| M3-1 judge dark | v1.13.0 (+ evalpack v1.0.0) | [m3-07](sprints/sprint-m3-07.md) | no | judge dark, born with its erase consumer | unchanged | n/a | — | ⬜ |
| M3-2 Run/Submit | v1.14.0 | [m3-13](sprints/sprint-m3-13.md) | no | `JUDGE_BASE_URL` set; judge cohort-only | 1.13.0 | n/a | — | ⬜ |
| P pilot | v1.15.0 (+ runner-v1.1.0) | [p-03](sprints/sprint-p-03.md) ([p-01](sprints/sprint-p-01.md)) | no | `preview` = cohort only | unchanged (consumers ≥ 1.7.0) | n/a | — | ⬜ |
| M4 platform AI | v1.16.0 | [m4-07](sprints/sprint-m4-07.md) | no | `LLM_PLATFORM_ENABLED` + cohort | unchanged | n/a | — | ⬜ |
| L-A/L-C front door | v1.17.0 | [l-04](sprints/sprint-l-04.md) | **yes (erase-class)** | `closed` (rehearsal `invite` → `closed`) | unchanged | — | — | ⬜ |
| **GA** | (`v2.0.0-rc.N`) → **v2.0.0** | [ga-01](sprints/sprint-ga-01.md) → [ga-02](sprints/sprint-ga-02.md) | **yes (GA)** | defaults flipped for every account; kill switches stay | no contract; R-b to `<2.0.0` = last 1.x | — | — | ⬜ |
| M6a dark | v2.0.x patch | [m6a-06](sprints/sprint-m6a-06.md) | no | cohort | unchanged | n/a | — | ⬜ |
| MI-16 gateway parts | v2.0.x patch | [mi-13](sprints/sprint-mi-13.md) | no | — | unchanged | n/a | — | ⬜ |
| M6b dark | v2.0.x patch | [m6b-03](sprints/sprint-m6b-03.md) | no | cohort | unchanged | n/a | — | ⬜ |
| **Interviewer GA** | **v2.1.0** | [m6b-04](sprints/sprint-m6b-04.md) | **yes (GA flip)** | interviewer defaults flipped | — | — | — | ⬜ |
| M5 evaluator-only | rides v2.1.0, or next minor ≥ v2.2.0 | [m5-01](sprints/sprint-m5-01.md) | per the tag it rides | env override = kill switch | reversible | — | — | ⬜ |

### Release streams

| Stream | Version | Sprint | Digest | `validated_against` / notes | Deployed | State |
|--------|---------|--------|--------|------------------------------|----------|-------|
| evalpack | `v0.1.0` (probe image; below every range) | [mi-07](sprints/sprint-mi-07.md) | — | CI anonymous-GET probe | never | ⬜ |
| evalpack | `v0.2.0` (optional pipeline proof) | [m3-02](sprints/sprint-m3-02.md) | — | — | never | ⬜ |
| evalpack | **`v1.0.0`** (≥ 1 stamped pack; policy `>=1.0.0 <2.0.0`, image before policy) | [m3-07](sprints/sprint-m3-07.md) | — | — | judge image volume | ⬜ |
| evalpack | `v1.N.0` (go-concurrency packs) | [p-03](sprints/sprint-p-03.md) | — | — | — | ⬜ |
| evalpack | `v1.x` (the 14 pilot packs as they land; D6 waves after GA) | `ev-packs-14`, `ev-d6-waves` | — | TLs provisional until [m3-13](sprints/sprint-m3-13.md) | — | ⬜ |
| runner | `runner-v1.0.0-rc.1` → **`runner-v1.0.0`** (reproducible; `runner-release.yml`) | [m3-15](sprints/sprint-m3-15.md) (deploy: [mi-10](sprints/sprint-mi-10.md)) | — | TL baselines | dark | ⬜ |
| runner | `runner-v1.1.0` (go-race profile) | [p-01](sprints/sprint-p-01.md) | — | go-race baseline | dark | ⬜ |

### Ranges and release line

| Item | Now | Changes | Order |
|------|-----|---------|-------|
| `.release-line` | `1` (xlearn#53) | `2` in the GA PR ([ga-01](sprints/sprint-ga-01.md)), merged on CI green (D40) | the GA PR, self-reviewed like any PR; `main` is then on the 2 line (1.x hotfixes branch from the last 1.x tag) |
| 7 fleet `xlearn-*` ImagePolicies | `>=1.0.0 <2.0.0` (infra#29) | `>=1.0.0 <3.0.0` at GA ([ga-02](sprints/sprint-ga-02.md)) | pre-flip check → **merge first, then tag** |
| `xlearn-judge` policy | — | born `>=1.0.0 <2.0.0` ([m3-07](sprints/sprint-m3-07.md)); widened with the fleet at GA | **tag first, then merge** (image before policy) |
| `xlearn-runner` policy | — | born `>=1.0.0 <2.0.0` ([mi-10](sprints/sprint-mi-10.md)); separate stream, not widened at GA | image before policy |
| `xlearn-evalpack` policy | — | born `>=1.0.0 <2.0.0` ([m3-07](sprints/sprint-m3-07.md)); separate stream, not widened at GA | image before policy |

### Pending contracts

Expand → backfill → contract. A sprint that expands and leaves a contract for later adds a row; the contract sprint closes it.

| Expanded in | What waits | Contract in | State |
|-------------|------------|-------------|-------|
| M1a/M1b (v1.6.0, v1.7.0) | the M1 CHECKs and `total_35` | [m1-08](sprints/sprint-m1-08.md) (v1.8.0, snapshot first) | ⬜ |

### Pending-smoke notes

Checks a merge-only sprint leaves for whichever tag carries it. **Every tag sprint reads this list before tagging**
and runs the notes that apply, then marks them done.

| Added by | For tag | Checks | Who runs | Done |
|----------|---------|--------|----------|------|
| — | — | — | — | — |

## Flag inventory

[ADR-0034 §2](../adr/0034-v2-release-labelling-gating-and-rollback.md#2-feature-gating-three-tiers-no-flag-service):
T-1 build-time · T-2 runtime env (infra PR) · T-3 cohort (`account.role`, never the JWT). **≤ 6 live non-kill
flags**, each with an owning and a removal milestone. Kill switches and operating modes are permanent and listed
separately. Rows marked *planned* are added live by the owning sprint.

### Live non-kill flags — 0 / 6 live

| Flag | Tier | Service(s) | Owning milestone (sprint) | Removal milestone (sprint) | State |
|------|------|------------|---------------------------|----------------------------|-------|
| `touchesEnabled` (touch-start guard) | T-1 build-time | gateway (BFF touch routes) | M2 ([m2-04](sprints/sprint-m2-04.md)) | M2 ([m2-05](sprints/sprint-m2-05.md)) | planned |
| judge cohort gate (`judgeCohortOnly`) | T-3 cohort (code default) | gateway (judge BFF) | M3 ([m3-09](sprints/sprint-m3-09.md)) | GA ([ga-01](sprints/sprint-ga-01.md)) | planned |
| platform-AI cohort gate (`platformAICohortOnly`) | T-3 cohort (code default) | judge | M4 ([m4-02](sprints/sprint-m4-02.md)) | GA ([ga-01](sprints/sprint-ga-01.md)) | planned |
| `COURSE_STATUS_OVERRIDE` | T-2 | every `xlearn-*` | P ([p-02](sprints/sprint-p-02.md)) | GA ([ga-01](sprints/sprint-ga-01.md)): remove, or promote to an operating mode (decided in ga-01's session, D40) | planned |
| interviewer cohort gate (`interviewAudience = "cohort"`) | T-1 code default (T-3 audience) | coach (`/api/interviews/*`) | M6a ([m6a-01](sprints/sprint-m6a-01.md)) | v2.1.0 ([m6b-04](sprints/sprint-m6b-04.md)) | planned |
| M6c extras audience default(s) | T-1 | coach, web | M6c | M6c | outline (v2.2); budget against the ≤ 6 limit |

Peak concurrent non-kill flags on this plan: 3 (before GA). Remove a flag with its code, config parsing, tests and
docs; a flag set in `../infra` is removed by its own infra PR after the tag.

### Permanent kill switches and operating modes

| Switch / mode | Tier | Service(s) | Introduced (sprint) | Current value | Kill path |
|---------------|------|------------|---------------------|---------------|-----------|
| `SIGNUP_MODE` (operating mode: `closed` · `invite` · `open` dev-only with `DEV_AUTH`) | T-2 | identity | live since v1.5.2; `DEV_AUTH` guard [m1-04](sprints/sprint-m1-04.md); `invite` [l-03](sprints/sprint-l-03.md); rehearsal [l-04](sprints/sprint-l-04.md) | **`closed`** (infra#30) | infra PR |
| `REVISION_ENTRY_RULE` | T-2 | review | [m2-05](sprints/sprint-m2-05.md) | not present | infra PR (stops new D2 anchors) |
| `JUDGE_BASE_URL` (unset = off) | T-2 | gateway, practice | set by [m3-13](sprints/sprint-m3-13.md) (after v1.14.0) | not present | unset via infra PR |
| grading override (`GRADING_OVERRIDE`, unset \| `self`) | T-2 | practice | [m3-08](sprints/sprint-m3-08.md) | not present | infra PR |
| `JUDGE_ADMISSION` (L15 admission kill switch) | T-2 | judge | [m3-05](sprints/sprint-m3-05.md) | not present | infra PR on `apps/xlearn-judge.yaml` |
| `LLM_PLATFORM_ENABLED` | T-2 | judge | present `false` via [mi-12](sprints/sprint-mi-12.md); `true` for the cohort via [m4-07](sprints/sprint-m4-07.md) | not present | infra PR |
| `VOICE_COMM_AI` | T-2 | coach | M6c ([m6c-02](sprints/sprint-m6c-02.md), outline) | — | infra PR |

### Config notes (operating parameters, not flags)

| Setting | Where | Value / rule | Set by |
|---------|-------|--------------|--------|
| `SEAT_CAP` | identity | 15; > 40 requires R2 first | [l-03](sprints/sprint-l-03.md) |
| `DEV_AUTH` | identity (compose only) | never set in production; `open` honoured only with it | [m1-04](sprints/sprint-m1-04.md) |
| `PRACTICE_SETTLE_WINDOW` | practice | dev/e2e only (requires `DEV_AUTH`); production never sets it | [m4-04](sprints/sprint-m4-04.md) |
| `COACH_ROLE`, `COACH_INTERVIEW_BASE_URL` | coach, gateway | deploy-shape config (only if S6 M7 failed) | [m6b-02](sprints/sprint-m6b-02.md) |

## Content status

Owner content track (~10 h/week) starts at the item-schema freeze (`ev-schema-freeze`, ≈ 2026-10-05).

| Item | Target | Now | Owner h | Sprint / event | State |
|------|--------|-----|---------|----------------|-------|
| Item schema v1 frozen | frozen ≈ 2026-10-05 | not frozen | — | [m1-01](sprints/sprint-m1-01.md) · `ev-schema-freeze` | ⬜ |
| DSA converted to `curriculum/courses/dsa/` + `ids.lock.json` | 14 items | v1 layout (`curriculum/dsa/`, 14 items) | — | [m1-09](sprints/sprint-m1-09.md) | ⬜ |
| Example-1 statements rewritten | 2 | 0 / 2 | — (agent-drafted; lands as drafted, D40) | [m1-09](sprints/sprint-m1-09.md) · `ev-m1-statements` (automatic) | ⬜ |
| T25/T26 tooling (pre-push fingerprint hook, packlint, `contract_hash`) | live | — | — | [m3-01](sprints/sprint-m3-01.md) | ⬜ |
| Evalpack pipeline (private CI gates, validator, image) | live | — | — | [m3-02](sprints/sprint-m3-02.md) | ⬜ |
| **Pilot packs stamped** (Go, C++, Python references) | 14 | **0 / 14** | 28–41 | `ev-packs-14` | ⬜ |
| Time limits re-gated (no longer provisional) | all packed items | provisional | — | [m3-13](sprints/sprint-m3-13.md) | ⬜ |
| go-concurrency pilot items (`preview`) | ~10 | 0 / ~10 | 10–20 | `ev-pilot-content` · [p-02](sprints/sprint-p-02.md), [p-03](sprints/sprint-p-03.md) | ⬜ |
| Analyzer acceptance set | ≥ 70 labelled (≥ 40 test + 30 dev) | 0 / 70 | 5–10 | `ev-acceptance-set` (template: [mi-12](sprints/sprint-mi-12.md)) | ⬜ |
| D6 wave: weeks 1–4 full packs | ~35 (incl. the 14) | 0 / ~35 | in D6 + D20 | `ev-d6-waves` (after GA) | ⬜ |
| D6 wave: DSA self tier | 151 | 14 live (v1 seed), 0 stamped | in D6 + D20 | `ev-d6-waves` (after GA) | ⬜ |
| M5 entry: every live DSA item packed | 151 | 0 / 151 | +230–340 | `ev-dsa-packs-all` | ⬜ |

**Items stamped per tier per week** (DSA; update as stamps land):

| DSA week | Items live (v1 seed) | Self tier stamped | Full pack stamped |
|----------|----------------------|-------------------|-------------------|
| 1 | 6 | 0 | 0 |
| 2 | 3 | 0 | 0 |
| 3 | 1 | 0 | 0 |
| 4 | 1 | 0 | 0 |
| 5–16 | 3 (weeks 8, 9, 10) | 0 | 0 |
| **Total** | **14 / 151** | **0** | **0** (pilot target 14) |
| go-concurrency (pilot) | 0 / ~10 | — | 0 |

## Artboards

Static HTML under `design-system/screens/v2/` (preview-only). States: ⬜ not drafted → 🔄 drafting (the `ds-*`
session is running) → ✅ **frozen (PR #N, date)** → consumed by `<sprint>`. **The `ds-*` merge is the freeze
(D40):** the design sprint records it here itself, in its own PR or a follow-up docs PR merged the same way, and a
consuming UI sprint only verifies it (repairing a missing row if needed). The owner may review a board after it
merges; a change to a frozen board is a follow-up design PR
([status protocol](sprints/README.md#design-sprints-the-owner-review-gate)).
Register: [build-plan → Artboard register](build-plan.md#artboard-register).

| Board | Drafted in | Freeze (automatic at the merge) | Frozen before | State |
|-------|------------|---------------------------------|---------------|-------|
| AB01 | [ds-m1-01](sprints/sprint-ds-m1-01.md) | `ds-m1-01` merged | M1b | ⬜ not drafted |
| AB02 | [ds-m1-01](sprints/sprint-ds-m1-01.md), [ds-p-01](sprints/sprint-ds-p-01.md) | `ds-m1-01` merged; `ds-p-01` merged (full fidelity) | M1b (ds-p-01 full fidelity: P) | ⬜ not drafted |
| AB03 | [ds-m1-01](sprints/sprint-ds-m1-01.md) | `ds-m1-01` merged | M1b | ⬜ not drafted |
| AB04★ | [ds-m2-01](sprints/sprint-ds-m2-01.md) | `ds-m2-01` merged | M2a | ⬜ not drafted |
| AB05 | [ds-m2-01](sprints/sprint-ds-m2-01.md), [ds-p-01](sprints/sprint-ds-p-01.md) | `ds-m2-01` merged; `ds-p-01` merged (full fidelity) | M2a (ds-p-01 full fidelity: P) | ⬜ not drafted |
| AB06 | [ds-m2-01](sprints/sprint-ds-m2-01.md) | `ds-m2-01` merged | M2a | ⬜ not drafted |
| AB07★ | [ds-m3-01](sprints/sprint-ds-m3-01.md) | `ds-m3-01` merged | the M3 UI sprint | ⬜ not drafted |
| AB08 | [ds-m3-01](sprints/sprint-ds-m3-01.md) | `ds-m3-01` merged | the M3 UI sprint | ⬜ not drafted |
| AB09 | [ds-m3-02](sprints/sprint-ds-m3-02.md) | `ds-m3-02` merged | the M3 UI sprint | ⬜ not drafted |
| AB10 | [ds-m3-02](sprints/sprint-ds-m3-02.md) | `ds-m3-02` merged | the M3 UI sprint | ⬜ not drafted |
| AB11 | [ds-m3-01](sprints/sprint-ds-m3-01.md) | `ds-m3-01` merged | the M3 UI sprint | ⬜ not drafted |
| AB12 | [ds-m3-02](sprints/sprint-ds-m3-02.md) | `ds-m3-02` merged | the M3 UI sprint | ⬜ not drafted |
| AB13 | [ds-m6a-01](sprints/sprint-ds-m6a-01.md) | `ds-m6a-01` merged | M6a | ⬜ not drafted |
| AB14 | [ds-p-01](sprints/sprint-ds-p-01.md) | `ds-p-01` merged | P | ⬜ not drafted |
| AB15 | [ds-p-01](sprints/sprint-ds-p-01.md) | `ds-p-01` merged | P | ⬜ not drafted |
| AB16★ | [ds-m4-01](sprints/sprint-ds-m4-01.md) | `ds-m4-01` merged | the M4 UI sprint | ⬜ not drafted |
| AB17 | [ds-m4-01](sprints/sprint-ds-m4-01.md) | `ds-m4-01` merged | the M4 UI sprint | ⬜ not drafted |
| AB18 | [ds-m4-01](sprints/sprint-ds-m4-01.md) | `ds-m4-01` merged | the M4 UI sprint | ⬜ not drafted |
| AB19★ | [ds-l-01](sprints/sprint-ds-l-01.md) | `ds-l-01` merged | L-A | ⬜ not drafted |
| AB20 | [ds-l-01](sprints/sprint-ds-l-01.md) | `ds-l-01` merged | L-A | ⬜ not drafted |
| AB21 | [ds-l-01](sprints/sprint-ds-l-01.md) | `ds-l-01` merged | **L-E** | ⬜ not drafted |
| AB22 | [ds-m2-01](sprints/sprint-ds-m2-01.md) | `ds-m2-01` merged | M2a | ⬜ not drafted |
| ~~AB23~~ | — | — | — | dropped (D33) |
| AB24 | [ds-m6a-01](sprints/sprint-ds-m6a-01.md) | `ds-m6a-01` merged | M6a | ⬜ not drafted |
| AB25 | [ds-m6a-01](sprints/sprint-ds-m6a-01.md) | `ds-m6a-01` merged | M6a | ⬜ not drafted |
| AB26 | [ds-m6a-02](sprints/sprint-ds-m6a-02.md) | `ds-m6a-02` merged | M6a | ⬜ not drafted |
| AB27 | [ds-m6a-02](sprints/sprint-ds-m6a-02.md) | `ds-m6a-02` merged | M6a | ⬜ not drafted |
| AB28 | [ds-m6a-02](sprints/sprint-ds-m6a-02.md) | `ds-m6a-02` merged | M6a | ⬜ not drafted |
| AB29 | [ds-m6b-01](sprints/sprint-ds-m6b-01.md) | `ds-m6b-01` merged | M6b | ⬜ not drafted |
| AB30★ | [ds-m6b-01](sprints/sprint-ds-m6b-01.md) | `ds-m6b-01` merged | M6b | ⬜ not drafted |
| AB31 | [m6c-03](sprints/sprint-m6c-03.md) (outline) | — | v2.2 | ⬜ outline only |

## ADRs

| ADR | Topic | Status | Acceptance |
|-----|-------|--------|------------|
| [0026](../adr/0026-per-course-extensibility-model.md) | per-course extensibility | ✅ Accepted | 2026-09-24 (BP2) |
| [0027](../adr/0027-content-evalpack-and-user-data-model.md) | content, eval pack, user data model | ✅ Accepted | 2026-09-24 (BP2) |
| [0028](../adr/0028-object-storage-and-backups.md) | object storage and backups | ✅ Accepted | 2026-09-24 (BP2) |
| [0029](../adr/0029-judge-contract-and-learning-signal.md) | judge contract and learning signal | ✅ Accepted | 2026-09-24 (BP2) |
| [0030](../adr/0030-runner-technology-and-host-hardening.md) | runner technology and host hardening | Proposed | at the sandbox spike GO, before M3: [m3-03](sprints/sprint-m3-03.md) task 1 (consumes [spk-01](sprints/sprint-spk-01.md) + [spk-02](sprints/sprint-spk-02.md); folds ADR-0035 §6). The 10-24 host window runs on the spike GO before this, by design ([decisions log](#decisions-log)) |
| [0031](../adr/0031-platform-ai-and-two-tier-keys.md) | platform AI and two-tier keys | Proposed | at the WIF spike, before M4: [mi-12](sprints/sprint-mi-12.md) (consumes [spk-03](sprints/sprint-spk-03.md): **WIF GO** 2026-09-25, `check_jti=false`; the §2 scope and other amendments are proposed in [t5 §15](research/t5-platform-ai.md#15-wif-spike-result-spk-03-2026-09-25)) |
| [0032](../adr/0032-realtime-ai-mock-interviewer.md) | realtime AI mock interviewer | Proposed | at S6, before the M6a design freeze: [ds-m6a-01](sprints/sprint-ds-m6a-01.md) task 1 (consumes [spk-04](sprints/sprint-spk-04.md)) |
| [0033](../adr/0033-invite-only-admission-and-owner-admin.md) | invite-only admission and owner admin | ✅ Accepted | 2026-09-24 (BP2) |
| [0034](../adr/0034-v2-release-labelling-gating-and-rollback.md) | v2 release labelling, gating and rollback | ✅ Accepted | 2026-09-24 (BP2) |
| [0035](../adr/0035-v2-operations-nats-auth-limits-capacity.md) | v2 operations: NATS auth, limits, capacity | ✅ Accepted | 2026-09-24 (BP2) |

New ADRs written during the build take the next free number after checking peers (0036 onward).

## Owner calendar events

Owner-only actions are **calendar events, not sprints** (BP4, tentative and owner-booked). **Since D40 nothing
waits on the owner mid-session.** An owner-only action is done **before launch** of the sprint that needs it: that
prompt's `## Before you launch (owner)` block lists it, and launching attests it's done. A sprint may prepare one
(runbook, scripts). If a prerequisite turns out missing, the session lands what doesn't depend on it and marks the
gap ⛔ here. Events that only froze boards or awaited a sign-off are **automatic** (second table). Tick the state here
(✅ + date) when it happens.

| Event | When | What | Prepared by | State |
|-------|------|------|-------------|-------|
| `ev-mi0` | Fri 2026-09-25 · before launch of spk-01 | MI-0 H0 reboot into kernel 6.8.0-142 | none (prepared by infra#28, done 2026-09-24; recorded in [mi-02](sprints/sprint-mi-02.md)) | ⬜ |
| `ev-mi1` | week 1 (by 2026-10-02) · at the latest before launch of mi-12 (provider accounts) | MI-1 owner hygiene | none | ⬜ |
| `ev-mi5b-dns` | weeks 2–4 · before launch of mi-04 (so before the first tester) | DNS record for ops.sujaykumar.dev at the registrar (MI-5b); console bookmarks once mi-04 lands | [mi-04](sprints/sprint-mi-04.md) | ⬜ |
| `ev-owner-role` | right after the v1.7.0 tag (an `identity admin` step m1-07's prompt can run, D40); at the latest before launch of l-02 | Owner role set once | [m1-04](sprints/sprint-m1-04.md) (runbook) | ⬜ |
| `ev-snap-v1.8.0` | before launch of m1-08 | Manual Hostinger snapshot in hPanel, 1-day retention (contract tag) | [m1-08](sprints/sprint-m1-08.md) | ⬜ |
| `ev-machine-user` | week 1–2 (by Fri 2026-10-09) · before launch of mi-07 | Evalpack machine user + PAT (MI-9) | [mi-07](sprints/sprint-mi-07.md) | ⬜ |
| `ev-spike-goahead` | by Fri 2026-10-09 (booked together with ev-spike-week and ev-host-window) · before launch of spk-02 | Spike go-ahead (D23): **automatic since D40**, since launching [spk-01](sprints/sprint-spk-01.md) is the go-ahead for P0–P3 + image volume and launching [spk-03](sprints/sprint-spk-03.md) is the WIF yes. Owner-only part: the **second-VPS answer** for [spk-02](sprints/sprint-spk-02.md) (A: an empty amd64 VPS with SSH access and reimage approval; else B: the private GitHub Actions scratch repo) | none (owner) | ✅ spk-01 go-ahead exercised 2026-09-25 (launch = D23 go-ahead, D40); D41 says spk-02 runs on `skriptvalley-vps` (private registry, MI-9/chart-0.3.0 gates dropped). **WIF: exercised 2026-09-25** (launching [spk-03](sprints/sprint-spk-03.md) was the WIF yes, D40) |
| `ev-spike-week` | **Fri 2026-09-25 → Sat 2026-09-26 (D41, spikes first)**; was Mon 2026-10-12 → Fri 2026-10-16 | Spike week: P0–P3 + image volume (+ WIF if it fits) | [spk-01](sprints/sprint-spk-01.md) | 🔄 in progress: spk-01 (P0–P2) done 2026-09-25; spk-02 done 2026-09-25 (re-run; P0–P3 GO, image volume GO → MI-10 ✅); spk-03 done 2026-09-25, a day early (WIF GO, `check_jti=false`); spk-04 on Sat 2026-09-26 |
| `ev-host-window` | Sat 2026-10-24 · before launch of mi-10 | October host window (MI-11, batched with MI-11a): **a session runs [mi-09](sprints/sprint-mi-09.md)'s runbook as its prompt** (D40). Before launch: the owner's hPanel snapshot, hPanel/VNC reachable, and the weekly-image date | [mi-09](sprints/sprint-mi-09.md) | ⬜ |
| `ev-first-tester` | in l-02's session, after MI-5b is live (≈ L-E) | First `tester` account: **minted by l-02's session** with `identity admin account create --role tester` (pre-approved, D40), after MI-5b is live; if a human sign-in turns out to be needed, the session records ⛔ | [l-02](sprints/sprint-l-02.md) | ⬜ |
| `ev-snap-v1.12.0` | before launch of l-02 | Manual Hostinger snapshot in hPanel, 1-day retention (erase tag) | [l-02](sprints/sprint-l-02.md) | ⬜ |
| `ev-packs-14` | ≈ 2026-10-05 → mid-November · ≥ 1 before launch of m3-07, all 14 before launch of m3-11 | Author and stamp the 14 pilot packs (Go, C++, Python refs; 28–41 owner h) | [m3-02](sprints/sprint-m3-02.md) | ⬜ |
| `ev-q5` | November · before launch of ds-p-01 (so before AB15 is drafted) | PRD Q5 confirmed (go-concurrency; SQL fallback) | [ds-p-01](sprints/sprint-ds-p-01.md) | ⬜ |
| `ev-pilot-content` | December · before launch of p-03 | Pilot course content (~10 items, 10–20 owner h) | [p-02](sprints/sprint-p-02.md) | ⬜ |
| `ev-provider-runbook` | December · before launch of mi-12 (workspace and limits); the WIF registration from mi-12's runbook before launch of m4-01 | Anthropic Console setup (MI-14): **first raise the org's monthly spend limit (now $5; spk-03)**; then the workspace, $15 limit, alerts and auto-reload off; then the service account, the issuer (**`check_jti` off**) and the rule (lifetime **1 h**, set explicitly) ([t5 §15](research/t5-platform-ai.md#15-wif-spike-result-spk-03-2026-09-25)) | [mi-12](sprints/sprint-mi-12.md) | ⬜ |
| `ev-acceptance-set` | December · before launch of m4-01 (rollout §3 M4 entry) | Analyzer acceptance set (≥ 70 labelled: ≥ 40 test + 30 dev; 5–10 owner h) | [mi-12](sprints/sprint-mi-12.md) | ⬜ |
| `ev-snap-v1.17.0` | before launch of l-04 | Manual Hostinger snapshot in hPanel, 1-day retention (erase-class tag: web erase for every non-owner) | [l-04](sprints/sprint-l-04.md) | ⬜ |
| `ev-l-rehearsal` | December (L exit), after the v1.17.0 tag | Tester invite round-trip on production (`invite` → `closed`): **run by l-04's session** (D40). Before launch: a second GitHub account for the GitHub-path redeem; ⛔ if a step needs a human sign-in | [l-04](sprints/sprint-l-04.md) | ⬜ |
| `ev-strangers` | before the GA PR, in ga-01's session | No active `learner` account | [ga-01](sprints/sprint-ga-01.md) | ⬜ |
| `ev-ga` | GA day (≈ Dec 2026 – Jan 2027) · before launch of ga-02 | GA snapshot in hPanel; ga-02 then merges the range PR and tags v2.0.0 | [ga-02](sprints/sprint-ga-02.md) | ⬜ |
| `ev-s6` | **Sat 2026-09-26, owner present (D41)** | S6 voice-shell bake-off (≤ 1 day, $10 hard limit): the owner creates spk-04's two throwaway OpenAI projects and keys before launching it, and stays present for the day | [spk-04](sprints/sprint-spk-04.md) | ⬜ |
| `ev-s6-recheck` | before launch of ds-m6a-01 (≈ Q1 2027) | ≤ 1 h S6 recheck: same shells, hard gates + M14 only; note if the winner or its price changed (D41, non-blocking) | [spk-04](sprints/sprint-spk-04.md) | ⬜ |
| `ev-snap-v2.1.0` | before launch of m6b-04 (≈ Q1 2027, owner present) | Manual Hostinger snapshot in hPanel, 1-day retention (interviewer GA tag) | [m6b-04](sprints/sprint-m6b-04.md) | ⬜ |
| `ev-pat-expiry` | recurring (date in status.md) | Evalpack PAT expiry manual check (D34); a rotation needs a new PAT (owner) | [mi-07](sprints/sprint-mi-07.md) | ⬜ |
| `ev-monthly-window` | monthly (D22) | Monthly reboot window | [mi-11](sprints/sprint-mi-11.md) | ⬜ |
| `ev-d6-waves` | after GA (v2.0.x) | D6 content waves | [m3-02](sprints/sprint-m3-02.md) | ⬜ |
| `ev-dsa-packs-all` | ≈ H2 2027 · before launch of m5-01 | Every live DSA item packed (M5 entry; +230–340 h) | [m3-02](sprints/sprint-m3-02.md) | ⬜ |
| `ev-v3-opening` | v3 (not in v2) | The opening ([rollout §11](rollout-plan.md#11-opening-gates-v3) gates, including the owner's privacy-notice review) | none (v3 planning) | ⬜ |

**Post-ship owner follow-ups (non-blocking, D40).** Things only the owner can do after a sprint ships. None of them gates a tag, a milestone or another sprint; findings become follow-up PRs.

| Event | When | What | Recorded by | State |
|-------|------|------|-------------|-------|
| `ev-nats-ops-first-use` | after mi-06 (N3) | First use of the offline ops seed (break-glass tunnel), logged in the NATS break-glass log | [mi-06](sprints/sprint-mi-06.md) | ⬜ |
| `ev-hook-install` | after m3-01 merges | Install the pre-push fingerprint hook in each authoring clone (`make install-hooks`) | [m3-01](sprints/sprint-m3-01.md) | ⬜ |
| `ev-m3-dogfood` | after `v1.14.0` | Dogfood the judge on real DSA items | [m3-13](sprints/sprint-m3-13.md) | ⬜ |
| `ev-gc-key-reconfirm` | after p-03 | Re-confirm the pilot's key stamps; a correction is a follow-up pack PR | [p-03](sprints/sprint-p-03.md) | ⬜ |
| `ev-pilot-run` | after `v1.15.0` | Run one graded pilot item end to end | [p-03](sprints/sprint-p-03.md) | ⬜ |
| `ev-m4-followup` | after `v1.16.0` | Tick the AI consents; read the Console rate limits and the ±5% ledger checks; optional dogfood and exhaustion drill | [m4-07](sprints/sprint-m4-07.md) | ⬜ |
| `ev-m6a-dogfood` | after the M6a `v2.0.x` patch | Run a full text mock interview | [m6a-06](sprints/sprint-m6a-06.md) | ⬜ |

**Automatic events: no owner action (D40).** They are kept so that references still resolve. Per-board freezes are
recorded in [Artboards](#artboards).

| Event | When | What | Recorded by | State |
|-------|------|------|-------------|-------|
| `ev-schema-freeze` | ≈ 2026-10-05, at m1-01's merge | Item schema frozen → content authoring unblocked (`ev-packs-14` can start) | [m1-01](sprints/sprint-m1-01.md) | ⬜ |
| `ev-m1-statements` | week 2, at m1-09's merge | The 2 copied Example-1 statements, rewritten by the agent, land as drafted and no longer gate m1-02's `v1.6.0` tag (D40). The owner may revise them later with a content PR | [m1-09](sprints/sprint-m1-09.md) | ⬜ |
| `ev-notice-text` | no v2 event: moved to the [v3 opening gates](rollout-plan.md#11-opening-gates-v3) (D40) | The privacy-notice text lands with l-05 as drafted; the owner reviews it before the first real invite, and a revision is a content PR | [l-05](sprints/sprint-l-05.md) | moved to v3 |
| `ev-m4-day1` | the day `LLM_PLATFORM_ENABLED=true` merges | M4 data window starts (≥ 2 weeks → `SEAT_CAP` re-size; an opening gate, not GA) | [m4-07](sprints/sprint-m4-07.md) | ⬜ |
| `ev-freeze-ds-m1-01` · `ev-freeze-ds-m2-01` · `ev-freeze-ds-l-01` · `ev-freeze-ds-m3-01` · `ev-freeze-ds-m3-02` · `ev-freeze-ds-p-01` · `ev-freeze-ds-m4-01` · `ev-freeze-ds-m6a-01` · `ev-freeze-ds-m6a-02` · `ev-freeze-ds-m6b-01` | at each `ds-*` merge | Design freeze: the board PR merging on CI green **is** the freeze (D40). The owner may review afterwards, and a change is a follow-up design PR | each `ds-*` sprint | automatic (per board in [Artboards](#artboards)) |

## Evalpack PAT expiry (manual, D34)

No alert exists for this. Check the date here during any evalpack or release sprint, and rotate before it expires
(infra PR updates the pull secret in `xlearn` and `flux-system`).

| Credential | Scope | Created | Expires | Rotate by | Last checked | State |
|------------|-------|---------|---------|-----------|--------------|-------|
| evalpack machine-user PAT (classic, `read:packages`) | GHCR pull of `xlearn-evalpack` | — | — | — | — | ⬜ not created ([mi-07](sprints/sprint-mi-07.md), `ev-machine-user`) |

## Manual-access logs

Every sanctioned manual path is logged (rollout §2.2). IDs only; no emails, no learner data.

### CLI-use log

Admin CLIs via `kubectl exec` (D33): `identity admin …`, `judge admin …`, `review admin …`.

| Date | Who | Command (verb) | Target (account id / object) | Why | Sprint / event |
|------|-----|----------------|------------------------------|-----|----------------|
| — | — | — | — | — | — |

### NATS break-glass log

`ssh sujaykumar-vps` port-forward to the `nats` Service + the `nats` CLI with the offline ops seed ([ADR-0035 §2](../adr/0035-v2-operations-nats-auth-limits-capacity.md#2-nats-auth-nkey-users-fine-acls-server-first)).

| Date | Who | Why | Commands | Outcome |
|------|-----|-----|----------|---------|
| — | — | — | — | — |

## L rehearsal record

The L exit ([rollout §4](rollout-plan.md#l-learner-gate--34-sprints-in-parallel-with-m3m4)): `SIGNUP_MODE=invite`
→ one invite per create path redeemed by a tester → accept, solve, erase on the web → seats back to baseline →
`SIGNUP_MODE=closed`. Prepared by [l-04](sprints/sprint-l-04.md); performed at `ev-l-rehearsal`. Counts only.

| Date | Tag | PR A (`invite`) | PR B (`closed`) | Email path | GitHub path | Seats (baseline → +2 → baseline) | Web erase | Back at `closed` | Notes |
|------|-----|-----------------|-----------------|------------|-------------|-----------------------------------|-----------|------------------|-------|
| — (planned: December) | v1.17.0 | — | — | ⬜ | ⬜ | — | ⬜ | ⬜ | — |

## Spike record

| Spike | Sprint | When | Verdict | Results | Teardown | Consumed by |
|-------|--------|------|---------|---------|----------|-------------|
| S0 read-only prod facts | (planning, D23) | 2026-09-24 | done | [t3](research/t3-sandbox.md); the 72 h vmstat sampler writes `/tmp/xlearn-s0-vmstat.log` on the node (copy it off before MI-0; `/tmp` empties at boot) | log copied: ⬜ | ADR-0035 §5 |
| Sandbox P0–P2 (multipass arm64) | [spk-01](sprints/sprint-spk-01.md) | Fri 2026-09-25 (D41) | **Q-A GO, Q-B GO** | [t3 §16.1](research/t3-sandbox.md#161-p0p2-spk-01-arm64-multipass) | VM `xl-spike` **purged 2026-09-25** by spk-02; `~/xl-spike/` kept for reference | [m3-03](sprints/sprint-m3-03.md) (ADR-0030), [mi-09](sprints/sprint-mi-09.md) |
| P3 amd64 replay + image volume (**MI-10 verdict**) | [spk-02](sprints/sprint-spk-02.md) | Fri 2026-09-25 (D41), env A `skriptvalley-vps` (first session interrupted; re-run the same day in three blocks) | **P0–P3 GO; image volume GO.** Q-C GO: TSAN needs no ASLR policy, and the amd64 allowlists (`go` 27, `cpp` 19, `python` 39, `go-race` 40) pass KILL with 0 unexpected SIGSYS. The x86_64-only pod profile is proposed. The AppArmor `remount,` finding is closed with `ro`-only scoped rules. Image volume GO with the kubelet defaults; the GOCACHE seed is measured | [t3 §16.2–16.4](research/t3-sandbox.md#162-p3-amd64-replay-spk-02) | ✅ 2026-09-25, both sessions: env A restored to its baseline. That means `k3s-uninstall.sh`; the registry, images, buildx refs, host files, sysctls and `/root/xl-spike` removed; the packages the spike added purged (the package list equals the pre-spike baseline); and apport and the kernel-meta holds as found. The sysctl, iptables and AppArmor-profile counts match the baseline, and the landing container is still serving (443 → 200). The kernel stays upgraded to 6.8.0-142 (approved). `xl-spike` purged. `~/xl-spike/` (throwaway harness, references, logs; no secrets) is kept on the Mac for the orchestrator; delete it with `rm -rf ~/xl-spike` when no longer needed | [m3-03](sprints/sprint-m3-03.md), [mi-09](sprints/sprint-mi-09.md), [m3-04](sprints/sprint-m3-04.md), [p-01](sprints/sprint-p-01.md), [m3-07](sprints/sprint-m3-07.md) |
| WIF (≤ ½ day) | [spk-03](sprints/sprint-spk-03.md) | Fri 2026-09-25 (D41; a day ahead of its Sat slot). VM `xlearn-wif` (arm64 multipass, k3s `v1.36.4+k3s1`); org Sujay's Individual Org | **WIF GO, with `check_jti=false`** on the one-rule issuer. `jti` is present. Rotation: 3600 s at 80.0 % of TTL (2881 s); 600 s at 81–91 %. An in-place restart re-presents the used `jti`, which gets an opaque 401 until the next rotation. Exchange p50 0.332 / max 0.473 s; first call p50 1.613 / max 1.824 s; `anthropic-workspace-id` 5/5. The scope is `workspace:developer` (Files and Batches return 200) | [t5 §15](research/t5-platform-ai.md#15-wif-spike-result-spk-03-2026-09-25) | ✅ 2026-09-25: namespace `wif-spike` deleted; VM `xlearn-wif` purged (`multipass list`: no instances; `xl-spike` was already gone). The Console objects are an owner follow-up ([Open owner items](#open-owner-items)) | [mi-12](sprints/sprint-mi-12.md) (ADR-0031), [m4-01](sprints/sprint-m4-01.md) |
| S6 voice-shell bake-off (≤ 1 day, $10 limit) | [spk-04](sprints/sprint-spk-04.md) | owner present, before the M6a design freeze | ⬜ | — | ⬜ | [ds-m6a-01](sprints/sprint-ds-m6a-01.md) (ADR-0032) |

## Capacity reads (TR-*)

Measured on demand; nothing alerts (D34). Read with `host-verify --cluster` (MI-8 extension, [mi-02](sprints/sprint-mi-02.md))
after host changes and before contract, erase or GA tags. Triggers and responses R0 → R3:
[ADR-0035 §5](../adr/0035-v2-operations-nats-auth-limits-capacity.md#5-capacity-the-memory-sum-rule-triggers-and-ordered-responses).

| Date | Memory sum vs capacity − 0.5 GiB | Steal (sar p95) | CPU busy | Disk / PVCs | OOMKills | Source | Notes |
|------|----------------------------------|-----------------|----------|-------------|----------|--------|-------|
| 2026-09-24 | v2 projection ≈ 0 margin before MI-11a; ≈ 0.9 GiB inside the rule after it | 9-day p95 5.2%; 7.1% on 9/24 (elevated, noisy) | p95 9.2%, max 33% | node 19 / 193 GB; PG ≈ 10.5 MB / 10 Gi; NATS 8.4 KB / 5 Gi | — | ADR-0035 §5 (S0) | limits 9,354 Mi; working set 4.38 GiB; re-read after MI-0 and at the 2026-09-27 collection point |

## Hand-offs

Cross-sprint hand-offs (values, env, file paths one sprint leaves for another). Also stated in the PR description.

| From | To | What | State |
|------|----|------|-------|
| [spk-01](sprints/sprint-spk-01.md) | [spk-02](sprints/sprint-spk-02.md) | VM `xl-spike` (stopped, k3s + guards + host files + positive pod) and `~/xl-spike/` (throwaway supervisor, corpus, manifests, logs, go-sandbox/containerd source) on the owner's Mac. Reuse for the image-volume replay; **spk-02 deletes both** with `multipass delete --purge` at week's end. Note: the D41 note says spk-02 runs on `skriptvalley-vps`; the arm64 `xl-spike` harness is still the reference for the P2 corpus. | ✅ consumed. spk-02 purged the VM on 2026-09-25; `~/xl-spike/` was kept (with spk-02's `p3/` and `refs/` added) |
| [spk-01](sprints/sprint-spk-01.md) | [mi-09](sprints/sprint-mi-09.md) | Host files ([t3 §16.1](research/t3-sandbox.md#161-p0p2-spk-01-arm64-multipass)): drop-in verbatim; **seccomp must add `pivot_root`**; AppArmor ro-bind-remount rule to finalise on amd64; subuid `kubelet:1073741824:7208960`; `getsubids`←`uidmap`. The shipped **pod seccomp comes from spk-02's amd64 replay**, not this arm64 one. | ✅ recorded |
| [spk-01](sprints/sprint-spk-01.md) | [mi-14](sprints/sprint-mi-14.md) | VAP diff (t3 §16.1): API pre-empts some single-violation B-shapes (B1/B2/B9 need companion fields); rule order (privileged/host-ns before hostUsers/APE); **add E1** (`kubectl debug` ephemeral) to the corpus; PSA warns `"baseline:latest"`. mi-14 has not run, so this is an input to its first PR (not yet an infra issue). | ✅ recorded — fold into mi-14 |
| [spk-01](sprints/sprint-spk-01.md) | [m3-03](sprints/sprint-m3-03.md) | ADR-0030 acceptance inputs: pin **go-sandbox v0.13.7** (commit `6a60e40…`); mechanism = `forkexec.Runner` + `CLONE_INTO_CGROUP`; **SETPCAP unused** by go-sandbox (bounding set not dropped) — decide whether to keep it. | ✅ recorded |
| [spk-01](sprints/sprint-spk-01.md) | [p-01](sprints/sprint-p-01.md) | P1b: `go test -race` + postgres (unix socket, uid 999) both run in the jail. overlayfs (GOCACHE seed) EPERMs under `hostUsers:false` — the pilot GOCACHE mechanism needs another delivery. | ✅ recorded |
| [spk-02](sprints/sprint-spk-02.md) | [mi-09](sprints/sprint-mi-09.md) | ([t3 §16.2](research/t3-sandbox.md#162-p3-amd64-replay-spk-02)) **The broad AppArmor `remount,` rule must not ship.** Under it, read-write remounts of `/` and `/sys` succeeded from the supervisor context. **Done in re-run block 3:** the profile is recorded verbatim in t3 §16.2 (sha256 `1d70ccd0…5775`). `remount,` is replaced by `remount options=(ro, nosuid, noatime, bind) /,` and `remount options=(ro, nosuid, nodev, rbind) /jail/**,`. The read-write remounts of `/`, `/sys` and `/jail/**` are now EACCES; they join the must-deny set together with the 17 probes and `move_mount`/`mount_setattr`, all still denied. Jail setup, references, go-race, GOCACHE and PG pass with every jail bind read-only. Supervisor rule for m3-04: read-only binds are `MS_BIND|MS_REC|MS_NOSUID|MS_NODEV|MS_RDONLY` without `MS_PRIVATE`. **The amd64 pod seccomp file is final** (re-run block 2): §16.1's recipe on amd64 with `architectures: [SCMP_ARCH_X86_64]` (x86_64-only, proposed to m3-03). It has 381 names and `pivot_root`, and is recorded verbatim in t3 §16.2 (sha256 `730a7a55…418d`). If m3-03 keeps the three-arch baseline, only that line changes. | ✅ recorded (re-run blocks 2–3) |
| [spk-02](sprints/sprint-spk-02.md) | [m3-03](sprints/sprint-m3-03.md) | ([t3 §16.2](research/t3-sandbox.md#162-p3-amd64-replay-spk-02), §16.4) ADR-0030 deltas from block 2: the mechanism reproduces on amd64 6.8.0-142. **Pod profile: x86_64-only proposed.** Nothing legitimate broke, and the ia32 `int $0x80` entry point (open under three-arch) is closed. Caveat: runc's bad-arch action is `KILL_THREAD`. **ASLR policy: none** (TSAN never calls `personality`; `ADDR_NO_RANDOMIZE` stays denied). The allowlists' home is m3-04, with fixed rules: non-x86_64/x32 → KILL, `clone3` → ENOSYS, `clone` with `CLONE_THREAD` only, `prctl` with `PR_SET_VMA` only. The AppArmor rules are final (block 3). | ✅ recorded (re-run blocks 2–3) |
| [spk-02](sprints/sprint-spk-02.md) | [p-01](sprints/sprint-p-01.md), [m3-04](sprints/sprint-m3-04.md) | ([t3 §16.2](research/t3-sandbox.md#162-p3-amd64-replay-spk-02)) **go-race is in** as far as the sandbox goes: TSAN works in the jail under `mmap_rnd_bits=32` with no ASLR policy (125/125 + 100/100). **amd64 exec allowlists** (m3-04): `go` 27 (with the netpoller's 4 calls for timers, and `prctl(PR_SET_VMA)` for Go ≥ 1.25), `cpp` 19, `python` 39, `go-race` 40. Under KILL: 66/66 per language, 0 unexpected SIGSYS. The compile-jail sets are recorded; compile filters must allow process spawning, and `go build` needs `GOROOT` set without `/proc`. Postgres + SQL balloon → MLE in the case cgroup. | ✅ recorded (re-run block 2) |
| [spk-02](sprints/sprint-spk-02.md) | [m3-07](sprints/sprint-m3-07.md), [m3-02](sprints/sprint-m3-02.md) | ([t3 §16.3](research/t3-sandbox.md#163-image-volume-spk-02)) **Image volume GO** with the kubelet defaults (`NeverVerifyPreloadedImages`; `KubeletEnsureSecretPulledImages` beta and on). m3-07 mounts the pack as a read-only image volume, as ADR-0027 says, and no fallback is needed. Variant A (the pack layers plus a static copier init image) is tested and stays the recommended fallback. m3-02: the pack can stay `linux/amd64`; multi-arch matters only for local arm64 dev (optional). | ✅ recorded (re-run block 1) |
| [spk-02](sprints/sprint-spk-02.md) | [mi-09](sprints/sprint-mi-09.md) | ([t3 §16.3](research/t3-sandbox.md#163-image-volume-spk-02)) **Node-level registry credentials defeat the pack's credential check.** The GHCR pack credential exists only as the `xlearn-evalpack-pull` imagePullSecret: never k3s `registries.yaml` `auth:`, `/var/lib/kubelet/config.json` or a root `~/.docker/config.json` on the node. With either of the last two, the kubelet marks the cached pack `nodePodsAccessible` for every pod, and that record is sticky. Add a host check. `AlwaysVerify` is optional hardening, not needed for GO. | ✅ recorded (re-run block 1) |
| [spk-02](sprints/sprint-spk-02.md) | [p-01](sprints/sprint-p-01.md), [m3-04](sprints/sprint-m3-04.md) | ([t3 §16.2](research/t3-sandbox.md#162-p3-amd64-replay-spk-02)) **GOCACHE seed:** `GOCACHE` = a read-only seed **in place** (baked into the runner image or an image volume; both work under `hostUsers:false`), with `TMPDIR` on the case tmpfs. It is built with the exec profile's exact toolchain and flags (34 MiB for the references). A compile drops from ~12 s to ~0.35 s with no per-case copy (env A, relative only). | ✅ recorded (re-run block 1) |
| [spk-03](sprints/sprint-spk-03.md) | [mi-12](sprints/sprint-mi-12.md) | ([t5 §15](research/t5-platform-ai.md#15-wif-spike-result-spk-03-2026-09-25)) **Create the production issuer with `check_jti` off** ("Enforce single-use tokens (JTI replay protection)" unticked; it defaults on), then run the **confirming re-run of Q-W3 (a) and (b) under `check_jti=false`**: the same file twice, and an in-place restart; expect 200 and 200. Other settings: the rule's token lifetime **1 h, set explicitly** (the Console defaults to 10 min); `expirationSeconds: 3600`; the inline JWKS is the **`keys` array**. Scope: `workspace:developer`, unless an `org:admin` Admin API call can create the rule with `workspace:inference` and a Files call then returns 403. Raise the org's monthly spend limit ($5 now) first. Runbook line: the JWKS survives k3s restarts and `k3s certificate rotate`. Re-paste it only after `rotate-ca` with a new `service.key`, or after a node rebuilt without the old one. Fold t5 §15's six ADR-0031 findings in at acceptance. | ⬜ open (mi-12) |
| [spk-03](sprints/sprint-spk-03.md) | [m4-01](sprints/sprint-m4-01.md) | ([t5 §15](research/t5-platform-ai.md#15-wif-spike-result-spk-03-2026-09-25)) **Exchange shape:** `POST /v1/oauth/token` with JSON `grant_type` (`urn:ietf:params:oauth:grant-type:jwt-bearer`), `assertion`, `federation_rule_id`, `organization_id`, `service_account_id` and `workspace_id`; no version or beta header. The 200 carries `access_token`, `token_type`, `expires_in`, `scope` and `workspace_id`, plus undocumented `next_challenge*` fields: decode leniently. **Every denial is the same opaque 401**, so `jti_reused` can't be detected: map any 401/403 to `llm.ErrAuth` and drop the dormant `jti_reused` branch. `expires_in` = `min(rule lifetime, 2 × the JWT's remaining life)`. Re-exchange only on a newer `iat`. Pin on the exchange response's `workspace_id`, then on the header. Add the `sk-ant-oat01-` prefix to the log canary. | ✅ recorded |

## Accepted-risk register

| Risk | Decision | Mitigation in v2 | Revisit |
|------|----------|------------------|---------|
| **No off-node backups**: a data-loss window up to ~7 days (the weekly Hostinger image); an R-d restore brings back erases done since the snapshot | **D12** ([ADR-0028](../adr/0028-object-storage-and-backups.md)) | MI-2 prune guards; a manual snapshot right before every contract, erase or GA tag; the weekly image ≤ 7 days old before risky steps; the R-d procedure re-runs listed erases; the privacy notice states the window | D12's triggers (backups to the second VPS); before the opening (v3) |
| **No alerting**: a failure is seen only when the owner looks | **D34** ([ADR-0035 §3](../adr/0035-v2-operations-nats-auth-limits-capacity.md#3-no-alerting-in-v2-owner-d34)) | `host-verify --cluster` (MI-8) after host changes and in tag checklists; landscape + kubescope; verify-by-looking after every tag; the PAT expiry date above | **before the first real invite** (a v3 opening gate): adopt the kept healthchecks.io design or re-accept in writing |
| Untrusted code runs on the production node | D21 ([ADR-0030](../adr/0030-runner-technology-and-host-hardening.md)) | R1 + H0 + D22 patch cadence; `sandbox-guards`; runner at the lowest priority; signup never `open` while the runner is on the node | any D21 trigger (open signup, unpatched reachable LPE > 7 d, TR-STEAL) → R2 |
| No staging environment | ADR-0034 §5 | compose at prod parity, `-rc` images, cohort dark launch on prod | a second node, or > 10 active learners after v3 with a contract migration touching their data |
| Single gateway replica (cache epoch + in-process limiter) | rollout §12 | recorded scale-out blocker | before any gateway scale-out |

## Open owner items

From [rollout §13](rollout-plan.md#13-open-owner-items), updated with this session's decisions.

| Item | Needed by | Default / decision | State |
|------|-----------|--------------------|-------|
| PRD Q5: the pilot course | P entry | go-concurrency (SQL fallback) · confirmed before launch of [ds-p-01](sprints/sprint-ds-p-01.md) (`ev-q5`, in its `## Before you launch (owner)` block) | ⬜ open |
| PRD Q7: per-problem time budgets | a future research session; not a gate | manifest default 45 min, hint at 15 (D18) | open (not gating) |
| Artboard production | before M1b | **BP3: agents draft all ~30 boards incl. the heroes.** Since **D40** each board PR lands on CI green and the merge is the freeze; the owner may review afterwards | ✅ decided 2026-09-24 (review gate replaced by D40, 2026-09-25) |
| Spike go-aheads (P0–P3 + image volume; WIF) | by Fri 2026-10-09 | one spike week, WIF included if it fits. **D40:** launching spk-01 (spk-03 for WIF) is the go-ahead; the second-VPS answer is due before launch of spk-02 (`ev-spike-goahead`) | ✅ exercised 2026-09-25 (D41): spk-01 and spk-02 (env A = `skriptvalley-vps`); spk-03 (WIF) |
| **Delete the spk-03 Console objects** | before mi-12 creates the production issuer, which uses the same default `iss`; mi-12's entry checks it | In the Console, in this order: delete the rule `xlearn-wif-spike-probe` (`fdrl_01JM53aDwbkXy1RJ2fhG9LCf`), then the issuer `xlearn-wif-spike-k3s` (`fdis_01VKrjGxwBjDDrCASJS1CUQw`), then the service account `xlearn-wif-spike` (`svac_019eVYj3HcGdou7MMvmDWzZN`); finally archive the workspace `xlearn-wif-spike` (`wrkspc_017ZKXnJ1a9BxwcEm7e9rhGi`). The issuer is already inert, because its signing key went with the purged VM. No API key was ever created ([t5 §15](research/t5-platform-ai.md#15-wif-spike-result-spk-03-2026-09-25)) | ⬜ open (owner follow-up, not a wait) |
| **Raise the org's monthly spend limit** (Sujay's Individual Org: **$5** today) | before launch of [mi-12](sprints/sprint-mi-12.md) (`ev-provider-runbook`) | At least the sum of the workspace limits the org holds: $15 for `xlearn-platform-prod` (dogfood) plus about $50 for `xlearn-calib` during M4 bring-up. Enough for $100 from the v3 opening. No workspace can spend past the org limit (spk-03) | ⬜ open (owner) |
| **Re-run spk-02's ⛔ rows** (~~TSAN, PG, amd64 allowlists + KILL, x86_64-only profile, scoped AppArmor remount rule, image volume (i)–(iii)~~) | before mi-09's host-window PR and before M3 | [spk-02](sprints/sprint-spk-02.md) relaunched on env A `skriptvalley-vps` on 2026-09-25 (second session), in three blocks | ✅ done 2026-09-25: all three blocks landed, MI-10 ✅ |
| October host-window date | when the spike is booked | **Sat 2026-10-24**, batched with MI-11a (BP4, tentative) | ✅ booked (tentative) |
| S6 scheduling | before the M6a design freeze (before launch of ds-m6a-01) | **Sat 2026-09-26 (D41)**, owner present; ≤ 1 h `ev-s6-recheck` before ds-m6a-01 | ✅ booked |
| Privacy-notice text review | the v3 opening ([rollout §11](rollout-plan.md#11-opening-gates-v3)) | **moved from l-05 by D40:** l-05 lands the agent-drafted notice; the owner reviews it before the first real invite, and a revision is a content PR | ⬜ v3 |
| infra#28 | before MI-0 | merged 2026-09-24 | ✅ |
| MI-2a / MI-2b / MI-2c | now | done 2026-09-24 (v1.5.2; infra#29/#30; xlearn#51–#53) | ✅ |
| MI-5b (DNS, cert, bookmarks) | before the first `tester` (L-E) | weeks 2–4 ([mi-04](sprints/sprint-mi-04.md), `ev-mi5b-dns`) | ⬜ |
| ADR sign-off (0026–0035) | this session | **BP2:** 7 Accepted; 0030 / 0031 / 0032 at their spikes | ✅ decided 2026-09-24 |
| Alerting revisit (D34) | before the first real invite (v3) | the kept healthchecks.io design | ⬜ v3 |

## Decisions log

Notable calls not (yet) worth a full ADR, newest first. Promote to an ADR if they harden.

| Date | Decision | Notes |
|------|----------|-------|
| 2026-09-25 | **spk-03: WIF GO, with `check_jti=false` on the one-rule issuer** (the pre-decided path). k3s v1.36.4 projected tokens carry a `jti`. The kubelet rotates the file at 80 % of TTL plus up to one pod sync: 2881 s (80.0 %) at 3600 s, and 81–91 % at 600 s. An in-place container restart keeps the file, so judge's boot exchange re-presents a used `jti`. It then gets an opaque 401 `Authentication failed` until the next rotation, and that 401 can't be told apart from a real auth failure. Exchange p50 0.33 s / max 0.47 s; first Messages call p50 1.61 s / max 1.82 s; `anthropic-workspace-id` matched 5/5. **This contradicts ADR-0031 §2, so mi-12 amends it at acceptance:** the Console offers no `workspace:inference` scope, so the rule is `workspace:developer` and the token reaches Files and Batches (200/200). The request builder, not the credential, enforces Messages-only. Further inputs for mi-12: set the rule lifetime to 1 h explicitly (the Console defaults to 10 min); the inline JWKS is the `keys` array; the org's $5 monthly spend limit must be raised before the $15 workspace limit means anything. | [t5 §15](research/t5-platform-ai.md#15-wif-spike-result-spk-03-2026-09-25). ADR-0031 stays Proposed, with six amendments proposed. The `check_jti=false` confirming re-run is handed to mi-12. The Console clean-up and the org-limit raise are owner items. MI-1 (Anthropic 2FA) is still not recorded as done. The org UUID was missing from the launch message and was supplied by message. No production change. |
| 2026-09-25 | **spk-02 done → MI-10 ✅: Spike P0–P3 GO and image-volume GO.** Re-run block 3 closes the first session's AppArmor finding. The broad `remount,` is replaced by two `ro`-only remount rules: the jail root `/` with `(ro, nosuid, noatime, bind)`, and read-only binds under `/jail/**` with `(ro, nosuid, nodev, rbind)`. The read-write remounts of `/`, `/sys` and `/jail/**` are now refused with EACCES, and the jail still sets up and runs Go, C++ and Python with every bind read-only. go-sandbox's `WithBind(…, true)` can't be used, because AppArmor 4.0 can't match its `rprivate` remount; m3-04 builds read-only binds without `MS_PRIVATE`. The final AppArmor profile and the pod seccomp file are recorded verbatim for mi-09. No fallback and no owner decision. | [t3 §16.2 block 3, §16.4](research/t3-sandbox.md#164-mi-10-verdict-and-proposed-adr-0030-deltas). The M3 checklist's first line is ticked. Env A is torn down to its baseline. No production change. |
| 2026-09-25 | **spk-02 re-run, block 2: Q-C GO.** On env A (amd64, kernel 6.8.0-142, `mmap_rnd_bits=32`, in the runner pod shape), the Go 1.26 race runtime works in the jail **without any ASLR policy**: TSAN never calls `personality`, `ADDR_NO_RANDOMIZE` stays denied, and go-race stays in the pilot (→ p-01). Postgres + SQL balloon → MLE in the case cgroup. **amd64 exec allowlists** (→ m3-04) come from RET_LOG through auditd (`lost 0`): `go` 27, `cpp` 19, `python` 39, `go-race` 40. The fixed rules are `clone` with `CLONE_THREAD` only, `prctl` with `PR_SET_VMA` only, `clone3` → ENOSYS, and non-x86_64/x32 → KILL. The KILL re-run gives 66/66 per language and 0 unexpected SIGSYS. **The pod profile's architectures: x86_64-only proposed** (proposed ADR-0030 delta → m3-03): nothing broke, and it closes the ia32 `int $0x80` entry point. The final pod seccomp JSON is recorded verbatim for mi-09. | [t3 §16.2 block 2, §16.4](research/t3-sandbox.md#162-p3-amd64-replay-spk-02). The pod seccomp file is sha256 `730a7a55…418d`. The `personality` variants are not needed. Block 3 (AppArmor) follows. No production change. |
| 2026-09-25 | **spk-02 re-run, block 1: image-volume GO (kubelet defaults), under the pre-decided chain (D40).** On env A's k3s v1.36.4 against a password-protected private registry: (i) a read-only image-volume mount under PSA baseline; (ii) a pod without the pull secret is refused the cached pack (`ErrImagePull … no basic auth credentials`; `ErrImageNeverPull` with `Never`), in another namespace and in the same one; (ii-b) still refused after a k3s restart. **No `AlwaysVerify`** is needed. (iii) The initContainer fallback also works (variant A recommended). **→ mi-09:** the pack credential must exist only as the imagePullSecret, because node-level credentials (`registries.yaml` `auth:`, or a node docker config) mark the cached image readable by every pod. **→ p-01/m3-04:** the GOCACHE seed is a read-only seed used in place, with `TMPDIR` on tmpfs. | [t3 §16.2–16.3](research/t3-sandbox.md#163-image-volume-spk-02). ADR-0027's image-volume line stands (no m3-07 amendment). Blocks 2–3 follow. No production change. |
| 2026-09-25 | **spk-02: partial, interrupted ⛔.** On env A (`skriptvalley-vps`, amd64, kernel 6.8.0-142, `mmap_rnd_bits=32`, k3s v1.36.4) the §16.1 jail setup reproduces, and spk-01's 17 negative probes plus `move_mount`/`mount_setattr` stay denied under the amd64 pod profile. **Finding → [mi-09](sprints/sprint-mi-09.md):** the broad AppArmor `remount,` rule lets the supervisor remount `/` and `/sys` read-write, so it must be replaced by jail-scoped remount rules before the host window. The line was stopped at the finding, as the probe rules require. **Observed:** the ia32 `int $0x80` entry point is open under the three-arch baseline (evidence for x86_64-only, m3-03); the RuntimeDefault-derived pod profile denies `personality(ADDR_NO_RANDOMIZE)` (ASLR policy still open, p-01). **Not run:** TSAN, PG, the allowlists, the KILL re-run, the x86_64-only variant and image volume (i)–(iii). No fallback was chosen, the pre-decided chain still applies, and the MI-10 line stays ⛔. | [t3 §16.2–16.4](research/t3-sandbox.md#162-p3-amd64-replay-spk-02). Teardown done the same day. No production change. |
| 2026-09-25 | **spk-01: R1 is viable (Q-A GO, Q-B GO).** Mechanism chosen: **go-sandbox `forkexec.Runner`** (low-level Runner, no user namespace), spawned into the case cgroup via `CLONE_INTO_CGROUP`. **nsjail (R1-N) was not needed**; no R1-U/R1b fallback; **no owner decision required.** No timing conclusions (arm64/HVF). | Full table [t3 §16.1](research/t3-sandbox.md#161-p0p2-spk-01-arm64-multipass). **Host-file deltas for [mi-09](sprints/sprint-mi-09.md):** (1) the seccomp profile **must add `pivot_root`** — it is not in containerd's RuntimeDefault and the jail EPERMs without it (the key finding); (2) the AppArmor ro-bind **remount** rule needs finalising on amd64 (spk-02) — fresh-mount `/jail` confinement is proven; (3) overlayfs mounts EPERM under `hostUsers:false` (GOCACHE seed must not be an in-pod overlay). **SETPCAP:** go-sandbox's `DropCaps` zeroes eff/prm/inh (no SETPCAP needed) but does **not** drop the bounding set; SETPCAP is currently unused — keep only for a future bounding-set drop. subuid `kubelet:1073741824:7208960`, ×50 recreate 50/50, `getsubids` from `uidmap`. |
| 2026-09-25 | **D41: spikes first (owner).** All four spikes run before any build sprint: spk-01 + spk-02 on Fri 2026-09-25 (agent-only, after MI-0), spk-03 + spk-04 on Sat 2026-09-26. spk-02 runs on the owner's spare `skriptvalley-vps` (free for PoCs; not the D12/D21 production-runner concern) with a private registry in place of mi-07's GHCR image; the MI-9 and chart-0.3.0 gates are dropped for it. `xlearn-wif` VM created 2026-09-25 for spk-03's Console step. S6 sole-passer tie-break answered **yes**; a ≤ 1 h `ev-s6-recheck` runs before ds-m6a-01 (non-blocking). The prod SSH alias is now `sujaykumar-vps` (the `vps` alias was removed); docs and infra swept. | Owner answers 2026-09-25 |
| 2026-09-25 | **D40: sprint merge directive (owner).** The owner's words: "I want each prompt to finish with changes land-and-sync and consider my approval for all changes for the given prompt. Do not stop for my review." **Launching a v2 sprint prompt is the owner's approval for every change it makes:** merges and tags (contract, erase and GA ones included), infra PRs, design-board freezes, ADR acceptances, owner content the agent drafts (the privacy-notice text, statement rewrites), and the production operations the prompt specifies (e.g. `identity admin …` over `kubectl exec`, host-window steps on their date). No session stops for review. | **What changed:** every prompt ends with `## Ship (land-and-sync — owner approval pre-granted)`. Design PRs merge on CI green and **the merge is the freeze**: board gates read "`ds-…` merged", and the `ev-freeze-ds-*` events are automatic. The GA PR merges on CI green. Launching a spike is its go-ahead (D23); a result outside every pre-decided path still lands, with the gate ⛔ "needs owner decision". Owner-only actions (hPanel snapshots, provider consoles, accounts, PATs and keys, DNS, S6 presence, pack authoring) move to each prompt's `## Before you launch (owner)` block; if one is missing, the session lands what doesn't depend on it and marks ⛔. Owner content lands as drafted, and the privacy-notice review moves to the v3 opening gates ([rollout §11](rollout-plan.md#11-opening-gates-v3)). **Supersedes** D38's (BP3) review gate and AGENT.md's former "v2 exceptions". **Unchanged:** hard entry gates, D34, D35, the release checklist, and the owner's in-session "hold / don't ship". [feasibility D40](feasibility.md#decisions-log-newest-first) · [AGENT.md](../../AGENT.md) |
| 2026-09-24 | **BP1: scaffold everything now.** Full sprint plans + self-contained prompts for every v2.0 sprint, every v2.1 sprint (M6a, M6b), M5 and the spikes; M6c (v2.2) gets plan-outline cards only. | 89 sprints: 86 plan + prompt pairs, 3 M6c outline cards. [build-plan](build-plan.md) · [sprints](sprints/README.md) · [prompts](prompts/README.md). |
| 2026-09-24 | **BP2: ADR sign-off.** 0026, 0027, 0028, 0029, 0033, 0034, 0035 → **Accepted**, with their amendments folded into the ADRs they amend. **0030, 0031, 0032 stay Proposed** until their spikes report. | Acceptance is a task in the consuming sprint: 0030 in [m3-03](sprints/sprint-m3-03.md) task 1 (sandbox spike GO, before M3); 0031 in [mi-12](sprints/sprint-mi-12.md) (WIF spike, before M4); 0032 in [ds-m6a-01](sprints/sprint-ds-m6a-01.md) task 1 (S6, before the M6a design freeze). |
| 2026-09-24 | **The October host window runs on the spike GO while ADR-0030 is still Proposed (intentional).** [mi-09](sprints/sprint-mi-09.md) turns the [spk-01](sprints/sprint-spk-01.md) + [spk-02](sprints/sprint-spk-02.md) GO (t3 §16) into the host sandbox block, L23 kubelet args and pid limits that `ev-host-window` (Sat 2026-10-24) applies to production; ADR-0030 is accepted after that, in [m3-03](sprints/sprint-m3-03.md) task 1 (late October, before M3). | The spike GO and t3 §16 are the window's authority, and mi-09's PR records any divergence from t3 §8.7. m3-03 task 1 then accepts ADR-0030 with the same spike numbers (folding ADR-0035 §6 into §5), so the ADR catches up with what the window applied. Runner code never waits on the window (BP2). |
| 2026-09-24 | **BP3: agents draft all v2 artboards** (AB01–AB30, the 5 heroes ★ included) as static HTML on `theme.css` under `design-system/screens/v2/`; the owner only reviews. | Replaces rollout §9's hybrid production. 10 design sprints (`ds-*`); AB23 dropped (D33); AB31 outline (v2.2). The freeze is required before the milestone's first UI sprint. **Review gate replaced by D40 (2026-09-25):** each `ds-*` PR merges on CI green and the merge is the freeze. |
| 2026-09-24 | **BP4: calendar (tentative, owner-booked).** H0 reboot Fri 2026-09-25 (MI-0); spike week Mon 2026-10-12 → Fri 2026-10-16 (P0–P3 + image-volume spike on a throwaway multipass/k3s; WIF ≤ ½ day if it fits, else before M4); October host window **Sat 2026-10-24** (MI-11 + L23 + pid limits + k3s/CNPG bumps, batched with MI-11a); S6 any time the owner is present, before the M6a design freeze; GA snapshot at the `v2.0.0` tag. | Owner-gated items are calendar events, not sprints; a sprint may prepare one (runbook, scripts, PRs). Full list: [Owner calendar events](#owner-calendar-events). **D40 (2026-09-25):** owner-only actions are done before launch of the sprint that needs it; review-only events are automatic. |
| 2026-09-24 | **Sprint ids are milestone-scoped** (`mi-`, `m1-`, `m2-`, `m3-`, `p-`, `m4-`, `l-`, `ga-`, `m5-`, `m6a-`, `m6b-`, `m6c-` outline, `spk-` spikes, `ds-<milestone>-` design). | `mi-NN` sprint ids are **not** rollout steps `MI-NN`; the [MI track](#mi-track-rollout-2) maps them. |
| 2026-09-24 | **Sizes vs the rollout ranges.** v2.0 = 60 build/release sprints (vs 41–52) + 7 design + 3 spikes; v2.1 = 11 (inside 9–12) + 3 design + 1 spike. | Over-range: MI 13 (single-PR slices, MI-4 on its own so a VAP stall never blocks N3), M1 10 (m1-01 and m1-07 split into m1-09, m1-10), L 5 (the L-A/L-C front door [l-05](sprints/sprint-l-05.md) held to v1.17.0), GA 2 (GA PR + cut). Engineering is not the long pole (rollout §6). |
| 2026-09-24 | **Carried constraints (never contradicted by any sprint).** v2 is owner-only (D35): invite flow + gate L built and rehearsed with a tester, real learners at v3. 1.x minors during the build; `v2.0.0` = owner-facing GA; `v2.1.0` = interviewer; M5 later 2.x (D32). No alerting anywhere (D34): MI-8 = the `host-verify --cluster` extension only. No backups or object store (D11, D12). | Stopgaps MI-2a/2b/2c already done (v1.5.2). Live release v1.5.2. |
| 2026-09-24 | **MI-2a/2b/2c done before the build (pre-build stopgaps).** Bounded ImagePolicies `<2.0.0` (infra#29) + `.release-line` (xlearn#53); auto-link fix (xlearn#51); `SIGNUP_MODE` closed (xlearn#52, infra#30). | Live in v1.5.2. The `DEV_AUTH` guard on `open` is M1b ([m1-04](sprints/sprint-m1-04.md)). MI-1a dropped (D34). |
| 2026-09-24 | **v2 planning merged** (xlearn#55): feasibility D0–D35, v2 PRD, ADRs 0026–0035 (Proposed), research T0–T7, rollout plan. | The source for this build plan. |

## Blocked / needs input

- Waiting on the owner for the MI-0 H0 reboot, Fri 2026-09-25; [mi-01](sprints/sprint-mi-01.md)'s MI-2 PR may go first if MI-0 slips.

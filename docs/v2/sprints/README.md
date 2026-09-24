# v2 sprints

Per-sprint **plans** for the v2 build. Each sprint is one flat file `sprint-<id>.md` and pairs with exactly one
execution **prompt** in [`../prompts/`](../prompts/), `prompt-<id>.md`, designed to run **one prompt per sprint per
session**. Sequencing, milestones, calendar and the artboard register live in [`../build-plan.md`](../build-plan.md);
the live cross-sprint tracker is [`../status.md`](../status.md); the source is the [rollout plan](../rollout-plan.md).

## Layout

```
docs/v2/
  rollout-plan.md          the source: MI track, milestone map, gates, tag timeline, artboards
  build-plan.md            the static plan: principles, milestone map, sprint table, graph, calendar, artboards
  status.md                the live tracker (sprint board, MI rows, tags, flags, content, events, logs, decisions)
  feasibility.md · research/t0…t7-*.md   decisions D0–D39 and the deep detail behind every sprint
  sprints/
    README.md              this index + the status protocol
    sprint-<id>.md         one plan per sprint (89: 86 full plans + 3 M6c outline cards)
  prompts/
    README.md
    prompt-<id>.md         one self-contained prompt per sprint (86; none for the M6c outline cards)
design-system/screens/v2/  the v2 artboards drafted by the ds-* sprints (preview-only, never shipped)
```

**Sprint ids are milestone-scoped.** The number is the sprint's slot within its milestone, not its global order
(the global order is the `#` column in the [build plan](../build-plan.md#sprint-table-recommended-order)).

| Prefix | Meaning | Example |
|--------|---------|---------|
| `mi-` | MI infra-first track. **`mi-NN` ≠ rollout step `MI-NN`**; [status.md → MI track](../status.md#mi-track-rollout-2) maps them | `mi-01` = MI-2 + MI-3 |
| `m1-` `m2-` `m3-` `m4-` `m5-` | product milestones | `m3-07` = M3-1 on prod |
| `l-` | learner gate L (∥ M3/M4) | `l-02` = L-E erase, v1.12.0 |
| `p-` | pilot course | `p-01` = go-race runner profile |
| `ga-` | the v2.0.0 GA PR and cut | `ga-02` = tag v2.0.0 |
| `m6a-` `m6b-` | interviewer (v2.1) | `m6b-04` = v2.1.0 |
| `m6c-` | v2.2 **outline cards** (plan only, no prompt) | `m6c-03` |
| `spk-` | throwaway spikes | `spk-01` = sandbox P0–P2 |
| `ds-<milestone>-` | design sprints (draft the boards; owner review = freeze) | `ds-m3-01` = AB07★, AB08, AB11 |

**Plan shape.** A header (milestone · track · order; prereqs; unblocks; **release action**; calendar; the prompt
link) → **Status** table with a **Repo** column → **Entry gates** as checkboxes → goal → scope in/out → tasks
(`### n · title [Repo]`) → acceptance criteria → **Release** (tag sprints carry the ADR-0034 §6 checklist) → DoD →
risks. Sprints that open M3 or cut GA carry the rollout §5 / §4 checklists **verbatim**.

## Index

Grouped by milestone; **Order** is the global recommended execution order. Tracks: `infra`, `product`, `content`,
`design`, `spike`; different tracks interleave and only the prereqs bind.

| Order | Sprint | Focus | Milestone | Track | Release | Plan | Prompt |
|-------|--------|-------|-----------|-------|---------|------|--------|
| 1 | `mi-01` | Prune guards + chart 0.3.0 knob union (MI-2, MI-3) | MI | infra | infra PRs only (MI-2 first, then MI-3) | [sprint-mi-01](sprint-mi-01.md) | [prompt-mi-01](../prompts/prompt-mi-01.md) |
| 3 | `mi-02` | host-verify --cluster extension (MI-8) | MI | infra | infra PR only (host script) | [sprint-mi-02](sprint-mi-02.md) | [prompt-mi-02](../prompts/prompt-mi-02.md) |
| 5 | `mi-07` | Evalpack plumbing (MI-9) | MI | infra | infra PRs only (+ evalpack repo; `v0.1.0` image, no `>=1.0.0`) | [sprint-mi-07](sprint-mi-07.md) | [prompt-mi-07](../prompts/prompt-mi-07.md) |
| 10 | `mi-05` | N0: NATS topology, dead letters, identity on NATS, client options, pool pins (MI-6) | MI | product | merge only (ships dark in v1.6.0) | [sprint-mi-05](sprint-mi-05.md) | [prompt-mi-05](../prompts/prompt-mi-05.md) |
| 12 | `mi-14` | sandbox-guards: empty default-deny xlearn-runner namespace + VAP (MI-4) | MI | infra | infra PRs only | [sprint-mi-14](sprint-mi-14.md) | [prompt-mi-14](../prompts/prompt-mi-14.md) |
| 13 | `mi-03` | Fences: databases/messaging ingress, xlearn ingress (MI-5, MI-5a) | MI | infra | infra PRs only (MI-5 → MI-5a) | [sprint-mi-03](sprint-mi-03.md) | [prompt-mi-03](../prompts/prompt-mi-03.md) |
| 15 | `mi-04` | Admin consoles to ops.sujaykumar.dev (MI-5b) | MI | infra | infra PRs only | [sprint-mi-04](sprint-mi-04.md) | [prompt-mi-04](../prompts/prompt-mi-04.md) |
| 16 | `spk-01` | Sandbox mechanism spike P0–P2 (multipass arm64, throwaway) | MI | spike | no merge (spike, throwaway); results docs PR | [sprint-spk-01](sprint-spk-01.md) | [prompt-spk-01](../prompts/prompt-spk-01.md) |
| 17 | `spk-02` | P3 amd64 replay + image-volume spike (throwaway) | MI | spike | no merge (spike, throwaway); results docs PR | [sprint-spk-02](sprint-spk-02.md) | [prompt-spk-02](../prompts/prompt-spk-02.md) |
| 18 | `spk-03` | WIF spike (≤ ½ day, throwaway) | MI | spike | no merge (spike, throwaway); results docs PR | [sprint-spk-03](sprint-spk-03.md) | [prompt-spk-03](../prompts/prompt-spk-03.md) |
| 19 | `mi-06` | NATS auth server-first: N1 → N2 → N3 (MI-7) | MI | infra | infra PRs only (N1, N2, N3) | [sprint-mi-06](sprint-mi-06.md) | [prompt-mi-06](../prompts/prompt-mi-06.md) |
| 25 | `mi-08` | Limit hygiene (MI-11a) + Track B slice 1: PSA labels, SA tokens off (MI-15 part) | MI | infra | infra PRs only (by the 10-24 window) | [sprint-mi-08](sprint-mi-08.md) | [prompt-mi-08](../prompts/prompt-mi-08.md) |
| 26 | `mi-09` | October host-window prep: sandbox block, L23 kubelet args, pid limits, bumps (MI-11) | MI | infra | infra PRs only (scripts applied in the window) | [sprint-mi-09](sprint-mi-09.md) | [prompt-mi-09](../prompts/prompt-mi-09.md) |
| 41 | `mi-10` | Runner dark on production (MI-12) | MI | infra | infra PRs only (runner dark) | [sprint-mi-10](sprint-mi-10.md) | [prompt-mi-10](../prompts/prompt-mi-10.md) |
| 52 | `mi-11` | Track B finish: xlearn egress, PG connection limits, Renovate, N4 (MI-15) | MI | infra | infra PRs only (+ xlearn PR, no tag) | [sprint-mi-11](sprint-mi-11.md) | [prompt-mi-11](../prompts/prompt-mi-11.md) |
| 56 | `mi-12` | M4 gates: judge 443 egress, LLM secret, provider runbook (MI-14) + ADR-0031 → Accepted | MI | infra | infra PRs only (+ ADR-0031 docs PR) | [sprint-mi-12](sprint-mi-12.md) | [prompt-mi-12](../prompts/prompt-mi-12.md) |
| 81 | `mi-13` | M6b gates: Permissions-Policy, camera/mic deny, coach sizing, SDP route, WSS egress (MI-16) → v2.0.x patch | MI | infra | **tag v2.0.x patch** (MI-16) + infra PRs first | [sprint-mi-13](sprint-mi-13.md) | [prompt-mi-13](../prompts/prompt-mi-13.md) |
| 2 | `ds-m1-01` | Design M1: coach states, course nav, revision v2 (AB01–AB03) | M1 | design | PR, stop for owner review (design) | [sprint-ds-m1-01](sprint-ds-m1-01.md) | [prompt-ds-m1-01](../prompts/prompt-ds-m1-01.md) |
| 4 | `m1-01` | Curriculum spine part 1: compose parity, course manifest + golden, item schema freeze (M1a) | M1 | product | merge only (ships in v1.6.0) | [sprint-m1-01](sprint-m1-01.md) | [prompt-m1-01](../prompts/prompt-m1-01.md) |
| 8 | `m1-09` | Curriculum spine part 2: converter, loader + guards, curriculum expand migration, content CI (M1a) | M1 | product | merge only (ships in v1.6.0) | [sprint-m1-09](sprint-m1-09.md) | [prompt-m1-09](../prompts/prompt-m1-09.md) |
| 11 | `m1-02` | path_slug everywhere + v2 envelope consumers + identity M1a columns → v1.6.0 | M1 | product | **tag v1.6.0** (+ infra PR before) | [sprint-m1-02](sprint-m1-02.md) | [prompt-m1-02](../prompts/prompt-m1-02.md) |
| 20 | `m1-03` | Course resolution: gateway, BFF, SPA /:course/*, producers emit v2 (M1b) | M1 | product | merge only (ships in v1.7.0) | [sprint-m1-03](sprint-m1-03.md) | [prompt-m1-03](../prompts/prompt-m1-03.md) |
| 21 | `m1-04` | Identity security floor: roles, sessions, admin CLI, DEV_AUTH guard, L3, CSP (M1b) | M1 | product | merge only (ships in v1.7.0) | [sprint-m1-04](sprint-m1-04.md) | [prompt-m1-04](../prompts/prompt-m1-04.md) |
| 22 | `m1-10` | Coach keys + provider hygiene: key_default(feature), catalog, AEAD AD, keyring re-wrap, store:false, classifier, usage | M1 | product | merge only (ships in v1.7.0) | [sprint-m1-10](sprint-m1-10.md) | [prompt-m1-10](../prompts/prompt-m1-10.md) |
| 23 | `m1-05` | Gateway limits + public-dashboard floor (L1, L2, L4–L6, L24; P1, P2, P4, P10, P11) | M1 | product | merge only (ships in v1.7.0) | [sprint-m1-05](sprint-m1-05.md) | [prompt-m1-05](../prompts/prompt-m1-05.md) |
| 24 | `m1-06` | withhold() on every surface + Markdown renderer + revision v2 (AB03) | M1 | product | merge only (ships in v1.7.0) | [sprint-m1-06](sprint-m1-06.md) | [prompt-m1-06](../prompts/prompt-m1-06.md) |
| 27 | `m1-07` | Coach D27 assist capture, mode gate, L18 caps + AB01 → v1.7.0 | M1 | product | **tag v1.7.0** | [sprint-m1-07](sprint-m1-07.md) | [prompt-m1-07](../prompts/prompt-m1-07.md) |
| 29 | `m1-08` | M1c contract → v1.8.0 | M1 | product | **tag v1.8.0** (contract; snapshot first) | [sprint-m1-08](sprint-m1-08.md) | [prompt-m1-08](../prompts/prompt-m1-08.md) |
| 6 | `ds-m2-01` | Design M2: Touch ★, catalog/agenda, public profile v2, visibility (AB04 ★, AB05, AB06, AB22) | M2 | design | PR, stop for owner review (design) | [sprint-ds-m2-01](sprint-ds-m2-01.md) | [prompt-ds-m2-01](../prompts/prompt-ds-m2-01.md) |
| 33 | `m2-01` | M2a touch-attempt engine + `touch_concluded` consumer | M2 | product | merge only (ships in v1.9.0) + own infra PR | [sprint-m2-01](sprint-m2-01.md) | [prompt-m2-01](../prompts/prompt-m2-01.md) |
| 34 | `m2-02` | M2b consumers + projections v2 → v1.9.0 | M2 | product | **tag v1.9.0** (+ ACL PR before) | [sprint-m2-02](sprint-m2-02.md) | [prompt-m2-02](../prompts/prompt-m2-02.md) |
| 36 | `m2-03` | `public-read`, `/public/stats`, visibility toggles, profile v2 (P3, P5, P6, P7, P9; AB06, AB22) | M2 | product | merge only (ships in v1.10.0) | [sprint-m2-03](sprint-m2-03.md) | [prompt-m2-03](../prompts/prompt-m2-03.md) |
| 37 | `m2-04` | Touch UI (AB04★) + catalog/agenda + Today in minutes (D4) (AB05) | M2 | product | merge only (ships in v1.10.0) | [sprint-m2-04](sprint-m2-04.md) | [prompt-m2-04](../prompts/prompt-m2-04.md) |
| 38 | `m2-05` | Producers on + `touch_scored` backfill + reader switch + replay + D2 (M2c) → v1.10.0 | M2 | product | **tag v1.10.0** | [sprint-m2-05](sprint-m2-05.md) | [prompt-m2-05](../prompts/prompt-m2-05.md) |
| 7 | `ds-l-01` | Design L: invite acceptance ★, privacy notice, erase account (AB19 ★, AB20, AB21) | L | design | PR, stop for owner review (design) | [sprint-ds-l-01](sprint-ds-l-01.md) | [prompt-ds-l-01](../prompts/prompt-ds-l-01.md) |
| 39 | `l-01` | L-E erase consumers (practice, review, assessment, coach) + identity ack path → v1.11.0 | L | product | **tag v1.11.0** (+ ACL/seed PRs before) | [sprint-l-01](sprint-l-01.md) | [prompt-l-01](../prompts/prompt-l-01.md) |
| 40 | `l-02` | L-E erase producer + `DELETE /api/me` + UI (AB21) → v1.12.0 | L | product | **tag v1.12.0** (erase; snapshot first) | [sprint-l-02](sprint-l-02.md) | [prompt-l-02](../prompts/prompt-l-02.md) |
| 47 | `l-03` | L-A admission backend (inert while closed): invites, seats, redeem, public checks, `SIGNUP_MODE=invite`, erase clears invite notes | L | product | merge only (ships dark, inert, in the next tag) | [sprint-l-03](sprint-l-03.md) | [prompt-l-03](../prompts/prompt-l-03.md) |
| 67 | `l-05` | L-A/L-C front door: auth-page invite state, acceptance step★, privacy notice, AI consents (AB19★, AB20) | L | product | merge only (ships in v1.17.0) | [sprint-l-05](sprint-l-05.md) | [prompt-l-05](../prompts/prompt-l-05.md) |
| 68 | `l-04` | Web erase for every non-owner → v1.17.0 (L-A/L-C) + L-exit rehearsal prep | L | product | **tag v1.17.0** (erase-class; + rehearsal PRs) | [sprint-l-04](sprint-l-04.md) | [prompt-l-04](../prompts/prompt-l-04.md) |
| 9 | `m3-01` | Authoring tooling T25/T26: canonical hashes, contract_hash, packlint, pre-push fingerprint hook | M3 | content | merge only (dev tools; rides the next tag) | [sprint-m3-01](sprint-m3-01.md) | [prompt-m3-01](../prompts/prompt-m3-01.md) |
| 14 | `m3-02` | Evalpack pipeline: private CI gates, generic validator, image build (E) + compose fixture pack | M3 | content | merge only (evalpack `main`; optional `v0.2.0`) | [sprint-m3-02](sprint-m3-02.md) | [prompt-m3-02](../prompts/prompt-m3-02.md) |
| 28 | `ds-m3-01` | Design M3 part 1: Workspace-Code ★, results dock, degradation badges (AB07 ★, AB08, AB11) | M3 | design | PR, stop for owner review (design) | [sprint-ds-m3-01](sprint-ds-m3-01.md) | [prompt-ds-m3-01](../prompts/prompt-ds-m3-01.md) |
| 30 | `m3-03` | Runner core: supervisor, jail, cgroups, API | M3 | product | merge only (ships in runner-v1.0.0) | [sprint-m3-03](sprint-m3-03.md) | [prompt-m3-03](../prompts/prompt-m3-03.md) |
| 31 | `m3-04` | Runner profiles Go/C++/Python + harness codecs + amd64 allowlists | M3 | product | merge only (ships in runner-v1.0.0) | [sprint-m3-04](sprint-m3-04.md) | [prompt-m3-04](../prompts/prompt-m3-04.md) |
| 32 | `m3-15` | Runner release: reproducible image, runner-release.yml, acceptance suite, TL baselines → runner-v1.0.0 | M3 | product | **runner-v1.0.0** | [sprint-m3-15](sprint-m3-15.md) | [prompt-m3-15](../prompts/prompt-m3-15.md) |
| 35 | `ds-m3-02` | Design M3 part 2: Problems, Arena, Week/Mistakes/Progress deltas (AB09, AB10, AB12) | M3 | design | PR, stop for owner review (design) | [sprint-ds-m3-02](sprint-ds-m3-02.md) | [prompt-ds-m3-02](../prompts/prompt-ds-m3-02.md) |
| 42 | `m3-05` | judge service: skeleton, schema, evalpack loader, erase consumer (M3-1) | M3 | product | merge only (ships dark in v1.13.0) | [sprint-m3-05](sprint-m3-05.md) | [prompt-m3-05](../prompts/prompt-m3-05.md) |
| 43 | `m3-06` | judge queue, runner lane, graders, contexts, internal context endpoints | M3 | product | merge only (ships dark in v1.13.0) | [sprint-m3-06](sprint-m3-06.md) | [prompt-m3-06](../prompts/prompt-m3-06.md) |
| 44 | `m3-14` | judge admission (L9–L15, L6), learner API + DTO allowlist, drafts, arena history/progress, telemetry, judge admin | M3 | product | merge only (ships dark in v1.13.0) | [sprint-m3-14](sprint-m3-14.md) | [prompt-m3-14](../prompts/prompt-m3-14.md) |
| 46 | `m3-07` | M3-1 on prod: evalpack v1.0.0, judge ACL, tag v1.13.0, judge HelmRelease (MI-13) | M3 | product | **evalpack v1.0.0 + tag v1.13.0** (ACL PR before, HelmRelease after) | [sprint-m3-07](sprint-m3-07.md) | [prompt-m3-07](../prompts/prompt-m3-07.md) |
| 48 | `m3-08` | practice: judge consumer, reconciler, D15/D16/D18 grading strategy | M3 | product | merge only (ships in v1.14.0) + ACL PR | [sprint-m3-08](sprint-m3-08.md) | [prompt-m3-08](../prompts/prompt-m3-08.md) |
| 49 | `m3-09` | gateway judge BFF, DTO allow/deny lists, typed 413, L16, degradation status | M3 | product | merge only (ships in v1.14.0) | [sprint-m3-09](sprint-m3-09.md) | [prompt-m3-09](../prompts/prompt-m3-09.md) |
| 50 | `m3-10` | review + assessment on judge signals (mistake pre-fill, P7 checked, P8) | M3 | product | merge only (ships in v1.14.0) | [sprint-m3-10](sprint-m3-10.md) | [prompt-m3-10](../prompts/prompt-m3-10.md) |
| 53 | `m3-11` | Workspace-Code UI (AB07★) + CodeMirror lazy load | M3 | product | merge only (ships in v1.14.0) | [sprint-m3-11](sprint-m3-11.md) | [prompt-m3-11](../prompts/prompt-m3-11.md) |
| 54 | `m3-12` | Results dock, Problems, Arena, degradation badges (AB08–AB11) | M3 | product | merge only (ships in v1.14.0) | [sprint-m3-12](sprint-m3-12.md) | [prompt-m3-12](../prompts/prompt-m3-12.md) |
| 55 | `m3-13` | Week/Mistakes/Progress deltas (AB12) + M3 exit tests → v1.14.0 | M3 | product | **tag v1.14.0** (+ `JUDGE_BASE_URL` PR after) | [sprint-m3-13](sprint-m3-13.md) | [prompt-m3-13](../prompts/prompt-m3-13.md) |
| 45 | `ds-p-01` | Design P: Workspace-Quiz, go-concurrency multi-file + race, AB02/AB05 full fidelity (AB14, AB15) | P | design | PR, stop for owner review (design) | [sprint-ds-p-01](sprint-ds-p-01.md) | [prompt-ds-p-01](../prompts/prompt-ds-p-01.md) |
| 57 | `p-01` | go-race runner profile → runner-v1.1.0 | P | product | **runner-v1.1.0** (judge code rides v1.15.0) | [sprint-p-01](sprint-p-01.md) | [prompt-p-01](../prompts/prompt-p-01.md) |
| 58 | `p-02` | Multi-course: go-concurrency manifest (preview), catalog, agenda, nav | P | product | merge only (ships in v1.15.0) | [sprint-p-02](sprint-p-02.md) | [prompt-p-02](../prompts/prompt-p-02.md) |
| 59 | `p-03` | Quiz widget (AB14) + race verdict UI (AB15) → v1.15.0 | P | product | **tag v1.15.0** + **evalpack v1.N.0** | [sprint-p-03](sprint-p-03.md) | [prompt-p-03](../prompts/prompt-p-03.md) |
| 51 | `ds-m4-01` | Design M4: AI suggestion/dispute ★, pointer notes, allowance + consents (AB16 ★, AB17, AB18) | M4 | design | PR, stop for owner review (design) | [sprint-ds-m4-01](sprint-ds-m4-01.md) | [prompt-ds-m4-01](../prompts/prompt-ds-m4-01.md) |
| 60 | `m4-01` | platform/llm extraction + WIF auth + retention policy | M4 | product | merge only (ships in v1.16.0) | [sprint-m4-01](sprint-m4-01.md) | [prompt-m4-01](../prompts/prompt-m4-01.md) |
| 61 | `m4-02` | judge ai: Scorer, ledger, llm lane, breaker, caps (L17) | M4 | product | merge only (ships in v1.16.0) | [sprint-m4-02](sprint-m4-02.md) | [prompt-m4-02](../prompts/prompt-m4-02.md) |
| 62 | `m4-03` | Analyzer (D16/D26), pointer notes, `evaluation_analyzed` + acceptance-set harness | M4 | product | merge only (ships in v1.16.0) + ACL PR | [sprint-m4-03](sprint-m4-03.md) | [prompt-m4-03](../prompts/prompt-m4-03.md) |
| 63 | `m4-04` | Provisional grades, dispute, re-grade, honor claims (D14) | M4 | product | merge only (ships in v1.16.0) | [sprint-m4-04](sprint-m4-04.md) | [prompt-m4-04](../prompts/prompt-m4-04.md) |
| 64 | `m4-05` | AI allowance + consents (`account_consent` AI kinds) | M4 | product | merge only (ships in v1.16.0) | [sprint-m4-05](sprint-m4-05.md) | [prompt-m4-05](../prompts/prompt-m4-05.md) |
| 65 | `m4-06` | M4 UI: AI suggestion/dispute (AB16★), pointer notes (AB17), allowance + consents (AB18) | M4 | product | merge only (ships in v1.16.0) | [sprint-m4-06](sprint-m4-06.md) | [prompt-m4-06](../prompts/prompt-m4-06.md) |
| 66 | `m4-07` | Canary log test, caps sizing, ledger check → v1.16.0 (+ LLM_PLATFORM_ENABLED for the cohort) | M4 | product | **tag v1.16.0** (+ `LLM_PLATFORM_ENABLED` PR) | [sprint-m4-07](sprint-m4-07.md) | [prompt-m4-07](../prompts/prompt-m4-07.md) |
| 69 | `ga-01` | GA PR: .release-line = 2, T-1 default flips, rc rehearsal | GA | product | merge only, owner-approved GA PR (+ optional `v2.0.0-rc.N`) | [sprint-ga-01](sprint-ga-01.md) | [prompt-ga-01](../prompts/prompt-ga-01.md) |
| 70 | `ga-02` | Cut v2.0.0: pre-flip check, widen ranges, snapshot, tag, verify | GA | product | **tag v2.0.0** (widen first; GA snapshot) | [sprint-ga-02](sprint-ga-02.md) | [prompt-ga-02](../prompts/prompt-ga-02.md) |
| 71 | `spk-04` | S6 voice-shell bake-off (owner present, throwaway) | M6a | spike | no merge (spike, throwaway); results docs PR | [sprint-spk-04](sprint-spk-04.md) | [prompt-spk-04](../prompts/prompt-spk-04.md) |
| 72 | `ds-m6a-01` | Design M6a part 1 + ADR-0032 → Accepted: Mock-v2, setup/consent/pre-flight, live HUD text (AB13, AB24, AB25) | M6a | design | ADR-0032 docs PR (merged) + PR, stop for owner review (design) | [sprint-ds-m6a-01](sprint-ds-m6a-01.md) | [prompt-ds-m6a-01](../prompts/prompt-ds-m6a-01.md) |
| 73 | `ds-m6a-02` | Design M6a part 2: grace/paused/resume, debrief + proposal, accessibility (AB26, AB27, AB28) | M6a | design | PR, stop for owner review (design) | [sprint-ds-m6a-02](sprint-ds-m6a-02.md) | [prompt-ds-m6a-02](../prompts/prompt-ds-m6a-02.md) |
| 74 | `m6a-01` | Interview core: schema, state machine, failsafes, caps (L19) | M6a | product | merge only (ships dark in a v2.0.x patch) | [sprint-m6a-01](sprint-m6a-01.md) | [prompt-m6a-01](../prompts/prompt-m6a-01.md) |
| 75 | `m6a-02` | Text brain: loop, classifier, give_hint, resume brief, store:false, replay tests | M6a | product | merge only (ships dark in a v2.0.x patch) | [sprint-m6a-02](sprint-m6a-02.md) | [prompt-m6a-02](../prompts/prompt-m6a-02.md) |
| 76 | `m6a-03` | Assessment deltas + proposal/accept + ScoreMock once + twin fairness gate | M6a | product | merge only (ships dark in a v2.0.x patch) | [sprint-m6a-03](sprint-m6a-03.md) | [prompt-m6a-03](../prompts/prompt-m6a-03.md) |
| 77 | `m6a-04` | judge mock budgeting + CodeMirror interview mode + bounded SSE | M6a | product | merge only (ships dark in a v2.0.x patch) | [sprint-m6a-04](sprint-m6a-04.md) | [prompt-m6a-04](../prompts/prompt-m6a-04.md) |
| 78 | `m6a-05` | M6a UI part 1: Mock-v2 (AB13), setup/consent/pre-flight/$ cap (AB24), live HUD text (AB25) | M6a | product | merge only (ships dark in a v2.0.x patch) | [sprint-m6a-05](sprint-m6a-05.md) | [prompt-m6a-05](../prompts/prompt-m6a-05.md) |
| 79 | `m6a-06` | M6a UI part 2: grace/paused/resume (AB26), debrief/proposal (AB27), accessibility (AB28) → v2.0.x patch | M6a | product | **tag v2.0.x patch** (M6a dark, cohort) | [sprint-m6a-06](sprint-m6a-06.md) | [prompt-m6a-06](../prompts/prompt-m6a-06.md) |
| 80 | `ds-m6b-01` | Design M6b: voice pre-flight/notices, voice live HUD ★ (AB29, AB30 ★) | M6b | design | PR, stop for owner review (design) | [sprint-ds-m6b-01](sprint-ds-m6b-01.md) | [prompt-ds-m6b-01](../prompts/prompt-ds-m6b-01.md) |
| 82 | `m6b-01` | VoiceShell adapter + SDP broker + sideband | M6b | product | merge only (ships dark in m6b-03's patch) | [sprint-m6b-01](sprint-m6b-01.md) | [prompt-m6b-01](../prompts/prompt-m6b-01.md) |
| 83 | `m6b-02` | Voice robustness: lease, re-attach, drain, cost + $ cap, PTT, rollover (+ coach-interview if M7 failed) | M6b | product | merge only (ships dark in m6b-03's patch) | [sprint-m6b-02](sprint-m6b-02.md) | [prompt-m6b-02](../prompts/prompt-m6b-02.md) |
| 84 | `m6b-03` | Voice UI: pre-flight/notices (AB29), live HUD★ (AB30), consent, EU gate, self-view → v2.0.x patch (M6b dark) | M6b | product | **tag v2.0.x patch** (M6b dark, cohort) | [sprint-m6b-03](sprint-m6b-03.md) | [prompt-m6b-03](../prompts/prompt-m6b-03.md) |
| 85 | `m6b-04` | Fake-media e2e + interviewer GA flip → v2.1.0 | M6b | product | **tag v2.1.0** (interviewer GA) | [sprint-m6b-04](sprint-m6b-04.md) | [prompt-m6b-04](../prompts/prompt-m6b-04.md) |
| 86 | `m5-01` | DSA evaluator-only flip (M5) | M5 | product | rides v2.1.0 if merged before m6b-04, else **tag the next minor ≥ v2.2.0** | [sprint-m5-01](sprint-m5-01.md) | [prompt-m5-01](../prompts/prompt-m5-01.md) |
| 87 | `m6c-01` | Outline: "show your work" photo + chat read-aloud + local recording download | M6c | product | outline only (v2.2) | [sprint-m6c-01](sprint-m6c-01.md) | — (outline card) |
| 88 | `m6c-02` | Outline: multi-speaker fairness check → AI-proposed voice Communication; history-calibrated estimates | M6c | product | outline only (v2.2) | [sprint-m6c-02](sprint-m6c-02.md) | — (outline card) |
| 89 | `m6c-03` | Outline: Safari leg, voice-lite on demand, weekly canary; AB31 canvas outline (SD course, v2.2+) | M6c | product | outline only (v2.2) | [sprint-m6c-03](sprint-m6c-03.md) | — (outline card) |

## Status protocol (way of working)

Progress is tracked in **two places that must stay in sync**: each sprint's **Status** table (`sprint-<id>.md`)
and the cross-sprint tracker ([`../status.md`](../status.md)). Statuses:

| Icon | Meaning |
|------|---------|
| ⬜ | Not started (M6c: "⬜ outline") |
| 🔄 | In progress (also: a design PR open and awaiting owner review; an owner task awaiting the owner) |
| ✅ | Done (its acceptance bullet passes) |
| ⛔ | Blocked (note why: an unmet entry gate, a missing owner action, a failed spike) |

**Before any work: the entry gates.** Every plan lists its entry gates as `[ ]` checkboxes (the prompt repeats them
under "verify first"). Check each one against the live state (status.md, `gh pr list`, `git ls-remote --tags`,
`ssh vps` read-only, the design freeze), tick it `[x]` with the evidence (date, PR, command output). **If a gate is
unmet, stop:** set the blocked task, or the whole sprint, to ⛔ with the gate named, and report. Never work around a
gate. The M3 hard checklist and the GA checklist are gates like any other.

**When a task changes state:**
1. Update that task's row in the sprint's **Status** table (⬜ → 🔄 when you pick it up; → ✅ when its acceptance
   criterion is met; → ⛔ if blocked, with a one-line reason). The **Repo** column is fixed by the plan:
   **X** xlearn · **I** `../infra` (a GitOps PR, never `kubectl apply`) · **H** host scripts (applied by hand over
   `ssh vps`) · **E** the private `xlearn-evalpack` repo · **O** owner action.
   An **O** task is prepared by the agent (runbook, draft PR, exact commands), then set 🔄 "awaiting owner"; carry on
   with the tasks that don't depend on it. Owner-gated milestones are **calendar events** (`ev-*` in status.md), not
   sprint work.
2. Update the sprint's **_Overall_** line (🔄 once any task is in progress; ✅ when all tasks are ✅).
3. Mirror it in [`../status.md`](../status.md): the **Sprint board** row, the **Milestones** row if the sprint
   carries a milestone, and **every other table the plan names** in its Status note — typically the **MI track**
   and **NATS** rows, **milestone → tag → floor → snapshot**, **release streams**, **ranges and release line**, the
   **flag inventory**, **content status**, **artboards**, **owner events**, **pending contracts**, **pending-smoke
   notes**, **hand-offs**, the **spike record**, the **CLI-use** and **NATS break-glass** logs, the **L rehearsal
   record** and **capacity reads**. Add a **Decisions log** line for any notable call (promote it to an ADR, next
   free number after checking peers, if it hardens).

`build-plan.md` is the *static* plan (it doesn't track live status); the sprint files + `status.md` are the single
source of progress truth. Executing a `prompt-<id>.md` includes these updates: every prompt ends with an **Update
status** step, and the status edits ride the sprint's own PR (or a small docs PR for infra-only sprints).

**Release action (every plan states exactly one; the sprint isn't ✅ until it's done):**

| Release action | What "done" means |
|----------------|-------------------|
| merge only (ships dark in `<tag>`) | squash-merged to `main`, CI green; nothing deploys until the named tag. Leave a **pending-smoke note** in status.md if the tag sprint must check something |
| tag `vX.Y.0` / a `v2.0.x` patch | the **ADR-0034 §6 release checklist** (copied into the plan) + the ADR-0035 §2 NetworkPolicy standing rule: peers' tags and PRs checked; the **next free** version, major = `.release-line`; ACL PRs merged first; image before policy; contract/erase/GA → `host-verify --cluster` green + snapshot; tag; **verify live by looking**; record milestone → tag → floor → snapshot and flag changes |
| `runner-vX.Y.Z` / evalpack `vX.Y.Z` | the stream's own workflow and tag; record it under **release streams** (digest, `validated_against`) |
| infra PR(s) only | each infra PR is its **own task**, merged on its own, **never folded into a tag**; `host-verify --cluster` after anything restart-inducing |
| no merge (spike, throwaway) | nothing from the spike is committed; only a results **docs PR** merges; teardown recorded in the spike record |
| PR, stop for owner review (design) | see below |
| outline only | M6c: nothing merges; the card is expanded at v2.2 planning |

**Tags are indicative.** Take the next free version at tag time; peers and v1 fixes may interleave. **Never move or
re-push a tag.** Before tagging, read status.md's pending-smoke notes and run the ones that apply.

### Design sprints: the owner-review gate

- A design sprint (`ds-*`) drafts its boards as static HTML on `design-system/theme.css` under
  `design-system/screens/v2/`, self-reviews them against the brief with screenshots, **opens one PR and STOPS**.
- **The agent never merges a design PR.** The owner's approval and merge (or an agent merge only on the owner's
  explicit approval in chat) **is the freeze** (`ev-freeze-ds-*` in status.md). Requested changes go on the same
  branch.
- **The design PR does not edit `status.md`** (it may stay open for days). In the PR the sprint file shows the
  drafting tasks ✅ and the PR task 🔄 "PR #N open, awaiting owner review".
- **Close-out after the owner's merge** is made by the **first consuming build sprint**, as four idempotent edits
  (skip any already done): the design sprint file's last task ✅ and _Overall_ ✅; its Sprint board row ✅; the
  boards' **Artboards** rows → "frozen (PR #N, date)"; the `ev-freeze-ds-*` owner event ✅.
- **A UI sprint whose boards aren't frozen is ⛔** (its entry gate). Boards contradicting an Accepted decision
  follow the decision and list the conflict under "Decisions to confirm" in the PR.
- Exception: [ds-m6a-01](sprint-ds-m6a-01.md) task 1 (ADR-0032 → Accepted) is its own docs PR, merged in-session
  under the BP2 pre-authorisation; it does update status.md.

### Spikes, outline cards, session end

- **Spikes (`spk-*`)** run on throwaway infrastructure (multipass/k3s VMs, scratch workspaces). Results land in the
  research appendix + the [spike record](../status.md#spike-record) via one docs PR; the ADR a spike gates is
  accepted by the **consuming** sprint (BP2).
- **Outline cards (`m6c-*`)** stay "⬜ outline" until v2.2 planning replaces each with a full plan and a prompt.
- **Session end: land and sync** ([AGENT.md](../../../AGENT.md)): branch → conventional commit(s) with the
  attribution lines → PR in every repo touched → CI green → squash-merge → Flux deploys (for a tag) → verify live →
  `git checkout main && git pull` everywhere. **Exceptions:** design PRs (stop for review); the **GA PR**
  ([ga-01](sprint-ga-01.md)) merges only with the owner's explicit approval; spikes merge only their docs PR; an
  owner "hold" overrides.

## Adding/adjusting a sprint

- Keep the flat `sprint-<id>.md` + `prompt-<id>.md` pairing (one prompt = one session). Follow the shape of an
  existing plan and prompt.
- **Never renumber.** A new sprint takes the next free number in its milestone (e.g. `m3-16`) and gets a row in all
  four index docs: [`../build-plan.md`](../build-plan.md) (table + graph if it's on a key path), [`../status.md`](../status.md)
  (Sprint board), this index and [`../prompts/README.md`](../prompts/README.md).
- A new flag, owner event, board or hand-off gets its status.md row in the same PR. Keep the flag inventory at
  **≤ 6 live non-kill flags**.
- Plans are pre-scaffolded for all of v2; refine a sprint's detail as it approaches if the code has drifted from
  the plan. Where a plan and an ADR disagree, the ADR wins; fix the plan in the sprint's PR.

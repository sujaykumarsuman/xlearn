# v2 execution prompts

> **Which prompt next?** See [execution-order.md](../execution-order.md): all 89 prompts clubbed into 19 workstreams, with the
> run order week by week and what each prompt brings.

**One prompt per sprint, one sprint per session.** Each `prompt-<id>.md` is a self-contained brief: paste it into
a fresh coding session at the repo root and it executes that whole sprint. It lists the owner-only prerequisites
to do **before launch**, names the docs to read first, the context, the **entry gates to verify first**, the
ordered work (each step tagged with its repo), constraints, deliverables, an **Update status** step, the "done
when" acceptance check, and the uniform **`## Ship`** ending. A few prompts plan a **re-run** when an owner-only step can only follow the session's own first push (mi-07 always; mi-04, mi-12, m3-07, m3-15, p-01, m4-07 and m6b-04 only when a step is missing): relaunch the same prompt after the owner acts, and it picks up at the ⛔ tasks ([status protocol](../sprints/README.md#status-protocol-way-of-working)).

**Launching a prompt is the owner's approval for every change it makes (D40).** That covers merges and tags
(contract, erase and GA ones included), infra PRs, design-board freezes, ADR acceptances, owner content the agent
drafts, and the production operations the prompt specifies. No session stops to wait for the owner; the owner's
in-session "hold / don't ship" still overrides.

- Pair: `prompt-<id>.md` ⇄ [`../sprints/sprint-<id>.md`](../sprints/) (the plan it executes; the plan wins on detail).
- Sequencing, milestones, calendar and dependencies: [`../build-plan.md`](../build-plan.md).
- Every prompt updates progress per the
  [status protocol](../sprints/README.md#status-protocol-way-of-working) (the sprint's Status table +
  [`../status.md`](../status.md)) in its **Update status** step, and its `## Ship` section lands those edits.

**Prompt shape**

| Section | Holds |
|---------|-------|
| header | the plan link, milestone (and rollout step ids), prereqs; then the one-prompt-one-session blockquote |
| `## Before you launch (owner)` | the owner-only prerequisites, e.g. a Hostinger manual snapshot in hPanel (1-day retention) before a contract, erase or GA tag, provider console setup, a machine user or PAT, DNS at the registrar, S6 presence, pack content. **Launching attests they're done.** If one turns out missing, the session lands everything that doesn't depend on it and marks the gap ⛔ in status.md; it doesn't wait |
| Read first | repo conventions (CLAUDE.md / AGENT.md), the plan, the rollout sections, ADRs and research appendices it implements, the `../infra` files to read before editing |
| Context | why the sprint exists now; live facts to re-read, not trust |
| Entry gates: verify first | the plan's checkboxes. **Stop and report if a hard prerequisite is unmet** (another sprint, merged code, a calendar date): that's a gate failure, not a review |
| Do this (in order) | numbered steps, each tagged **[X]** xlearn · **[I]** `../infra` · **[H]** host · **[E]** `xlearn-evalpack` · **[O]** owner-only (only as a before-launch check, never a mid-run wait) |
| Constraints | service boundaries, goose + sqlc, GitOps only, D34, the memory-sum rule, consumers-before-producers, ACL-before-tag, parallel sessions |
| Deliverables · Update status · Done when | what lands where; the status edits; the acceptance checklist |
| `## Ship (land-and-sync — owner approval pre-granted)` | always the last section: the D40 line, then the numbered land-and-sync steps with this sprint's release action filled in (below) |

**How each kind ends: the same way (D40)**

Every prompt's last section is `## Ship (land-and-sync — owner approval pre-granted)` ([AGENT.md](../../../AGENT.md),
[status protocol](../sprints/README.md#status-protocol-way-of-working)). It replaces any older "Ship at session
end …" or "Shipping: …" paragraph, so a prompt never has two endings. It opens with this line:

> Launching this prompt is the owner's approval for every change it makes (D40); don't stop for review.

Then come the numbered steps: branch → conventional commit(s) with the attribution lines → push → a PR in every
repo touched (`../infra` first where the order requires it) → CI green (fix, then merge) → squash-merge (never
auto-merge) → the release action → status (the sprint file + `../status.md`) → `git checkout main && git pull` in
every repo touched. The release action is the only part that differs:
- **Tag sprints:** the ADR-0034 §6 checklist, push the tag, let Flux deploy, verify live by looking.
- **Merge-only and infra sprints:** nothing deploys from the merge; name the tag that ships it (infra PRs are
  their own PRs, never folded into a tag).
- **Design sprints (`ds-*`):** the board PR merges on CI green, and **the merge is the freeze**. The design sprint
  records its freeze in `status.md` itself; the owner may review afterwards.
- **Spikes (`spk-*`):** launching is the go-ahead; the code is throwaway and never committed; only the results
  docs PR lands, even when the result needs an owner decision (⛔ in status.md).
- **GA PR ([prompt-ga-01](prompt-ga-01.md)):** merges on CI green like any other merge-only sprint; no separate
  approval.
- **M6c:** no prompt yet; v2.2 planning writes `prompt-m6c-0N.md` when it expands each outline card.

## Index

Grouped by milestone; **Order** is the global recommended execution order ([build plan](../build-plan.md#sprint-table-recommended-order)).

| Order | Prompt | Executes | Milestone | Release |
|-------|--------|----------|-----------|---------|
| 1 | [prompt-mi-01](prompt-mi-01.md) | [sprint-mi-01](../sprints/sprint-mi-01.md) — Prune guards + chart 0.3.0 knob union (MI-2, MI-3) | MI | infra PRs only (MI-2 first, then MI-3) |
| 3 | [prompt-mi-02](prompt-mi-02.md) | [sprint-mi-02](../sprints/sprint-mi-02.md) — host-verify --cluster extension (MI-8) | MI | infra PR only (host script) |
| 5 | [prompt-mi-07](prompt-mi-07.md) | [sprint-mi-07](../sprints/sprint-mi-07.md) — Evalpack plumbing (MI-9) | MI | infra PRs only (+ evalpack repo; `v0.1.0` image, no `>=1.0.0`) |
| 10 | [prompt-mi-05](prompt-mi-05.md) | [sprint-mi-05](../sprints/sprint-mi-05.md) — N0: NATS topology, dead letters, identity on NATS, client options, pool pins (MI-6) | MI | merge only (ships dark in v1.6.0) |
| 12 | [prompt-mi-14](prompt-mi-14.md) | [sprint-mi-14](../sprints/sprint-mi-14.md) — sandbox-guards: empty default-deny xlearn-runner namespace + VAP (MI-4) | MI | infra PRs only |
| 13 | [prompt-mi-03](prompt-mi-03.md) | [sprint-mi-03](../sprints/sprint-mi-03.md) — Fences: databases/messaging ingress, xlearn ingress (MI-5, MI-5a) | MI | infra PRs only (MI-5 → MI-5a) |
| 15 | [prompt-mi-04](prompt-mi-04.md) | [sprint-mi-04](../sprints/sprint-mi-04.md) — Admin consoles to ops.sujaykumar.dev (MI-5b) | MI | infra PRs only |
| 16 | [prompt-spk-01](prompt-spk-01.md) | [sprint-spk-01](../sprints/sprint-spk-01.md) — Sandbox mechanism spike P0–P2 (multipass arm64, throwaway) | MI | no merge (spike, throwaway); results docs PR |
| 17 | [prompt-spk-02](prompt-spk-02.md) | [sprint-spk-02](../sprints/sprint-spk-02.md) — P3 amd64 replay + image-volume spike (throwaway) | MI | no merge (spike, throwaway); results docs PR |
| 18 | [prompt-spk-03](prompt-spk-03.md) | [sprint-spk-03](../sprints/sprint-spk-03.md) — WIF spike (≤ ½ day, throwaway) | MI | no merge (spike, throwaway); results docs PR |
| 19 | [prompt-mi-06](prompt-mi-06.md) | [sprint-mi-06](../sprints/sprint-mi-06.md) — NATS auth server-first: N1 → N2 → N3 (MI-7) | MI | infra PRs only (N1, N2, N3) |
| 25 | [prompt-mi-08](prompt-mi-08.md) | [sprint-mi-08](../sprints/sprint-mi-08.md) — Limit hygiene (MI-11a) + Track B slice 1: PSA labels, SA tokens off (MI-15 part) | MI | infra PRs only (by the 10-24 window) |
| 26 | [prompt-mi-09](prompt-mi-09.md) | [sprint-mi-09](../sprints/sprint-mi-09.md) — October host-window prep: sandbox block, L23 kubelet args, pid limits, bumps (MI-11) | MI | infra PRs only (scripts applied in the window) |
| 41 | [prompt-mi-10](prompt-mi-10.md) | [sprint-mi-10](../sprints/sprint-mi-10.md) — Runner dark on production (MI-12) | MI | infra PRs only (runner dark) |
| 52 | [prompt-mi-11](prompt-mi-11.md) | [sprint-mi-11](../sprints/sprint-mi-11.md) — Track B finish: xlearn egress, PG connection limits, Renovate, N4 (MI-15) | MI | infra PRs only (+ xlearn PR, no tag) |
| 56 | [prompt-mi-12](prompt-mi-12.md) | [sprint-mi-12](../sprints/sprint-mi-12.md) — M4 gates: judge 443 egress, LLM secret, provider runbook (MI-14) + ADR-0031 → Accepted | MI | infra PRs only (+ ADR-0031 docs PR) |
| 81 | [prompt-mi-13](prompt-mi-13.md) | [sprint-mi-13](../sprints/sprint-mi-13.md) — M6b gates: Permissions-Policy, camera/mic deny, coach sizing, SDP route, WSS egress (MI-16) → v2.0.x patch | MI | **tag v2.0.x patch** (MI-16) + infra PRs first |
| 2 | [prompt-ds-m1-01](prompt-ds-m1-01.md) | [sprint-ds-m1-01](../sprints/sprint-ds-m1-01.md) — Design M1: coach states, course nav, revision v2 (AB01–AB03) | M1 | land-and-sync; the merge is the design freeze |
| 4 | [prompt-m1-01](prompt-m1-01.md) | [sprint-m1-01](../sprints/sprint-m1-01.md) — Curriculum spine part 1: compose parity, course manifest + golden, item schema freeze (M1a) | M1 | merge only (ships in v1.6.0) |
| 8 | [prompt-m1-09](prompt-m1-09.md) | [sprint-m1-09](../sprints/sprint-m1-09.md) — Curriculum spine part 2: converter, loader + guards, curriculum expand migration, content CI (M1a) | M1 | merge only (ships in v1.6.0) |
| 11 | [prompt-m1-02](prompt-m1-02.md) | [sprint-m1-02](../sprints/sprint-m1-02.md) — path_slug everywhere + v2 envelope consumers + identity M1a columns → v1.6.0 | M1 | **tag v1.6.0** (+ infra PR before) |
| 20 | [prompt-m1-03](prompt-m1-03.md) | [sprint-m1-03](../sprints/sprint-m1-03.md) — Course resolution: gateway, BFF, SPA /:course/*, producers emit v2 (M1b) | M1 | merge only (ships in v1.7.0) |
| 21 | [prompt-m1-04](prompt-m1-04.md) | [sprint-m1-04](../sprints/sprint-m1-04.md) — Identity security floor: roles, sessions, admin CLI, DEV_AUTH guard, L3, CSP (M1b) | M1 | merge only (ships in v1.7.0) |
| 22 | [prompt-m1-10](prompt-m1-10.md) | [sprint-m1-10](../sprints/sprint-m1-10.md) — Coach keys + provider hygiene: key_default(feature), catalog, AEAD AD, keyring re-wrap, store:false, classifier, usage | M1 | merge only (ships in v1.7.0) |
| 23 | [prompt-m1-05](prompt-m1-05.md) | [sprint-m1-05](../sprints/sprint-m1-05.md) — Gateway limits + public-dashboard floor (L1, L2, L4–L6, L24; P1, P2, P4, P10, P11) | M1 | merge only (ships in v1.7.0) |
| 24 | [prompt-m1-06](prompt-m1-06.md) | [sprint-m1-06](../sprints/sprint-m1-06.md) — withhold() on every surface + Markdown renderer + revision v2 (AB03) | M1 | merge only (ships in v1.7.0) |
| 27 | [prompt-m1-07](prompt-m1-07.md) | [sprint-m1-07](../sprints/sprint-m1-07.md) — Coach D27 assist capture, mode gate, L18 caps + AB01 → v1.7.0 | M1 | **tag v1.7.0** |
| 29 | [prompt-m1-08](prompt-m1-08.md) | [sprint-m1-08](../sprints/sprint-m1-08.md) — M1c contract → v1.8.0 | M1 | **tag v1.8.0** (contract; snapshot first) |
| 6 | [prompt-ds-m2-01](prompt-ds-m2-01.md) | [sprint-ds-m2-01](../sprints/sprint-ds-m2-01.md) — Design M2: Touch ★, catalog/agenda, public profile v2, visibility (AB04 ★, AB05, AB06, AB22) | M2 | land-and-sync; the merge is the design freeze |
| 33 | [prompt-m2-01](prompt-m2-01.md) | [sprint-m2-01](../sprints/sprint-m2-01.md) — M2a touch-attempt engine + `touch_concluded` consumer | M2 | merge only (ships in v1.9.0) + own infra PR |
| 34 | [prompt-m2-02](prompt-m2-02.md) | [sprint-m2-02](../sprints/sprint-m2-02.md) — M2b consumers + projections v2 → v1.9.0 | M2 | **tag v1.9.0** (+ ACL PR before) |
| 36 | [prompt-m2-03](prompt-m2-03.md) | [sprint-m2-03](../sprints/sprint-m2-03.md) — `public-read`, `/public/stats`, visibility toggles, profile v2 (P3, P5, P6, P7, P9; AB06, AB22) | M2 | merge only (ships in v1.10.0) |
| 37 | [prompt-m2-04](prompt-m2-04.md) | [sprint-m2-04](../sprints/sprint-m2-04.md) — Touch UI (AB04★) + catalog/agenda + Today in minutes (D4) (AB05) | M2 | merge only (ships in v1.10.0) |
| 38 | [prompt-m2-05](prompt-m2-05.md) | [sprint-m2-05](../sprints/sprint-m2-05.md) — Producers on + `touch_scored` backfill + reader switch + replay + D2 (M2c) → v1.10.0 | M2 | **tag v1.10.0** |
| 7 | [prompt-ds-l-01](prompt-ds-l-01.md) | [sprint-ds-l-01](../sprints/sprint-ds-l-01.md) — Design L: invite acceptance ★, privacy notice, erase account (AB19 ★, AB20, AB21) | L | land-and-sync; the merge is the design freeze |
| 39 | [prompt-l-01](prompt-l-01.md) | [sprint-l-01](../sprints/sprint-l-01.md) — L-E erase consumers (practice, review, assessment, coach) + identity ack path → v1.11.0 | L | **tag v1.11.0** (+ ACL/seed PRs before) |
| 40 | [prompt-l-02](prompt-l-02.md) | [sprint-l-02](../sprints/sprint-l-02.md) — L-E erase producer + `DELETE /api/me` + UI (AB21) → v1.12.0 | L | **tag v1.12.0** (erase; snapshot first) |
| 47 | [prompt-l-03](prompt-l-03.md) | [sprint-l-03](../sprints/sprint-l-03.md) — L-A admission backend (inert while closed): invites, seats, redeem, public checks, `SIGNUP_MODE=invite`, erase clears invite notes | L | merge only (ships dark, inert, in the next tag) |
| 67 | [prompt-l-05](prompt-l-05.md) | [sprint-l-05](../sprints/sprint-l-05.md) — L-A/L-C front door: auth-page invite state, acceptance step★, privacy notice, AI consents (AB19★, AB20) | L | merge only (ships in v1.17.0) |
| 68 | [prompt-l-04](prompt-l-04.md) | [sprint-l-04](../sprints/sprint-l-04.md) — Web erase for every non-owner → v1.17.0 (L-A/L-C) + L-exit rehearsal prep | L | **tag v1.17.0** (erase-class; + rehearsal PRs) |
| 9 | [prompt-m3-01](prompt-m3-01.md) | [sprint-m3-01](../sprints/sprint-m3-01.md) — Authoring tooling T25/T26: canonical hashes, contract_hash, packlint, pre-push fingerprint hook | M3 | merge only (dev tools; rides the next tag) |
| 14 | [prompt-m3-02](prompt-m3-02.md) | [sprint-m3-02](../sprints/sprint-m3-02.md) — Evalpack pipeline: private CI gates, generic validator, image build (E) + compose fixture pack | M3 | merge only (evalpack `main`; optional `v0.2.0`) |
| 28 | [prompt-ds-m3-01](prompt-ds-m3-01.md) | [sprint-ds-m3-01](../sprints/sprint-ds-m3-01.md) — Design M3 part 1: Workspace-Code ★, results dock, degradation badges (AB07 ★, AB08, AB11) | M3 | land-and-sync; the merge is the design freeze |
| 30 | [prompt-m3-03](prompt-m3-03.md) | [sprint-m3-03](../sprints/sprint-m3-03.md) — Runner core: supervisor, jail, cgroups, API | M3 | merge only (ships in runner-v1.0.0) |
| 31 | [prompt-m3-04](prompt-m3-04.md) | [sprint-m3-04](../sprints/sprint-m3-04.md) — Runner profiles Go/C++/Python + harness codecs + amd64 allowlists | M3 | merge only (ships in runner-v1.0.0) |
| 32 | [prompt-m3-15](prompt-m3-15.md) | [sprint-m3-15](../sprints/sprint-m3-15.md) — Runner release: reproducible image, runner-release.yml, acceptance suite, TL baselines → runner-v1.0.0 | M3 | **runner-v1.0.0** |
| 35 | [prompt-ds-m3-02](prompt-ds-m3-02.md) | [sprint-ds-m3-02](../sprints/sprint-ds-m3-02.md) — Design M3 part 2: Problems, Arena, Week/Mistakes/Progress deltas (AB09, AB10, AB12) | M3 | land-and-sync; the merge is the design freeze |
| 42 | [prompt-m3-05](prompt-m3-05.md) | [sprint-m3-05](../sprints/sprint-m3-05.md) — judge service: skeleton, schema, evalpack loader, erase consumer (M3-1) | M3 | merge only (ships dark in v1.13.0) |
| 43 | [prompt-m3-06](prompt-m3-06.md) | [sprint-m3-06](../sprints/sprint-m3-06.md) — judge queue, runner lane, graders, contexts, internal context endpoints | M3 | merge only (ships dark in v1.13.0) |
| 44 | [prompt-m3-14](prompt-m3-14.md) | [sprint-m3-14](../sprints/sprint-m3-14.md) — judge admission (L9–L15, L6), learner API + DTO allowlist, drafts, arena history/progress, telemetry, judge admin | M3 | merge only (ships dark in v1.13.0) |
| 46 | [prompt-m3-07](prompt-m3-07.md) | [sprint-m3-07](../sprints/sprint-m3-07.md) — M3-1 on prod: evalpack v1.0.0, judge ACL, tag v1.13.0, judge HelmRelease (MI-13) | M3 | **evalpack v1.0.0 + tag v1.13.0** (ACL PR before, HelmRelease after) |
| 48 | [prompt-m3-08](prompt-m3-08.md) | [sprint-m3-08](../sprints/sprint-m3-08.md) — practice: judge consumer, reconciler, D15/D16/D18 grading strategy | M3 | merge only (ships in v1.14.0) + ACL PR |
| 49 | [prompt-m3-09](prompt-m3-09.md) | [sprint-m3-09](../sprints/sprint-m3-09.md) — gateway judge BFF, DTO allow/deny lists, typed 413, L16, degradation status | M3 | merge only (ships in v1.14.0) |
| 50 | [prompt-m3-10](prompt-m3-10.md) | [sprint-m3-10](../sprints/sprint-m3-10.md) — review + assessment on judge signals (mistake pre-fill, P7 checked, P8) | M3 | merge only (ships in v1.14.0) |
| 53 | [prompt-m3-11](prompt-m3-11.md) | [sprint-m3-11](../sprints/sprint-m3-11.md) — Workspace-Code UI (AB07★) + CodeMirror lazy load | M3 | merge only (ships in v1.14.0) |
| 54 | [prompt-m3-12](prompt-m3-12.md) | [sprint-m3-12](../sprints/sprint-m3-12.md) — Results dock, Problems, Arena, degradation badges (AB08–AB11) | M3 | merge only (ships in v1.14.0) |
| 55 | [prompt-m3-13](prompt-m3-13.md) | [sprint-m3-13](../sprints/sprint-m3-13.md) — Week/Mistakes/Progress deltas (AB12) + M3 exit tests → v1.14.0 | M3 | **tag v1.14.0** (+ `JUDGE_BASE_URL` PR after) |
| 45 | [prompt-ds-p-01](prompt-ds-p-01.md) | [sprint-ds-p-01](../sprints/sprint-ds-p-01.md) — Design P: Workspace-Quiz, go-concurrency multi-file + race, AB02/AB05 full fidelity (AB14, AB15) | P | land-and-sync; the merge is the design freeze |
| 57 | [prompt-p-01](prompt-p-01.md) | [sprint-p-01](../sprints/sprint-p-01.md) — go-race runner profile → runner-v1.1.0 | P | **runner-v1.1.0** (judge code rides v1.15.0) |
| 58 | [prompt-p-02](prompt-p-02.md) | [sprint-p-02](../sprints/sprint-p-02.md) — Multi-course: go-concurrency manifest (preview), catalog, agenda, nav | P | merge only (ships in v1.15.0) |
| 59 | [prompt-p-03](prompt-p-03.md) | [sprint-p-03](../sprints/sprint-p-03.md) — Quiz widget (AB14) + race verdict UI (AB15) → v1.15.0 | P | **tag v1.15.0** + **evalpack v1.N.0** |
| 51 | [prompt-ds-m4-01](prompt-ds-m4-01.md) | [sprint-ds-m4-01](../sprints/sprint-ds-m4-01.md) — Design M4: AI suggestion/dispute ★, pointer notes, allowance + consents (AB16 ★, AB17, AB18) | M4 | land-and-sync; the merge is the design freeze |
| 60 | [prompt-m4-01](prompt-m4-01.md) | [sprint-m4-01](../sprints/sprint-m4-01.md) — platform/llm extraction + WIF auth + retention policy | M4 | merge only (ships in v1.16.0) |
| 61 | [prompt-m4-02](prompt-m4-02.md) | [sprint-m4-02](../sprints/sprint-m4-02.md) — judge ai: Scorer, ledger, llm lane, breaker, caps (L17) | M4 | merge only (ships in v1.16.0) |
| 62 | [prompt-m4-03](prompt-m4-03.md) | [sprint-m4-03](../sprints/sprint-m4-03.md) — Analyzer (D16/D26), pointer notes, `evaluation_analyzed` + acceptance-set harness | M4 | merge only (ships in v1.16.0) + ACL PR |
| 63 | [prompt-m4-04](prompt-m4-04.md) | [sprint-m4-04](../sprints/sprint-m4-04.md) — Provisional grades, dispute, re-grade, honor claims (D14) | M4 | merge only (ships in v1.16.0) |
| 64 | [prompt-m4-05](prompt-m4-05.md) | [sprint-m4-05](../sprints/sprint-m4-05.md) — AI allowance + consents (`account_consent` AI kinds) | M4 | merge only (ships in v1.16.0) |
| 65 | [prompt-m4-06](prompt-m4-06.md) | [sprint-m4-06](../sprints/sprint-m4-06.md) — M4 UI: AI suggestion/dispute (AB16★), pointer notes (AB17), allowance + consents (AB18) | M4 | merge only (ships in v1.16.0) |
| 66 | [prompt-m4-07](prompt-m4-07.md) | [sprint-m4-07](../sprints/sprint-m4-07.md) — Canary log test, caps sizing, ledger check → v1.16.0 (+ LLM_PLATFORM_ENABLED for the cohort) | M4 | **tag v1.16.0** (+ `LLM_PLATFORM_ENABLED` PR) |
| 69 | [prompt-ga-01](prompt-ga-01.md) | [sprint-ga-01](../sprints/sprint-ga-01.md) — GA PR: .release-line = 2, T-1 default flips, rc rehearsal | GA | merge only: the GA PR merges on CI green (+ optional `v2.0.0-rc.N`) |
| 70 | [prompt-ga-02](prompt-ga-02.md) | [sprint-ga-02](../sprints/sprint-ga-02.md) — Cut v2.0.0: pre-flip check, widen ranges, snapshot, tag, verify | GA | **tag v2.0.0** (widen first; GA snapshot) |
| 71 | [prompt-spk-04](prompt-spk-04.md) | [sprint-spk-04](../sprints/sprint-spk-04.md) — S6 voice-shell bake-off (owner present, throwaway) | M6a | no merge (spike, throwaway); results docs PR |
| 72 | [prompt-ds-m6a-01](prompt-ds-m6a-01.md) | [sprint-ds-m6a-01](../sprints/sprint-ds-m6a-01.md) — Design M6a part 1 + ADR-0032 → Accepted: Mock-v2, setup/consent/pre-flight, live HUD text (AB13, AB24, AB25) | M6a | ADR-0032 docs PR, then land-and-sync; the merge is the design freeze |
| 73 | [prompt-ds-m6a-02](prompt-ds-m6a-02.md) | [sprint-ds-m6a-02](../sprints/sprint-ds-m6a-02.md) — Design M6a part 2: grace/paused/resume, debrief + proposal, accessibility (AB26, AB27, AB28) | M6a | land-and-sync; the merge is the design freeze |
| 74 | [prompt-m6a-01](prompt-m6a-01.md) | [sprint-m6a-01](../sprints/sprint-m6a-01.md) — Interview core: schema, state machine, failsafes, caps (L19) | M6a | merge only (ships dark in a v2.0.x patch) |
| 75 | [prompt-m6a-02](prompt-m6a-02.md) | [sprint-m6a-02](../sprints/sprint-m6a-02.md) — Text brain: loop, classifier, give_hint, resume brief, store:false, replay tests | M6a | merge only (ships dark in a v2.0.x patch) |
| 76 | [prompt-m6a-03](prompt-m6a-03.md) | [sprint-m6a-03](../sprints/sprint-m6a-03.md) — Assessment deltas + proposal/accept + ScoreMock once + twin fairness gate | M6a | merge only (ships dark in a v2.0.x patch) |
| 77 | [prompt-m6a-04](prompt-m6a-04.md) | [sprint-m6a-04](../sprints/sprint-m6a-04.md) — judge mock budgeting + CodeMirror interview mode + bounded SSE | M6a | merge only (ships dark in a v2.0.x patch) |
| 78 | [prompt-m6a-05](prompt-m6a-05.md) | [sprint-m6a-05](../sprints/sprint-m6a-05.md) — M6a UI part 1: Mock-v2 (AB13), setup/consent/pre-flight/$ cap (AB24), live HUD text (AB25) | M6a | merge only (ships dark in a v2.0.x patch) |
| 79 | [prompt-m6a-06](prompt-m6a-06.md) | [sprint-m6a-06](../sprints/sprint-m6a-06.md) — M6a UI part 2: grace/paused/resume (AB26), debrief/proposal (AB27), accessibility (AB28) → v2.0.x patch | M6a | **tag v2.0.x patch** (M6a dark, cohort) |
| 80 | [prompt-ds-m6b-01](prompt-ds-m6b-01.md) | [sprint-ds-m6b-01](../sprints/sprint-ds-m6b-01.md) — Design M6b: voice pre-flight/notices, voice live HUD ★ (AB29, AB30 ★) | M6b | land-and-sync; the merge is the design freeze |
| 82 | [prompt-m6b-01](prompt-m6b-01.md) | [sprint-m6b-01](../sprints/sprint-m6b-01.md) — VoiceShell adapter + SDP broker + sideband | M6b | merge only (ships dark in m6b-03's patch) |
| 83 | [prompt-m6b-02](prompt-m6b-02.md) | [sprint-m6b-02](../sprints/sprint-m6b-02.md) — Voice robustness: lease, re-attach, drain, cost + $ cap, PTT, rollover (+ coach-interview if M7 failed) | M6b | merge only (ships dark in m6b-03's patch) |
| 84 | [prompt-m6b-03](prompt-m6b-03.md) | [sprint-m6b-03](../sprints/sprint-m6b-03.md) — Voice UI: pre-flight/notices (AB29), live HUD★ (AB30), consent, EU gate, self-view → v2.0.x patch (M6b dark) | M6b | **tag v2.0.x patch** (M6b dark, cohort) |
| 85 | [prompt-m6b-04](prompt-m6b-04.md) | [sprint-m6b-04](../sprints/sprint-m6b-04.md) — Fake-media e2e + interviewer GA flip → v2.1.0 | M6b | **tag v2.1.0** (interviewer GA) |
| 86 | [prompt-m5-01](prompt-m5-01.md) | [sprint-m5-01](../sprints/sprint-m5-01.md) — DSA evaluator-only flip (M5) | M5 | rides v2.1.0 if merged before m6b-04, else **tag the next minor ≥ v2.2.0** |
| 87 | — (none: outline card) | [sprint-m6c-01](../sprints/sprint-m6c-01.md) — Outline: "show your work" photo + chat read-aloud + local recording download | M6c | outline only (v2.2) |
| 88 | — (none: outline card) | [sprint-m6c-02](../sprints/sprint-m6c-02.md) — Outline: multi-speaker fairness check → AI-proposed voice Communication; history-calibrated estimates | M6c | outline only (v2.2) |
| 89 | — (none: outline card) | [sprint-m6c-03](../sprints/sprint-m6c-03.md) — Outline: Safari leg, voice-lite on demand, weekly canary; AB31 canvas outline (SD course, v2.2+) | M6c | outline only (v2.2) |

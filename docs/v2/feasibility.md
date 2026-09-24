# xLearn v2 — research, feasibility & design

> **Status:** all topics settled (T0–T7, 2026-09-24) · **next:** the build-plan session, using [rollout-plan.md](rollout-plan.md) · **Started:** 2026-09-23 · **Owner:** @sujaykumarsuman
> **Scope:** research + feasibility + design only — **no build, no deploy, no `../infra` change**.
> This doc grows one section per settled topic; a later session turns it into the v2 build plan,
> sprints and prompts (mirroring [`../v1/`](../v1/)). ADR drafts for load-bearing calls live in
> [`../adr/`](../adr/) (**0026+**, since 0025 is the v1.5.0 profile-URL ADR; `Proposed` until signed off).
> Detailed per-topic research lives in [`research/`](research/). Appendices cite `scratchpad/…` files (models, critiques, source copies): those are the planning session's uncommitted working files and are not in the repo.

## Decisions log (newest first)

| # | Date | Topic | Decision | ADR |
|---|------|-------|----------|-----|
| D35 | 2026-09-24 | T7 | **v2 audience: owner-only use; real learners not before v3.** The invite flow and the whole learner gate **L** (admission, acceptance, privacy notice, 18+, region, erase, the `tester` role) **stay in v2 scope**, ship, and are exercised by the owner and testers. An invite round-trip is rehearsed on production with a tester, then production returns to `SIGNUP_MODE=closed`. Inviting real users ("the opening") is **the owner's choice, planned for v3**, not a technical limit of v2. **Opening gates** (for v3): MI-5b live; alerting revisited (D34); `SEAT_CAP` re-sized from ≥ 2 weeks of M4 data; privacy notice and erase live; `SIGNUP_MODE=invite`; open signup (D21) or `SEAT_CAP` > 40 (a T7 threshold) requires R2. **v2.0 GA = MI + M1–M4 + pilot + L, complete for the owner.** D6's content becomes the scope of the **v2.x line**, delivered in waves; `v2.0.0` doesn't wait for it. | [0033](../adr/0033-invite-only-admission-and-owner-admin.md) · [0034](../adr/0034-v2-release-labelling-gating-and-rollback.md) |
| D34 | 2026-09-24 | T7 | **No alerting in v2.** No push channel, healthchecks.io, host-check timer, opscheck CronJobs, Flux Alerts, or metrics or logs stack. The owner monitors with **landscape and kubescope**, plus `host-verify --cluster` after host changes. **Accepted risk:** a failure (node down, dead-lettered events, an eval-pack exposure, PAT expiry, a spend anomaly) is seen only when the owner looks. **What remains (not alerts):** CI checks (the evalpack anonymous-GET probe, the NATS ACL golden file, the stream-budget and subject-registry tests), dead-letter rows in Postgres, in-app degradation badges, judge's spend caps, and the provider consoles' own limits. **Recommended:** extend `host-verify --cluster` (on demand) with cheap cluster reads. **Revisit before the first real invite**; the healthchecks.io design is kept as the ready option. Supersedes T1's opscheck + push channel, T5's push channel, T6's opscheck interview counters and ADR-0027's PAT-expiry alert. | [0035](../adr/0035-v2-operations-nats-auth-limits-capacity.md) |
| D33 | 2026-09-24 | T7 | **Admission and admin.** **Close v1 signup now,** as a v1.5.x stopgap in two parts: GitHub stops auto-linking into any account that has a password (the pre-account-hijack fix; amends ADR-0023 §3; merged 2026-09-24 as xlearn#51), and `SIGNUP_MODE` (`closed` by default in production) is added (MI-2c). **Done:** both live in v1.5.2 (xlearn#51, #52; infra#30 sets `closed`); signup closed in production since 2026-09-24. v1.5.2 honours `open` without `DEV_AUTH`; that guard is M1b. **v2, owned by identity:** `SIGNUP_MODE ∈ {closed, invite, open}` (`open` = local/dev only); **owner-minted single-use invite links** (128-bit code stored as sha256, 7-day TTL with a 30-day max, carried in the URL fragment, redeemed inside the account-create transaction on both the email and GitHub paths, a uniform `invite_invalid`, the invite checked before the email); **`SEAT_CAP`** 15; an **acceptance step** (18+, privacy-notice version, region; M4's two unticked AI consents); `account.role ∈ {learner, tester, owner}` + `status` in the DB, **never in the JWT**; a CLI-minted **`tester`** role (outside `SEAT_CAP`, erasable on the web); an **`identity admin` CLI** via `kubectl exec`, every verb written to `admin_audit`; a `mailto:` "request an invite". **No web admin page and no waitlist form.** kubescope, landscape and the Longhorn UI move to `ops.sujaykumar.dev` (MI-5b) **before the first non-owner account**. | [0033](../adr/0033-invite-only-admission-and-owner-admin.md) |
| D32 | 2026-09-24 | T7 | **Release labels.** Every v2 milestone ships as a **1.x minor** (the next free minor at tag time). **`v2.0.0` = v2.0 GA:** it flips the v2 defaults (judge and platform AI on, pilot course `active`) for the owner once MI + M1–M4 + pilot + L are complete. **`v2.1.0` = interviewer GA** (M6a + M6b, as in D28). M5 is a later 2.x minor. The HTTP API stays `/api/v1`. **Guard now** (MI-2a): every `xlearn-*` ImagePolicy bounded to `>=1.0.0 <2.0.0`, and `deploy.yml` fails a tag whose major ≠ `.release-line` (`1` until GA). **Done 2026-09-24:** infra#29 + xlearn#53. **At GA:** `.release-line`=2, and the infra range `>=1.0.0 <3.0.0` merged **before** the tag (optionally `v2.0.0-rc.N` first; prereleases never auto-deploy). Runner and evalpack streams use `>=1.0.0 <2.0.0`; there a major is a contract break. | [0034](../adr/0034-v2-release-labelling-gating-and-rollback.md) |
| D31 | 2026-09-24 | T6 | **Public profile: mock count only.** No mock best or average publicly, whether AI-proposed or self-scored. Amends ADR-0024. | [0032](../adr/0032-realtime-ai-mock-interviewer.md) |
| D30 | 2026-09-24 | T6 | **A paused interview not resumed within 24 h becomes `incomplete`**: unscored and left out of trends. There is a free partial view; paid improvement areas only on a click. | [0032](../adr/0032-realtime-ai-mock-interviewer.md) |
| D29 | 2026-09-24 | T6 | **The AI shares the interview like a second interviewer.** It holds a live two-way voice conversation and follows the **on-screen answer widgets** as structured state (editor, canvas, text, choices). Updates go **on change every 2–3 s and on every candidate turn**, and the AI steers the interview from them. The final analysis uses the whole transcript plus the snapshot timeline. **No video goes to the AI.** **Future: peer-to-peer interviews with video**; the TURN decision is deferred and nothing is built in v2. | [0032](../adr/0032-realtime-ai-mock-interviewer.md) |
| D28 | 2026-09-24 | T6 | **The interviewer is milestone M6 (v2.1), not v2.0.** P0 is a text interviewer plus all the failsafes. P1 is voice (browser↔OpenAI WebRTC, SDP brokered by coach, no media on the node). **Spike S6 is approved** (≤ 1 day, owner's key, $10 hard limit). It needs the owner present, so it is scheduled before the M6a design freeze. | [0032](../adr/0032-realtime-ai-mock-interviewer.md) |
| D27 | 2026-09-24 | T5 | **Coach-assist cap is per problem (refines D18).** Only a coach chat about **that problem** during its open counted attempt needs a confirmation and caps the grade at **Assisted**. Asking from another page is an accepted, honor-based bypass. There is no mock or realtime lock during attempts. The coach stays locked during live touches and mocks. | [0031](../adr/0031-platform-ai-and-two-tier-keys.md) |
| D26 | 2026-09-24 | T5 | **Pass-review scope (refines D16):** **course passes**, plus a **revision-touch pass whose solution is materially different** from the last reviewed one (normalized-code fingerprint). If it is correct but improvable, the revision item is marked **"correct, with improvements"** and gets pointer notes. No effect on pass/fail, grade or ladder; advisory tier (shed first). | [0031](../adr/0031-platform-ai-and-two-tier-keys.md) |
| D25 | 2026-09-24 | T5 | **Platform-AI ceiling: $100/month** as the provider workspace hard limit, with the app cap at **$80**. Per account: $6/month and $1/day to start. Dogfood defaults: $15 / $12. | [0031](../adr/0031-platform-ai-and-two-tier-keys.md) |
| D24 | 2026-09-24 | T5 | **Retention:** accept `RetentionPolicy` for pack-bearing Score. That means provider retention ≤ 30 days, no training, Messages API only (no Batch, Files, tools or Covered Models), disclosed, and ZDR requested opportunistically. Also recorded here: **judge owns platform AI** with Anthropic `claude-sonnet-5` only (Opus 5.5 for escalation and re-grade), a **WIF** credential (spike before M4), and **no BYO for platform tasks**. | [0031](../adr/0031-platform-ai-and-two-tier-keys.md) |
| D23 | 2026-09-24 | T3 | **Spike:** only **S0 (read-only prod facts)** was approved and run. The mechanism spike **P0–P3 is deferred to a build-plan gate before M3**; R1's go/no-go questions stay open until then, with the nsjail fallback. A 72 h steal sample is running until ~2026-09-27. | [0030](../adr/0030-runner-technology-and-host-hardening.md) |
| D22 | 2026-09-24 | T3 | **Patch cadence:** unattended security upgrades (with the **kernel meta-packages unheld**), a monthly reboot window, monthly k3s patch releases pinned in git, Renovate for the runner pins. **No Livepatch** (owner, 2026-09-24): kernel fixes land at the monthly reboot. **H0** (patch the live kernel 6.8.0-90 → noble 6.8.0-142, fix the apport core pipe) is a separate, urgent task. | [0030](../adr/0030-runner-technology-and-host-hardening.md) |
| D21 | 2026-09-24 | T3 | **Untrusted code runs on the production node** (R1 plus H0 plus the D22 cadence) while signup is invite-only. It **moves to a dedicated runner VPS** (never the backup VPS) on any trigger: open signup, an unpatched reachable LPE older than 7 days, the spike forcing R1b, or chronic steal. | [0030](../adr/0030-runner-technology-and-host-hardening.md) |
| D20 | 2026-09-24 | T3 | **Launch languages: Go, C++ and Python at M3** (PRD Q6). That means three runner profiles, harness codecs, amd64 allowlists and per-language time-limit multipliers, and about 35–50 h more authoring for per-language references on the fully packed items. go-race and sql-pg come with the pilot. | [0030](../adr/0030-runner-technology-and-host-hardening.md) |
| D19 | 2026-09-24 | T4 | **Canvas = Excalidraw** (MIT, open source), chosen to minimise complexity. Typed shapes use `customData`; freehand is ignored by grading. The stored payload is the library-neutral `canvas-graph@1`, and the AI grades its **text export**. | [0029](../adr/0029-judge-contract-and-learning-signal.md) |
| D18 | 2026-09-24 | T4 | **DSA time budget:** a **45-minute** default per problem, with the **hint unlocking at 15 minutes**. **Clean:** pass ≤ 20 min, no hint, no coach, ≤ 3 failed submits. **Rough:** pass ≤ 45 min without hint or coach. **Assisted:** pass after the hint **or with AI-coach help during the attempt**. **Miss:** timeout or give-up. These are manifest defaults with per-item overrides. **Per-problem time budgets are a separate future research session.** | [0029](../adr/0029-judge-contract-and-learning-signal.md) |
| D17 | 2026-09-24 | T4 | **Arena is unrestricted**: no locks during live attempts or touches. An early solution reveal is recorded but **doesn't cap** the course grade (honor system). | [0029](../adr/0029-judge-contract-and-learning-signal.md) |
| D16 | 2026-09-24 | T4 | **A pass finishes the attempt**, and there is no mandatory re-implement. After a fail, the solution unlocks for study and the Day-1 revision is the from-memory re-attempt. **The AI assessment also reviews passing solutions**, and can attach **pointer notes to the problem** plus an **optional revisit** (no effect on grade or ladder). | [0029](../adr/0029-judge-contract-and-learning-signal.md) |
| D15 | 2026-09-24 | T4 | **Hard time limit.** The timer starts when the problem starts. **No pass by the limit concludes the attempt as Miss** at the deadline, and it is re-attempted via the schedule (Day 1). An explicit give-up is also a Miss. Touches follow the same rule; a touch that was never shown, or has only infra-inconclusive evidence, is voided. | [0029](../adr/0029-judge-contract-and-learning-signal.md) |
| D14 | 2026-09-24 | T4 | **AI-graded results are suggestions with reasoning** (grade, mistake, notes, concepts). The learner can **accept, edit, or request one blind re-grade**, and **submitting** triggers the downstream actions. **Manual entry is the fallback** whenever AI is unavailable. Edits are bounded by deterministic ceilings and labelled self-set. An untouched suggestion auto-submits after 24 h. | [0029](../adr/0029-judge-contract-and-learning-signal.md) |
| D13 | 2026-09-24 | T2 | **Controlled, invite-only signup** for opening xLearn to other learners, **for capacity reasons** (not backups). Mechanism is designed in T7 (authz / limits). | [0028](../adr/0028-object-storage-and-backups.md) |
| D12 | 2026-09-24 | T2 | **No off-node backups for now.** The future target is the owner's **second VPS**. The T2 appendix keeps the ready design (CNPG Barman Cloud Plugin, post-restore sequence, drills). **No backup gate** on opening to learners. The **erase ledger is deferred** with backups, so erase is delete plus tombstones (amends D8). Hostinger's weekly images stay on as the only safety net (2FA on the Hostinger login). | [0028](../adr/0028-object-storage-and-backups.md) |
| D11 | 2026-09-24 | T2 | **No object store in v2.0**, neither in the cluster nor external, for app data. Canvas scenes and drafts plus transcripts stay **inline in Postgres permanently**, and T1's `object_key` is deferred. **MinIO is rejected** (community server archived 2026-04-25, operator archived 2026-03-20). Audio is T6's call. | [0028](../adr/0028-object-storage-and-backups.md) |
| D10 | 2026-09-24 | T1 | **Problems arena:** arena **Submits are kept as a submission history** (Runs are not), with an **arena done marker separate from course done**. A course attempt starts its server timer when the attempt starts and shows the full question plus the coding area. The arena has a **manual timer, off by default**. Arena activity never counts toward course progress, grades or learning signals, and is learner-private in v2.0. | [0027](../adr/0027-content-evalpack-and-user-data-model.md) |
| D9 | 2026-09-24 | T1 | **Licence:** MIT for everything, content included. The eval pack is proprietary. Rights stance: original statements, LeetCode as outbound links only, `provenance` on every item. | [0027](../adr/0027-content-evalpack-and-user-data-model.md) |
| D8 | 2026-09-24 | T1 | **Account erase at v2.0:** delete rows and outbox rows plus tombstones in every service, with an **erase ledger** outside the backup that is replayed after any restore. Pseudonymous NATS events stay (re-seal runbook on request). Requires NATS auth first. | [0027](../adr/0027-content-evalpack-and-user-data-model.md) |
| D7 | 2026-09-24 | T1 | **Public-profile defaults:** the profile is public. Each course is visible by its manifest default (behavioral hidden). **Every header aggregate is computed from visible courses only.** | [0027](../adr/0027-content-evalpack-and-user-data-model.md) |
| D6 | 2026-09-24 | T1 | **v2.0 content scope:** all **151 DSA items at the self-path tier**, **full packs for weeks 1–4** (~35), and a ~10-item second-course pilot after M3. M5 waits until every live DSA item has a pack. About 180–285 h. | [0027](../adr/0027-content-evalpack-and-user-data-model.md) |
| D5 | 2026-09-24 | T1 | **Private eval pack:** a fresh private repo `xlearn-evalpack` → its CI builds a **private GHCR data image** → a **judge-only read-only image volume** (`EVALPACK_DIR`; fallbacks: initContainer, then ORAS). No pack bytes in Postgres. A narrow `contract_hash` handles skew between the two repos. | [0027](../adr/0027-content-evalpack-and-user-data-model.md) |
| D4 | 2026-09-24 | T0 | **Today and budget across courses:** one budget per account **in minutes** (`est_minutes` per format from each manifest). Due touches from **all** courses come first, R-SR5 blocks new work **per course**, and the remainder splits across active enrollments. | [0026](../adr/0026-per-course-extensibility-model.md) |
| D3 | 2026-09-24 | T0 | **Method universality:** the Day 1·3·7·21·45 ladder and the 4-grade scale (clean/rough/assisted/miss) are universal. Per course: the touch format per level band, timers, grading strategy (from a closed Go set) and thresholds, mistake categories (on top of a universal core), and the mock rubric, rail and targets. | [0026](../adr/0026-per-course-extensibility-model.md) |
| D2 | 2026-09-24 | T0 | **Revision entry:** every concluded, counted attempt of any grade (including miss or give-up) anchors L1–L5, and below-clean also opens a mistake. Universal, and it **amends PRD R-SR1**. New courses start on it. DSA switches in a later release (M2), after the M1 restructure ships with no behaviour change. This closes v1's stuck-mistake hole. | [0026](../adr/0026-per-course-extensibility-model.md) |
| D1 | 2026-09-24 | T0 | **Plug-in mechanism:** a settings-only **course manifest** (`curriculum/<slug>/course.json`), compiled into every image and validated in CI, plus **closed code registries keyed by part type or grader kind, never by course**. One new service, `judge`, produces evidence only. **practice is the single writer of learning signals** (`problem_solved` v2, `touch_concluded`). | [0026](../adr/0026-per-course-extensibility-model.md) |
| D0 | 2026-09-23 | Process | Topic order **T0 → T1 → T2 → T4 → T3 → T5 → T6 → T7** (the common judge contract is settled *before* the code sandbox). Scope additions: T1 + where private content lives (repo is public) and how the missing problems/tests get authored; T2 + off-node backups; T3 + cluster hardening prerequisites; T7 + release labelling/`v2.0.0` auto-deploy/missing v2 artboards. The public dashboard is **not** its own topic (mostly built in F009/ADR-0024): its data deltas go in T1, its tasks in T7. | — |

## Topic map

| Topic | Question | Status | ADR | Section |
|-------|----------|--------|-----|---------|
| T0 | Per-curriculum extensibility frame (seams: nav · content · solving ground · evaluator; what stays common) | ✅ settled 2026-09-24 | [0026](../adr/0026-per-course-extensibility-model.md) (Proposed) | [T0](#t0--guiding-frame) |
| T1 | Content & data model (authored vs per-user; Postgres / JSONB / blobs; private content; authoring; dashboard deltas) | ✅ settled 2026-09-24 | [0027](../adr/0027-content-evalpack-and-user-data-model.md) (Proposed) | [T1](#t1--content--data-model) |
| T2 | Object storage + off-node backups | ✅ settled 2026-09-24 (backups deferred) | [0028](../adr/0028-object-storage-and-backups.md) (Proposed) | [T2](#t2--object-storage--off-node-backups) |
| T4 | Judge types & the common submission → evaluation → learning-signal contract | ✅ settled 2026-09-24 | [0029](../adr/0029-judge-contract-and-learning-signal.md) (Proposed) | [T4](#t4--judge-types--the-common-contract) |
| T3 | Code-execution sandbox / online-judge engine (+ cluster hardening) | ✅ settled 2026-09-24 (spike pending) | [0030](../adr/0030-runner-technology-and-host-hardening.md) (Proposed) | [T3](#t3--code-execution-sandbox--cluster-hardening) |
| T5 | Platform AI + two-tier keys | ✅ settled 2026-09-24 (WIF spike + egress gate before M4) | [0031](../adr/0031-platform-ai-and-two-tier-keys.md) (Proposed) | [T5](#t5--platform-ai--two-tier-keys) |
| T6 | Realtime AI mock interviewer (voice + video) | ✅ settled 2026-09-24 (M6 / v2.1; spike S6 approved, pending) | [0032](../adr/0032-realtime-ai-mock-interviewer.md) (Proposed) | [T6](#t6--realtime-ai-mock-interviewer) |
| T7 | Cross-cutting + infra-first rollout + v2 milestone map | ✅ settled 2026-09-24 (v2 owner-only; real learners from v3) | [0033](../adr/0033-invite-only-admission-and-owner-admin.md) · [0034](../adr/0034-v2-release-labelling-gating-and-rollback.md) · [0035](../adr/0035-v2-operations-nats-auth-limits-capacity.md) (Proposed) | [T7](#t7--cross-cutting-and-infra-first-rollout) |

Legend: ⬜ not started · 🔄 in discussion · ✅ settled (ADR Proposed) · 🔒 signed off (ADR Accepted).

## Baseline — where v1 stands (orientation, 2026-09-23)

Verified by reading the code, `../infra`, and read-only inspection of the live node. v2 is designed
against these facts.

**Platform (live).**
- **Node:** 1 × k3s v1.36 node. 4 vCPU (EPYC, KVM guest), 15 GiB RAM, **no swap**, 176 GB of 193 GB free, Ubuntu 24.04, kernel 6.8, cgroup v2. About 0.3 vCPU and 4.5 GiB are in use.
- **No `/dev/kvm`** (no nested virtualisation), so Firecracker and Kata are impossible.
- **Ubuntu blocks unprivileged user namespaces** through AppArmor.
- **containerd has only `runc`.** k3s auto-created RuntimeClass objects for crun and the wasm runtimes, but those binaries don't exist.
- **k3s enforces NetworkPolicy**, but the `xlearn`, `databases` and `messaging` namespaces have **none**. The v1 plan's "S12" network hardening never landed.
- **NATS has no auth.**
- **Firewall (ufw):** only TCP 22, 80 and 443 are open; no UDP. Traefik serves `web`/`websecure` only, and the TLS cert covers `projects.sujaykumar.dev` via HTTP-01.
- **No backups anywhere.** CNPG `spec.backup` is unset, the Longhorn backup target is empty, and nothing is copied off the node.
- **Postgres (CNPG):** single instance, 10 Gi volume, 1 Gi memory limit, `max_connections` 100.
- **NATS JetStream:** 5 Gi volume. Streams are created by the app with no retention limits.

**Data (live, aggregate counts only).**
- 1 account, 19 events, and `xlearndb` is about 10.5 MB.
- **So breaking id or schema changes are cheap right now.**

**Repo exposure.**
- `sujaykumarsuman/xlearn` is **public**, and GHCR images are anonymously pullable.
- Content is embedded into the curriculum image, so **hidden test data cannot live in the repo seed or in an image.**

**What v2 inherits as reusable seams:**
- **Catalog and enrollment:** the path catalog (`curriculum.path`), `identity.path_enrollment`, and the slug-derived shell with its `navForPath` switch.
- **Review:** the five-touch engine and the mistake-journal mechanics.
- **Mocks:** the lifecycle (server timer, score-once).
- **Events:** outbox → JetStream → inbox-deduped durable consumers.
- **Coach:** the provider interface, envelope-encrypted keys, and the SSE relay.
- **Public profile:** the per-course loop.
- **Automation seams are server functions already.** Practice `LogOutcome` emits `problem_solved`, which review and assessment consume. Review `Score(ScoreInput)`, `PATCH /mistakes` and assessment `ScoreMock` are the others. A judge or AI that calls these leaves the scheduler, journal and projections unchanged.

**What blocks a second course:**
- **Global ids.** `curriculum.problem.id` and `concept.slug` are global, and the seed upserts **re-parent on conflict**.
- **`"dsa"` is pinned** in 8 gateway call sites and in the static SPA `dsa/*` routes.
- **The DSA method is DB `CHECK` enums:** stages, the 4 outcomes, the 8 mistake categories, and the 7-dimension `/35` rubric.
- **No `path_slug`** in events or in the review and assessment tables.
- **The coach prompt** is DSA/Go.

**What does not exist:**
- Code execution, a real editor (today: a textarea with a hard-coded 3Sum stub), and submission storage.
- Object storage.
- AI usage metering and rate limits.
- Metrics.
- v2 artboards.
- Content: 14 of 151 DSA problems are seeded, statements are 1–2 sentence summaries, and there are no test cases.

**Namespace rule (owner, 2026-09-23): profiles live at `/xlearn/u/<username>`, courses at `/xlearn/<course-id>`.**
- **Shipped in v1.5.0 (live 2026-09-24): ADR-0025 plus its update, xlearn#46, #47.** Public profiles moved from `/xlearn/<username>` to `/xlearn/u/<username>`, with no redirect. Usernames no longer share the app's route namespace, so v2 can use dynamic course routes (`/:course/*`) and add top-level routes freely.
- **Course slugs are *not* reserved as usernames.** The reserved list is now just 22 impersonation and system words. #43 had briefly reserved course slugs; #47 removed that before the tag.
- **The guard runs the other way instead.** A course slug must not equal a static top-level SPA segment (`u`, `auth`, `settings`) or a gateway-owned path (`api`, `assets`, `healthz`, `readyz`, `.well-known`). Otherwise React Router's static-first ranking would hijack the course's routes. This is the **course-slug guard**, a seed-time check plus a test, specified in T0 / ADR-0026.
- **Also live:** the v1.4.2 service-version stamping fix (#44/#45).

**Request-path limits v2 must design around:**
- Gateway: 1 MiB request bodies, a 10 s upstream timeout, and a 60 s server write timeout.
- Event consumers: a 25 s handler timeout and a 30 s AckWait.

## Dependency decision matrix

_Filled in as topics settle: each platform dependency v2 adds (or explicitly declines), its
placement, single-node fit, GitOps/SOPS fit, cost, and the topic/ADR that decided it._

| Dependency | Decision | Where it runs | Single-node fit | GitOps / SOPS | Cost | Decided in |
|------------|----------|---------------|-----------------|---------------|------|------------|
| Course manifest | **Adopt**: a data file per course, compiled into every image | No process | ✅ none | No infra or SOPS change | none | T0 · ADR-0026 |
| `judge` service | **Adopt** (new bounded context: submissions, evaluations, grader registry, private eval pack). Shape is decided in T4, the sandbox in T3 | `xlearn` ns, +1 Deployment, port 8087 | ✅ ~50m / 64 Mi | Standard HelmRelease, CNPG schema and role via the ADR-0005 4-step, SOPS DB secret, image automation | ~0 | T0 (exists) · T4/T3 (shape) |
| Runner (sandboxed execution) | **Adopt in principle**; technology is decided in T3 | Own namespace, default-deny, ingress only from judge | Envelope: ≤ 2 concurrent jobs, ≤ 2 vCPU / 3 GiB limits, lowest eviction priority | New namespace, NetworkPolicy, chart knobs (off by default) | ~0 (node) | T0 (envelope) · T3 |
| New database engine | **Declined.** Postgres (lz4 TOAST, JSONB) covers every per-user payload; the pack is an OCI artefact | — | ✅ | — | — | T1 · ADR-0027 |
| `xlearn-evalpack` private repo + private GHCR data image | **Adopt**: judge-only read-only **image volume** (`EVALPACK_DIR`). Fallbacks: initContainer, then ORAS. **Spike before M3** | node containerd store (~30–40 MB per version, capped at 150 MB) | ✅ zero Postgres / zero RAM (cases read lazily) | ImageRepository (`secretRef`) + ImagePolicy `>=1.0.0 <2.0.0`; SOPS pull secret in `xlearn` and `flux-system` (existing `apps/secrets` rule); **no chart change** (`extraVolumes` / `imagePullSecrets` already pass through) | GHCR private quota 500 MB storage / 1 GB/mo transfer if billed; machine user + PAT | T1 · ADR-0027 |
| NATS auth (per-service credentials) + `messaging` NetworkPolicy | **Adopt; hard precondition for M3 and for erase.** nkey users in `$G`, rolled out **server-first** (N0–N4: a `no_auth_user: legacy` bridge, one NATS restart), with **fine ACLs rendered from `topology.go`** in the same step | existing NATS | ✅ | **No SOPS decryption in `messaging`:** nkey public keys are plaintext in its values; seeds are SOPS secrets in `apps/secrets`; the ops seed stays offline | — | T1 (requirement) · T7 (rollout, ADR-0035) |
| `xlearn-opscheck` CronJob + push channel (ntfy/email) | **Deferred (D34):** no alerting in v2; revisit before the first real invite. The owner monitors with landscape, kubescope and an on-demand `host-verify --cluster` | — | — | — | — | T1 · T7 (D34, ADR-0035) |
| Object storage (MinIO, Garage, SeaweedFS, R2, B2, S3) for app blobs | **Declined for v2.0.** Canvas and transcripts are inline `bytea` in Postgres; audio is T6's call | — | ✅ nothing added | — | $0 | T2 · ADR-0028 |
| Off-node backups (CNPG Barman Cloud Plugin to an S3 target) | **Deferred** (owner). Future target: the owner's second VPS. The design is ready in the T2 appendix | future: second VPS | plugin ~10m/32 Mi plus a sidecar ~64 Mi when adopted | HelmRelease `plugin-barman-cloud`, ObjectStore, ScheduledBackup, SOPS secrets under the existing rule | $0 now | T2 · ADR-0028 |
| Hostinger weekly VPS images | **Kept on** as the only safety net until backups land | Hostinger | — | outside GitOps; the Hostinger login gets 2FA (images hold plaintext Secrets) | included in KVM 4 | T2 · ADR-0028 |
| Judge job queue | **Postgres `SKIP LOCKED` table** in judge's schema; **no new broker**, no River | inside judge | ✅ runner lane 2 workers, llm lane 2 | none | $0 | T4 · ADR-0029 |
| Code editor: CodeMirror 6 (MIT) | **Adopt** for code parts (lazy-loaded) | SPA bundle | ✅ | — | $0 | T4 · ADR-0029 |
| Canvas: Excalidraw (MIT) | **Adopt** for SD and LLD (lazy-loaded; built when the SD course is scheduled) | SPA bundle | ✅ | — | $0 | T4 · ADR-0029 (D19) |
| judge memory envelope | **128 Mi request / 256 Mi limit** (was ~64 Mi in T0); generated perf inputs stream from an `emptyDir` file | `xlearn` ns | ✅ +~64 Mi requested | HelmRelease values | — | T4 · ADR-0029 |
| `xlearn-runner` (sandbox runner) | **Adopt: build a thin Go runner (R1)** with go-sandbox `forkexec` (MIT); nsjail is Plan B. Own public image, **own release stream**. It holds Go, g++ and Python toolchains at M3 | new `xlearn-runner` ns, 1 pod, 2 slots | req 100m / 512 Mi, **limit 2 vCPU / 3 GiB**, lowest priority | own Flux Kustomization **off v1's critical path**; runner-v* ImagePolicy; bearer token in SOPS | $0 | T3 · ADR-0030 |
| Host sandbox block (containerd `judge` drop-in, AppArmor + seccomp Localhost profiles, kubelet subuid range, sysctls) | **Adopt** as a manual step, **scripted** in `hack/host-bootstrap.sh` plus `host-verify.sh` | host | — | outside GitOps (like iscsid); `host-verify` after every reboot or k3s upgrade | $0 | T3 · ADR-0030 |
| Sandbox guards (VAP, RuntimeClass `xlearn-judge`, PriorityClass, ResourceQuota, default-deny NetworkPolicy) | **Adopt** | `xlearn-runner` ns | ✅ | Flux `sandbox-guards` Kustomization | $0 | T3 · ADR-0030 |
| **H0 host patch gate** (kernel meta-package + current noble kernel, core_pattern/suid_dumpable, apport off, module denylist) | **Adopt now; urgent** (protects v1 today). Separate task | host | reboot blip | manual, scripted | $0 | T3 · ADR-0030 |
| Canonical Livepatch (Ubuntu Pro personal) | **Declined** (owner): unattended upgrades plus the monthly reboot instead | — | — | — | — | T3 · D22 |
| gVisor (runsc) | **Declined for v2.0** (no per-process limits inside a sandbox); revisit for the execute step on open signup | — | — | — | — | T3 · ADR-0030 |
| Platform LLM provider: Anthropic `claude-sonnet-5` (+ Opus 5.5 for escalation and re-grade) | **Adopt**; one provider in v2.0 | external API | ✅ no node cost | dedicated workspace `xlearn-platform-prod`; hard limit + Console alerts; credits with auto-reload off | **≤ $100/month ceiling** (≈ $26–63 at 20 learners, DSA-only) | T5 · ADR-0031 |
| Anthropic Workload Identity Federation (k3s SA token → short-lived token, `workspace:inference`) | **Adopt, gated by a ≤ ½-day spike before M4**; fallback: an expiring single-workspace key | judge pod (projected token) | ✅ | inline JWKS pasted into Anthropic (re-paste if the k3s SA signing key rotates) | $0 | T5 · ADR-0031 |
| SOPS secret `xlearn-judge-llm` (HMAC salt; break-glass key) | **Adopt** | `xlearn` ns | ✅ | existing `apps/secrets` rule; rotation via `podAnnotations` bump | $0 | T5 · ADR-0031 |
| judge **egress** NetworkPolicy (default deny; DNS, PG, NATS, runner, 443 to non-cluster) | **Adopt as an M4 gate** (moved from T3 Track B) | `xlearn` ns | ✅ | chart 0.3.0 egress template | $0 | T5 · ADR-0031 |
| `xlearn-calib` calibration workspace | **Adopt** for acceptance and calibration runs (never in the cluster) | owner machine / private CI | — | GitHub OIDC WIF or a short-expiry personal key | ~$50/month during M4 bring-up; ~$10–30 analyzer + $20–60 per rubric one-off | T5 · ADR-0031 |
| OpenAI voice shell (`gpt-live-1` or `gpt-realtime-2.1-mini`, chosen by S6) on the **learner's** key | **Adopt for M6b (v2.1)**; browser↔OpenAI WebRTC with SDP brokered by coach | learner's browser ↔ OpenAI | ✅ no media on the node | coach 500m/256 Mi, `coder/websocket`, 20 s SDP route, drain and make-before-break (or a `coach-interview` Deployment if S6 M7 fails) | $0 to the owner (≈ $2–4 per voice interview on the learner's key) | T6 · ADR-0032 |
| TURN relay (for future peer-to-peer video interviews) | **Deferred** (not designed in v2): managed TURN vs self-hosted coturn with UDP ports | — | — | would need ufw/UDP or a managed provider | — | T6 · D29 · deferred by T7 (no peer-interview milestone in v2) |
| Object store for interview audio | **Declined** (audio is never stored) | — | — | — | — | T6 · ADR-0032 |
| Accidental-major release guard (ImagePolicy `>=1.0.0 <2.0.0` + a `.release-line` check in `deploy.yml`) | **Adopted; done 2026-09-24** (MI-2a: infra#29, xlearn#53). At GA: `.release-line`=2, range `<3.0.0` merged before the `v2.0.0` tag | infra ImagePolicies + xlearn CI | ✅ none | an infra PR bounding every `xlearn-*` ImagePolicy; a 1-line `.release-line` file | $0 | T7 · ADR-0034 (D32) |
| Kubelet `system-reserved` (250m / 1 GiB) + `eviction-hard memory.available<500Mi` (L23) | **Adopt** in the October host window (MI-11, the same k3s restart) | host k3s config | ✅ evicts before the kernel OOM-kills | host scripts (`host-bootstrap.sh`, asserted by `host-verify`); outside GitOps | $0 | T7 · ADR-0035 |
| MI-11a limit hygiene (6 Flux controllers 1 GiB → 512 Mi; limits for Traefik, cert-manager, metrics-server; Longhorn budgeted at p95 × 1.5) | **Adopt before the runner** (MI-12) | `flux-system` + those charts' values | ✅ frees ≈ 3 GiB of limits, so the memory-sum rule holds | infra PR | $0 | T7 · ADR-0035 |
| Hostinger manual snapshot before any contract, erase or GA tag | **Adopt.** Restore only by the R-d order (git pinned back first) | Hostinger | — | outside GitOps; one at a time, auto-deleted after 1 day | included | T7 · ADR-0034 |
| healthchecks.io dead-man + router | **Declined for v2** (D34); the design is kept ready for the opening | would be external SaaS | — | would hold only ping URLs (SOPS) | $0 (free tier) | T7 · ADR-0035 |
| VictoriaMetrics + VictoriaLogs | **Declined** (D34). Adopt on: a second node or the runner VPS, two undiagnosable incidents in a quarter, or the opening | would be the node | ≈ 0.4–0.6 GiB (inferred); needs L23 + MI-11a first | HelmReleases | $0 | T7 · ADR-0035 |
| `xlearn-staging` namespace | **Declined.** Substitutes: compose parity, the k3d rehearsal, `-rc` images, owner/tester-cohort dark launch | — | ❌ ≈ 1.3 GiB of limits; can't host a runner | would need a second NATS, ~25 objects, a second OAuth app | — | T7 · ADR-0034 |
| R1: VPS upgrade KVM 4 → KVM 8 | **Designed; on a trigger** (TR-CPU, TR-MEM, TR-DISK). Fixes neither steal nor isolation | same VPS | 8 vCPU / 32 GB | hPanel, ≤ 10 min; re-pin pools | ≈ +$21/month | T7 · ADR-0035 |
| R2: dedicated KVM 2 runner VPS as its **own k3s + Flux** | **Designed; on a trigger** (D21's triggers, TR-STEAL, TR-QUEUE, `SEAT_CAP` > 40). Never the backup VPS, never a k3s agent | new VPS (`clusters/runner`); runner reached over 443 with mTLS + bearer | takes the runner off the production node | its own Flux; ufw allows only the main node | $8.99 → $14.99/month | T7 · ADR-0035 |
| `identity admin` CLI + invite links | **Adopt** (CLI verbs at M1b; the invite flow in L) | identity image, run via `kubectl exec` | ✅ no new process | `SIGNUP_MODE` / `SEAT_CAP` HelmRelease env; no new secret | $0 | T7 · ADR-0033 |

## T0 — Guiding frame

> **Settled 2026-09-24** (D1–D4) · ADR: [0026](../adr/0026-per-course-extensibility-model.md) (Proposed) ·
> Full frame: [research/t0-extensibility-frame.md](research/t0-extensibility-frame.md).

**Method.** Three independent candidate models were written: evolving v1 with minimal disruption, deriving the model from course diversity, and starting from architecture, ops and security. Each got two adversarial critiques: a six-course end-to-end walk, and a review of solo-dev cost, single-node ops, security and migration. Every candidate scored 6–7/10. The synthesis takes evolve-v1 as the base, grafts in the best ideas from the other two, and fixes about 9 blocker findings.

**Recommendation (adopted).**
- A course is **data**: a settings-only manifest compiled into every image.
- A capability is **code**: closed registries keyed by **part type** (SPA widgets) and **grader kind** (judge), never by course.
- The loop after the learning signal is one course-blind engine.
- **judge** produces evidence only.
- **practice is the single writer** of learning signals: `problem_solved` v2 for course attempts and `touch_concluded` for revision touches. practice owns both kinds of timed attempt.

**Concepts.**
- **Course:** the path slug.
- **Manifest:** settings only.
- **Item:** the generalized problem, with a global id that is never re-parented and a role of core, reinforcement or drill.
- **Part:** a typed answer slot (`code`, `text`, `choice`/`blank`, `canvas`; `audio` reserved) with a cadence of iterate or final.
- **Grader:** one of `code`, `key`, `ai_rubric`, `composite`, plus an optional `analyzer`. It returns passed, failed or inconclusive, plus checks and a score.
- **Private eval pack:** judge-only.
- **Attempt:** course or touch; it concludes exactly once.
- **Learning signal.**
- **Common loop.**

**Six-course fit.**

| Course | Parts | Graders | Fit |
|---|---|---|---|
| dsa | code | code (Go, later C++), analyzer; self until tests exist | ✅ The manifest reproduces v1's constants exactly. Content is the gap (14/151, no tests). |
| system-design | text, blank, choice, canvas | composite: key (estimates, drills) plus one ai_rubric over canvas and text | ✅ New: the canvas widget and ai_rubric. |
| go-concurrency | code (multi-file), choice | code (`go-race` profile, `-count=N`), key | ✅ Needs `inconclusive` for flaky runs. |
| lld-ood | canvas (UML), code, text | composite: facade tests, then ai_rubric | ✅ |
| sql | code (SQL) plus a dataset explorer | code (`sql-pg`: throwaway seeded DB, result-set and EXPLAIN checks), key, ai_rubric | ✅ Learner SQL never touches CNPG. |
| behavioral | text (STAR); audio later | ai_rubric only | ⚠️ Partial. A learner-owned story bank needs a later ADR, plus consent and erase paths. Not a pilot. |

**Security / threat notes.**
- Hidden tests **and answer keys** exist only in the private eval pack. The repo and images are public.
- Events carry categories, numbers and refs, never learner prose or code.
- The arena and item surfaces withhold solution stages and pattern for any item with a live counted attempt or touch.
- Arena and Run never count and never use the platform AI key.
- AI analysis never blocks conclusion.
- No unauthenticated internal endpoint may spend money or return PII.

**Single-node fit.**
- 8 services plus one runner deployment, about **2.2 vCPU / 2.9 GiB requested of 4 / 15**.
- The runner envelope is ≤ 2 concurrent jobs with hard cgroup limits.
- The memory-limit sum rises to about 12.5 Gi with no swap, so runners get the lowest eviction priority.

**GitOps.** T0 itself needs no infra change. judge and the runner land later, in T3/T7.

**Migration (the manual loop coexists).**
- **M1:** the spine with no behaviour change (manifest plus golden test, per-course seed and id guard, `path_slug` everywhere, course routes with DSA aliases, expand → backfill → contract).
- **M2:** touch attempts in practice, the heatmap and mastery rebuilt from completed touches, and **DSA switches to the D2 revision-entry rule**.
- **M3:** judge plus the code grader, after sandbox isolation and admission limits.
- **M4:** the AI analyzer.
- **M5:** DSA becomes evaluator-only.
- The second course follows M3.
- Consumers of a new subject ship one release before its producer.

**Found while verifying.** v1 schedules the ladder only on a **first clean** solve (`review/store/store.go:307-309`), and `solved` is terminal. So a rough, assisted or miss problem is never revised, and its mistake can never close (closing needs 2 clean revisits). D2 fixes this.

**What T0 constrains downstream.**
- **T1:**
  - Model the manifest, items (role, topic, parts, per-language starters, revision probes), course assets (a public descriptor plus a private companion), and the **private eval pack**. The eval pack is never `problem_section` rows and never readable via the repo, images, curriculum GETs or the BFF.
  - Ids are global, retired rather than deleted, and the seed rejects re-parenting.
  - `path_slug` goes on every per-user row and event.
  - Also: the attempt `purpose`, `mock_session_item`, public-profile per-course visibility and grade provenance, and a sanitizing Markdown renderer.
  - Decide where public content and the eval pack live.
- **T2:**
  - judge-owned artefacts are stored by key, and events carry refs only (NATS `max_payload` is 1 MiB).
  - Uploads over 1 MiB bypass the gateway.
  - Backups must cover the judge schema and the eval-pack source.
  - Needs a per-account erase path across services.
- **T4:**
  - The Submission → Evaluation contract: context `course|touch|mock|arena|run`; action `run|submit|final|give_up`; status `passed|failed|inconclusive|error`; checks and score; versions.
  - Synchronous enqueue, then poll or SSE.
  - FIFO per attempt.
  - Only counted contexts emit events.
  - AI never flips a deterministic verdict.
  - Decide AI-grade finality (candidate: provisional plus one dispute) and abandoned attempts (candidate: resumable, explicit give-up only).
- **T3:**
  - Runner profiles `go`, `go-race`, `sql-pg`, later `cpp`, chosen by grader config.
  - Hidden-case feedback is verdict-only, and verdicts are computed outside the untrusted process where possible.
  - `inconclusive` for flaky runs.
  - Own namespace, default-deny, admitted only from judge.
  - Must work without KVM and with AppArmor's user-namespace restriction.
- **T5:**
  - The platform key is used only for guided course and touch evaluation and analysis, never for arena or Run.
  - Separate scoring and feedback calls, strict JSON out, and learner content delimited as data.
  - Budget exhaustion degrades to self-report and never blocks the loop.
  - Decide the key holder (candidate: judge, with a shared `platform/llm` package).
- **T6:**
  - Reads the rail and rubric from the manifest and scores via `ScoreMock` with `scored_by=ai`.
  - Uses the BYO key.
  - Transcripts stay out of assessment and off the public route.
  - Media bypasses the gateway's body and timeout limits.
- **T7:**
  - Order M1 → M5, and the second course after M3.
  - Sandbox default-deny first, then the xlearn, databases and messaging NetworkPolicies.
  - Admission control and rate limits before any evaluator reaches open signup.
  - Chart knobs (off by default): `automountServiceAccountToken`, `runtimeClassName`, `priorityClassName`, NetworkPolicy, split probes.
  - A release label other than "v2", and handling that `v2.0.0` auto-deploys.
  - v2 artboards for the workspace widgets and the course nav.

**Parked to later topics:** AI-grade finality and abandoned attempts → T4 · platform LLM key holder → T5 · where content and the eval pack live → T1.

## T1 — Content & data model

> **Settled 2026-09-24** (D5–D10) · ADR: [0027](../adr/0027-content-evalpack-and-user-data-model.md) (Proposed) ·
> Full model: [research/t1-content-data-model.md](research/t1-content-data-model.md).

**Method.** Four parallel research slices were run:
- the authored content model;
- private content delivery and content rights;
- per-user data, volumes and public-dashboard deltas;
- the authoring pipeline, including how Polygon, the Kattis package format and DMOJ structure problems.

They were synthesized into one draft, which then got two adversarial critiques: leaks and threat model (6.5/10), and solo-dev cost, migration and ops (6/10). Each critique raised one blocker. Both blockers and about 15 major findings were fixed in the revision.

**The rule.** If the statement would say it, it is **public**. If it encodes an answer not published elsewhere, it is **private**.

| Where | What |
|---|---|
| **This repo** (`curriculum/courses/<slug>/items/<id>/`), compiled into every image | Manifests, items, statements, samples, signatures, reference code, hints and editorial **once owner-stamped**, solution facts, asset descriptors, rubric definitions, media. The item schema cannot hold an answer. |
| **`xlearn-evalpack`** → private GHCR image → **judge-only** image volume | Hidden cases (expected outputs computed from the reference), keys that can't be derived from public text, SQL hidden data, rubric anchors and exemplars. **Nothing in Postgres.** Skew between the repos goes through one narrow `contract_hash`; on a mismatch the item falls back to self-report with a badge. |
| **Postgres**, schema per service, `path_slug` everywhere | Attempts (practice). Submissions, drafts and evaluations (judge: counted submissions **plus arena submits**, bodies inline). Touch results and mistakes (review). Mocks and projections (assessment). Visibility and erase (identity). **No new DB engine.** |
| **T2 blobs** | Canvas scenes and drafts, interview transcripts, audio. Video is not stored. Keys are `u/<account>/<kind>/<id>`, so erase is a prefix delete. |

**Numbers.**
- About **86 KB per active learner-day**.
- The owner alone is about 0.03 GiB a year; 100 learners about 2.1 GiB a year.
- The resize trigger (60% of PG or NATS) is about 285 learner-years.
- Arena submits, added after the model (D10), push this somewhat higher; re-model in T4.
- JetStream `MaxBytes` total is budgeted at 3.25 GiB of the 5 Gi store.
- Outboxes are kept as the durable event log, so JetStream needs no snapshots.

**Found while verifying (v1).**
- v1 copies LeetCode's Example 1 for Two Sum and 3Sum. Replace both in M1a.
- `pattern` is ungated in list/bulk, on the Revision card, in the arena and in the coach prompt, and coach review mode keys on `solved`. Fix: **one gateway `withhold()` on every surface**.
- The public route's token is never role-checked. Fix: an enforced `public-read` role and a new `/public/stats`.
- There is no account erase.
- NATS has no auth, so once judge results and erase requests travel as events, they could be forged.

**Security guarantees added.**
- **NATS auth plus the `messaging` NetworkPolicy are a hard precondition for M3 and for erase.**
- Hidden-case feedback is **passed count plus first failure class only**, which tightens T0.
- `key` and `ai_rubric` parts are evaluated only in counted contexts, never in arena or Run.
- Grades carry **`trust`**: honor for public-key probes and in-process tests, checked otherwise. "Judge-checked %" excludes honor.
- Pack material goes only into platform-key scoring calls with retention off. The BYO-key interviewer sees public descriptors only.
- Learner-content tables carry no fallible CHECKs, so Postgres logs can't capture learner code.

**Authoring.**
- AI drafts, machines verify, the owner stamps. **Expected outputs are never AI-written**, and keys are owner-confirmed.
- CI gates: the reference agrees with a brute oracle, wrong solutions get their declared verdicts, inputs are validated from `constraints[]`, `tests.lock` reproduces, leak lints pass, and the stamp gate holds.
- **v2.0 scope (D6; since D35 the scope of the v2.x line, delivered in waves):** 151 DSA items at the self-path tier, plus full packs for weeks 1–4 (about 35), plus a ~10-item second-course pilot. That is about **180–285 h** (±40%). The full six-course programme is about 450–650 h.
- **Rights:** original statements, LeetCode only as outbound links (never fetched), `provenance` required. **Licence: MIT for everything (D9).**

**Public profile (D7).**
- Per course, for enrolled and visible courses only:
  - concluded, passed and total;
  - grade mix with provenance, plus judge-checked %;
  - acceptance rate and languages;
  - touch pass rates;
  - mocks as a percentage.
- **Header totals come from visible courses only**, and behavioral is hidden by default.
- The heatmap becomes `proj_activity` (attempts, touches, mocks), which fixes the +6 first-solve spike.
- Never public: code, answers, prose, mistakes, notes, transcripts.

**Arena (D10).**
- Arena Submits are kept as history, with an **arena done marker separate from course done**.
- The course timer starts at attempt start, with the full question plus the coding area.
- The arena has a manual timer, off by default.
- Arena activity never counts and is learner-private in v2.0.

**Erase (D8).**
- `DELETE /api/me` → identity deletes → each service re-verifies, deletes its rows and outbox rows, and writes a tombstone.
- An erase ledger outside the backup is replayed after restores.
- The PITR window is the erase SLA.

**What T1 constrains downstream.**
- **T2:**
  - The blob kinds are only canvas, transcripts and audio.
  - **CNPG PITR (14–30 d) is the single critical backup** and gates grading for non-owner learners.
  - The WAL archive must be sized for draft churn.
  - The erase ledger is kept outside the CNPG backup and replayed on restore.
  - No pack in any in-cluster bucket.
- **T4:**
  - `payload_ref` = (submission, part).
  - Counted contexts and arena submits persist; **only counted contexts emit events**.
  - An allowlisted evaluation DTO.
  - `not_evaluated_here` for key and AI parts in arena and Run.
  - `inconclusive(contract_changed)`.
  - `trust` on evaluations and signals.
  - Tombstone checks.
  - Draft sweeper at 90 days.
  - Re-model volumes with arena submits.
- **T3:**
  - Public runner images, identical in CI and production.
  - Versioned harness codecs and a closed type/checker registry.
  - Comparison outside the sandbox (in-process profiles are honor-grade, with a separate compile step and every declared test required to pass).
  - A fresh tmpfs per job.
  - Run uses samples only.
  - Publish a time-limit baseline.
- **T5:**
  - Pack material only in platform-key scoring calls with zero retention.
  - The calibration set is synthetic or owner-authored.
  - Prefer `key` over AI where possible.
  - AI prose capped at 8 KiB.
- **T6:**
  - The BYO interviewer uses public rubric descriptors only (`ai-byo`).
  - Transcripts and audio are owned outside assessment.
- **T7:**
  - NATS auth before M3 and erase.
  - The stream budget and options knob.
  - The evalpack repo, machine user, pull secret, ImagePolicy and the image-volume spike.
  - `opscheck` plus UI badges.
  - Pre-push hook, ignore entries and the AGENT.md rule.
  - The erase UI, visibility toggles, and the `public-read` rate limit.
  - Compose parity (PG 18, NATS 2.14).
  - The Markdown renderer upgrade.
  - Choose the second course: go-concurrency (cheap, honor-grade) or SQL (cheap, checked).

## T2 — Object storage + off-node backups

> **Settled 2026-09-24** (D11–D13) · ADR: [0028](../adr/0028-object-storage-and-backups.md) (Proposed; amends 0027 §3, §6) ·
> Research and the ready backup design: [research/t2-object-storage-backups.md](research/t2-object-storage-backups.md).

**Method.** Four parallel research slices:
- in-cluster S3, including MinIO's 2026 status;
- external S3 providers and pricing;
- the CNPG backup design;
- app integration.

They were synthesized into one draft, which then got two adversarial critiques: resilience and security (6/10) and solo-dev cost and YAGNI (7/10). Every finding was fixed in the revision.

**Decided.**
- **No object store in v2.0 (D11).**
  - Canvas scenes and drafts and transcripts stay inline in Postgres permanently. They ride along with their rows, and erase stays pure SQL.
  - **MinIO is out.** The community server was archived 2026-04-25 (source only). The operator was archived 2026-03-20. The Bitnami charts are frozen. AIStor Free is proprietary: its licence expires and it has no encryption at rest.
  - In-cluster stores (Garage, SeaweedFS) share the node's disk, so they can never be the backup.
- **No off-node backups for now (D12).** The owner's second VPS is the future target.
  - The recommended design stays ready in the appendix: CNPG Barman Cloud Plugin v0.15.0, a lineage per restore, 5-minute RPO, the mandatory post-restore sequence, and the drill plan.
  - The recommended B2 Amsterdam primary, AWS Mumbai second copy and governance lock were **not adopted**. They remain the fallback option set.
  - Re-target the design at an S3 endpoint on the second VPS (e.g. Garage single-node) when backups land.
- **Controlled, invite-only signup (D13)** for capacity reasons. The mechanism is designed in T7.

**Risk accepted for v2.0.**
- A node or disk loss loses all learner data since the last Hostinger weekly image, up to about 7 days.
- There is no point-in-time recovery, so a bad migration can only be rewound to the last weekly image.

Mitigations carried into T7:
- Keep Hostinger's weekly images on, and protect the Hostinger login with 2FA. The images hold `sops-age` and every plaintext Secret.
- Keep two offline copies of the age key.
- `prune: disabled` on the CNPG Cluster and Namespace, plus the "never revert-re-add a deleted Cluster" rule. A revert would `initdb` an empty database.
- opscheck disk and volume checks, and the 60% resize trigger.

**Findings worth keeping for when backups land.**
- **Plugin bug #828** (open; reproduced on CNPG 1.30 + PG 18) can silently stop WAL archiving after a restart. That needs an hourly `ContinuousArchiving` check plus a restart assertion at the first drill.
- **R2 can't hold Barman base backups**, because of its equal-part multipart rule.
- **A B2 compliance lock doesn't survive account closure.**
- **A PITR restore revives revoked sessions and deleted BYO keys.** The post-restore sequence rotates the JWT key, revokes sessions and suspends BYO keys.
- **Never boot an old image or snapshot while the archive holds newer WAL.**

**Amendments to T1 (ADR-0027, via ADR-0028).**
- §3: blobs stay inline permanently, and `object_key` is deferred until a trigger fires (judge body bytes > 1 GiB or 25% of the PG volume, or a payload > 1 MiB).
- §6: the erase ledger, its replay and the backup-erase SLA are **deferred with backups**. v2.0 erase is delete plus outbox delete plus tombstones.
- "CNPG backups gate non-owner grading" is **withdrawn**; controlled signup replaces it.

**What T2 constrains downstream.**
- **T4:**
  - Canvas and draft bodies are inline `bytea`.
  - judge enforces its own body limit (512 KiB scene plus a 32 KiB export).
  - The gateway switches to `http.MaxBytesReader` with a typed 413 (today it uses a silent `io.LimitReader`, `bff.go:164`).
  - Raise the SPA canvas save timeout (15 s today).
  - Erase is SQL only.
  - Re-model WAL and volume with arena submits and drafts.
- **T3:** runners get no object-store egress or S3 credentials, and runner scratch is tmpfs and never persisted.
- **T5:** AI prose stays inline, and LLM calls never get bucket access.
- **T6:**
  - Decide whether and where audio is stored. With no bucket, the options are Postgres chunks, the second VPS, or not storing it.
  - Transcripts are inline in T6's own schema.
- **T7:**
  - The invite-only signup mechanism (D13).
  - Hostinger 2FA and offline age-key copies (owner checks).
  - `prune: disabled` on the Cluster and Namespace.
  - A future "backups to the second VPS" milestone, using the appendix design.

## T4 — Judge types & the common contract

> **Settled 2026-09-24** (D14–D19) · ADR: [0029](../adr/0029-judge-contract-and-learning-signal.md) (Proposed) ·
> Full design: [research/t4-judge-contract.md](research/t4-judge-contract.md). Its §13 overrides the body where they conflict.

**Method.** Four parallel research slices: contract and lifecycle; archetypes and plug-ins; evidence → learning signal; learner flows. They were synthesized into one draft, which got two adversarial critiques: six-course loop correctness (6/10, 1 blocker) and integrity and ops (6/10, 2 blockers). All 3 blockers and 17 majors were fixed in the revision. The owner then reshaped five answers (D14–D18) and picked Excalidraw (D19).

**The design.**
- **judge is a course-blind evidence engine.**
  - Closed registries: 5 part types, and 3 grader kinds (`code`, `key`, `ai_rubric`) under one `composite@1` root, plus a post-hoc `analyzer`.
  - **Archetypes are compositions**: A code IDE (dsa, go-concurrency, sql); B structured response (system-design, lld-ood, behavioral: key steps plus one AI rubric over text, blanks, choices and an Excalidraw canvas); C quiz/key (drills and touch recall probes).
  - A new course that reuses them needs no code.
- **Contract.**
  - Submissions are idempotency-keyed. The server sets the context, pin, time and `attempt_seq`.
  - Key-only work is evaluated inline. Runner and LLM work goes through a Postgres `SKIP LOCKED` job table.
  - Every submission ends in exactly one terminal evaluation (`passed | failed | inconclusive | error`).
  - The learner DTO is an allowlist. Hidden tests show passed/total, one perf bit and the first failure class.
  - Transport is poll plus ETag; SSE is deferred.
- **Conclusion.**
  - practice stores evaluations as facts, always acks, and pulls gaps from judge.
  - It locks the grade on a gap-free prefix and concludes **once**.
  - The new signal fields include `anchor_at`: scheduling and projections date from the moment the grade was determined.
- **Owner-shaped rules.**
  - **Hard time limit (D15/D18):** DSA gets 45 minutes, with the hint at 15. Using the hint or the AI coach caps the grade at Assisted. A timeout concludes as Miss and is re-attempted via the schedule.
  - **A pass finishes the attempt (D16):** no re-implement stage; the AI can attach pointer notes and an optional revisit to passing solutions.
  - **AI results are editable suggestions (D14):** accept, edit, or one blind re-grade; manual entry is the fallback.
  - **The arena is unrestricted (D17).**
- **Learning loop.**
  - Revisable items get L1–L5 at `anchor_at`.
  - Mistakes pre-fill from judge signals via manifest rules; the learner always wins. This switches on the weak area in M3.
  - Up to 3 concepts to revise per attempt.
  - Mock evaluations are evidence only (`ScoreMock` stays self-scored in v2.0).
  - Touch formats and pass criteria are defined per course and band.
- **Large performance tests:** a closed, public generator registry (seed plus spec). The 256 KiB literal cap stays; a stamped exception allows ≤ 2 MiB.
- **Capacity.**
  - About 20–60 code submits per minute with 2 runner workers.
  - An AI-graded SD final costs about $0.05–0.15.
  - Volume is re-modelled to **about 124 KB per active learner-day**, so the resize trigger comes at about 198 learner-years.

**Behaviour changes vs v1 (release-note material).**
- The full statement is shown at start.
- The hard 45-minute limit.
- Revealing the solution before passing counts as a give-up (Miss).
- "Complexity stated **correctly**".
- Ahead-of-schedule items go to the arena.
- Using the coach caps the grade at Assisted.
- No re-implement stage.
- The pattern chip is hidden during a live attempt (the artboard leaked it).

**Amendments to settled docs** (21, listed in the appendix §11.4). The key ones:
- `run` is an action, not a context.
- T0's FIFO assumption is replaced by seq, a gap-free lock and a pull reconciler.
- `passed ⇔ grade ≠ miss` on the signal.
- PRD R-SR1 schedules from grade lock (`anchor_at`).
- T1's hidden classes gain OLE, RACE, DEADLOCK and LEAK.
- The pack carries generator specs and seeds.
- The practice and judge schemas gain the lifecycle columns and fact tables.

**Open item (owner):** a separate research session to set per-problem time budgets. For now the manifest default is 45 minutes with the hint at 15.

**What T4 constrains downstream.**
- **T3 (runner contract):**
  - compile kept separate from execution;
  - per-case results measured outside the untrusted process;
  - positioned diagnostics;
  - typed infra errors, a `throttled` flag and a goroutine dump;
  - streamed input ≤ 8 MiB per case;
  - ≤ 2 concurrent jobs;
  - a fresh tmpfs with no network;
  - a published time-limit baseline;
  - must work without KVM and under AppArmor's userns restriction.
- **T5:**
  - implement `Scorer{Reserve, Score, Feedback, Analyze}` with the strict schemas;
  - zero retention; ≤ 8 calls per evaluation; k-sampling;
  - pack material only in `Score`;
  - budget exhaustion falls back to manual entry;
  - **the analyzer also runs on passing solutions (D16)**, which is new cost;
  - the key holder and per-account budgets.
- **T6:**
  - mock AI scoring is a BYO-key proposal, then `ScoreMock` once;
  - mock evaluations are silent in v2.0.
- **T7:**
  - the M3 minimum cut vs M4 (AI);
  - NATS auth before M3;
  - coach-assist capture (the gateway records coach use on the open attempt);
  - artboards A1–A9 (Excalidraw canvas workspace included);
  - `RELAY_INTERVAL` 1 s; SPA timeouts 25 s; judge 128/256 Mi;
  - the single-replica cache epoch documented as a scale-out blocker.

## T3 — Code-execution sandbox + cluster hardening

> **Settled 2026-09-24** (D20–D23; **spike pending**) · ADR: [0030](../adr/0030-runner-technology-and-host-hardening.md) (Proposed) ·
> Full design plus S0 results: [research/t3-sandbox.md](research/t3-sandbox.md). Its §12 overrides the body where they conflict.

**Method.** Five parallel research slices:
- isolation tech on this host;
- OSS judges vs build;
- language images and the SQL sandbox;
- cluster hardening and threat model;
- performance and limits on 4 vCPU.

They were synthesized into one draft, which got two adversarial critiques: escape/red team (6/10, 1 blocker) and solo-ops and performance (6/10). Every finding was fixed in the revision. The owner approved only the read-only S0 facts. The kernel gap was verified separately.

**Threat model in one line.** An invited learner runs arbitrary code on the node that holds everything: CNPG with **no backups**, the SOPS age key, kubescope's cluster-admin, and the JWT and coach master keys. So **the sandbox boundary carries the platform's security.**

**Decided design (R1).**
- **Build, don't adopt.** Judge0 and Piston are privileged and store expected outputs. DMOJ uses ptrace and is AGPL. go-judge pools containers. gVisor has no per-process limits inside its sandbox.
- **The runner pod.** A thin Go runner (`xlearn-runner`) in its own namespace, running in a **user-namespaced runc pod** (`hostUsers:false`). It has namespaced capabilities, Localhost AppArmor and seccomp profiles, a VAP-pinned shape, and no token, no DNS and no network.
- **Per test case.** A fresh jail without a user namespace: a fresh tmpfs, zero capabilities, a kill-by-default seccomp allowlist and its own cgroup leaf. **All verdict evidence is measured outside the learner's process.** Expected outputs stay in judge.
- **Privilege split.** A capability-less front parses learner bytes; the privileged spawner never does.
- **Primitive.** go-sandbox `forkexec` (MIT); nsjail is Plan B.
- **Languages.** **Go, C++ and Python at M3 (D20).** go-race and SQL come with the pilot. SQL gets a fresh Postgres per job inside the jail, never CNPG.
- **Timing.**
  - 2 slots of 1 CPU each, with a CPU-time TL = max(3 × reference, 1 s).
  - **cgroup CPU time includes steal on this kernel (S0)**, so `throttled` comes only from steal or a canary, and a failing time verdict gets one quiet re-run.
  - Per-account runner budgets plus a duty-cycle breaker stay below Hostinger's CPU throttle.
  - Run p50 is about 1.5–2.5 s (inferred).
- **Placement (D21).** On the production node while signup is invite-only. It moves to a dedicated runner VPS on a trigger.
- **Patching (D22).** Unattended upgrades with the kernel metas unheld, a monthly reboot, pinned k3s patches, Renovate. **No Livepatch.**

**Found and verified (urgent, affects v1 today).**
- The node runs kernel **6.8.0-90** against noble's **6.8.0-142**, and the kernel meta-package isn't installed, so it never auto-updates.
- Livepatch is off.
- **Core dumps are piped to apport as host root**, with `fs.suid_dumpable=2`.
- The fix is **H0**, done in this session on **2026-09-24**:
  - **The root cause was holds, not a missing package.** The Hostinger image had put `linux-virtual`/`linux-image-virtual` on hold. They are now unheld; `cloud-init` stays held.
  - **Live, with no reboot:** `core_pattern=core`, `suid_dumpable=0`, apport masked, and a 40-module denylist.
  - **Upgrade:** `apt full-upgrade` (59 upgraded, 8 new) installed **kernel 6.8.0-142** as the GRUB default. **The reboot is scheduled for Friday 2026-09-25** (reminder task `vps-reboot-reminder-h0`).
  - **Health:** 39/39 pods, Longhorn and CNPG stayed healthy throughout.
  - **Recorded as code:** `hack/host-bootstrap.sh` and `hack/host-verify.sh` in the infra repo (PR).

**Hardening plan.**
- **Track A (gates M3):**
  1. A0: H0.
  2. A1: S0 (done).
  3. A2: the spike (deferred gate).
  4. A3: chart 0.3.0 knobs.
  5. A4: **databases and messaging ingress NetworkPolicies**.
  6. A5: **NATS nkeys**.
  7. A6: the host sandbox block.
  8. A7: `sandbox-guards` via Flux.
  9. A8: the runner deployed dark, with its acceptance suite.
- **Track B (non-gating):** PSA labels, tokens off, egress rules, fine NATS ACLs, CNPG ≥ 18.6, k3s v1.36.5.
- **Operating model.** Host steps are scripted (`host-bootstrap.sh`, `host-verify.sh`). The runner has its **own release stream and Flux Kustomization, off v1's critical path**.

**Deferred spike (D23), now a pre-M3 gate.**
- Parts:
  - P0–P2 on a multipass Ubuntu 24.04 VM (the drop-in, the pod userns, the profiles, userns-less jails, INV-14 OOM placement, cleanup);
  - P3, an amd64 replay (allowlists for **Go, C++ and Python**, and TSAN with `mmap_rnd_bits=32`).
- Budget: about 10 h plus time for the extra languages, hard-stopped at 1.5–2 days.
- Environment: throwaway.

**S0 highlights.**
- The systemd cgroup driver is confirmed ✓.
- The drop-in dir doesn't exist yet.
- runc is 1.4.2 and containerd 2.3.4.
- CPU PSI is about 2–3%.
- **`PARAVIRT_TIME_ACCOUNTING` is off**, with about 2.4% steal since boot.
- L3 is split in 4 slices.
- `mmap_rnd_bits=32`.
- To fix: `io_uring_disabled=0`, `kptr_restrict=1`, `suid_dumpable=2`.
- No kubelet subuid range.
- None of the denylist modules are loaded.
- **The 72 h steal sample runs until about 2026-09-27 07:52 UTC** (`/tmp/xlearn-s0-vmstat.log`; collect, then delete).

**What T3 constrains downstream.**
- **T5:** the runner has no egress, no secrets and no LLM path. The analyzer never sees hidden inputs, expected outputs, per-case timings or test sources. Anything the AI wants executed goes through judge as a public-sample Run on the learner's budget.
- **T6:** mock coding rounds reuse the runner via judge (context `mock`), capped at ≤ 1 Run per 10 s, and no media reaches the runner.
- **T7:**
  - Track A and B ordering, and H0 first;
  - the spike gate before M3;
  - the runner image and release stream, the `sandbox.yaml` Kustomizations, the second ImageUpdateAutomation;
  - chart 0.3.0; host scripts plus the DR-runbook step;
  - NATS nkeys, NetworkPolicies, PSA;
  - production calibration (per-language TL multipliers);
  - collecting the S0 steal data.
- **Amendments** (appendix §10): T4's runner lane gets budgets, the CE cache and the breaker; `Job`/`Result` fields; runtime SIGSYS maps to RE (counted); T1 pack lints (Σ TL ≤ 40 CPU-s, `mem_mb` ≥ 2 × the reference peak); no always-on sandbox PG; `messaging` needs no SOPS decryption for nkeys.

## T5 — Platform AI + two-tier keys

> **Settled 2026-09-24** (D24–D27) · ADR: [0031](../adr/0031-platform-ai-and-two-tier-keys.md) (Proposed; narrows ADR-0007) ·
> Full design: [research/t5-platform-ai.md](research/t5-platform-ai.md). Its §12 overrides the body where they conflict.

**Method.** Four parallel research slices (key architecture and secrets; cost and budgets; quality, safety and privacy; coach and BYO) were synthesized into one draft. It then faced two adversarial critiques: cost, abuse and ops (6/10) and security, privacy and grading quality (7/10). All 14 majors were fixed. Prices, retention terms and features were verified on the web on 2026-09-24.

**Design.**
- **Two tiers.**
  - **judge owns platform AI.** One owner-paid Anthropic credential is used only for counted course and touch evaluation and analysis. The code lives in `internal/judge/ai` (the `Scorer` and the ledger) and `internal/platform/llm` (adapters, catalog, prices).
  - **The coach stays BYO** for chat and T6 interviews.
  - **No BYO for platform tasks** in v2.0.
- **Credential.**
  - **Workload Identity Federation:** a k3s SA token is exchanged for a short-lived `workspace:inference` token in a dedicated workspace. A ½-day spike before M4 gates it; the fallback is an expiring key.
  - **judge gets default-deny egress** (an M4 gate).
  - No tools, Batch or Files. Prompts and learner content are never logged, and a canary test enforces that.
- **Model.**
  - `claude-sonnet-5` ($2/$10 per MTok) with an explicit effort. Opus 5.5 is used only for escalation and as the independent re-grade configuration.
  - The analyzer's effort and caps are set by an acceptance run (≥ 40 test + 30 dev).
  - The calibration gate is strengthened: QWK ≥ 0.70 with a bootstrap lower bound ≥ 0.60, n ≥ 50 per rubric, and outcome-based injection twins.
- **Cost.**
  - DSA-only runs **≈ $1.3–1.9 per learner-month at `low` or $2.8–3.2 at `medium`**, so **≈ $26–63/month at 20 learners**.
  - The SD course at 10% of attempts adds ≈ $0.7–1.1 per learner-month.
  - Spend has four layers: the provider limit **$100** (D25), the app cap **$80**, per account $6/month and $1/day, and pace-based shedding of pass-review notes.
  - Degrade order: advisory → counted → final. **Nothing blocks the loop.**
- **Privacy (D24).**
  - `RetentionPolicy`: ≤ 30-day provider retention and no training; ZDR is sales-only.
  - Processing is outside India. This is lawful now; the DPDP duties from ~2027-05-13 are shipped anyway before M4 opens.
  - Two unticked opt-ins at invite acceptance plus a behavioral opt-in, all switchable off.
  - 18+ attestation.
  - The privacy notice names Anthropic, the retention period and the toggles.
- **Pass review (D26).** Course passes, plus materially different touch passes. The latter get a "correct, with improvements" marker and pointer notes; they are advisory and shed first.
- **Coach.**
  - A per-course persona.
  - The coach is `locked` during touches and mocks; `attempt` mode withholds pattern and concepts.
  - **Per-problem assist cap (D27):** a confirmation is required, and the grade is capped at Assisted.
  - The learner's own usage is displayed; limits are 300 messages/day.
  - ADR-0007 fixes: quota errors no longer disable keys, `max_tokens` and effort are set, OpenAI `store:false`, AEAD binding, a master-key keyring, and a model catalog.
  - UI naming: **"xLearn AI (included)"** vs **"Your AI coach (your key)"**.

**Found and verified (live v1 bugs, suggested as a separate task).**
- The onboarding API-key field is uncontrolled, so **keys typed at onboarding are discarded**. It also offers an unsupported "Google" provider.
- OpenAI `insufficient_quota` **disables the learner's key**.
- Anthropic replies are capped at `max_tokens` 1024.

**What T5 constrains downstream.**
- **T6:**
  - BYO only, via the `interview` default key; there is no platform key for mocks.
  - The raw key never leaves the coach. Realtime uses **derived credentials minted by the coach** (shortest TTL, one per session, logged) or streams proxied through the coach.
  - Only public rubric descriptors are sent; results are labelled `ai-byo` / `trust=honor`, and `ScoreMock` is called once.
  - The coach is locked during a mock.
  - `ErrQuota` means "top up", never a key disable.
- **T7:**
  - **M1:** the coach P0 fixes, the mode gate plus `withhold()`, D27 capture, and the BYO daily cap.
  - **Before M4:** the WIF spike, the chart 0.3.0 egress template plus judge default-deny egress, the provider runbook (dedicated org if shared, workspace, limit and alerts, credits with auto-reload off), and a push channel judge can call.
  - **M4:** extract `platform/llm`, build `internal/judge/ai` and the ledger tables, the analyzer acceptance set with caps sized from it, `/api/me/ai-allowance`, consent toggles, the opscheck AI digest, and the canary test.
  - **Before learners:** the privacy notice, 18+ attestation, and two weeks of dogfood re-measurement.
  - **SD-course milestone:** calibration sets (≥ 80 per rubric) and the independent re-grade configuration.

## T6 — Realtime AI mock interviewer

> **Settled 2026-09-24** (D28–D31; **spike S6 approved, pending**) · ADR: [0032](../adr/0032-realtime-ai-mock-interviewer.md) (Proposed) ·
> Full design: [research/t6-realtime-interviewer.md](research/t6-realtime-interviewer.md). Its §13 overrides the body where they conflict.

**Method.** Four parallel research slices: realtime model options, architecture and media path, failsafes and BYO billing, and assessment, privacy and ethics. They were synthesized into one draft, which then faced two adversarial critiques: feasibility and cost (6.5/10) and privacy, ethics and security (7/10). All 15 majors were fixed. Model names, prices and limits were verified on the web on 2026-09-24. The voice APIs are weeks old, so S6 re-checks them.

**Decided.**
- **Scope.** The interviewer is **M6 (proposed v2.1), not v2.0**. It comes after M3 and runs alongside M4/M5.
  - **P0** is a **text interviewer** on any `interview` model (Anthropic or OpenAI), plus all the owner's failsafes.
  - **P1** is **voice** via **one** OpenAI shell chosen by **S6** (GPT-Live-1 vs `gpt-realtime-2.1-mini`).
  - **P2** adds extras.
- **Architecture.**
  - It lives in coach (`internal/coach/interview`).
  - A text **director brain** writes all evaluative speech; a thin voice shell only listens and speaks.
  - **The browser talks WebRTC directly to OpenAI.** Coach brokers the SDP server-side, so no credential reaches the browser, and holds an outbound sideband.
  - **No UDP, Traefik, object-store or namespace change.**
- **Perception (D29).** A **second interviewer sharing the session**:
  - **Inputs.** It works from live voice plus the **on-screen answer widgets** as structured state, never screenshots.
  - **Cadence.** Updates go **on change every 2–3 s and on every turn**, held in one replaceable "current screen" item.
  - **Final analysis.** It covers the **whole transcript plus the on-screen state at each point**.
  - **Never assessed:** tone, emotion, accent, fluency or appearance (EU AI Act Art. 5(1)(f)).
  - **No video goes to the AI.**
- **Failsafes (owner spec).**
  - **5-minute top-up grace:** clock frozen, free checkpoint, hang up, countdown, probe.
  - **Pause up to 24 h** (fixed deadline, ≤ 3 pauses), then **`incomplete`, unscored** (D30).
  - **Resume modal.** The free deterministic state shows instantly, and **[Summarise & resume]** makes one ≈ $0.04 call on the learner's key, cached per pause.
- **Scoring.** A brain-written debrief with improvement areas, then an `ai-byo` proposal the learner must **explicitly accept or edit** (no auto-accept). Then `ScoreMock` runs once. In voice mode Communication is self-scored until a fairness check passes. **The public profile shows the mock count only (D31).**
- **Privacy and limits.**
  - **Audio and video are never stored.** Transcript plus snapshot timeline are C4, inline in coach, deleted 30 days after scoring by default.
  - Consent is taken per session. The interviewer discloses that it is an AI.
  - Voice is **Chrome/Edge only** at launch, and **no voice for EU/EEA** until a legal review.
  - **Cost on the learner's key:** text ≈ $0.45, voice ≈ $2.1–3.1 per 45 minutes, plus GST. There is a mandatory $ cap.
- **Future (D29).** **Peer-to-peer interviews between learners with video.** Signalling reuses the SDP-relay pattern; a TURN provider is deferred.

**Spike S6 (approved, pending).**
- ≤ 1 day on the owner's OpenAI key in a throwaway project with a **$10 hard limit**; expected spend is about $7.
- **The owner must be present:** create the project and key, and play the candidate in a scripted 15-minute mock.
- **Measures:** 60-minute survival, silence and typing discipline, push-to-talk, latency, code bluffing, quota fixtures, surviving a coach restart (M7, which decides the deploy shape), browser tampering, and **the cost and latency of D29's 2–3 s screen updates**.
- **When:** before the M6a design freeze.

**What T6 constrains downstream (T7).**
- **Sequencing:** M6 comes after M3; S6 before M6a.
- **Gates:** M1's coach P0 fixes (typed errors, quota ≠ disable, `ErrModelAccess`) and the `interview` default key and catalog; M3's judge `mock` context and CodeMirror; a twin fairness gate.
- **P1 infra:** coach 500m/256 Mi, drain and make-before-break, `coder/websocket`, the 20 s SDP route, and Permissions-Policy/CSP (including camera and mic denied on sibling apps via a Traefik middleware in `../infra`).
- **Operations:** opscheck interview counters and a release-runbook check for live interviews.
- **Product artboards:** setup and pre-flight, the live HUD, grace, pause and resume, debrief and proposal, and text mode.
- **Amendments:**
  - ADR-0007: SDP-brokering credential and key lifetime.
  - ADR-0024: mock count only (D31).
  - T4: `scored_by ∈ {self, ai-byo}`, `ScoreMock` only from `finished`, no auto-accept for mocks.
  - T1: the transcript and snapshot-timeline owner is coach, and `mock_session.format/status/time_multiplier`.
  - T2: audio is never stored.

## T7 — Cross-cutting and infra-first rollout

> **Settled 2026-09-24** (D32–D35) · ADRs: [0033](../adr/0033-invite-only-admission-and-owner-admin.md) (admission) ·
> [0034](../adr/0034-v2-release-labelling-gating-and-rollback.md) (releases) · [0035](../adr/0035-v2-operations-nats-auth-limits-capacity.md) (operations), all Proposed ·
> Full design: [research/t7-cross-cutting-and-rollout.md](research/t7-cross-cutting-and-rollout.md).
> **Authoritative plan for the build-plan session: [rollout-plan.md](rollout-plan.md).**

**Method.** Four parallel research slices:
- pipeline, NATS auth, observability, limits and capacity;
- authz deltas, invite-only signup and the public dashboard;
- release labelling, feature gating and rollback;
- the MI track, the milestone map and the missing artboards.

Evidence came from code at `main`, `../infra`, read-only `ssh vps` (get/describe/top, NATS `/varz` and `/jsz`, `sar`) and the web. The slices were synthesized into one draft, which faced an adversarial critique (7/10). All 6 majors were fixed; the 13 minors were routed to the ADRs and the rollout plan. The owner verified the live-security findings before deciding.

**Found in production (verified 2026-09-24).**

| Finding | Evidence | Fix |
|---|---|---|
| **Signup is open to the internet** on both create paths (email signup and the first GitHub sign-in); no invite, no email verification | `internal/identity/authemail.go`, `handlers.go` | **Fixed 2026-09-24 in v1.5.2 / infra#30:** `SIGNUP_MODE=closed` (MI-2c, xlearn#52). A stranger's account created before then stays a v1-only account until M1a/M1b can mark or suspend it; nobody can run code before M3 |
| **Pre-account hijack:** GitHub sign-in auto-links into *any* account with a matching email, and password accounts are never email-verified ([USENIX Sec '22](https://arxiv.org/abs/2205.10174)) | `internal/identity/store/store.go` (`FindOrCreateAccount`); ADR-0023 §3 | **Fixed 2026-09-24 in v1.5.2:** auto-link only into password-less accounts (MI-2b, xlearn#51) |
| **Roles are never checked:** every JWT mint is `["learner"]` and nothing reads `Roles` | `gateway/bff.go`; `platform/auth/auth.go` | `account.role` (M1a), `RequireRole` (M1b) |
| **No NetworkPolicy** in `xlearn`, `databases` or `messaging`, while identity's `/sessions/*` and `/internal/*` are unauthenticated | `kubectl get networkpolicy -A`; `identity/service.go` | MI-5 (`databases`, `messaging`) and MI-5a (`xlearn` ingress), before M3 |
| **No rate limits** in the app or Traefik. bcrypt under identity's 250m CPU limit can stall `/sessions/validate`, which every API call needs | `kubectl get middleware -A` | limits L1–L6 at M1b |
| **Admin consoles share xLearn's origin.** kubescope (cluster-admin + exec), landscape and the read-write Longhorn UI sit on `projects.sujaykumar.dev`, isolated only by path. One XSS while the owner is signed in to kubescope = cluster-admin → `sops-age` | `../infra` README, same-origin caveat | MI-5b: move them to `ops.sujaykumar.dev` before the first non-owner account |
| **Unbounded ImagePolicies** (`>=1.0.0`): a stray `v2.0.0` tag deploys the whole fleet in ~4 min, and no 1.x tag can deploy again | `../infra` `apps/image-automation.yaml` | **Fixed 2026-09-24 by infra#29** (every range `<2.0.0`), with xlearn#53's `.release-line` check (D32, MI-2a) |
| **The documented rollback doesn't work:** image automation rewrites a pinned tag within ~1 min | [`../git-strategy.md`](../git-strategy.md) "Rollback" | R-a → R-d (ADR-0034) |
| **No alerting of any kind** (no Flux Alerts, CronJobs or metrics stack). Poison events are skipped after ~8 h of retries with no record | `kubectl get providers,alerts,receivers,cronjobs -A`; `platform/events/consumer.go` | accepted for v2 (D34); a dead-letter hook in M1 |
| **The kubelet has no memory eviction** and no system reservation | kubelet `configz` | L23 in the October host window |
| **Memory margin ≈ 0 after v2.** Σ limits ≈ 12.6 GiB + 23 limitless containers ≈ 1.16 GiB + host 1.8–2.3 GiB against 15.6 GiB; a fleet rollout adds ≈ +1.1 GiB | `kubectl top`, `get pods -o json`, `describe node` | MI-11a (≈ −3 GiB of Flux limits) before the runner; the memory-sum rule |

**Decided (owner).**
- **D32 release labels.** 1.x minors through the build; **`v2.0.0` = the owner-facing v2.0 GA**, **`v2.1.0` = the interviewer GA**, M5 a later 2.x minor; the API stays `/api/v1`. **Guard now** (MI-2a, done 2026-09-24: infra#29, xlearn#53): ImagePolicies `>=1.0.0 <2.0.0` plus the `.release-line` check. At GA the widening to `<3.0.0` merges first, after checking that no non-prerelease `2.*` git tag or GHCR tag exists; a new policy or a raised floor goes tag first (image before policy); a narrowing is only the R-b rollback, where the merge is the rollback (ADR-0034 §1.4).
- **D33 admission and admin.** Signup is closed (v1.5.2, live 2026-09-24): the auto-link fix (xlearn#51) and `SIGNUP_MODE` (MI-2c, xlearn#52; infra#30 sets `closed`). v1.5.2 honours `open` without `DEV_AUTH`; that guard is M1b. In v2: `SIGNUP_MODE`, owner-minted single-use invite links (7 d, 30 d max), `SEAT_CAP` 15, an acceptance step, `account.role` (learner, tester, owner) and `status` in the DB and never in the JWT, a CLI-minted `tester` role, and the `identity admin` CLI (every verb audited). No web admin, no waitlist form. Admin consoles move to `ops.sujaykumar.dev` (MI-5b) before the first non-owner account.
- **D34 no alerting.** The owner monitors with landscape, kubescope and `host-verify --cluster`. A failure is seen only when the owner looks (accepted). Revisit before the first real invite.
- **D35 audience.** v2 is used by the owner (and testers) only; **the learner gate L still ships in v2**; real learners are invited from **v3**, by the owner's choice. v2.0 GA = MI + M1–M4 + pilot + L, complete for the owner. D6's content is the v2.x line's scope, in waves.

**Adopted recommendations (no owner question; the ADRs stay Proposed).**
- **Pipeline (ADR-0035).** No new component: JetStream + the outbox + judge's Postgres `SKIP LOCKED` queue. **Five code changes ship dark in M1 (N0):**
  - `internal/platform/events/topology.go` as the single source of truth for streams and consumers, with golden-ACL, Σ `MaxBytes` budget and subject-registry tests;
  - a dead-letter hook (an `event_dead_letter` row, `Term()`, an ERROR log);
  - identity publishing `XLEARN_IDENTITY` (erase needs it);
  - nkey client options (all optional env);
  - pinned `pgxpool` `MaxConns` (4 per service, 8 for judge).

  A new `XLEARN_COACH` stream (128 MiB) carries coach's erase ack, so Σ `MaxBytes` = 3.375 of the 3.75 GiB ceiling.
- **NATS auth (ADR-0035; corrects ADR-0030 A5).** nkey users in `$G`, **server-first** with a `no_auth_user: legacy` bridge: N0 code → N1 users + bridge (the one restart) → N2 seeds per service → N3 legacy denied → N4 bridge removed (a reload). **Fine ACLs rendered from `topology.go` in the same step**, proven by a compose test against the rendered block before N2. Public keys plaintext in `messaging` values; seeds SOPS in `apps/secrets`; **no SOPS decryption in `messaging`**; the ops seed stays offline (break-glass: `ssh` port-forward plus the `nats` CLI). **N3 + MI-5 gate M3 and erase.** A new stream or consumer needs its infra ACL PR merged before the service tag.
- **Limits (ADR-0035).** A limits inventory with the gateway and identity controls at M1b, judge's at M3 and platform AI's at M4. **Kubelet L23** (`system-reserved` 1 GiB, `eviction-hard memory.available<500Mi`) joins the October host window.
- **Capacity (ADR-0035).**
  - **Memory-sum rule:** Σ limits + Σ p95 working set of limitless containers + the largest rollout surge + host ≤ capacity − 0.5 GiB. **MI-11a** makes it hold before the runner.
  - **What binds first:** platform AI at 13–17 seats; CPU only at ≈ 95 learners active in the same hour.
  - **Triggers** TR-* are measured by the owner on demand (no alerts). **Responses:** R0 tune → R1 KVM 8 (≈ +$21/month) → R2 a KVM 2 runner VPS as its own k3s + Flux ($8.99 → $14.99/month; never the backup VPS, never a k3s agent) → R3 out of scope.
  - Steal is elevated and noisy: 9-day p95 5.2%, 7.1% on 9/24.
- **Gating (ADR-0034).** T-1 build-time manifest defaults · T-2 env kill switches · T-3 the `account.role` cohort (owner, tester). No flag service; ≤ 6 live non-kill flags.
- **Migrations (ADR-0034).** Expand/contract with a `-- xlearn:contract floor=vX.Y.Z` marker; never `Down` in production; consumers one tag before producers (or the producer dark behind a T-2 switch), plus the subject-registry test; the event envelope is append-only.
- **Rollback (ADR-0034).** R-a kill switch → R-b narrow the ImagePolicy → R-c revert + patch tag → R-d Hostinger snapshot, **as an ordered procedure**: pin git back first, confirm the tags match the snapshot, list the erases completed since the snapshot, restore, `host-verify --cluster`, re-run the listed erases through the CLI, and record the lost-writes window. A Hostinger manual snapshot right before any contract, erase or GA tag. **No staging namespace.**
- **Rollout.**
  - The **MI track interleaves** with product work and is **hard-gated before M3**.
  - Chart 0.3.0 is **one PR** with the full knob union (0.2.2 today).
  - Pilot course: **go-concurrency**, confirmed at P entry (PRD Q5 stays open with this recommendation).
  - Artboards are **hybrid**: the owner designs the 5 heroes; agents draft the rest as static HTML on `theme.css` under `design-system/screens/v2/` for owner review. A milestone's boards freeze before its first UI sprint.

**Milestone map** (the same numbers as [rollout-plan.md](rollout-plan.md)).

| Milestone | Goal | Sprints | Earliest (inferred) |
|---|---|---|---|
| **M0** ✅ | Namespace rule; coach P0 fixes (v1.5.0, v1.5.1) | done | done |
| **MI** | Infra track, interleaved; hard gate before M3. MI-0 = the H0 reboot on Fri 2026-09-25. Alerting items dropped: no host-check timer; MI-8 = the on-demand `host-verify --cluster` extension | 6–8 | Sep 25 → Nov (spike week mid-Oct → October host window → runner dark in Nov) |
| **M1** | Spine, no behaviour change, plus the security floor (limits, `RequireRole`, admin CLI verbs, N0; entry gate: the MI-2a release guard, done) | 6–7 | October |
| **M2** | Attempt engine, projections, D2, `public-read` | 4–5 | late October |
| **M3** | Judge + code grader (Go, C++, Python) | 13–16 | November |
| **P** | Pilot course (go-concurrency) | 3–4 | December |
| **M4** | Platform AI (owner cohort) | 6–8 | December |
| **L** | Learner gate: invites, acceptance, notice, erase, tester role (in parallel with M3/M4) | 3–4 | with M3/M4 |
| **v2.0.0** | Owner-facing GA: MI + M1–M4 + P + L | **≈ 41–52 in total** | **≈ Dec 2026–Jan 2027** |
| **M6a + M6b** | Text, then voice interviewer → **v2.1.0** | 5–7 + 4–5 = **9–12** | Q1 2027 |
| M5 | DSA evaluator-only | 1 | a later 2.x, after the last DSA pack is stamped |
| M6c | Interviewer extras | 3+ | v2.2 |
| **Opening** | Real learners by invite | — | **v3** (the owner's call) |

- **A sprint ≈ one focused session** (v1 ran S01–S12 in two calendar days), so engineering is rarely the long pole. Owner go-aheads, the spike, the host window and content hours are.
- **Owner hours gating v2.0:** 14 pilot packs (28–41 h), pilot-course content (10–20 h), the M4 analyzer acceptance set (5–10 h), hero artboards and reviews (15–25 h).
- **After GA, D6's waves continue:** the week 1–4 packs (~35 in total) and the full 151-item self tier. M5 needs +230–340 h.

**Supersedes / amends.**
- **T1 (dependency matrix):** the `xlearn-opscheck` CronJob + push-channel adoption → deferred (D34).
- **T5:** "a push channel judge can call" before M4 → dropped; spend protection is the provider console's limits plus judge's in-app caps. T5's "before learners" items (notice, 18+, dogfood re-measurement) now gate the v3 opening; the notice and 18+ still ship with L (D35).
- **T6:** "opscheck interview counters" → dropped. The release-runbook check for live interviews stays.
- **Every other `opscheck` mention in T1–T6** (e.g. T1's opscheck plus badges, T2's disk and volume checks, T5's opscheck AI digest) is superseded by D34: in-app state, Postgres rows, CI checks and the on-demand `host-verify --cluster` instead.
- **ADR-0030 §5 A5:** "coarse", "clients first" → server-first with fine ACLs in the same step; no SOPS in `messaging`; L23 joins the October window; the chart is 0.2.2 today.
- **ADR-0027:** the opscheck PAT-expiry alert → evalpack CI plus a manual check; Σ `MaxBytes` 3.375 GiB with `XLEARN_COACH`; D6's "v2.0 content scope" (§8) → the v2.x line's scope, in waves (D35).
- **T5/T6 "M1: the coach P0 fixes"** → shipped in v1.5.1 (F010, xlearn#49: the onboarding key is saved; typed provider errors, so only auth errors disable a key; a 4,096-token reply cap with cut-off replies marked). M1 keeps the P1 items (`store:false`, AEAD binding, keyring, catalog) and `ErrModelAccess`, as in [rollout-plan.md](rollout-plan.md) §4.
- **ADR-0028:** the opscheck disk and volume watch → PVC usage in the on-demand `host-verify --cluster`.
- **ADR-0031:** the breaker's push alert → in-app state plus the provider console; the "before learners" items → the v3 opening.
- **ADR-0021 and ADR-0009:** refined by ADR-0034 (bounded ranges, `.release-line`, `-rc` prereleases, the GA range flip, no staging; the API stays v1).
- **ADR-0014:** stream config moves into `topology.go` with limits; "no auth" ends at N3 (ADR-0035).
- **ADR-0023 §3:** amended by the v1.5.2 auto-link fix (xlearn#51; no auto-link into password accounts), referenced in ADR-0033.
- **ADR-0024:** the enforced `public-read` role and enrolled ∩ visible courses (ADR-0033); mock count only (ADR-0032, D31).
- **ADR-0006:** roles enforced; `account.role` in the DB, never in the JWT (ADR-0033).
- **ADR-0016:** the unauthenticated internal endpoints are fenced by the `xlearn` ingress NetworkPolicy (MI-5a); still no service tokens (ADR-0033).

**What T7 hands to the build-plan session** (detail: [rollout-plan.md](rollout-plan.md) §12).
- Build `docs/v2/build-plan.md` and `docs/v2/status.md` from the rollout plan: the milestone table, the MI table, **milestone → tag → rollback floor**, the flag inventory, and a content status table.
- Every sprint lists its entry gates as checkboxes and names the repo each task touches; the M3 hard entry checklist is copied verbatim.
- Owner-gated items are calendar events, not sprints: the H0 reboot on 9/25 (copy the S0 log off `/tmp` first), the spike week, the October host window, the WIF spike, S6, the GA snapshot.
- Separate tasks outside the build plan, **all done 2026-09-24:** the auto-link fix (MI-2b, xlearn#51) and the `SIGNUP_MODE` patch (MI-2c, xlearn#52 + infra#30), both live in v1.5.2; the release-line guard (MI-2a, xlearn#53 + infra#29). The `DEV_AUTH` guard on `open` stays for M1b.
- Rewrite `docs/git-strategy.md`: rollback R-a → R-d, the `-rc` convention, the runner and evalpack streams, "image before HelmRelease or policy", "infra ACL PR before a new consumer" (`.release-line` and the GA steps already landed with xlearn#53).
- Move ADRs 0026–0035 from Proposed to Accepted at sign-off.

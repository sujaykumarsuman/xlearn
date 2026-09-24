# Prompt — Sprint m5-01 · DSA evaluator-only flip (M5)

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-m5-01.md`](../sprints/sprint-m5-01.md)   ·   **Milestone:** M5 (DSA evaluator-only, a later 2.x minor)   ·   **Prereqs:** [ga-02](../sprints/sprint-ga-02.md) (`v2.0.0` live) · on `main`: [m3-08](../sprints/sprint-m3-08.md), [m3-09](../sprints/sprint-m3-09.md), [m3-11](../sprints/sprint-m3-11.md), [m3-12](../sprints/sprint-m3-12.md), [m1-01](../sprints/sprint-m1-01.md) · the frozen AB07 ([ds-m3-01](../sprints/sprint-ds-m3-01.md)) · owner event `ev-dsa-packs-all` · ride path only: [m6b-03](../sprints/sprint-m6b-03.md) tagged, [m6b-04](../sprints/sprint-m6b-04.md) not yet

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md): conventions and land-and-sync.
- The plan: [`../sprints/sprint-m5-01.md`](../sprints/sprint-m5-01.md). It spells out the enforcement table, the variant frames and copy, the badge mapping and the release paths.
- **ADRs:**
  - [ADR-0034](../../adr/0034-v2-release-labelling-gating-and-rollback.md): **§1.1** (M5 = a later 2.x minor; it rides `v2.1.0` only if ready; the minor moves only at a GA flip), §1.6 (tag timeline), **§2** (T-1 per-course `allowed | evaluator_only`; the grading override is a permanent T-2 kill switch; presence by config), §4.1 and **§4.4** (M5 row: "the env override brings the picker back", residue none), **§6** (release checklist).
  - [ADR-0026](../../adr/0026-per-course-extensibility-model.md) §1 (the manifest holds the self-report policy) and §6 (M5).
  - [ADR-0029 §3](../../adr/0029-judge-contract-and-learning-signal.md#3-conclusion-practice-the-single-writer): practice is the single writer; hard limit; give-up = Miss; a pass finishes.
  - [ADR-0027 §2](../../adr/0027-content-evalpack-and-user-data-model.md#2-where-content-lives-and-how-it-ships): `contract_hash`; mismatch → self-report today.
  - [ADR-0035](../../adr/0035-v2-operations-nats-auth-limits-capacity.md) §2 (standing NetworkPolicy rule), §3 (D34: no alerting), §5 (memory sum).
- **Research:**
  - [t0 §4](../research/t0-extensibility-frame.md#4-universal-vs-per-course-method) (the "Grading mode and self-report" row) and [§9](../research/t0-extensibility-frame.md#9-how-live-dsa-migrates) (the M5 step: "for outcomes. Give-up means miss");
  - [t1 §3.3](../research/t1-content-data-model.md#33-private-eval-pack) (judge per-item status `ok | spec_mismatch | unsupported | invalid`) and [§3.4](../research/t1-content-data-model.md#34-versioning-and-pinning) (mismatch rule 1: "blocks counted attempts if the course is `evaluator_only`");
  - [t4 §3.3](../research/t4-judge-contract.md#33-transitions) (the `self_grade_pending` triggers), [§3.4](../research/t4-judge-contract.md#34-give-up-which-for-evaluated-items-includes-reveal-full-solution), and [§13](../research/t4-judge-contract.md) D15/D16/D18.
- **Plan docs:** [rollout §3](../rollout-plan.md#3-milestone-map) (M5 row), [§4 M5](../rollout-plan.md#m5-later-2x-and-m6c-v22), [§7](../rollout-plan.md#7-indicative-tag-timeline), [§8](../rollout-plan.md#8-what-ships-where-content-hours-the-d6-reading); PRD [R-OL1](../../prd/xlearn-v2-prd.md#52-amended-requirements).
- **Peer plans:**
  - [m3-08](../sprints/sprint-m3-08.md) tasks 3, 5 and 7: the judged start, INV-9, `GRADING_OVERRIDE`, the kill-switch sweep, `self_grade_pending`.
  - [m3-09](../sprints/sprint-m3-09.md) tasks 1 and 7: the `judge` block, `/api/judge/status`, the DTO allowlist.
  - [m3-11](../sprints/sprint-m3-11.md) tasks 1–2: `ProblemRoute`, the Workspace frames.
  - [m3-12](../sprints/sprint-m3-12.md) task 4: `JudgeBadges.tsx`.
  - [m3-14](../sprints/sprint-m3-14.md): drafts, `judge admin status`.
  - [m1-01](../sprints/sprint-m1-01.md) task 2: the `grading.self_report` block and the golden rule.
  - [m3-13](../sprints/sprint-m3-13.md): the kill-switch drill runs in compose.
  - [m6b-04](../sprints/sprint-m6b-04.md): the `v2.1.0` checklist, for the ride path.
- **Artboards** (frozen, reference only): `design-system/screens/v2/AB07-workspace-code.html` (F1, F11, F12, F14), `AB08-results-dock.html` (F15) and `AB11-degradation-badges.html` (F1–F4); [`../../../design-system/theme.css`](../../../design-system/theme.css).
- **Code:**
  - `curriculum/courses/dsa/course.json`, `internal/course/{manifest.go,validate.go,golden_test.go}`;
  - `internal/practice/{config.go,handlers.go,service.go,judgeconsumer.go,ticker.go}`, `internal/practice/store/{store.go,migrations/,queries/}`, `internal/platform/judgeapi/`;
  - `internal/gateway/{judge.go,judge_dto.go,judge_status.go}`, `docs/architecture/openapi.yaml`;
  - `web/src/router.tsx`, `web/src/screens/Workspace.tsx`, `web/src/screens/workspace/*`, `web/src/components/judge/{JudgeBadges.tsx,ResultsDock.tsx}`, `web/src/screens/{Problems,Dashboard}.tsx`, `web/src/lib/judge.ts`, `web/src/test/fixtures/judge/`;
  - `internal/e2e/`, `docker-compose.yml`;
  - `docs/architecture/{api,services,data-model}.md`, `docs/v2/runbooks/practice-judge-lifecycle.md`.
- **Infra** (read-only here; there's no infra PR in this sprint): `../infra/apps/xlearn-practice.yaml` (the env block the R-a recipe edits).

## Context

v2.0 GA (`v2.0.0`) turned judge on for every account, but DSA still honours self-report: an item with no pack or a `spec_mismatch` falls back to the four-grade picker (m3-11's self-path variant, AB11's "Grading pending · self-report"). The D6 content waves and the owner's pack work (`ev-dsa-packs-all`, +230–340 h) have now packed every live DSA item.

**This sprint flips DSA to evaluator-only for outcomes**, a T-1 build-time default labelled by a 2.x minor (riding `v2.1.0` if ready by then, else its own ≥ `v2.2.0`):
- a counted DSA attempt is **always judged or doesn't start**;
- a paused item shows a badge instead of the picker;
- **give-up = Miss** everywhere;
- `self_grade_pending` becomes `voided` (nothing recorded, no self pick);
- **the grading override (`GRADING_OVERRIDE=self`) brings the picker back** as the permanent R-a kill switch.

`touch` and `mock` self-report stay `allowed`. Other courses are unchanged. Production has the owner and CLI-minted testers only (D35).

## Entry gates — verify first (stop and report if any is unmet)

- [ ] **Every live DSA item packed and stamped** (`ev-dsa-packs-all`). The status.md content table shows DSA at 100%. On prod, judge's per-item `GET /internal/evaluable?path=dsa` shows every live (`status == "live"`) DSA item `ok` with `submit_available` (0 `no_pack` / `spec_mismatch` / `unsupported` / `invalid`) and `admission: open`. Write and run plan task 1's script to check it. A gap is a **stop**: report the ids.
- [ ] **`v2.0.0` live** ([ga-02](../sprints/sprint-ga-02.md)): `.release-line = 2`; the 8 fleet `xlearn-*` ImagePolicies are `>=1.0.0 <3.0.0`, while `xlearn-runner` and `xlearn-evalpack` stay `<2.0.0` (ga-02 leaves them alone); judge on for every account; `JUDGE_BASE_URL` set on the gateway and practice; `GRADING_OVERRIDE` unset.
- [ ] On `main`: m3-08, m3-09, m3-11, m3-12, m1-01 as listed in the plan's entry gates.
- [ ] The evalpack PAT expires more than 14 days from now (status.md `ev-pat-expiry`).
- [ ] **Release path decided** ([ADR-0034 §1.1](../../adr/0034-v2-release-labelling-gating-and-rollback.md#11-scheme): M5 "rides v2.1.0 if it's ready by then"). Ride `v2.1.0` if `v2.1.0` isn't tagged and all of these hold: m6b-03's `v2.0.x` patch is tagged, m6b-04 hasn't started its tag, and the owner confirms **no `v2.0.x` patch will be cut from `main`** before `v2.1.0`. Otherwise (including when `v2.1.0` is already tagged) take the own minor ≥ `v2.2.0`.
- [ ] No DSA item or contract change is waiting to go live without its stamped pack.
- [ ] **Parallel sessions:**
  - no open PR edits `curriculum/courses/dsa/course.json`, `internal/course/`, `internal/practice/`, `internal/gateway/judge*.go`, `web/src/screens/Workspace.tsx`, `web/src/screens/workspace/*`, `web/src/components/judge/*` or `web/src/router.tsx`;
  - the next free practice goose version and ADR number are known (`gh pr list`, `git worktree list`, `git ls-remote --tags origin`, ListAgents).

## Do this (in order)

1. **[X] Branch** `feat/m5-dsa-evaluator-only` from an up-to-date `main`.
2. **[X] Coverage check** (plan task 1): write `hack/m5-coverage-check.sh`.
   - **Live list:** `curriculum/courses/dsa/items/*/item.json` filtered with `jq 'select(.status == "live")'` (items can also be `retired` or `withdrawn`).
   - **Per-item source:** judge's `GET /internal/evaluable?path=dsa` (`{admission, pack{version}, items[{item_id, status, submit_available, …}]}`) through a background node-local port-forward (the mi-06/p-01 pattern): `ssh -L 18087:127.0.0.1:18087 vps 'k3s kubectl -n xlearn port-forward deploy/xlearn-judge 18087:8087'`, then `curl -s 'http://127.0.0.1:18087/internal/evaluable?path=dsa'`; `trap` kills the tunnel. Fallback: the owner-session `GET /api/judge/status?path=dsa` and its per-item `items{}` map.
   - **Count cross-check:** `ssh vps 'k3s kubectl exec -n xlearn deploy/xlearn-judge -- judge admin status --json'`. It has **no `--path` flag** and reports only counts by course and status (no item ids, no `no_pack`); like every `judge admin` verb it **writes an `admin_audit` row**, so call it once per run.
   - **Asserts:** every live id is in `items[]` with `status == "ok"` and `submit_available == true` (missing = `no_pack`); `admission == "open"`; the admin DSA `ok` count equals the live count with other DSA statuses 0; `pack.version` is the latest evalpack `v1.x`.
   - It also prints the informational list of open DSA course attempts pinned `self`.
   - Run it now and keep its output for the PR. It exits non-zero on any gap; if it does, **stop**.
3. **[X] The flip + effective policy** (plan task 2):
   - Set `grading.self_report.outcome` to `evaluator_only` in `curriculum/courses/dsa/course.json` (`touch`/`mock` stay `allowed`).
   - Update the DSA row of `internal/course/golden_test.go`, citing M5 (ADR-0026 §6, t0 §9, ADR-0034 §2 T-1).
   - Add a validation rule in `internal/course/validate.go`: `evaluator_only` requires `strategy ≠ self@1`.
   - Add `internal/practice/policy.go` with `EffectiveSelfReport(slug, signal) (value, source)`. The rules, first match wins: `GRADING_OVERRIDE=self` → `allowed`/`override`; practice's `JUDGE_BASE_URL` unset → `allowed`/`no_judge`; else the manifest value.
   - Add practice's `GET /internal/grading-policy?path=` → `{path, selfReport{outcome,touch,mock}, source}`.
4. **[X] practice enforcement** (plan task 3, its table), for DSA course attempts whose effective `outcome` is `evaluator_only`:
   - **Start:** an item that isn't `ok` → **409 `evaluation_pending{reason}`**. An item that is `ok` but `submit_available=false` (judge's L15 `JUDGE_ADMISSION=closed`, or judge's `RUNNER_BASE_URL` unset) → **503 `evaluation_unavailable{fallback:"none"}`** (the SPA shows V3). Judge unreachable, or no `judge: true` → the same 503. In every case there is **no attempt, timer or outbox row**; m3-08's "otherwise the v1 start runs unchanged" fall-through must be unreachable for a DSA course start under `evaluator_only`. Resume and pins are unchanged.
   - **Outcome:** `POST /problems/{id}/outcome` → **409 `evaluator_only`**, except for a pinned self attempt or an existing `self_grade_pending`.
   - **Reveal:** the solution reveal stays 409 `use_give_up` (give-up = Miss).
   - **The `self_grade_pending` triggers** (`contract_changed` with counted evals, `budget_exhausted`, the 3rd released close) go through one helper, `failOpenAttemptTx`. It chooses `voided` + `void_reason` under `evaluator_only`, and m3-08's `self_grade_pending` under `allowed`. The void is one guarded update: `state='voided', resolution='voided', void_reason, ended_at=now(), concluded_at=now() … WHERE concluded_at IS NULL`, plus closing the timer; no outcome, no outbox, `user_problem_state` untouched. `ended_at` keeps it closed to v1's `GetOpenAttempt` (`ended_at IS NULL`, the course-attempt open check m2-01 keeps for R-b safety); `concluded_at` keeps m2-01's startup heal and guarded `Conclude()` off it. The kill-switch sweep and touches are unchanged.
   - **`void_reason`:** add the expand-only migration, `void_reason text NULL` with **no CHECK**, validated by a closed Go enum (`contract_changed | budget_exhausted | released_limit`), like m2-01's `resolution` and m4-04's `provisional_reason` (next free goose version; reuse an existing reason column if M2a/m3-08 left one). Then the queries, `sqlc generate`, and **`sqlc diff` clean**. `voidReason` goes on `GET /state` and `GET /state?ids=`.
   - **Boot WARN:** a background goroutine iterates the embedded course registry and, for each manifest whose effective `outcome` is `evaluator_only`, calls `judgeapi.Client.Evaluable(slug)`; it logs `WARN evaluator_only_uncovered path=<slug> count=N ids=[≤10]` when any live item isn't `ok` with `submit_available`. No course-slug literal; it never touches the readiness path.
5. **[X] gateway** (plan task 4):
   - Read the policy through the practice client, cached 30 s. Keep the last value on error; with no value yet, default to the embedded manifest's `grading.self_report.outcome` for that path (fails closed for an `evaluator_only` course), never a `"dsa"` literal.
   - Add `selfReport` to the `judge` block on `GET /api/problems/{id}` and to `/api/judge/status`, and pass `voidReason` through the poll composition. Add all three to the `judge_dto.go` allowlist, with tests.
   - Set the 503 `fallback` from the effective policy (`self` iff `allowed`, else `none`), never from judge's hint.
   - Pass through 409 `evaluation_pending` and 409 `evaluator_only`.
   - Update `docs/architecture/openapi.yaml` + `docs/architecture/api.md`.
6. **[X] web: the AB07 evaluator-only variant** (plan task 5). Use `theme.css` verbatim, and draft the copy exactly as the plan's table.
   - The `ProblemRoute` rows in `web/src/router.tsx`: not evaluable + `selfReport=evaluator_only` → the paused variant; a `voided` latest attempt → the voided card.
   - New `web/src/screens/workspace/PausedCover.tsx` (V2/V3) and `VoidedCard.tsx` (V5: the cause line per `voidReason` from the plan's V5 table, then **Start again · keep my code**, which copies the voided context's latest draft through the drafts API).
   - Also: V1's "Every attempt is auto-graded."; V4's dock line on a 503 `fallback: none`.
   - The `JudgeBadges.tsx` mapping: "Grading paused" and the "Paused" row chip; the `evaluable = 0` banner in Problems and Dashboard.
   - Types in `web/src/lib/judge.ts`.
   - **The picker never renders for DSA while `selfReport = evaluator_only`.**
7. **[X] Kill-switch drill + runbook** (plan task 6). Compose uses m3-02's fixture pack:
   - baseline: fixture item judged, other items paused;
   - set `GRADING_OVERRIDE=self` on practice → new starts pin `self` with the **picker**, and open judged attempts → `self_grade_pending` within 30 s, **never Miss**;
   - unset → evaluator-only again, **nothing re-graded**;
   - record the times.

   Add the "Evaluator-only (M5)" section to `docs/v2/runbooks/practice-judge-lifecycle.md`: the R-a infra-PR recipe on `../infra/apps/xlearn-practice.yaml`, when to use it, the coverage script, the voided-attempts SQL, and the **pack-first content rule**. No prod drill unless the owner asks.
8. **[X] Tests + verify** (plan task 7):
   - the policy table; refused starts write nothing, one case per refusal (409 per non-`ok` status; 503 for `ok` + `submit_available=false` with fake judges for admission closed and runner lane off; 503 for judge unreachable / no `judge: true`); outcome 409; the triggers → `voided` under `-race` (consumer + ticker + close);
   - a voided course attempt has `ended_at` + `concluded_at`: `GetOpenAttempt` skips it, the next start is fresh, `Conclude()` is a no-op, and a restart's heal leaves it `voided`;
   - **the route sweep:** no route writes a `graded_by=self` DSA course outcome or pins `grading_mode=self` on a DSA course start under `evaluator_only`, pins aside, driven with every fake-judge state above;
   - the gateway DTO allowlist, `fallback` rewrite and cache (no-value default from the embedded manifest; the literal gate green);
   - Vitest for V1–V6, one V5 case per `voidReason`, and the `ProblemRoute` table (no picker);
   - compose e2e: pass → Clean, give-up → Miss, timeout → Miss, paused → 409 + V2, the drill.

   Then run `gofmt`, `go vet`, `go test -race ./...`, `npm --prefix web run test`, `-tags e2e`, the migration lint and **`sqlc diff`**.
9. **[X] Docs + ADR** (plan task 8):
   - The ADR at the **next free number** (check peers first), Accepted on merge: "DSA evaluator-only (M5): the effective self-report policy and the no-self lifecycle".
   - One-line "Amended by" notes in ADR-0027 and ADR-0029.
   - `docs/architecture/{api,services,data-model}.md`.
10. **[X] PR with the variant screenshots** (plan task 9):
    - Take V1–V6 at 1440 px and 390 px, beside the frozen AB07 F1/F12 crops. Use a throwaway vite mock config (never committed) or compose.
    - Commit them as `design-system/screens/v2/shots/AB07-m5-V<n>@1440.png` / `@390.png` (≲ 500 KB each).
    - The PR body carries the frame and copy table, the coverage output, the drill times, and **"Decisions to confirm"**: `touch`/`mock` stay `allowed`; `no_judge` → picker; every former `self_grade_pending` trigger → `voided` (contract change mid-attempt, the third infra-released close, budget exhausted), each with its V5 cause line; an `ok` item with its submit lane off → refused (V3), not self; pre-flip self attempts honoured; the release path.
    - Conventional commit(s) ending with the attribution lines.
11. **[O] Owner approval: STOP until it arrives.** The owner approves the variant screenshots and copy in the PR (BP3: this stands in for a design freeze). **Do not merge without explicit owner approval.** If it doesn't come this session, leave the PR open, set task 10 ⛔ "awaiting owner review", and end the session.
12. **[X] Release** (plan Release), after the approval and green CI. ADR-0034 §1.1: ride `v2.1.0` if ready and every ride condition holds; otherwise the own minor.
    - **Ride `v2.1.0`** (only under the entry-gate conditions):
      1. Re-run `hack/m5-coverage-check.sh` and the PAT > 14-day check.
      2. Squash-merge **untagged**.
      3. Record "rides `v2.1.0`" in status.md, plus a Decisions-log hand-off for m6b-04: **right before tagging `v2.1.0`, re-run the coverage check and the PAT > 14-day check; if either fails, hold M5 (a PR reverting the DSA manifest line and its golden row to `allowed`) and tag without it; otherwise list M5 in the notes and run the M5 smoke after its verify.** The flip goes live only at that tag.
      4. **Never tag `v2.1.0` here.**
    - **Own minor (otherwise):**
      1. Squash-merge.
      2. Re-run `hack/m5-coverage-check.sh` and the PAT check.
      3. Walk the release checklist below.
      4. Tag the **next free minor after `v2.1.0`** (≥ `v2.2.0`; major = `.release-line`), titled `v2.N.0 — DSA evaluator-only (M5)`, with the release notes listing `git log <last-tag>..main`.
      5. Verify by looking, then run the M5 smoke.

## Constraints

- **Service boundaries ([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)):**
  - practice owns the policy and its enforcement (the single writer, ADR-0026 §3);
  - the gateway only mirrors practice's effective value;
  - judge is unchanged;
  - no cross-schema reads;
  - the client never decides grading, and the raw manifest's `self_report` is **not** the effective policy.
- **goose + sqlc:** the `void_reason` migration is expand-only (no `xlearn:contract` marker), takes the next free version at rebase, and commits its generated code. **`sqlc diff` must pass.**
- **Outbox/inbox:** a voided attempt writes **no** outbox row. A refused start writes nothing. The guarded `Conclude()` and `UNIQUE(outcome.attempt_id)` stay the only conclusion path.
- **Consumers before producers:** n/a. There is no new subject or event field (`problem_solved`/`touch_concluded` are unchanged), so no NATS ACL PR.
- **theme.css verbatim;** match the frozen AB07/AB08/AB11. The variant is derived from them, never a redesign. Never edit the frozen boards, `index.html` or `board.css`.
- **GitOps:** never `kubectl apply`. `ssh vps` stays read-only, apart from the coverage script's node-local `port-forward` to judge (the mi-06/p-01 pattern) and one sanctioned admin-CLI `kubectl exec … judge admin status --json` per run (it writes an `admin_audit` row, like every `judge admin` verb). The R-a recipe is an infra PR, documented but not executed here.
- **Course literals:** m1-03's `hack/lint-course-literals.sh` (CI) rejects `"dsa"` in `internal/` and `web/src/` outside its allowlist. The gateway's no-value default, the boot WARN and every policy branch key off the course registry / manifest, never a slug literal.
- **D34:** no alert, no ping, no opscheck. The boot WARN and the in-app badges are the signal.
- **Memory-sum rule ([ADR-0035 §5](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#5-capacity-the-memory-sum-rule-triggers-and-ordered-responses)):** no new pod; limits unchanged.
- **NetworkPolicy standing rule:** no new in-cluster caller (gateway → practice exists; `/internal/grading-policy` is in MI-5a's `/internal/*` family), so no infra PR. Confirm it and write "n/a" in the checklist.
- **Release labels ([ADR-0034 §1.1](../../adr/0034-v2-release-labelling-gating-and-rollback.md#11-scheme)):**
  - M5 rides `v2.1.0` if it's ready by then and the ride conditions hold, otherwise gets its own minor (≥ `v2.2.0`);
  - **never cut `v2.1.0` for M5 alone**;
  - never let the flip ship in a `v2.0.x` patch;
  - never move or re-push a tag.
- **Parallel sessions:** check peers' PRs, tags, worktrees and ListAgents before numbering the migration, claiming the ADR number, or tagging. **Before the tag: no live interviews** (`coach admin interviews --live` empty).
- **Owner review gate:** no merge before the owner approves the variant (step 11). It is the only exception to land-and-sync in this sprint.

## Deliverables

- `hack/m5-coverage-check.sh` and its output (in the PR).
- The DSA manifest flip; the golden update; the validation rule; `internal/practice/policy.go`; `GET /internal/grading-policy`.
- practice enforcement: 409/503 starts, 409 outcome, `voided` + `void_reason` (migration + sqlc), `voidReason` on `/state`, the boot WARN.
- gateway: `selfReport` in the `judge` block and `/api/judge/status`, the `fallback` rewrite, the DTO allowlist, openapi.
- web: `PausedCover.tsx`, `VoidedCard.tsx`, the Cover/dock/badge/banner edits, the `ProblemRoute` rows, Vitest + fixtures.
- The compose override drill; the runbook section; the ADR; the api/services/data-model docs.
- Variant screenshots under `design-system/screens/v2/shots/AB07-m5-*`.
- A merge that rides `v2.1.0` (if ready and the ride conditions hold), or the tag `v2.N.0` (≥ `v2.2.0`).

## Update status

- [`../sprints/sprint-m5-01.md`](../sprints/sprint-m5-01.md): each task 🔄 → ✅ (task 10 ⛔ while it awaits the owner); _Overall_ ✅ once released (own minor) or merged to ride.
- [`../status.md`](../status.md):
  - **Sprint board** row m5-01;
  - **Milestones:** M5 ✅ with `v2.N.0`, or "rides `v2.1.0`" until m6b-04 verifies;
  - **milestone → tag → floor → snapshot:** `M5 → v2.N.0 → floor unchanged → snapshot n/a` (`void_reason` is expand-only; on the ride path the row is m6b-04's `v2.1.0`);
  - **flag inventory:** the grading override (permanent kill switch) "also restores the DSA picker since M5"; the T-1 line "DSA `self_report.outcome = evaluator_only`". No new flag;
  - **content table:** "DSA 100% packed", plus the **pack-first rule** for any later DSA item or contract change;
  - **Artboards:** the AB07 row "M5 evaluator-only variant approved (PR #, date)".
- **Decisions log:**
  - the effective-policy rules (`no_judge` → `allowed`);
  - start refuses rather than falling back;
  - `self_grade_pending` → `voided` under `evaluator_only` (all three triggers; the void sets `ended_at` + `concluded_at`);
  - pins honoured;
  - `touch`/`mock` stay `allowed`;
  - the drill timings;
  - the release path taken (and, if riding, the no-`v2.0.x`-patch commitment plus the m6b-04 hand-off).
- The ADR: the lifecycle mapping (next free number, after checking peers).

## Done when (acceptance)

- [ ] **M5 exit: no self path left in DSA.** With judge present and the override unset:
  - no DSA course attempt pins `self` (it is judged, or refused with 409/503 and no rows);
  - outcome → 409 `evaluator_only`, pins aside;
  - no `self_grade_pending`, only `voided`;
  - no picker in the SPA;
  - give-up = Miss.

  The route sweep, Vitest and compose e2e are green.
- [ ] An item with `spec_mismatch` blocks counted attempts with the "Grading paused" badge; the arena still works.
- [ ] The override restores the picker within minutes: the compose drill passes, nothing is re-graded, and the times are recorded; the R-a recipe is in the runbook.
- [ ] The golden is updated with its citation, the validation rule is tested, `sqlc diff` is clean, and CI is green.
- [ ] The owner approved the AB07 evaluator-only variant in the PR; the screenshots are committed.
- [ ] The coverage check and the PAT > 14-day check are green right before the tag that makes the flip live: this sprint's own tag, or, on the ride path, at merge time and again by m6b-04 right before `v2.1.0` (the hand-off is recorded).
- [ ] Released: `v2.N.0` verified by the checklist plus the M5 smoke (no picker on DSA; `/api/judge/status?path=dsa` shows `selfReport: evaluator_only`). Or merged to ride `v2.1.0`, with the hand-off recorded.

Release checklist for the tag ([ADR-0034 §6](../../adr/0034-v2-release-labelling-gating-and-rollback.md#6-release-checklist) + the ADR-0035 §2 standing rule), on the own-minor path:
- [ ] Before the tag: peers' tags and PRs are checked (parallel sessions; `git ls-remote --tags origin`, `gh pr list`, `git worktree list`, ListAgents)
- [ ] Before the tag: it is the next free version, and its major equals `.release-line`
- [ ] Before the tag: ACL PRs for new streams and consumers are merged
- [ ] Before the tag: a new service's image comes before its policy
- [ ] Before the tag: for a contract: rehearsed in compose, floor marked
- [ ] Before the tag: for a contract, erase or GA tag: `host-verify --cluster` is green (ADR-0035), the host has settled, and the snapshot is taken
- [ ] Before the tag: from M6: no live interviews
- [ ] After the tag (by looking, D34): `/xlearn/api/v1/healthz` reports the version
- [ ] After the tag: `k3s kubectl get deploy -n xlearn` shows the new images
- [ ] After the tag: every `xlearn-*` ImagePolicy's latest equals the tag, and the HelmReleases are Ready
- [ ] After the tag: smoke-test login, the dashboard and coach
- [ ] Record milestone → tag → floor → snapshot and any flag changes in `docs/v2/status.md`
- [ ] (ADR-0035 §2 standing rule, not part of ADR-0034 §6) Every new in-cluster HTTP or NATS caller this tag introduces has its NetworkPolicy (ingress and egress) change in its own infra PR, merged before the tag

Here, ACL, new service, contract and NetworkPolicy are all n/a. A snapshot isn't required (not a contract, erase or GA tag; R-a residue none). **No live interviews applies.**

Ship per AGENT.md land-and-sync with **this sprint's release action** (ADR-0034 §1.1: M5 "rides v2.1.0 if it's ready by then"):
- **Ride:** merge untagged to ride `v2.1.0` when it is ready before m6b-04 and every ride condition holds, with the m6b-04 pre-tag re-check hand-off recorded; never cut `v2.1.0` for M5 alone.
- **Own minor (otherwise):** tag the next free minor after `v2.1.0` (≥ `v2.2.0`).

In both cases the merge happens **only after the owner approves the AB07 variant** (step 11). Without that approval, leave the PR open and stop. Afterwards run `git checkout main && git pull` in xlearn. There is no infra PR, so `../infra` needs only a `git pull`.

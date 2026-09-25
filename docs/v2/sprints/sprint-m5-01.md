# Sprint m5-01 — DSA evaluator-only flip (M5)

> **Milestone:** M5 — DSA evaluator-only (the whole milestone is this one sprint) · **Track:** product · **Order:** 86
> **Prereqs:** [ga-02](sprint-ga-02.md) (`v2.0.0` live: judge on for every account, `.release-line = 2`, ranges `<3.0.0`) · on `main`: [m3-08](sprint-m3-08.md) (judged start, INV-9, `GRADING_OVERRIDE`, kill-switch sweep, `self_grade_pending`), [m3-09](sprint-m3-09.md) (the `judge` block, `/api/judge/status`), [m3-11](sprint-m3-11.md) + [m3-12](sprint-m3-12.md) (Workspace, badges), [m1-01](sprint-m1-01.md) (manifest `grading.self_report` + golden) · the frozen AB07 ([ds-m3-01](sprint-ds-m3-01.md)) · owner event `ev-dsa-packs-all` (every live DSA item packed) · for the ride path only: [m6b-03](sprint-m6b-03.md) tagged, [m6b-04](sprint-m6b-04.md) not yet
> **Unblocks:** — (no later sprint depends on M5; the [m6c-01](sprint-m6c-01.md)…[m6c-03](sprint-m6c-03.md) outlines are independent of it)
> **Release action:** per [ADR-0034 §1.1](../../adr/0034-v2-release-labelling-gating-and-rollback.md#11-scheme) ("it rides v2.1.0 if it's ready by then"): **merge untagged so it rides `v2.1.0`** if it's ready before [m6b-04](sprint-m6b-04.md) and every ride condition in Release holds; **otherwise tag the next free minor ≥ `v2.2.0`** (the expected path on the calendar: M5 ≈ H2 2027, `v2.1.0` ≈ Q1 2027). **Never cut `v2.1.0` for M5 alone.** No infra PR.
> **Artboards:** AB07★ **evaluator-only variant**, derived from frozen AB07 F1/F11/F12 and AB11 F1–F4. No separate design sprint and no new board file: the variant screenshots committed with the PR are its reference, and the merge freezes the variant (D40: it lands as drafted; the owner may revise it later with a follow-up PR).
> **Calendar:** ≈ H2 2027, after `ev-dsa-packs-all` (+230–340 owner h of packs, [rollout §8](../rollout-plan.md#8-what-ships-where-content-hours-the-d6-reading)) · no owner time in-session: the variant lands as drafted and the session verifies the tag (D40); the owner may review both afterwards
> **Execute with:** [`../prompts/prompt-m5-01.md`](../prompts/prompt-m5-01.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | Pre-flip coverage check `hack/m5-coverage-check.sh` (every live DSA item `ok` on prod) | X | ⬜ |
| 2 | The flip: DSA `grading.self_report.outcome = evaluator_only` + golden + validation; practice's effective policy | X | ⬜ |
| 3 | practice enforcement: no self start, outcome 409, `self_grade_pending` → `voided`, boot WARN, `/internal/grading-policy` | X | ⬜ |
| 4 | gateway: `selfReport` in the `judge` block and `/api/judge/status`; 503 `fallback` from the effective policy | X | ⬜ |
| 5 | web: the AB07 evaluator-only variant (paused cover, voided card, no picker) + the AB11 badge mapping | X | ⬜ |
| 6 | Kill switch: the override restores the picker (compose drill) + runbook section | X | ⬜ |
| 7 | Tests + verify | X | ⬜ |
| 8 | Docs + ADR (the evaluator-only lifecycle) | X | ⬜ |
| 9 | PR with the variant screenshots | X | ⬜ |
| 10 | Release: ride `v2.1.0` if ready and the ride conditions hold, else tag ≥ `v2.2.0`; verify, record | X | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (the Sprint board row, the **M5**
> Milestones row, milestone → tag → floor → snapshot, the flag inventory, the content table, the AB07 Artboards row and the Decisions log).
> Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] **Every live DSA item packed and stamped** (owner event `ev-dsa-packs-all`). The [`../status.md`](../status.md) content table shows DSA at 100% full-pack tier. On prod, judge's per-item `GET /internal/evaluable?path=dsa` (task 1's source) reports **every live (`status == "live"`) DSA item `ok` with `submit_available`**: 0 `no_pack`, 0 `spec_mismatch`, 0 `unsupported`, 0 `invalid`, `admission: open`. Task 1's script checks this and exits non-zero on any gap.
- [ ] **`v2.0.0` live** ([ga-02](sprint-ga-02.md)): `.release-line = 2`; the **8 fleet `xlearn-*` ImagePolicies** are `>=1.0.0 <3.0.0`, while `xlearn-runner` and `xlearn-evalpack` stay `<2.0.0` (separate streams, untouched by ga-02); judge is on for every account (the T-3 cohort gate was removed at GA by [ga-01](sprint-ga-01.md)); `JUDGE_BASE_URL` is set on the gateway and on practice; `GRADING_OVERRIDE` is **unset**.
- [ ] On `main`: m3-08's judged start, INV-9, `GRADING_OVERRIDE`, the kill-switch sweep, and the `self_grade_pending` and `voided` states; m3-09's `judge` block and `GET /api/judge/status`; m3-11's `ProblemRoute` and `Workspace.tsx`; m3-12's `JudgeBadges.tsx`; m1-01's `grading.self_report{outcome,touch,mock}` with its golden test.
- [ ] **The evalpack pull PAT doesn't expire within 14 days** (the date in status.md, `ev-pat-expiry`). An expired PAT plus a judge restart leaves judge with 0 evaluable items. After this flip that pauses every counted DSA attempt until the override is set.
- [ ] **Release path decided** (see Release): if `v2.1.0` isn't tagged yet and every ride condition holds, ride it ([ADR-0034 §1.1](../../adr/0034-v2-release-labelling-gating-and-rollback.md#11-scheme)). If `v2.1.0` is already tagged, or any ride condition fails, take the own-minor path.
- [ ] **No DSA item waiting to go live without its pack.** Every DSA item or contract change merged since the last evalpack tag has its pack stamped. After this sprint, an item without a pack shows as paused.
- [ ] **Parallel sessions:** no open peer PR edits `curriculum/courses/dsa/course.json`, `internal/course/`, `internal/practice/`, `internal/gateway/judge*.go`, `web/src/screens/Workspace.tsx`, `web/src/screens/workspace/*`, `web/src/components/judge/*` or `web/src/router.tsx`. The next free practice goose version is known (`gh pr list`, `git worktree list`, ListAgents).

## Goal

Once every live DSA item has a pack, make DSA **evaluator-only for outcomes**:
- **No self path is left in DSA.** A counted DSA course attempt is always judged, or it doesn't start.
- **Give-up = Miss** everywhere in DSA.
- **An item with a `spec_mismatch` blocks counted attempts** and shows a badge instead of falling back to the self picker.
- **The grading override is the kill switch** that brings the picker back within minutes ([ADR-0034 §4.4](../../adr/0034-v2-release-labelling-gating-and-rollback.md#44-reversibility-by-step), M5 row: residue none).

The flip is a **T-1 build-time default** ([ADR-0034 §2](../../adr/0034-v2-release-labelling-gating-and-rollback.md#2-feature-gating-three-tiers-no-flag-service)): one manifest field, labelled by a 2.x minor, either `v2.1.0` (ride) or its own ≥ `v2.2.0` ([§1.1](../../adr/0034-v2-release-labelling-gating-and-rollback.md#11-scheme): "labels mark GA flips"). It realizes [ADR-0026 §6](../../adr/0026-per-course-extensibility-model.md#6-migration-the-v1-manual-loop-coexists-throughout) M5, [t0 §9](../research/t0-extensibility-frame.md#9-how-live-dsa-migrates) ("set DSA to `evaluator_only` for outcomes. Give-up means miss"), [t1 §3.4](../research/t1-content-data-model.md#34-versioning-and-pinning) mismatch rule 1 ("blocks counted attempts if the course is `evaluator_only`"), and PRD [R-OL1](../../prd/xlearn-v2-prd.md#52-amended-requirements) ("self-report stays available … until that course's items all have evaluators").

## Scope

**In**
- `curriculum/courses/dsa/course.json`: `grading.self_report.outcome` changes from `allowed` to **`evaluator_only`**. `touch` and `mock` stay `allowed`. The golden test is updated with its citation, and a validation rule is added.
- practice: the **effective policy** (manifest, the override, judge configured), enforced at start, at outcome and in the `self_grade_pending` transitions. Also a read-only `GET /internal/grading-policy` and a WARN at boot.
- gateway: the effective `selfReport` in the problem aggregate's `judge` block and in `/api/judge/status`, plus the 503 `fallback` value.
- web: the **AB07 evaluator-only variant** (the picker never renders for DSA while the policy is `evaluator_only`), plus the AB11 badge mapping for paused items.
- The override drill in compose, a runbook section, an ADR for the lifecycle mapping, and the api/services/data-model docs.
- The release: riding `v2.1.0` if ready and every ride condition holds, otherwise the next free minor ≥ `v2.2.0`.

**Out**
- **Other courses**: go-concurrency and any later course stay `allowed`. Each course flips in its own milestone, once its items are all packed.
- **`self_report.touch` and `self_report.mock`** stay `allowed`:
  - With every item packed, a judged touch's `correct_in_timer` is already checked evidence whenever judge is present ([m3-08](sprint-m3-08.md) task 4). The touch attestation fallback reappears only during a mismatch window or while the kill switch is on.
  - Flipping `touch` would make one mismatched item's due touch un-doable, and R-SR5 would then block all new DSA work.
  - The mock keeps `ScoreMock` and M6a's proposal/accept flow.
  - Both are a later owner call (Risks).
- **Removing the self path's code** (v1 `Problem.tsx`, `self@1`, m3-11's self-path variant). They stay, because they are the R-a kill-switch screens (ADR-0034 §2: the grading override is permanent).
- **A prod kill-switch drill.** The drill runs in compose ([m3-13](sprint-m3-13.md) precedent). A prod rehearsal is two infra PRs, done only if the owner asks.
- **Pack authoring** (the owner's content track, `ev-dsa-packs-all`), per-problem time budgets (PRD Q7), and M6c ([m6c-01](sprint-m6c-01.md)…[m6c-03](sprint-m6c-03.md)).

## Tasks

### 1 · Pre-flip coverage check [X]

New `hack/m5-coverage-check.sh` (read-only on the cluster, run from the owner's machine). Run it at session start (the entry gate) and again right before the tag that makes the flip live: this sprint's own tag, or, on the ride path, at merge time and once more by [m6b-04](sprint-m6b-04.md) right before it tags `v2.1.0` (see Release).
- **Live list.** The live DSA item ids on `main`: `curriculum/courses/dsa/items/*/item.json` filtered with `jq 'select(.status == "live")'`. Items can be `live`, `retired` or `withdrawn` ([m1-01](sprint-m1-01.md) task 3), so "non-retired" is not enough.
- **Per-item source.** judge's `GET /internal/evaluable?path=dsa` ([m3-05](sprint-m3-05.md) task 3: `{admission, pack{version,…}, items[{item_id, status, contract_hash, grader_kinds, run_available, submit_available}]}`), read through the node-local port-forward pattern of [mi-06](sprint-mi-06.md) / [p-01](sprint-p-01.md): a background `ssh -L 18087:127.0.0.1:18087 vps 'k3s kubectl -n xlearn port-forward deploy/xlearn-judge 18087:8087'`, then `curl -s 'http://127.0.0.1:18087/internal/evaluable?path=dsa'`; the script kills the tunnel on exit (`trap`). The route needs no JWT, and the port-forward enters the pod's netns through the kubelet, so MI-5a's `/internal/*` fence doesn't block it. The path slug is a script argument (`--path dsa`); `hack/` is outside m1-03's literal gate anyway.
  - Fallback if the port-forward is unavailable: the owner-session `GET /api/judge/status?path=dsa` ([m3-09](sprint-m3-09.md) task 7), whose per-item `items{<id>: evaluable|contract_mismatch|unsupported|invalid|no_pack}` map gives the same ids (no `submit_available` there, so the `admission`/`submit_available` assertions fall back to `state: ok`).
- **Count cross-check.** `ssh vps 'k3s kubectl exec -n xlearn deploy/xlearn-judge -- judge admin status --json'` ([m3-14](sprint-m3-14.md) task 5; there is **no `--path` flag**, and `status` reports only evaluable **counts** by course and status `ok|spec_mismatch|unsupported|invalid`, with no item ids and no `no_pack` count). This is the sanctioned admin-CLI path ([ADR-0033](../../adr/0033-invite-only-admission-and-owner-admin.md), D33). It is not side-effect free: **every `judge admin` verb, reads included, writes an `admin_audit` row**, so the script calls it once per run.
- **Assertions:**
  - every live id appears in `items[]` with `status == "ok"` and `submit_available == true` (a live id missing from `items[]` counts as `no_pack`);
  - `admission == "open"`;
  - no `spec_mismatch`, `unsupported`, `invalid` or `no_pack` among the live ids;
  - the admin `status` DSA `ok` count equals the live count, and its other DSA statuses are 0;
  - the live evalpack version (`pack.version`) equals the latest `v1.x` tag (the `xlearn-evalpack` ImagePolicy's latest).

  The script prints a per-status table and exits non-zero on any gap. Attach its output to the PR.
- **Informational (never fails the script):** it also lists any open DSA course attempts pinned `grading_mode=self`, using the read-only on-demand SQL pattern of m3-08's runbook `docs/v2/runbooks/practice-judge-lifecycle.md` (created by [m3-08](sprint-m3-08.md)). These attempts are honoured to conclusion, and the owner may prefer to finish them before the tag.
- A gap is a **stop**, not a fix-in-session. Report the item ids; the owner stamps the pack first.

### 2 · The flip and the effective policy [X]

- **Manifest:** in `curriculum/courses/dsa/course.json`, set `"grading": {"self_report": {"outcome": "evaluator_only", "touch": "allowed", "mock": "allowed"}, …}`. `strategy` stays `verdict_timer@1` (m3-08), and `params` are unchanged.
- **Golden:** update the DSA `grading.self_report.outcome` row in `internal/course/golden_test.go`, citing M5 ([ADR-0026 §6](../../adr/0026-per-course-extensibility-model.md#6-migration-the-v1-manual-loop-coexists-throughout), [t0 §9](../research/t0-extensibility-frame.md#9-how-live-dsa-migrates), [ADR-0034 §2](../../adr/0034-v2-release-labelling-gating-and-rollback.md#2-feature-gating-three-tiers-no-flag-service) T-1). m1-01 requires this: every deliberate change to a DSA method value updates the golden in the same PR.
- **Validation** (`internal/course/validate.go`): `evaluator_only` on any signal requires `grading.strategy ≠ self@1`, so a course can't be evaluator-only on the self strategy. A table test covers it.
- **Effective policy** (new `internal/practice/policy.go`):

  ```go
  type SelfReport string   // "allowed" | "evaluator_only"
  type PolicySource string // "manifest" | "override" | "no_judge"
  func (s *Service) EffectiveSelfReport(slug string, sig course.Signal) (SelfReport, PolicySource)
  ```

  The rules, first match wins:

  | Rule | Result | Source |
  |---|---|---|
  | `GRADING_OVERRIDE=self` (m3-08's name; use status.md's name if it differs) | `allowed` | `override` |
  | practice's `JUDGE_BASE_URL` unset | `allowed` | `no_judge` |
  | otherwise | the manifest value | `manifest` |

  - **Why `no_judge` counts as allowed:** presence by config ([ADR-0034 §2](../../adr/0034-v2-release-labelling-gating-and-rollback.md#2-feature-gating-three-tiers-no-flag-service)). `evaluator_only` is a property of an evaluator that must be present. An R-a kill switch must never lock the owner out of a course.
  - **When it is evaluated:** at each start and each transition. The env is read at boot, so changing it means an infra PR and a pod restart (1–2 min, T-2).
- **Internal read:** practice's `GET /internal/grading-policy?path=<slug>` returns `{path, selfReport: {outcome, touch, mock}, source}`, all effective values. It needs no JWT. It is in the `/internal/*` family that MI-5a fences to the `xlearn` namespace, and the gateway is its only caller.

### 3 · practice enforcement [X]

Scope: `purpose=course` attempts on a course whose effective `outcome` is `evaluator_only` (DSA after this sprint). Sources: [t4 §3.3](../research/t4-judge-contract.md#33-transitions), [§3.4](../research/t4-judge-contract.md#34-give-up-which-for-evaluated-items-includes-reveal-full-solution), [t4 §13](../research/t4-judge-contract.md) D15/D16/D18, [ADR-0029 §3](../../adr/0029-judge-contract-and-learning-signal.md#3-conclusion-practice-the-single-writer).

| Path | Today (`allowed`, m3-08) | M5 (effective `evaluator_only`) |
|---|---|---|
| Start: `judge: true` and the item is `ok` with `submit_available` for practice's `contract_hash` | judged pin (45:00, D18 `stage_params`) | **unchanged** |
| Start: the item isn't `ok` (`spec_mismatch`, `no_pack`, `unsupported`, `invalid`) | pins `self` → the picker | **409 `evaluation_pending{reason}`** (reason from that enum); **no attempt row, no timer, no outbox row** |
| Start: the item is `ok` but `submit_available=false` (judge's L15 `JUDGE_ADMISSION=closed`, [m3-05](sprint-m3-05.md) task 5; or judge's `RUNNER_BASE_URL` unset so the runner lane is off, [m3-06](sprint-m3-06.md) task 2) | falls through to the v1 self start (m3-08 pins `judge` only on `ok` **with** `submit_available`) | **503 `evaluation_unavailable{fallback: "none"}`**; **no attempt row, no timer, no outbox row**; the SPA shows V3 |
| Start: judge unreachable (5 s), or the gateway didn't send `judge: true` | pins `self` (WARN) | **503 `evaluation_unavailable{fallback: "none"}`**; no attempt, no timer; one WARN line |
| Resume of an open attempt | same | same: **pins are honoured**. A policy change never re-routes or strands an open attempt |
| `POST /problems/{id}/outcome` | INV-9: 409 `evaluated_item` unless `grading_mode=self` or `self_grade_pending` | **409 `evaluator_only`**, except (a) an open attempt pinned `grading_mode=self` before the flip or under the override, or (b) an attempt already in `self_grade_pending` (only ever entered under `allowed`) |
| `POST /problems/{id}/reveal` → solution | 409 `use_give_up` on judged attempts | the same for every new DSA attempt (all are judged): **reveal = give-up = Miss**. v1's "owes another attempt in 3 days" and re-implement survive only on pinned self attempts |
| → `self_grade_pending` on `contract_changed` with counted evals, `budget_exhausted`, or the 3rd released close | capped picker, **Retry grading** once | **`voided`** with `void_reason`: the timer is closed, **no `outcome` row, no `problem_solved`/`attempt_logged`**, and `user_problem_state` is untouched. The next start (once the item is `ok`) is a fresh attempt |
| Kill-switch sweep (override set, or `JUDGE_BASE_URL` unset) | open judged → `self_grade_pending`, never Miss | **unchanged**: the effective policy is `allowed` then, and that is how "the override restores the picker" |
| Touches (`purpose=touch`) | m3-08 | **unchanged** (`self_report.touch` stays `allowed`) |

- **Where it lives:**
  - start: the judged-start branch in `internal/practice/handlers.go` / the store's start path. Under `evaluator_only`, **every** start that doesn't end in a judged pin is a refusal: m3-08's "otherwise the v1 start runs unchanged" fall-through must be unreachable for a DSA course start (the three refusal rows above are the complete list);
  - INV-9: `handleOutcome`;
  - `self_grade_pending` entries: in the consumer (`internal/practice/judgeconsumer.go`), the close route and `internal/practice/ticker.go`. Route all of them through one helper, `failOpenAttemptTx(attempt, reason)`, which picks `self_grade_pending` or `voided` from the effective policy.
- **`void_reason`:** add `practice.attempt.void_reason text NULL` (**no CHECK**) in `internal/practice/store/migrations/000NN_m5_void_reason.sql`, validated by a closed Go enum `VoidReason` (`contract_changed | budget_exhausted | released_limit`). This follows the practice precedent of closed Go enums without a DB CHECK ([m2-01](sprint-m2-01.md)'s `resolution`, m3-08's `close_action`, [m4-04](sprint-m4-04.md)'s `provisional_reason`), so a later reason needs no DROP/ADD CONSTRAINT (which m1-02's contract lint would flag).
  - Only if M2a/m3-08 left no equivalent reason column; if they did, reuse it.
  - It is expand-only: no `xlearn:contract` marker, and it takes the next free goose version at rebase.
  - Then queries, `sqlc generate`, and **`sqlc diff` clean**.
- **Voided course attempt:** one guarded update, `SET state='voided', resolution='voided', void_reason=$2, ended_at=now(), concluded_at=now() WHERE id=$1 AND concluded_at IS NULL`, plus closing the attempt timer.
  - **Why `ended_at` and `concluded_at`:** for `purpose=course`, "open" is still v1's `ended_at IS NULL` (`GetOpenAttempt`; [m2-01](sprint-m2-01.md) task 1 keeps that derivation so an R-b rollback stays safe). Without `ended_at`, the voided attempt would stay open to the resume and one-open-attempt checks, both now and after an R-b to a pre-M5 tag. Without `concluded_at`, m2-01's startup heal (`HealEndedCourseAttempts`, `WHERE purpose='course' AND ended_at IS NOT NULL AND concluded_at IS NULL`) would rewrite it to `concluded` with a NULL grade, and m2-01's guarded `Conclude()` (`WHERE concluded_at IS NULL`) could still conclude it. With both set, every pre-M5 open, heal and conclude guard treats it as closed.
  - The erase delete list covers the column with no change, since the row is already listed.
- **State read:** `GET /state/{problemId}` and `GET /state?ids=` add `voidReason` for a latest attempt that is `voided`. This is additive; m3-09's poll allowlist gains it in task 4.
- **Boot WARN (D34: a log line, never an alert):** a background goroutine after start (never on the readiness path). It iterates the embedded course registry (`internal/course`), and for each manifest whose effective `outcome` is `evaluator_only` it calls `judgeapi.Client.Evaluable(slug)`. If any live item isn't `ok` with `submit_available`, it logs one line: `WARN evaluator_only_uncovered path=<slug> count=N ids=[≤10]`. No course slug appears as a literal (m1-03's `hack/lint-course-literals.sh` rejects `"dsa"` in `internal/`).

### 4 · gateway [X]

- **Policy read:** `internal/gateway/judge.go` reads practice's `GET /internal/grading-policy?path=` through the practice client.
  - It is cached **30 s in-process**. Add the cache to the L24 single-replica list in [`../../architecture/services.md`](../../architecture/services.md).
  - Practice unreachable: keep the last value. With no value yet, answer the embedded manifest's `grading.self_report.outcome` for that path (via `internal/course`, so an `evaluator_only` course fails closed and an `allowed` course stays `allowed`). Fail closed: the server refuses anyway, and the UI must not offer a picker it can't honour. Keep it course-agnostic: no `"dsa"` literal in `internal/` (m1-03's `hack/lint-course-literals.sh`, CI).
- **`judge` block** on `GET /api/problems/{id}` ([m3-09](sprint-m3-09.md) task 1): add `selfReport` (`allowed | evaluator_only`, effective, outcome signal). `GET /api/judge/status?path=` gets the same `selfReport`. Poll composition passes practice's `voidReason` through.
  - Add all three to the `judge_dto.go` allowlist, with its test. The denylist test is unchanged.
- **503 `evaluation_unavailable{fallback}`** on start and on submit: the gateway sets `fallback` from the effective policy: `self` iff `allowed`, else `none`. It never forwards judge's own hint (m3-14's L15 answers `fallback: self` regardless of the course).
- **Typed pass-through:** 409 `evaluation_pending{reason}` on start (an enum only, no internals), and 409 `evaluator_only` on outcome.
- **Docs:** `docs/architecture/openapi.yaml` + [`../../architecture/api.md`](../../architecture/api.md) (the drift test enforces them).

### 5 · web: the AB07 evaluator-only variant [X]

The variant comes from the **frozen** AB07 (`design-system/screens/v2/AB07-workspace-code.html`: F1 Cover, F11 `self_grade_pending`, F12 self-path) and AB11 (`AB11-degradation-badges.html`: F1–F4). The copy below is drafted here and lands as drafted (D40); the owner can revise it later with a content PR. Use [`theme.css`](../../../design-system/theme.css) verbatim: `ds-badge--info`, `ds-card`, `ds-btn` and the `xl-*` classes.

**Screen selection** (m3-11's `ProblemRoute` table in `web/src/router.tsx`, server data only):

| Condition | Screen |
|---|---|
| `judge.enabled`, `judge.evaluable` | judged `Workspace.tsx` (unchanged) |
| `judge.enabled`, not evaluable, `judge.selfReport = evaluator_only` | **Workspace, paused variant (M5): never the picker** |
| `judge.enabled`, not evaluable, `selfReport = allowed` (the override is on) | m3-11's self-path variant (unchanged) |
| open attempt pinned `gradingMode = self` / `self_grade_pending` | unchanged (pins honoured) |
| latest attempt `voided` (`voidReason` set) | **voided card (M5)** |
| no `judge` block (`JUDGE_BASE_URL` unset on the gateway) | v1 `Problem.tsx` (the R-a screen, unchanged) |

**Variant frames**

| Frame | Derived from | Must show |
|---|---|---|
| V1 Cover | AB07 F1 | F1 unchanged plus one rules line: "Every attempt is auto-graded." No picker or "pick your own grade" wording anywhere on DSA |
| V2 Paused: this item | AB07 F12a + AB11 F1/F3 | HUD chip **"Grading paused"** (`ds-badge--info`); Start disabled as "Start attempt · paused"; body "Counted attempts on this problem are paused while its auto-grading is refreshed. Nothing is lost: your progress and reviews are unchanged." · "You can still practise it in the arena. Nothing there counts." **[Open in arena]** |
| V3 Paused: the grader is off (`evaluable = 0`, `state: off`, or a 503 on Start) | AB11 F4 | "Auto-grading isn't available right now, so counted attempts can't start. Try again in a few minutes, or practise in the arena." Start disabled, **[Open in arena]** |
| V4 Submit refused mid-attempt (503 `fallback: none`) | AB08 F15 | a dock line: "Couldn't grade this submit right now. It isn't counted. Submit again in a moment; the timer keeps running." |
| V5 Voided | AB07 F11 (replaces it) | A cause line chosen by `voidReason` (table below), then "Nothing is recorded against you." **[Start again · keep my code]**, enabled once the item is evaluable. It starts a fresh attempt, then copies the voided context's latest draft into the new context through [m3-14](sprint-m3-14.md)'s drafts API via the gateway. While the item isn't evaluable yet, it shows V2's paused note |
| V6 < 1024 px | AB07 F14 | V2 and V5 at 390 px |

**V5 cause lines** (one per `voidReason`, following AB07 F11's causes; F11's kill-switch cause has no V5 row, because under the override the effective policy is `allowed` and the attempt goes to `self_grade_pending` instead):

| `voidReason` | V5 cause line |
|---|---|
| `contract_changed` | "This problem was updated while you worked, so this attempt won't count." (F11's `contract_changed` cause) |
| `released_limit` | "The grader couldn't get a reliable result three times, so this attempt won't count." (F11's third-release cause) |
| `budget_exhausted` | "We couldn't grade this attempt automatically, so it won't count." (F11's generic line; m3-09 maps `budget_exhausted` to `pendingReason: other`) |

**Badges** (`web/src/components/judge/JudgeBadges.tsx`, m3-12), when `selfReport = evaluator_only`:

| Condition | Badge |
|---|---|
| item mismatch / `no_pack` | **"Grading paused"**, replacing "Grading pending · self-report". Popover: "This problem's auto-grading is being refreshed. Counted attempts are paused until it's back. You can still practise it in the arena." |
| Problems row chip | "Paused", replacing "Self-report" |
| `evaluable = 0` / `off` | Dashboard and Problems banner "Auto-grading is paused, so counted attempts are paused too. Reviews and the arena still work." |
| saturated / breaker | unchanged ("Runner busy") |

- **Files:**
  - new `web/src/screens/workspace/PausedCover.tsx` and `VoidedCard.tsx`;
  - edits to the Cover and the dock status line (`DockStatus.tsx` / `web/src/components/judge/ResultsDock.tsx`), `JudgeBadges.tsx`, `web/src/screens/Problems.tsx`, `web/src/screens/Dashboard.tsx`, and `web/src/router.tsx`;
  - `web/src/lib/judge.ts` types: `selfReport`, `evaluation_pending`, `fallback: "none"`, `voidReason`.
- **The client never decides grading.** Every branch mirrors a server field. The raw manifest's `self_report` (visible through curriculum's manifest view) is **not** the effective policy; read `judge.selfReport` only.
- **Accessibility:**
  - status lines use `aria-live="polite"`;
  - a disabled Start carries its reason via `aria-describedby`;
  - badges have icon + text, contrast ≥ 4.5:1, and `role="status"`.

### 6 · Kill switch: the override restores the picker [X]

- **Compose drill** (`internal/e2e/m5_override_test.go`, `-tags e2e`, or a scripted manual drill recorded in the PR). Compose runs with m3-02's fixture pack, so the DSA items outside it are paused and the fixture items are judged.
  1. Baseline: a fixture item starts judged; a non-fixture item → 409 `evaluation_pending`, and the SPA shows V2; no picker anywhere.
  2. Set `GRADING_OVERRIDE=self` on practice (`docker compose up -d practice` with the env) and wait for practice Ready. The next page load shows `selfReport = allowed` once the 30 s gateway cache expires, or at once after a gateway restart. A new start on the paused item pins `self` and shows the **picker** (m3-11's self-path variant). An open judged attempt moves to `self_grade_pending` (capped picker) within one ticker interval (30 s), and **never to Miss**.
  3. Unset the override. Items are paused or judged again. Attempts concluded under the override keep their grades, and **nothing is re-graded**. An attempt still in `self_grade_pending` keeps its capped pick.
  4. Record the measured times (override → picker visible) in the Decisions log.
- **Runbook:** add a section **"Evaluator-only (M5)"** to `docs/v2/runbooks/practice-judge-lifecycle.md` (m3-08's runbook; if it is missing on `main`, create it with this section and note the gap against m3-08):
  - the R-a recipe: an infra PR adding `GRADING_OVERRIDE: "self"` to the `env` block of `../infra/apps/xlearn-practice.yaml`, where Flux restarts practice in about 1–2 min; a revert PR lifts it;
  - when to use it: `evaluable = 0`, a PAT expiry, a bad pack, or a long `spec_mismatch`;
  - `hack/m5-coverage-check.sh`;
  - a read-only SQL for recent voided course attempts;
  - the **pack-first content rule**: after M5, a new or contract-changed DSA item ships its stamped pack in an evalpack tag that is live before, or with, the xlearn tag that carries the public half. Otherwise the item shows "Grading paused".
- No prod drill by default ([m3-13](sprint-m3-13.md) precedent). If the owner asks, it is two infra PRs (set, then revert), each verified by looking (D34).

### 7 · Tests + verify [X]

- **practice:**
  - a policy table test (manifest × override × `JUDGE_BASE_URL` → value and source);
  - the refused starts write **no** attempt, timer or outbox row, one case per refusal row: 409 `evaluation_pending` for each non-`ok` status; 503 for an `ok` item with `submit_available=false` (fake judge answering `admission: closed`, and a fake with the runner lane off); 503 for judge unreachable and for a start without `judge: true`;
  - outcome → 409 `evaluator_only`; the pinned self attempt and a `self_grade_pending` pick are still accepted;
  - every `self_grade_pending` trigger → `voided` with its reason and no outbox row, under `-race` with the consumer, the ticker and a close racing on one attempt;
  - a voided course attempt has `ended_at` and `concluded_at` set: `GetOpenAttempt` doesn't return it, the next start creates a fresh attempt, m2-01's guarded `Conclude()` is a no-op on it, and a practice restart (`HealEndedCourseAttempts`) leaves it `voided`;
  - the kill-switch sweep is unchanged; m3-08's D18 golden table is unchanged for judged attempts.
- **The route sweep** ("no self path left"): enumerate practice's router, and assert that no route can write a `graded_by=self` DSA course outcome, or pin `grading_mode=self` on a DSA course start, while the effective policy is `evaluator_only`, except for pinned attempts. Drive the start routes with each fake-judge state above, including `ok` + `submit_available=false`.
- **gateway:** the DTO allowlist including `selfReport`/`voidReason`; the `fallback` rewrite on start and submit; the policy cache (hit, expiry, and the no-value default taken from the embedded manifest per path, for an `evaluator_only` and an `allowed` fixture course); the literal gate stays green.
- **web (Vitest):**
  - a test per frame V1–V6 from new fixtures in `web/src/test/fixtures/judge/` (`evaluator_only_*.json`), plus one V5 case per `voidReason` (`contract_changed`, `released_limit`, `budget_exhausted`) asserting its cause line;
  - the `ProblemRoute` table: **no picker renders for DSA while `selfReport = evaluator_only`**;
  - the badge mapping.
- **compose e2e (`-tags e2e`):** a judged pass → Clean; give-up → Miss (the solution unlocks); timeout → Miss at the deadline; a paused item → 409 + V2; the task-6 drill.
- `gofmt`, `go vet`, `go test -race ./...`, `npm --prefix web run test`, `-tags e2e`, the migration lint, **`sqlc diff`**, the course golden and validation tests.

### 8 · Docs + ADR [X]

- **ADR** at the next free number (check `docs/adr/`, open PRs, peers' worktrees and ListAgents before claiming it), MADR-style, Accepted on merge: **"DSA evaluator-only (M5): the effective self-report policy and the no-self lifecycle"**. It records:
  - the effective-policy rules, including `no_judge` → `allowed`;
  - start refuses rather than falling back;
  - `self_grade_pending` → `voided` under `evaluator_only`;
  - pins are honoured;
  - `touch`/`mock` stay `allowed`.

  It amends [ADR-0027 §2](../../adr/0027-content-evalpack-and-user-data-model.md#2-where-content-lives-and-how-it-ships) ("on a mismatch the item falls back to self-report" → paused for an `evaluator_only` course) and [ADR-0029 §3](../../adr/0029-judge-contract-and-learning-signal.md#3-conclusion-practice-the-single-writer), and cites [ADR-0034 §2/§4.4](../../adr/0034-v2-release-labelling-gating-and-rollback.md#44-reversibility-by-step). Add a one-line "Amended by ADR-00NN" note to 0027 and 0029. Don't rewrite them; ADRs are append-only.
- **Architecture docs:**
  - [`../../architecture/api.md`](../../architecture/api.md): `GET /internal/grading-policy`, 409 `evaluation_pending`, 409 `evaluator_only`, `fallback: none`, `selfReport`, `voidReason`;
  - [`../../architecture/services.md`](../../architecture/services.md): gateway → practice policy read, and the L24 cache;
  - [`../../architecture/data-model.md`](../../architecture/data-model.md): `void_reason`.
- The runbook section (task 6).

### 9 · PR with the variant screenshots [X]

- **Branch and commit:** branch `feat/m5-dsa-evaluator-only`. Commit(s) conventional, e.g. `feat(practice,gateway,web): DSA evaluator-only (M5)`, ending with the attribution lines.
- **Screenshots:** V1–V6 at **1440 px** and **390 px**, each beside the frozen AB07 F1/F12 crop it derives from.
  - Capture them with the temp vite mock config pattern (a throwaway config, never committed) or from compose.
  - Commit them as `design-system/screens/v2/shots/AB07-m5-V<n>@1440.png` / `@390.png` (≲ 500 KB each, ds-m1-01's convention). They become the reference for the variant.
- **PR body:** the frame list, the copy table, the task-1 coverage output, the drill timings, and a **"Decisions to confirm"** list (it doesn't block the merge: each item states the default that lands, and the owner may revisit any of them afterwards with a follow-up PR, D40):
  - `touch`/`mock` stay `allowed`;
  - `JUDGE_BASE_URL` unset also restores the picker (`no_judge`);
  - every former `self_grade_pending` trigger now **voids** the attempt rather than offering a capped pick, each through no fault of the learner: (a) a contract change mid-attempt with counted evaluations (`contract_changed`), (b) the third infra-released close (`released_limit`), (c) the grading budget exhausted (`budget_exhausted`); with the V5 cause line for each;
  - a start on an `ok` item whose submit lane is off (judge's L15 admission closed, or the runner lane off) is refused with V3 rather than falling back to self;
  - pre-flip self attempts are honoured;
  - the release path (ride `v2.1.0` vs own minor).

### 10 · Release [X]

See **Release** below. On the ride path: CI green → coverage + PAT checks → squash-merge **untagged** → record "rides `v2.1.0`" plus the hand-off for [m6b-04](sprint-m6b-04.md) (re-run both checks right before its tag). On the own-minor path: CI green → squash-merge → the pre-tag checklist → tag → verify → record.

## Acceptance criteria

- [ ] **M5 exit: no self path left in DSA.** With judge present and the override unset:
  - no DSA course attempt can pin `grading_mode=self`: start either judges or refuses (409 `evaluation_pending` / 503 `fallback: none`, including an `ok` item with `submit_available=false`), with no attempt or timer row;
  - `POST /problems/{id}/outcome` → 409 `evaluator_only`, except for pinned pre-flip/override attempts;
  - no DSA course attempt enters `self_grade_pending` (it becomes `voided`, with no signal);
  - the SPA never renders the self picker for DSA;
  - **give-up = Miss**, with reveal routed to give-up.

  The route sweep, the Vitest table and the compose e2e are green.
- [ ] **An item with `spec_mismatch` blocks counted attempts with a badge.** It shows "Grading paused" and V2; the arena still works; nothing is recorded against the learner.
- [ ] **The override restores the picker within minutes.** The compose drill passes: new starts pin `self` with the picker, open judged attempts → `self_grade_pending` within 30 s and never a Miss, unsetting it restores evaluator-only, and nothing is re-graded. The times are recorded, and the prod recipe is in the runbook.
- [ ] The DSA manifest golden is updated with its citation; the validation rule is tested; `sqlc diff` is clean; CI is green.
- [ ] The AB07 evaluator-only variant **lands as drafted** (no owner approval step, D40), and its screenshots are committed under `design-system/screens/v2/shots/`.
- [ ] `hack/m5-coverage-check.sh` and the PAT > 14-day check are green on prod **right before the tag that makes the flip live**: this sprint's own tag, or, on the ride path, at merge time and again by [m6b-04](sprint-m6b-04.md) right before it tags `v2.1.0` (the hand-off is recorded).
- [ ] **Released:** the own-minor tag is verified per the checklist, with `/api/judge/status?path=dsa` reporting `selfReport: evaluator_only` and a DSA problem opening the judged workspace with no picker. Or, on the ride path, the M5 checks are recorded for m6b-04's verify.

## Release

[ADR-0034 §1.1](../../adr/0034-v2-release-labelling-gating-and-rollback.md#11-scheme): M5 "rides v2.1.0 if it's ready by then". So: **ride `v2.1.0` if M5 is ready before [m6b-04](sprint-m6b-04.md) and every ride condition below holds; otherwise take the next free minor ≥ `v2.2.0`.** On the calendar (M5 ≈ H2 2027 after `ev-dsa-packs-all`, `v2.1.0` ≈ Q1 2027) the own minor is the likely outcome, but the ride is checked first.

**Ride `v2.1.0` (only if every condition holds):**
- `v2.1.0` isn't tagged yet, and [m6b-04](sprint-m6b-04.md) hasn't started its tag;
- [m6b-03](sprint-m6b-03.md)'s `v2.0.x` patch **is already tagged**. **No `v2.0.x` patch may be cut from `main` between this merge and `v2.1.0`.** A content-wave patch cut after this merge would ship the T-1 flip as a patch, which breaks §1.1. The session records this commitment in the Decisions log (D40: launching the prompt is the owner's approval), so later sessions see it before cutting a patch. (These extra conditions follow from §1.1's "the minor moves only at a GA flip"; they narrow the ride, they don't replace it);
- the coverage check and the PAT > 14-day check are green at merge time.

Then merge untagged. Record the M5 milestone row as "rides `v2.1.0`" in status.md, and add a Decisions-log hand-off for m6b-04: **right before tagging `v2.1.0`, re-run `hack/m5-coverage-check.sh` and the evalpack PAT > 14-day check; if either fails, hold M5 (a PR reverting the DSA manifest line and its golden row back to `allowed`; every other M5 branch keys off the effective policy, so the rest ships dark) and tag `v2.1.0` without it; otherwise list M5 in the release notes and run the M5 smoke below after its verify.** The flip only goes live at that tag, so a DSA item or contract change that lands without its pack, or a PAT drifting toward expiry, between this merge and `v2.1.0` must be caught there. Never cut `v2.1.0` for M5 alone.

**Own minor (otherwise):** tag **the next free minor after `v2.1.0`** (≥ `v2.2.0`; its major equals `.release-line = 2`). Title: `v2.N.0 — DSA evaluator-only (M5)`.
- From `v2.0.0` on, the minor moves only at a GA flip ([ADR-0034 §1.1](../../adr/0034-v2-release-labelling-gating-and-rollback.md#11-scheme)). This T-1 default flip is one, so a minor is correct.
- If M5 takes `v2.2.0`, M6c's indicative "v2.2" label simply moves to the next free minor ([§1.6](../../adr/0034-v2-release-labelling-gating-and-rollback.md#16-indicative-tag-timeline): take the next free minor at tag time).
- The release notes list every commit since the last tag (`git log <last-tag>..main --oneline`). Dark M6c or content-wave commits riding along are fine, but they must be named.

Release checklist ([ADR-0034 §6](../../adr/0034-v2-release-labelling-gating-and-rollback.md#6-release-checklist) + the ADR-0035 §2 standing rule):
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

For this tag:

| Checklist item | Here |
|---|---|
| **ACL / new service image** | n/a: no stream, consumer, subject or service |
| **Contract** | n/a: `void_reason` (if added) is expand-only, so the contract-header lint stays silent |
| **Snapshot** | not required: not a contract, erase or GA tag (the same reading as [m6b-04](sprint-m6b-04.md)'s `v2.1.0`), and R-a reverses it with no residue ([ADR-0034 §4.4](../../adr/0034-v2-release-labelling-gating-and-rollback.md#44-reversibility-by-step)). The owner may take one anyway |
| **From M6** | applies. `coach admin interviews --live` must be empty before the tag, because the tag restarts coach |
| **NetworkPolicy** | n/a: gateway → practice already exists, and `/internal/grading-policy` sits in MI-5a's `/internal/*` family |
| **Memory sum** | no new pod, limits unchanged ([ADR-0035 §5](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#5-capacity-the-memory-sum-rule-triggers-and-ordered-responses)) |
| **Pre-tag** | `hack/m5-coverage-check.sh` green; the evalpack PAT more than 14 days from expiry (on the ride path, m6b-04 re-runs both right before `v2.1.0`) |
| **M5 smoke** (owner account) | a DSA problem opens the judged workspace with **no picker**; `/api/judge/status?path=dsa` reports `selfReport: evaluator_only` and `evaluable` = the live count; a give-up concludes as Miss; the Problems list shows no "Paused" chip. An agent never enters credentials: without an already-signed-in owner browser session, run the credential-free checks and record "owner login smoke pending" as a pending-smoke note in status.md |
| **Record** | `M5 → v2.N.0 → floor unchanged → snapshot n/a` (`void_reason` is expand-only, so the rollback floor doesn't move; the same reading as [m6b-04](sprint-m6b-04.md)'s "floor unchanged"); T-1 (DSA `self_report.outcome = evaluator_only`); the grading-override row ("restores the DSA picker since M5") |

**Rollback** ([ADR-0034 §4.1](../../adr/0034-v2-release-labelling-gating-and-rollback.md#41-mechanisms-fastest-first)):
- **R-a:** `GRADING_OVERRIDE=self` via an infra PR (1–2 min) brings the picker back. Unsetting `JUDGE_BASE_URL` does the same, but it also turns judge off.
- **R-b:** narrow the xlearn ranges with `!=2.N.0`, never below the floor.
- **R-c:** revert the manifest line and its golden row, and tag a patch (`v2.N.1`).
- **Residue:** none. Voided attempts stay voided, and nothing is re-graded. After an R-b to a pre-M5 tag, the old code treats a voided course attempt as closed, because the void write sets `ended_at` and `concluded_at` (task 3); it ignores `void_reason`.

## Definition of Done

- CI is green (including `sqlc diff`, the migration lint, the golden and validation tests, and the e2e).
- The PR is merged (the variant lands as drafted, D40).
- It is merged to ride `v2.1.0` (with the ride conditions and the m6b-04 re-check hand-off recorded), or tagged and verified on its own minor, with no hand `kubectl`.
- The acceptance criteria are met.
- Statuses are updated: this file, plus [`../status.md`](../status.md) (the board row; the M5 Milestones row ✅ with its tag, or "rides `v2.1.0`"; milestone → tag → floor → snapshot; the flag inventory's grading-override note; the T-1 line; the content table's "DSA 100% packed · pack-first rule"; the Artboards AB07 row "M5 variant merged (PR #, date)").
- The ADR is merged, and `main` is synced.

## Risks / watch-outs

- **A late `spec_mismatch` blocks attempts.** For example, a public contract change merged without its pack. Guards:
  - the "Grading paused" badge plus V2 (the owner sees it in-app, D34);
  - the pack-first content rule in the runbook;
  - the boot WARN;
  - the override as the fallback.
- **judge `evaluable = 0`** (PAT expiry plus a restart, a bad pack, an unsupported format major) now pauses **every** counted DSA attempt, not just grading. The mitigations are the PAT entry gate, the owner's PAT check (`ev-pat-expiry`), and R-a (`GRADING_OVERRIDE=self`) as the documented remedy.
- **Riding `v2.1.0` can leak the flip into a patch.** If any `v2.0.x` is cut from `main` after the merge, the T-1 flip ships unlabelled. Ride only under the Release conditions; when in doubt, take the own minor. The ride also leaves a gap between the merge and the `v2.1.0` tag that makes the flip live; m6b-04's pre-tag re-check (coverage + PAT, hold M5 on failure) closes it.
- **Gateway/practice config skew.** With `JUDGE_BASE_URL` unset on the gateway but set on practice, the v1 page starts attempts without `judge: true`, and practice answers 503. Always flip the two together, or set the override. This is m3-13's hand-off, restated in the runbook.
- **"No self path" read to include touches.** M5 flips `outcome` only, per [t0 §9](../research/t0-extensibility-frame.md#9-how-live-dsa-migrates) ("for outcomes"). The touch attestation survives only during mismatch windows or with the kill switch on. If the owner wants `touch` flipped too, it is a later patch, and it needs the R-SR5 coupling solved first: an un-doable due touch must not block new DSA work.
- **Open self attempts at flip time** are honoured to conclusion. That's deliberate (no stranding), but the owner may prefer to finish them before the tag. Task 1's script lists them (informational).
- **Voiding loses the elapsed time** of a mid-attempt contract change. The code is kept via the drafts copy, and no Miss is recorded. That is fairer than a capped self pick under an evaluator-only policy; it is listed under "Decisions to confirm" in the PR and lands as drafted (D40).
- **The 30 s gateway policy cache** lags the override by up to 30 s. That is within "minutes"; a gateway restart makes it instant.

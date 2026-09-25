# Prompt — Sprint m6a-06 · M6a UI part 2: grace/paused/resume (AB26), debrief/proposal (AB27), accessibility (AB28) → v2.0.x patch

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-m6a-06.md`](../sprints/sprint-m6a-06.md)   ·   **Milestone:** M6a (proves the M6a exit; tags a `v2.0.x` patch, M6a dark)   ·   **Prereqs:** [m6a-05](../sprints/sprint-m6a-05.md), [ds-m6a-02](../sprints/sprint-ds-m6a-02.md) (AB26–AB28 frozen); the M6a chain [m6a-01](../sprints/sprint-m6a-01.md)…[m6a-04](../sprints/sprint-m6a-04.md) on `main`

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md): conventions, stack, land-and-sync.
- The plan: [`../sprints/sprint-m6a-06.md`](../sprints/sprint-m6a-06.md). Its AB26 frame table (task 1), the audit table (task 4), the e2e scenarios (task 5) and the release checklist are authoritative for this session.
- **The frozen boards:** `design-system/screens/v2/AB26-grace-paused-resume.html`, `design-system/screens/v2/AB27-debrief-proposal.html`, `design-system/screens/v2/AB28-accessibility-settings.html` (via `design-system/screens/v2/index.html`); [`design-system/theme.css`](../../../design-system/theme.css) verbatim.
- [t6](../research/t6-realtime-interviewer.md): **§4** (failsafes 1–3, failure modes, checkpoints, coach lock), **§6** (flow, `ai-byo` conditions, no auto-accept, reminders, enforcement, fairness gates), **§7** (consent, retention, accessibility, S1–S4), **§9** (the P0 list for the audit), **§11** (QA: replay regression), **§13** D30/D31, **§14** (risks). [ADR-0032](../../adr/0032-realtime-ai-mock-interviewer.md) §4–§6. [PRD §5.6](../../prd/xlearn-v2-prd.md) R-MI3–R-MI8.
- Releases: [ADR-0034 §1.1](../../adr/0034-v2-release-labelling-gating-and-rollback.md#11-scheme), [§1.6](../../adr/0034-v2-release-labelling-gating-and-rollback.md#16-indicative-tag-timeline), [§4](../../adr/0034-v2-release-labelling-gating-and-rollback.md#4-rollback), [§6](../../adr/0034-v2-release-labelling-gating-and-rollback.md#6-release-checklist); [ADR-0035 §2](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#2-nats-auth-nkey-users-fine-acls-server-first) (NetworkPolicy standing rule), [§4](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#4-limits-inventory) L19, [§5](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#5-capacity-the-memory-sum-rule-triggers-and-ordered-responses) (memory sum); [rollout §3](../rollout-plan.md#3-milestone-map) (the M6a exit) and [§7](../rollout-plan.md#7-indicative-tag-timeline).
- Peer plans: [m6a-01](../sprints/sprint-m6a-01.md)…[m6a-05](../sprints/sprint-m6a-05.md) (routes, FSM states, clock, replay fixtures, scoring, streams, `SnapshotAt`, the screens and m6a-05's gap notes), [spk-04](../sprints/sprint-spk-04.md) (S6 fixtures in `docs/v2/research/t6-s6-fixtures/`), [mi-13](../sprints/sprint-mi-13.md) (coach sizing, if pulled forward), [l-01](../sprints/sprint-l-01.md) (erase coverage test), [m2-03](../sprints/sprint-m2-03.md) (Settings rail pattern).
- Code: `web/src/screens/mock/*`, `web/src/lib/interview/*`, `web/src/components/AppShell.tsx`, `web/src/screens/Settings.tsx`, `web/src/screens/Dashboard.tsx` (Today's "Scores waiting" row), `web/src/lib/settings.ts`; `internal/coach/interview/`, `internal/coach/store/`, `internal/coach/{providers,catalog}.go` (the per-provider `billing_url`), `internal/gateway/` (interview proxy), `internal/assessment/`; `internal/e2e/`; `docs/architecture/openapi.yaml`; `../infra/hack/host-verify.sh` (read-only).

## Context

- The M6a server side ([m6a-01](../sprints/sprint-m6a-01.md)…[m6a-04](../sprints/sprint-m6a-04.md)) and the first three screens ([m6a-05](../sprints/sprint-m6a-05.md)) are on `main`, dark behind the interview cohort gate.
- **This sprint finishes the text interviewer:** the owner's failsafes on screen (grace, paused banner, resume modal, `incomplete`), the debrief and explicit-accept proposal, and the accessibility settings. It then **proves the M6a exit** ("a 45-minute text mock survives pause and resume and scores once") with a CI e2e and **tags a `v2.0.x` patch** (M6a stays dark); afterwards the owner dogfoods one real text mock on production (a post-ship owner event; the session doesn't wait for it).
- **Owner-decided (never contradict):** 5-minute grace; pause ≤ 24 h from the first pause, ≤ 3 pauses; after 24 h → `incomplete`, unscored, out of trends (D30); resume shows the free state at once and the paid brief only on a click, cached per pause; the learner must **explicitly** accept (no 24-hour auto-submit for mocks); `ScoreMock` once; mock count only in public (D31); no AI call on the learner's key without their action; transcripts deleted 30 days after scoring unless kept; no alerting (D34).

## Entry gates — verify first (stop and report if any is unmet)

- [ ] AB26, AB27, AB28 on `main` ([ds-m6a-02](../sprints/sprint-ds-m6a-02.md) merged; the merge is the freeze).
- [ ] [m6a-05](../sprints/sprint-m6a-05.md) merged; its Decisions-log gap notes read.
- [ ] **Twin fairness gate green** per reviewing catalog model (m6a-03's record); failing models set to `ai: self_only`.
- [ ] m6a-01 task 7's NetworkPolicy outcome recorded (none expected: `pause_exposure` goes through the gateway; any callee PR it opened is merged); m6a-04's "no NetworkPolicy PR needed" record present.
- [ ] `v2.0.0` live; `.release-line` = `2`; ranges `>=1.0.0 <3.0.0`.
- [ ] **Parallel sessions:** `git ls-remote --tags origin 'refs/tags/v2.0.*'`, `gh pr list`, `git worktree list`, ListAgents — the next free patch is known (D6 content waves also tag `v2.0.x`); no open peer PR on `web/src/screens/mock/**`, `web/src/lib/interview/**`, `Settings.tsx`, `Dashboard.tsx`, `AppShell.tsx` or coach migrations.

## Do this (in order)

1. **[X] AB26** (plan task 1) — every frame F1–F12 of the board:
   - `Grace.tsx` (F1/F2/F4/F11): the simple `role="alertdialog"` modal over a read-only HUD for `interrupted(quota)` — provider-aware copy, Top up from the provider's `billing_url` (add it here: a per-provider `billing_url` for `openai`/`anthropic` in coach's provider registry / catalog, copied from each console on the day, exposed on `GET /interviews/{id}` as `provider{id,label,billing_url}`; m6b-01's per-voice-entry link wins when present), "add at least ⟨remaining + $1⟩", "We'll keep checking until ⟨grace_until⟩ · checked Ns ago", the spend meter, Check now (≤ 1 / 10 s), 30 s browser probes while visible (m6a-02's probe), Save & resume later (`pause`), End interview (never default), Finish & get feedback ≥ 75 % / End without feedback while dry; the other interruption reasons inline (F4; `ErrAuth` → paused); ≥ 90 % → Finish without the rest.
   - F3 credits found (no paid summary; the clock restarts on the re-entry message).
   - `PausedBanner.tsx` (F5) in an app-wide `AppShell.tsx` slot from `GET /api/interviews/active` (cached 30 s; refetch on focus).
   - `ResumeModal.tsx` (F6–F10): m6a-01's `resume` + `GET …/resume-state` for the free state at once; Summarise & resume → m6a-02's JSON `brief` (409 `still_no_credit` → F7, nothing charged) → F8 brief ready (cached per pause) → `resume_confirmed`; Finish now ≥ 75 %; Abandon (never default); the consent reaffirmation; the exposure chip (F9); last pause and expiry (F10).
   - `Interrupted.tsx` / `Incomplete.tsx`; F12 narrow sheets. The grace warning is the only assertive announcement; times in the account timezone with the weekday.

2. **[X] AB27** (plan task 2) — every frame F1–F12: the debrief inside the HUD during `wrapping` (never a score); "Preparing your review" with the aggregate judge line; `ProposalReview.tsx` at `mock/:id/review` (bands or "No evidence", verified quotes linking to the transcript + `SnapshotAt` code, why, sliders, evidence, strengths, improvements with drill links, caveats, provenance; Accept all · Re-propose once (`…/proposal/repropose`) · Score it myself → `POST /api/interviews/{id}/submit {mode}` → `ScoreMock` once, idempotent); edited and second-proposal frames; the saved view with the **server's** `scored_by` chip + `honor`; self-scored-only variants; "Scores waiting" on the hub **and Today** (client-side from `GET /api/interviews/active`; in-app only); the `incomplete` free partial view with the paid improvements only on a click (`…/partial-feedback`); reminders from `pending_since`; review-failed / cut-short; the transcript with per-message delete, keep 12 months and delete now (m6a-01's routes).

3. **[X] AB28** (plan task 3): coach migration `interview_pref` (`default_mode`, `time_multiplier`, `reduced_motion`, `announce`; next free version; sqlc; `sqlc diff`), `GET/PUT /interviews/prefs`, **added to coach's erase transaction** (coverage test green); gateway `GET/PUT /api/interviews/prefs` (cohort) + `openapi.yaml`; the Settings "Mock interviews" section (the opt-in toggle on m6a-01's `optin`, default mode, extra time with its cost line, reduced motion, announcements); in the HUD, whole replies under reduced motion, "Jump to latest", the focus order, the `?` shortcut sheet (no shortcut for Finish/End; chords checked against CodeMirror's `defaultKeymap`), the neutral "1.5× time" / "Text interview" chips; m6a-05's setup defaults to the saved multiplier. Voice preferences (AB28 F3) are M6b's: record the hand-off (Decisions log + PR) — [m6b-02](../sprints/sprint-m6b-02.md) widens the `default_mode` CHECK to `('text','voice')` and adds the caption / screen-reader-preset / self-view columns, expand-only; [m6b-03](../sprints/sprint-m6b-03.md) renders AB28 F3.

4. **[X] P0 audit** (plan task 4): walk the plan's table (t6 §9 P0, §6 enforcement, §7, L19, cross-service, m6a-05's gaps) against `main`; put the item · sprint · evidence · status table in the PR; fix only small gaps (≈ 1 h, in the owning service, with a test); record larger ones as v2.1.0 blockers for [m6b-04](../sprints/sprint-m6b-04.md). Must confirm: `incomplete`/`abandoned` reach assessment; the server scales the rail and idle timers by the multiplier; `coach admin interviews --live` works distroless.

5. **[X] Exit e2e** (plan task 5): `internal/e2e/interview_exit_test.go` (`//go:build e2e`) — in-process identity (owner), gateway, coach, assessment, practice, judge (fake runner); the replay fake provider switching on cue to the **spend-limit path**'s quota code (the one S6 recorded, e.g. `project_spend_limit_exceeded`, via m6a-02's text-path fixtures in `internal/coach/interview/replay/testdata/`; `credit_balance_exhausted` stays in m6a-02's classifier tests — S6 has no prepaid fixture); one injected clock (never sleep). Scenarios **A** (the exit: grace → paused → resume with a cached brief → finish → re-propose once → accept → exactly one `ScoreMock`, `ai-byo`), **B** (`incomplete` after 24 h, no provider call without a click, out of trends), **C** (arena exposure during the pause → `self`), **D** optional (cap 85 % / 100 %). Log-capture: no transcript, code or key text in any log. Then the full CI set: `go test ./...`, `sqlc diff`, drift + route-enumeration tests, the e2e lane, and in `web/` `npm run typecheck`, `npm run lint`, `npm test`, `npm run build`.

6. **[X] Ship:** see **Ship** below. It merges this sprint's PR (branch `feat/m6a-06-mock-ui-part-2`; board screenshots at 1440/390 and the audit table in the PR), then runs steps 7–9 in its order.

7. **[X] Tag `v2.0.N`** (plan task 6) from `main`: run every line of the release checklist in the plan's *Release* section, including `ssh vps 'sudo k3s kubectl exec -n xlearn deploy/xlearn-coach -- coach admin interviews --live'` reporting **`live=0`** (no interview lines) **right before the tag** (log the exec). Confirm the ACL and contract lines are n/a with the two `git diff` checks in the plan. Title `v2.0.N — v2.1 build · M6a (dark)`; release notes per the plan. **Never `v2.1.0`; never move or re-push a tag.**

8. **[H + X] Verify + coach memory** (plan task 7): healthz version, images, ImagePolicies' latest, HelmReleases Ready; smoke login, dashboard, coach, and as the owner `/dsa/mock` → Mock-v2 up to the pre-flight (no paid interview). An agent never enters credentials: use an already-signed-in owner browser session; without one, run the credential-free checks and record "owner login smoke pending" as a pending-smoke note in status.md. `k3s kubectl top pod -n xlearn` for coach vs 128 Mi; `ssh vps 'bash -s -- --cluster --expect-sandbox --nats-stage=<live stage recorded in status.md>' < ../infra/hack/host-verify.sh` (`n4`, or `n3` if [mi-11](../sprints/sprint-mi-11.md) deferred N4; `--expect-sandbox` on every run after the host window, [mi-09](../sprints/sprint-mi-09.md)). Above **70 %** of the limit (t6 §3's coach split-out trigger) → record it as that trigger in the Decisions log beside the resize decision, and open an infra PR raising coach's limit, memory-sum checked, merged on its own (or pull [mi-13](../sprints/sprint-mi-13.md)'s sizing forward). Record the numbers.

9. **[X] Record docs PR** (plan task 9): branch `docs/m6a-06-record`, `docs(v2): record v2.0.N — M6a dark` with the status.md records below (tag → floor, the M6a row ✅ on this session's evidence, owner event `ev-m6a-dogfood` added (plan task 8), Artboards, the P0 audit summary and v2.1.0 blockers, coach memory and host-verify numbers, the `--live` exec log); CI green, squash-merge, `git checkout main && git pull`. Name in its description where the dogfood result will land (a second small docs PR, or the next sprint's status update).

10. **Owner event after ship, non-blocking — the owner's dogfood** (`ev-m6a-dogfood`; step 9 records the event, plan task 8): one real ~45-minute text mock on production with a pause and a resume, scored once. Don't wait for it; it gates nothing (not _Overall_ ✅, not the M6a row). When the owner reports, record wall time, spend vs estimate, pauses, whether the event stream held through Traefik, and any issue in the place step 9 named (normally a small `docs(v2): record M6a dogfood` PR, CI green, squash-merged, `main` pulled). Anything it finds becomes a follow-up PR; a blocker is fixed forward with another patch.

## Constraints

- **Release discipline:** a **patch** only (ADR-0034 §1.1); the cohort gate stays on; the checklist line "from M6: no live interviews" is mandatory; infra PRs stand alone and are never folded into the tag; image-automation and tags are never edited by hand.
- **Service boundaries ([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)):** coach owns interview state, prefs, transcripts and proposals; assessment owns `mock_session` and is the only writer of the mock signal (`ScoreMock`); the gateway composes. goose expand-only migrations (next free version at rebase), `sqlc generate` committed, `sqlc diff` clean.
- **Privacy:** no AI call on the learner's key without a click (the brief, partial improvement areas); transcripts and codes never logged; S1–S4 hold in every new view; the public route shows the mock count only (D31).
- **Frontend:** `theme.css` verbatim, the frozen AB26–AB28 copy and layout, icon + text states, `prefers-reduced-motion`, keyboard-reachable modals with focus traps and Esc behaviour per the boards.
- **GitOps:** never `kubectl apply`; `ssh vps` read-only apart from the sanctioned `kubectl exec` of the admin read verb (logged). **No alerting, opscheck or push channel (D34).** Any coach resize follows the memory-sum rule (ADR-0035 §5) and carries a memory limit.
- **Parallel sessions:** re-check peers' tags and PRs immediately before tagging and before claiming an ADR number (none expected).

## Deliverables

- `web/src/screens/mock/{Grace,PausedBanner,ResumeModal,Interrupted,Incomplete,Debrief,ProposalReview}.tsx`, the `mock/:id/review` route, the AppShell banner slot, the Settings "Mock interviews" section.
- coach `interview_pref` (migration, sqlc, routes, erase); the per-provider `billing_url` on `GET /interviews/{id}`; gateway prefs pass-through; `openapi.yaml`; the Today "Scores waiting" row in `Dashboard.tsx`.
- Small P0 gap fixes with tests; the audit table.
- `internal/e2e/interview_exit_test.go` (scenarios A–C, D optional).
- The `v2.0.N` tag and GitHub release; the post-tag record docs PR (verification, memory, audit, owner event `ev-m6a-dogfood`), then the dogfood result later, where that PR names; an infra PR only if the memory read needs one.

## Update status

- [`../sprints/sprint-m6a-06.md`](../sprints/sprint-m6a-06.md): each task 🔄 → ✅ (⛔ with a reason); set _Overall_.
- [`../status.md`](../status.md) — **after the tag, in the record docs PR (step 9)**; the dogfood lines in the follow-up it names:
  - **Sprint board** row; **Milestones:** M6a → "✅ dark in v2.0.N — exit: e2e <date>; GA at v2.1.0" (the owner's dogfood doesn't hold the ✅);
  - **owner events:** add `ev-m6a-dogfood` (after the tag: one ~45-minute text mock on production with a pause and a resume, scored once; non-blocking), ticked ✅ in the follow-up when the owner reports;
  - **milestone → tag → floor:** `v2.0.N`, floor unchanged, no snapshot (not a contract, erase or GA tag);
  - **Artboards:** AB26–AB28 → "frozen (PR #, date) · consumed by m6a-06";
  - **flag inventory:** unchanged (the interview cohort gate is a T-1 constant flipped at v2.1.0);
  - **Decisions log:** the P0 audit summary and every v2.1.0 blocker, the coach memory numbers (and any resize PR; a crossing of 70 % recorded as t6 §3's split-out trigger), the host-verify result, the `billing_url` source and date, the M6b prefs hand-off, the dogfood result, the `kubectl exec` of `coach admin interviews --live`.
- No ADR expected (ADR-0032 covers it; m6a-03 recorded the twin-gate tolerance).

## Done when (acceptance)

- [ ] **M6a exit:** a 45-minute text mock survives pause and resume and scores once — e2e scenario A green (the owner's dogfood follows as the post-ship owner event `ev-m6a-dogfood`; it doesn't gate this).
- [ ] Every AB26, AB27, AB28 frame matches the frozen board (1440/390 screenshots in the PR).
- [ ] Grace → paused → resume per spec; 24 h → `incomplete`, unscored, out of trends, no AI call without a click.
- [ ] Explicit accept only; re-propose once; the server's `scored_by` label; exposure → `self`.
- [ ] Accessibility prefs persist, are erased with the account, and default the setup's multiplier.
- [ ] The P0 audit is complete; gaps fixed or recorded as v2.1.0 blockers.
- [ ] `v2.0.N` tagged and verified per the checklist (no live interviews before the tag); status.md updated.

## Ship (land-and-sync — owner approval pre-granted)

> Launching this prompt is the owner's approval for every change it makes (D40); don't stop for review.

1. Branch, then conventional commit(s) with the attribution lines, then push, then a PR in every repo touched (`../infra` PRs first where the order requires it; infra PRs are never folded into a tag). Here, in step 3's order: the sprint PR on `feat/m6a-06-mock-ui-part-2` (board screenshots at 1440/390 and the audit table in the body); a coach-memory PR in `../infra` only if step 8 needs one; the record docs PR on `docs/m6a-06-record` after the tag (step 9).
2. Once CI is green (fix, then merge, on failure), squash-merge. Never enable auto-merge. infra has no CI: paste the local checks (the memory-sum check, `host-verify --cluster`) into the infra PR body and merge on them.
3. **Release action — tag a `v2.0.x` patch (M6a dark, cohort):** in this order:
   1. The sprint PR squash-merged (step 2), then `git checkout main && git pull`.
   2. Walk the release checklist (ADR-0034 §6; the plan's Release checklist), including "from M6: no live interviews": `coach admin interviews --live` reports `live=0` right before the tag (log the exec); confirm the ACL and contract lines are n/a with the plan's two `git diff` checks. No snapshot (not a contract, erase or GA tag).
   3. Push the tag `v2.0.N`, the next free patch (re-check peers' tags first; **never `v2.1.0`**; never move or re-push a tag), titled `v2.0.N — v2.1 build · M6a (dark)` (step 7). Let Flux deploy, then verify live by looking (step 8).
   4. Any coach-memory PR in `../infra` (step 8) is its own PR, merged on its own, never folded into the tag.
   5. The record docs PR (step 9): CI green, squash-merged.
   6. Don't wait for the owner's dogfood (step 10): it's the post-ship owner event `ev-m6a-dogfood`.
4. Update status: the sprint file and `docs/v2/status.md`, in the same PR or a follow-up docs PR merged the same way. Here: the record docs PR after the tag (step 9); leave no open PR and no uncommitted status.md edit behind.
5. Run `git checkout main && git pull` in every repo touched (xlearn, plus `../infra` if step 8 opened a PR). If a clean peer worktree holds `main`, use `git -C <worktree> merge --ff-only origin/main` and then `git switch --detach main`.

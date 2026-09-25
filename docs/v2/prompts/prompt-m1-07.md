# Prompt — Sprint m1-07 · Coach D27 assist capture, mode gate, L18 caps + AB01 → v1.7.0

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-m1-07.md`](../sprints/sprint-m1-07.md) · **Milestone:** M1 (M1b — this sprint cuts `v1.7.0`) · **Prereqs:** [m1-03](../sprints/sprint-m1-03.md), [m1-04](../sprints/sprint-m1-04.md), [m1-05](../sprints/sprint-m1-05.md), [m1-06](../sprints/sprint-m1-06.md), [m1-10](../sprints/sprint-m1-10.md), [ds-m1-01](../sprints/sprint-ds-m1-01.md) (AB01 frozen)

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md) — conventions, land-and-sync, status updates.
- The plan: [`../sprints/sprint-m1-07.md`](../sprints/sprint-m1-07.md) — tasks, tables, acceptance, release checklist.
- [`../rollout-plan.md`](../rollout-plan.md) — [§3](../rollout-plan.md#3-milestone-map) (M1 exit), §4 M1, [§2.2](../rollout-plan.md#22-operating-rules-every-mi-step-and-every-tag) (operating rules), [§7](../rollout-plan.md#7-indicative-tag-timeline) (`v1.7.0` row), [§9](../rollout-plan.md#9-artboards-by-milestone) (AB01).
- [`../research/t5-platform-ai.md`](../research/t5-platform-ai.md) — [§9](../research/t5-platform-ai.md#9-coach-and-byo-changes-in-v2) (mode table, D18 capture, limits) and **§12 D27, which overrides §9**: per problem only; the "no mock or realtime while any attempt is open" rule is dropped.
- [ADR-0031 §7](../../adr/0031-platform-ai-and-two-tier-keys.md#7-coach-byo-changes) (coach changes; the ADR stays *Proposed* — §7 rests on the settled D27), [feasibility decisions log](../feasibility.md#decisions-log-newest-first) (D27, D31).
- [ADR-0029](../../adr/0029-judge-contract-and-learning-signal.md) — the grade table (Assisted includes coach help during the attempt).
- [ADR-0035 §4](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#4-limits-inventory) (L18, L24), §2 (ACL standing rule), [§5](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#5-capacity-the-memory-sum-rule-triggers-and-ordered-responses) (memory sum).
- [ADR-0034](../../adr/0034-v2-release-labelling-gating-and-rollback.md) — §1.6 (tag timeline), [§2](../../adr/0034-v2-release-labelling-gating-and-rollback.md#2-feature-gating-three-tiers-no-flag-service) (flag tiers), [§3](../../adr/0034-v2-release-labelling-gating-and-rollback.md#3-migration-and-event-compatibility-rules) (envelope append-only, contract rules), §6 (release checklist).
- [ADR-0033 §11](../../adr/0033-invite-only-admission-and-owner-admin.md#11-admin-console-isolation-mi-5b) (MI-5b soft gate), [ADR-0020](../../adr/0020-coach-service-realization-and-behaviour-gate.md) and [ADR-0007](../../adr/0007-ai-coach-byo-key-and-secrets.md) (v1 coach).
- [`../research/t1-content-data-model.md`](../research/t1-content-data-model.md) §4 (coach, *Expand → backfill → contract*) and §10 (`withhold()`).
- The AB01 board `design-system/screens/v2/AB01-*.html` (frozen by [ds-m1-01](../sprints/sprint-ds-m1-01.md)), [`theme.css`](../../../design-system/theme.css), [`design-system/README.md`](../../../design-system/README.md), v1 [`Problem.dc.html`](../../../design-system/screens/Problem.dc.html) (coach panel, reference only).
- What M1 already built — read their Status tables and decisions-log lines in [`../status.md`](../status.md):
  - [m1-01](../sprints/sprint-m1-01.md): `internal/course` (the compiled manifest), `coach.persona` / `coach.off_during`, DSA's persona lines, PG 18 compose parity;
  - [mi-05](../sprints/sprint-mi-05.md): `internal/platform/events/topology.go` and the subject-registry / authorization golden (entry gate 4, step 9);
  - [m1-02](../sprints/sprint-m1-02.md): envelope `DecodeEnvelope` (ignores unknown fields), `mock_session_scored_total_check`, `coach_message.path_slug`, `coach.key_default`, the `xlearn:contract` / `xlearn:relax` lint;
  - [m1-09](../sprints/sprint-m1-09.md): curriculum expand — concept `(path_slug, slug)` and `problem_section (problem_id, stage, "order", language)` uniques (step 9's conflict-target check);
  - [m1-03](../sprints/sprint-m1-03.md): course resolution, v2 producers, the weak-area upsert on `(account_id, path_slug, week_of)`, `hack/lint-dropped-columns.sh`, coach `page_context` prefixes;
  - [m1-04](../sprints/sprint-m1-04.md): `identity admin` + `docs/runbooks/identity-admin.md`, CSP;
  - [m1-05](../sprints/sprint-m1-05.md): limits, the L24 list in `services.md`;
  - [m1-06](../sprints/sprint-m1-06.md): `internal/gateway/withhold.go` — `itemState`, the exported `live(s)` predicate, the coach-context strip;
  - [m1-10](../sprints/sprint-m1-10.md): `key_default`, catalog, typed errors with `reason`, `coach_message` usage columns, `internal/coach/contract_test.go`, and the Settings "Your AI coach (your key)" heading.
- Code: `internal/gateway/coach.go` (`handleCoachChat`, `handleCoachThread`, `chatStream` at 96 — the mode goes upstream only today, `coachEnrich` ≈ lines 276–318, `obj["pattern"]` at 313, `relayStream` at 366), `internal/gateway/coach_test.go` (`newCoachHarness`, the httptest pattern to extend), `internal/coach/{prompt.go,handlers.go,store/}`, `internal/practice/{service.go,handlers.go,store/store.go}` (`LogOutcome`), `internal/assessment/{service.go,mock.go,store/queries/mock_session.sql}`, `web/src/components/Coach.tsx`, `web/src/lib/settings.ts`, `web/src/screens/Problem.tsx`, `docs/architecture/{api.md,events.md,services.md,data-model.md,openapi.yaml}`.

## Context

M1 turns DSA into a course like any other with no behaviour change, plus the security floor. `v1.6.0` shipped the
M1a expand and v2-envelope consumers. m1-03 … m1-06 and m1-10 then merged M1b dark on `main`: course resolution with
producers on the v2 envelope, the identity floor, gateway limits, `withhold()` and the coach key work. This sprint
adds the last piece — the coach's v2 behaviour — and cuts **`v1.7.0`**.

- **D27** (owner, 2026-09-24): only a coach chat about **that problem** during its open counted attempt needs a
  confirmation, is recorded on the attempt (fail-closed), and caps the grade at **Assisted**. Asking from another page
  is an accepted, honor-based bypass. The coach stays **locked** during live touches and mocks.
- **Mode gate:** `locked` / `attempt` / `review` / `general`, server-authoritative; `attempt` never sees the
  pattern, concepts or solution facts. In v1, `coach.go:313` forwarded the pattern and `prompt.go:84-86` printed
  it. m1-06 now strips it in the gateway; this sprint makes the mode itself, with a coach-side guard.
- **L18:** 20 msg/min, 2 streams, 300/day, history 20 turns / 32 KiB — bounds what a stolen session can spend from
  a learner's key.

After the tag, M1c ([m1-08](../sprints/sprint-m1-08.md)) drops the v1 columns, and it may only drop what `v1.7.0`
neither reads nor writes. Task 5 is that gate.

## Entry gates — verify first (stop and report if any is unmet)

- [ ] AB01 frozen: ds-m1-01 merged (the merge is the freeze, D40; `ls design-system/screens/v2/AB01*`), status.md artboard row "frozen".
- [ ] m1-03, m1-04, m1-05, m1-06 and m1-10 merged on `main` (their sprint files ✅; `git log origin/main`).
- [ ] `v1.6.0` live (`curl -s https://projects.sujaykumar.dev/xlearn/api/v1/healthz`).
- [ ] NATS topology unchanged since `v1.6.0` (subject-registry / `topology.go` golden) — or, if mi-06 N1 is live and it changed, the ACL PR is merged in `../infra`.
- [ ] Soft: MI-5b live (mi-04). If not, continue and record it in the decisions log.
- [ ] Parallel sessions: `gh pr list --state open`, `git worktree list`, ListAgents — no peer PR edits `internal/coach`, `internal/gateway/coach*.go` or adds practice / coach / assessment migrations without an agreed order.

## Do this (in order)

1. **[X] Branch** `feat/m1-07-coach-d27-mode-gate` from an up-to-date `main`.
2. **[X] practice — D27 capture.** Migration `internal/practice/store/migrations/0000N_m1b_coach_assist.sql`
   (next free version at rebase; expand only, no contract marker): `attempt.coach_assist_at timestamptz NULL`.
   sqlc queries: `MarkCoachAssist` (`UPDATE … SET coach_assist_at = COALESCE(coach_assist_at, now()) WHERE id=$1 AND account_id=$2 AND ended_at IS NULL RETURNING coach_assist_at`),
   `ListOpenAttempts` (by `account_id`, optional `problem_id`). Routes in `internal/practice/service.go`:
   `POST /attempts/{id}/assist` (404 unknown / not `sub`'s, 409 `attempt_closed`, else 200 `{attemptId, coachAssistAt}`, idempotent)
   and `GET /attempts/open[?problem_id=]` → `{attempts:[{attemptId, problemId, pathSlug, purpose, startedAt, stageReached, coachAssistAt}], problem?: {problemId, status, firstSolvedAt}}`
   (`purpose` is `"course"` until M2a; m2-01 reports open touches in the same list as `purpose: "touch"` — no separate `touches` slot).
   In `LogOutcome`: if `coach_assist_at` is set and ≤ the outcome's `logged_at`, record `clean`/`rough` as
   `assisted` (`miss` unchanged), recompute `below_clean`, return `cappedBy: "coach"`; add
   `"assist": {"hint": stage_reached != "attempt", "coach": coach_assist_at != null}` to the `problem_solved` data.
   Add `coachAssistAt` to `GET /state/{problemId}`. Store tests: idempotency, the clamp matrix, the payload.
3. **[X] assessment — live mock read.** `GET /mocks/live` (JWT-scoped): the account's `status='live'` session with
   `deadline_at > now()` or `{"live": null}`; query in `store/queries/mock_session.sql`; handler test.
4. **[X] gateway — the gate** (`internal/gateway/coach_gate.go` + `coach_gate_test.go`, wired into
   `handleCoachChat`). Per chat, in parallel with short timeouts: practice `/attempts/open?problem_id=` (only for
   `problem:<id>` contexts; account-wide list otherwise), review's due queue (as m1-06's `itemStates` reads it) and
   assessment `/mocks/live`. Build the item's `itemState` and use m1-06's exported `live(s)` predicate
   (`internal/gateway/withhold.go`) — m1-06 already strips pattern / concepts / facts from the coach context; keep
   that. Then, in this order:
   - any lookup fails → 503 `coach_state_unavailable` (nothing sent);
   - live mock (or, from M2a, an open `purpose: "touch"` attempt) and the course manifest's `coach.off_during` lists it → 409 `coach_paused` `{reason}`;
   - `problem:<id>` with open `purpose: "course"` attempt `A` on that problem:
     - `A.coachAssistAt` already set → no confirm (already capped; a reload must not re-prompt);
     - otherwise no/wrong `assist_ack` → 409 `assist_confirm_required` `{attemptId, problemId}`;
     - ack = `A.id` → coach `GET /coach/admission` first (step 6; a 429 is relayed as is and the attempt is **not**
       capped), then practice `POST /attempts/{A.id}/assist`; failure → 503 `assist_unavailable`;
   - mode: `attempt` when `live(s) || !s.Solved`, `review` otherwise, `general` for non-item pages (with `live_items`);
   - rewrite the body: strip `assist_ack`; `pattern` only in `review`; `live_items` in `general`; forward with
     `X-Coach-Mode`, `X-Coach-Course` (resolved `path_slug`) and `X-Coach-Attempt` (when an assist is recorded).
   **Mode to the client:** set `X-Coach-Mode` on the relayed chat response too, and add
   `gate: {mode, reason?, attempt?: {attemptId, coachAssistAt}}` to `GET /api/coach/thread?context=`
   (`handleCoachThread`), from the same lookup run read-only (no assist write, never a 409; omit `gate` if a lookup
   fails — the thread still loads). `relayStream` copies `Retry-After`. The coach path no longer calls `/state/{id}`.
   Table tests on `newCoachHarness` (`internal/gateway/coach_test.go`; there is no `fakes_test.go` in the gateway):
   no ack → 409 and coach never called; practice 500 after ack → 503, coach never called; probe then assist, each
   exactly once and before the chat; admission 429 → relayed, practice never called; `coachAssistAt` set → no 409,
   no assist call; other-problem context → no confirm; live mock → 409 `coach_paused`; pattern absent in
   `attempt`; `Retry-After` relayed; `X-Coach-Mode` on the response; `gate` on the thread GET (and omitted on a
   lookup failure).
5. **[X] coach — prompt v2, persona, store.** `internal/coach/prompt.go`: `const PromptVersion = "coach-prompt@2"`,
   order frame → persona (from `internal/course` by `X-Coach-Course`; DSA's persona = v1's course lines at `prompt.go:56-57`, per m1-01) →
   mode rules as hard constraints (incl. the `general` guard naming `live_items`) → context; ignore `Pattern` unless
   `review`. Migration `internal/coach/store/migrations/0000N_m1b_prompt_v.sql`: `coach_message.prompt_v text NULL`,
   `coach_message.attempt_id uuid NULL`; `AppendMessage` writes `prompt_v`, `path_slug`, `attempt_id` on both rows.
   Golden prompt snapshots per mode in `internal/coach/testdata/prompt/`.
6. **[X] coach — L18.** `internal/coach/limits.go` (+ tests): per-account concurrency (2, released by `defer`) → 429
   `coach_busy` `Retry-After: 5`; token bucket 20/min → 429 `coach_rate_limited`; migration adding
   `coach.message_quota_day(account_id uuid, day date, n int NOT NULL, PRIMARY KEY (account_id, day))` with the atomic
   conditional upsert (`… WHERE message_quota_day.n < 300 RETURNING n`; no row ⇒ 429 `coach_daily_cap`,
   `Retry-After` to the next UTC midnight). Check order: streams → per-minute → daily, all **before** persisting the
   user turn or decrypting the key. Add `GET /coach/admission` (JWT-scoped, read-only): the same three checks
   without consuming anything → 204 or the same typed 429 (test that it leaves the bucket and `n` unchanged).
   `ThreadHistory` last 20 messages; `buildTurns` trims to ≤ 32 KiB keeping the current turn. Constants, not env
   flags. Race test on the upsert. The day stays the **UTC day**; the SPA shows the reset as a local clock time.
7. **[X] web — AB01 F1–F10** (m1-10 built F11–F15). **The board's copy wins** over any string in the plan or here.
   `web/src/lib/settings.ts`: `assist_ack` on the body; `CoachChatError` gains `status`, `retryAfter`, `attemptId`,
   `reason`; type the thread's `gate` and read `X-Coach-Mode` from the chat response. `web/src/components/Coach.tsx`,
   exactly as AB01 frames them:
   - F1/F6/F7 mode chip (from `gate.mode`, then `X-Coach-Mode`; no chip without `gate`);
   - F2 confirm → resend with the ack (no client memory: once recorded, the gateway skips the confirm);
   - F3 recorded note (from `gate.attempt.coachAssistAt` or after an acked send), and invalidate the `["problem", id]`
     query so the Problem HUD chip (`TimerHUD`, from `/state`'s `coachAssistAt`) appears;
   - F4 fail-closed 503;
   - F5 paused;
   - F8 three 429 variants, the daily one showing the reset as a **local clock time** from `Retry-After` (the board
     says "00:00 in your timezone", but the day is UTC). Flag this copy delta to the owner in the PR;
   - F9 cut-short marker from `done.truncated` with **[Continue]** (sends "Continue from where you stopped." through
     the normal send path);
   - F10 provider limited (m1-10's typed `reason`; the key stays on);
   - the "Your AI coach (your key)" naming.

   No layout AB01 doesn't show; no inline styles (CSP). `web/src/screens/Problem.tsx`: the HUD chip and the capped
   outcome line from `coachAssistAt` / `cappedBy`. Tests in `Coach.test.tsx` and `Problem.test.tsx`; screenshots at
   1440 and 390 px next to AB01 F1–F10.
8. **[X] Docs.** [`api.md`](../../architecture/api.md) + [`openapi.yaml`](../../architecture/openapi.yaml) (coach chat 409/429/503 codes, `assist_ack`),
   [`events.md`](../../architecture/events.md) (`problem_solved` `assist`), [`services.md`](../../architecture/services.md)
   (practice/assessment new routes, coach `/admission`; coach joins the L24 scale-out list), [`data-model.md`](../../architecture/data-model.md)
   (`attempt.coach_assist_at`, `coach_message.prompt_v/attempt_id`, `message_quota_day` and m1-02's `key_default` —
   both erase-by-`account_id`, which [l-01](../sprints/sprint-l-01.md)'s coach delete list doesn't name yet; see Update status).
   `api.md` / `openapi.yaml` also get the thread's `gate` and the chat's `X-Coach-Mode` response header.
9. **[X] M1b exit + M1c readiness (task 5).** Run the full suite: `gofmt -l`, `go vet ./...`, `go test -race ./...`,
   `XLEARN_TEST_DATABASE_URL=… go test -tags e2e ./internal/e2e/...` (without the env var the e2e tests silently
   skip — see `internal/e2e/coreloop_test.go`), `sqlc generate && sqlc diff`, web typecheck / lint / test / build
   (then `git checkout -- web/dist/.gitkeep`), `hack/lint-migrations.sh`, topology golden, route-enumeration,
   public allowlist, OpenAPI drift. Walk compose (`docker compose up --build`; dev sign-in) against v1 and list every
   visible delta. The **expected** ones are the D27 / ADR-0031 §7 family and D31: the assist confirm, capped note,
   HUD chip and outcome line; `coach_paused` during a live mock (v1 mounted the coach on every page, mocks included);
   the fail-closed 503 notices; the mode chip; the BYO naming; the L18 notices; [Continue]; the public mock count
   only. Flag anything outside that list, and never "fix" an item on it. Then the M1c check over HEAD: m1-03's
   `hack/lint-dropped-columns.sh` with coach `is_default` taken **off** its allowlist (m1-10 removed every use; its
   `internal/coach/contract_test.go` must pass), plus a manual grep of what that script doesn't scan —
   `internal/*/store/gen/*.go`, raw SQL in `internal/*/store/*.go`, the curriculum loader — for `is_reinforcement`,
   `leetcode_url`, `neetcode_url`, `code_template`, `total_35`, `is_default` → zero hits (event-payload / JSON names
   are fine); no v1 conflict target left — `ON CONFLICT (account_id, week_of)` on `review.weak_area_snapshot`
   (m1-03 switched it), `ON CONFLICT (slug)` on `curriculum.concept` (m1-09 switched it) and
   `ON CONFLICT (problem_id, stage, "order")` on `curriculum.problem_section` (v1 `section.sql:15`; nothing has switched
   it yet: target m1-09's `(problem_id, stage, "order", language)` or a plain insert after the delete-then-reinsert).
   **Switch any that remain here**, with a test, so m1-08 can drop all three uniques; m1-02's `mock_session_scored_total_check`
   is present; category / dimension validation exists with 422 tests (add it if missing); no migration since
   `v1.6.0` carries `-- xlearn:contract`. Paste the results into the PR body.
10. **[X] PR → CI green → squash-merge** (see Ship). Conventional title, e.g. `feat(coach): D27 assist capture, mode gate, L18 caps (m1-07)`,
    with the attribution lines. Fix-then-merge on red.
11. **[X] Tag `v1.7.0`** with the release checklist in the plan (verbatim): peers checked; next free minor; major =
    `.release-line`; no ACL PR needed (topology unchanged); no new caller or pod. Push the tag, create the GitHub
    release **`v1.7.0 — v2 build · M1b`**. After Flux rolls: healthz reports the version; `ssh vps 'k3s kubectl get deploy -n xlearn -o wide'`
    shows the new images; ImagePolicies' latest = the tag, HelmReleases Ready; smoke login, dashboard, coach
    (confirm shows on an open attempt; reply streams after confirming).
12. **[H] `ev-owner-role`**, run by you (D40: the launch approves this production operation). Find the owner's account with
    `ssh vps 'k3s kubectl exec -n xlearn deploy/xlearn-identity -- identity admin account list'` (production has exactly one
    account, the owner's, until L-E's first tester; keep the output in the terminal), then run
    `ssh vps 'k3s kubectl exec -n xlearn deploy/xlearn-identity -- identity admin account set-role <owner> owner'`;
    confirm with `… identity admin account list --role owner` and log the CLI use in status.md. If the owner's account can't
    be told apart, don't guess: record task 7 ⛔ "owner account ambiguous" and carry on.

## Constraints

- **Service boundaries ([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)):** practice owns `coach_assist_at` and the clamp; assessment owns the live-mock read; coach owns prompt, history and L18; the gateway only orchestrates. No cross-schema reads.
- **goose + sqlc:** expand-only migrations (no contract marker — the lint will fail otherwise), next free version per service at rebase; commit `sqlc generate` output; `sqlc diff` clean.
- **Events:** `assist` is additive on an existing subject; envelope append-only; decoders forever. No new subject/stream/consumer in this tag (else an ACL PR merged before the tag, [ADR-0035 §2](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#2-nats-auth-nkey-users-fine-acls-server-first)); consumers before producers.
- **Fail closed:** never forward a chat when the attempt state or the assist record is unknown.
- **Frontend:** `theme.css` tokens/components verbatim, dark theme, AB01 is the spec; CSP-safe (no inline style/script).
- **GitOps:** no `kubectl apply`; `ssh vps` is read-only except the sanctioned admin CLI via `kubectl exec` (this session runs `set-role` once, D40).
- **D34:** no alert, timer, CronJob, push channel or Flux Alert — verification is by looking. **D12:** no backups/object store.
- **Memory-sum rule:** no new always-on pod in this tag; if that changes, check `host-verify --cluster` and add a memory limit.
- **Parallel sessions:** check peers' PRs, tags and worktrees (and ListAgents) before tagging and before claiming an ADR number; re-check right before pushing the tag.

## Deliverables

- practice: `coach_assist_at` migration; `GET /attempts/open`; `POST /attempts/{id}/assist`; the Assisted clamp; `problem_solved.assist`; `coachAssistAt` on `/state`.
- assessment: `GET /mocks/live`.
- gateway: `coach_gate.go` (modes, D27 409/503 with the skip-once-recorded rule, admission probe, `coach_paused`, headers), `X-Coach-Mode` on the chat response, `gate` on the thread GET, `Retry-After` relay; tests.
- coach: `coach-prompt@2` + persona, `prompt_v` / `attempt_id`, `limits.go` + `message_quota_day` + `GET /coach/admission`, history cap; golden prompt snapshots.
- web: AB01 F1–F10 in `Coach.tsx`, the HUD chip and capped outcome line in `Problem.tsx`; tests.
- Docs: api.md, openapi.yaml, events.md, services.md (L24), data-model.md.
- Tag `v1.7.0`, deployed and verified; owner role set.

## Update status

- Set each task in [`../sprints/sprint-m1-07.md`](../sprints/sprint-m1-07.md) 🔄 / ✅ / ⛔ as you go; _Overall_ ✅ at the end.
- [`../status.md`](../status.md): Sprint board row; **Milestones** (M1b ✅ `v1.7.0`); **milestone → tag → rollback floor** (`v1.7.0` → **1.6.0**, snapshot n/a); **flag inventory** starts — `SIGNUP_MODE` under operating modes (permanent); artboards (AB01 consumed); the owner-role CLI use; the "first path-aware release = `v1.7.0`" note (consumer floor for non-DSA events).
- **Decisions log:**
  - the fail-closed 503 choice;
  - the UTC day for the daily cap, and the F8 copy delta flagged to the owner (reset shown as a local clock time);
  - **the residual cap-on-429 race** (probe passes, then a concurrent send takes the last slot, at most once per
    attempt), recorded as this sprint's accepted decision, not attributed to t5 §9;
  - MI-5b state at tag time;
  - the M1c-readiness result (hand-off to m1-08), including which conflict targets were switched here;
  - the **erase hand-off**: "l-01 coach erase must also delete `coach.message_quota_day` and `coach.key_default`
    (verify against the schema)";
  - the unowned `review`-mode slots (final submission, AI feedback), deferred until a sprint is assigned.

  Record an ADR only for a genuinely new decision, after checking the next free number with peers.
- **If the session runs long:** stop once the PR is squash-merged (it ships dark; 1.x deploys only on a tag), mark
  tasks 5–7 ⬜ with a note, and finish from step 9 in a continuation session that ends with the same Ship section. That's
  a time stop, not an owner wait. Never tag at the end of a rushed run.

## Done when

- [ ] During an open counted attempt on problem P, a chat in P's context without `assist_ack` → 409 `assist_confirm_required`; with it, the admission probe passes and practice records `coach_assist_at`, both before coach is called; an admission 429 leaves the attempt uncapped; practice down → 503, nothing sent; once recorded, no further confirm.
- [ ] An assisted attempt logs Clean/Rough as Assisted; `problem_solved` carries `assist{hint, coach}`; v1/v2 consumers accept it; other contexts unaffected.
- [ ] Live mock → 409 `coach_paused`; `attempt` prompts carry no pattern / concepts / facts; `review` / `general` per the table; `prompt_v = coach-prompt@2`; the mode reaches the SPA (`X-Coach-Mode`, thread `gate`).
- [ ] L18 typed 429s with `Retry-After`; history ≤ 20 turns / 32 KiB.
- [ ] Coach panel matches AB01 F1–F10 at 1440 and 390 px; the F8 reset-copy delta flagged to the owner.
- [ ] M1b live: every visible change vs v1 is on the expected-change list (step 9); every v1 e2e green.
- [ ] No reader or writer of an M1c-drop column in `v1.7.0`.
- [ ] `v1.7.0` verified; floor 1.6.0 and the flag inventory recorded; owner role set.

## Ship (land-and-sync — owner approval pre-granted)

> Launching this prompt is the owner's approval for every change it makes (D40); don't stop for review.

1. Branch `feat/m1-07-coach-d27-mode-gate`, then conventional commit(s) with the attribution lines, then push, then the PR. No `../infra` PR is expected; if entry gate 4 required an ACL PR, it goes first, as its own PR, merged before the tag.
2. Once CI is green (fix, then merge, on failure), squash-merge. Never enable auto-merge.
3. **Release action — tag `v1.7.0`** (the next free minor): walk the release checklist (ADR-0034 §6, in the plan), push the tag, let Flux deploy, then verify live by looking (step 11), and set the owner's role (step 12, `ev-owner-role`).
4. Update status: the sprint file and `docs/v2/status.md`, in the same PR or a follow-up docs PR merged the same way (the tag record and the owner-role CLI log need the follow-up).
5. Run `git checkout main && git pull`. If a clean peer worktree holds `main`, use `git -C <worktree> merge --ff-only origin/main` and then `git switch --detach main`.

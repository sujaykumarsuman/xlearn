# Prompt — Sprint l-02 · L-E erase producer + `DELETE /api/me` + UI (AB21) → v1.12.0

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-l-02.md`](../sprints/sprint-l-02.md)   ·   **Milestone:** L (L-E, producer half)   ·   **Prereqs:** [l-01](../sprints/sprint-l-01.md) (v1.11.0 live), [ds-l-01](../sprints/sprint-ds-l-01.md) (AB21 frozen), [mi-04](../sprints/sprint-mi-04.md) (MI-5b), [m1-04](../sprints/sprint-m1-04.md) (admin CLI, session fields)

## Before you launch (owner)

Launching this prompt attests these are done (D40). If one turns out to be missing, land everything that doesn't depend on it and record the gap as ⛔ in `status.md`; don't wait.

- [ ] **`ev-snap-v1.12.0`:** take a Hostinger manual snapshot in hPanel (one at a time, 1-day retention) right before you launch, and put its time or id in your launch message. The erase tag lands the same day.
- [ ] Optional: a browser profile signed in to xLearn as you, that the session can drive, for the post-tag check that your Settings shows the owner-refused card (F7, no button). Without it, the tests cover that path.

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md): conventions and the land-and-sync rule. This sprint **tags an erase release**.
- The plan: [`../sprints/sprint-l-02.md`](../sprints/sprint-l-02.md). Its transaction steps, the `DELETE /me` response table, task 6's merge-then-tag order, the release checklist and the rollback are authoritative.
- [`../status.md`](../status.md): l-01's record (v1.11.0, the consumers, coach's nkey), the MI-5b row, the tag → floor rows, the artboard rows and any "merged, untagged" lines from peers.
- **Erase design:** [t1 §6.6](../research/t1-content-data-model.md#66-erase-path-none-exists-today-no-delete-apime-in-bffgo) step 2; [ADR-0027 §6](../../adr/0027-content-evalpack-and-user-data-model.md#6-account-erase-v20) as amended by [ADR-0028 §3](../../adr/0028-object-storage-and-backups.md#3-amendments-to-adr-0027).
- **Authz:** [ADR-0033 §7](../../adr/0033-invite-only-admission-and-owner-admin.md#7-roles-and-status-live-in-identitys-database-never-in-the-jwt) (owner never on the web; testers from L-E), [§8](../../adr/0033-invite-only-admission-and-owner-admin.md#8-the-owner-admin-cli-identity-admin-run-via-kubectl-exec) (`account erase` verb), [§9](../../adr/0033-invite-only-admission-and-owner-admin.md#9-erase-and-abuse-controls) (typed confirmation, session < 5 min, seat, username cooldown), [§11](../../adr/0033-invite-only-admission-and-owner-admin.md#11-admin-console-isolation-mi-5b) (MI-5b before the first tester), [§13](../../adr/0033-invite-only-admission-and-owner-admin.md#13-public-dashboard-authz-deltas) (erase → 404, 60-day hold); [ADR-0035 §4](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#4-limits-inventory) L8.
- **Releases:** [ADR-0034 §4.2](../../adr/0034-v2-release-labelling-gating-and-rollback.md#42-r-d-is-a-procedure-not-a-button) (R-d; list erases since the snapshot), [§4.3](../../adr/0034-v2-release-labelling-gating-and-rollback.md#43-snapshot-rule) (snapshot before an erase tag), [§4.4](../../adr/0034-v2-release-labelling-gating-and-rollback.md#44-reversibility-by-step) (L-E row), [§6](../../adr/0034-v2-release-labelling-gating-and-rollback.md#6-release-checklist); [rollout §7](../rollout-plan.md#7-indicative-tag-timeline) (v1.12.0, floor 1.11.0), [§10](../rollout-plan.md#10-public-dashboard-tasks) (P11), [§2.2](../rollout-plan.md#22-operating-rules-every-mi-step-and-every-tag) (sanctioned `kubectl exec`).
- **The frozen board:** `design-system/screens/v2/AB21-erase-account.html` and its screenshots under `design-system/screens/v2/shots/` (merged by [ds-l-01](../sprints/sprint-ds-l-01.md)); read the merged PR's "Decisions to confirm" (the merge froze each item's stated default, D40) and any owner answer recorded since in status.md.
- **Sibling plans:** [l-01](../sprints/sprint-l-01.md) (the contract, fixture, `ErasureAckServices()`, `erase_request`/`released_username`, `identity admin erasures`, the e2e), [m1-04](../sprints/sprint-m1-04.md) tasks 1 and 5 (session fields; the CLI and `admin_audit`), [m1-05](../sprints/sprint-m1-05.md) task 7 (P11: no positive status cache).
- **Code:**
  - `internal/identity/{handlers.go,service.go,usernames.go,username.go,session.go}` (`currentSession`, `clearSessionCookie`), `internal/identity/store/`, `internal/identity/admin/`, `cmd/identity/main.go`;
  - `internal/platform/events/testdata/account_erasure_requested.v2.json`, `topology.go`;
  - `internal/gateway/bff.go` (`apiRoutes`, `handleAuthSignup`/`handleAuthLogin` proxy pattern, `handleMe`), `internal/gateway/public.go`, the cache epoch; `docs/architecture/openapi.yaml` + `internal/gateway/openapi_drift_test.go`;
  - `web/src/screens/Settings.tsx` (the rail, "Sign-in & security"), `web/src/screens/Auth.tsx`, `web/src/lib/settings.ts`, their tests;
  - `internal/e2e/erase_test.go` (from l-01); `docs/runbooks/identity-admin.md` (from m1-04).

## Context

- [l-01](../sprints/sprint-l-01.md)'s v1.11.0 bound every erase consumer (practice, review, assessment, coach) and identity's ack path;
  nothing can trigger them yet. **This sprint adds the producer** and turns web erase on **for testers only**:
  - identity's erase transaction (delete the account; open `erase_request` with the expected acks; hold the username; emit
    `account_erasure_requested`, byte-equal to l-01's fixture);
  - `DELETE /api/me` — identity checks everything from the session itself; the gateway proxies it;
  - `identity admin account erase` for the owner and suspended accounts;
  - the AB21 Settings UI.
- There are **no backups** (D12): an erase is final, and a snapshot restore (R-d) would resurrect every erase after the snapshot. So this
  is an **erase tag**: `host-verify --cluster` green and the owner's manual snapshot, taken right before launch (`ev-snap-v1.12.0`, D40).
- The **owner is never erasable on the web**. You mint the first **tester** yourself (`ev-first-tester`, CLI-minted only after MI-5b;
  D40) and run the prod acceptance with it.
- Floor after v1.12.0: **1.11.0**. Gate state: web erase testers only; owner refused (CLI only).

## Entry gates — verify first (stop and report if any is unmet)

- [ ] **v1.11.0 live, consumers bound:** healthz ≥ 1.11.0; JetStream shows `practice-erase`, `review-erase`, `assessment-erase`, `coach-erase` on `XLEARN_IDENTITY` and `identity-erase-acks` on each service stream; coach on its nkey (l-01's status record).
- [ ] **AB21 frozen:** ds-l-01's PR is merged on `main` (the board file exists).
- [ ] **m1-04 live (v1.7.0):** `identity admin` exists; session-validate returns `role`, `status` and the session's `created_at`; the owner role is set.
- [ ] **MI-5b live** (mi-04's status row) — required before any tester exists (step 12 mints one).
- [ ] **The launch message carries the `ev-snap-v1.12.0` snapshot's time or id** (the before-launch block).
- [ ] **Parallel sessions:** `git ls-remote --tags origin`, `gh pr list`, `git worktree list`, ListAgents → the **next free minor** (use it instead of 1.12.0 everywhere if taken); no open peer PR edits `internal/identity/`, `internal/gateway/bff.go`, `docs/architecture/openapi.yaml` or `web/src/screens/Settings.tsx`.

## Do this (in order)

1. **[X] Branch** `feat/l-02-erase-producer` from an up-to-date `main`.

2. **[X] Erase transaction** — `internal/identity/erase.go` + `store.EraseAccount(ctx, target, via)`, one transaction, exactly the plan's
   steps: lock the row → guards (web: owner → `erase_owner_cli_only`; role outside the code allow-list `{tester}` → `erase_not_available`;
   CLI: refuse the last active owner) → `erase_request` with `expected_acks = events.ErasureAckServices()` (the one-open index →
   `erase_in_progress`) → `released_username` (sha256 of the normalized lower-case name, 60 days) → delete identity's own outbox rows
   for the account → null any PII in `admin_audit.detail` for the target → `DELETE FROM identity.account` (cascades sessions, OAuth
   links, onboarding, enrollments) → outbox `account_erasure_requested` **byte-equal to the fixture**, `event_id = uuidv5(ns,
   "account_erasure_requested:"+id)`, `outbox.account_id = NULL`. Store integration tests for every step and guard.

3. **[X] `DELETE /me` + `DELETE /api/me`.**
   - identity `DELETE /me` (session-cookie route, read with `currentSession`): the plan's response table — 401 / 403 `erase_owner_cli_only` /
     403 `erase_not_available` / 403 `reauth_required` (session `created_at` ≥ 5 min ago) / 400 `confirmation_mismatch` (username, else email;
     case-insensitive, trimmed) / 409 `erase_in_progress` / **202** `{erase_request_id, username_available_at}` with the session cookie cleared.
   - gateway `{"DELETE", "/api/me", g.handleDeleteMe, true}`: fetch-based proxy like `handleAuthSignup` (cookie in, `Set-Cookie` out); the
     JSON/`Sec-Fetch-Site` checks and the L5 mutating bucket apply; on 202 bump the in-process public-cache epoch.
   - `GET /api/me` gains `erase_web` (`allowed` | `owner_cli_only` | `not_available`) and `session_fresh_until`.
   - Update `docs/architecture/openapi.yaml` (drift test) and `docs/architecture/api.md`. Handler + proxy tests for every row.

4. **[X] CLI** `identity admin account erase <user> --confirm <user>` in `internal/identity/admin/`: `EraseAccount(…, via=cli)`; prints the
   request id, expected acks and the `identity admin erasures --open` hint; `admin_audit` detail `{erase_request_id}` only; tests.

5. **[X] P11.** A warm-cache test: erase (web path, then CLI path) → the very next `GET /api/u/<username>` is the uniform 404. Username
   hold in `handleSetUsername`/`handleUsernameAvailable`: a held hash with `available_at > now()` looks exactly like a taken name;
   tests at day 0, 59, 61 (injected clock).

6. **[X] e2e on the real producer.** Switch `internal/e2e/erase_test.go` from l-01's helper to `DELETE /me` (tester, fresh session) and the
   CLI's store call: fan-out → rows gone in all four services → 4 acks → `closed_at`; owner → 403 and nothing deleted; learner → 403;
   stale session → `reauth_required`; mismatch → 400.

7. **[X] UI (AB21).** In `web/src/screens/Settings.tsx`, at the frozen board's placement: the Erase account card (What's erased, the 60-day
   username line), **Sign in again** when `session_fresh_until` has passed or the server says `reauth_required` (signs out; returns to
   Settings after login; no PII in the URL), the typed-confirmation modal (Cancel default focus; the danger button only on a match;
   mismatch; "Erasing…"), the owner-refused and tester-only states **with no button**, other errors; on 202 → `/xlearn/auth?erased=1`
   and `Auth.tsx` shows the erased banner. `useEraseAccount()` in `web/src/lib/settings.ts`. `theme.css` verbatim; match the board's
   copy exactly. Tests in `Settings.test.tsx` and `Auth.test.tsx`; check 390 px.

8. **[X] Runbook.** Add the "Erase" section to `docs/runbooks/identity-admin.md`: the CLI verb; `identity admin erasures --open`; a stuck
   erase (read the missing service's `event_dead_letter` and logs, fix, let the durable redeliver — only within its ~8 h backoff window; a
   dead-lettered erase is not redelivered and has no re-emit verb yet: record and raise it; never close by hand); **R-d step 3**
   (before a restore: `identity admin erasures --since <snapshot time>`; after it: re-run each `account erase <id> --confirm <id>`; record
   the lost-writes window); pseudonymous stream copies (historic `account_created` carries `display_name`) and the on-request re-seal via
   NATS break-glass. Update `docs/architecture/services.md` (identity: erase producer; `DELETE /me`).

9. **[X] PR, CI green; merge it in step 11, right before the tag.** Conventional commits with the attribution lines; CI green (`go test ./...`, `sqlc diff`,
   OpenAPI drift, web tests, `e2e`). **Keep the squash-merge for step 11** (after the rehearsal and `host-verify`), back to back with the
   tag: `deploy.yml` builds only on `v*` tags, so a merged-but-untagged erase producer on `main` would ship with whichever tag comes next
   (a parallel M3 sprint's, or a v1 patch) with no `host-verify`, no snapshot and possibly out of version order.

10. **[X] Compose rehearsal at the PR head** (the exact commit you will merge): `docker compose up`;
    `identity admin account create --role tester --email t@example.test` (inside the identity container); sign in as the tester; create an
    attempt, a mistake, a scored mock and a coach thread; sign in again; erase from Settings; check row counts 0 in practice, review,
    assessment and coach, 4 acks and `closed_at`, the profile 404, the username unavailable. Paste the transcript into the PR.

11. **[H] `host-verify`, then merge and tag back to back.** `ssh sujaykumar-vps 'bash -s -- --cluster' < ../infra/hack/host-verify.sh` → green; the
    host has settled. Record **`ev-snap-v1.12.0`** (taken by the owner before launch) from the launch message: its id and time. **Then
    at once:** re-check peers' tags and PRs; squash-merge the PR (still the rehearsed head — if `main` moved, rebase, re-run CI and the
    rehearsal); run the plan's *Release* checklist; tag the merge commit; GitHub release title **`v1.12.0 — v2 build · L-E erase`**,
    notes per the plan. The tag lands the same day as the snapshot. After the tag, verify by looking: healthz, images, ImagePolicy
    latest = tag, HelmReleases Ready, smoke login/dashboard/coach, and — in the owner's signed-in profile, if one is available — his
    Settings shows the owner-refused card (no button).

12. **[H] Prod acceptance with a throwaway tester** (`ev-first-tester`, D40). After checking MI-5b in status.md, mint it:
    `ssh sujaykumar-vps 'k3s kubectl exec -n xlearn deploy/xlearn-identity -- identity admin account create --role tester --email <an unused @example.test address>'`.
    Keep the one-time password in the terminal only (never in a file, log, PR or status.md). Operate the tester yourself in a private
    window or separate browser profile: sign in → claim a username → create a little data → sign in again → erase from Settings. Then,
    through the sanctioned admin CLI and read-only views only (no raw `psql` into other services' schemas):
    `ssh sujaykumar-vps 'k3s kubectl exec -n xlearn deploy/xlearn-identity -- identity admin erasures --since <today>'`
    → **closed with 4 acks** — each service writes its ack in the same transaction as its deletes, so 4 acks prove all four services
    erased; `/xlearn/u/<username>` → 404; the username unavailable (from the owner's signed-in profile if available); no ERROR on the
    erase paths in the five services' logs (`k3s kubectl logs -n xlearn deploy/xlearn-<svc> --since=30m`, read-only). **Never send
    `DELETE /api/me` from the owner's account.** Log each admin-CLI `kubectl exec` in status.md.
    **If a tester step needs a human** (a tool or rule stops you, e.g. typing the password), don't wait: CLI-erase the tester
    (`identity admin account erase <id> --confirm <id>`), record step 12 ⛔ "tester run needs a human" with these steps in
    `status.md` → Open owner items, and land everything else.

13. **[X] Record** — see *Update status*.

## Constraints

- **Consumers before producers:** the consumers shipped in v1.11.0; this tag only adds the producer. The event's subject and payload must match l-01's constants and fixture byte for byte ([ADR-0034 §3](../../adr/0034-v2-release-labelling-gating-and-rollback.md#3-migration-and-event-compatibility-rules)).
- **identity is the authority** for every L8 check, read from its own session row — never from a gateway header or a JWT claim.
- **The owner is never erasable on the web**; learners stay refused until [l-04](../sprints/sprint-l-04.md); the allow-list is a code constant, not an env flag (keep the flag inventory small).
- **Service boundaries ([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)):** identity deletes only its own schema; the other services erase through their consumers.
- **Outbox:** the erase and its event in one transaction; deterministic event id; `outbox.account_id = NULL` on the event row.
- **goose + sqlc:** no migration is expected (l-01 created the tables); if one is needed it is expand-only with the next free version; `sqlc diff` clean. **Never run `Down` in prod.**
- **Erase tag rules:** `host-verify --cluster` green, host settled, **snapshot taken** (by the owner, right before launch, D40); the squash-merge and the tag follow each other at once, the same day ([ADR-0034 §4.3](../../adr/0034-v2-release-labelling-gating-and-rollback.md#43-snapshot-rule)); the producer never sits on `main` untagged. No off-node `pg_dump`.
- **GitOps:** never `kubectl apply`; the only prod writes outside GitOps are the sanctioned `kubectl exec` admin CLI runs, each logged in status.md; no raw `psql` into service schemas. Never move or re-push a tag.
- **Frontend:** `theme.css` verbatim; match the frozen AB21 board; keep the dark theme.
- **No new pod, no new caller** — the memory sum and NetworkPolicies are unchanged. **No alerting (D34):** stuck erases are read with `identity admin erasures --open`.
- **Logs carry ids only**; no username, email or confirmation value in logs or `admin_audit`.
- **Parallel sessions:** check peers' tags, PRs and worktrees (+ ListAgents) before tagging or numbering an ADR, and again right before the tag push.

## Deliverables

- identity: `EraseAccount` (web + CLI), `DELETE /me`, the `/api/me` fields, the username hold on claim/availability, `identity admin account erase`.
- gateway: `DELETE /api/me` + the epoch bump; `openapi.yaml` and `api.md` updated.
- web: the AB21 erase section in Settings; the erased banner on the auth page; tests.
- `internal/e2e/erase_test.go` on the real producer; the compose rehearsal transcript.
- The "Erase" section in `docs/runbooks/identity-admin.md` (incl. R-d step 3).
- Merge + tag **v1.12.0** back to back after `host-verify` (the snapshot was taken before launch); the tester acceptance run recorded (or ⛔ with its steps).

## Update status

- [`../sprints/sprint-l-02.md`](../sprints/sprint-l-02.md): task rows ✅ as they land; _Overall_ ✅ after step 12.
- [`../sprints/sprint-ds-l-01.md`](../sprints/sprint-ds-l-01.md): confirm its rows and _Overall_ read ✅ (its own merge closed it under D40); repair if missing.
- [`../status.md`](../status.md):
  - **Sprint board** l-02 ✅ (confirm ds-l-01 ✅); **Milestones** L: "L-E done (v1.11.0 → v1.12.0)" (L stays 🔄 until the L exit);
  - **Artboards:** confirm AB19, AB20, AB21 read "frozen (PR #, date)" (repair if missing);
  - **Owner events:** `ev-snap-v1.12.0` ✅ (id/time); `ev-first-tester` ✅ (minted and erased by this session);
  - **Tag → floor → snapshot:** v1.12.0 → floor **1.11.0** → the `ev-snap-v1.12.0` snapshot id/time;
  - **Gate state:** web erase testers only; owner refused (CLI only);
  - **Erase log:** the tester's erase (request id, date, via web, 4 acks, closed at);
  - **Manual-path log:** every admin-CLI `kubectl exec` of step 12;
  - **Flag inventory:** none added;
  - **Decisions log:** L8 checks read from identity's own session (no JWT claim); the web-erase allow-list as code; `expected_acks` snapshot; username hold indistinguishable from taken; owner refusal verified by tests, never on prod.
- Note on the board that [l-03](../sprints/sprint-l-03.md) is unblocked.
- An ADR only if you depart from ADR-0033/0034 (check peers before numbering).

## Done when (acceptance)

- [ ] A tester erases on the web; all 4 acks arrive and the request closes; the profile 404s at once; the username is held for 60 days.
- [ ] The owner is refused on the web (tests; no button on prod); learners, stale sessions and mismatches are refused with their codes.
- [ ] The CLI erases a non-last owner or a suspended account, audited without PII.
- [ ] The event is byte-equal to l-01's fixture; the e2e runs on the real producer; the compose rehearsal passed.
- [ ] `host-verify --cluster` green and **the owner's before-launch snapshot id recorded** before the merge; the merge and the tag back to back; v1.12.0 live and verified; status.md updated.

## Ship (land-and-sync — owner approval pre-granted)

> Launching this prompt is the owner's approval for every change it makes (D40); don't stop for review.

1. Branch `feat/l-02-erase-producer`, then conventional commit(s) with the attribution lines, then push, then the PR. This repo only: no `../infra` PR.
2. Once CI is green (fix, then merge, on failure) **and** the compose rehearsal (step 10) and `host-verify --cluster` (step 11) have passed, squash-merge. Never enable auto-merge.
3. **Release action — tag `v1.12.0`, the erase tag** (the next free minor): walk the release checklist (ADR-0034 §6, in the plan; the erase line is `host-verify` green plus the owner's before-launch snapshot), push the tag on the merge commit **right after** the squash-merge, the same day as the snapshot, let Flux deploy, then verify live by looking (step 11) and run the tester acceptance (step 12). Never leave the producer merged but untagged.
4. Update status: the sprint file and `docs/v2/status.md`, in the same PR or a follow-up docs PR merged the same way (the acceptance record needs the follow-up).
5. Run `git checkout main && git pull`. If a clean peer worktree holds `main`, use `git -C <worktree> merge --ff-only origin/main` and then `git switch --detach main`.

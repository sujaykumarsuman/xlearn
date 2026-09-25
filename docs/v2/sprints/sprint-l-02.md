# Sprint l-02 — L-E erase producer + `DELETE /api/me` + UI (AB21) → v1.12.0

> **Milestone:** L — learner gate (**L-E**, producer half; L-E done)   ·   **Track:** product (identity-centred; runs beside M3)
> **Prereqs:** [l-01](sprint-l-01.md) (v1.11.0: all 4 consumers bound) · [ds-l-01](sprint-ds-l-01.md) (merged: AB21 frozen at the merge, D40) · [mi-04](sprint-mi-04.md) (MI-5b live before the first tester) · [m1-04](sprint-m1-04.md) (`identity admin` CLI, session-validate `role`/`created_at`, `admin_audit`)
> **Unblocks:** [l-03](sprint-l-03.md) (L-A backend extends this erase transaction)
> **Release action:** **tag v1.12.0 — an erase tag: `host-verify --cluster` green and the owner's manual snapshot first** (the snapshot is taken before launch, D40; indicative: the next free minor). The xlearn PR merges on CI green once the compose rehearsal and `host-verify` pass, and **the tag follows the merge at once**, so the erase producer never sits on `main` untagged for a peer's tag to ship. Nothing is held for the owner. No infra PR.
> **Calendar:** early November, right after v1.11.0. **Owner event:** `ev-snap-v1.12.0` (the manual Hostinger snapshot, before launch). **`ev-first-tester`** is run by this session (D40): it mints the throwaway acceptance tester with the CLI once MI-5b is verified live (prepared by [m1-04](sprint-m1-04.md)).
> **Execute with:** [`../prompts/prompt-l-02.md`](../prompts/prompt-l-02.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | Erase transaction in identity (+ outbox `account_erasure_requested`); e2e on the real producer | X | ⬜ |
| 2 | `DELETE /api/me`: typed confirmation, session < 5 min, 1 open; testers only; owner → 403 | X | ⬜ |
| 3 | CLI `identity admin account erase <user>` (owner / suspended path; audited) | X | ⬜ |
| 4 | P11 erase part: uniform 404 at once; username held 60 days | X | ⬜ |
| 5 | Settings erase UI (AB21) | X | ⬜ |
| 6 | Runbook (erase + R-d list) · compose rehearsal at the PR head · `host-verify --cluster` · the owner's snapshot `ev-snap-v1.12.0` (taken before launch) | X + H + O (before launch) | ⬜ |
| 7 | Squash-merge + tag v1.12.0 back to back (erase tag; release checklist) | X | ⬜ |
| 8 | Post-tag acceptance on prod: a throwaway tester (`ev-first-tester`, minted here) erases on the web | H | ⬜ |
| 9 | Record: L-E done, AB19–AB21 frozen, floor 1.11.0, snapshot id, gate state | X | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row, the
> L milestone row, the artboard rows). Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] **v1.11.0 live (all 4 consumers bound)** ([l-01](sprint-l-01.md)): healthz ≥ 1.11.0; `XLEARN_IDENTITY` shows `practice-erase`, `review-erase`, `assessment-erase`, `coach-erase`; each service stream shows `identity-erase-acks`; coach on its nkey.
- [ ] **AB21 frozen:** [ds-l-01](sprint-ds-l-01.md) merged (the merge is the freeze, D40). AB19–AB20 froze in the same merge, and ds-l-01's own PR recorded all three; this sprint only confirms them.
- [ ] **m1-04 live (v1.7.0):** `identity admin` CLI and `admin_audit`; session-validate returns `role`, `status` and the session's `created_at`; the owner role is set (`ev-owner-role`).
- [ ] **MI-5b live** ([mi-04](sprint-mi-04.md)) **before the first tester is minted**: no admin console answers on `projects.sujaykumar.dev` (ADR-0033 §11). Task 8 mints that tester itself (`ev-first-tester`, D40); tasks 1–7 don't need it.
- [ ] **The owner's snapshot `ev-snap-v1.12.0` was taken before launch** (attested by the launch; its id or time is in the launch message).
- [ ] **Parallel sessions:** the next free minor is known (`git ls-remote --tags origin`, `gh pr list`, `git worktree list`, ListAgents); no open peer PR edits `internal/identity/`, the gateway route table (`internal/gateway/bff.go`), `docs/architecture/openapi.yaml` or `web/src/screens/Settings.tsx` (serialize if one does).

## Goal

**Turn erase on, for testers, and close L-E.**
- identity's **erase transaction**: delete the account (sessions die with it), open an `erase_request` with the expected acks,
  hold the username for 60 days, and emit `account_erasure_requested` — which l-01's four consumers already handle.
- **`DELETE /api/me`** for role `tester` only (L8: typed confirmation, a session under 5 minutes old, one open request per account);
  the **owner is always refused on the web** (CLI only); learners are refused until the L exit ([l-04](sprint-l-04.md)).
- **`identity admin account erase <user>`**, the only path for the owner or a suspended account.
- **P11's erase part:** the erased profile 404s on the very next request; the username stays locked for 60 days.
- The **Settings erase UI** built against frozen **AB21**.
- Cut **v1.12.0 as an erase tag**: `host-verify --cluster` green and the owner's manual snapshot, taken right before launch (D40)
  ([ADR-0034 §4.3](../../adr/0034-v2-release-labelling-gating-and-rollback.md#43-snapshot-rule)).

## Scope

**In**
- identity: the erase transaction (store + service), `DELETE /me` (session-cookie route, proxied by the gateway), the CLI verb, the
  username-hold check on claim and availability, the additive `/api/me` fields the UI needs.
- gateway: `DELETE /api/me` in the route table (+ OpenAPI), the public-cache epoch bump on success.
- web: the AB21 erase section in `Settings.tsx`, the erased banner on the auth page.
- Runbook: erase, stuck erases, and the R-d "list erases since the snapshot" step.
- Tag v1.12.0 (snapshot first); the tester acceptance run on prod.

**Out (later sprints)**
- Web erase for every non-owner (learners) → [l-04](sprint-l-04.md) (v1.17.0, L exit).
- Clearing `note` on invites the account redeemed → [l-03](sprint-l-03.md), which creates `invite` and extends this transaction.
- judge's erase consumer and its expected ack → [m3-05](sprint-m3-05.md) (born at M3-1; `topology.go` then yields 5 expected acks).
- The acceptance step, `403 acceptance_required` and the privacy page (AB19–AB20) → [l-05](sprint-l-05.md).
- The optional re-seal runbook (purge + republish pseudonymous stream copies) → on request only (ADR-0027 §6 note); documented, not run.

## Tasks

### 1 · Erase transaction [X]

Sources: [t1 §6.6](../research/t1-content-data-model.md#66-erase-path-none-exists-today-no-delete-apime-in-bffgo) step 2,
[ADR-0033 §9](../../adr/0033-invite-only-admission-and-owner-admin.md#9-erase-and-abuse-controls), [ADR-0027 §6](../../adr/0027-content-evalpack-and-user-data-model.md#6-account-erase-v20) as amended by [ADR-0028 §3](../../adr/0028-object-storage-and-backups.md#3-amendments-to-adr-0027).

`internal/identity/erase.go` + `store.EraseAccount(ctx, target, via) (EraseRequest, error)` — **one transaction**:
1. Lock the account row (`SELECT … FOR UPDATE`); not found → `ErrNotFound`.
2. **Guards:** `via=web` refuses role `owner` (`erase_owner_cli_only`) and any role outside the web-erase allow-list — a code constant,
   `{tester}` at L-E; [l-04](sprint-l-04.md) adds `learner` (not an env flag). `via=cli` refuses the **last active owner** (m1-04's guard).
3. `INSERT INTO erase_request (id, account_id, requested_via, expected_acks)` with `expected_acks = events.ErasureAckServices()` from the
   running version's `topology.go` (4 at L-E). The one-open partial unique index turns a concurrent duplicate into `erase_in_progress`.
4. If the account has a username: `INSERT INTO released_username (username_sha256, released_at, available_at) VALUES (sha256(lower(normalized)), now(), now() + 60 days) ON CONFLICT DO UPDATE`.
5. `DELETE FROM identity.outbox WHERE account_id = $1` (identity's own earlier rows for the account, e.g. `account_created`).
6. `admin_audit` hygiene: if any earlier verb stored PII for the target in `detail` (check m1-04's `account create` detail), null those keys.
7. `DELETE FROM identity.account WHERE id = $1` — cascades `oauth_identity`, `session` (every session dies: this is revoke-all on erase),
   `onboarding`, `path_enrollment`. The seat frees itself (seats count active learner accounts).
8. Outbox: `xlearn.identity.account_erasure_requested` `{erase_request_id, account_id, requested_at}` as the v2 envelope l-01 fixed —
   **byte-equal to** `internal/platform/events/testdata/account_erasure_requested.v2.json` — with
   `event_id = uuidv5(ns, "account_erasure_requested:" + erase_request_id)` and **`outbox.account_id = NULL`**.

No cross-pod cache bump is needed for a CLI erase: the gateway **never positively caches account status** ([m1-05](sprint-m1-05.md) task 7),
so every public compose re-resolves the username and gets a 404.

Tests (PG 18): each step; the guards (owner via web, learner via web, last owner via CLI, suspended via CLI allowed); the one-open index;
the outbox row matches the fixture and has `account_id IS NULL`; a second erase of the same account → not found. **Switch
[l-01](sprint-l-01.md)'s `internal/e2e/erase_test.go` from its test helper to the real producer** (drive `DELETE /me` and the CLI's store
call): fan-out → 4 acks → `closed_at` set; `identity admin erasures` shows it closed.

### 2 · `DELETE /api/me` [X]

Sources: [ADR-0033 §7, §9](../../adr/0033-invite-only-admission-and-owner-admin.md#7-roles-and-status-live-in-identitys-database-never-in-the-jwt), [ADR-0035 §4](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#4-limits-inventory) L8, [ADR-0033 §12](../../adr/0033-invite-only-admission-and-owner-admin.md#12-authz-deltas-across-v2) row 12.

identity is the authority for every L8 check, reading the **session itself** — the same way `?link=1` reads it (`currentSession` in
`internal/identity/handlers.go`), so no JWT claim changes:
- **identity** `DELETE /me` (session-cookie route, next to `/auth/*`), body `{"confirm": "<username, or email when there is none>"}`:

  | Check | Response |
  |---|---|
  | no or invalid session | `401 unauthenticated` |
  | role `owner` | `403 erase_owner_cli_only` |
  | role not in the web-erase allow-list (`learner` until l-04) | `403 erase_not_available` |
  | `now() − session.created_at ≥ 5 min` | `403 reauth_required` |
  | `confirm` ≠ username (or email), case-insensitive, trimmed | `400 confirmation_mismatch` |
  | an open request exists | `409 erase_in_progress` |
  | ok | `202 {erase_request_id, username_available_at}` + the session cookie cleared (`clearSessionCookie`) |

- **gateway** `{"DELETE", "/api/me", g.handleDeleteMe, true}` in `apiRoutes()` — a fetch-based proxy like `handleAuthSignup`/`handleAuthLogin`
  (the cookie travels; the `Set-Cookie` comes back); m1-04's JSON + `Sec-Fetch-Site` checks and m1-05's L5 mutating bucket apply; on
  `202`, bump the in-process public-cache epoch (the call the visibility toggles use). Update `docs/architecture/openapi.yaml` (the drift
  test) and [`api.md`](../../architecture/api.md).
- **`GET /api/me` additive fields** (for the UI to render the right AB21 frame without probing): `erase_web: "allowed" | "owner_cli_only" | "not_available"`
  and `session_fresh_until` (session `created_at` + 5 min). OpenAPI updated.
- Tests: every row of the table (identity handler + gateway proxy), the cookie cleared on 202, the epoch bump, `/api/me` fields per role.

### 3 · CLI `identity admin account erase <user>` [X]

Sources: [ADR-0033 §8](../../adr/0033-invite-only-admission-and-owner-admin.md#8-the-owner-admin-cli-identity-admin-run-via-kubectl-exec).
In `internal/identity/admin/` (m1-04's layout): `account erase <user> --confirm <same user>` — `<user>` is an email, username or id; the
`--confirm` value must repeat it (non-interactive `kubectl exec` has no prompt). Runs `EraseAccount(…, via=cli)`: the only path for the
owner (refused only as the last active owner) and for suspended accounts. Prints the erase request id, the expected acks and the hint
`identity admin erasures --open`. Writes `admin_audit` (verb `account erase`, target = account id, detail `{erase_request_id}` — no PII)
plus a log line. Store integration test + output test.

### 4 · P11: erased → uniform 404, username held 60 days [X]

Sources: [rollout §10](../rollout-plan.md#10-public-dashboard-tasks) P11, [ADR-0033 §13](../../adr/0033-invite-only-admission-and-owner-admin.md#13-public-dashboard-authz-deltas).
- **404 at once:** identity's by-username resolver finds no account → the gateway's uniform 404 ("No profile for @name", identical to
  unknown/private/suspended). Test **with a warm composed cache**: web path (`DELETE /api/me`) and CLI path (store call) → the very next
  `GET /api/u/<username>` is 404.
- **Username hold:** `handleSetUsername` and `handleUsernameAvailable` (`internal/identity/usernames.go`) treat a normalized username whose
  sha256 is in `released_username` with `available_at > now()` as **taken** — the same `username_taken` / `available: false` as a live
  account (no "recently erased" oracle). Expired rows are ignored (no pruner in v2.0). Tests: blocked at day 0 and day 59, free at day 61
  (clock injected).

### 5 · Settings erase UI (AB21) [X]

Build `design-system/screens/v2/AB21-erase-account.html` (frozen by [ds-l-01](sprint-ds-l-01.md)) with
[`theme.css`](../../../design-system/theme.css) verbatim, in `web/src/screens/Settings.tsx` at the placement the frozen board chose:
- the **Erase account** card (F1) with the "What's erased" list and the 60-day username line; **Sign in again** (F2) when
  `session_fresh_until` has passed or the server says `reauth_required` — it signs out and returns to Settings after login (no PII in the URL);
- the typed-confirmation modal (F3–F5: **[Cancel]** default focus, the danger button enabled only on a case-insensitive match,
  `confirmation_mismatch`, "Erasing…");
- the owner-refused (F7) and tester-only (F8) states with **no button**; other errors (F9);
- on `202`: navigate to `/xlearn/auth?erased=1`; `Auth.tsx` shows the F6 banner.
- `web/src/lib/settings.ts`: `useEraseAccount()`; types for the new `/api/me` fields.
- Tests: `Settings.test.tsx` (each frame's state by role and freshness; the match gate; mismatch; 202 → redirect), `Auth.test.tsx` (the banner);
  390 px layout per F10.

### 6 · Runbook, `host-verify`, snapshot [X + H + O (before launch)]

Sources: [ADR-0034 §4.2–§4.3](../../adr/0034-v2-release-labelling-gating-and-rollback.md#42-r-d-is-a-procedure-not-a-button), [rollout §2.2](../rollout-plan.md#22-operating-rules-every-mi-step-and-every-tag) (snapshots).
- **[X] Runbook** — an "Erase" section in `docs/runbooks/identity-admin.md` ([m1-04](sprint-m1-04.md)'s runbook):
  the CLI verb and `identity admin erasures --open`; a stuck erase (a missing ack → read that service's `event_dead_letter` and logs, fix,
  let the durable redeliver — a nak'd erase redelivers with backoff for up to ~8 h, but once dead-lettered (terminal, or after the
  last delivery) it is **not** redelivered and no re-emit verb exists yet: record it in status.md and raise it; never close a request by hand); **R-d step 3**: before any restore, list the erases since the snapshot
  (`identity admin erasures --since <snapshot time>`, account ids only), and after the restore re-run each through
  `identity admin account erase <id> --confirm <id>` and record the lost-writes window; the pseudonymous stream copies (incl. historic
  `account_created` with `display_name`) and the on-request re-seal via NATS break-glass.
- **Merge and tag back to back.** The xlearn PR is brought to CI-green and rehearsed before it merges, and the tag follows the merge
  at once: `deploy.yml` builds only on `v*` tags, so a merged-but-untagged producer on `main` would ship with **whichever tag comes
  next** — a parallel M3 sprint's (e.g. [m3-07](sprint-m3-07.md)'s v1.13.0, whose merge-only predecessors run beside L) or a v1 patch —
  with no `host-verify`, no snapshot and possibly out of version order, breaking [ADR-0034 §4.3/§6](../../adr/0034-v2-release-labelling-gating-and-rollback.md#43-snapshot-rule) and the D12 guard.
- **[X + H] Compose rehearsal at the PR head** (task 7's script) — rehearse the exact commit that will be merged.
- **[H] `host-verify --cluster`** green; the host has settled (no host change pending).
- **[O, before launch] `ev-snap-v1.12.0`** — the owner took the manual Hostinger snapshot right before launching (D40); record its
  id/time from the launch message. **Then, at once:** re-check peers, squash-merge the PR (it must still be the rehearsed head; if
  `main` moved, rebase, re-run CI and the rehearsal first), and tag the merge commit (task 7). The snapshot keeps ~1 day, so the tag
  lands the same day as the launch.

### 7 · Tag v1.12.0 [X]

The **compose rehearsal** runs at the PR head before the merge (task 6: `docker compose up`; create a tester through the CLI; seed an
attempt, a mistake, a mock, a coach thread; sign in fresh; erase from the UI; check the four services' row counts are 0, 4 acks, the
profile 404, the username unavailable). After the rehearsal and `host-verify` (the owner's snapshot `ev-snap-v1.12.0` was taken before
launch): squash-merge, then the **release checklist** (below; erase-tag lines apply) and the tag on the merge commit, back to back.
GitHub release title **`v1.12.0 — v2 build · L-E erase`**.
Notes: web erase for testers; owner CLI-only; learners at the L exit; username hold; no backups (D12) — an erase is final.
**Floor after: 1.11.0.** Gate state: **web erase testers only; owner refused (CLI only)**.

### 8 · Post-tag acceptance on prod [H]

**`ev-first-tester` runs here** (D40: launching approves the mint and the erase). After confirming MI-5b is live (status.md), mint a
throwaway tester with the CLI:
`ssh sujaykumar-vps 'k3s kubectl exec -n xlearn deploy/xlearn-identity -- identity admin account create --role tester --email <an unused @example.test address>'`.
Keep its one-time password in the terminal only (never in a file, log, PR or status.md); the account is erased minutes later. Then
operate the tester yourself, in a private window or a separate browser profile you drive:
1. Sign in as the tester; claim a username; create a little data (start an attempt, open a mistake, visit coach).
2. Sign in again (fresh session) and erase from Settings → the erased banner.
3. Checks, through the sanctioned admin CLI and read-only views only (no raw `psql` against other services' schemas):
   `ssh sujaykumar-vps 'k3s kubectl exec -n xlearn deploy/xlearn-identity -- identity admin erasures --since <today>'` → **closed, 4 acks**. That
   is the per-service proof: each service writes its ack in the **same transaction** as its deletes and outbox deletes (l-01 task 2),
   so an ack means that service's rows are gone. Then `/xlearn/u/<username>` → 404; the owner's session shows the username as
   unavailable (in his signed-in profile, if available; task 4's tests cover it otherwise); no ERROR on the erase paths (`k3s kubectl logs -n xlearn deploy/xlearn-<svc> --since=30m` for practice, review,
   assessment, coach and identity, read-only).
4. The **owner-refused** path is verified by tests, and by the owner's Settings showing F7 (no button) if a browser profile signed
   in as the owner is available to the session (an optional before-launch item) — **never send `DELETE /api/me` from the owner's
   account on prod**.

If a tester step needs a human (a tool or rule stops you, e.g. typing the password), don't wait: CLI-erase the tester if it exists
(`identity admin account erase <id> --confirm <id>`), record task 8 ⛔ "tester run needs a human" with these exact steps in
`status.md` → Open owner items, and land everything else. [l-04](sprint-l-04.md)'s entry gate re-checks tester web erase before
the L exit.
Log the `kubectl exec` admin-CLI uses in status.md (the sanctioned manual path, rollout §2.2).

### 9 · Record [X]

status.md: L-E ✅ (v1.11.0 → v1.12.0); tag → floor → snapshot row (v1.12.0 → 1.11.0 → `ev-snap-v1.12.0` id); gate state "web erase:
testers only; owner refused (CLI only)"; confirm the artboard rows AB19, AB20, AB21 read "frozen (PR #, date)" and
[ds-l-01](sprint-ds-l-01.md) reads ✅ (its own merge recorded them under D40; repair either if missing); owner events `ev-snap-v1.12.0`
✅ and `ev-first-tester` ✅ (minted and erased by this session); the erase log (request id, date, via, acks); the manual-path log;
Decisions-log lines. Note [l-03](sprint-l-03.md) is unblocked.

## Acceptance criteria

- [ ] **A tester erases on the web; all acks arrive** (`identity admin erasures` closed with 4); **the profile 404s** on the next request; the username is unavailable for 60 days.
- [ ] **The owner is refused on the web** (`403 erase_owner_cli_only`, tested; no button in the UI); learners get `403 erase_not_available`; a stale session gets `reauth_required`; a mismatch gets `confirmation_mismatch`.
- [ ] The CLI erases an owner (not the last) or a suspended account, audited without PII.
- [ ] The outbox row is byte-equal to l-01's fixture with `account_id` NULL; the e2e runs on the real producer.
- [ ] `host-verify --cluster` green and **the owner's before-launch snapshot id recorded** before the merge; the merge and the tag back to back; v1.12.0 live and verified; status.md updated.

## Release

**Tag `v1.12.0`** (indicative: the next free minor; major = `.release-line` = 1) — **the erase tag**: consumers shipped one tag earlier
(v1.11.0), and this tag turns the producer on ([rollout §7](../rollout-plan.md#7-indicative-tag-timeline), [ADR-0034 §1.6](../../adr/0034-v2-release-labelling-gating-and-rollback.md#16-indicative-tag-timeline)).

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
- **Erase line applies:** `host-verify --cluster` green, host settled, **snapshot taken** (`ev-snap-v1.12.0`, by the owner right before launch, D40); the squash-merge and the tag follow each other at once, the same day.
- ACL line: **no new stream or durable** — identity's publish grant `xlearn.identity.>` already covers `account_erasure_requested`, and
  l-01 bound every consumer. Confirm against the golden file; n/a.
- New service image, contract, M6: n/a (expand-only; no migration expected — l-01 created the tables).
- NetworkPolicy line: no new caller (gateway → identity already exists); n/a.
- Flags: none added (the web-erase allow-list is code). Also smoke the Settings erase card's owner state (F7), in the owner's signed-in profile if one is available.

**Rollback** ([ADR-0034 §4](../../adr/0034-v2-release-labelling-gating-and-rollback.md#44-reversibility-by-step), L-E row):
- There is **no R-a kill switch** for erase; the web path is role-gated to testers.
- **R-b** never below **1.11.0** (consumers must stay bound while erase requests can exist).
- **R-c** (revert + patch tag) is the default for a code fault.
- **Irreversible:** erased data is gone (D12, no backups). An **R-d** restore resurrects every erase after the snapshot — follow the
  runbook: list them first (`identity admin erasures --since`), restore, re-run each through the CLI, record the window.

## Definition of Done

CI green (`go test ./...`, `sqlc diff`, the OpenAPI drift test, web tests, `e2e`) · the compose rehearsal at the PR head · `host-verify --cluster`
green and the owner's before-launch snapshot recorded; the merge + tag back to back · v1.12.0 deployed by Flux and verified by looking · the tester acceptance run on prod (or recorded ⛔ with its steps if it needed a human) · acceptance criteria met ·
statuses updated (this file, [`../status.md`](../status.md); ds-l-01's rows confirmed) · local `main` synced.

## Risks / watch-outs

- **Deleted data is gone** (D12, no backups), and **R-d resurrects later erases** — the runbook lists them first and re-runs them after a
  restore. Only testers can erase on the web in this tag.
- **Erasing the owner by mistake.** The web path refuses role `owner` server-side, the UI renders no button, and the CLI refuses the last
  owner. Never exercise the refusal against the owner's real account on prod — tests cover it.
- **Fresh-session bypass.** The check reads the session's own `created_at` in identity; a JWT or header from the gateway is never trusted
  for it.
- **The typed confirmation as an oracle.** It only confirms the caller's own username/email to the caller; fine.
- **Username-hold oracle.** A held name must look exactly like a taken one on both claim and availability.
- **Stuck erase with no alert (D34).** Read `identity admin erasures --open` after every erase (task 8) and in the runbook; a missing ack
  is a dead-letter row in that service.
- **Pseudonymous copies stay in JetStream**, including historic `account_created` events that carry `display_name` — say so in the
  runbook; the re-seal (ops break-glass purge + republish) runs only on request.
- **In-flight requests racing the erase** (a write already past session-validate when the account is deleted) can recreate a row in a
  service after its consumer ran; the window is milliseconds against seconds of relay latency. Accepted; the tombstone keeps async paths
  clean ([l-01](sprint-l-01.md) task 5).
- **The first tester needs MI-5b** (ADR-0033 §11). If MI-5b slips, tasks 1–7 still land, no tester is minted, and task 8 is recorded
  ⛔ "MI-5b not live" for a later session; nothing waits.

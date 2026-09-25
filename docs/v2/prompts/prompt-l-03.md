# Prompt — Sprint l-03 · L-A admission backend (inert while closed): invites, seats, redeem, public checks, `SIGNUP_MODE=invite`, erase clears invite notes

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-l-03.md`](../sprints/sprint-l-03.md)   ·   **Milestone:** L (L-A backend)   ·   **Prereqs:** [l-02](../sprints/sprint-l-02.md) (`v1.12.0` live), [m1-04](../sprints/sprint-m1-04.md) + [m1-05](../sprints/sprint-m1-05.md) (`v1.7.0` live)

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md).
- The plan: [`../sprints/sprint-l-03.md`](../sprints/sprint-l-03.md). It has the table DDL, the CLI verb table, the transaction order, the error codes and the tests.
- [ADR-0033](../../adr/0033-invite-only-admission-and-owner-admin.md) (Accepted 2026-09-24):
  - **§3** (modes, what counts as a seat, the lock, the R0 seat freeze);
  - **§4** (the L-A schema row);
  - **§5** (mint, the fragment link, `/invites/check`, the email and GitHub redeem transactions, uniform `invite_invalid`, invite before email);
  - **§8** (the CLI, audit, no secrets);
  - **§9** (erase clears `note`; the threat table);
  - §1 (no invite in production outside the L rehearsal).
- [ADR-0035 §4](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#4-limits-inventory) (L2 includes the invite check; L3; L7).
- [ADR-0034](../../adr/0034-v2-release-labelling-gating-and-rollback.md) §2 (`SIGNUP_MODE` is a permanent operating mode) and §3 (expand only; the envelope is append-only).
- [ADR-0005](../../adr/0005-data-ownership-and-migrations.md) (schema-per-service, goose); [ADR-0023](../../adr/0023-email-password-auth-and-account-linking.md) (email auth; §3 no auto-link into password accounts, amended).
- [t7 §2.3](../research/t7-cross-cutting-and-rollout.md#23-admission-design-identity-owns-it); [rollout §4](../rollout-plan.md#4-per-milestone-detail) (L-A row).
- The plans of [m1-04](../sprints/sprint-m1-04.md) (the admin CLI, `seats`, the seat lock, L3, L7), [m1-05](../sprints/sprint-m1-05.md) (the `internal/gateway/limit` package, L2) and [l-02](../sprints/sprint-l-02.md) (the erase transaction).
- Code:
  - `internal/identity/{authemail.go,handlers.go,session.go,oauth.go,config.go,service.go}`, `internal/identity/admin/`;
  - `internal/identity/store/{store.go,queries/*.sql,migrations/}`, `cmd/identity/main.go`;
  - `internal/gateway/{bff.go,gateway.go,openapi_drift_test.go}`, `internal/gateway/limit/`;
  - `docker-compose.yml`, `docs/architecture/{api.md,openapi.yaml,data-model.md,events.md}`, `docs/runbooks/identity-admin.md`.

## Context

D33 closed signup in v1.5.2, and v2 admits people only through owner-minted invites. D35 keeps v2 owner-only, but the whole learner gate is still built and rehearsed. The groundwork is in place:
- M1a (`v1.6.0`) added `account.role`, `status`, `admitted_via`, `invite_id`, `accepted_at`, `region` and `admin_audit`.
- M1b (`v1.7.0`) added the audited `identity admin` CLI, `seats`, the L7 guard (which parses `invite` but runs it as `closed`), L3's bcrypt gate and the gateway limiter.
- L-E (`v1.11.0` → `v1.12.0`) added erase.

This sprint builds the **backend half of L-A**:
- the `invite` and `account_consent` tables;
- the invite CLI and seat accounting;
- redemption on both create paths;
- `POST /api/invites/check` and `GET /api/auth/config`;
- `SIGNUP_MODE=invite`;
- erase clearing invite notes.

Production stays `closed` with no invite, so everything here is **inert**. It merges during M3 and rides the next tag dark. The owner-visible front door (the auth page state, the acceptance step, the notice) is [l-05](../sprints/sprint-l-05.md), labelled with [l-04](../sprints/sprint-l-04.md)'s `v1.17.0`.

## Entry gates — verify first (stop and report if any is unmet)

- [ ] `v1.12.0` live (`curl -s https://projects.sujaykumar.dev/xlearn/api/v1/healthz`); identity's erase transaction is on `main` (l-02).
- [ ] `v1.7.0`'s pieces on `main`: `internal/identity/admin/`, `resolveSignupMode`, the `identity.seats` lock in `reactivate`/`set-role`, `internal/gateway/limit` with L2.
- [ ] Production `SIGNUP_MODE=closed`. Check read-only in `../infra/apps/xlearn-identity.yaml` and with `ssh vps 'k3s kubectl -n xlearn get deploy xlearn-identity -o jsonpath="{.spec.template.spec.containers[0].env}"'`.
- [ ] No open peer PR touches `internal/identity/` or the gateway `apiRoutes` table (`gh pr list`, `git worktree list`, ListAgents). If an M3 gateway PR is open, rebase after it merges.

## Do this (in order)

1. **[X] Branch** `feat/la-admission-backend` from an up-to-date `main`.
2. **[X] Schema** (plan task 1):
   - Add the next free identity migration with `invite` and `account_consent`: indexes, CHECKs, no CHECK on `tier`/`kind`, `redeemed_account_id … ON DELETE SET NULL`, `account_consent.account_id … ON DELETE CASCADE`, and the partial unique `(account_id, kind) WHERE withdrawn_at IS NULL`.
   - Add the `account.invite_id` FK (`NOT VALID` + `VALIDATE`).
   - Write `queries/invite.sql` and `queries/consent.sql`, then `sqlc generate`. Add the Go registries `invite.go` (tiers `{standard}`) and `consent.go` (empty; l-05 and m4-05 register kinds).
3. **[X] Seats + CLI** (plan task 2):
   - `SeatsUsed` and `lockSeats` (`hashtext('identity.seats')`), used by mint, redeem, `reactivate` and `set-role … learner`.
   - `SEAT_CAP` parse (default 15; > 40 → 40 plus an ERROR log).
   - `identity admin invite create|list|revoke`: a 16-byte `crypto/rand` code, 22-char base64url, sha256 stored, the link printed once from `PUBLIC_BASE_URL`, TTL 1h–30d (default 7d), `--note` required and ≤ 120. `seats` shows invites.
   - `admin_audit` on every verb with no code, note or email in `detail`. Store integration tests, including the cap and the R0 freeze.
4. **[X] Email redeem** (plan task 3):
   - `handleSignup` by mode. For `invite`: `invite_required`, a malformed code → `invite_invalid`, the cheap pre-check, format validation, hash **before** the transaction, then `CreateInvitedEmailAccount` (lock → conditional `UPDATE … RETURNING` → seat check → email insert → account fields + `redeemed_account_id` + onboarding → outbox `account_created{tier}`).
   - Map errors to 403 `invite_invalid` / 403 `no_seats` / 409 `email_taken`, with one byte-identical `invite_invalid` body.
5. **[X] GitHub redeem** (plan task 4):
   - `oauthTx.Invite` (`json:"i"`) from the start form outside link mode.
   - The callback: an existing account signs in untouched; a new account in `invite` mode needs the invite (`/auth?error=invite_required`) and redeems in the create transaction. `intended_email` is matched against **any** of the GitHub account's **verified** emails (`/user/emails`, fetched before the transaction). That reads ADR-0033 §5's "the GitHub-verified email" as "a verified email": log it as a deliberate widening in the decisions log. Errors redirect to `?error=invite_invalid|no_seats`.
   - Handler and race tests.
6. **[X] Public endpoints** (plan task 5):
   - identity `POST /invites/check` (`{valid, expires_at}`, always `false` in `closed`) and `GET /auth/config` (the effective mode).
   - Gateway `apiRoutes` rows `POST /api/invites/check` (m1-05's L2 limiter) and `GET /api/auth/config` (a 30 s in-process cache). Tests.
7. **[X] `SIGNUP_MODE=invite`** (plan task 6):
   - `resolveSignupMode` returns `invite`; update the table test.
   - The **`closed` golden test** (signup with and without `invite`, and a new-user GitHub callback, are byte-identical to `v1.12.0`).
   - `account create --role tester` works in every mode; add a commented `SIGNUP_MODE: invite` hint in `docker-compose.yml`.
8. **[X] Erase** (plan task 7): in l-02's erase transaction, before the account delete, run `UPDATE identity.invite SET note = NULL, intended_email = NULL WHERE redeemed_account_id = $1`. Test: redeem → erase → the invite is `erased`, note and email NULL, seats drop by one, no consent rows.
9. **[X] Docs** (plan task 8): the `docs/runbooks/identity-admin.md` invite section (incl. "no invite in production outside the L rehearsal"); `api.md` + `openapi.yaml` (routes, the `invite` field, error codes, `?error=` values); `data-model.md`; `events.md` (`tier`).
10. **[X] Verify:**
    - `gofmt -l`, `go vet ./...`, `go test -race ./...`, `go test -tags e2e ./internal/e2e/...` (PG 18), `sqlc diff`, the migration lint, the OpenAPI drift test, and web typecheck/test (no web change expected).
    - With `docker compose up --build` and identity overridden to `SIGNUP_MODE: invite` (no `DEV_AUTH` needed for this check), mint with `docker compose exec identity identity admin invite create --note test`. Redeem via `curl` on `/xlearn/api/v1/auth/signup`, reuse it (→ `invite_invalid`), check `/xlearn/api/v1/invites/check` and `/xlearn/api/v1/auth/config`, then erase the account through the CLI and confirm the invite shows `erased`.
11. **[X] PR** with conventional commits (`feat(identity): …`, `feat(gateway): …`, `docs: …`) and the attribution lines → CI green → squash-merge (see Ship). **No tag.**

## Constraints

- **Service boundaries ([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)):** everything lives in schema `identity`. The gateway only routes, limits and caches the config answer. judge reads `tier` and consents later over identity's internal HTTP (m4-02), never cross-schema.
- **goose + sqlc:** expand only (nullable columns, new tables, an FK added `NOT VALID` then validated). No `-- xlearn:contract`. Take the next free version at rebase. **`sqlc diff` clean.**
- **Outbox:** `account_created` is written in the same transaction as the account. `tier` is an additive field on an existing subject: **no new subject, stream or consumer, so no NATS ACL PR** and no consumers-before-producers ordering applies.
- **No secrets in logs, audit or events:** invite codes appear once on the owner's terminal. Store only sha256, and never log the code (log `invite_id` after redemption). No note or email in `admin_audit`.
- **No oracle:** the invite is checked before the email; `invite_invalid` is uniform; `closed` still refuses before reading the body.
- **The seat lock is never held across bcrypt or HTTP.**
- **GitOps / production:** no infra PR, never `kubectl apply`. **Do not mint an invite or change `SIGNUP_MODE` on production.** That is only for [l-04](../sprints/sprint-l-04.md)'s rehearsal. The CLI via `kubectl exec` is the sanctioned manual path (rollout §2.2), and read-only checks only here.
- **D34:** the `SEAT_CAP` > 40 guard and L7 are ERROR log lines, not alerts.
- **Memory-sum rule:** no new pod or container, so it is unaffected. The gateway's config cache is a few bytes.
- **Parallel sessions:** check peers' PRs, worktrees and ListAgents before editing identity or the route table, and check peers' ADR numbers before claiming one. No ADR is expected: this implements ADR-0033.

## Deliverables

- An identity migration (`invite`, `account_consent`, the `invite_id` FK), sqlc queries, and the tier and consent-kind registries.
- `SeatsUsed`/`lockSeats` shared by mint, redeem, `reactivate` and `set-role`; `SEAT_CAP` with the > 40 guard.
- `identity admin invite create|list|revoke` and the extended `seats`, all audited.
- Invite redemption on `POST /auth/signup` and the GitHub callback; `account_created{tier}`.
- identity `POST /invites/check` and `GET /auth/config`; gateway `POST /api/invites/check` (L2) and `GET /api/auth/config`.
- `SIGNUP_MODE=invite`; the `closed` golden test.
- Erase clears the invite `note` and `intended_email`.
- Docs: the admin runbook, `api.md`, `openapi.yaml`, `data-model.md`, `events.md`.

## Update status

- The plan's Status table ([`../sprints/sprint-l-03.md`](../sprints/sprint-l-03.md)): tasks 🔄 → ✅; _Overall_ ✅.
- [`../status.md`](../status.md):
  - the Sprint board row: l-03 ✅ "merged, rides <next tag> dark";
  - the L milestone: "L-A backend merged (inert)";
  - the flag inventory: `SIGNUP_MODE` (operating mode) accepts `closed|invite|open(dev)`; `SEAT_CAP` defaults to 15 in code with the > 40 guard;
  - limits: L2 (invite check) and L7 (invite + `SEAT_CAP`) built.
  - Add a **pending-smoke note** for the carrying tag (its plan, indicatively m3-13, doesn't list these checks, so this note is the only handoff; the tag sprint's agent must read it before tagging): check `auth/config`=`closed` (read-only), the auth page unchanged, and `identity admin seats` → invites 0. `seats` is a `kubectl exec` that writes `admin_audit` (m1-04 audits reads), so it is a sanctioned manual path (rollout §2.2): the tag sprint's session runs it itself (approved by that prompt's launch, D40), and it goes in the CLI-use log.
- Decisions log:
  - the lock key;
  - no CHECK on `kind`/`tier`;
  - the `no_seats` code;
  - `intended_email` cleared on erase;
  - `intended_email` matched against any GitHub-verified email, not only the primary (a deliberate widening of ADR-0033 §5 / t7 §2.3's wording; every `verified` address proves control, and primary-only would refuse testers for no security gain);
  - the audit excludes note and email;
  - the `SEAT_CAP` > 40 guard;
  - `/invites/check` answering `false` in `closed`.

## Done when (acceptance)

- [ ] Seat races are impossible under concurrent mint, redeem and reactivate; a code redeems at most once (race tests).
- [ ] Uniform `invite_invalid` for used, expired, revoked, unknown and malformed codes on every path.
- [ ] The invite is checked before the email; a taken email doesn't consume the invite.
- [ ] GitHub path: the new account redeems in its create transaction; an `intended_email` mismatch is refused; an existing account is untouched; no invite → `invite_required`.
- [ ] The CLI prints the code once, stores only sha256, and audits every verb without secrets.
- [ ] `closed` is byte-for-byte unchanged (golden); `open` without `DEV_AUTH` still runs `closed`.
- [ ] Erase nulls the redeemed invite's `note` (and `intended_email`) and its consent rows.
- [ ] `/api/invites/check` is limited (L2); `/api/auth/config` reports the effective mode.
- [ ] CI green (`sqlc diff`, the migration lint, OpenAPI drift); merged to `main`.

## Ship (land-and-sync — owner approval pre-granted)

> Launching this prompt is the owner's approval for every change it makes (D40); don't stop for review.

1. Branch `feat/la-admission-backend`, then conventional commit(s) with the attribution lines, then push, then the PR. This repo only: no `../infra` PR.
2. Once CI is green (fix, then merge, on failure), squash-merge. Never enable auto-merge.
3. **Release action — merge only:** nothing deploys (`main` is build-only). It ships dark and inert (production stays `SIGNUP_MODE=closed` with no invite) in **the next tag after the merge**: indicatively `v1.14.0` ([m3-13](../sprints/sprint-m3-13.md)), else `v1.15.0` or `v1.16.0`. The pending-smoke note tells that tag's session what to check. No tag here.
4. Update status: the sprint file and `docs/v2/status.md`, in the same PR or a follow-up docs PR merged the same way.
5. Run `git checkout main && git pull`. If a clean peer worktree holds `main`, use `git -C <worktree> merge --ff-only origin/main` and then `git switch --detach main`.

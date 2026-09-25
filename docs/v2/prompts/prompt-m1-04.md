# Prompt — Sprint m1-04 · Identity security floor: roles, sessions, admin CLI, DEV_AUTH guard, L3, CSP (M1b)

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-m1-04.md`](../sprints/sprint-m1-04.md)   ·   **Milestone:** M1 (M1b)   ·   **Prereqs:** [m1-03](../sprints/sprint-m1-03.md) (merged), [m1-02](../sprints/sprint-m1-02.md) (`v1.6.0` live)

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md).
- The plan: [`../sprints/sprint-m1-04.md`](../sprints/sprint-m1-04.md) (verb table, CSP string, status codes).
- [ADR-0033](../../adr/0033-invite-only-admission-and-owner-admin.md) — **§3 (modes, the `DEV_AUTH` guard, seats), §6 (validate fields), §7 (roles/status in the DB, never the JWT), §8 (the admin CLI), §9 (abuse controls), §12 rows 2/4/5/6/8/15**, §11 (MI-5b gate for the first tester).
- [ADR-0035](../../adr/0035-v2-operations-nats-auth-limits-capacity.md) §4 (L3, L7; L1/L2 are m1-05's).
- [ADR-0034](../../adr/0034-v2-release-labelling-gating-and-rollback.md) §2 (T-3 cohort; `preview` hidden outside it).
- [ADR-0006](../../adr/0006-authn-authz.md) (as amended by 0033 §14), [ADR-0022](../../adr/0022-path-enrollment-and-dev-login.md) (`DEV_AUTH`), [ADR-0023](../../adr/0023-email-password-auth-and-account-linking.md) (email auth, password change).
- [t7 §2 (authz deltas, admission, owner role)](../research/t7-cross-cutting-and-rollout.md); [`../rollout-plan.md`](../rollout-plan.md) §4 M1 (scope T7), §2.2 (sanctioned manual paths).
- Code: `internal/identity/{config.go,session.go,handlers.go,authemail.go,password.go,service.go,devauth.go}`,
  `internal/identity/store/{store.go,queries/session.sql}`, `cmd/identity/main.go` (the `-version` precedent),
  `deploy/identity.Dockerfile` (distroless, `/usr/local/bin/identity`), `internal/platform/auth/{auth.go,jwt.go}`,
  every service's `requireJWT` (`internal/{identity,practice,review,assessment,coach}/handlers.go`),
  `internal/gateway/{gateway.go,bff.go,course.go}` (`course.go` from m1-03), `web/src/lib/api.ts` (`apiFetch`),
  `web/src/lib/settings.ts` (`streamCoachChat`, the coach chat's raw SSE `fetch`),
  `web/src/screens/{Auth,Settings}.tsx`, `web/index.html`, `docker-compose.yml`.

## Context

M1b's security floor. `v1.5.2` closed signup (`SIGNUP_MODE=closed` via infra#30) but honours `open` without `DEV_AUTH`;
`v1.6.0` (m1-02) added `account.role/status/admitted_via/accepted_at` and `admin_audit` with no reader. Today sessions
never check account status, nothing reads JWT roles, login skips bcrypt for unknown identifiers (a timing oracle),
bcrypt runs unbounded under a 250m limit, and the shared origin has no CSP. This sprint makes roles and status real
(in identity's DB, never the JWT), adds the audited owner CLI used via `kubectl exec`, the L7 and L3 guards, a strict CSP
with cross-site write checks, and cohort visibility for `preview` courses. It merges only; `v1.7.0` is cut by m1-07.
Right after that tag, m1-07's session sets the owner's role once (`ev-owner-role`, D40). The gateway router is serialized:
m1-03 is merged, m1-05 waits on this PR.

## Entry gates — verify first (stop and report if any is unmet)

- [ ] `v1.6.0` live (healthz; the identity columns and `admin_audit` exist — `\d identity.account` in compose at `v1.6.0`)
- [ ] [m1-03](../sprints/sprint-m1-03.md) merged (`internal/gateway/course.go` with `courseVisible` on `main`)
- [ ] No open peer PR edits `internal/identity/` or the gateway router (`gh pr list`, `git worktree list`, ListAgents)

## Do this (in order)

1. **[X] Branch** `feat/m1b-identity-floor` from an up-to-date `main`.
2. **[X] Sessions** (plan task 1): `GetValidSession` joins `status='active'` and returns role/status/accepted_at/session
   `created_at`; `RevokeAllSessions`; `/sessions/validate` adds `role`, `status`, `accepted`, `created_at`; revoke-all on
   password change (`{ok:true, reauth:true}` → SPA to `/auth` with a notice); suspended accounts can't start a session
   (uniform 401 on email login; `/auth?error=account_unavailable` from OAuth). Gateway: `sessionInfo` + `authSession`; never
   cache status or role. `sqlc generate`.
3. **[X] RequireRole** (plan task 2): `internal/platform/auth/require.go` (`RoleLearner`, `RolePublicRead`, `RequireRole`,
   `ClaimsFrom`); swap every service's `requireJWT` to it; `internal/gateway/cohort.go` `inCohort`; route-walk tests
   (`public-read` → 403 everywhere); assert the gateway mints only `["learner"]`.
4. **[X] L7** (plan task 3): `resolveSignupMode(raw, devAuth)`; ERROR log for `open` without `DEV_AUTH`; `invite` → closed
   until L-A; full table test. Compose (`open` + `DEV_AUTH=true`) must still allow sign-up.
5. **[X] L3** (plan task 4): 2-slot bcrypt gate (non-blocking → 429 `too_many_requests`, `Retry-After: 1`) around hash and
   compare; startup dummy hash used for unknown / password-less / suspended identifiers; counting-hasher and concurrency tests.
6. **[X] Admin CLI** (plan task 5): `identity admin` dispatch in `cmd/identity/main.go`; `internal/identity/admin/` +
   `queries/admin.sql`; verbs `account list|suspend|reactivate|revoke-sessions|set-role|create --role tester` and `seats`;
   `<user>` = email|id|username; last-owner guard (`identity.owners` advisory lock); seat lock + `SEAT_CAP` (default 15) on
   `reactivate`/`set-role … learner`; `admin_audit` + a stderr log line per verb, same tx; no secret anywhere; one-time
   tester password on stdout; no migrations run. Integration tests per verb.
7. **[X] CSP + cross-site checks** (plan task 6): `internal/gateway/security.go` in `Handler()` — the CSP string from the plan
   on SPA shell responses (`form-action` includes `https://github.com`), `nosniff` everywhere, `Referrer-Policy: same-origin`;
   mutating `/api/*` → 403 `cross_site_request` / 415 `unsupported_media_type`, with the OAuth start form POST exempt from the
   JSON check only. SPA: `apiFetch` sends `Content-Type: application/json` on every non-GET; `streamCoachChat`
   (`web/src/lib/settings.ts`) already does — keep it; one vitest covers both.
8. **[X] Cohort `preview`** (plan task 7): identity enrollment allows `preview` for DB role owner/tester; the gateway passes
   `inCohort(info)` to `courseVisible`; tests with m1-03's `preview` fixture.
9. **[X] Runbook** (plan task 8): `docs/runbooks/identity-admin.md` — every verb via
   `ssh sujaykumar-vps 'k3s kubectl exec -n xlearn deploy/xlearn-identity -- identity admin …'`, `ev-owner-role`, `ev-first-tester`
   (only after MI-5b is live), stranger triage, "log each use in `docs/v2/status.md`".
10. **[X] Verify:** `gofmt`, `go vet`, `go test -race ./...`, `sqlc diff`, web typecheck/lint/test/build, `-tags e2e`;
    `docker compose up --build`, then click through **every** screen (sign-in, onboarding, Catalog, Roadmap, Today, Week,
    Concept, Problem, Revision, Mistakes, Mock, Progress, Settings incl. password change and Connect GitHub, coach chat, the
    public `/u/` page) with the browser console open — zero CSP violations; exercise every CLI verb against compose
    (`docker compose exec identity identity admin …`).
11. **[X] PR** → conventional commits (`feat(identity): …`, `feat(gateway): …`) with the attribution lines → CI green →
    squash-merge (see Ship). **No tag.**

## Constraints

- **Roles and status live in identity's DB, never in a JWT** (ADR-0033 §7): no `owner`/`tester` claim, ever; the gateway reads
  role from session-validate; identity reads its own row.
- **Service boundaries ([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)):** the CLI runs inside the identity image
  against identity's schema only, with the pod's own credentials; no new secret, no web admin surface.
- **goose + sqlc:** no schema change is expected (M1a added the columns); if one is needed, it is expand-only at the next free
  version and passes the migration lint; **`sqlc diff`** clean.
- **No secrets logged or stored** (passwords, session ids, codes); uniform error bodies for auth failures (no oracle).
- **Frontend:** only the `api.ts` header change (plus the `settings.ts` vitest) and the password-changed notice; `theme.css` verbatim.
- **GitOps:** no infra change in this sprint; never `kubectl apply`; the CLI via `kubectl exec` is the sanctioned manual path
  (rollout §2.2). **D34:** the L7 misconfiguration is an ERROR log line, not an alert.
- **Parallel sessions:** check peers' PRs/worktrees before editing identity or the gateway router; check peers' ADR numbers
  before claiming one.

## Deliverables

- Session join + validate fields + revoke-all (identity) and `sessionInfo`/`authSession` (gateway).
- `internal/platform/auth/require.go`; every service on `RequireRole`; `internal/gateway/cohort.go`.
- L7 (`config.go`) and L3 (`password.go`, `authemail.go`) with tests.
- `identity admin` CLI (`cmd/identity`, `internal/identity/admin/`, `queries/admin.sql`) with integration tests.
- `internal/gateway/security.go` (CSP + cross-site checks) + SPA header changes + tests.
- Cohort `preview` enrollment and visibility.
- `docs/runbooks/identity-admin.md`.

## Update status

- This plan's Status table ([`../sprints/sprint-m1-04.md`](../sprints/sprint-m1-04.md)): tasks 🔄 → ✅; _Overall_ ✅.
- [`../status.md`](../status.md): Sprint board row (m1-04 ✅, "merged, ships in v1.7.0"); owner events `ev-owner-role` (run by
  m1-07's session after `v1.7.0`) and `ev-first-tester` (run by l-02's session after MI-5b) → "prepared: runbook"; flag inventory: `SIGNUP_MODE` operating mode now guarded
  by `DEV_AUTH` (L7), `SEAT_CAP` default 15 (code); limits L3 and L7 → built (ships in `v1.7.0`).
- Decisions log: the CSP string and its `form-action` GitHub entry, revoke-all including the caller on password change, reads
  audited too, the OAuth `account_unavailable` redirect. ADR only for a call beyond ADR-0033/0035 (check peers' numbers first).

## Done when (acceptance)

- [ ] Suspend kills every session at once; revoke-sessions and password change revoke all; last-owner guard refuses.
- [ ] `open` without `DEV_AUTH` runs closed and logs ERROR; `invite` runs closed; compose sign-up unchanged.
- [ ] Login timing uniform for unknown identifiers; a 3rd concurrent bcrypt → 429.
- [ ] CLI verbs audited, no secret logged, tester password printed once.
- [ ] `public-read` → 403 on every user route; JWTs remain `["learner"]`.
- [ ] CSP present; cross-site writes 403, non-JSON writes 415; no CSP violation across the compose smoke.
- [ ] `preview` visible/enrollable for the cohort only.
- [ ] CI green (`sqlc diff`); merged to `main`.

## Ship (land-and-sync — owner approval pre-granted)

> Launching this prompt is the owner's approval for every change it makes (D40); don't stop for review.

1. Branch `feat/m1b-identity-floor`, then conventional commit(s) with the attribution lines, then push, then the PR. This repo only: no `../infra` PR.
2. Once CI is green (fix, then merge, on failure), squash-merge. Never enable auto-merge.
3. **Release action — merge only:** nothing deploys (`main` is build-only). It ships in **`v1.7.0`**, which [m1-07](../sprints/sprint-m1-07.md) tags; m1-07's session then runs `ev-owner-role`. No tag here.
4. Update status: the sprint file and `docs/v2/status.md`, in the same PR or a follow-up docs PR merged the same way.
5. Run `git checkout main && git pull`. If a clean peer worktree holds `main`, use `git -C <worktree> merge --ff-only origin/main` and then `git switch --detach main`.

# Sprint l-03 — L-A admission backend (inert while closed): invites, seats, redeem, public checks, `SIGNUP_MODE=invite`, erase clears invite notes

> **Milestone:** L — learner gate (**L-A**, backend half) · **Track:** product · **Order:** 47
> **Prereqs:** [l-02](sprint-l-02.md) (`v1.12.0` live: identity's erase transaction) · [m1-04](sprint-m1-04.md) (`v1.7.0`: `identity admin`, `seats`, the L7 `resolveSignupMode`, the seat lock on `reactivate`) · [m1-05](sprint-m1-05.md) (`v1.7.0`: the gateway limiter, L2)
> **Unblocks:** [m4-02](sprint-m4-02.md) (`invite.tier`, `account_consent`) · [m4-05](sprint-m4-05.md) (`account_consent`) · [l-05](sprint-l-05.md) (the owner-visible front door)
> **Release action:** **merge only. It ships dark** in the next tag after it merges: `v1.14.0` if it lands before [m3-13](sprint-m3-13.md) tags, otherwise `v1.15.0` or `v1.16.0`. Production stays `SIGNUP_MODE=closed` with no invite, so every path added here is inert. L-A is **labelled** at `v1.17.0` ([l-04](sprint-l-04.md), [ADR-0034 §1.6](../../adr/0034-v2-release-labelling-gating-and-rollback.md#16-indicative-tag-timeline)).
> **Calendar:** November, in parallel with M3 (after `v1.12.0`). No owner time.
> **Execute with:** [`../prompts/prompt-l-03.md`](../prompts/prompt-l-03.md). One prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | Schema: `invite`, `account_consent`, the `account.invite_id` FK | X | ⬜ |
| 2 | Seat accounting, `identity admin invite create\|list\|revoke`, `seats` with invites, `SEAT_CAP` | X | ⬜ |
| 3 | Redeem on the email path (one transaction, invite before email) | X | ⬜ |
| 4 | Redeem on the GitHub path (`oauthTx` carries the invite; `intended_email`) | X | ⬜ |
| 5 | Public endpoints: `POST /api/invites/check` (L2), `GET /api/auth/config` | X | ⬜ |
| 6 | `SIGNUP_MODE=invite` honoured; `closed` unchanged (golden) | X | ⬜ |
| 7 | Erase clears the redeemed invite's `note` (and `intended_email`); consents cascade | X | ⬜ |
| 8 | Docs: admin runbook, API + OpenAPI, data model, events | X | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row, plus the L milestone).
> Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] **`v1.12.0` live** ([l-02](sprint-l-02.md)). identity's erase transaction exists, and so do `erase_request` and `released_username` from `v1.11.0` ([l-01](sprint-l-01.md)).
- [ ] **`v1.7.0` live** ([m1-04](sprint-m1-04.md), [m1-05](sprint-m1-05.md)). This means `identity admin` (the `account` verbs, `seats`, `admin_audit`), `resolveSignupMode(raw, devAuth)` (L7, `invite` parsed but run as `closed`), the `identity.seats` advisory lock on `reactivate` and `set-role … learner`, and the gateway limiter package `internal/gateway/limit` with L2.
- [ ] **Production is `SIGNUP_MODE=closed`.** Check it read-only in `../infra/apps/xlearn-identity.yaml` and with `ssh vps 'k3s kubectl -n xlearn get deploy xlearn-identity -o jsonpath="{.spec.template.spec.containers[0].env}"'`. The "inert" release claim depends on it.
- [ ] **No open peer PR edits `internal/identity/` or the gateway route table** (`internal/gateway/bff.go` `apiRoutes`). M3's gateway sprints ([m3-09](sprint-m3-09.md)) may be in flight: rebase onto them, don't collide. Check with `gh pr list`, `git worktree list` and ListAgents.

## Goal

Build the invite-only admission backend that D33 designed
([ADR-0033 §3–§5](../../adr/0033-invite-only-admission-and-owner-admin.md#5-invites-mint-and-redeem)):
- owner-minted, single-use, 128-bit invites under `SEAT_CAP`, minted with a CLI;
- redemption inside the account-create transaction on **both** create paths, with the invite checked **before** the email and one uniform `invite_invalid`;
- the two small public endpoints the auth page needs;
- `SIGNUP_MODE=invite`;
- the `account_consent` table that M4 and the acceptance step write.

All of it is **inert while production is `closed`**: no invite exists and every create path still answers `signup_closed`. So it can merge during M3 without changing anything the owner sees. The auth-page UI, the acceptance step and the privacy notice are [l-05](sprint-l-05.md)'s, and they ride `v1.17.0`.

## Scope

**In**
- identity migrations: `invite`, `account_consent`, the `account.invite_id` FK. Expand only.
- Seat accounting (`active` learners plus outstanding invites) under `pg_advisory_xact_lock(hashtext('identity.seats'))` on mint, redeem, `reactivate` and `set-role … learner`. `SEAT_CAP` comes from env (default 15; > 40 runs as 40 with an ERROR log, since R2 is required, ADR-0033 §1).
- `identity admin invite create|list|revoke`. `seats` gains the invite count. Every verb is audited, and no code, note or email goes into the audit.
- Redeem on `POST /auth/signup` and on the GitHub callback. The error codes are `invite_required`, `invite_invalid` and `no_seats`. `account_created` gains `tier`.
- identity `POST /invites/check` and `GET /auth/config`, and the gateway routes `POST /api/invites/check` (L2: 5/min per IP) and `GET /api/auth/config` (a 30 s in-process cache).
- `SIGNUP_MODE=invite` resolves to `invite` (m1-04 ran it as `closed`).
- The erase transaction nulls `note` and `intended_email` on invites the account redeemed. `account_consent` rows cascade.

**Out**
- The auth page's invite-only state, the `#invite=` fragment handling, the acceptance step, `403 acceptance_required`, the privacy notice and the L-C consents at acceptance → [l-05](sprint-l-05.md) (rides `v1.17.0`).
- The AI consent kinds, `PATCH /api/me/consents` and `GET /api/me/ai-allowance` → [m4-05](sprint-m4-05.md). judge reading `tier` and consents → [m4-02](sprint-m4-02.md).
- Opening web erase to learners, tagging `v1.17.0`, and the production invite rehearsal → [l-04](sprint-l-04.md).
- **Any invite in production.** None is minted outside the L rehearsal (`ev-l-rehearsal`, [l-04](sprint-l-04.md)). `SIGNUP_MODE=invite` in production happens only during that rehearsal, and otherwise at the v3 opening ([rollout §11](../rollout-plan.md#11-opening-gates-v3)).
- Any infra change. `SIGNUP_MODE: closed` stays, and `SEAT_CAP` uses the code default until l-04's rehearsal PR sets it explicitly.

## Tasks

### 1 · Schema [X]

Sources: [ADR-0033 §4](../../adr/0033-invite-only-admission-and-owner-admin.md#4-schema-identity-the-adr-0005-expand-rules), §5, §9; [ADR-0034 §3](../../adr/0034-v2-release-labelling-gating-and-rollback.md#3-migration-and-event-compatibility-rules) (expand rules); [t7 §2.3](../research/t7-cross-cutting-and-rollout.md#23-admission-design-identity-owns-it).
Add `internal/identity/store/migrations/000NN_la_invite_consent.sql`, taking the next free version at rebase (CI fails on a duplicate). It is expand only, with no `-- xlearn:contract` marker:
- `identity.invite`:
  - `id uuid PK DEFAULT gen_random_uuid()`, `code_sha256 bytea NOT NULL UNIQUE CHECK (octet_length(code_sha256) = 32)`;
  - `created_at timestamptz NOT NULL DEFAULT now()`, `expires_at timestamptz NOT NULL`, with `CHECK (expires_at > created_at AND expires_at <= created_at + interval '30 days')`;
  - `note text NULL CHECK (char_length(note) <= 120)`, `intended_email text NULL`;
  - `tier text NOT NULL DEFAULT 'standard'`, `region text NULL CHECK (region ~ '^[A-Z]{2}$')`;
  - `redeemed_at timestamptz NULL`, `redeemed_account_id uuid NULL REFERENCES identity.account(id) ON DELETE SET NULL`, `revoked_at timestamptz NULL`.
  - Indexes: a partial index on `(expires_at) WHERE redeemed_at IS NULL AND revoked_at IS NULL` (the seat count), and `(redeemed_account_id)` (erase).
  - `tier` has **no CHECK**. A Go registry (`internal/identity/invite.go`, today `{"standard"}`) validates it, so M4 can add a tier without a migration.
- `identity.account_consent`:
  - `id uuid PK DEFAULT gen_random_uuid()`, `account_id uuid NOT NULL REFERENCES identity.account(id) ON DELETE CASCADE`;
  - `kind text NOT NULL`, `version text NOT NULL`, `granted_at timestamptz NOT NULL DEFAULT now()`, `withdrawn_at timestamptz NULL`;
  - `CREATE UNIQUE INDEX … ON identity.account_consent (account_id, kind) WHERE withdrawn_at IS NULL`, so an account has one live grant per kind and the history stays.
  - **No CHECK on `kind`.** A Go registry (`internal/identity/consent.go`) validates it. l-05 registers `privacy_notice` and `age_18`, and m4-05 registers the AI kinds and the behavioral opt-in. Nothing writes the table in this sprint.
- `account.invite_id` (added by M1a without an FK): add `REFERENCES identity.invite(id) ON DELETE SET NULL NOT VALID`, then `VALIDATE CONSTRAINT` in the same file. Every existing value is NULL.
- sqlc: `internal/identity/store/queries/invite.sql` and `consent.sql` (only the reads m4-02 needs: live grants by account). Run `sqlc generate`; `sqlc diff` must be clean. Update `internal/identity/store/schema.sql` if the repo keeps it in sync.

### 2 · Seat accounting + invite CLI [X]

Sources: [ADR-0033 §3](../../adr/0033-invite-only-admission-and-owner-admin.md#3-modes-enforcement-and-seats) (what counts, enforcement, R0 seat freeze), §5 (mint), [§8](../../adr/0033-invite-only-admission-and-owner-admin.md#8-the-owner-admin-cli-identity-admin-run-via-kubectl-exec).
- **One seat function.** In `internal/identity/store`, `SeatsUsed(ctx, q) (learners, invites int)` is `active` accounts with role `learner`, plus invites that are unexpired, unredeemed and unrevoked. The owner and testers are outside the cap. m1-04's `reactivate` and `set-role … learner` switch to it, so all four paths count the same way.
- **One lock.** `lockSeats(ctx, tx)` runs `SELECT pg_advisory_xact_lock(hashtext('identity.seats'))`, the key m1-04 already uses. Mint, redeem, `reactivate` and `set-role … learner` each take it inside their transaction before counting.
- **`SEAT_CAP`** (`internal/identity/config.go`): default 15. A value that isn't an integer, or is < 0, is refused at startup. A value **> 40** runs as 40 and logs at ERROR ("`SEAT_CAP` > 40 requires R2, ADR-0033 §1"). This mirrors L7's log-not-alert guard (D34). Record the guard in the decisions log.
- **CLI verbs** in `internal/identity/admin/invite.go`. They use the same distroless-safe dispatch, `flag` parsing, 1-conn pool and "never run migrations" rule as m1-04:

| Verb | Behaviour |
|---|---|
| `invite create --note "…" [--ttl 7d] [--tier standard] [--email x] [--region IN] [--json]` | `--note` is required (1–120 chars); `--ttl` defaults to 7d, minimum 1h, maximum 30d; `--tier` must be in the registry; `--region` must be two uppercase letters; `--email` is stored normalised. One transaction: take the lock, then check `learners + invites + 1 ≤ SEAT_CAP` (else exit 3, `no seats (15/15)`). Generate 16 bytes with `crypto/rand`; the code is its unpadded base64url (22 chars). Insert **only** `sha256(raw 16 bytes)`, then write `admin_audit`. It prints once, to stdout: `https://projects.sujaykumar.dev/xlearn/auth#invite=<code>` (from `PUBLIC_BASE_URL`), the invite id, the expiry and the seats after (e.g. `5/15`). |
| `invite list [--status outstanding\|redeemed\|expired\|revoked\|all] [--json]` | Newest first: id, status, created, expires, tier, region, email-bound, note, and the redeemer (username or email or id; `erased` when `redeemed_account_id` is NULL on a redeemed invite). **Never a code.** |
| `invite revoke <invite-id>` | Sets `revoked_at` when the invite is still unredeemed and unrevoked. It frees the seat. A redeemed invite gets an error: "already redeemed — use `account suspend`". |
| `seats` (m1-04's verb, extended) | `learners N · outstanding invites M · used N+M/SEAT_CAP`, replacing m1-04's "n/a until L-A". |

- **Audit.** Every verb writes `admin_audit` in the same transaction as its change (reads too, as m1-04 does). `detail` holds `{invite_id, ttl_hours, tier, region, email_bound}` and **never the code, the note or the email**. Erase clears the note (task 7), and an audit row can't be cleared. `list` output goes only to the owner's terminal.
- **Tests** (PG 18 store integration, `internal/identity/admin/*_test.go`):
  - the effect of each verb, and its audit row with no code, note or email in `detail` or the captured log;
  - the TTL bounds;
  - `create` exits 3 at the cap;
  - lowering the cap below the outstanding count blocks mint and redeem (the R0 seat freeze);
  - `revoke` of a redeemed invite errors.

### 3 · Redeem on the email path [X]

Sources: [ADR-0033 §5](../../adr/0033-invite-only-admission-and-owner-admin.md#5-invites-mint-and-redeem) (the five-step transaction, invite before email, uniform `invite_invalid`), §9 (threat table); [ADR-0035 §4](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#4-limits-inventory) L3.
`internal/identity/authemail.go` `handleSignup`, by mode:
- **`closed`:** unchanged. 403 `signup_closed` **before the body is read** (task 6 pins it with a golden test).
- **`open`** (compose and dev only; L7 already demotes it to `closed` without `DEV_AUTH`): unchanged. The `invite` field is ignored.
- **`invite`:**
  1. Decode `{email, password, invite}` with a small body cap.
  2. A missing or empty `invite` → **403 `invite_required`**, with no email check.
  3. A malformed code (not 22 base64url chars decoding to 16 bytes) → **403 `invite_invalid`**, with no DB hit.
  4. **Cheap pre-check** outside the transaction: `SELECT 1 … WHERE code_sha256=$1 AND redeemed_at IS NULL AND revoked_at IS NULL AND expires_at > now()`. None → `invite_invalid`. This keeps floods of junk invites from spending bcrypt (L3 still gates bcrypt).
  5. Validate the email format and password (422 `invalid_email` / `weak_password`; these reveal nothing about registration), then hash through m1-04's bcrypt gate. The hash is made **before** the transaction, so the seat lock is never held across bcrypt.
  6. Call the store `CreateInvitedEmailAccount(ctx, codeHash, email, hash, displayName, seatCap)`. It runs **one transaction**:
     1. `lockSeats`.
     2. `RedeemInvite`: `UPDATE identity.invite SET redeemed_at = now() WHERE code_sha256=$1 AND redeemed_at IS NULL AND revoked_at IS NULL AND expires_at > now() RETURNING id, tier, intended_email, region`. No row → `ErrInviteInvalid` (a race loser or an expiry since the pre-check).
     3. Seats: `learners + invites + 1 ≤ SEAT_CAP`, counted after the update (the redeemed invite's seat moves to the account). Else `ErrNoSeats`, rolled back so the invite is not consumed.
     4. Email: `CreateEmailAccount`'s insert. A unique violation → `ErrEmailTaken`, rolled back so the invite is not consumed.
     5. Account row: role `learner` (default), `admitted_via='invite'`, `invite_id`, `region` = the invite's region (NULL if none). `UPDATE invite SET redeemed_account_id = …`. Onboarding row.
     6. Outbox `account_created` (the v2 envelope) with `tier`. Commit.
  7. Map errors to 403 `invite_invalid`, 403 `no_seats` and 409 `email_taken`. On success, `startSession` runs as today.
- **`--email` is not enforced on the email path** (ADR-0033 §5: a typed email is unverified).
- **Uniformity:** every bad-invite case (missing aside) returns the **same status and byte-identical body**: `{"error":{"code":"invite_invalid","message":"This invite link isn't valid. Ask for a new one."}}`. That covers unknown, used, expired, revoked and malformed. An invite holder can still learn that an email is taken (accepted, ADR-0033 §9).
- `marshalAccountCreated` gains `tier`: the invite's tier, or `"standard"` when there is no invite (CLI testers, dev login, `open`). The field is additive on the existing subject: no new subject and no ACL change. Consumers ignore unknown fields (m1-02's `DecodeEnvelope`).

### 4 · Redeem on the GitHub path [X]

Sources: [ADR-0033 §5](../../adr/0033-invite-only-admission-and-owner-admin.md#5-invites-mint-and-redeem) ("Redeem, GitHub path"), §2 (no auto-link into password accounts).
- `oauthTx` (`internal/identity/session.go`) gains `Invite string \`json:"i,omitempty"\``. `handleStart` (`internal/identity/handlers.go`) reads `r.PostFormValue("invite")` from the form POST (bounded `ParseForm`; m1-04 exempted this form POST from the JSON check). It stores the value, capped at 64 chars, **only outside link mode**. The cookie stays HttpOnly with its 10-minute TTL. It isn't signed, but the code is itself the secret and is validated at the callback.
- `handleCallback` → `store.FindOrCreateAccount(ctx, OAuthUpsert{…, Mode, InviteCodeHash, VerifiedEmails, SeatCap})`:
  - **Existing account** (by provider identity, or auto-linked by email into an OAuth-only account as today): sign in. **The invite is untouched.**
  - **Password account with that email:** `account_exists_password`, unchanged (§2 still applies).
  - **New account:**
    - `closed` → `/auth?error=signup_closed` (unchanged);
    - `invite` with no invite in the cookie → **`/auth?error=invite_required`**, and nothing is created;
    - `invite` with an invite → redeem in the create transaction with the same steps 1–3 and 5–6 as the email path. **If `intended_email` is set**, it must match (case-insensitively) one of the GitHub account's **verified** emails. The handler fetches `/user/emails` (the `user:email` scope is already requested) only when an invite is present, and passes the `verified: true` addresses in. A mismatch is `invite_invalid` and is rolled back.
    - **This reads ADR-0033 §5 and t7 §2.3 ("must equal the GitHub-verified email") as "a GitHub-verified email", not only the primary one.** GitHub verifies every address it marks `verified`, so any of them proves the invitee controls the bound address, and matching only the primary would refuse a tester whose primary is a different address, for no security gain. It is a deliberate widening of the ADR's wording: record it in the decisions log (DoD). The account's own email is still resolved as today (`githubPrimaryEmail` in `internal/identity/oauth.go`); the binding check only gates redemption.
    - The errors redirect to `/auth?error=invite_invalid` or `/auth?error=no_seats`.
  - `open` → as today.
- On success the OAuth account row gets `admitted_via='invite'`, `invite_id`, `region` and `redeemed_account_id` exactly as on the email path. The code lives only in the cookie and in memory, and it is never logged: the log line carries `invite_id` after redemption, never the code.
- **Tests** (handler tests with the existing provider fakes, plus a PG store test):
  - new account plus a valid invite → created and redeemed;
  - new account with no invite → `invite_required`, no row;
  - an `intended_email` mismatch → `invite_invalid`, the invite still outstanding;
  - an existing account signs in with an invite in the cookie → the invite is still outstanding;
  - link mode ignores `invite`;
  - two concurrent callbacks with the same invite → exactly one account.

### 5 · Public endpoints [X]

Sources: [ADR-0033 §5](../../adr/0033-invite-only-admission-and-owner-admin.md#5-invites-mint-and-redeem) (`/invites/check` returns only `{valid, expires_at}`, rate-limited); [rollout §4](../rollout-plan.md#4-per-milestone-detail) L-A (the auth page reads the signup mode); [ADR-0035 §4](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#4-limits-inventory) L2.
- **identity (no JWT; ClusterIP, behind MI-5a):**
  - `POST /invites/check {code}` → 200 `{"valid":true,"expires_at":"…"}` or 200 `{"valid":false}` (malformed, unknown, used, expired and revoked are identical). Read-only; it never consumes an invite. In `closed` mode it always answers `{"valid":false}`, so a code can't be probed while nothing can be redeemed.
  - `GET /auth/config` → `{"signup":"closed"|"invite"|"open"}`, the **effective** mode after L7 (`open` without `DEV_AUTH` reports `closed`).
- **gateway** (`internal/gateway/bff.go` `apiRoutes`; `Doc: true`):
  - `{"POST", "/api/invites/check", g.handleInviteCheck, true}`: m1-05's L2 limiter (5/min per IP, `X-Real-Ip`), then forward. The typed 429 with `Retry-After` comes from m1-05.
  - `{"GET", "/api/auth/config", g.handleAuthConfig, true}`: identity's answer cached in process for 30 s, so an anonymous caller can't amplify onto identity. No session is needed.
  - Both are reachable at `/xlearn/api/v1/…` through the existing alias. m1-04's JSON and `Sec-Fetch-Site` checks apply to the POST.
- **No SPA change here.** The auth page reads these routes in [l-05](sprint-l-05.md).
- **Tests:** the identity handler matrix; gateway tests (the 6th check from one IP within a minute → 429; the config cache serves the 2nd call without identity); the OpenAPI drift test (`internal/gateway/openapi_drift_test.go`) goes green after task 8 documents both routes.

### 6 · `SIGNUP_MODE=invite` honoured [X]

Sources: [ADR-0033 §3](../../adr/0033-invite-only-admission-and-owner-admin.md#3-modes-enforcement-and-seats); [ADR-0035 §4](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#4-limits-inventory) L7; [ADR-0034 §2](../../adr/0034-v2-release-labelling-gating-and-rollback.md#2-feature-gating-three-tiers-no-flag-service) (`SIGNUP_MODE` is a permanent operating mode).
- `internal/identity/config.go` `resolveSignupMode`: `invite` → **`invite`** (m1-04 logged it at INFO and ran `closed`). `open` without `DEV_AUTH` stays `closed` with an ERROR log, and anything else stays `closed`. Update m1-04's table test.
- Both create paths now **require** a valid invite in `invite` mode (tasks 3–4). `closed` stays the production value, so every path above is inert on production.
- **Golden test for `closed`** (`internal/identity/closed_golden_test.go`): in `closed` mode, `POST /auth/signup` (with and without an `invite` field) and a new-user GitHub callback give byte-identical status, headers and body to the `v1.12.0` behaviour. This is the "nothing visible changes" guarantee.
- **CLI provisioning is not signup.** `account create --role tester` works in every mode (ADR-0033 §3), and a test pins it.
- compose keeps `SIGNUP_MODE: open` plus `DEV_AUTH`. Add a commented `# SIGNUP_MODE: invite` line in `docker-compose.yml` showing how to exercise invites locally.

### 7 · Erase clears invite notes [X]

Sources: [ADR-0033 §9](../../adr/0033-invite-only-admission-and-owner-admin.md#9-erase-and-abuse-controls) (identity's part: free the seat, null the redeemer, **clear `note`**).
- Extend identity's erase transaction ([l-02](sprint-l-02.md); `identity admin account erase` and `DELETE /api/me` share it). **Before** the `DELETE FROM identity.account`, run `UPDATE identity.invite SET note = NULL, intended_email = NULL WHERE redeemed_account_id = $1`. The FK then nulls `redeemed_account_id`, and `account_consent` rows go with the account (`ON DELETE CASCADE`).
- **`intended_email` is cleared too.** On a redeemed invite it is the erased person's own address, so the ADR's reason for clearing the note ("may name the person") applies to it as well. This goes slightly beyond §9's text: record it in the decisions log.
- The seat frees itself: the account row is gone, and the invite is redeemed so it doesn't count.
- **Test** (PG 18): redeem → erase → the invite has `note IS NULL`, `intended_email IS NULL`, `redeemed_account_id IS NULL` and `redeemed_at` still set; `invite list` shows `erased`; `seats` drops by one; no `account_consent` row remains. [l-04](sprint-l-04.md)'s rehearsal checks the same thing on production.

### 8 · Docs [X]

- `docs/runbooks/identity-admin.md` (m1-04's runbook): add the `invite create|list|revoke` examples through `ssh vps 'k3s kubectl exec -n xlearn deploy/xlearn-identity -- identity admin invite …'`, the extended `seats` output, the R0 seat-freeze lever (lower `SEAT_CAP` by infra PR), and the rule **"no invite in production outside the L rehearsal ([l-04](sprint-l-04.md)) until the v3 opening"**. Log each use in `docs/v2/status.md`.
- `docs/architecture/api.md` + `openapi.yaml`:
  - the two new routes;
  - `invite` on `POST /api/auth/signup`;
  - the new error codes (`invite_required`, `invite_invalid`, `no_seats`);
  - the OAuth start form's `invite` field;
  - the new `/auth?error=` values.
- `docs/architecture/data-model.md`: identity gains `invite` and `account_consent`, and both are per-user data covered by erase. `docs/architecture/events.md`: `identity.account_created` gains `tier`.

## Acceptance criteria

- [ ] **Seat races are impossible.** In PG 18 race tests: 20 concurrent `invite create` at cap 5 → exactly 5 succeed; 10 concurrent redeems of one code (mixing email and GitHub paths) → exactly 1 account; concurrent mint, redeem and `reactivate` never push `learners + invites` above `SEAT_CAP`.
- [ ] **Uniform `invite_invalid`:** the same status and a byte-identical body for used, expired, revoked, unknown and malformed codes on email signup. The GitHub callback redirects to `?error=invite_invalid` in each case, and `/invites/check` gives `{"valid":false}` in each case.
- [ ] **Invite before email:** a bad invite plus a taken email → `invite_invalid` (never `email_taken`); a good invite plus a taken email → 409, and the invite stays outstanding.
- [ ] **GitHub path:** a new account redeems in the create transaction; an `intended_email` mismatch → `invite_invalid`; an existing account signs in with the invite untouched; no invite → `invite_required`.
- [ ] **CLI:** `create` prints the link, id, expiry and seats once. The code is stored only as sha256 and appears in no log or `admin_audit`. `list`, `revoke` and `seats` work, and every verb is audited.
- [ ] **`closed` is byte-for-byte unchanged** (golden test). `open` without `DEV_AUTH` still runs `closed`.
- [ ] **Erase nulls the redeemed invite's `note` (and `intended_email`)** and removes the account's `account_consent` rows (test).
- [ ] `POST /api/invites/check` 429s on the 6th call per IP per minute; `GET /api/auth/config` reports the effective mode.
- [ ] `sqlc diff` clean; migration lint and OpenAPI drift test green; `go test -race ./...` and `-tags e2e` green; CI green; merged to `main`.

## Release

**Merge only. It ships dark in the next tag** after the merge (indicatively `v1.14.0`, cut by [m3-13](sprint-m3-13.md); `v1.15.0` or `v1.16.0` if it lands later). It adds no NATS subject or consumer (`tier` is a field on an existing subject), no in-cluster caller, no pod and no infra change, so the memory sum is unchanged. Its migration is expand only, so the rollback floor is unchanged.
**For the tag sprint that carries it:** add three checks to that tag's smoke, and record them in [`../status.md`](../status.md). They prove "nothing visible changes on production":
- `curl -s https://projects.sujaykumar.dev/xlearn/api/v1/auth/config` → `{"signup":"closed"}` (read-only);
- `identity admin seats` → `outstanding invites 0`. This runs through `ssh vps 'k3s kubectl exec -n xlearn deploy/xlearn-identity -- identity admin seats'`, and m1-04 audits every verb, reads included, so it writes an `admin_audit` row: it is a **sanctioned manual path** ([rollout §2.2](../rollout-plan.md#22-operating-rules-every-mi-step-and-every-tag)), not a read-only check. The **tag sprint's session** runs it itself (approved by that prompt's launch, D40), and it is logged in status.md's CLI-use log;
- the auth page looks as it does in `v1.12.0` (read-only).

The carrying tag sprint's plan (indicatively [m3-13](sprint-m3-13.md)) doesn't list these checks, so the handoff is the note this sprint leaves in status.md (Definition of Done). **Every tag sprint's agent must read status.md's pending-smoke notes before tagging** and run any that apply to its tag.

## Definition of Done

CI green (including `sqlc diff`, the migration lint and the OpenAPI drift test) · PR squash-merged · acceptance criteria met · runbook and architecture docs updated · statuses updated (this file, plus [`../status.md`](../status.md): the board row "merged, rides <next tag> dark"; a **pending-smoke note** for the next tag listing the three checks above, with the tag session running `seats` itself; the flag inventory notes `SIGNUP_MODE` now accepts `invite` and that `SEAT_CAP` defaults to 15 with the > 40 guard; limits L2 (invite check) and L7 (invite, `SEAT_CAP`) → built) · decisions-log lines for: the `hashtext('identity.seats')` key, no CHECK on `kind`/`tier`, `no_seats`, clearing `intended_email` on erase, `intended_email` matched against **any** GitHub-verified email (a deliberate widening of ADR-0033 §5's wording, with the reason), the audit excluding note/email, the `SEAT_CAP` > 40 guard, and `/invites/check` answering `false` in `closed`.

## Risks / watch-outs

- **Invite holders can learn that an email is taken** (a 409 after a valid invite). This is accepted in ADR-0033 §9. Nobody without an invite can.
- **The seat lock is a single global lock.** Never hold it across bcrypt or an HTTP call. Hash first, and fetch GitHub's verified emails first.
- **The GitHub auto-link branch** signs a new GitHub identity into an existing OAuth-only account with the same email, without redeeming. That is intended ("existing accounts sign in untouched"); the invite stays usable. Test it so nobody "fixes" it.
- **Merged while M3 is busy:** identity and gateway route-table conflicts with M3 sprints. Rebase often, and keep gateway edits to two route rows and two small handlers.
- **Don't mint anything on production** to "try it". An outstanding invite is a seat and a live credential, and production must stay `closed` with no invite until the l-04 rehearsal.
- **`account_consent` has no writer yet.** m4-02 reads it (empty = no consent), which is the safe default. Don't seed rows.
- **Migration ordering:** identity migrations are sequential. Take the next free version at rebase; l-01, l-02 and M3 may have added some.

# ADR-0033 — Invite-only admission, account roles and the owner admin CLI

- **Status:** Accepted (2026-09-24, v2 build-plan sign-off). Its §14 amendments are folded into [ADR-0006](0006-authn-authz.md), [ADR-0016](0016-mistake-journal-and-worker-service-auth.md), [ADR-0023](0023-email-password-auth-and-account-linking.md) §2 and [ADR-0024](0024-public-user-dashboards-and-usernames.md) (2026-09-24). Code lands at **M1a/M1b** (columns, `RequireRole`, the CLI) and at **L** (the invite flow, the acceptance step, erase). The **v1.5.x stopgap** (§2) **shipped in v1.5.2** (2026-09-24): the GitHub auto-link fix (xlearn#51) and `SIGNUP_MODE` (xlearn#52; infra#30 sets `closed`). **Production signup has been closed since 2026-09-24.** It doesn't carry §3's `DEV_AUTH` guard on `open`, which lands at **M1b**.
- **Date:** 2026-09-24
- **Deciders:** @sujaykumarsuman
- **Related:** amends [0006](0006-authn-authz.md), [0016](0016-mistake-journal-and-worker-service-auth.md) and [0024](0024-public-user-dashboards-and-usernames.md). [0023](0023-email-password-auth-and-account-linking.md) §3 is amended by v1.5.2 (xlearn#51), which records the change there. Extends [0019](0019-account-settings-onboarding-and-reminder-gating.md) (onboarding gains a step 0). Builds on [0028](0028-object-storage-and-backups.md) (D13, invite-only signup), [0030](0030-runner-technology-and-host-hardening.md) (D21, runner placement), [0031](0031-platform-ai-and-two-tier-keys.md) (tier, consents, 18+, owner AI limit), [0032](0032-realtime-ai-mock-interviewer.md) (region, D31), [0027](0027-content-evalpack-and-user-data-model.md) (erase, visibility), [0025](0025-public-profiles-under-u-prefix.md) (course-slug guard) and [0022](0022-path-enrollment-and-dev-login.md) (the `DEV_AUTH` gate). Siblings: [0034](0034-v2-release-labelling-gating-and-rollback.md) (labels, the T-3 cohort) and [0035](0035-v2-operations-nats-auth-limits-capacity.md) (limits, NATS auth, no alerting).
- **Links:** v2 topic T7: [feasibility § T7](../v2/feasibility.md#t7--cross-cutting-and-infra-first-rollout), the [rollout plan](../v2/rollout-plan.md) (milestone and MI ids) and the [research appendix](../v2/research/t7-cross-cutting-and-rollout.md).

## Context

D13 settled **controlled, invite-only signup** for capacity reasons and left the mechanism to T7. When T7 was researched (2026-09-24, before v1.5.2) nothing enforced it, and the auth surface was built for one user.

| Area | Today | Evidence |
|---|---|---|
| Signup | **Open to the internet.** `POST /api/auth/signup` creates a usable account at once (no invite, no email verification), and a first GitHub sign-in creates one too. **Fixed in v1.5.2** (xlearn#52 + infra#30): `SIGNUP_MODE=closed`, closed in production since 2026-09-24 | `internal/identity/authemail.go:14-51`; `internal/identity/handlers.go:124-134` (at v1.5.1) |
| **Pre-account hijack** | `FindOrCreateAccount` links a GitHub identity into *any* account whose email matches, and email/password accounts are never verified. An attacker pre-registers `victim@x` with a password; the victim later "continues with GitHub", lands in the attacker's account, and the attacker keeps password access (including the victim's BYO coach key). ADR-0023 §3 reasoned only about the *provider's* email being verified. **Fixed in v1.5.2** (xlearn#51, live 2026-09-24) | `internal/identity/store/store.go:194-207` (196-209 at v1.5.1); [USENIX Sec '22](https://arxiv.org/abs/2205.10174) |
| Enumeration | Signup returns `409 email_taken`; login skips bcrypt for an unknown identifier, so the "uniform 401" leaks through timing | `authemail.go:42-44,66-67` |
| Sessions | 30-day TTL, no account-status check, no revoke-all | `internal/identity/handlers.go:164-181` (validate returns only `account_id`/`expires_at`, no status check); `store/store.go:559-565`; `config.go:106` (30-day TTL) |
| Roles | Every mint is `roles: ["learner"]`, and **nothing reads `Roles`** | `internal/gateway/bff.go:703,841`; `internal/platform/auth/auth.go:37-38` |
| Internal HTTP | `/sessions/*` and `/internal/*` are unauthenticated and rely on a NetworkPolicy **that doesn't exist**: `xlearn`, `databases` and `messaging` have none (only `flux-system`, `kubescope` and `longhorn-system` do) | `internal/identity/service.go:70-84`; `k3s kubectl get networkpolicy -A` (2026-09-24) |
| Rate limits | **None.** Traefik has only strip-prefix, redirect-slash and Longhorn ForwardAuth middlewares. identity runs bcrypt under a 250m CPU limit, so about 4–8 password logins a second (inferred) stall `/sessions/validate`, which every API call needs | `k3s kubectl get middleware -A`; `../infra` `apps/xlearn-identity.yaml:70-71` |
| Public profile | The only unauthenticated `/api` route mints an ordinary `learner` JWT for the resolved account, shows every active path, and publishes mock best and average | `internal/gateway/public.go:56-60,140,144`; `bff.go:699-709` |
| **Shared origin** | `projects.sujaykumar.dev` also serves **kubescope (cluster-admin, exec)**, landscape and the read-write Longhorn UI. kubescope's cookie is scoped only by **path**, which isolates nothing within one origin. **One XSS in xLearn while the owner is signed in to kubescope is cluster-admin**, and from there `sops-age` decrypts every secret, with no backups (D12). v2 adds AI prose, Markdown and profile content to this origin | `../infra` `README.md`, "Admin consoles" and "Same-origin caveat" |

Production had one account, the owner's, on 2026-09-23.

**Owner decisions (2026-09-24):**
- **D33:** close v1 signup now; v2 admits through owner-minted invite links, managed by a CLI, with no web admin.
- **D35:** v2 is for the owner only, and real learners come no earlier than v3. **The invite flow stays in v2 scope anyway.** Deferring real users is the owner's own choice, not a technical limit of v2.

## Decision

### 1. Audience: owner-only v2, invite flow built (owner: D35)

| | v2 (build line through `v2.x`) | The opening (planned for v3; the owner's call) |
|---|---|---|
| Who uses production | The owner and CLI-minted `tester` accounts | Invited learners, up to `SEAT_CAP` |
| `SIGNUP_MODE` in production | `closed`, except during the L rehearsal | `invite` |
| Invite flow, acceptance, privacy notice, 18+, region, erase, `tester` | **Built, shipped and exercised** | Used by real learners |

- **L's exit test** is an invite round-trip **on production** with a tester-operated browser, on both create paths: the owner sets `SIGNUP_MODE=invite` and mints an invite; the account is created (role `learner`), accepts, solves and is graded on the self path, and then erases itself on the web. Then production goes back to `SIGNUP_MODE=closed`.
  - judge stays cohort-gated before GA, so a `learner` account gets no judge grade. To rehearse one, the owner may run `identity admin account set-role <user> tester` after redemption; that also frees the seat.
- **v2.0 GA** (MI + M1–M4 + pilot + L, complete for the owner) flips the v2 defaults for every account. In v2 that means the owner and testers ([0034](0034-v2-release-labelling-gating-and-rollback.md)).
- **Opening gates**, recorded now for v3. All must hold before the first real invite:
  - MI-5b is live (§11);
  - alerting has been revisited ([0035](0035-v2-operations-nats-auth-limits-capacity.md), D34);
  - `SEAT_CAP` has been re-sized from at least 2 weeks of M4 data;
  - the privacy notice and erase are live;
  - production is set to `SIGNUP_MODE=invite`;
  - open signup (D21's trigger) or `SEAT_CAP` > 40 (a T7 threshold that extends D21) requires R2, the dedicated runner VPS.

### 2. Close v1 signup now (v1.5.x stopgap: shipped in v1.5.2)

- **`SIGNUP_MODE`**, with production at `closed`, blocks every self-service create path: email signup and the new-account branch of the GitHub callback. Login keeps working. **Live in v1.5.2** (MI-2c: xlearn#52, `1b90d2b`; infra#30 sets `closed` on identity), so **production signup has been closed since 2026-09-24**. As shipped:
  - `SIGNUP_MODE ∈ {open, closed}`, with `invite` reserved for v2; anything but `open`, unset included, resolves to `closed`;
  - `POST /auth/signup` returns 403 `signup_closed`, checked before the body is read, so it is no email oracle; a new GitHub user lands on `/xlearn/auth?error=signup_closed`; existing accounts log in as before;
  - docker-compose sets `open`;
  - **it honours `open` without `DEV_AUTH`.** The §3 `DEV_AUTH` guard is therefore **M1b** work, not part of the stopgap.
- **GitHub auto-links only into accounts without a password hash.** Otherwise: "sign in with your password, then Connect GitHub". This amends ADR-0023 §3, and that patch records it there. **Live in v1.5.2** (MI-2b: xlearn#51, `e915479`); it changes no signup behaviour.
- **Accounts created before v1.5.2** stay v1-only until M1a marks them `grandfathered` and M1b brings the `suspend` verb. They can't run code before M3, and judge is cohort-gated until GA anyway.
- **Rule:** before the `v2.0.0` flip, `identity admin account list` must show no `active` account with role `learner`. Each stranger is suspended or erased through the CLI.

### 3. Modes, enforcement and seats

- **`SIGNUP_MODE ∈ {closed, invite, open}`** and **`SEAT_CAP`** are env vars on the `xlearn-identity` HelmRelease. Changing one is an infra PR (the T-2 tier, minutes).
  - `closed` blocks every self-service create path. CLI provisioning of a tester (§8) is not signup, and it works in every mode.
  - `invite` requires a valid invite on both create paths.
  - **`open` is for local and dev only.** identity honours `open` only when `DEV_AUTH` is also set (the ADR-0022 gate). Otherwise it runs as `closed` and logs at ERROR. This replaces the opscheck assertion that D34 dropped. **This guard is M1b work** (L7 in [0035](0035-v2-operations-nats-auth-limits-capacity.md)): v1.5.2 honours `open` without `DEV_AUTH`, and until M1b only infra#30's explicit `closed` keeps production closed. **Production at `open` is D21's open-signup trigger**, so enabling it is a code change behind a new ADR, never an env flip.
- **What counts as a seat:** an `active` account with role `learner`, plus every unexpired, unredeemed, unrevoked invite. The owner and testers are outside the cap.
- **Sizing:**
  - `SEAT_CAP = floor((AI_APP_CAP − OWNER_LIMIT) / (1.2 × P90 learner-month))`, using D25's $80 app cap and T5's ≈ $12 owner limit.
  - That gives 17 at DSA `medium` ($3.17), 13 with SD at 10% ($4.27), and 29 at `low`.
  - **Initial value: 15.** Platform AI binds first, at 13–17 seats. CPU binds only at about 95 learners active in the same hour.
  - The value is re-sized from M4 data before any opening.
- **Enforcement:**
  - Mint, redeem and `reactivate` each take `pg_advisory_xact_lock('identity.seats')` and check the cap.
  - Redeem re-checks `seats_used ≤ SEAT_CAP`, because the invite already holds its seat. Lowering the cap therefore freezes outstanding invites too; this is the R0 "seat freeze" lever.

### 4. Schema (identity; the ADR-0005 expand rules)

| When | Change |
|---|---|
| **M1a** (expand) | `account` gains the following columns, all nullable or with constant defaults:<ul><li>`role` (`learner`\|`tester`\|`owner`, default `learner`)</li><li>`status` (`active`\|`suspended`)</li><li>`admitted_via` (`grandfathered`\|`invite`\|`cli`\|`dev`)</li><li>`invite_id`</li><li>`accepted_at`</li><li>`region`</li></ul>Also new: `admin_audit(id, at, verb, target, detail jsonb)`, because the M1b CLI writes it. |
| **L-A** | `invite(id, code_sha256 bytea UNIQUE, created_at, expires_at, note ≤ 120 chars, intended_email NULL, tier DEFAULT 'standard', region NULL, redeemed_at, redeemed_account_id → account ON DELETE SET NULL, revoked_at)`. Also `account_consent(account_id, kind, version, granted_at, withdrawn_at)`, holding the notice version and the 18+ attestation. |
| **M4** | `account_consent` gains the two AI consent kinds from [0031](0031-platform-ai-and-two-tier-keys.md) plus the behavioral opt-in |

### 5. Invites: mint and redeem

- **Mint (L):**
  - Run `ssh vps 'k3s kubectl exec -n xlearn deploy/xlearn-identity -- identity admin invite create --ttl 7d [--tier standard] [--email x] [--region IN] --note "…"'`.
  - It prints `https://projects.sujaykumar.dev/xlearn/auth#invite=<22 chars>`, plus the invite id, the expiry and the seats (e.g. `5/15`).
  - **The code is 128 bits** (16 random bytes, base64url). Only its **sha256** is stored. The TTL defaults to **7 days**, with a **30-day maximum**.
  - The owner sends the link over his own channel; **xLearn sends no email**.
- **The link:**
  - The code rides in the **URL fragment**, so it never reaches Traefik logs or the `Referer` header. `/auth` already exists, so the course-slug guard is unchanged.
  - The SPA stashes the code in `sessionStorage`, calls `history.replaceState` to drop it from the address bar, and opens on Sign up.
  - `POST /api/invites/check {code}` returns only `{valid, expires_at}`, and it is rate-limited.
- **Redeem, email path.** The client sends `POST /api/auth/signup {email, password, invite}`. Everything runs in **one transaction**:
  1. Take the seat lock.
  2. Run the conditional `UPDATE invite … WHERE code_sha256=$1 AND redeemed_at IS NULL AND revoked_at IS NULL AND expires_at > now() RETURNING tier, intended_email, region`.
  3. Check the email.
  4. Create the account (`learner`, `admitted_via=invite`).
  5. Write the outbox event `account_created{tier}`.

  - **The invite is checked before the email**, so signup is not an email oracle for anyone without an invite.
  - Invalid, expired, used and revoked invites all get the same **`invite_invalid`**. A taken email rolls back the transaction, so the invite is not consumed.
- **Redeem, GitHub path.** The form POST to `/api/auth/github/start` carries a hidden `invite` field, and identity keeps it in the existing HttpOnly `oauthTx` cookie (10 min).
  - A **new** account redeems inside the same create transaction. If `intended_email` is set, it must equal the GitHub-verified email.
  - An **existing** account just signs in, and the invite is untouched.
  - With no invite, the user goes to `/auth?error=invite_required`, and nothing is created.
  - The §2 no-auto-link rule still applies.
- On the email path `--email` is not enforced, because a typed email is unverified.

### 6. The acceptance step (onboarding step 0; L-A)

- **Where it runs:** before ADR-0019's `path` step, **every account passes it once**, the owner and testers included. Each item below is recorded.
- **What it asks for:**
  - an **18+ attestation**;
  - the **privacy-notice version**;
  - the **region**, pre-filled from the invite (default `IN`) and confirmed by the learner. [0032](0032-realtime-ai-mock-interviewer.md)'s EU/EEA voice gate uses it;
  - from M4, the **two unticked AI consents**.
- **Gateway enforcement:** it returns **`403 acceptance_required`** on every non-onboarding API until the step is complete, and again whenever the notice version changes.
- **Session-validate** returns `role`, `status`, `accepted` and `created_at`.
- **The notice** names Anthropic, provider retention of up to 30 days, no training and processing outside India. It lists the toggles and states D12's data-loss window: with no backups, up to about 7 days of data can be lost with the node or its disk.
- **Any new static SPA segment** (e.g. `privacy`) joins the course-slug guard.

### 7. Roles and status live in identity's database, never in the JWT

| Role | How it's set | `SEAT_CAP` | T-3 cohort ([0034](0034-v2-release-labelling-gating-and-rollback.md)) | Web erase |
|---|---|---|---|---|
| `owner` | `identity admin account set-role <user> owner`, once at M1b. The CLI refuses to demote or suspend the last owner | outside | yes | **never** (CLI only), so no XSS can wipe the owner |
| `tester` | CLI only: `account create --role tester` or `set-role <user> tester`. Never through an invite | outside | yes | yes, from L-E |
| `learner` | an invite redemption (or `grandfathered`) | counts | no | yes, from the L exit |

- **Testers** exist to exercise non-owner paths (acceptance, erase, the invite round-trip, preview gating) on production without exposing the owner account.
  - They have no admin power: admin needs the kubeconfig.
  - **Keep them few:** they spend platform AI from the same app cap while sitting outside `SEAT_CAP`.
- **JWTs stay `["learner"]` for every account's user routes**, and `["public-read"]` is used only for the public profile (M2b).
  - **No `owner` or `tester` role is ever minted.**
  - The gateway reads `role` from session-validate for cohort gating, and judge keys the owner's overrides by account id ([0031](0031-platform-ai-and-two-tier-keys.md)).
- **`status=suspended`:**
  - session-validate joins on `status='active'`, so every session dies at once;
  - the public profile returns 404 immediately;
  - the seat is freed;
  - the username is kept.

### 8. The owner admin CLI (`identity admin`, run via `kubectl exec`)

The kubeconfig is already the root of trust, so the CLI adds no web surface and no new secret: it uses the pod's own DB credentials. There is precedent in `identity -version` (`cmd/identity/main.go:34-38`, a distroless image).

| Verb | Milestone |
|---|---|
| `account list [--dormant 30d] [--role r] [--status s]` · `account suspend \| reactivate \| revoke-sessions <user>` · `account set-role <user> learner\|tester\|owner` | M1b |
| `account create --role tester --email <e>`: prints an initial random password **once**, which the tester changes in Settings (the ADR-0023 endpoint) | M1b |
| `seats`: used / cap and outstanding invites (invites are counted from L) | M1b |
| `invite create \| list \| revoke` | L-A |
| `account erase <user>`: the only erase path for the owner or a suspended account | L-E |
| judge adds `breaker`, `ai-disable`, `llm-limit`, `review-flags` and `disputes export` with the same pattern and its own audit rows | M3 / M4 |

- `<user>` is an email, a username or an account id.
- **Every verb writes `admin_audit`** plus a log line.
- **No secrets are stored or logged:** invite codes and tester passwords appear only once, on the owner's terminal.
- With no alerting (D34), seat and invite counts are read **on demand** with `seats` and `account list`, not from a digest.

### 9. Erase and abuse controls

- **Erase** (the mechanics are in [0027](0027-content-evalpack-and-user-data-model.md) §6, as amended by [0028](0028-object-storage-and-backups.md)).
  - **Web requirements:** `DELETE /api/me` needs a typed confirmation and **a session under 5 minutes old**.
  - **Who can use it:** open to testers at L-E and to every non-owner at the L exit. The owner is always refused on the web.
  - **identity's part:**
    - frees the seat;
    - nulls `redeemed_account_id` through the FK;
    - **clears `note`** on invites the account redeemed, because the owner's note may name the person;
    - releases the username after its 60-day cooldown.
  - **Acknowledgements:** identity waits only for the erase consumers listed in `topology.go` of the running version. There are 4 at L-E; judge is born with its consumer at M3-1.
  - **Gates:** erase needs NATS auth through N3 plus MI-5, the `databases`/`messaging` NetworkPolicies ([0035](0035-v2-operations-nats-auth-limits-capacity.md)). The L track also needs MI-5a.

| Threat | Control |
|---|---|
| Leaked or shared link | Single use (the first redeemer wins). `invite list` shows the redeemer; the owner can `invite revoke` and `account suspend` |
| Brute force | 128-bit codes, lookup by sha256; per-IP limits on login, signup, GitHub start and invite check (L1–L2 in [0035](0035-v2-operations-nats-auth-limits-capacity.md)) |
| Enumeration | The invite is checked before the email; login runs a dummy bcrypt for unknown identifiers (L3). An invite holder can still learn that an email is taken (accepted) |
| bcrypt DoS | At most 2 bcrypt in flight, then 429 (L3); the login limits (L1) |
| Seat races | The advisory lock on mint, redeem and reactivate |
| Pre-account hijack | §2 |
| Cross-site writes from sibling `*.sujaykumar.dev` hosts (SameSite=Lax admits them) | Mutating calls require `Content-Type: application/json` and `Sec-Fetch-Site ∈ {same-origin, none}`; xLearn CSP (M1b) |

### 10. No web admin, no waitlist

- **No `/xlearn/admin` page.** Any XSS in a same-origin app could drive it with the owner's session. Revisit only on a separate host, after §11.
- **No waitlist form.** The auth screen instead shows "xLearn is invite-only" with a **`mailto:` "Request an invite"** link to the owner's contact address, set in config.
- The `AB23` admin artboard is dropped.

### 11. Admin-console isolation (MI-5b)

"No web admin" doesn't remove the existing path: an XSS on the shared origin can drive kubescope to `exec` into identity and run this very CLI. So:

- **MI-5b (infra + owner).**
  - Add a DNS record and a cert for **`ops.sujaykumar.dev`**.
  - Move the IngressRoutes for **kubescope, landscape and the Longhorn UI** there, together with Longhorn's ForwardAuth, plus airlift's `/api/admin` if it is exposed.
  - After that, `projects.sujaykumar.dev` carries no admin console.
- **The gate.** **The first non-owner account on production, a CLI-minted `tester` included, needs MI-5b live.** It is also an opening gate (§1).
  - It is recommended earlier, ideally before the M1 Markdown renderer ships.
  - **Interim:** an IP allowlist on kubescope and Longhorn, or kubescope exec switched off.
- **Same-site caveat.** A sibling subdomain is cross-origin but still **same-site**, so the consoles' cookies still ride requests that an XSS on `projects.sujaykumar.dev` triggers, and WebSocket upgrades (kubescope's exec) aren't covered by CORS.
  - MI-5b's acceptance therefore checks that the consoles reject cross-origin requests (an `Origin` / `Sec-Fetch-Site` check on mutating calls and WebSocket upgrades).
  - If they don't, the IP allowlist stays.
- Owner bookmarks change. landscape and kubescope remain the owner's monitoring tools (D34).

### 12. Authz deltas across v2

| # | Surface | v2 change | Milestone |
|---|---|---|---|
| 1 | Account creation | closed → invite + `SEAT_CAP` + acceptance; production stays `closed` in v2 apart from the L rehearsal | ✅ stopgap (MI-2c, `closed` live in v1.5.2) → M1a (columns) → M1b (`DEV_AUTH` guard on `open`) → L (flow) |
| 2 | Login | Per-IP and per-identifier limits; dummy hash; bcrypt semaphore | M1b |
| 3 | GitHub auto-link | Only into password-less accounts | ✅ stopgap (xlearn#51, live in v1.5.2) |
| 4 | Sessions | `status` join; revoke-all on suspend, password change and erase; validate returns `role`, `accepted`, `status` and `created_at` | M1b |
| 5 | JWT roles | Shared `auth.RequireRole`: user routes need `learner`; `/public/stats` accepts **only** `public-read`; no `owner`/`tester` role in JWTs | middleware M1b; `public-read` M2b |
| 6 | Owner/admin | `account.role` and `admin_audit` (M1a); `identity admin account` and `seats` (M1b); `invite` verbs (L); `judge admin` (M3/M4) | M1a / M1b / L / M3–M4 |
| 7 | Internal HTTP | **The NetworkPolicy is the fence** (MI-5a): the gateway admits only Traefik; internal routes admit only the `xlearn` namespace; the runner is denied. Still no service tokens | **before M3** |
| 8 | Enrollment | The slug must be an `active` course, or a `preview` course for the owner and testers | M1b |
| 9 | judge, learner-facing | `aud=judge` JWT; every read scoped to `sub` (IDOR tests); per-account idempotency; quotas; a DTO allowlist | M3 |
| 10 | judge, internal | practice → judge `/internal/*` behind MI-5a; regrade only after practice authorizes | M3 / M4 |
| 11 | Runner / eval pack | Bearer token (SOPS); ingress only from judge; the pack never goes through the BFF (denylist test) | M3 |
| 12 | Erase | §9: fresh session + typed confirmation; the owner is refused on the web; testers from L-E, every non-owner from the L exit; consumers re-verify via `/internal/erasures/{id}` | L-E (after N3) → L exit |
| 13 | Platform AI | judge reads `tier`, `status`, consents and `ai_disabled` from identity (5-minute cache) | M4 |
| 14 | Interviewer | `aud=coach`; one non-terminal interview; ≤ 2 starts a day; EU/EEA voice gate from `account.region` | M6 |
| 15 | Same origin | Admin consoles to `ops.sujaykumar.dev` (MI-5b); xLearn CSP; `Sec-Fetch-Site` + JSON checks; Permissions-Policy | MI-5b before the first non-owner account; M1b (CSP, checks); M6b (Permissions-Policy) |

### 13. Public-dashboard authz deltas

| Task | Change | Milestone |
|---|---|---|
| D31 | **Mock count only**: `best` and `average` leave the public payload and tile; the authed Progress keeps them | M1b (the v1.5.2 stopgap didn't carry it) |
| Scope | Only **enrolled ∩ visible ∩ active** courses (D7); `preview` is hidden everywhere, public stats included. The resolver returns `visible_courses[]` | M1a columns → M1b filter |
| `public-read` | The public route mints `["public-read"]`, and only assessment's `GET /public/stats` accepts it; `/progress/*` and `/mocks/*` reject it; it gets its own cache namespace | **M2b** |
| Abuse | 60/min per IP (burst 20); 404s cached negatively for 60 s; ≤ 8 concurrent cold composes | M1b |
| Shape | A public-shape allowlist test: no item ids, per-item pattern, sub-day timestamps, arena or Run counts, prose, `scored_by` or transcripts | M1b |
| Visibility | Profile and per-course toggles. Private returns the same 404 as unknown. A toggle bumps the cache epoch synchronously. Header totals come from visible courses only | M2b |
| Suspend / erase | Uniform 404 at once; username held for 60 days after erase | M1b / L-E |

### 14. Amendments to earlier ADRs

| ADR | Section | Change |
|---|---|---|
| **0023** Email/password sign-in + GitHub↔email account linking | §3 "Collision = auto-link by verified email" | Auto-link only into accounts with no password hash. **Made by v1.5.2 (xlearn#51) and recorded in 0023 by it**; referenced here |
| 0023 | §2 Endpoints | (This ADR) `POST /auth/signup` and a new GitHub account need a valid invite in `invite` mode and are refused in `closed` mode. `closed` shipped in v1.5.2 (xlearn#52, which added a dated note to 0023 §2); `invite` comes with L. Login runs a dummy bcrypt, so the uniform 401 is also uniform in time |
| **0006** AuthN/AuthZ | Authorization | "Single-role (`learner`) + ownership, no admin UI" becomes:<ul><li>roles are **enforced** (`RequireRole`), and `public-read` is added;</li><li>the account role (`learner`/`tester`/`owner`) and `status` live in identity's DB, **never in the JWT**;</li><li>admin is a CLI, and there is still no admin UI;</li><li>sessions gain revoke-all and a status check;</li><li>"NetworkPolicy as defence in depth" becomes **the fence** (MI-5a);</li><li>broker auth moves to nkeys ([0035](0035-v2-operations-nats-auth-limits-capacity.md)).</li></ul> |
| **0016** Mistake journal, weekly weak-area & the worker service-auth path | §3 "Background workers reach identity over a ClusterIP internal endpoint (no JWT)" | "When a NetworkPolicy lands (S12)": S12 never landed one, and **MI-5a is that fence**, before M3. `/internal/accounts/{id}` grows beyond scheduling prefs at M4 (`tier`, `status`, consents, `ai_disabled`), which is allowed only because the fence is live by then. Still no service tokens |
| **0024** Public user dashboards + usernames | §3 composition; Consequences | The ordinary `learner` mint becomes an enforced `public-read`. Courses are enrolled ∩ visible ∩ active. Mock stats are count only (D31, also recorded by 0032). The "private toggle later" arrives at M2b. The route also gains the limits, allowlist test and suspend/erase 404 from §13 |

## Consequences

- ✅ **D13 is enforced** since 2026-09-24, well before L: v1.5.2's `SIGNUP_MODE` (MI-2c, xlearn#52) runs `closed` in production (infra#30). The pre-account hijack is fixed in the same release (xlearn#51).
- ✅ **The smallest admin surface:**
  - no web admin and no new secret;
  - every admin action sits behind the kubeconfig and is written to `admin_audit`;
  - the role never rides in a bearer token.
- ✅ **The v3 opening is a config change plus its gates, not a build.** The flow ships in v2 and is rehearsed on production.
- ✅ **Non-owner paths are testable on production** (testers), without exposing or erasing the owner.
- ✅ **Capacity is an auditable number in git**, tied to the AI budget, with a freeze lever.
- ⚠️ **Until M1b, only the env keeps production closed.** v1.5.2 honours `open` without `DEV_AUTH`, so a mistaken infra PR setting `open` would reopen signup; §3's guard closes that at M1b.
- ⚠️ **Minting needs `ssh`**, so there is no phone flow. xLearn sends no email: a lost link is revoked and re-minted by hand.
- ⚠️ **`admin_audit` is a trail for honest operations, not a tamper-proof log.** Anyone with exec can also edit it, but that person is already root.
- ⚠️ **The NetworkPolicy is the only service-auth fence**, and a regression fails open. With no alerting (D34), detection is `host-verify --cluster`'s NetworkPolicy-presence check, run on demand.
- ⚠️ **MI-5b moves the owner's consoles**, and a sibling subdomain is still same-site: the consoles must reject cross-origin requests, or the IP allowlist stays.
- ⚠️ **18+ and region are attestations**, not verification.
- ⚠️ **The in-process rate limiter pins the gateway to one replica** (a scale-out blocker, recorded in [0035](0035-v2-operations-nats-auth-limits-capacity.md)).
- ⚠️ **External feedback arrives only after the v3 opening.** This is the owner's accepted trade (D35).

## Alternatives considered

| Option | Why not |
|---|---|
| Owner-only web page `/xlearn/admin` with an `owner` role in the JWT | An XSS in any same-origin app could mint invites or suspend learners with the owner's session. It needs CSRF and role plumbing plus an artboard. Revisit only on its own host, after MI-5b |
| In-app waitlist form | It stores PII of people who weren't admitted (notice and erase duties), needs bot protection, and there is no email sender, so the owner delivers links by hand anyway. The `mailto:` link captures demand at no cost |
| Email allowlist | Unsafe with unverified email signup: anyone who types a listed address gets in. It also has no expiry and isn't single-use. Kept only as the optional `--email` binding, enforced on the GitHub path |
| Open signup with a cap and an auto-waitlist | That is open signup: bots, enumeration, stranger PII, and D21's move of the runner to R2 |
| `owner` (or `tester`) role in the JWT | No web surface needs it, and a bearer claim widens what any leaked or XSS-driven token can do. judge keys overrides by account id |
| Leave v1 signup open until the L gate | Strangers could create accounts for months against D13, and each would need triage before M3 |
| Bearer-token admin API (the airlift `/api/admin` pattern) | An internet-exposed privileged route behind a static secret shared across apps |
| Raw SQL runbook | No seat lock, no audit, codes hashed by hand. Emergency fallback only |
| Invite real learners during v2 (a friends beta after M3, or ≤ 5 after M2b) | The owner keeps v2 closed (D35). It would also pull L-A, the notice, MI-5a/MI-5b and erase forward, and fix `SEAT_CAP` before any M4 data |
| Drop the invite flow from v2, since nobody is invited before v3 | The owner kept it in scope. Built and rehearsed on production in v2, the opening becomes a config change rather than a rushed build |

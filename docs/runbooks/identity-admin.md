# Runbook — the owner admin CLI (`identity admin`)

**What:** the only admin surface xLearn has ([ADR-0033](../adr/0033-invite-only-admission-and-owner-admin.md)
§8, §10: no web admin). It lives in the identity binary and runs **inside the running identity pod** with
the pod's own database credentials. It adds no web surface and no secret, and it never runs migrations.
Built in [m1-04](../v2/sprints/sprint-m1-04.md); ships in `v1.7.0`.

**Why `kubectl exec` is allowed:** it is a sanctioned manual path ([rollout plan §2.2](../v2/rollout-plan.md)).
The kubeconfig is already the root of trust. It is a data operation on schema `identity`, never an infra
change: no `kubectl apply`, and GitOps is untouched.

> **Log every use** in [`docs/v2/status.md`](../v2/status.md) (decisions log or the owner-events table):
> the date, the verb, the target account id (never an email), and why. `identity.admin_audit` is the
> in-database trail. The command's output reaches your terminal, not `kubectl logs`.

## The call

The image is distroless (no shell), so every call is one exec of the binary:

```sh
ssh sujaykumar-vps 'k3s kubectl exec -n xlearn deploy/xlearn-identity -- identity admin <verb> …'
```

`<user>` is an **email** (contains `@`), an **account id** (UUID) or a **username**. Email and username
match case-insensitively. `identity admin help` prints the usage.

| Exit | Meaning |
|---|---|
| `0` | done |
| `1` | database or unexpected error (nothing changed; the audit row says `outcome: error`) |
| `2` | usage error (nothing ran, nothing audited) |
| `3` | `<user>` matches no account |
| `4` | refused by a guard: the last active owner, the seat cap, or an email that is taken |

Every verb, reads included, writes **one `identity.admin_audit` row** in the same transaction as its
change. A refused or failed verb rolls back and still writes an audit row saying why (`detail.outcome`:
`ok`, `refused_last_owner`, `refused_seat_cap`, `refused_email_taken`, `user_not_found`, `error`). It
also writes **one JSON log line on stderr** (`"msg":"identity admin"`, with `verb`, `target` and
`outcome`). Audit rows and log lines hold ids, roles, statuses and counts. They **never** hold a
password, a code or an email. `target` is the account id, or `-`.

## Verbs

### `account list [--dormant 30d] [--role r] [--status s] [--json]`

```sh
… identity admin account list --role owner
… identity admin account list --dormant 30d --status active --json
```

It prints a table: `ID EMAIL USERNAME ROLE STATUS ADMITTED CREATED LAST SESSION`, then `N account(s)`.
With `--json` it prints an array of `{id, email, username, role, status, admitted_via, created_at,
last_session_at}`. `--dormant` accepts `Nd` or a Go duration (`72h`) and keeps accounts with no session
created within the window, "never signed in" included.

### `account suspend <user>`

```sh
… identity admin account suspend <user>
# suspended <id> <email> (was active); revoked 2 session(s)
```

It sets `status='suspended'` and revokes every session, in one transaction. The account's very next API
call gets a 401, because session-validate joins `status='active'`. The account can't sign in, by
password (the uniform 401) or by GitHub (`/auth?error=account_unavailable`). Its seat is freed and its
username is kept. **It refuses the last active owner** (exit 4). A repeat is a no-op that still audits.

### `account reactivate <user>`

It sets `status='active'`. Reactivating a **learner** takes a seat: it takes the seats lock and refuses
at `SEAT_CAP` (exit 4). Reactivating the owner or a tester never counts against the cap.

### `account revoke-sessions <user>`

```sh
# revoked 3 session(s) of <id> <email>
```

It signs the account out everywhere. The account stays active.

### `account set-role <user> learner|tester|owner`

```sh
# role of <id> <email>: learner -> owner
```

It changes `role` only, never `admitted_via`.

- **It refuses to demote the last active owner** (the `identity.owners` lock).
- An active account **becoming a learner** takes a seat and is refused at `SEAT_CAP`.
- **Promoting is always allowed**, which is how the first owner is set.
- Roles live in identity's database only. JWTs stay `["learner"]` for every account, and the gateway
  reads the role from session-validate on every request, so a change applies on the next request.

### `account create --role tester --email <email>`

```sh
… identity admin account create --role tester --email <tester-email>
# created tester <id> <email>
# one-time password (shown once; the tester changes it in Settings):
# <24 characters>
```

In M1b it creates only `tester` accounts (learners arrive by invite from L-A).

- It sets `admitted_via='cli'` and creates an onboarding row and an `account_created` outbox event, like
  email sign-up.
- The random 24-character password (crypto/rand, base64url) is printed **once**, on stdout. It is never
  stored in plain text, logged or audited. Hand it over out of band. The tester changes it in Settings,
  which signs them out everywhere and back in.
- An email that is already taken is refused (exit 4).
- **The CLI can't know whether MI-5b is live. You gate it** (below).

### `seats`

```sh
# seats: 3 used / 15 (SEAT_CAP) — active learners; the owner and testers are outside the cap
# invites: n/a until L-A
```

It prints active learners against `SEAT_CAP`. `SEAT_CAP` is identity env, and the code default is 15
until L-A sets it on the HelmRelease. Invites join the count at L-A (l-03).

## Owner events

### `ev-owner-role` — once, right after the `v1.7.0` verify (m1-07's session, D40)

```sh
ssh sujaykumar-vps 'k3s kubectl exec -n xlearn deploy/xlearn-identity -- identity admin account set-role <owner-email> owner'
ssh sujaykumar-vps 'k3s kubectl exec -n xlearn deploy/xlearn-identity -- identity admin account list --role owner'
```

Expect `learner -> owner`, then exactly one row. Log it in `status.md` (`ev-owner-role` ✅, with the
account id). Until this runs there is no owner, so nothing is in the preview cohort and the last-owner
guard has nothing to protect. l-02 relies on the role to refuse web erase for the owner. **Production
has no account yet** (status.md): if `<owner-email>` matches nothing (exit 3), the event waits for the
owner's first sign-in. Record that as ⛔ and don't create the account.

### `ev-first-tester` — only after MI-5b is live (l-02's session)

**Gate:** MI-5b must be ✅ ([mi-04](../v2/sprints/sprint-mi-04.md); ADR-0033 §11). While it is 🔄 (4b
pending) no tester may exist on production, because a non-owner account on the shared origin is exactly
what MI-5b protects against.

```sh
ssh sujaykumar-vps 'k3s kubectl exec -n xlearn deploy/xlearn-identity -- identity admin account create --role tester --email <tester-email>'
```

Hand the one-time password over out of band. The tester signs in with email and password and changes it
in Settings. Log `ev-first-tester` in `status.md` with the account id.

## Stranger triage (the GA rule, ADR-0033 §2)

Any active account that isn't the owner, a tester, or an invited learner is a stranger.

```sh
… identity admin account list --role learner --status active
… identity admin account suspend <user>      # one per stranger; erase arrives at L-E (l-02)
… identity admin seats
```

Suspending kills their sessions at once and frees the seat. `account erase` arrives with L-E.

## Notes

- **Locks:** `pg_advisory_xact_lock(hashtext('identity.owners'))` guards the last-owner check, and
  `…('identity.seats')` guards the cap. They are taken in that order, then the account row
  (`FOR UPDATE`).
- **Honest-operator trail:** `admin_audit` is not tamper-proof. Anyone with exec in the pod is already
  root (accepted in ADR-0033). Read it with
  `SELECT at, verb, target, detail FROM identity.admin_audit ORDER BY at DESC LIMIT 20;`.
- **Local (compose):** `docker compose exec identity identity admin …`. The compose identity image is
  the same binary.

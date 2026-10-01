# External / BFF API surface

The **only** public API is the gateway's, mounted at **`/xlearn/api/v1`** on `projects.sujaykumar.dev`
(same origin as the SPA → cookie auth, no CORS). The unversioned **`/xlearn/api`** prefix is kept as a
compat alias, served identically ([ADR-0021](../adr/0021-release-tagging-and-api-versioning.md)). Internal
service APIs are in [`services.md`](services.md). Auth model: [ADR-0006](../adr/0006-authn-authz.md).

## Conventions

- **Auth:** HttpOnly `Secure` `SameSite=Lax` session cookie. `credentials: include` from the SPA.
  Unauthenticated → `401` with `{ "error": { "code": "unauthenticated" } }`; the SPA redirects to OAuth.
- **Content type:** `application/json`; streaming coach responses use **SSE** (`text/event-stream`).
- **Cross-site write checks (M1b, m1-04; [ADR-0033](../adr/0033-invite-only-admission-and-owner-admin.md) §9):**
  every mutating call (`POST`/`PUT`/`PATCH`/`DELETE` on `/api/…`) must be `Content-Type: application/json`
  (with or without a body) → else `415 unsupported_media_type`, and a `Sec-Fetch-Site` header, when present,
  must be `same-origin` or `none` → else `403 cross_site_request`. The OAuth start form POST
  (`/auth/{provider}/start`) keeps only the `Sec-Fetch-Site` check. The SPA shell carries a strict CSP
  (`script-src 'self'`, `form-action 'self' https://github.com`); every response carries `nosniff`.
- **Sessions (m1-04):** a suspended account's sessions stop validating at once. A password change revokes
  every session of the account, the caller's included: `POST /me/password` answers `{ok:true, reauth:true}`
  and clears the cookie. Login, sign-up and password change may answer `429 too_many_requests` with
  `Retry-After: 1` while identity's two bcrypt slots are busy (L3). OAuth for a suspended account redirects
  to `/auth?error=account_unavailable`.
- **Errors:** consistent envelope `{ "error": { "code", "message", "details"? } }`; HTTP status
  reflects the class (`400/401/403/404/409/413/422/429/5xx`).
- **Limits (M1b, m1-05; [ADR-0035](../adr/0035-v2-operations-nats-auth-limits-capacity.md) §4) fail loudly:**
  - `429` with a `Retry-After` header (whole seconds) and
    `{"error":{"code":"rate_limited","message":…,"retry_after":N}}` — L1 `POST /auth/login` (10/min per IP,
    burst 5; and 5 failures / 15 min per identifier, answered without calling identity), L2
    `POST /auth/signup` and `POST /auth/{provider}/start` (5/min per IP each), L4 `GET /u/{username}` (60/min
    per IP, burst 20), L5 every authenticated call (20/s per account, burst 40; writes also 5/s, burst 10).
  - `429 {"error":{"code":"busy",…,"retry_after":1}}` (`Retry-After: 1`) — the public profile's cold-compose
    cap (≤ 8 at once).
  - Other services' 429s keep their own codes (identity's `too_many_requests`, the coach's
    `provider_limited`); the SPA keeps the server's `code`/`reason` and only adds `retryAfter`.
  - `413 {"error":{"code":"body_too_large","message":…,"limit":N}}` — any request body over its cap
    (`N` bytes; 1 MiB by default, 8 KiB for `POST /auth/login`), including the proxied auth routes.
  - The per-IP key is `X-Real-Ip` as Traefik sets it; exempt: `/healthz`, `/readyz`, `/api/healthz`, JWKS,
    static assets.
- **IDs & time:** string ids as in the domain; timestamps ISO-8601 UTC. Pagination via `?cursor=&limit=`.
- **Versioning:** `/xlearn/api/v1` is the 1.0 surface (introduced at the 1.0 milestone, S12); the
  unversioned `/xlearn/api` is kept as a same-origin compat alias ([ADR-0021](../adr/0021-release-tagging-and-api-versioning.md)).
- **BFF aggregation:** endpoints marked **`agg`** fan out to several services server-side.
- **Courses (M1b, m1-03; [ADR-0026](../adr/0026-per-course-extensibility-model.md) §5, t0 §7):**
  course-scoped routes live under the existing `/paths/{slug}/…` prefix (no parallel `/courses` tree);
  the gateway resolves `{slug}` against the course manifests compiled into it. Unknown or retired courses,
  a `coming_soon` course's data routes and a `preview` course outside the owner/tester cohort all get the
  same `404 {"error":{"code":"course_not_found"}}`. Items keep **global** ids (`/problems/{id}`,
  `/revision/{itemId}/score`, `/mistakes/{id}`, `/mocks/{id}`); the course is the item's `path_slug`.
  Internal calls carry the course as `?path=<slug>` (defaulting to `course.DefaultSlug` when absent).
- **DSA aliases ([ADR-0034](../adr/0034-v2-release-labelling-gating-and-rollback.md) §1.1):** the v1 routes
  without a course — `/dashboard`, `/progress`, `/revision/due`, `GET|POST /mistakes`, `/weak-area`,
  `POST /mocks`, `/mocks/trend`, `/concepts/{slug}` — run the course-scoped handler with the DSA course and
  are byte-identical to `/paths/dsa/…`. OpenAPI marks them `deprecated`. The SPA stops calling them in
  `v1.7.0`; they stay at least through `v1.8.0` (earliest removal: `v1.9.0`).
- **Withholding (M1b, m1-06; [ADR-0027](../adr/0027-content-evalpack-and-user-data-model.md) §1):** while an
  item is **live** (an open counted attempt or a due touch), every route that shows it drops its
  answer-bearing fields — `pattern`, `concepts`, `solution_facts` and the hint and solution stages — by one
  gateway function (`internal/gateway/withhold.go`): lists (problem index, week, Today, due queue, mistakes,
  weak area, the score result) hide them while live; the workspace shows the pattern from the hint stage of
  an unsolved item or open attempt (a solved, not-live item as v1) and only the statement on a due touch;
  the arena (`?practice=1`) shows a live item's attempt stage only; the coach context drops them for a live
  or never-solved problem. A withheld field is **absent**, not empty. If practice or review can't answer,
  the gateway withholds as if live (fail closed). `withhold_routes_test.go` pins every route's policy.

## Endpoints (v1)

### Auth & account
| Method | Path | Purpose | Backed by |
|--------|------|---------|-----------|
| `GET` | `/auth/{github\|google}/start` | Begin OAuth; 302 to provider. | identity |
| `GET` | `/auth/{github\|google}/callback` | OAuth callback; sets session cookie; 302 into app. | identity |
| `POST` | `/auth/logout` | Revoke session, clear cookie. | identity |
| `GET` | `/me` | Current account + onboarding state. | identity |
| `PATCH` | `/me` | Update profile / study budget / timezone / reminders. | identity |
| `POST` | `/onboarding/step` | Advance the 3-step onboarding. | identity |

### Curriculum (read)
| Method | Path | Purpose | Backed by |
|--------|------|---------|-----------|
| `GET` | `/paths` | Catalog — the courses this caller may see (active, `coming_soon`; `preview` for the cohort) + status + the learner-safe `course` view. | curriculum (filtered by the gateway) |
| `GET` | `/paths/{slug}` | Roadmap — phases, weeks, totals (+ `path.course`). | curriculum |
| `GET` | `/paths/{slug}/problems` | The course's problem index (the Problems arena). | curriculum |
| `GET` | `/paths/{slug}/weeks/{n}` **agg** | Week thesis + concepts + problem list **with the user's five-touch state**. | curriculum + practice + review |
| `GET` | `/paths/{slug}/concepts/{c}` | Concept reading + code template, keyed on (course, concept). | curriculum |
| `POST` | `/paths/{slug}/start` | Start a course (idempotent). Only `active`: `404 course_not_found` / `409 course_not_available`. | identity |
| `GET` | `/concepts/{slug}` | *Deprecated DSA alias* of `/paths/dsa/concepts/{slug}`. | curriculum |

### Problem workspace
| Method | Path | Purpose | Backed by |
|--------|------|---------|-----------|
| `GET` | `/problems/{id}` **agg** | Problem + **only unlocked stage sections** + user state + active timer. | curriculum + practice |
| `POST` | `/problems/{id}/attempt/start` | Start/resume the attempt; starts the 15-min server timer. | practice |
| `POST` | `/problems/{id}/reveal` | Reveal hint/solution; returns the **penalty ack** if solution revealed early. | practice |
| `POST` | `/problems/{id}/outcome` | Log Clean/Rough/Assisted/Miss (→ may open a mistake). | practice (→ review via event) |

### Revision & mistakes
| Method | Path | Purpose | Backed by |
|--------|------|---------|-----------|
| `GET` | `/paths/{slug}/revision/due` | The course's prioritised due queue. | review |
| `POST` | `/revision/{itemId}/score` | Submit a re-solve; auto-scores → advance or reset. The result carries the item's `problem` (its pattern revealed now the touch concluded, m1-06). | review + curriculum |
| `GET` | `/paths/{slug}/mistakes?status=open\|closed` | The course's journal. | review |
| `POST` | `/paths/{slug}/mistakes` | Create an entry in the course (root cause, insight, category). | review |
| `PATCH` | `/mistakes/{id}` | Update status / revisit. | review |
| `GET` | `/paths/{slug}/weak-area` | The course's current weekly weak-area banner. | review |
| `GET` · `GET` · `POST` · `GET` | `/revision/due` · `/mistakes` · `/mistakes` · `/weak-area` | *Deprecated DSA aliases* of the rows above. | review |

### Mock & progress
| Method | Path | Purpose | Backed by |
|--------|------|---------|-----------|
| `POST` | `/paths/{slug}/mocks` | Start a 45-min mock in the course (setup → session). | assessment |
| `GET` | `/paths/{slug}/mocks/trend` | The course's scored history vs the readiness targets. | assessment |
| `GET` | `/mocks/{id}` | Live session + phase rail state. | assessment |
| `POST` | `/mocks/{id}/score` | Submit the 7-dim rubric → /35 + trend. | assessment |
| `GET` | `/paths/{slug}/dashboard` **agg** | The course's "Today": daily plan, due reviews, weak area, streak, stats. | practice + review + assessment + curriculum |
| `GET` | `/paths/{slug}/progress` **agg** | The course's coverage, heatmap, mastery, rubric trend, outcome mix. | assessment (+ review) |
| `POST` · `GET` · `GET` · `GET` | `/mocks` · `/mocks/trend` · `/dashboard` · `/progress` | *Deprecated DSA aliases* of the rows above. | as above |
| `GET` | `/u/{username}` **agg** | The **public** profile — no session (F009, [ADR-0024](../adr/0024-public-user-dashboards-and-usernames.md)). See below. | identity (resolver) + assessment + curriculum |

**Public profile shape (M1b, m1-05; [ADR-0033](../adr/0033-invite-only-admission-and-owner-admin.md) §13).**
`user{username, displayName, joinedAt, region}` · `totals{solved, streak{current, longest}}` · `mock{count}` ·
`heatmap{days[]{date, solves, reviews}}` (or `null`) · `courses[]{slug, title, solved, total, pct, phases[], patterns[]}`
— and nothing else (an allowlist test pins it). `joinedAt` is a date (`YYYY-MM-DD`); no value carries a
sub-day timestamp. The mock tile is the **count only** (D31: best/average stay on the authed Progress).
`courses` lists only enrolled ∩ publicly visible ∩ `active` courses (`preview` never). An unknown,
malformed or **suspended** username is the same `404 not_found`, at once; a 404 is cached for 60 s, so a
newly claimed or reactivated username can take up to a minute to appear.

### Coach
| Method | Path | Purpose | Backed by |
|--------|------|---------|-----------|
| `GET` | `/coach/key` | Masked key + provider + model + enabled (never the raw key). | coach |
| `PUT` | `/coach/key` | Store/replace the user's provider key (encrypted). | coach |
| `DELETE` | `/coach/key` | Remove the key. | coach |
| `POST` | `/coach/chat` (SSE) | Send a message with page context; streams the coach reply. Optional `assist_ack: <attemptId>` (D27 confirm). Response header `X-Coach-Mode`. | `agg` coach (+ practice, review, assessment) |
| `GET` | `/coach/thread?context=` | Thread history for a page context, plus `gate` (the mode the server applies here). | `agg` coach (+ practice, review, assessment) |

Coach page contexts (m1-03, t0 §7): items keep `problem:<id>`; account-wide contexts stay `catalog`,
`settings`, `general`; every course-scoped context is `<course>:<ctx>` (`<course>:concept:<slug>`,
`<course>:week:<n>`, `<course>:roadmap|dashboard|revision|mistakes|mock|progress`). The gateway and coach
run one shared parser (`course.NormalizeCoachContext`) that maps the v1 forms from open `v1.6.0` tabs to the
DSA course.

**Coach mode gate and D27 (M1b, m1-07; [ADR-0031](../adr/0031-platform-ai-and-two-tier-keys.md) §7, t5 §9).**
For every chat the gateway resolves one server-authoritative gate (`internal/gateway/coach_gate.go`) from
practice's open attempts, review's due queue and assessment's live mock, using m1-06's `live()` predicate:

| Mode | When | Chat |
|---|---|---|
| `locked` | a live mock (from M2a, an open touch) and the course's `coach.off_during` lists it | `409 coach_paused {reason: "mock" \| "touch"}`; nothing sent |
| `attempt` | a problem page with an open counted attempt, a due touch, or never solved | no `pattern` / `concepts` / `solution_facts` reach coach |
| `review` | a problem page, concluded, no touch due, no open attempt | the pattern reaches coach |
| `general` | every other page | coach gets `live_items` (the account's live items) to keep off-limits |

- `GET /coach/thread` adds `gate: {mode, reason?, attempt?: {attemptId, coachAssistAt}}` (`attempt` on a
  problem page with an open counted attempt; `coachAssistAt` is `null` until the coach is used on it). The
  read never writes and never answers 409; if a lookup fails, `gate` is omitted and the thread still loads.
- `POST /coach/chat` relays coach's stream with `X-Coach-Mode: attempt|review|general` and coach's
  `Retry-After`. In order, it may instead answer:
  - `503 coach_state_unavailable` — practice, review or assessment didn't answer; nothing was sent (fails
    closed);
  - `409 coach_paused {reason}` (`X-Coach-Mode: locked`);
  - `409 assist_confirm_required {attemptId, problemId}` — a chat in `problem:<id>`'s context while a counted
    attempt on `<id>` is open and not yet assisted. Resending with `assist_ack: <attemptId>` runs coach's L18
    admission probe (a `429` is relayed and the attempt stays uncapped), then records the assist in practice
    (`503 assist_unavailable` if it can't, nothing sent), then forwards. Once recorded, the attempt is capped at
    **Assisted** and later chats (and reloads) skip the confirm. Chats from any other page don't ask (D27's
    honor-based bypass);
  - coach's L18 `429`s, each with `Retry-After` (whole seconds): `coach_busy` (2 replies streaming on the
    account; `Retry-After: 5`), `coach_rate_limited` (20 messages a minute), `coach_daily_cap` (300 messages a
    **UTC** day; `Retry-After` runs to the next UTC midnight). They are distinct from the provider's
    `provider_limited`.
- `GET /problems/{id}`'s embedded `state` gains `coachAssistAt` — the open attempt's (`null` until the coach
  is used on it, and once the attempt is concluded: the recorded grade then carries the cap), and
  `POST /problems/{id}/outcome` answers `{state, cappedBy: "coach"}` when a self-reported Clean/Rough was
  recorded as Assisted because the coach was used on the attempt.

### System
| Method | Path | Purpose |
|--------|------|---------|
| `GET` | `/healthz` / `/readyz` | Liveness / readiness (per service; gateway aggregates readiness). |

## OpenAPI

The machine-readable contract lives in [`openapi.yaml`](openapi.yaml) (OpenAPI 3.1), kept in
lock-step with the gateway route table by a CI drift check (`internal/gateway/openapi_drift_test.go`):
a route added or removed without updating the spec fails the build ([ADR-0021](../adr/0021-release-tagging-and-api-versioning.md)).
This table is the human-readable companion.

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
  reflects the class (`400/401/403/404/409/422/429/5xx`).
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
| `POST` | `/revision/{itemId}/score` | Submit a re-solve; auto-scores → advance or reset. | review |
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

### Coach
| Method | Path | Purpose | Backed by |
|--------|------|---------|-----------|
| `GET` | `/coach/key` | Masked key + provider + model + enabled (never the raw key). | coach |
| `PUT` | `/coach/key` | Store/replace the user's provider key (encrypted). | coach |
| `DELETE` | `/coach/key` | Remove the key. | coach |
| `POST` | `/coach/chat` (SSE) | Send a message with page context; streams the coach reply. | coach |
| `GET` | `/coach/thread?context=` | Thread history for a page context. | coach |

Coach page contexts (m1-03, t0 §7): items keep `problem:<id>`; account-wide contexts stay `catalog`,
`settings`, `general`; every course-scoped context is `<course>:<ctx>` (`<course>:concept:<slug>`,
`<course>:week:<n>`, `<course>:roadmap|dashboard|revision|mistakes|mock|progress`). The gateway and coach
run one shared parser (`course.NormalizeCoachContext`) that maps the v1 forms from open `v1.6.0` tabs to the
DSA course.

### System
| Method | Path | Purpose |
|--------|------|---------|
| `GET` | `/healthz` / `/readyz` | Liveness / readiness (per service; gateway aggregates readiness). |

## OpenAPI

The machine-readable contract lives in [`openapi.yaml`](openapi.yaml) (OpenAPI 3.1), kept in
lock-step with the gateway route table by a CI drift check (`internal/gateway/openapi_drift_test.go`):
a route added or removed without updating the spec fails the build ([ADR-0021](../adr/0021-release-tagging-and-api-versioning.md)).
This table is the human-readable companion.

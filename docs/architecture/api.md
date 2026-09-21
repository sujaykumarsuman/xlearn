# External / BFF API surface

The **only** public API is the gateway's, mounted at **`/xlearn/api/v1`** on `projects.sujaykumar.dev`
(same origin as the SPA → cookie auth, no CORS). The unversioned **`/xlearn/api`** prefix is kept as a
compat alias, served identically ([ADR-0021](../adr/0021-release-tagging-and-api-versioning.md)). Internal
service APIs are in [`services.md`](services.md). Auth model: [ADR-0006](../adr/0006-authn-authz.md).

## Conventions

- **Auth:** HttpOnly `Secure` `SameSite=Lax` session cookie. `credentials: include` from the SPA.
  Unauthenticated → `401` with `{ "error": { "code": "unauthenticated" } }`; the SPA redirects to OAuth.
- **Content type:** `application/json`; streaming coach responses use **SSE** (`text/event-stream`).
- **Errors:** consistent envelope `{ "error": { "code", "message", "details"? } }`; HTTP status
  reflects the class (`400/401/403/404/409/422/429/5xx`).
- **IDs & time:** string ids as in the domain; timestamps ISO-8601 UTC. Pagination via `?cursor=&limit=`.
- **Versioning:** `/xlearn/api/v1` is the 1.0 surface (introduced at the 1.0 milestone, S12); the
  unversioned `/xlearn/api` is kept as a same-origin compat alias ([ADR-0021](../adr/0021-release-tagging-and-api-versioning.md)).
- **BFF aggregation:** endpoints marked **`agg`** fan out to several services server-side.

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
| `GET` | `/paths` | Catalog — all paths + status. | curriculum |
| `GET` | `/paths/{slug}` | Roadmap — phases, weeks, totals. | curriculum |
| `GET` | `/paths/{slug}/weeks/{n}` **agg** | Week thesis + concepts + problem list **with the user's five-touch state**. | curriculum + practice + review |
| `GET` | `/concepts/{slug}` | Concept reading + code template. | curriculum |

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
| `GET` | `/revision/due` | The prioritised due queue. | review |
| `POST` | `/revision/{itemId}/score` | Submit a re-solve; auto-scores → advance or reset. | review |
| `GET` | `/mistakes?status=open\|closed` | Journal list. | review |
| `POST` | `/mistakes` | Create/edit an entry (root cause, insight, category). | review |
| `PATCH` | `/mistakes/{id}` | Update status / revisit. | review |
| `GET` | `/weak-area` | Current weekly weak-area banner. | review |

### Mock & progress
| Method | Path | Purpose | Backed by |
|--------|------|---------|-----------|
| `POST` | `/mocks` | Start a 45-min mock (setup → session). | assessment |
| `GET` | `/mocks/{id}` | Live session + phase rail state. | assessment |
| `POST` | `/mocks/{id}/score` | Submit the 7-dim rubric → /35 + trend. | assessment |
| `GET` | `/dashboard` **agg** | "Today": daily plan, due reviews, weak area, streak, stats. | practice + review + assessment + curriculum |
| `GET` | `/progress` **agg** | Coverage, heatmap, mastery, rubric trend, outcome mix. | assessment (+ review) |

### Coach
| Method | Path | Purpose | Backed by |
|--------|------|---------|-----------|
| `GET` | `/coach/key` | Masked key + provider + model + enabled (never the raw key). | coach |
| `PUT` | `/coach/key` | Store/replace the user's provider key (encrypted). | coach |
| `DELETE` | `/coach/key` | Remove the key. | coach |
| `POST` | `/coach/chat` (SSE) | Send a message with page context; streams the coach reply. | coach |
| `GET` | `/coach/thread?context=` | Thread history for a page context. | coach |

### System
| Method | Path | Purpose |
|--------|------|---------|
| `GET` | `/healthz` / `/readyz` | Liveness / readiness (per service; gateway aggregates readiness). |

## OpenAPI

The machine-readable contract lives in [`openapi.yaml`](openapi.yaml) (OpenAPI 3.1), kept in
lock-step with the gateway route table by a CI drift check (`internal/gateway/openapi_drift_test.go`):
a route added or removed without updating the spec fails the build ([ADR-0021](../adr/0021-release-tagging-and-api-versioning.md)).
This table is the human-readable companion.

# Services

Per-service responsibility, API surface, data owned, and events. Boundaries and consolidation are set
in [ADR-0003](../adr/0003-service-decomposition.md); all internal APIs are HTTP/JSON on ClusterIP,
authenticated by the gateway-minted JWT ([ADR-0006](../adr/0006-authn-authz.md)). Events use the
subjects in [`events.md`](events.md).

Legend — **Deploy:** `edge` = has Traefik route; `internal` = ClusterIP only. **Store:** owning schema.

---

## gateway (BFF) · `edge`

- **Responsibility:** the only internet-facing service. Serves the embedded React SPA under `/xlearn`,
  exposes `/xlearn/api/*`, validates the session cookie, mints internal JWTs, and **aggregates**
  multi-service data for screens (Dashboard, Week, Progress). Proxies coach streaming responses.
- **Owns:** nothing (read-through; no schema).
- **Calls:** all services (sync). **Events:** none.
- **Key endpoints:** see [`api.md`](api.md) (external surface = the gateway's surface).
- **Notes:** screen aggregation lives here precisely because services can't cross-join
  ([ADR-0005](../adr/0005-data-ownership-and-migrations.md)).

## identity · `internal`

- **Responsibility:** OAuth 2.0/OIDC with GitHub & Google; account records; server-side sessions;
  onboarding state (path chosen, budget set, key added); RS256 JWT issuance + JWKS.
- **Owns:** schema `identity` — `account`, `oauth_identity`, `session`, `onboarding`.
- **API:** `POST /auth/{provider}/start`, `GET /auth/{provider}/callback`, `POST /sessions/validate`,
  `POST /sessions/revoke`, `GET /accounts/{id}`, `GET /.well-known/jwks.json`, `POST /paths/{slug}/start`
  (only an `active` course: unknown or `preview` → `404 course_not_found`, `coming_soon`/`retired` →
  `409 course_not_available`; `public_visible` from the manifest — m1-03, ADR-0033 §12 row 8).
- **Emits:** `xlearn.identity.account_created` (a v2 envelope from `v1.7.0`; account-scoped, no
  `path_slug`). **Consumes:** —.

## curriculum · `internal`

- **Responsibility:** the content rail — paths, phases, weeks, problems and their content sections.
  Read-heavy; **seeded** from the versioned `curriculum/` source ([PRD Q2](../prd/xlearn-prd.md#10-open-questions)).
  Enforces *content* structure (stage content, links); per-user gating lives in `practice`.
- **Owns:** schema `curriculum` — `path`, `phase`, `week`, `problem`, `problem_section`, `concept`.
- **API:** `GET /paths` (every non-retired course + its learner-safe `course` view), `GET /paths/{slug}`,
  `GET /paths/{slug}/problems`, `GET /paths/{slug}/weeks/{n}`, `GET /paths/{slug}/concepts/{c}` (keyed on
  course + concept), `GET /problems/{id}` (with stage-scoped sections), `GET /problems?ids=`, and
  `GET /concepts/{slug}` (the DSA alias, m1-03).
- **Emits:** —. **Consumes:** —.

## practice · `internal`

- **Responsibility:** per-user problem lifecycle — stage gating (attempt→hint→solution→re-implement→log),
  **server-authoritative timers** (15/10 min), reveal-penalty rule, and outcome logging
  (Clean/Rough/Assisted/Miss). The gatekeeper of "solved".
- **Owns:** schema `practice` — `user_problem_state`, `attempt`, `stage_event`, `timer`, `outcome`, `outbox`.
- **API:** `GET /state?week=`, `GET /state/{problemId}`, `POST /problems/{id}/attempt/start`,
  `POST /problems/{id}/reveal` (returns penalty ack), `POST /problems/{id}/outcome`. The writes take
  `?path=<course>` (the item's course, from the gateway; absent → `course.DefaultSlug`, m1-03).
- **Emits:** `xlearn.practice.attempt_logged`, `xlearn.practice.problem_solved`,
  `xlearn.practice.solution_revealed_early`. **Consumes:** —.

## review · `internal`

- **Responsibility:** two aggregates in one service —
  1. **Five-touch scheduler:** on solve, schedule Day 1·3·7·21·45; auto-score re-solves (pattern < 2 min
     ∧ correct-in-timer ∧ complexity stated); fail → reset to Day 1; Day 21/45 under mock conditions.
  2. **Mistake journal:** open entries on below-Clean outcomes/fails; 8 categories; weekly weak-area;
     close after 2 clean revisits, re-open on later fail.
  Plus a **periodic sweep** materialising "due today" and a **notifications worker** (reminders).
- **Owns:** schema `review` — `revision_item`, `touch_result`, `mistake_entry`, `weak_area_snapshot`,
  `reminder`, `outbox`.
- **API:** `GET /revisions/due`, `POST /revisions/{id}/score`, `GET /mistakes?status=`,
  `POST /mistakes`, `PATCH /mistakes/{id}`, `GET /weak-area/current`. The course-scoped reads and the
  create take `?path=<course>` (absent → `course.DefaultSlug`, m1-03); the weekly weak-area snapshot is
  per `(account, course, week)`.
- **Emits:** `xlearn.review.revision_scheduled`, `xlearn.review.revision_due`,
  `xlearn.review.mistake_opened`, `xlearn.review.mistake_closed`.
  **Consumes:** `practice.problem_solved`, `practice.attempt_logged`, `practice.solution_revealed_early`.

## assessment · `internal`

- **Responsibility:** two aggregates —
  1. **Mock:** 45-min timed sessions, phase rail, 7-dim rubric scoring (/35), trend vs targets.
  2. **Progress:** read-model **projections** (coverage, revision heatmap, pattern mastery, rubric
     trend, outcome mix) built from practice/review/mock events — so screens read fast without cross-joins.
- **Owns:** schema `assessment` — `mock_session`, `rubric_score`, and projection tables
  `proj_coverage`, `proj_heatmap`, `proj_mastery`, `proj_outcome_mix`, plus consumer offsets/`inbox`.
- **API:** `POST /mocks`, `GET /mocks/{id}`, `POST /mocks/{id}/score`, `GET /mocks/trend`,
  `GET /progress/summary`, `GET /progress/heatmap`, `GET /progress/mastery`. `POST /mocks`, the trend and
  the summary's mock figures take `?path=<course>` (absent → `course.DefaultSlug`, m1-03); the projections
  stay account-grain until M2b. `POST /mocks` for a course with no mock (or a rubric other than DSA's,
  the only one scored so far) is `404 {"error":{"code":"not_found","message":"course has no mock"}}`.
  Every internal `?path=` names a known course or gets `404 course_not_found`.
- **Emits:** `xlearn.assessment.mock_completed`.
  **Consumes:** `practice.*`, `review.*` (to update projections).

## coach · `internal`

- **Responsibility:** the AI-coach gateway — stores each user's provider key **encrypted**
  ([ADR-0007](../adr/0007-ai-coach-byo-key-and-secrets.md)), builds page-context prompts (Socratic
  during attempts, reviewer post-solve), and fans out to the user's LLM provider. Never returns the key.
- **Owns:** schema `coach` — `api_key_config` (encrypted), `coach_thread`, `coach_message`.
- **API:** `PUT /keys` (store), `GET /keys` (masked only), `DELETE /keys`,
  `POST /chat` (context + prompt → provider; streams back; `?path=` = a problem context's course),
  `GET /threads?context=`. Every context is normalized with the shared parser
  (`course.NormalizeCoachContext`, m1-03): course-scoped contexts are keyed `<course>:<ctx>`.
- **Emits:** —. **Consumes:** page context via API (reads practice/curriculum through the gateway).

## notifications *(worker inside `review` in v1)*

- **Responsibility:** turn `review.revision_due` + study-budget windows into reminders. v1 has a single
  in-app channel (surfaced on Dashboard); email/push would justify splitting it into its own service.
- **Owns:** `review.reminder` (v1). **Consumes:** `review.revision_due`.

## web (SPA)

- **Responsibility:** all 12 screens (React + TS, Vite). Built to static assets, **embedded in and
  served by gateway** ([ADR-0008](../adr/0008-frontend-stack.md)). Talks only to `/xlearn/api`.
- **Owns:** —.

---

## Service ↔ screen matrix

Which services back each screen (`A` = aggregated by gateway from several).

| Screen | identity | curriculum | practice | review | assessment | coach |
|--------|:--:|:--:|:--:|:--:|:--:|:--:|
| Auth / onboarding | ● | ○ | | | | ○ |
| Catalog | | ● | | | | |
| Roadmap | | ● | ○ | | ○ | |
| Dashboard `A` | ○ | ○ | ● | ● | ● | ○ |
| Week | | ● | ● | ○ | | |
| Concept | | ● | | | | ○ |
| Problem | | ● | ● | | | ● |
| Revision | | ○ | ○ | ● | | ○ |
| Mistakes | | ○ | | ● | | ○ |
| Mock | | ● | | | ● | ○ |
| Progress `A` | | ○ | ○ | ● | ● | |
| Settings | ● | | | | | ● |

● primary · ○ secondary/context

---

## Network fences (v2)

Ingress NetworkPolicies, live since 2026-09-28 ([mi-03](../v2/sprints/sprint-mi-03.md): MI-5 infra#42, MI-5a
infra#44; [ADR-0035 §2](../adr/0035-v2-operations-nats-auth-limits-capacity.md), ADR-0030 A4). Callers are
selected by namespace (`kubernetes.io/metadata.name`) and `app.kubernetes.io/instance`, **never
`app.kubernetes.io/part-of`**: the chart puts that label on the Deployment only, so it matches no pod.

**MI-5a is the internal-HTTP fence.** identity's unauthenticated `/sessions/*` and `/internal/*` and review's
service-to-service worker calls rely on it ([ADR-0006](../adr/0006-authn-authz.md) and
[ADR-0016](../adr/0016-mistake-journal-and-worker-service-auth.md) as amended by
[ADR-0033](../adr/0033-invite-only-admission-and-owner-admin.md) §12 row 7 / §14). There are still no service
tokens. The gateway also serves **in-namespace JWKS**: every JWT-verifying service fetches
`xlearn-gateway:8080/.well-known/jwks.json` in-cluster, so the gateway admits same-namespace pods as well as
Traefik. "Traefik only" would break every service's JWT check on its next pod start, hidden for up to 1 h by
the stale-key tolerance in `internal/platform/auth/jwks.go`.

| Caller (`instance`, ns `xlearn`) | PG 5432 | NATS 4222 | HTTP it calls (in `xlearn`) |
|---|---|---|---|
| `xlearn-gateway` | **no** | no | identity :8081, curriculum :8082, practice :8083, review :8084, assessment :8085, coach :8086 |
| `xlearn-identity` | yes | forward (N2, mi-06) | gateway :8080 (JWKS) |
| `xlearn-curriculum` | yes | no | none |
| `xlearn-practice` | yes | yes | gateway :8080 (JWKS) |
| `xlearn-review` | yes | yes | gateway :8080 (JWKS); identity :8081 and curriculum :8082 (ADR-0016 workers) |
| `xlearn-assessment` | yes | yes | gateway :8080 (JWKS) |
| `xlearn-coach` | yes | forward (L-E, l-01) | gateway :8080 (JWKS) |
| `xlearn-judge` (MI-13, m3-07) | forward | forward | gateway :8080 (JWKS); called by the gateway and practice |
| Traefik (`kube-system`, `app.kubernetes.io/name: traefik`) | — | — | gateway :8080 only |
| CNPG operator (`cnpg-system`) | :8000 status only | — | — |
| node / host (kubelet probes, the API-server proxy `host-verify` uses) | node-local, always admitted | 8222 node-local | probes |

| Policy | Admits |
|---|---|
| `databases/projects-pgstore-ingress` | 5432 ← the 6 DB-owning services + judge (**not the gateway**); 8000 ← `cnpg-system`; instance↔instance 5432/8000 |
| `messaging/nats-ingress` | 4222 ← practice, review, assessment + identity, judge, coach (forward-declared); **no 8222 rule** |
| `xlearn/xlearn-gateway` | 8080 ← Traefik **and** same-namespace pods (one rule, two peers) |
| `xlearn/xlearn-{identity,curriculum,practice,review,assessment,coach}` | own `containerPort` ← same-namespace pods only (`from: null` drops the chart's Traefik default); `xlearn-runner` is admitted nowhere |

- **Standing rule (ADR-0035 §2):** a new caller updates these policies in its own infra PR, merged before its
  tag. Forward-declared selectors are harmless; a missing caller is blocked **silently**.
- **A regression fails open**, and with no alerting (D34) the only detector is `host-verify --cluster`'s
  NetworkPolicy-presence check against `hack/expected-netpol.tsv` (run on demand).
- **Known limit:** k3s always admits node-local traffic (kubelet probes, the API-server proxy).

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
  `POST /sessions/revoke`, `GET /accounts/{id}`, `GET /.well-known/jwks.json`.
- **Emits:** `xlearn.identity.account_created`. **Consumes:** —.

## curriculum · `internal`

- **Responsibility:** the content rail — paths, phases, weeks, problems and their content sections.
  Read-heavy; **seeded** from the versioned `curriculum/` source ([PRD Q2](../prd/xlearn-prd.md#10-open-questions)).
  Enforces *content* structure (stage content, links); per-user gating lives in `practice`.
- **Owns:** schema `curriculum` — `path`, `phase`, `week`, `problem`, `problem_section`, `concept`.
- **API:** `GET /paths`, `GET /paths/{slug}`, `GET /paths/{slug}/weeks/{n}`,
  `GET /problems/{id}` (with stage-scoped sections), `GET /concepts/{slug}`.
- **Emits:** —. **Consumes:** —.

## practice · `internal`

- **Responsibility:** per-user problem lifecycle — stage gating (attempt→hint→solution→re-implement→log),
  **server-authoritative timers** (15/10 min), reveal-penalty rule, and outcome logging
  (Clean/Rough/Assisted/Miss). The gatekeeper of "solved".
- **Owns:** schema `practice` — `user_problem_state`, `attempt`, `stage_event`, `timer`, `outcome`, `outbox`.
- **API:** `GET /state?week=`, `GET /state/{problemId}`, `POST /problems/{id}/attempt/start`,
  `POST /problems/{id}/reveal` (returns penalty ack), `POST /problems/{id}/outcome`.
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
  `POST /mistakes`, `PATCH /mistakes/{id}`, `GET /weak-area/current`.
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
  `GET /progress/summary`, `GET /progress/heatmap`, `GET /progress/mastery`.
- **Emits:** `xlearn.assessment.mock_completed`.
  **Consumes:** `practice.*`, `review.*` (to update projections).

## coach · `internal`

- **Responsibility:** the AI-coach gateway — stores each user's provider key **encrypted**
  ([ADR-0007](../adr/0007-ai-coach-byo-key-and-secrets.md)), builds page-context prompts (Socratic
  during attempts, reviewer post-solve), and fans out to the user's LLM provider. Never returns the key.
- **Owns:** schema `coach` — `api_key_config` (encrypted), `coach_thread`, `coach_message`.
- **API:** `PUT /keys` (store), `GET /keys` (masked only), `DELETE /keys`,
  `POST /chat` (context + prompt → provider; streams back), `GET /threads/{pageContext}`.
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

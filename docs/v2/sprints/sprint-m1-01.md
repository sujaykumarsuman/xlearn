# Sprint m1-01 — Curriculum spine part 1: compose parity, course manifest + golden, item schema freeze (M1a)

> **Milestone:** M1 — spine (**M1a expand**) · **Track:** product · **Order:** 4
> **Prereqs:** [mi-01](sprint-mi-01.md) (MI-2 prune guard merged — the rollout's M1 entry gate)
> **Unblocks:** [m1-09](sprint-m1-09.md) (converter, loader, curriculum expand) · [m3-01](sprint-m3-01.md) (authoring tooling on the frozen schema) · [mi-05](sprint-mi-05.md) (its NATS-auth test needs compose NATS 2.14)
> **Release action:** **merge only** (ships dark in `v1.6.0`, cut by [m1-02](sprint-m1-02.md))
> **Calendar:** week 1 (2026-09-25 → 10-02); the schema freeze lands **≈ 2026-10-05 at the latest** → owner event `ev-schema-freeze` (starts the owner content track)
> **Execute with:** [`../prompts/prompt-m1-01.md`](../prompts/prompt-m1-01.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | Compose + CI parity (`postgres:18`, `nats:2.14`) | X | ⬜ |
| 2 | `internal/course` manifest package + DSA manifest + two-sided golden test | X | ⬜ |
| 3 | Freeze the item schema (JSON Schema + Go types, answer-free, ADR-0029 fields, freeze guard) | X | ⬜ |
| 4 | Verify (tests, PG 18 store tests, e2e unchanged, no API change) | X | ⬜ |
| 5 | Record the freeze → `ev-schema-freeze`; M1 size note | X | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row,
> the M1 milestone row, the content-status table and the owner-events table). Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] MI-2 merged: the prune guard on the CNPG Cluster, `databases` and `messaging` ([mi-01](sprint-mi-01.md) task 1) — the rollout's M1 entry gate ([rollout §3](../rollout-plan.md#3-milestone-map))
- [ ] MI-2a live ✅ (bounded ranges `>=1.0.0 <2.0.0`, infra#29; `.release-line` guard, xlearn#53)
- [ ] Parallel sessions: no open PR creates `internal/course/` or edits `docker-compose.yml` / `.github/workflows/ci.yml` in a conflicting way (`gh pr list`, `git worktree list`, ListAgents)

## Goal

Unblock owner content authoring as early as possible, with **no behaviour change**: prod-parity compose
(PG 18, NATS 2.14), the typed **course manifest** in a new `internal/course` package with
`curriculum/courses/dsa/course.json` proven equal to v1 (**golden = v1**), and the **frozen item schema** —
including every ADR-0029 / t4 §11.4 #19 content field — so authored items never have to be re-stamped.
Nothing reads the manifest at runtime yet; the first readers arrive in M1b and M2.

## Scope

**In**
- Compose and CI parity: `postgres:18`, `nats:2.14` (`docker-compose.yml:28,45`), CI e2e on PG 18, the embedded
  test NATS server on the 2.14 line.
- `internal/course`: manifest types, strict decoding, validation, a loader over an `fs.FS`, the universal constants (D3).
- `curriculum/courses/dsa/course.json` (golden = v1) plus minimal `coming_soon` stub manifests for the five catalog
  courses, embedded additively in `curriculum/embed.go`.
- A **two-sided golden test**: the manifest equals v1's constants and grading rule (practice grades, review ladder,
  the 7×1–5 rubric, the 8 mistake categories, timers, rail, targets), and each service's own constants equal the manifest.
- Item schema **v1 frozen**: `curriculum/_schema/item.schema.json` + `course.schema.json` + Go types; no field can hold
  an answer; the ADR-0029 field test; a freeze guard (additive-only after this sprint).

**Out**
- One-shot converter, `curriculum/courses/dsa/items/*` layout, glob loader, id/slug guards, the curriculum expand
  migration, the public `content` CI job, the two Example-1 rewrites → [m1-09](sprint-m1-09.md).
- `policy_version` / `content_hash` / `contract_hash` hashing (the canonical-hash package) → [m3-01](sprint-m3-01.md)
  (m1-09 needs `content_hash` first — see its task 3); `problem.spec` → M2a ([m2-01](sprint-m2-01.md)).
- Serving the manifest (`GET /paths/{slug}/manifest`), gateway course resolution, manifest-driven SPA nav →
  [m1-03](sprint-m1-03.md). Readers on new columns (M1b), contract drops ([m1-08](sprint-m1-08.md)).
- Making any dormant manifest value operative: D18 timers via `verdict_timer@1` → [m3-08](sprint-m3-08.md);
  `mistakes.prefill` values → [m3-10](sprint-m3-10.md); `est_minutes` in Today → [m2-04](sprint-m2-04.md);
  `revision.bands` driving touches → [m2-01](sprint-m2-01.md); `coach.persona` in the prompt → [m1-07](sprint-m1-07.md).

## Tasks

### 1 · Compose and CI parity [X]

- `docker-compose.yml:28` `postgres:16-alpine` → `postgres:18-alpine`. PG 18 images keep data under
  `/var/lib/postgresql/18/docker` and refuse a volume mounted at `/var/lib/postgresql/data`: mount `pgdata:/var/lib/postgresql`.
  An existing local PG 16 volume is incompatible — add "PG major bump: `docker compose down -v`" to `deploy/local/README.md` (Reset).
- `docker-compose.yml:45` `nats:2.10-alpine` → `nats:2.14-alpine` (prod runs the 2.14 line,
  `../infra/infrastructure/messaging/release.yaml`); keep `-js -sd /data -m 8222`.
- `deploy/local/initdb.sql`: confirm it runs unchanged on PG 18 (schema creation only).
- `.github/workflows/ci.yml` e2e service `postgres:16` → `postgres:18`.
- The e2e job embeds NATS in-process (`github.com/nats-io/nats-server/v2 v2.10.22` in `go.mod`): move it to the latest
  v2.14.x so tests run the production server line ([mi-05](sprint-mi-05.md)'s NATS-auth test relies on 2.14). `go mod tidy`.
- Run **every store integration test** (`internal/*/store/*_integration_test.go`) against compose PG 18 with
  `XLEARN_TEST_DATABASE_URL` and fix any PG 18 difference. *Recommended:* run them in the e2e job too (it already has a
  PG service), so the next major bump is caught by CI. Production is already PG 18
  (`../infra/infrastructure/database/cluster/cluster.yaml:24`, `postgresql:18.4-system-trixie`).

### 2 · `internal/course` + manifests + golden [X]

**Layout** ([ADR-0027 §2](../../adr/0027-content-evalpack-and-user-data-model.md#2-where-content-lives-and-how-it-ships),
[t1 §3.2](../research/t1-content-data-model.md#32-public-content-layout-and-delivery)): manifests live at
`curriculum/courses/<slug>/course.json` (the register's shorthand `curriculum/dsa/course.json` means this path). The v1
seed files stay in `curriculum/dsa/` until m1-09's converter moves them. `curriculum/embed.go` is extended **additively**:
`//go:embed paths.json dsa/*.json courses/*/course.json _schema/*.json` (m1-09 replaces it with `all:courses`).

**Package** (stdlib only, so every service can import it without cycles):
`internal/course/manifest.go` (types) · `load.go` (`Load(fs.FS) (map[string]*Manifest, error)`: glob
`courses/*/course.json`, `json.Decoder.DisallowUnknownFields`, reject trailing data, then `Validate`) · `validate.go` ·
`universal.go` (D3: ladder `[1,3,7,21,45]`, grades `[clean, rough, assisted, miss]`, core mistake categories
`{misread, time_management, communication}`, the closed nav screen keys, the closed strategy ids
`{self@1, verdict_timer@1, rubric_pct@1, weighted_gate@1}`) · `golden_test.go`.

**Manifest blocks and DSA values** ([ADR-0026 §1, §4](../../adr/0026-per-course-extensibility-model.md#4-universal-vs-per-course-method),
[t0 §6 dsa row](../research/t0-extensibility-frame.md#6-six-course-fit): "v1's constants, pinned by a golden test"):

| Block | Fields | DSA value (golden = v1 unless marked dormant) | v1 source → first runtime reader |
|---|---|---|---|
| identity | `format: 1`, `slug`, `title`, `status` (active\|preview\|coming_soon\|retired), `id_prefix` | `dsa`, "Data Structures & Algorithms", `active`, `dsa` (DSA ids stay bare — m1-09's id guard) | `curriculum/paths.json` (consistency test) → m1-09 |
| `nav` | `item_noun`; `groups[{cap?, items[{screen, label}]}]`, screen ∈ closed set | "problem"; Today · Roadmap · Problems · Progress / cap "Practice loop": Revision · Mistakes · Mock interview | `web/src/nav.ts` `navForPath` → m1-03 |
| `stages` | `attempt{duration_s,label}`, `hint{duration_s,label}`, `solution{label}`, `reimplement` (required\|optional\|off), `early_reveal_penalty_days` | 900 · 600 · `required` · 3 | `internal/practice/store/store.go:39-44` → M2a |
| `grading` | `strategy`; `self_report{outcome,touch,mock}` (allowed\|evaluator_only); `params{<strategy id>: {…}}` | `self@1` (cap none = v1's pick, `POST /problems/{id}/outcome`); all `allowed`; **dormant** `params["verdict_timer@1"]` = D18: `time_limit_s 2700`, `hint_at_s 900`, `clean_within_s 1200`, `max_failed_for_clean 3`, `free_classes [CE, REJECTED]` | → m3-08 selects `verdict_timer@1` |
| `grades` | the universal four, with labels | clean "no help, in time" · rough "solved, ugly" · assisted "needed a hint" · miss "didn't get it" | `web/src/screens/Problem.tsx:21-26`; practice `last_outcome` CHECK |
| `revision` | `ladder_days` (= universal); `bands[{levels, format, label, timer_s, parts, criteria[{key, threshold_s?}], mock_mode}]` (no per-band minutes — `plan.est_minutes` is the one source); `drills` | L1–3: `format resolve`, label "Re-solve", 1200 s, criteria `pattern_named_fast` (< 120 s), `correct_in_timer`, `complexity_stated`, `mock_mode false`; L4–5: same, `mock_mode true`; `drills false` | `internal/review/store/store.go:27-43,712`; `web/src/screens/Revision.tsx:14` → m2-01 |
| `mistakes` | `categories[{id,label}]` ⊇ core (ids append-only); `prefill[{signal, category, strength}]` | the 8 v1 ids `misread, wrong_pattern, right_pattern_wrong_state, off_by_one, language_bug, complexity_misjudged, communication, time_management`; `prefill: []` | `internal/review/store/migrations/00002_*.sql:28-30` → m3-10 fills prefill ([t4 §6.3](../research/t4-judge-contract.md#63-mistake-pre-fill-two-tiers-data-only)) |
| `mock` | `duration_s`; `rail[{label, start_min, end_min, prompt}]`; `rubric{id, dims[{id,label}], scale [1,5]}` (max = 5 × dims); `targets{w13,w15,pre}`; `pools[]`; `evidence_hints[]` | 2700; the 6-phase v1 rail; `dsa-mock@1`, 7 dims `communication, problem_understanding, brute_force, optimisation, code_quality, edge_cases, complexity` → /35; 24 / 28 / 30; `pools []`; `evidence_hints []` | `internal/assessment/mock.go:30-43,147`; `internal/assessment/store/migrations/00001_*.sql` → M1b/M6a |
| `plan` | `est_minutes{course_attempt, touch_<format>…, mock?}` (D4) — **the single source of minutes**, keyed by band `format` (`touch_resolve`, `touch_recall`, …); readers (AB03, m1-06, m2-04) look up `plan.est_minutes["touch_" + band.format]` | **dormant** proposal `course_attempt 45` · `touch_resolve 20` · `touch_recall 5` · `mock 45` (v1 plans by a count of 3, `internal/gateway/dashboard.go:23`; `touch_recall` unused until a DSA band uses recall) | → m2-04 |
| `public_stats` | `default_visible`; `metrics[]` (closed set) | `true`; v1's public tiles minus mock best/average (D31 lands in m1-05) | D7 → m2-03 |
| `coach` | `persona` (≤ 600 chars), `primary_language`, `off_during` | the course-specific lines of `internal/coach/prompt.go:56-57`; `go`; `[touch, mock]` | [t5 §9](../research/t5-platform-ai.md#9-coach-and-byo-changes-in-v2) → m1-07 |

**Coming-soon stubs:** `curriculum/courses/{system-design,go-concurrency,lld-ood,sql,behavioral}/course.json` with only
`format, slug, title, status: coming_soon, id_prefix, public_stats.default_visible` (behavioral `false`, D7). Proposed
prefixes `sd`, `gc`, `lld`, `sql`, `beh` (any unique `^[a-z]{2,4}$`; m1-09 locks them via `path.id_prefix`). The schema
lets a `coming_soon` manifest omit the method blocks; `active` and `preview` require all of them.

**Validation** (`Validate`, run by `Load`): enums; `slug` matches `^[a-z0-9]+(-[a-z0-9]+)*$` (m1-09 adds the reserved-segment
list); `id_prefix` unique across manifests; ladder and grades equal the universal constants; categories ⊇ core, unique,
`^[a-z][a-z0-9_]*$`; every `prefill.category` is a declared category; rubric 1–10 dims, scale `[1,5]`; `persona` ≤ 600 chars;
nav screens from the closed set, no duplicates; `strategy` from the closed set and `params` keys ⊆ that set; every duration > 0;
band `format` matches `^[a-z][a-z0-9_]*$` and **every band format has a `plan.est_minutes["touch_<format>"]` entry**, plus
`course_attempt` (and `mock` when the `mock` block exists) — every minutes value > 0;
manifest `slug`/`title`/`status` equal the `curriculum/paths.json` row (one catalog truth until the loader owns it).

**Two-sided golden test:**
- `internal/course/golden_test.go`: the loaded DSA manifest equals a literal table of v1 values, each with its `file:line`
  citation (the table above), including the dormant blocks (so changing them is also deliberate).
- Mirror tests **inside each owning package** (so unexported constants are reachable, no production import added):
  `internal/practice/store` (`AttemptTimer`, `HintTimer`, `EarlyRevealPenaltyDays`, the `last_outcome` CHECK parsed from
  its embedded `migrationsFS`), `internal/review/store` (`touchDays`, `NamePatternMaxSecs`, `isMockTouch` levels, the
  category CHECK parsed from `00002_mistakes_notifications.sql`), `internal/assessment` (`phases`, `readinessTargets`) and
  `internal/assessment/store` (the `rubric_score.dimension` CHECK from `00001_init.sql`). Drift on either side fails CI.
- Any later sprint that changes a DSA method value on purpose (m2-05 D2, m3-08 D18, m3-10 prefill, m2-04 minutes) updates
  the golden in the same PR and cites the decision.

### 3 · Freeze the item schema [X]

`curriculum/_schema/item.schema.json` and `course.schema.json` (JSON Schema 2020-12, `additionalProperties: false` on
every object) + Go types in `internal/course/item.go` with `Validate`. This freeze is the content-authoring unblock
(owner event `ev-schema-freeze`). Shape per [t1 §7.1](../research/t1-content-data-model.md#71-package-format),
[ADR-0029 §1, §4](../../adr/0029-judge-contract-and-learning-signal.md#4-learning-loop) and
[t4 §11.4 #19](../research/t4-judge-contract.md#114-amendments-to-settled-docs-every-flag):

| Field | Shape |
|---|---|
| identity | `id` (DSA `^[1-9][0-9]{0,2}$`, others `^<prefix>-[0-9]{3}$`), `course`, `week_n`, `sort_order`, `title`, `difficulty` (easy\|med\|hard), `pattern`, `role` (core\|reinforcement\|drill), `status` (live\|retired\|withdrawn) |
| `provenance` (**required**) | `origin` (original\|adapted\|licensed — `copied` is rejected), `inspired_by[]`, `license?`, `attribution?`, `authored_by` ([t1 §8](../research/t1-content-data-model.md#8-content-rights-stance)) |
| `links[]` | `{kind: leetcode\|neetcode\|other, url}` — `https://` only |
| `review` | stamps `{statement?, hints?, editorial?}` (ISO dates) |
| **`concepts[]`** | ≤ 3 refs `<course>:concept:<slug>`, primary first (ADR-0029 "concepts to revise") |
| `parts[]` | `{id, type (code\|text\|choice\|blank\|canvas; audio reserved), cadence (iterate\|final), grading (auto\|ai\|none), required, config}`; code `config`: `signature{mode (function\|class), name, params[{name,type}], returns, ops?}`, `languages[]`, `harness` (`name@v`), `checker{name, params?}`, `constraints[]`, `samples[{id, args, expected}]`, `limits{time_ms, memory_mb}`; choice `options[{id,label}]` (**no correctness flag**); blank `fields[{id,label}]`; text `max_chars` |
| `grader[]` | the steps of the item's single, non-nested `composite@1` root: `{step, kind (code\|key\|ai_rubric), inputs[], required, weight?}` (the analyzer is not authored) |
| `solution_facts` | `{complexity{time[], space[]}}` — public, served only with the solution stage |
| `revision.probes[]` | `{id, type (text\|blank\|choice), band, grading: key, key_source, criterion, timer_s?, prompt_md, config?}` — `key_source` is `public:<field>` (honor-grade) or `pack` (a private key judge holds); never an inline value |
| `assets[]` | `id@v` refs |

Sections are Markdown sidecar files, not `item.json` fields; the schema's `$comment` records the sidecar naming that
m1-09 implements. An item with no `parts` is valid (the v1 self path).

**Tests** (`internal/course/schema_test.go`, fixtures in `internal/course/testdata/`):
1. **Types ⇄ schema parity:** every JSON field of the Go types (reflection over `json` tags) is a schema property and
   vice versa, for both item and manifest.
2. **Answer-free:** walk **four trees** — the item schema, the item Go types, the manifest schema and the manifest Go
   types (as in test 1); fail on any property matching `answer*`, `correct*`, `expected*`, `hidden*`, `secret*`,
   `solution` (as a key), `key`/`keys`, `anchor*`, `exemplar*`, `must_cover`, `red_flag*`, `cases`, `tests`, `rationale`,
   except an explicit allowlist **by full JSON path** (the same name anywhere else still fails), each with its reason:
   - item `parts[].config.samples[].expected` — public sample output, shown with the statement;
   - item `solution_facts` — public complexity facts, served only with the solution stage;
   - item `revision.probes[].key_source` — names where a key lives; value must match `^(public:[a-z_.]+|pack)$`, never a key;
   - manifest `stages.solution` — the solution **stage's config** (`{label}` only; the test asserts that object has no
     other property), not solution content;
   - manifest `revision.bands[].criteria[].key` — a criterion **identifier** (e.g. `pattern_named_fast`); value must match
     `^[a-z][a-z0-9_]*$` and it names a check, never an expected value.

   Fixtures that must **fail**: an `answer` on a choice option, a `correct: true` flag, a `hidden_tests` block,
   `key_source: "B"`, and a manifest whose `stages.solution` gains a `code` property.
3. **ADR-0029 / t4 §11.4 #19 presence (the register's acceptance):** item `concepts`; manifest `mistakes.prefill`,
   `revision.bands[*].parts`, `revision.bands[*].criteria`, `revision.bands[*].mock_mode`, `revision.drills`,
   `mock.evidence_hints` exist in both the schema and the Go types.
4. **Freeze guard:** `internal/course/testdata/item.schema.v1.frozen.json` (a copy at the freeze) and a test that the live
   schema is an additive superset — no removed property, no type narrowing, no new `required`, no removed enum value.
5. **Fixtures that must pass:** `valid-self.json` (a v1-style item, no parts), `valid-code.json` (an **original**
   function-mode example — not copied from LeetCode), `valid-probes.json`; and must fail: unknown field, `origin: copied`,
   an `http://` link, 4 concepts, a malformed id.
6. **Schema validity:** validate the fixtures against the JSON Schema files with a pure-Go validator (e.g.
   `github.com/santhosh-tekuri/jsonschema/v6`), **test-only**; runtime validation stays Go types + `Validate`.

### 4 · Verify [X]

`gofmt -l .` empty · `go vet ./...` · `go test -race ./...` · `sqlc diff` (unchanged: no SQL here) · web typecheck / lint /
test / build · e2e (`-tags e2e`) on PG 18 · `docker compose up --build` on PG 18 + NATS 2.14 and a click-through (login,
Today, a problem attempt, revision, mock, coach). No handler, query or route changes, so the v1 API is byte-identical.

### 5 · Record [X]

In [`../status.md`](../status.md): the Sprint board row; M1 🔄; **content status** "item schema v1 frozen — <date>,
<commit>" and owner event `ev-schema-freeze` ✅ (the owner content track starts, ~10 h/week — [rollout §6](../rollout-plan.md#6-critical-path-parallel-tracks-owner-calendar));
the **M1 size note**: "M1 = 10 build/release sprints + 1 design sprint, above the 6–7 range, because m1-01 and m1-07 were
split (m1-09, m1-10)"; the Decisions log (layout `curriculum/courses/<slug>/`; dormant D18 params; coming-soon prefixes;
the test-only JSON Schema validator). Note for the owner: in-repo `item.json` files are authored once m1-09 has landed the
`items/` layout; until then, draft against the frozen schema.

## Acceptance criteria

- [ ] DSA manifest golden test green — both the `internal/course` literal table and the service-side mirror tests.
- [ ] The schema test proves every ADR-0029 / t4 §11.4 #19 content field exists and that no field can hold an answer (denylist + failing fixtures).
- [ ] Freeze guard in place: frozen snapshot + additive-only test.
- [ ] Compose runs `postgres:18` + `nats:2.14`; CI e2e runs on PG 18; every store integration test is green on PG 18.
- [ ] No API behaviour change (no handler, query or route touched; e2e unchanged).
- [ ] `docs/v2/status.md` records the freeze (date, commit) → `ev-schema-freeze`.

## Release

**Merge only — ships dark in `v1.6.0`**, which [m1-02](sprint-m1-02.md) tags (M1a expand; floor none). Nothing in
`v1.6.0` reads the manifest at runtime; the compose and CI changes do not ship. No infra PR.

## Definition of Done

CI green · merged to `main` (squash, conventional commit) · acceptance criteria met · statuses updated (this file +
[`../status.md`](../status.md)) · `ev-schema-freeze` recorded · notable decisions logged (ADR only if you change a
decided shape — then an ADR amendment, not a silent change).

## Risks / watch-outs

- **A field added after the freeze forces re-stamping of authored items** — hence the ADR-0029 field test and the freeze
  guard. If something is genuinely missing, add it (optional) **before** recording `ev-schema-freeze`.
- **PG 18 behaviour differences** in integration tests (the volume path, new `initdb` defaults) — run every store test on
  PG 18 locally; don't rely on the e2e job alone.
- **Dormant values mistaken for live ones:** D18 lives under `grading.params["verdict_timer@1"]` while
  `grading.strategy` is `self@1`; nothing in M1 reads `params`. Say so in the golden test's comments.
- **Two catalog sources** (`paths.json` and the manifest) until m1-09's loader owns `path` — the consistency check keeps them equal.
- **Import cycles:** keep `internal/course` stdlib-only; [m3-01](sprint-m3-01.md) adds `internal/course/canon` on top the moment this merges.

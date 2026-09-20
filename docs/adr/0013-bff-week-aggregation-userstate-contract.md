# ADR-0013 — BFF week-aggregation `userState` contract

- **Status:** Accepted
- **Date:** 2026-09-21
- **Deciders:** @sujaykumarsuman
- **Related:** [0005](0005-data-ownership-and-migrations.md), [0006](0006-authn-authz.md), [0012](0012-curriculum-content-model-and-seeding.md), [api.md](../architecture/api.md#curriculum-read)

## Context

S04 adds the Week screen, which shows curriculum content (thesis, concepts, the problem
list) **stitched to the learner's five-touch / solve state**. That per-user state is owned by
the `practice` (S05) and `review` (S06) services, which do not exist yet. Because a service
may touch only its own schema and cannot cross-join ([ADR-0005](0005-data-ownership-and-migrations.md)),
the stitching must happen in the **gateway BFF** — this is the `agg` endpoint
`GET /paths/{slug}/weeks/{n}` in [api.md](../architecture/api.md#curriculum-read).

The risk this ADR guards against: the Week screen is built now against a **placeholder** user
state, but S05/S06 must be able to populate the *same* fields later **without a frontend rewrite**.
Renaming a key or changing the `touches` array shape after the client ships is the expensive
mistake. So the shape is frozen here.

## Decision

### The endpoint is a gateway-side aggregation; curriculum content passes through unchanged

- The gateway calls curriculum's content endpoint `GET /paths/{slug}/weeks/{n}` and returns
  **every field curriculum produced** (`week`, `phase`, `path`, `concepts`, `problems`) verbatim,
  adding exactly one key: **`userState`**. Curriculum's own status/envelope (e.g. `404` for an
  unknown week) is **propagated unchanged** — the gateway does not mask it.
- The merge preserves unknown curriculum fields (it unmarshals to a generic object and re-emits),
  so the content model can evolve without a gateway change.
- The route stays **session-gated** ([ADR-0006](0006-authn-authz.md)); the account is resolved
  before curriculum is called. No user JWT is forwarded **yet** — the placeholder state needs no
  user-scoped call. When S05/S06 land, the gateway mints the internal JWT and fans out to
  practice/review, filling the same shape.

### Curriculum's week endpoint gains phase + slim path context

So the Week eyebrow ("Week N of {week_total} · Phase X <name>") renders from one call, curriculum's
week response now also returns the **`phase`** that contains the week (resolved from the phase's
`week_from..week_to` range) and a **slim `path`** object (`slug`, `title`, `problem_total`,
`week_total`). Still pure curriculum data, no user state.

### The FROZEN `userState` shape

```jsonc
"userState": {
  "week": {
    "solved": 0,                 // core problems solved
    "coreTotal": <int>,          // count of non-reinforcement problems in the week
    "byDifficulty": { "easy": <int>, "med": <int>, "hard": <int> }, // over the core problems
    "populated": false           // false while un-sourced; S05/S06 flip it to true
  },
  "problems": {                  // keyed by problem id
    "<id>": {
      "status": "available",     // locked | available | attempting | solved
      "lastOutcome": null,       // clean | rough | assisted | miss | null
      "currentTouch": 0,         // 0..5 revision touches completed
      "touches": [               // ALWAYS 5 entries, levels 1..5 = Day 1/3/7/21/45
        { "level": 1, "dueDate": null, "result": "none" },
        { "level": 2, "dueDate": null, "result": "none" },
        { "level": 3, "dueDate": null, "result": "none" },
        { "level": 4, "dueDate": null, "result": "none" },
        { "level": 5, "dueDate": null, "result": "none" }
      ]
    }
  }
}
```

Field names are **camelCase** (deliberately distinct from the snake_case curriculum content) and are
part of the contract. `touches` is always length 5. `result` is `none` until a touch is attempted
(then `pass`/`fail`; `due`/`mock` are rendering states the review service may set).

### The placeholder is honest, never faked

Until practice/review exist, every problem is emitted as `status:"available"`, `lastOutcome:null`,
`currentTouch:0`, five `result:"none"` touches; the week rollup is `solved:0`, `populated:false`.
The UI therefore renders neutral dots, an "Available" chip, and a `0/coreTotal` meter — an honest
empty state, **never** a green/solved/attempting/due signal. `populated:false` is the explicit
"un-sourced" flag so S05/S06 (and any consumer) can tell placeholder from real.

## Consequences

- ✅ The Week screen ships now; S05/S06 fill `userState` in place with **no client change** — they
  only flip `populated` and set real statuses/touches/rollup.
- ✅ Aggregation stays server-side and ADR-0005-clean: no cross-schema SQL, content untouched,
  curriculum errors preserved.
- ✅ The placeholder is unambiguous (`populated:false`) and honest — no faked progress leaks to users.
- ⚠️ `userState` is camelCase while curriculum content is snake_case — a deliberate seam (per-user vs
  content), documented here so it is not "fixed" into inconsistency later.
- ⚠️ The gateway now parses curriculum's week payload rather than blind-proxying it; a malformed
  upstream body yields a `502` (logged) instead of a passthrough. Acceptable: curriculum is a
  trusted internal service and the content is seed-controlled.
- ⚠️ `coreTotal`/`byDifficulty` count **non-reinforcement** problems only (the "core" the meter
  tracks); reinforcement problems still get a per-problem `userState` entry but do not inflate the
  rollup.

## Alternatives considered

| Option | Why not |
|--------|---------|
| **Aggregate in curriculum** | Violates ADR-0005 (curriculum would need practice/review data it cannot own or read). |
| **Return only content now; add `userState` later** | Would change the response shape when S05 lands → a client rewrite; freezing the shape now is the whole point. |
| **`touches` as a variable-length list** | The five-touch schedule is fixed (Day 1/3/7/21/45); a fixed 5-entry array with explicit `level` is simplest for the client to render dots against. |
| **Fabricate plausible progress for the demo** | Directly violates the "no faked state" rule — a green dot would be a lie until practice/review exist. |
| **`userState.problems` as an array aligned to `problems`** | A map keyed by id is robust to client-side filtering/reordering and to partial population by S05/S06. |

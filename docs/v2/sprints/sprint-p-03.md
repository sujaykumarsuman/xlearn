# Sprint p-03 — Quiz widget (AB14) + race verdict UI (AB15) → v1.15.0

> **Milestone:** P — pilot course (go-concurrency; **milestone exit**)   ·   **Track:** product
> **Prereqs:** [p-02](sprint-p-02.md) (gc manifest `preview`, multi-course UI, multi-file workspace) · [p-01](sprint-p-01.md) (`runner-v1.1.0`, `gotest@1`) · boards from [ds-p-01](sprint-ds-p-01.md) · owner event `ev-pilot-content`   ·   **Unblocks:** [m4-01](sprint-m4-01.md) (M4 starts only once v1.15.0 is cut, so P and M4 never edit `internal/judge` at the same time) · [ga-01](sprint-ga-01.md)
> **Release action:** **tag `v1.15.0`** (indicative: the next free minor at tag time, [ADR-0034 §1.6](../../adr/0034-v2-release-labelling-gating-and-rollback.md#16-indicative-tag-timeline)) **+ evalpack `v1.N.0`** (the next free evalpack minor; it adds the go-concurrency layer) · no infra PR   ·   **Calendar:** December, after the owner's pilot content is stamped (`ev-pilot-content`). Nothing waits on the owner in-session (D40). Owner events after ship, gating nothing: the re-read of the keys whose mode task 1 sets (≈ 10 min, `ev-gc-key-reconfirm`) and, unless this session grades one quiz and one race item through an already-signed-in browser session, the owner's own graded run on prod (≈ 30 min, `ev-pilot-run`)
> **Execute with:** [`../prompts/prompt-p-03.md`](../prompts/prompt-p-03.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | Key grading for the pilot quizzes: the gc modes in the one evaluator (`internal/course/keys`), `mode`/`kind` on the gc items, the reveal rule, signals, strategy rows | X | ⬜ |
| 2 | Parts/results registries, quiz widget, Workspace-Quiz, Touch recall probes, arena quiz study mode (AB14) | X | ⬜ |
| 3 | Race verdict + goroutine-dump result view, multi-file diagnostics (AB15) | X | ⬜ |
| 4 | Verify P-01's `gotest@1` lints on the real gc packs; add the quiz-key and mode/key-shape lints; authoring guide | X | ⬜ |
| 5 | Tests + compose rehearsal (synthetic fixtures in CI; the real pack mounted locally only; the code leg on Linux) | X | ⬜ |
| 6 | Merge + tag v1.15.0 (release checklist) | X | ⬜ |
| 7 | Pilot pack: go-race gates in a jail-capable CI lane + the go-concurrency layer → evalpack `v1.N.0` | E | ⬜ |
| 8 | Prod verify by looking (gc items `ok`, 0 `spec_mismatch`, no gc trace outside the cohort) | X | ⬜ |
| 9 | P exit recorded (milestone, tag → floor, streams, content status, decisions) | X | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row + any
> milestone). Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] **Pilot content authored and stamped** (owner event `ev-pilot-content`, 10–20 owner h):
  - the ~10 public halves `curriculum/courses/go-concurrency/items/gc-0NN/` are merged on xlearn `main` with the `content` CI job green (statements, starter modules, `visible_test.go`, quiz parts, `review` stamps);
  - the private halves are merged on `xlearn-evalpack` `main` with review stamps: hidden `*_test.go`, go-race params, mutants under `submissions/wrong/`, and quiz keys the owner confirmed ([t1 §7.3](../research/t1-content-data-model.md#73-where-ai-may-help): AI proposes keys, the owner confirms);
  - if the frozen item schema has no mode selector ([m1-01](sprint-m1-01.md) task 3: `choice.options[{id,label}]`, `blank.fields[{id,label}]`), the gc quiz parts were authored without one. That's expected: task 1 adds `choice.mode` / `blank.fields[].kind`, and task 7 refreshes the affected items' hashes (the owner re-reads those keys after ship, `ev-gc-key-reconfirm`).
- [ ] **No evalpack tag has shipped gc items yet.** The gc private halves reached `xlearn-evalpack` `main` at `ev-pilot-content`, before this sprint's go-race gates existed. Any evalpack tag cut from `main` since then (a DSA wave minor, a TL patch) would carry stamped gc items that never passed them. Check `git -C ../xlearn-evalpack tag --contains <the gc merge commit>`. For each such tag, its `packcheck` build job summary (ids and counts only) must list **no gc item as included**. If one does, stop and report: that tag needs a patch that leaves the gc items out before this sprint's pack. Until task 7's gates pass, no evalpack tag may carry gc items. Tell any peer session that cuts an evalpack tag meanwhile; the owner can also leave the gc key stamps unset until then.
- [ ] **AB14 F11 (arena quiz) as frozen.** DS-P-01's PR lists it under "Decisions to confirm": F11 draws "answers behind a spoiler confirm" (D17), but [t4 §4.3](../research/t4-judge-contract.md#43-archetype-c-quiz--key) and [ADR-0029 §2](../../adr/0029-judge-contract-and-learning-signal.md#2-the-contract) say keys and rationale appear only after a counted conclusion. The ds-p-01 merge froze F11 as drawn; a later follow-up design PR may have changed it. Read F11 on the board on `main`, and record the resolution in the Decisions log before task 2. Not a wait (D40): the frozen board is the answer, and it decides whether this session amends ADR-0029 §2 (task 2).
- [ ] **[p-02](sprint-p-02.md) merged**: the gc manifest (`status: preview`, `id_prefix: gc`, the recall-quiz L1–3 band, `mistakes.prefill` rows for `misconception:*`), `preview` hidden outside the cohort, `COURSE_STATUS_OVERRIDE` in the status.md flag inventory, AB02/AB05 at full fidelity and the multi-file workspace.
- [ ] **[p-01](sprint-p-01.md) done**: `runner-v1.1.0` (go-race + `gotest@1`) is deployed dark by image automation, and P-01's judge `gotest@1` mapping and lints (Σ TL ≤ 40 CPU-s, `mem_mb` ≥ 2 × the reference peak) are on `main`.
- [ ] **M3 shipped**: v1.14.0 is live ([m3-13](sprint-m3-13.md)), `JUDGE_BASE_URL` is set on the gateway, and judge features are cohort-only (T-3).
- [ ] **AB14 and AB15 frozen**: the [ds-p-01](sprint-ds-p-01.md) PR is merged (the merge is the freeze), and status.md's artboard rows read "frozen (PR #, date)" (set by DS-P-01, or by P-01's close-out).
- [ ] **PRD Q5 = go-concurrency** is recorded in status.md (`ev-q5`, DS-P-01 task 1). **If Q5 recorded the SQL fallback, stop and report:** this plan is written for go-concurrency, AB15 would be the SQL explorer, and P-01…P-03 need re-planning.
- [ ] **The evalpack pull PAT is not about to expire**: the expiry date in status.md (`ev-pat-expiry`) is more than 14 days away. An expired PAT means the pack bump can't be pulled; the old judge pod keeps serving, but the pilot never becomes gradable.
- [ ] **Parallel sessions**: the next free xlearn minor and the next free evalpack minor are known (`git ls-remote --tags origin` in both repos, `gh pr list`, `git worktree list`, ListAgents). **No peer PR edits `internal/judge`** (M4 waits for this tag; if an M4 PR is open, coordinate before touching judge).

## Goal

**Close P.** Finish the pilot's two capabilities and ship the course, still `preview`:
- the **A7 quiz** (archetype C) end to end: the key modes that the go-concurrency quizzes use, in the **one** key evaluator judge's `key@1` already calls, graded by `weighted_gate@1` in every counted context, with the key and rationale revealed **only after a counted conclusion**;
- the **race verdict view**: RACE / DEADLOCK / LEAK verdicts with a goroutine dump that shows learner-package frames only, on top of P-02's multi-file workspace;
- the **pilot pack** (hidden tests, go-race parameters, quiz keys) as an evalpack minor, gated by the go-race detection checks that [t1 §7.2](../research/t1-content-data-model.md#72-ci-validation) deferred until a course picked go-race.

Then **tag v1.15.0**, ship the evalpack minor, and prove it on prod by looking. The P exit criterion ([rollout §3](../rollout-plan.md#3-milestone-map)): the course ships as **manifest + content, with widget/profile code only**, and the manifest stays `preview` until GA.

## Scope

**In**
- judge + `internal/course/keys`: any gc quiz mode ([t4 §4.3](../research/t4-judge-contract.md#43-archetype-c-quiz--key)) that M3-06 didn't land, added to the one evaluator `internal/course/keys` that judge's `key@1` calls (never a second key package). Also: gc-shaped golden fixtures; M3's `concluded` reveal path exercised with real quiz shapes, adding rationale and the misconception label if they're missing; the learner DTO fields for `gotest@1` results (reason codes; the SPA owns the text); and the denylist tests extended to both.
- curriculum / course package: at most two **optional, additive** config fields (`choice.mode`, `blank.fields[].kind`) if the frozen item schema lacks them, with `contract_hash` unchanged for every item that doesn't set them; then set them on the gc quiz parts and probes that need them.
- practice (only if missing, additive): a failed recall touch carries its key probes' `misconception:*` as `mistake_hint`, the same way judged re-solves do.
- web: the part and result-view registries of [t4 §5.5](../research/t4-judge-contract.md#55-widget-and-result-view-contracts-ts) (created here if M3/P-02 didn't create them); the `choice` and `blank` widgets; the Workspace-Quiz variant of the one Workspace shell (AB14); recall probes inside Touch for gc L1–3; **arena quiz study mode** (AB14 F11, as frozen); the `key` result view; the `gotest@1` race/deadlock/leak result view with the goroutine-dump panel (AB15); and `onFocusPart` with a file/line location.
- `cmd/packlint`, `cmd/contentlint` and the public `content` CI job: verify P-01's `gotest@1` lints on the real gc packs; add the quiz-key and mode/key-shape rules; update `docs/v2/authoring.md`.
- xlearn-evalpack: the go-race gates as a jail-capable `packcheck.yml` job, the go-concurrency course layer, the gc `accepts_contract_hashes` refresh, and the `v1.N.0` tag.
- Tag v1.15.0; prod verification; the P exit record; the post-ship owner events (the key re-read, and the owner's graded run unless this session does it through an already-signed-in browser session).

**Out (later sprints)**
- **Flipping go-concurrency to `active`** and removing the cohort gate from defaults: GA ([ga-01](sprint-ga-01.md)).
- **"I meant this" claims** for honor keys and AI-provisional grades: M4 ([m4-04](sprint-m4-04.md)). The gc quiz keys come from the pack (`trust=checked`), so they never need a claim.
- **Mock-context rendering** of quiz items (aggregate only unless concluded) in the interviewer: M6a ([m6a-05](sprint-m6a-05.md), the Mock-v2 UI). This sprint only guarantees the DTO rule and its test.
- **Other archetype-C courses** (sql, lld-ood and system-design recall quizzes): they reuse this widget with no new code once their manifests exist.
- **More gc content** after the pilot: D6-style waves on the evalpack `v1.x` minors (owner content track).
- **SQL** (`sql-pg`): only if PRD Q5 flips, which re-plans P.

## Tasks

### 1 · Key grading for the pilot quizzes [X]

M3 built judge's `key@1` grader (inline, counted contexts only, once per context; [m3-06](sprint-m3-06.md) task 4) and `weighted_gate@1` ([m3-08](sprint-m3-08.md)). By design there is **one key evaluator**, `internal/course/keys` ([m2-01](sprint-m2-01.md) task 3; [t4 §12](../research/t4-judge-contract.md#12-alternatives-considered) "one evaluator", recorded in M3-06's ADR). Judge's `key@1` calls it, and so do practice's DSA probe lock-ins. M3-06 was scoped to add every [t4 §4.3](../research/t4-judge-contract.md#43-archetype-c-quiz--key) mode and both signals there, but it was allowed to move the quiz modes to a follow-up ("record the split in the decisions log"). This task makes sure the modes the gc items use exist **in that one evaluator**, and that the public config says which mode a part uses.

- **Read the split first.** status.md's Decisions log (M3-06's entry) and M3-06's ADR say which modes landed in `internal/course/keys`. Then inventory the modes the stamped gc items and probes use (their `parts[]` / `revision.probes[]` configs plus their pack keys). P-02's scaffolds imply this set: gc-008 `choice.single`/`choice.multi`, gc-009 `blank.predict_output`, gc-010 `choice.single`/`choice.order`, and the code items' recall probes (`choice.single`, `blank.predict_output`):

  | Mode | Value | Pack key | Scoring |
  |---|---|---|---|
  | `choice.single` | `{selected:[id]}` | `answer`, `distractors{id: misconception}`, `rationale_md` | exact |
  | `choice.multi` | `{selected:[ids]}` | `answer[]`, `distractors`, `rationale_md` | `all_or_nothing` (default) \| `right_minus_wrong` (≥ 0) \| `per_option` |
  | `choice.order` | `{order:[ids]}` | `order[]`, `rationale_md` | `exact` (default) \| `adjacent_pairs` \| `kendall` |
  | `blank.predict_output` | `{raw}` | `accept[]`, `rationale_md` | exact after `trim_lines` + `crlf`; **no typo tolerance** |
  | `blank.text` / `blank.numeric` | per t4 §4.3 | per t4 §4.3 | only if an item uses them |

  - **A mode that landed in M3-06:** add gc-shaped golden fixtures only (synthetic, test-only ids).
  - **A missing mode:** implement it **in `internal/course/keys`** (normalizers beside the existing text and `big_o` ones), in its closed registry (no `init()` self-registration, [t4 §5.1](../research/t4-judge-contract.md#51-registries)), reached through judge's `key@1`. **Never create a second key package under `internal/judge`**: the DSA probe path and judge must grade with the same code.
  - Golden fixtures per mode: correct, each distractor, empty, over-selected and malformed values. A new mode costs 1–3 days and needs no ADR ([t4 §5.7](../research/t4-judge-contract.md#57-cost-of-adding-capability-inferred)).
- **Public config: the mode selector.** The frozen M1 schema has `choice.options[{id,label}]` and `blank.fields[{id,label}]` ([M1-01 task 3](sprint-m1-01.md)). Without a mode, a `multi`, `order` or `predict_output` question would render and grade as `single`/`text`. If the schema has no mode selector, add **`choice.mode`** (`single|multi|order`, default `single`) and **`blank.fields[].kind`** (`text|numeric|big_o|predict_output`, default `text`) as **optional** properties, on `parts[].config` and on `revision.probes[].config`. The files are `curriculum/_schema/item.schema.json` and `internal/course/item.go` (`Validate`), and the rules are:
  - the M1 freeze-guard test against `internal/course/testdata/item.schema.v1.frozen.json` stays green (an additive superset: no new `required`, no removed enum value; **never edit the frozen copy**);
  - the answer-free schema test stays green (no correctness flag, ever);
  - **`contract_hash` must stay byte-identical for every item that doesn't set the new fields** (every DSA item). The canonical struct in `internal/course/canon` marshals them only when set (omit-empty). Assert it with M3-01's golden vectors and a test that re-hashes every live DSA item and compares with the embedded values. A hash drift would turn stamped DSA packs into `spec_mismatch`.
  - When set, the new fields **are** part of `contract_hash` (a mode change needs a new key anyway, [t1 §3.4](../research/t1-content-data-model.md#34-versioning-and-pinning)).
  - **Then set them on the gc content:** `choice.mode` on every multi/order part and probe, and `kind: predict_output` on every predict-output field (a `single`/`text` default needs nothing). Re-run the `content` job. Those items' `contract_hash` changes on purpose, so task 7 refreshes their `accepts_contract_hashes` in the pack (`packlint check` names them); the owner re-reads their keys after ship (`ev-gc-key-reconfirm`). If the schema already had a mode selector, check that the owner set it on every gc part that needs it.
- **Reveal rule in `Present()`** ([t4 §5.2](../research/t4-judge-contract.md#52-grader-contract-go), [ADR-0029 §2](../../adr/0029-judge-contract-and-learning-signal.md#2-the-contract)). M3 already gates `key` marks and `answer_reveal` on `concluded=true`. The gateway passes it only when practice's state says the context has concluded ([m3-14](sprint-m3-14.md), [m3-09](sprint-m3-09.md)); in M3 a fixture was the only thing that exercised this path. Exercise it with real quiz shapes and add what the quiz needs **on that same path** (don't invent a second one):
  - **Before the counted context concludes:** at most "answered / not answered" per question. No per-question correctness, no `answer`, no `accept[]`, no `rationale_md`, no distractor tags.
  - **After the counted conclusion** (course or touch, including a Miss): per-question marks, the key (answer ids / accepted answers), `rationale_md` (rendered by the shared Markdown renderer from [m1-06](sprint-m1-06.md), no raw HTML), and the chosen distractor's misconception label. Add `rationale_md` and the label to the allowlisted DTO if M3 lacks them.
  - **Mock** (unconcluded): the aggregate only. **Arena and Run:** the key never runs; the result is `skipped(not_evaluated_here)`. The only arena exception is the D17 spoiler, and only if the frozen AB14 F11 draws it (entry gate; task 2 specifies the path).
- **Signals** ([t4 §6.3](../research/t4-judge-contract.md#63-mistake-pre-fill-two-tiers-data-only)). M3-06's `key@1` already emits `key_wrong:<field>` and `misconception:<category>` (deterministic, from the pack's `distractors` map) in `evaluation_completed` only. Don't re-specify them: prove each gc mode emits them with synthetic fixtures, and add emission only for a mode this task adds. The gc manifest's `mistakes.prefill` maps `misconception:*` as **strong**, so review pre-fills the category ([m3-10](sprint-m3-10.md)). Mistakes open **only on revisable items** (I8). Every gc quiz is `role: drill` with `revision.drills false` (P-02), so in the pilot the pre-fill fires only from a **failed recall touch on a core code item**. Check that `touch_concluded` carries the probes' `mistake_hint`: [m3-08](sprint-m3-08.md) added it "for judged re-solves". If it's missing for key-probe touches, derive it the same way from the touch's key evaluations (the same `mistakes.prefill` mapping; the field already exists on the event, so no migration and no new subject). If that turns out to be more than a small additive change, record a follow-up instead and cover the acceptance bullet with the synthetic test.
- **Strategy check.** `weighted_gate@1` with the quiz parameters `clean_pct 1.0, pass_pct 0.8` ([t4 §6.2](../research/t4-judge-contract.md#62-strategies-a-closed-go-set-the-manifest-picks-one-and-sets-thresholds)). Add golden rows for a synthetic 6-question quiz on the gc manifest's timers (attempt 45:00):

  | Facts | Grade |
  |---|---|
  | 6/6, `final` before `deadline_at`, no hint | clean |
  | 5/6 (0.83), `final` before `deadline_at` | rough |
  | 4/6 (0.67) | miss |
  | give-up, or no `final` by the deadline (D15; the ticker concludes, unanswered = wrong) | miss |
  | `final` twice in one context | the second is refused (once per context) |
  | `final` after `deadline_at + 2 s` | 409 `attempt_expired` at the gateway ([m3-09](sprint-m3-09.md)); the ticker concludes Miss (D15) |

  **"Late" for a quiz.** The gc manifest gives `weighted_gate@1` only `{clean_pct, pass_pct}` (P-02); `clean_within_s 1200` belongs to `verdict_timer@1`. M3-08 says `weighted_gate@1` keeps "the same hint/coach/late ceilings" but names no quiz clean window. Read M3-08's `weighted_gate@1` fixture table. **If it reads a clean-window param**, name that manifest field, set it in the gc manifest's `weighted_gate@1` params (data only), and add the row "6/6 with the `final` after that window but ≤ `deadline_at` → rough (I2)". **If it has none**, there's no late band for quizzes (a `final` is either on time or refused). Drop the row and record why in the Decisions log. If the frozen AB14 F9 shows a graded pass after the deadline, D15 wins.
- **Trust.** Pack keys → `trust=checked`. A `public:*` key source → `honor` ([t1 §7.1](../research/t1-content-data-model.md#71-package-format)). The gc code items stay `honor` (in-process tests, task 3).

### 2 · Registries + quiz widget + Workspace-Quiz + recall probes in Touch + arena study mode (AB14) [X]

Build to the frozen `design-system/screens/v2/AB14-*.html` with `theme.css` verbatim (no Tailwind; the dark theme).

- **Registries first** ([t4 §5.5](../research/t4-judge-contract.md#55-widget-and-result-view-contracts-ts): `PartWidget<C, V>`, `ResultView`). No upstream plan creates them. M3-11 built the code editor as `web/src/screens/workspace/CodeEditor.tsx` (the lazy chunk), M3-12 built the dock as `web/src/components/judge/ResultsDock.tsx`, and P-02 task 5 extended the code widget for `gotest@1` (it says "`web/src/parts/`, the part registry"). Check what's on `main`:
  - if `web/src/parts/registry.ts` and/or `web/src/results/registry.ts` exist, add to them;
  - if not, create them: closed maps keyed by part type and by result `kind` (no self-registration). Register M3's `CodeEditor` (with P-02's multi-file variant) as the `code` widget and the dock's existing code rendering as the `code` result view. `ResultsDock.tsx` then renders each step's result through `web/src/results/registry.ts`. M3's and P-02's vitests stay green **unchanged**, and the lazy editor chunk stays lazy.
  - Record in the Decisions log which case you found.
- **Part widgets**: `web/src/parts/choice/` and `web/src/parts/blank/`, registered in `web/src/parts/registry.ts` beside `code`; the mode comes from `choice.mode` / `blank.fields[].kind` (task 1).
  - Contract: `empty`, `validate` (a UX mirror; the server is authoritative), `serialize` (the request size is checked **before** sending), `deserialize`, draft `{debounceMs: 5000, timeoutMs: 25000}`, and the modes `edit | readOnly | review | locked`.
  - **Accessibility** ([t4 §4.3](../research/t4-judge-contract.md#43-archetype-c-quiz--key)): native radio and checkbox groups in `fieldset`/`legend`; ordering by move buttons and **Alt+Arrow**; numbers with `inputmode=decimal`; `labelId`/`describedById` wired; focus order as the board's notes say.
  - **Options are shuffled per context**: a deterministic permutation seeded by the context id, so a reload keeps the order. The server grades by ids, so it never depends on the order. An `order` question never starts in its answer order.
  - If M3 created `cmd/partsgen` and `registry.gen.json`, the vitest drift check stays green with the new widgets. If it didn't, don't create it here; note it in the Decisions log.
- **Workspace-Quiz** is the A7 variant of the **one Workspace shell** ([t4 §8](../research/t4-judge-contract.md#8-learner-flows-and-v2-artboards-needed)), not a new route. `web/src/screens/Workspace.tsx` ([m3-11](sprint-m3-11.md)) picks the archetype from the item's `grader[]` (key-only → quiz). Frames per AB14:
  - question cards (each question is one `final` part), the **"4 / 6 answered"** counter, the server timer ring (no pause);
  - **Submit answers** (the `final` action: one submission with every part) with the unanswered-questions confirm as AB14 words it;
  - the settling phase ("Recording your result…") until practice reports the conclusion;
  - **per-question marks with the key and rationale after conclusion**, and the misconception label on a chosen distractor;
  - the timeout frame: no `final` by the limit is a Miss (D15), with the board's warning before it;
  - < 1024 px: the Statement / Work / Results tabs.
- **Result view**: `web/src/results/key/` registered as `kind: 'key'` in `web/src/results/registry.ts`, rendered by `ResultsDock.tsx` (its Feedback tab already shows `key` marks and `answer_reveal` "if the DTO carries them", [m3-12](sprint-m3-12.md)). It renders only what the DTO carries, so the reveal rule stays server-side.
- **Recall probes in Touch.** The gc L1–3 band is a recall quiz ([t4 §6.6](../research/t4-judge-contract.md#66-touch-formats-and-pass-criteria-per-course)): spot-the-race / predict-output from pack keys, ~5 min, `checked`, pass = s ≥ `pass_pct` ∧ the required questions correct. `web/src/screens/Touch.tsx` ([m2-04](sprint-m2-04.md)) renders the band's probe parts through the same registry, with **lock-in** per probe where the band sets `timer_s`. The result shows the server criteria and "advances to Day N" / "resets to Day 1".
- **Arena quiz study mode (AB14 F11).** [m3-12](sprint-m3-12.md) built the generic study mode (editable parts, a revealable reference, **Mark studied**) and explicitly handed quiz study mode to this sprint. It's a deliverable here:
  - `web/src/screens/Arena.tsx` renders a key-graded item's `choice`/`blank` parts through the parts registry in `edit` mode. Nothing is submitted for grading (`key` never runs in the arena; any key step shows the dock's `not_evaluated_here` copy). The screen also has the "Nothing here counts toward your course." banner and **Mark studied** (M3-12's `POST /api/problems/{id}/arena/studied`, `source=manual`).
  - **If the frozen F11 keeps "no key in the arena"** (t4 §4.3 / ADR-0029 §2 unchanged): F11 shows public content only (the statement, concept links and the widgets) and no spoiler control. The denylist stays as written: no key, `accept[]`, rationale or distractor tag on any arena response.
  - **If the frozen F11 draws the D17 spoiler** (as DS-P-01 drew it): the spoiler confirm posts M3-09's arena reveal route (`POST /api/problems/{id}/arena/reveal`, which records `arena_revealed_at` once and never caps). Only after practice has recorded the reveal does the gateway ask judge's `Present()` for the key, under a new **`arena_revealed`** visibility (judge never takes it from the client). The widgets switch to `review` and show the key and rationale, but **no per-question marks** (nothing was graded). Amend [ADR-0029 §2](../../adr/0029-judge-contract-and-learning-signal.md#2-the-contract) ("keys and rationale only after a counted conclusion **or a recorded arena reveal (D17)**") with a dated amendment note in the ADR, plus a Decisions-log row. Change the denylist accordingly: arena before the reveal → no key; after it → key + rationale only.
  - Vitests for F11 in whichever variant applies, plus a gc-shaped fixture proving Mark studied changes only the arena marker.
- **Client.** Submit `final` through `web/src/lib/judge.ts` (M3-11); keep the 25 s fetch timeout and TanStack Query polling.

### 3 · Race verdict + goroutine-dump result view (AB15) [X]

P-01 built the go-race profile and `gotest@1` (test2json, `-count=1` on Run, per-test deadline, goroutine dump, the RACE / DEADLOCK / LEAK classes) and judge's result mapping. This task presents those results to the learner, per [t3 §5.8](../research/t3-sandbox.md#58-goroutine-dump-go-race) and the frozen `design-system/screens/v2/AB15-*.html`.

- **judge DTO** (extend the M3 allowlist, [m3-09](sprint-m3-09.md) / [m3-14](sprint-m3-14.md)):
  - **Run** (the learner's own `visible_test.go`): per declared test `{name, status, class}`, the bounded output the runner returns, and learner frames for RACE/DEADLOCK. Run shows full detail (R-JG1).
  - **Submit** (hidden tests): hidden passed/total, the **first failure class** only, and for DEADLOCK/RACE the **learner-package frames**: `{func, file, line}` limited to the item's editable files, with package-relative paths, **no argument words, no PC offsets, no hidden-test frames, no hidden test names, no `TestEvent.Output`, no per-test timings**. Frames outside learner packages collapse to a count.
  - **The compile-leak fixture** ([t4 §5.6 #15](../research/t4-judge-contract.md#56-ci-lints-public-ci-and-again-at-judge-start)): a hidden test that calls a missing learner symbol yields `class=CE` with the fixed hidden-API **reason code** and nothing else (no hidden file name or line). P-01 added the fixture for `gotest@1`; keep it green.
  - **Reason codes, not prose.** The DTO carries `graded_by=auto`, `trust=honor`, the class and reason codes (hidden-API CE, `runner_throttled`, …). **The SPA owns every string.** M3-06's judge-side CE sentence isn't learner copy.
  - Extend the per-endpoint **denylist test** to every new field.
- **Result view**: extend the `code` result view for `harness: gotest@1`. That's M3-12's code rendering in `web/src/components/judge/ResultsDock.tsx`, now behind the `code` entry of `web/src/results/registry.ts` (task 2). Put the gotest pieces under `web/src/results/code/gotest/` (or beside the dock's code view if task 2 found no registry and left it in place):
  - verdict banners for **RACE**, **DEADLOCK**, **TLE** and **LEAK**, precedence RACE > DEADLOCK > TLE > LEAK > RE > WA ([t3 §6.2](../research/t3-sandbox.md#62-profile-table-limits-are-proposed-a8-tunes-them));
  - the per-test list on Run; hidden passed/total and the first failure class on Submit;
  - a **goroutine-dump panel**: goroutines grouped as the board shows, learner frames only (`pkg.Func  file.go:42`). Clicking a frame jumps to that file tab at that line (the navigation contract below);
  - the provenance chip, as AB15 draws it: **"honor · tests run in-process"** while solving (F1) and **"Judge · honor"** on the concluded grade (F13), with F13's tooltip ("The tests run inside your program, so this counts on your honour. Judge-checked % leaves it out."). So the result never reads as judge-checked (P7 excludes honor; [t4 §4.1](../research/t4-judge-contract.md#41-archetype-a-code-ide) go-concurrency row). `auto · honor` is the data pair (`graded_by`·`trust`), never UI copy.
- **Copy: the frozen AB15 wins for every string.** Use the board's text as drawn: F3/F5 race, F6 deadlock, F7 TLE ("Too slow · a test hit its deadline, but no goroutine was stuck in your code…"), F8 leak, F9 WA, F10 CE (the gc hidden-API variant is "Hidden tests don't compile against your code. Check the exported names in the statement. (Not counted.)"), F11 REJECTED, F12 inconclusive ("We couldn't get a reliable result (the server was busy). Not counted. [Submit again]") and the F13 chip. The SPA picks the AB15 text for `gotest@1` items and keeps AB08's text for DSA ("Check the required signature"). AB08 copy applies only to states AB15 doesn't draw.
- **Navigation contract.** t4 §5.5's `onFocusPart(id: string)` carries no location. Extend it to `onFocusPart(id: string, loc?: {file: string; line: number})`, and make P-02's multi-file widget accept a focus request: open the `file` tab and scroll to and highlight `line`. A read-only file opens without an edit affordance. Change the TS contract in one place and record it in the Decisions log (a UI-contract detail, no ADR).
- **Multi-file diagnostics**: CE and vet positions map to the right file tab (P-02's editable list) through the same `loc`; positions in read-only files (`visible_test.go`) show without an edit affordance.
- **Contexts**: the same view renders in a course attempt, in the Touch re-solve (gc L4–5, `go-race`, honor) and in the arena (arena submits never count, D10).

### 4 · `packlint` + content CI for the pilot [X]

Much of this shipped in [p-01](sprint-p-01.md) (tasks 2–4). On the `gotest@1` side, `cmd/packlint` checks `tests[]` + go-race params, that hidden tests compile only against `api[]`, no `func TestMain`, Σ TL ≤ 40 CPU-s, `mem_mb` ≥ 2 × the reference peak and the #15 compile-leak fixture. The content CI runs the reference against `visible_test.go` in the runner image with `--network none`, and there's the `gotest@1` `Present()` denylist. **Verify those are green on the real gc packs, and add only what's missing.**

- **Verify P-01's `gotest@1` lints** on the real content: run `packlint check` locally against `../xlearn-evalpack` (output: pass/fail and ids only) and check that the `content` job is green on `main`. Fix only what the real packs expose. If P-01 left anything out, add it: hidden `*_test.go` listed in `pack.json` with the starter module's package clause; per-test deadline ≥ 10 × the published go-race baseline; every mutant in `submissions/wrong/` declares its class.
- **`cmd/packlint`, quiz keys** ([m3-01](sprint-m3-01.md)), new here:
  - every `key`-graded part and every `key_source: pack` probe has a key; `answer ⊆` the part's option ids; `accept[]` is non-empty; each distractor's misconception category is in the gc manifest's categories; each question has `rationale_md`; key-graded parts are `cadence: final`.
  - **the key's shape matches the part's mode/kind** (task 1): `choice.single` → exactly one `answer`, so a key with more than one answer on a part without `mode: multi` fails; `choice.multi` → `answer[]`; `choice.order` → `order[]`, a permutation of every option id, which needs `mode: order`; `blank.predict_output` → `accept[]` with only the `trim_lines`/`crlf` normalizers and no `typo`, which needs `kind: predict_output`; `blank.text`/`big_o` → `accept[]`; `blank.numeric` → `value, unit, tol`.
- **`cmd/contentlint`** ([m1-09](sprint-m1-09.md), extended by [m3-01](sprint-m3-01.md)) and the public `content` job: quiz parts and probe configs decode strictly, option and field ids are unique, and the optional `mode`/`kind` fields validate. **Verify** that M3-01's label-edit flag covers `choice` option labels on parts and probes, and that P-01's gate compiles the gc starters from `go.mod.tmpl`. Extend either only if it doesn't.
- **`docs/v2/authoring.md`** (M3-01's guide): the gc pack format (hidden tests, go-race params, mutants), the quiz key format with `mode`/`kind`, and "AI proposes keys, the owner confirms".
- **No leak**: the pre-push fingerprint hook covers `*_test.go` bodies and `rationale_md` shingles from `../xlearn-evalpack`. Nothing from the private repo is copied here (AGENT.md rule).

### 5 · Tests + compose rehearsal [X]

- **Public CI** (no secrets): key-mode and presenter tests on **synthetic** items and a synthetic fixture pack under `internal/judge/testdata/` (test-only item ids, never real gc ids or keys); strategy golden rows; the denylist tests (arena per the F11 resolution); the widget vitests for every AB14/AB15 frame state (F11 included); M3's and P-02's vitests unchanged after the registry move; P-02's `preview` tests still green; the `contract_hash` no-drift test; the synthetic `misconception:*` → pre-fill test for a failed recall touch on a revisable item, and none for a drill.
- **Compose rehearsal with the real pack, locally only.** In `../xlearn-evalpack` run `make build`, then start compose with `EVALPACK_DIR=../xlearn-evalpack/build` (the `x-evalpack-mount` anchor, [m3-02](sprint-m3-02.md) task 7; [t1 §3.7](../research/t1-content-data-model.md#37-local-dev-and-public-ci)). **Never copy it into this repo.**

  **Where each leg runs.** The runner jail can't run on macOS ([m3-03](sprint-m3-03.md), [m3-06](sprint-m3-06.md): the real-runner lane is Linux-only). On the owner's Mac, compose runs legs 1, 3 and 4: key grading is inline in judge and needs no runner. Leg 2 runs either in the arm64 multipass VM with the same compose stack (`multipass launch 24.04 --cpus 4 --memory 8G`; seccomp LOG on arm64, so it's indicative only), or it's covered by CI's jail-capable e2e on the synthetic `gotest@1` fixture plus task 7's go-race gates. Record which.

  **Making a touch due.** The recall legs need a gc L1–3 touch that's due now. Conclude a core gc code item in a course attempt first; on macOS a give-up → Miss needs no runner. Then pull its L1 due date forward with a dev-only SQL update on the local DB (the "reset via SQL" pattern), for example `docker compose -p xlearn-local exec -T postgres psql -U xlearn -d xlearndb -c "update review.revision_item set due_date = now() - interval '1 minute' where account_id = '<owner id>' and touch_level = 1 and problem_id = 'gc-00N'"`. Those are v1's column names; use whatever M1/M2 left, and never run this against prod.

  As an owner/tester cohort account:
  1. **a gc quiz item (a drill):** answer one question with a distractor → the grade the strategy table gives. After conclusion the marks, key and rationale show. **No mistake entry opens**, and the card reads "Drill · not scheduled for revision" (I8: `role: drill`, `revision.drills false`).
  2. **a gc code item** (Linux lane only): a mutant with a data race → RACE with learner frames; a deadlock mutant → DEADLOCK with the dump panel; a leak mutant → LEAK; the reference → pass, Clean with the **"Judge · honor"** chip.
  3. **a gc L1–3 recall touch on that core code item:** (a) lock in a distractor on a pack-keyed probe → "Not passed — resets to Day 1", and the mistake entry opens with the `misconception:*` category pre-filled (strong). This is the positive pre-fill check (task 1, Signals). (b) Make the touch due again and pass it, with lock-in per probe → "Passed — advances to Day N".
  4. a non-cohort account and an anonymous profile → no gc trace (P-02's rule, re-checked).

  Paste **only** pass/fail lines and grades into the public PR. Never keys, hidden test names or dump frames from the real pack.
- **P exit check:** `git grep -n 'go-concurrency\|"gc"' -- 'internal/**/*.go' 'web/src/**' ':!**/*_test.go' ':!**/*.test.*' ':!**/testdata/**'` finds nothing. The course is data under `curriculum/courses/go-concurrency/`; the code is capabilities ([ADR-0026 §1](../../adr/0026-per-course-extensibility-model.md#1-a-course-is-data-a-capability-is-code)).
- **Gates:** `gofmt -l .` empty · `go vet ./...` · `go test -race ./...` · `sqlc diff` clean · the contract-header lint silent (no contract migration) · web typecheck / lint / test / build · e2e (`-tags e2e`).

### 6 · Merge + tag v1.15.0 [X]

- One xlearn PR (or a short, serialized series): branch `feat/p-03-quiz-race-ui`, conventional commits with the attribution lines, CI green, squash-merge.
- Run the **release checklist** (below). GitHub release title: **`v1.15.0 — v2 build · P pilot (preview)`**.
- Release notes: the second course (go-concurrency, `preview` = owner and testers only), the quiz widget, the race/deadlock/leak view, `runner-v1.1.0` (go-race) already live, and the behaviour that the quiz key and rationale appear only after conclusion.
- **Rollback floor after:** unchanged from v1.14.0's recorded floor (**1.13.0**, [rollout §7](../rollout-plan.md#7-indicative-tag-timeline)). P adds only the constraint that consumers stay **≥ 1.7.0** for `path_slug ≠ dsa` events, which 1.13.0 satisfies ([ADR-0034 §4.4](../../adr/0034-v2-release-labelling-gating-and-rollback.md#44-reversibility-by-step) "P pilot" row). Not a contract, erase or GA tag, so no snapshot.

### 7 · Pilot pack → evalpack `v1.N.0` [E]

In `../xlearn-evalpack` (private; [t1 §3.3](../research/t1-content-data-model.md#33-private-eval-pack), [ADR-0034 §1.5](../../adr/0034-v2-release-labelling-gating-and-rollback.md#15-other-release-streams)):
- **Contract refresh first.** Task 1 set `mode`/`kind` on some gc parts and probes, which changed their `contract_hash`. `packlint check` names the stale items: add the new hashes to their `accepts_contract_hashes`. Nothing waits for the owner (D40): the owner-confirmed key stamps stay as they are (the session never sets or edits a stamp), and task 4's key-shape lint proves each key's shape fits the mode task 1 set. The owner re-reads exactly those items' keys after ship (owner event `ev-gc-key-reconfirm`, about 10 min; the mode now fixes how the key is read); a correction is a follow-up pack PR and patch tag.
- **go-race CI gates** (deferred by t1 §7.2 until a course picked go-race). **Where they run:** detection rates need the real go-race profile on amd64 Linux, and the runner jail can't run on macOS ([m3-03](sprint-m3-03.md)). Add a `gorace` job to the private repo's `.github/workflows/packcheck.yml` ([m3-02](sprint-m3-02.md) task 5) that mirrors P-01's jail-capable job: `ubuntu-24.04` amd64; the `runner-v1.1.0` image pinned by digest; `--privileged --cgroupns=private --memory=3g --cpus=2`; `vm.mmap_rnd_bits=32` set first. It covers changed gc items only, plus the `full-run` label, with a job summary of ids, rates and counts only (keep an eye on the < 2,000 Actions min/month budget). **That job's rates are the gate**, recorded as stamps. **Timing isn't taken from CI:** per-test deadlines come from P-01's prod-calibrated go-race baseline in `docs/architecture/runner-tl-baselines.md` ([m3-15](sprint-m3-15.md) creates it) and status.md's runner-stream row. Locally, `make packcheck` runs the non-jail gates (lint, lock, fingerprint, stamp). The arm64 multipass VM is fine for iterating, but its rates don't count. The gates:
  - the reference passes every hidden test **20/20** under `-race` with goleak;
  - each declared mutant gets its class: **RACE ≥ 19/20, DEADLOCK 20/20**, LEAK and WA 20/20 (P-01's acceptance numbers). A mutant below the bar is rewritten or dropped, **never loosened**;
  - per-test deadlines ≥ 10 × P-01's prod go-race baseline; `testing/synctest` for time-based tests ([t3 §6.2](../research/t3-sandbox.md#62-profile-table-limits-are-proposed-a8-tunes-them));
  - reproducibility (`tests.lock`), the review-stamp gate (unstamped items are left out and fall back to self), and `packlint` from task 4.
- **Quiz keys**: the owner-confirmed stamps are present (the session never sets or edits one); keys are never AI-final; task 4's key-shape lint is green, so every key's shape fits the mode task 1 set.
- **Image**: a new **`go-concurrency` layer** (`FROM scratch`, one layer per course); `/manifest.json` lists the gc items with `accepts_contract_hashes`, `grader_kinds` (`code` or `key`) and file hashes; `validated_against` = the public commit of **v1.15.0** (or later). No generators, oracles, mutants or calibration data in the image.
- **Tag `v1.N.0`** (MINOR = items added) **after v1.15.0 is live**, so judge already knows the new key modes and `gotest@1` presentation when the pack lands. Either order is safe (an older judge marks unknown items `unsupported` and stays Ready), but this one makes the first pack load the one that grades.
- **After the push**: the anonymous manifest GET returns **401 or 403**; record the digest. Flux's IUA bumps judge's image-volume `reference:` and judge restarts (one replica: the old pod serves until the new one is Ready).
- Update status.md's **content-status** rows (per tier, per course).

### 8 · Prod verify [X]

- **By looking (D34)**, after both tags: `judge admin status` via `kubectl exec` shows the gc items `ok` with **0 `spec_mismatch`** and the DSA evaluable count unchanged; judge Ready; `k3s kubectl top pod -n xlearn` shows judge well under its memory limit with the new layer (no limit change, so the memory sum is unchanged, [ADR-0035 §5](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#5-capacity-the-memory-sum-rule-triggers-and-ordered-responses)); an anonymous `/xlearn/api/v1/public/stats` and the owner's public profile show **no go-concurrency**.
- **The graded run** (≈ 30 min, a cohort account on prod): enroll if needed, complete **one quiz item** (graded by the key; marks, key and rationale after conclusion) and **one race item** (a submit graded with `trust=honor` and the "Judge · honor" chip; ideally one failing submit first, to see the verdict view). The agent never signs in: it does this run only through an already-signed-in browser session, and records the result here. Otherwise the run is the owner's, as the owner event **`ev-pilot-run`** after ship (this checklist; task 9 adds it to status.md). It gates nothing (D40): _Overall_ ✅ rests on the look above, and anything the owner finds becomes a follow-up PR. The tag unblocks M4-01 either way, since its gate is "v1.15.0 cut".

### 9 · P exit recorded [X]

In [`../status.md`](../status.md):
- **Milestones**: P ✅ with runner-v1.1.0 → v1.15.0 → evalpack `v1.N.0`; exit: "manifest + content with widget/profile code only; manifest `preview`; gc items graded by judge on prod (`ok`, 0 `spec_mismatch`)". P ✅ rests on this session's evidence (D40), not on the owner events below.
- **Tag → floor → snapshot**: v1.15.0 → floor 1.13.0 (unchanged; consumers ≥ 1.7.0) → no snapshot (not required).
- **Release streams**: evalpack `v1.N.0` + digest + `validated_against`.
- **Content status**: go-concurrency rows (items packed per tier: code, quiz; stamps; pack version).
- **Flag inventory**: no new flag; `COURSE_STATUS_OVERRIDE` (P-02) unchanged, removal milestone as P-02 recorded.
- **Decisions log**: where the gc key modes live (what M3-06's split left; every mode in `internal/course/keys`); the additive `mode`/`kind` fields (or "not needed") and which gc items and probes set them; the reveal-rule wiring (M3's `concluded` path); the AB14 F11 resolution as frozen (and the ADR-0029 §2 amendment if F11 draws the D17 spoiler); the quiz "late" definition (the named clean-window field, or "no late band"); the parts/results registries (found or created) and the `onFocusPart` `loc` extension; the touch `mistake_hint` for key probes (present, added, or a follow-up); the go-race gate lane (the private `gorace` job) and its rates; the judge-first/pack-second order; the `honor` labelling of gc code results; and whether `partsgen` exists.
- **Owner calendar events** (after ship, gating nothing, D40): `ev-gc-key-reconfirm` (re-read the keys of the items task 1 gave a `mode`/`kind`, ≈ 10 min) if task 1 set any, and `ev-pilot-run` (task 8's graded run, ≈ 30 min) unless task 8 did it in-session. A finding from either becomes a follow-up PR.
- **Pending-smoke notes**: "owner login smoke pending" if the release checklist's login smoke couldn't run in an already-signed-in browser session.
- **Sprint board**: P-03 ✅; note M4-01 and GA-01 are unblocked on the P side.

## Acceptance criteria

- [ ] **P exit:** the course ships as manifest + content with widget/profile code only (the `git grep` check is empty), and the manifest is `preview`: no gc trace outside the cohort on prod (anonymous stats and profile checked).
- [ ] **Prod verified by looking** (task 8): the gc items `ok` with 0 `spec_mismatch`, and the DSA evaluable count unchanged. One quiz and one race item graded on prod through an already-signed-in browser session, or else `ev-pilot-run` recorded in status.md as an owner event after ship (it doesn't gate _Overall_ ✅, D40).
- [ ] Quiz: the key modes the gc items use live in the one evaluator `internal/course/keys` (no second key package) with golden fixtures, and every gc multi/order/predict-output part and probe carries its `mode`/`kind`. `weighted_gate@1` gives the table's grades. The key and rationale appear **only after a counted conclusion**, never in Run or an unconcluded mock, and in the arena only as the frozen F11 allows (denylist tests green). `misconception:*` pre-fills the mistake on revisable items only: it's seen in the rehearsal's failed recall touch, and a drill opens none.
- [ ] Arena: a gc quiz item opens in study mode with its widgets and Mark studied, per the frozen F11 (vitests green).
- [ ] Race view: RACE / DEADLOCK / TLE / LEAK / CE / inconclusive render with AB15's copy as drawn. Submit DTOs carry learner-package frames only (no hidden names, output, timings, arguments or PC offsets). The compile-leak fixture yields only the hidden-API reason code, and the SPA shows AB15 F10's text. Results carry `graded_by=auto, trust=honor` and show the "Judge · honor" chip. A dump frame opens its file tab at its line.
- [ ] `contract_hash` is byte-identical for every existing item (test green); `packlint` and the `content` job cover gc packs and quiz keys.
- [ ] evalpack `v1.N.0` passed the go-race gates (RACE ≥ 19/20, DEADLOCK 20/20) in the private jail-capable `gorace` job, with deadlines from P-01's prod baseline. The anonymous GET is 401/403, and `judge admin status` shows the gc items `ok` with 0 `spec_mismatch`.
- [ ] v1.15.0 is live and verified per the checklist, and status.md records P, the tag → floor, the stream row and the content rows.

## Release

**Tag `v1.15.0`** (indicative: the next free minor, with major = `.release-line` = 1), then **evalpack `v1.N.0`**. No infra PR: no new stream, consumer, service or in-cluster caller; the evalpack ImagePolicy (`>=1.0.0 <2.0.0`) already selects a minor ([rollout §7](../rollout-plan.md#7-indicative-tag-timeline), [ADR-0034 §1.6](../../adr/0034-v2-release-labelling-gating-and-rollback.md#16-indicative-tag-timeline)).

Release checklist ([ADR-0034 §6](../../adr/0034-v2-release-labelling-gating-and-rollback.md#6-release-checklist) + the ADR-0035 §2 standing rule):
- [ ] Before the tag: peers' tags and PRs are checked (parallel sessions; `git ls-remote --tags origin`, `gh pr list`, `git worktree list`, ListAgents)
- [ ] Before the tag: it is the next free version, and its major equals `.release-line`
- [ ] Before the tag: ACL PRs for new streams and consumers are merged
- [ ] Before the tag: a new service's image comes before its policy
- [ ] Before the tag: for a contract: rehearsed in compose, floor marked
- [ ] Before the tag: for a contract, erase or GA tag: `host-verify --cluster` is green (ADR-0035), the host has settled, and the snapshot is taken
- [ ] Before the tag: from M6: no live interviews
- [ ] After the tag (by looking, D34): `/xlearn/api/v1/healthz` reports the version
- [ ] After the tag: `k3s kubectl get deploy -n xlearn` shows the new images
- [ ] After the tag: every `xlearn-*` ImagePolicy's latest equals the tag, and the HelmReleases are Ready
- [ ] After the tag: smoke-test login, the dashboard and coach
- [ ] Record milestone → tag → floor → snapshot and any flag changes in `docs/v2/status.md`
- [ ] (ADR-0035 §2 standing rule, not part of ADR-0034 §6) Every new in-cluster HTTP or NATS caller this tag introduces has its NetworkPolicy (ingress and egress) change in its own infra PR, merged before the tag

For this tag:
- **ACL** and **new service image**: n/a (no new stream, consumer or service).
- **Contract / snapshot**: n/a. No contract migration (the contract-header lint is silent); not an erase or GA tag.
- **From M6**: n/a.
- **NetworkPolicy**: n/a (judge → runner and gateway → judge already exist).
- **Extra smoke** as a cohort account: the Catalog shows go-concurrency with the `preview` badge; a gc quiz and a gc code item open; a DSA problem still grades as before. Like the login smoke above, it runs only through an already-signed-in browser session; otherwise record "owner login smoke pending" in status.md's pending-smoke notes and carry on (D40).

Evalpack `v1.N.0` (the stream's own checks, [ADR-0034 §1.5](../../adr/0034-v2-release-labelling-gating-and-rollback.md#15-other-release-streams)):
- [ ] the next free evalpack minor (`git -C ../xlearn-evalpack ls-remote --tags origin`); private CI green including the go-race gates
- [ ] the anonymous manifest GET returns 401/403; the digest is recorded
- [ ] the IUA bumped judge's `reference:`; judge Ready; `judge admin status` shows the gc items `ok`, 0 `spec_mismatch`

**Rollback** ([ADR-0034 §4](../../adr/0034-v2-release-labelling-gating-and-rollback.md#41-mechanisms-fastest-first)):
- **R-a:** `COURSE_STATUS_OVERRIDE` via an infra PR (as P-02 wired it) hides the course; unsetting `JUDGE_BASE_URL` returns every judged item to the self path (the M3 kill switch).
- **R-b:** narrow the xlearn ranges, never below the floor (1.13.0); for the pack, pin the previous evalpack digest or narrow its policy ([t1 §3.5](../research/t1-content-data-model.md#35-rollback)).
- **R-c:** revert plus a patch tag (the default), in either repo.
- Residue: enrollments and events with `path_slug ≠ dsa` stay (consumers ≥ 1.7.0 decode them); concluded attempts are never re-graded.

## Definition of Done

CI green in both repos · v1.15.0 and evalpack `v1.N.0` cut and deployed by Flux (no hand `kubectl apply`) · the release checklist ticked · compose rehearsed with the real pack locally (the code leg on Linux; nothing private committed) · acceptance criteria met on the session's own evidence (the owner's key re-read and graded run are owner events after ship, D40) · statuses updated (this file + [`../status.md`](../status.md): Sprint board, Milestones P ✅, tag → floor, release streams, content status, Decisions log) · local `main` synced in every repo touched (xlearn and `../xlearn-evalpack`).

## Risks / watch-outs

- **Honor-grade in-process tests.** The gc code tests run inside the learner's process, so a determined learner can forge a pass. Every gc code result carries `trust=honor` and shows AB15's "Judge · honor" chip; judge-checked % excludes it (P7); the quiz keys (pack) stay `checked`.
- **The key leaking before conclusion** is the quiz's main integrity risk. The reveal rule lives only in judge's `Present()`; the widget renders what the DTO carries; the denylist tests cover arena (per the F11 resolution), Run, mock and every pre-conclusion response.
- **A second key evaluator.** Building gc modes in a new `internal/judge` package would make the DSA probe path and judge grade differently. Every mode goes into `internal/course/keys`, which `key@1` calls.
- **Mode fields after authoring.** Adding `mode`/`kind` after `ev-pilot-content` changes those gc items' `contract_hash`. That's why task 7 refreshes `accepts_contract_hashes` (and the owner re-reads those keys after ship, `ev-gc-key-reconfirm`), and why the entry gate checks that no evalpack tag has shipped gc items yet.
- **`contract_hash` drift.** A new config field in the canonical struct can silently change every DSA hash, turning stamped packs into `spec_mismatch` (self-graded). Omit-empty plus the re-hash test guard it; if the test fails, stop.
- **False DEADLOCK on a slow runner** ([t4 §14](../research/t4-judge-contract.md#14-risks)): deadlines ≥ 10 × baseline, the dump must show learner-package frames, and the `throttled` quiet re-run applies ([t3 §5.7](../research/t3-sandbox.md#57-throttled-and-the-quiet-re-run-learner-proof-bounded)).
- **Flaky race detection.** TSAN finds a race only when the interleaving happens (`GOMAXPROCS=2`, `-count=N`). The gate is ≥ 19/20, not 20/20; a mutant below it is rewritten, never accepted with a looser bar.
- **MCQ guessing** (a 4-option guess is right 25% of the time): prefer predict-output blanks in recall bands; options are shuffled per context.
- **Pasting rehearsal output into a public PR.** Only pass/fail lines and grades; never keys, hidden test names or dump frames from the real pack.
- **The PAT expiring** mid-sprint stops the pack bump (the old pod keeps serving). The entry gate checks the date.
- **M4 overlap on `internal/judge`.** M4-01 waits for this tag. Merge every judge change before tagging, and don't start M4 work in this session.
- **Owner availability** for the graded run: nothing waits on it (D40). P's exit rests on the session's own evidence; the owner's run is `ev-pilot-run` unless task 8 did it through an already-signed-in browser session, and anything it finds becomes a follow-up PR.

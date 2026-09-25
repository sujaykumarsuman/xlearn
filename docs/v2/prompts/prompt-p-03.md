# Prompt — Sprint p-03 · Quiz widget (AB14) + race verdict UI (AB15) → v1.15.0

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-p-03.md`](../sprints/sprint-p-03.md)   ·   **Milestone:** P (exit)   ·   **Prereqs:** [p-02](../sprints/sprint-p-02.md) merged, [p-01](../sprints/sprint-p-01.md) (`runner-v1.1.0` live), [ds-p-01](../sprints/sprint-ds-p-01.md) boards frozen, owner event `ev-pilot-content` done

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md): conventions, the land-and-sync rule, and **"never copy content from `../xlearn-evalpack` into this repo"**. This sprint **tags** (xlearn and evalpack).
- The plan: [`../sprints/sprint-p-03.md`](../sprints/sprint-p-03.md). Its tasks, release checklist and rollback are authoritative.
- **Pilot scope:** [rollout §4 P](../rollout-plan.md#p-pilot-course--34-sprints), [§3](../rollout-plan.md#3-milestone-map) (P exit), [§7](../rollout-plan.md#7-indicative-tag-timeline) (v1.15.0 + runner-v1.1.0), [§9](../rollout-plan.md#9-artboards-by-milestone) (AB14, AB15).
- **Quiz (archetype C):**
  - [t4 §4.3](../research/t4-judge-contract.md#43-archetype-c-quiz--key) (modes, reveal rule, diagnosis, widgets, guessing) and [§4.4](../research/t4-judge-contract.md#44-which-steps-run-in-which-context) (steps per context);
  - [t4 §5.2](../research/t4-judge-contract.md#52-grader-contract-go) (`Present(sr, v)`), [§5.5](../research/t4-judge-contract.md#55-widget-and-result-view-contracts-ts) (`PartWidget`, `ResultView`), [§5.6](../research/t4-judge-contract.md#56-ci-lints-public-ci-and-again-at-judge-start) (lints), [§6.2](../research/t4-judge-contract.md#62-strategies-a-closed-go-set-the-manifest-picks-one-and-sets-thresholds) (`weighted_gate@1`), [§6.3](../research/t4-judge-contract.md#63-mistake-pre-fill-two-tiers-data-only) (`misconception:*`), [§6.6](../research/t4-judge-contract.md#66-touch-formats-and-pass-criteria-per-course) (gc touch bands), [§8](../research/t4-judge-contract.md#8-learner-flows-and-v2-artboards-needed) (one Workspace shell; "Quiz (C)");
  - [ADR-0029 §1–§4](../../adr/0029-judge-contract-and-learning-signal.md#2-the-contract) (keys only after a counted conclusion; `key` never in arena/Run).
- **Race view:** [t3 §5.8](../research/t3-sandbox.md#58-goroutine-dump-go-race) (the dump: learner-package names and file:line only), [§6.2](../research/t3-sandbox.md#62-profile-table-limits-are-proposed-a8-tunes-them) (go-race profile, verdict precedence), [§5.7](../research/t3-sandbox.md#57-throttled-and-the-quiet-re-run-learner-proof-bounded) (`throttled`); [t4 §4.1](../research/t4-judge-contract.md#41-archetype-a-code-ide) (the go-concurrency row: honor, DEADLOCK rule).
- **Content and pack:** [t1 §3.3](../research/t1-content-data-model.md#33-private-eval-pack) (pack image, judge at start), [§3.4](../research/t1-content-data-model.md#34-versioning-and-pinning) (`contract_hash`, pack semver), [§3.7](../research/t1-content-data-model.md#37-local-dev-and-public-ci) (`EVALPACK_DIR`), [§7.1](../research/t1-content-data-model.md#71-package-format) (key sources; the per-course table), [§7.2](../research/t1-content-data-model.md#72-ci-validation) (gates; go-race was deferred to now), [§7.3](../research/t1-content-data-model.md#73-where-ai-may-help); `docs/v2/authoring.md` (M3-01's guide, created when M3-01 runs).
- **Releases:** [ADR-0034 §1.5](../../adr/0034-v2-release-labelling-gating-and-rollback.md#15-other-release-streams) (evalpack stream), [§2](../../adr/0034-v2-release-labelling-gating-and-rollback.md#2-feature-gating-three-tiers-no-flag-service) (`preview` hidden everywhere; the flag inventory), [§4.4](../../adr/0034-v2-release-labelling-gating-and-rollback.md#44-reversibility-by-step) ("P pilot" row), [§6](../../adr/0034-v2-release-labelling-gating-and-rollback.md#6-release-checklist); [ADR-0026 §1](../../adr/0026-per-course-extensibility-model.md#1-a-course-is-data-a-capability-is-code) (a course is data).
- **Boards:** the frozen `design-system/screens/v2/AB14-*.html` and `AB15-*.html` (preview-only; never imported by `web/`), plus AB08 (the results dock copy, for states AB15 doesn't draw) and AB10 (arena study mode). **AB15's copy wins for every `gotest@1` string.** Reuse [`../../../design-system/theme.css`](../../../design-system/theme.css) verbatim.
- **Code:**
  - **the one key evaluator:** `internal/course/keys` ([m2-01](../sprints/sprint-m2-01.md) task 3; M3-06 added the quiz modes there, or recorded a split in status.md's Decisions log);
  - judge: `internal/judge/grader` (the closed registry, `key@1` calling `internal/course/keys`, `composite@1`, `Present`), `internal/judge/dto/` (the allowlist; `concluded=true` gates `key` marks and `answer_reveal`, [m3-14](../sprints/sprint-m3-14.md)), the denylist tests, P-01's `gotest@1` mapping, `internal/judge/testdata/`;
  - course/curriculum: `curriculum/_schema/item.schema.json`, `internal/course/item.go` (`Validate`), the freeze guard against `internal/course/testdata/item.schema.v1.frozen.json`, the answer-free test, `internal/course/canon` (`contract_hash`, golden vectors), `curriculum/courses/go-concurrency/`;
  - practice: `internal/practice/grading` (`weighted_gate@1`, [m3-08](../sprints/sprint-m3-08.md)'s fixture table), the touch conclusion (`touch_concluded`'s `mistake_hint`);
  - web: `web/src/screens/Workspace.tsx` and `web/src/screens/workspace/CodeEditor.tsx` ([m3-11](../sprints/sprint-m3-11.md)), `web/src/components/judge/ResultsDock.tsx` and `web/src/screens/Arena.tsx` ([m3-12](../sprints/sprint-m3-12.md)), P-02's multi-file code widget, `web/src/screens/Touch.tsx`, `web/src/lib/judge.ts`. `web/src/parts/registry.ts` / `web/src/results/registry.ts` exist only if M3/P-02 created them (step 2 creates them otherwise);
  - tooling: `cmd/packlint`, `cmd/contentlint`, `hack/git-hooks/pre-push`, `.github/workflows/ci.yml` (the `content` job);
  - private: `../xlearn-evalpack` (`courses/go-concurrency/`, `.github/workflows/packcheck.yml`, `make packcheck`, `make build`).

## Context

- **M3 shipped (v1.14.0):** judge with the `code` and `key@1` graders (key inline, counted contexts only, once per context, calling the one evaluator `internal/course/keys`), `composite@1`, `weighted_gate@1` in practice, the DTO allowlist (`concluded=true` gates key reveals) and denylist, CodeMirror, the results dock, and the generic arena study mode (M3-12 handed **quiz** study mode to this sprint). `JUDGE_BASE_URL` is set and judge features are cohort-only.
- **P-01** shipped `runner-v1.1.0` (go-race with TSAN, `gotest@1`, goroutine dump, RACE/DEADLOCK/LEAK) and judge's mapping and lints. **P-02** merged the go-concurrency manifest (`preview`, prefix `gc`), `preview` gating, the multi-course catalog/agenda/nav and the multi-file workspace; it ships in this tag.
- **The owner authored and stamped the pilot** (`ev-pilot-content`): public halves on xlearn `main`, hidden tests, mutants and quiz keys on `xlearn-evalpack` `main`.
- **This sprint closes P:** the quiz end to end (AB14), the race verdict view (AB15), `packlint`/content CI for the pilot, **tag v1.15.0**, then the **evalpack minor** with the go-race gates, then the prod verification by looking. The owner's own graded run on prod follows as an owner event after ship (`ev-pilot-run`), unless you do it through an already-signed-in browser session.
- **Floor after the tag:** unchanged (1.13.0; consumers ≥ 1.7.0 for non-DSA events). No infra PR, no new pod, no new caller.

## Entry gates — verify first (stop and report if any is unmet)

- [ ] **Pilot content stamped:** `curriculum/courses/go-concurrency/items/gc-*` on `main` with the `content` job green; `../xlearn-evalpack` `main` has the gc hidden tests, mutants and owner-confirmed quiz keys with review stamps. (If the frozen schema has no mode selector, the quiz parts carry none yet; step 1 adds it.)
- [ ] **No evalpack tag has shipped gc items yet:** `git -C ../xlearn-evalpack tag --contains <the gc merge commit>`; for each such tag, its `packcheck` build summary (ids and counts only) lists no gc item as included. If one does, stop and report.
- [ ] **AB14 F11 as frozen:** read F11 on the board on `main` (arena answers behind a D17 spoiler vs t4 §4.3 / ADR-0029 §2). The DS-P-01 merge froze it as drawn and its PR lists it under "Decisions to confirm"; a later follow-up design PR may have changed it. Record the resolution in the Decisions log. Not a wait (D40): the frozen board is the answer.
- [ ] **P-02 merged** (manifest `preview`, gating tests, `COURSE_STATUS_OVERRIDE` in the status.md flag inventory, the multi-file workspace, AB02/AB05 full fidelity).
- [ ] **P-01 done:** read-only `ssh vps 'k3s kubectl get deploy -n xlearn-runner -o wide'` shows `runner-v1.1.0`, and P-01's judge `gotest@1` mapping + lints are on `main`.
- [ ] **v1.14.0 live**, `JUDGE_BASE_URL` set on the gateway, judge features cohort-only.
- [ ] **AB14/AB15 frozen** (the DS-P-01 PR merged, which is the freeze; status.md artboard rows "frozen (PR #, date)").
- [ ] **PRD Q5 = go-concurrency** in status.md. **If SQL was chosen, stop and report**: P needs re-planning.
- [ ] **The evalpack PAT expiry** in status.md is more than 14 days away.
- [ ] **Parallel sessions:** `git ls-remote --tags origin` (xlearn and `../xlearn-evalpack`), `gh pr list` in both, `git worktree list`, ListAgents. Find the **next free xlearn minor** (use it instead of 1.15.0 everywhere if taken) and the **next free evalpack minor** (`v1.N.0`). **No peer PR may be editing `internal/judge`**; if an M4 PR is open, stop and coordinate.

## Do this (in order; step N = plan task N)

1. **[X] Key grading for the gc quizzes.**
   - **Read M3-06's split first** (status.md's Decisions log and M3-06's ADR): which t4 §4.3 modes landed in `internal/course/keys`. Then **inventory** the modes the stamped gc items and probes use. The expected set is `choice.single`, `choice.multi` (`all_or_nothing` default), `choice.order` (`exact` default; P-02's gc-010) and `blank.predict_output` (`trim_lines` + `crlf`, exact, no typo tolerance); `blank.text` / `blank.numeric` only if used.
     - A mode that landed: gc-shaped synthetic golden fixtures only.
     - A missing mode: implement it **in `internal/course/keys`** (its closed registry; normalizers beside `text`/`big_o`), reached through judge's `key@1`. **Never create a second key package under `internal/judge`.**
     - Fixtures: correct, each distractor, empty, over-selected, malformed.
   - **Mode selector (if the frozen schema lacks one):** add optional `choice.mode` (`single|multi|order`, default `single`) and `blank.fields[].kind` (`text|numeric|big_o|predict_output`, default `text`) on `parts[].config` and `revision.probes[].config`, in `curriculum/_schema/item.schema.json` and `internal/course/item.go`.
     - The freeze guard (vs `internal/course/testdata/item.schema.v1.frozen.json`, never edited) and the answer-free test stay green.
     - **`contract_hash` stays byte-identical for every item that doesn't set them:** marshal omit-empty in `internal/course/canon`, and add a test that re-hashes every live DSA item against the embedded values. **If any hash drifts, stop.**
     - **Then set them on the gc content** (every multi/order part and probe; every predict-output field) and re-run the `content` job. Those gc hashes change on purpose; step 7 refreshes the pack (the owner re-reads those keys after ship, `ev-gc-key-reconfirm`).
   - **`Present()` reveal rule on M3's path** (`concluded=true`, set by the gateway from practice's state; M3-14/M3-09). Don't add a second path.
     - Before the counted conclusion: only answered/unanswered per question.
     - After it (a Miss included): per-question marks, the key, `rationale_md` (via M1-06's Markdown renderer) and the chosen distractor's misconception label. Add the last two to the allowlisted DTO if M3 lacks them.
     - Unconcluded mock: aggregate only. Run and arena: `skipped(not_evaluated_here)`. The one exception is the arena D17 spoiler, only if the frozen F11 draws it (step 2).
   - **Signals:** M3-06's `key@1` already emits `key_wrong:<field>` / `misconception:<category>` in `evaluation_completed`. Prove each gc mode emits them (synthetic fixtures); add emission only for a mode you add.
     - Every gc quiz is a drill (`revision.drills false`), so pre-fill fires only from a **failed recall touch on a core code item**.
     - Check that `touch_concluded` carries the probes' `mistake_hint` (M3-08 added it "for judged re-solves"). If it doesn't, derive it the same way from the touch's key evaluations (additive, existing field, no migration), or record a follow-up if it's bigger than that.
   - **Strategy golden rows** (`weighted_gate@1`, `clean_pct 1.0`, `pass_pct 0.8`, attempt 45:00):
     - 6/6 before `deadline_at` → clean; 5/6 → rough; 4/6 → miss;
     - give-up, or no `final` by the deadline → miss (D15);
     - a second `final` in one context → refused;
     - a `final` after `deadline_at + 2 s` → 409 `attempt_expired` (M3-09), and the ticker concludes Miss.
     - **"Late":** read M3-08's `weighted_gate@1` fixture table. If it uses a clean-window param, name that field, set it in the gc manifest's `weighted_gate@1` params, and add "6/6 after the window, ≤ `deadline_at` → rough". If not, there is no late band: drop that row and record why. (`clean_within_s` is a `verdict_timer@1` param, not a quiz one.)
   - **Extend the denylist tests:** no `answer`, `accept`, `rationale`, `distractors` or per-question correctness in any response before conclusion, in Run, in an unconcluded mock, or in the arena (after a recorded reveal, key + rationale only, and only if the frozen F11 draws the spoiler).

2. **[X] Registries + quiz widget + Workspace-Quiz + Touch probes + arena quiz study mode (AB14).**
   - **Registries** (t4 §5.5). No upstream plan creates them: M3-11's editor is `web/src/screens/workspace/CodeEditor.tsx`, and M3-12's dock is `web/src/components/judge/ResultsDock.tsx`.
     - If `web/src/parts/registry.ts` / `web/src/results/registry.ts` exist (P-02 may have started one), add to them.
     - Otherwise create them as closed maps. Register M3's `CodeEditor` (with P-02's multi-file variant) as `code` and the dock's code rendering as the `code` result view, and have `ResultsDock.tsx` render step results through the registry. M3's/P-02's vitests stay green unchanged, and the editor stays lazy.
     - Record which case you found.
   - `web/src/parts/choice/` and `web/src/parts/blank/` as `PartWidget`s, driven by `mode`/`kind`: `empty/validate/serialize/deserialize`, drafts 5 s / 25 s, modes `edit|readOnly|review|locked`.
     - Accessibility: native radio/checkbox in `fieldset`/`legend`, move buttons + Alt+Arrow for order, `inputmode=decimal`.
     - A deterministic per-context option shuffle (seeded by the context id).
     - Keep the `registry.gen.json` drift test green if M3 created `cmd/partsgen`; don't create it otherwise.
   - **Workspace-Quiz** = the A7 variant of `web/src/screens/Workspace.tsx` (key-only `grader[]` → quiz), not a new route. Frames per AB14:
     - question cards, "4 / 6 answered", the server timer ring;
     - **Submit answers** (one `final`) + the unanswered confirm; the settling phase;
     - marks + key + rationale after conclusion, the misconception label;
     - the timeout → Miss (D15);
     - < 1024 px tabs.
   - `web/src/results/key/` registered as `kind: 'key'`, rendered by `ResultsDock.tsx`. It renders only DTO content.
   - **Touch:** `web/src/screens/Touch.tsx` renders the gc L1–3 recall-quiz probes via the registry, with lock-in where the band sets `timer_s`; pass = s ≥ `pass_pct` ∧ the required questions correct; "advances to Day N" / "resets to Day 1".
   - **Arena quiz study mode (AB14 F11):** a deliverable here, since M3-12 handed it over. `web/src/screens/Arena.tsx` renders the item's `choice`/`blank` parts in `edit` mode (nothing graded; key steps show `not_evaluated_here`), with the "Nothing here counts toward your course." banner and **Mark studied**.
     - **Frozen F11 keeps "no key in the arena":** public content only, no spoiler control, denylist unchanged.
     - **Frozen F11 draws the D17 spoiler** (as DS-P-01 drew it): the confirm posts M3-09's `POST /api/problems/{id}/arena/reveal` (records `arena_revealed_at`, no cap). Only after that record does the gateway ask judge's `Present()` for key + rationale, under a new `arena_revealed` visibility (never from the client), with no per-question marks. Add a dated amendment note to ADR-0029 §2 and a Decisions-log row.
     - Vitests for F11.

3. **[X] Race verdict view (AB15).**
   - **judge DTO:**
     - Run: per-test `{name, status, class}`, the bounded output, learner frames.
     - Submit: hidden passed/total + the first failure class + learner-package frames `{func, file, line}` in editable files only. No hidden names, `TestEvent.Output`, per-test timings, argument words, PC offsets or hidden-test frames; other frames collapse to a count.
     - **Reason codes, not prose:** `graded_by=auto`, `trust=honor`, class and reason codes (hidden-API CE, `runner_throttled`, …). The SPA owns every string. P-01's compile-leak fixture yields the hidden-API reason code only.
     - Extend the denylist test.
   - **web:** extend the `code` result view (M3-12's dock code rendering, behind `web/src/results/registry.ts`) for `gotest@1`, with the pieces under `web/src/results/code/gotest/`:
     - RACE/DEADLOCK/TLE/LEAK banners (precedence RACE > DEADLOCK > TLE > LEAK > RE > WA);
     - the Run test list;
     - the goroutine-dump panel (learner frames; a click opens that file tab at that line).
   - **Copy: frozen AB15 wins for every string**, as drawn:
     - F7 TLE;
     - F10 CE, whose gc variant is "Hidden tests don't compile against your code. Check the exported names in the statement. (Not counted.)" (DSA keeps AB08's "Check the required signature");
     - F11 REJECTED;
     - F12 inconclusive ("We couldn't get a reliable result (the server was busy). Not counted. [Submit again]");
     - the chips: **"honor · tests run in-process"** while solving (F1) and **"Judge · honor"** + F13's tooltip on the concluded grade. `auto · honor` is data, not copy.
   - **Navigation:** extend `onFocusPart(id)` to `onFocusPart(id, loc?: {file, line})`. P-02's multi-file widget opens the `file` tab and highlights `line`; read-only files get no edit affordance. Record it in the Decisions log.
   - Multi-file CE/vet positions map to the right tab through the same `loc`.
   - The same view in a course attempt, the Touch re-solve (gc L4–5) and the arena.

4. **[X] `packlint` + content CI: verify P-01's, add only what's missing.**
   - **Verify** P-01's `gotest@1` lints and gates on the real gc packs: `packlint check` locally against `../xlearn-evalpack` (pass/fail and ids only) and the `content` job on `main`. That covers `tests[]` + go-race params, hidden tests vs `api[]`, no `TestMain`, Σ TL ≤ 40 CPU-s, `mem_mb` ≥ 2 × the reference peak, the #15 fixture, the reference vs `visible_test.go` in the runner image, and the `gotest@1` Submit denylist. Add anything P-01 left out (hidden tests listed in `pack.json`, deadline ≥ 10 × the go-race baseline, mutants declaring a class).
   - **`cmd/packlint`, quiz keys (new):**
     - keys present for every key part and `key_source: pack` probe; `answer ⊆` option ids; `accept[]` non-empty; distractor categories ∈ the manifest's categories; `rationale_md` per question; key parts `cadence: final`;
     - **key shape ↔ mode/kind:** `single` → exactly one `answer`; `multi` → `answer[]`; `order` → `order[]`, a permutation of every option id; `predict_output` → `accept[]` with `trim_lines`/`crlf` only and no `typo`; `numeric` → `value, unit, tol`.
   - **`cmd/contentlint`** + the `content` job: strict quiz-part and probe-config decoding, unique ids, the optional `mode`/`kind`. **Verify** that M3-01's label-edit flag covers `choice` labels and that P-01's gate compiles gc starters from `go.mod.tmpl`.
   - Update `docs/v2/authoring.md` (M3-01's guide) with the gc pack format, the quiz key format with `mode`/`kind`, and "AI proposes keys, the owner confirms". Extend the pre-push hook's shingles to `*_test.go` bodies and `rationale_md`.

5. **[X] Tests + compose rehearsal.**
   - **Public CI:**
     - synthetic items and a synthetic fixture pack under `internal/judge/testdata/` only (test-only ids, never real gc keys);
     - the strategy rows and the denylists (arena per F11);
     - vitests for every AB14/AB15 frame (F11 included), with M3's/P-02's vitests unchanged;
     - P-02's `preview` tests and the `contract_hash` no-drift test;
     - the synthetic failed-recall-touch → `misconception:*` pre-fill test (none for a drill).
   - **Locally:** `make build` in `../xlearn-evalpack`, then `EVALPACK_DIR=../xlearn-evalpack/build docker compose up --build` (**never copy it into this repo**). The jail can't run on macOS: on the Mac run legs 1, 3 and 4, and run leg 2 in the arm64 multipass VM (indicative) or rely on CI's jail-capable e2e plus step 7's gates. Record which. As a cohort account:
     1. a gc quiz (a drill) with a distractor → the table's grade; the reveal only after conclusion; **no** mistake entry ("Drill · not scheduled for revision");
     2. gc code (Linux lane): a race mutant → RACE, a deadlock mutant → DEADLOCK + the dump, a leak mutant → LEAK, the reference → Clean with "Judge · honor";
     3. a gc L1–3 recall touch on a core code item. Conclude the item first (a give-up works without a runner), then make its L1 touch due with a dev-only SQL update, e.g. `docker compose -p xlearn-local exec -T postgres psql -U xlearn -d xlearndb -c "update review.revision_item set due_date = now() - interval '1 minute' where …"` (v1's column names; use what M1/M2 left; never on prod).
        - (a) lock in a distractor → "resets to Day 1" + the mistake with the `misconception:*` category pre-filled;
        - (b) make it due again and pass it → "advances to Day N";
     4. a non-cohort account and an anonymous profile → no gc trace.
   - Paste **only** pass/fail lines and grades into the PR. Never keys, hidden test names or real dump frames.
   - **P exit grep:** `git grep -n 'go-concurrency\|"gc"' -- 'internal/**/*.go' 'web/src/**' ':!**/*_test.go' ':!**/*.test.*' ':!**/testdata/**'` must be empty.
   - **Gates:** `gofmt -l .` · `go vet ./...` · `go test -race ./...` · `sqlc diff` · the contract-header lint silent · web typecheck/lint/test/build · e2e `-tags e2e`.

6. **[X] Merge + tag v1.15.0.**
   - Branch `feat/p-03-quiz-race-ui`, conventional commits with the attribution lines, CI green, squash-merge.
   - Then run the plan's **release checklist** and tag. Title **`v1.15.0 — v2 build · P pilot (preview)`**; release notes per the plan (the second course `preview` for the cohort, the quiz, the race view, the reveal rule).
   - After the tag, by looking: healthz, `get deploy`, ImagePolicy latest = tag, HelmReleases Ready, the smoke test (login, dashboard, coach), plus the Catalog `preview` badge, a gc item opening, and a DSA problem grading as before. The login-dependent checks run only through an already-signed-in browser session (you never sign in); otherwise record "owner login smoke pending" in status.md's pending-smoke notes and carry on.

7. **[E] Evalpack `v1.N.0`** in `../xlearn-evalpack` (after v1.15.0 is live):
   - **Contract refresh:** `packlint check` names the gc items whose hash step 1 changed; add the new hashes to their `accepts_contract_hashes`. Nothing waits for the owner (D40): never set or edit a key stamp, and step 4's key-shape lint proves each key fits its new mode. The owner re-reads those keys after ship (`ev-gc-key-reconfirm`, about 10 min); a correction is a follow-up pack PR.
   - **The go-race gates run in a jail-capable Linux lane, not on macOS.** Add a `gorace` job to `.github/workflows/packcheck.yml` mirroring P-01's job:
     - `ubuntu-24.04` amd64; the `runner-v1.1.0` image by digest; `--privileged --cgroupns=private --memory=3g --cpus=2`; `vm.mmap_rnd_bits=32`;
     - changed gc items only (+ `full-run`); a summary of ids, rates and counts only; mind the < 2,000 min/month budget.
     - Its rates are the gate: the reference 20/20; mutants **RACE ≥ 19/20, DEADLOCK 20/20**, LEAK/WA 20/20 (rewrite or drop a mutant below the bar, **never loosen it**).
     - **Deadlines come from P-01's prod-calibrated go-race baseline** (`docs/architecture/runner-tl-baselines.md`, status.md's runner-stream row) at ≥ 10 × it, not from CI timing.
     - Local `make packcheck` runs the non-jail gates: `tests.lock`, the stamp gate, `packlint`, the fingerprint.
   - The `go-concurrency` layer + `/manifest.json`, with `validated_against` = the v1.15.0 commit or later.
   - PR → CI green → squash-merge → tag `v1.N.0`.
   - The anonymous manifest GET → 401/403; record the digest.
   
   The IUA bumps judge's `reference:`, and judge restarts.

8. **[X] Prod verify.**
   - Read-only: `ssh vps 'k3s kubectl exec -n xlearn deploy/xlearn-judge -- judge admin status'` shows the gc items `ok`, **0 `spec_mismatch`** and the DSA count unchanged. `k3s kubectl top pod -n xlearn` shows judge under its limit. Anonymous `/xlearn/api/v1/public/stats` and the owner's public profile show no go-concurrency.
   - The graded run: as a cohort account, enroll, complete **one quiz item** and **one race item** (one failing submit first). Do it only through an already-signed-in browser session; you never sign in. Otherwise add the owner event `ev-pilot-run` (this checklist, after ship) to status.md and carry on: it gates nothing, and anything the owner finds becomes a follow-up PR.

9. **[X] Record the P exit.** See *Update status*.

## Constraints

- **Service boundaries ([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)).** judge produces evidence (the key grader, `Present()`); practice alone concludes and writes the learning signal; review pre-fills mistakes from signals. No cross-schema reads. The reveal rule lives **only** in judge's `Present()`: the SPA renders DTOs.
- **A course is data.** No `go-concurrency`/`gc` literal in service or web code; new capability code is keyed by part type, mode or harness, never by course.
- **Closed registries**, no `init()` self-registration. **Answer-free public schema:** no correctness flag, key or rationale ever in the public repo.
- **goose + sqlc:** no migration is planned. If one turns out to be needed, it's **expand-only** (the contract lint stays silent), `sqlc generate` is committed and `sqlc diff` is clean. Never run `Down` in prod.
- **Outbox/inbox:** the signals travel in the existing `evaluation_completed` (outbox, same transaction as the evaluation). No new subject, stream or consumer, so **no ACL PR**, and consumers-before-producers is untouched.
- **`theme.css` verbatim**, matching the frozen boards; the boards are never imported or shipped.
- **One key evaluator:** every key mode lives in `internal/course/keys` (T4 §12, M3-06's ADR); judge's `key@1` calls it. Never a second key package.
- **Where things run:** the runner jail is Linux-only. The go-race gates' rates come from the private `gorace` CI job (amd64); their deadlines come from P-01's prod baseline; the macOS compose runs no code leg.
- **Private content:** never copy anything from `../xlearn-evalpack` into this repo, its PRs or its CI logs. AI may draft tests and mutants; **keys are owner-confirmed**; expected outputs are never AI-written ([t1 §7.3](../research/t1-content-data-model.md#73-where-ai-may-help)).
- **GitOps:** never `kubectl apply`; Flux deploys both tags. Never move or re-push a tag. Don't suspend the shared IUA. The only prod exec is the read-only `judge admin status`.
- **Memory-sum rule ([ADR-0035 §5](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#5-capacity-the-memory-sum-rule-triggers-and-ordered-responses)):** no new pod and no limit change; the pack layer is read lazily by judge.
- **No new in-cluster caller**, so no NetworkPolicy PR. **No alerting (D34):** verification is by looking.
- **M4 isolation:** land every `internal/judge` change before the tag; start no M4 work here.
- **Parallel sessions:** check peers' tags, PRs and worktrees (+ ListAgents) in **both** repos before each tag and before claiming any ADR number, and again right before each tag push.

## Deliverables

- The gc key modes in `internal/course/keys` (+ golden fixtures); the reveal rule exercised on M3's `concluded` path (+ rationale and misconception label); the signals proven per mode; the touch `mistake_hint` for key probes (or a recorded follow-up); `weighted_gate@1` golden rows with "late" defined or dropped; the optional `mode`/`kind` fields (or a recorded "not needed"), set on the gc content, with the `contract_hash` no-drift test.
- The part/result registries (found or created); the `choice`/`blank` widgets, Workspace-Quiz, the `key` result view, the Touch recall-quiz probes and arena quiz study mode per the F11 resolution (AB14).
- The `gotest@1` learner DTO fields (reason codes) and the race/deadlock/leak view with the goroutine-dump panel, AB15's copy and chips, and `onFocusPart(id, loc?)` (AB15); the extended denylist tests.
- P-01's lints verified on the real packs; the quiz-key and key-shape lints; the `contentlint` checks; `docs/v2/authoring.md`; the pre-push hook extension.
- **v1.15.0** tagged and verified; **evalpack `v1.N.0`** with the `gorace` CI job's gates, refreshed gc hashes and its digest recorded; the graded run done through an already-signed-in browser session, or `ev-pilot-run` recorded.

## Update status

- In [`../sprints/sprint-p-03.md`](../sprints/sprint-p-03.md): set the task rows ✅ (or ⛔ with the reason), and set _Overall_ ✅ on this session's own evidence (D40: the owner's key re-read and graded run don't gate it).
- In [`../status.md`](../status.md):
  - **Sprint board** P-03 ✅. **Milestones** P ✅ (runner-v1.1.0 → v1.15.0 → evalpack `v1.N.0`; exit: manifest + content, widget/profile code only, `preview`, gc items graded by judge on prod);
  - **Tag → floor → snapshot:** v1.15.0 → 1.13.0 (unchanged; consumers ≥ 1.7.0) → none (not required);
  - **Release streams:** evalpack `v1.N.0`, digest, `validated_against`;
  - **Content status:** the go-concurrency rows (packed per tier: code, quiz; stamps; pack version);
  - **Flag inventory:** no new flag; `COURSE_STATUS_OVERRIDE` unchanged;
  - **Owner calendar events** (after ship, gating nothing): `ev-gc-key-reconfirm` if step 1 set any `mode`/`kind` (re-read those keys, ≈ 10 min), and `ev-pilot-run` unless step 8 did the graded run in-session (≈ 30 min);
  - **Pending-smoke notes:** "owner login smoke pending" if step 6's login-dependent checks couldn't run;
  - **Decisions log:** where the gc key modes live (M3-06's split outcome); the `mode`/`kind` fields or "not needed", and which gc items set them; the reveal-rule wiring; the AB14 F11 resolution (and any ADR-0029 §2 amendment); the quiz "late" definition or "no late band"; the registries (found or created) and `onFocusPart` `loc`; the touch `mistake_hint` for key probes; the go-race gate lane and rates; judge first, pack second; the `honor` labelling; `partsgen` present or not;
  - note that M4-01 and GA-01 are unblocked on the P side.
- Record an ADR only if you depart from ADR-0026/0027/0029/0034 (an accepted F11 spoiler is such a departure: amend ADR-0029 §2 with a dated note). Check peers before numbering.

## Done when (acceptance)

- [ ] **P exit:** manifest + content with widget/profile code only (the grep is empty); the manifest is `preview`; no gc trace outside the cohort on prod.
- [ ] Prod verified by looking (step 8); one quiz and one race item graded on prod through an already-signed-in browser session, or else `ev-pilot-run` recorded as an owner event after ship (it doesn't gate _Overall_ ✅).
- [ ] Quiz: the used modes in `internal/course/keys` with golden fixtures, with `mode`/`kind` on the gc content; the `weighted_gate@1` rows; the key and rationale only after a counted conclusion, and in the arena only per F11 (denylists green); `misconception:*` pre-fill on revisable items only (seen on the failed recall touch; none for a drill).
- [ ] Arena: a gc quiz opens in study mode with its widgets and Mark studied, per F11.
- [ ] Race view: RACE/DEADLOCK/TLE/LEAK/CE/inconclusive with AB15's copy as drawn; learner-package frames only; the compile-leak case is a reason code with AB15 F10's text; `graded_by=auto, trust=honor` with the "Judge · honor" chip; a frame click opens its file at its line.
- [ ] `contract_hash` unchanged for every item that doesn't set `mode`/`kind`; P-01's lints green on the real packs; the quiz-key and key-shape lints and the `content` job cover the pilot.
- [ ] Evalpack `v1.N.0`: the go-race gates passed in the `gorace` CI job (deadlines from P-01's prod baseline), the key-shape lint green for every key whose mode changed, anonymous GET 401/403, gc items `ok`, 0 `spec_mismatch`. v1.15.0 is live and verified; status.md is updated.

## Ship (land-and-sync — owner approval pre-granted)

> Launching this prompt is the owner's approval for every change it makes (D40); don't stop for review.

1. Branch, then conventional commit(s) with the attribution lines, then push, then a PR in every repo touched (`../infra` PRs first where the order requires it; infra PRs are never folded into a tag). (xlearn `feat/p-03-quiz-race-ui` first; the `../xlearn-evalpack` PR after v1.15.0 is live; no infra PR.)
2. Once CI is green (fix, then merge, on failure), squash-merge. Never enable auto-merge.
3. **Release action — tag `v1.15.0` + evalpack `v1.N.0`:** in order:
   - Walk the release checklist (ADR-0034 §6; the plan's Release checklist, re-checking peers' tags right before the push), push the tag `v1.15.0` (the next free version), let Flux deploy, then verify live by looking (step 6's after-tag reads). Not a contract, erase or GA tag, so no snapshot; no infra PR.
   - Then follow the evalpack stream's own tag procedure (step 7): its PR → CI green with the `gorace` gates → squash-merge → tag `v1.N.0` (the next free evalpack minor) → the anonymous manifest GET is 401/403 → the IUA bumps judge's `reference:` → verify by looking (step 8). Record it under **release streams** (digest, `validated_against`).
4. Update status: the sprint file and `docs/v2/status.md`, in the same PR or a follow-up docs PR merged the same way.
5. Run `git checkout main && git pull` in every repo touched (xlearn and `../xlearn-evalpack`). If a clean peer worktree holds `main`, use `git -C <worktree> merge --ff-only origin/main` and then `git switch --detach main`.

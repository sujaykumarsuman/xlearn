# Sprint m6a-06 — M6a UI part 2: grace/paused/resume (AB26), debrief/proposal (AB27), accessibility (AB28) → v2.0.x patch

> **Milestone:** M6a — text interviewer plus failsafes (T6 **P0**; **this sprint proves the M6a exit** and ships M6a **dark** in a `v2.0.x` patch)   ·   **Track:** product   ·   **Order:** 79
> **Prereqs:** [m6a-05](sprint-m6a-05.md) (AB13/AB24/AB25 screens) · [ds-m6a-02](sprint-ds-m6a-02.md) (AB26–AB28 frozen) · the whole M6a chain on `main`: [m6a-01](sprint-m6a-01.md), [m6a-02](sprint-m6a-02.md), [m6a-03](sprint-m6a-03.md) (incl. the twin fairness gate), [m6a-04](sprint-m6a-04.md)
> **Unblocks:** [mi-13](sprint-mi-13.md) (MI-16 gates; needs M6a live dark) · [m6b-01](sprint-m6b-01.md) (voice builds on the live text interviewer)
> **Release action:** **tag a `v2.0.x` patch** (the next free patch, e.g. `v2.0.N`; M6a stays dark behind the cohort gate) — [ADR-0034 §1.1](../../adr/0034-v2-release-labelling-gating-and-rollback.md#11-scheme): after `v2.0.0` the minor moves only at a GA flip, so this is **not** `v2.1.0`. Infra PR(s) only if task 7's memory read requires one (their own PR, never folded into the tag).
> **Artboards:** **AB26** grace / paused / resume (`design-system/screens/v2/AB26-grace-paused-resume.html`) · **AB27** debrief + proposal (`AB27-debrief-proposal.html`) · **AB28** accessibility settings (`AB28-accessibility-settings.html`)
> **Calendar:** Q1 2027. **Owner event after ship** (`ev-m6a-dogfood`, non-blocking): the owner dogfoods one ~45-minute text mock on production after the tag; task 8 records the event in status.md, and it gates nothing (not _Overall_ ✅, not the M6a row).
> **Execute with:** [`../prompts/prompt-m6a-06.md`](../prompts/prompt-m6a-06.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | AB26: grace modal, paused banner (app-wide), resume modal with the AI brief, `interrupted` / "Still there?", `incomplete` view | X | ⬜ |
| 2 | AB27: debrief, proposal review (accept · edit · re-propose once · self), score once, `incomplete` partial view, in-app reminders | X | ⬜ |
| 3 | AB28: accessibility settings + `coach.interview_pref` + the setup default | X | ⬜ |
| 4 | P0 coverage audit (t6 §9 P0, §6 enforcement, L19, m6a-05's gap notes); fix small gaps | X | ⬜ |
| 5 | M6a exit e2e: a 45-minute text mock with pause/resume scores once (fake clock, fake provider + replay fixtures); `incomplete` and exposure paths | X | ⬜ |
| 6 | Tag the `v2.0.x` patch (release checklist incl. **no live interviews**) | X | ⬜ |
| 7 | Post-tag verify on prod + coach memory read (infra PR only if needed) | H + X (+ I) | ⬜ |
| 8 | Record the owner's post-ship dogfood (one ~45-minute text mock with a pause and resume, scored once) as owner event `ev-m6a-dogfood` in status.md; the event gates nothing | X | ⬜ |
| 9 | Record: M6a row, tag → floor, decisions — in a post-tag docs PR (dogfood result in a second small docs PR) | X | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row, the
> **M6a** milestone row, milestone → tag → floor, the Artboards rows, the owner events (`ev-m6a-dogfood`), the Decisions log). Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] **AB26, AB27, AB28 frozen:** [ds-m6a-02](sprint-ds-m6a-02.md) merged (the merge is the freeze); the three files exist under `design-system/screens/v2/`.
- [ ] **m6a-05 merged:** `MockRoute`, `MockHub`, `ClassicSession`, `InterviewSetup`, `LiveHud` and `web/src/lib/interview/api.ts`, with placeholder cards for the states this sprint renders; its Decisions-log gap notes read.
- [ ] **Twin fairness gate green** ([m6a-03](sprint-m6a-03.md); [t6 §6](../research/t6-realtime-interviewer.md#6-assessment) "before M6a ships"): recorded per catalog model that can review; any failing model is set to `ai: self_only`. The tolerance is written in ADR-0032's follow-up (m6a-03).
- [ ] **NetworkPolicy outcomes recorded:** [m6a-01](sprint-m6a-01.md) task 7's own outcome (none expected: `pause_exposure` goes through the gateway; if it did add a callee, that infra PR is merged) and [m6a-04](sprint-m6a-04.md)'s "no NetworkPolicy PR needed" record.
- [ ] **Post-GA release line:** `v2.0.0` live, `.release-line` = `2`, every `xlearn-*` ImagePolicy `>=1.0.0 <3.0.0` ([ga-02](sprint-ga-02.md)).
- [ ] **Parallel sessions:** the next free patch is known (`git ls-remote --tags origin 'refs/tags/v2.0.*'`, `gh pr list`, `git worktree list`, ListAgents). **D6 content waves also cut `v2.0.x` patches** — don't race a peer's tag. No open peer PR edits `web/src/screens/mock/**`, `web/src/lib/interview/**`, `web/src/screens/Settings.tsx`, `web/src/screens/Dashboard.tsx`, `web/src/components/AppShell.tsx` or coach migrations (take the next free goose version at rebase).

## Goal

Finish the text interviewer and **prove M6a's exit — a 45-minute text mock survives pause and resume and scores once** ([rollout §3](../rollout-plan.md#3-milestone-map)) — then ship it **dark** (owner/tester cohort) in a `v2.0.x` patch:
- **AB26** — the owner's failsafes on screen: the 5-minute top-up **grace** modal (text form), the **paused** banner on every page (≤ 24 h, ≤ 3 pauses), the **resume** modal with the free deterministic state at once and the paid AI brief on a click (cached per pause), and the **`incomplete`** view after 24 h ([t6 §4](../research/t6-realtime-interviewer.md#4-session-state-machine--the-owners-failsafes); D30; R-MI3–R-MI5).
- **AB27** — the brain-authored **debrief**, the AI **proposal** on public descriptors with verified quotes, and the **explicit accept** (no 24-hour auto-submit) → `ScoreMock` **once**; re-propose once; the free partial view for `incomplete` ([t6 §6](../research/t6-realtime-interviewer.md#6-assessment); D14 amended; R-MI6).
- **AB28** — text-mode accessibility settings and the **time multiplier** default ([t6 §7](../research/t6-realtime-interviewer.md#7-privacy-consent-retention--accessibility); WCAG 2.2.1).
- An **audit** that every T6 P0 item exists, an **exit e2e** in CI, the **patch tag** with "no live interviews" checked, and afterwards the owner's **dogfood** on production (a post-ship owner event; it doesn't gate the sprint).

## Scope

**In**
- `web/src/screens/mock/{Grace,PausedBanner,ResumeModal,Interrupted,Incomplete,Debrief,ProposalReview}.tsx`, the `mock/:id/review` route (replacing m6a-05's placeholder), the app-wide banner slot in `AppShell.tsx`, the "Mock interviews" section in `Settings.tsx`, the "Scores waiting" row on Today (`web/src/screens/Dashboard.tsx`, AB27 F8).
- coach: `interview_pref` (table, routes, erase coverage); the per-provider `billing_url` for the grace modal (task 1); gateway pass-through; small P0 gap fixes (task 4).
- `internal/e2e/interview_exit_test.go`.
- The `v2.0.x` patch tag, post-tag verification, the coach memory read, status records (the owner's dogfood follows as a post-ship owner event).

**Out**
- Voice anything: the countdown-grace UI with the mic muted, voice pre-flight and notices, the voice HUD, captions, push-to-talk, "hold voice", rollover and make-before-break → M6b ([m6b-01](sprint-m6b-01.md)…[m6b-03](sprint-m6b-03.md)); AB29/AB30★.
- Interviewer GA (`v2.1.0`, the cohort-gate flip) → [m6b-04](sprint-m6b-04.md).
- The MI-16 gates (Permissions-Policy, SDP route, coach 500m / 256 Mi) → [mi-13](sprint-mi-13.md), unless task 7's memory read pulls the coach sizing forward.
- The public relabel of mock aggregates (O4) — D31 already shows the count only; nothing changes here.
- A large P0 gap (more than a small fix): recorded as a v2.1.0 blocker for [m6b-04](sprint-m6b-04.md), not built here.

## Tasks

### 1 · AB26 grace, paused, resume, interrupted, incomplete [X]

Sources: [t6 §4](../research/t6-realtime-interviewer.md#4-session-state-machine--the-owners-failsafes) (Failsafes 1–3, other failure modes, checkpoints), [ADR-0032 §4](../../adr/0032-realtime-ai-mock-interviewer.md#4-session-state-and-failsafes-owner-spec), D30, D17/D27 (arena open, coach unlocked while paused), [PRD](../../prd/xlearn-v2-prd.md) R-MI3–R-MI5, [ds-m6a-02](sprint-ds-m6a-02.md) task 2 (the frame list). Build **every AB26 frame (F1–F12)** in its layout and copy; this table is what the frames need from the server.

| AB26 frame | Content (summary; the board's copy is final) | Server |
|---|---|---|
| **F1 grace modal** (text, simple; `role="alertdialog"`) over a read-only HUD | "Your ⟨provider⟩ credits ran out — interview paused at 23:41."; [Top up ↗] (the provider's `billing_url`, new tab; see below the table); "add at least ⟨remaining + $1⟩"; "We'll keep checking until 14:10 · checked 20 s ago"; the spend meter vs the cap; [Check now] (≤ 1 per 10 s) · [Save & resume later] · [End interview] (never the default button). The 5:00 countdown ring is M6b's variant | m6a-01's `interrupted(quota)` + `grace_until`; m6a-02's probe (the browser probes every 30 s while visible; the server allows ≤ 1 / 10 s and ≤ 15 per grace; the sweeper's final probe at `grace_until`); `pause` / `abandon` |
| **F2** grace at ≥ 75 % | adds [Finish & get feedback], reading [End without feedback] while the key is still dry | `finish_now` |
| **F3** credits found | "Credits found — picking up where you left off." then the interviewer's re-entry — **no paid summary**; the clock restarts on that message | `cleared` → `connecting` → `live` (60 s stable rule, ≤ 2 re-primes) |
| **F4** other interruptions | offline · provider outage · rate limit (the same 5-minute window, clock stopped); **key rejected** (`ErrAuth`) → straight to paused with the board's copy | `interrupted(network\|provider\|rate\|idle\|tab_closed)`; `auth` → `paused` |
| **F5 paused banner** (every page, every device) | "Interview paused — resume by Thu 14:05" [Resume]; "Pause 1 of 3 · 23:41 done"; the coach-available and arena-noted lines | `GET /api/interviews/active`, rendered in an app-wide slot of `AppShell.tsx` (cached 30 s; refetch on focus) |
| **F6 resume modal — free state, instant** | per-phase elapsed vs budget, time left, hints given, the last code (read-only), Runs and the last result, pauses and reasons, "About $X to finish on your key", the meter, the consent reaffirmation; [Summarise & resume — uses your key (~$0.04)] · [Finish now & get feedback] (≥ 75 %) · [Abandon] | m6a-01's `resume` (→ `resuming`) and `GET …/resume-state` (no model call) |
| **F7** still no credit | "Still no credit on your key — nothing was charged." | m6a-02's `brief` → 409 `still_no_credit` (probe first; `still_dry` → `paused`, same pause) |
| **F8** brief ready | summary, ≤ 5 progress bullets, next step; [Resume interview] (also the user gesture); "we reuse this summary — you won't pay for it twice" | m6a-02's `brief` (structured JSON, cached in `interview_pause.brief`); `resume_confirmed` |
| **F9** pause exposure | chip "Viewed the solution during the pause" + the self-scored note | `interview.pause_exposure` (m6a-01) |
| **F10** last pause; expiry | "This is your last pause (3 of 3)."; **expired (D30)** → [See what you did (free)] (→ AB27 F9) · [Start a new mock] | `pause_count`; `incomplete` via the sweeper |
| **F11** credit ran out at ≥ 90 % | [Finish without the rest] (default) · [Save & resume later] | `finished(report_pending=quota)` |
| **F12** < 1024 px / 390 px | full-screen sheets; the rail as a vertical list; a compact banner | — |

- **`billing_url` (added here; no earlier sprint provides it for text).** m1-10's catalog entries carry no billing link and [m6b-01](sprint-m6b-01.md) adds one only to `voice_shell` entries. Add a **per-provider** `billing_url` to coach's provider registry / catalog (`openai`, `anthropic`; the URLs copied from each provider's console on the day, never guessed, with `as_of`), and expose it on coach's `GET /interviews/{id}` as an additive `provider: {id, label, billing_url}` (gateway pass-through, `openapi.yaml`). A voice entry's own `billing_url` (m6b-01) takes precedence when present. Static data, no model call; the P0 audit (task 4) checks it.
- Replace m6a-05's placeholder cards: `LiveHud` routes m6a-01's `interrupted` (grace = `interrupted` + `grace_until`: reason `quota` shows F1/F2, other reasons F4), `paused`, `resuming` and `incomplete` here; the editor stays read-only in all of them.
- Behaviour per the board's notes: times in the account timezone with the weekday; the grace warning is the **only** `aria-live="assertive"` announcement; [End interview] and [Abandon] are never the default button; **no AI call runs on the key without a click**.

### 2 · AB27 debrief + proposal review [X]

Sources: [t6 §6](../research/t6-realtime-interviewer.md#6-assessment) (flow steps 1–7, the `ai-byo` conditions, no auto-accept, reminders at 24 h and 7 d, `incomplete` after 30 days unscored), [ADR-0032 §5](../../adr/0032-realtime-ai-mock-interviewer.md#5-assessment-and-scoring), [m6a-03](sprint-m6a-03.md) (`GET /interviews/{id}/proposal`, `POST …/proposal` (retry after quota), `…/proposal/repropose` (once), `…/partial-feedback`, gateway `POST /api/interviews/{id}/submit {mode: accept_all|edited|self}` → `ScoreMock` once; `pending_since` on `/interviews/active`), [m6a-04](sprint-m6a-04.md) (`SnapshotAt`, evidence), [ds-m6a-02](sprint-ds-m6a-02.md) task 3, R-MI6–R-MI8. Build **every AB27 frame (F1–F12)**; `mock/:id/review` replaces m6a-05's placeholder (`Debrief.tsx`, `ProposalReview.tsx`).

| AB27 frame | Must do |
|---|---|
| **F1 debrief** (`wrapping`, in the AB25 HUD) | chip "Debrief · up to 3 minutes"; the brain-authored closing turns with 2–3 improvement areas — **never a score or band**; [End debrief] |
| **F2** preparing the review | "one call on your key (about $0.06)"; the aggregate judge line ("hidden tests 37 / 40") from m6a-04's evidence |
| **F3 proposal** (`proposed`) | "AI-proposed scores — nothing is saved until you accept."; per dimension of the session's rubric: band 1–5 **or "No evidence"** with its reason, one verified quote with a message link (opens the transcript at that turn beside the on-screen code via `SnapshotAt`), a one-line why, a slider; the evidence column; ≤ 3 strengths; ≤ 3 improvements with drill links; summary; caveats; [Accept all] · [Re-propose once — uses your key (~$0.06)] · [Score it myself]; "Scores lock when saved."; provenance "Proposed by ⟨model⟩ · ⟨prompt v⟩ · from your messages and code only" |
| **F4** edited | the row tagged `edited`; "saved as self-scored" banner |
| **F5** second proposal | both proposals side by side, differences marked; accept either unchanged or score it yourself (the board's "decision to confirm" — follow the frozen board: the default the ds-m6a-02 merge froze, or a follow-up design PR's change); chip "Re-propose used" |
| **F6 saved** | "Saved · 26 / 35"; the **server-computed** chips (`AI-proposed · accepted` or `self`) + `honor`; notes pre-filled; [Talk it over with your coach →] (review mode); the public count-only line (D31) |
| **F7** self-scored only | one variant per reason (custom model, model not cleared, `review_flag`, pause exposure), none naming a heuristic; quotes and evidence shown; empty sliders |
| **F8** scores waiting | on the Mock hub history **and Today** (`Dashboard.tsx` reads `GET /api/interviews/active` client-side; no new aggregate): "Scores waiting for you"; "never saved automatically"; the reminder copy at 24 h and 7 d from `pending_since` (read-time only, m6a-03); the expiry date (30 days → `incomplete`). **No email or push** |
| **F9** incomplete — free partial view | from the checkpoints (m6a-01's `resume-state`); [Get improvement areas — uses your key (~$0.06)] **only on a click** (m6a-03's `partial-feedback`, probe first; improvements only, never bands) |
| **F10** review failed; cut short | [Try again] · [Score it myself]; the `cut_short=cap` and `report_pending=quota` captions |
| **F11** transcript and retention | per-message delete with the board's confirm (m6a-01 `DELETE …/turns/{seq}`), "deleted automatically on ⟨date⟩", [Keep for 12 months] (`PATCH …/retention`), [Delete transcript now] (`DELETE /interviews/{id}`, terminal only) |
| **F12** < 1024 px / 390 px | dimension rows as cards; [Accept all] first |

- Submit → `POST /api/interviews/{id}/submit {mode: accept_all | edited | self, scores?, notes?}` (m6a-03) → `ScoreMock` **once**; a double click or retry is idempotent and shows the saved view (use the refusal codes m6a-03 merged; the board suggests `already_scored`, `repropose_used`). The UI **never** infers `scored_by`; a `qa_slice` interview (owner QA) can't be scored (409 `qa_session_unscored`).

### 3 · AB28 accessibility settings [X]

Sources: [t6 §7](../research/t6-realtime-interviewer.md#7-privacy-consent-retention--accessibility) "Accessibility", [t6 §8](../research/t6-realtime-interviewer.md#8-cost-on-byo-keys) (the multiplier's cost shown), [ds-m6a-02](sprint-ds-m6a-02.md) task 4 (F1–F7), WCAG 2.2.1.
- **coach** (next free goose version; sqlc; `sqlc diff`): `coach.interview_pref(account_id uuid PRIMARY KEY, default_mode text NOT NULL DEFAULT 'text' CHECK (default_mode IN ('text')), time_multiplier numeric(3,2) NOT NULL DEFAULT 1 CHECK (time_multiplier IN (1, 1.25, 1.5, 2)), reduced_motion text NOT NULL DEFAULT 'system' CHECK (reduced_motion IN ('system','on','off')), announce text NOT NULL DEFAULT 'phases_warnings' CHECK (announce IN ('all','phases_warnings')), updated_at timestamptz NOT NULL DEFAULT now())`. **Hand-off to M6b** (a Decisions-log line + the PR description): [m6b-02](sprint-m6b-02.md) (coach) widens the `default_mode` CHECK to `('text','voice')` and adds the AB28 F3 voice columns (captions, the voice screen-reader preset, camera self-view) on this table, **expand-only**, served by the same `GET/PUT /interviews/prefs`; [m6b-03](sprint-m6b-03.md) renders AB28 F3 and reads them for the voice setup. `GET /interviews/prefs`, `PUT /interviews/prefs` (aud=coach); **add the table to coach's erase transaction** (l-01's coverage test goes red otherwise). Gateway `GET/PUT /api/interviews/prefs` behind the cohort gate; `openapi.yaml`.
- **Settings** (`web/src/screens/Settings.tsx`, "Mock interviews", AB28 F1–F2): the account opt-in toggle (m6a-01's `PUT …/optin`, AB24's strings exactly; off is immediate), "Default mode" [Text], **"Extra time"** 1× · 1.25× · 1.5× · 2× with the board's never-penalised, no-reason copy and its cost line (from the estimate read), "Reduced motion" (system / on / off), "Screen reader announcements" (every interviewer message / only phase changes and warnings — the default). Toggles are `<button role="switch">` with `aria-checked`.
- **In the HUD** (AB28 F4–F6): under reduced motion, interviewer replies render whole (no streaming animation); "Jump to latest" in the turn log; a visible focus order across log, input, editor and bottom bar; the **shortcut sheet** on `?` (send, new line, move between panes, Run, hint, Hold/Talk — Finish and End have **no** shortcut; chords checked against CodeMirror's `defaultKeymap`, AB07's shortcuts, macOS Option input and common screen-reader keys); the "1.5× time" and "Text interview" chips on results and history (neutral, never warn/err).
- **Setup:** m6a-05's multiplier chooser defaults to the saved preference; the audit (task 4) confirms the **server** scales the rail, the idle check-in (3 min × multiplier) and quiet-during-Code (90 s × multiplier) and sets `mock_session.time_multiplier`.

### 4 · P0 coverage audit [X]

Walk every item below against `main` and record a table (item · sprint · evidence: file/test · ✅/gap) in the PR and a summary line in status.md. **Fix a gap here only if it's small** (about an hour, inside the owning service, with a test); anything larger is recorded as a **v2.1.0 blocker** for [m6b-04](sprint-m6b-04.md) with a one-line scope.

| Source | Items |
|---|---|
| [t6 §9](../research/t6-realtime-interviewer.md#9-phased-plan) P0 | brain loop + voice-ready segments; server clock and rail; manifest `mock.interviewer`, `mock.rubric.dims[].evidence`, `ai: propose \| self_only \| self_only_voice`; classifier incl. `ErrModelAccess` + probe; grace, pause ≤ 24 h (≤ 3), resume brief cached per pause, sweeper, `incomplete`; deterministic checkpoints; `pause_exposure`; `give_hint`; review → proposal → explicit accept → `ScoreMock` once; re-propose once; twin gate; `interview_*` tables, consent, retention (30 days after scoring; 12 months opt-in; per-turn delete), erase incl. segment log and consent rows; **`store:false` acceptance test**; CodeMirror interview mode; judge mock Runs and final; assessment deltas; text-mode accessibility; replay regression tests (CI) |
| [t6 §6](../research/t6-realtime-interviewer.md#6-assessment) enforcement | evidence enum has no audio/video/prosody/face member + the CI never-list lint; evaluative speech brain-authored only; the words-triggered distress card (Tele-MANAS 14416); the post-hoc never-list scan → `review_flag`; the deny-lexicon on the written review; custom model ids never get a proposal; missing evidence → `band: null` |
| [t6 §7](../research/t6-realtime-interviewer.md#7-privacy-consent-retention--accessibility) | S1 lint on instruction text; hint text only via `give_hint` (S4); the logging canary covers `/api/interviews/*`; consent version stored; "I'm an AI interviewer" disclosed at the start |
| [ADR-0035 §4](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#4-limits-inventory) L19 | 1 non-terminal per account; ≤ 2 starts / day; mandatory $ cap (85 % wrap, 100 % cut); ≤ 1 snapshot / 2 s at 64 KiB; mock Run ≤ 1 / 10 s; aud=coach |
| Cross-service | the grace modal's per-provider `billing_url` (task 1) for both text providers; coach's `incomplete` / `abandoned` reaches assessment's `mock_session.status` (abandon at once, `incomplete` lazily on the next read — m6a-03), and trends count `scored` only either way; coach `locked` in-session and unlocked while paused; `coach admin interviews --live` works distroless |
| [m6a-05](sprint-m6a-05.md) | every gap it logged; the text-mode Hold (AB25 F7) and the estimate read exist (built by a merged sprint or by m6a-05's conditional pieces); the beacon reaches the interrupt route |

### 5 · M6a exit e2e [X]

Sources: [rollout §3](../rollout-plan.md#3-milestone-map) (the M6a exit), [t6 §11](../research/t6-realtime-interviewer.md#11-what-t6-constrains-downstream) QA (replay regression; the 0.33× slice rail is for manual QA only), [m6a-02](sprint-m6a-02.md) (replay fixtures), the S6 fixtures under `docs/v2/research/t6-s6-fixtures/` ([spk-04](sprint-spk-04.md)).

`internal/e2e/interview_exit_test.go` (`//go:build e2e`; the core-loop harness: real Postgres, services in-process): identity (an owner-role account), gateway, coach, assessment, practice, judge with m3-06's fake runner; the **fake provider** is m6a-02's replay harness (`internal/coach/interview/replay/`: recorded, scrubbed fixtures) switching on cue to a quota failure on the **spend-limit path** — the error code the S6 capture actually recorded (e.g. `project_spend_limit_exceeded`; [spk-04](sprint-spk-04.md)'s `*-quota-spend-limit.jsonl`), carried by m6a-02's text-path fixtures in `internal/coach/interview/replay/testdata/`. S6 has no prepaid-exhaustion fixture ([t6 §4](../research/t6-realtime-interviewer.md#4-session-state-machine--the-owners-failsafes): the probe arbitrates there), so `credit_balance_exhausted` stays covered by m6a-02's classifier tests, not by this e2e; **one injectable clock** drives coach's FSM and sweeper (m6a-01's clock; add a test-only option if it's missing). Never sleep: advance the clock.

| Scenario | Steps | Must hold |
|---|---|---|
| **A — the exit** (45-min rail, 1×) | setup → opt-in + consent → cap $1.00 → pre-flight probe → live → turns in every phase, snapshots, 2 Runs, one `give_hint` → at 23:41 the provider returns the spend-limit fixture's quota code → **grace** (clock frozen, a deterministic checkpoint, editor read-only, 30 s probes failing) → +5 min → **paused** → +3 h → **resume**: probe ok → brief (1 provider call) → a second brief request makes **no** provider call → live from 23:41 → rail done → **wrapping** (debrief ≤ 3 min) → **finished** → finals ensured, evidence aggregate only → review → proposal → re-propose once (a second → 409) → accept unchanged | **exactly one** `ScoreMock` (assessment `scored`, one `mock_completed` outbox row; a second accept replays); `scored_by = ai-byo` (clean session, catalog model that passed the twin gate); while paused coach `/chat` works and the item is withheld on lists; the trend includes it; the public profile's mock count rises by one, no score shown |
| **B — `incomplete`** | pause → +24 h 1 min → sweeper → a gateway read (m6a-03 mirrors terminal states lazily on read) | coach and assessment `incomplete`; no `ScoreMock`; excluded from trends; the partial-view data present; **zero** provider calls after the pause without a click |
| **C — exposure** | pause → an arena reveal on the mock's item → resume → finish → accept unchanged | `pause_exposure` set; `scored_by = self`; the caveat on the proposal |
| **D — cap** (optional) | a cheap cap reached | 85 % → `wrapping`; 100 % → `finished(cut_short=cap)` |

Across all: fake spend ≤ the cap; transcript and snapshot sizes within their caps; a log-capture assertion that no transcript text, code or key material appears in any service log.

### 6 · Tag the `v2.0.x` patch [X]

Run the **release checklist** (see *Release*) from `main` after this sprint's PR is merged. Title **`v2.0.N — v2.1 build · M6a (dark)`**. Release notes: the text interviewer (setup, live HUD, grace/pause/resume, debrief and explicit-accept scoring, accessibility settings), Mock-v2 for the cohort, judge mock Runs, bounded interview streams; **no change for accounts outside the owner/tester cohort**; `/api/v1` additive only.

### 7 · Post-tag verify on prod + coach memory read [H + X (+ I)]

By looking (D34), read-only over `ssh sujaykumar-vps`:
- `/xlearn/api/v1/healthz` reports `2.0.N`; `k3s kubectl get deploy -n xlearn -o wide` shows the new images; every `xlearn-*` ImagePolicy's latest = the tag; HelmReleases Ready.
- Smoke: login, the dashboard and coach (a chat streams); as the owner, `/dsa/mock` shows Mock-v2 and the setup reaches the pre-flight (don't start a paid interview here — the owner's post-ship dogfood does, task 8); a non-cohort account (a tester set to `learner`, or reasoning from the gate test if none exists) would see v1. An agent never enters credentials: use an already-signed-in owner browser session; without one, run the credential-free checks and record "owner login smoke pending" as a pending-smoke note in status.md.
- **Coach memory:** `k3s kubectl top pod -n xlearn` for coach vs its limit (128 Mi until [mi-13](sprint-mi-13.md) sets 256 Mi). If the working set sits above **70 %** of the limit at idle or climbs past it during the owner's dogfood (task 8) — [t6 §3](../research/t6-realtime-interviewer.md#3-architecture--media-path)'s coach split-out trigger ("coach above 70% of its memory limit") — record the crossing as that trigger in the Decisions log beside the resize decision, and open an infra PR raising coach's limit **checked against the memory-sum rule** ([ADR-0035 §5](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#5-capacity-the-memory-sum-rule-triggers-and-ordered-responses); run `host-verify --cluster`), merged on its own — or pull mi-13's coach-sizing task forward. Record the numbers either way.
- `ssh sujaykumar-vps 'bash -s -- --cluster --expect-sandbox --nats-stage=<live stage recorded in status.md>' < ../infra/hack/host-verify.sh` green (read-only; the memory sum after the tag). The live stage is `n4`, or `n3` if [mi-11](sprint-mi-11.md) reverted N4 to the monthly window; `--expect-sandbox` because every run after the October host window passes it ([mi-09](sprint-mi-09.md)).

### 8 · Owner dogfood: record the owner event [X]

**A post-ship owner event (`ev-m6a-dogfood`), non-blocking.** The session doesn't wait for it: task 9's record PR adds `ev-m6a-dogfood` to status.md's owner events (that is this task's ✅), and _Overall_ ✅ and the M6a milestone ✅ rest on the session's own evidence (the exit e2e, the tag verify). The event gates nothing.

After the tag, the owner runs **one real ~45-minute text mock on production** with his own `interview` key: consent and cap, one **pause** ("Save & resume later") and a **resume** with the paid brief, Finish, then accept or edit the proposal — **one score**. When the owner reports, record: wall time, spend vs the estimate (the provider console), pauses, whether the event stream held through Traefik across its 50 s reconnects, anything wrong. Anything it finds becomes a follow-up PR; a blocker is fixed forward with another patch (never move or re-push a tag).

### 9 · Record [X]

The sprint PR merged before the tag, so these records land in a **post-tag docs PR** (`docs(v2): record v2.0.N — M6a dark`, branch `docs/m6a-06-record`): CI green, squash-merge, then `git checkout main && git pull`. It updates [`../status.md`](../status.md): the M6a milestone row → "✅ dark in v2.0.N (exit: e2e <date>); GA at v2.1.0" (on the session's own evidence; the owner's dogfood doesn't hold it); the owner events: `ev-m6a-dogfood` (after the tag; non-blocking, task 8); milestone → tag → floor (floor unchanged; no snapshot, not a contract, erase or GA tag); the Artboards rows AB26–AB28 → "frozen (PR #, date) · consumed by m6a-06"; the flag inventory (no new env flag; the interview cohort gate is a T-1 constant flipped at v2.1.0); the P0 audit summary and any v2.1.0 blockers; the coach memory numbers and the host-verify result (task 7); the `kubectl exec` of `coach admin interviews --live` (a sanctioned manual path, logged). **The owner's dogfood** comes after the session: its result (and `ev-m6a-dogfood` ✅) lands in a second small docs PR (`docs(v2): record M6a dogfood`) when the owner reports, or in the next sprint's status update — the first PR's description names which.

## Acceptance criteria

- [ ] **M6a exit:** a 45-minute text mock survives pause and resume and **scores once** — the e2e scenario A green in CI (the owner's production dogfood follows as the post-ship owner event `ev-m6a-dogfood` and doesn't gate this).
- [ ] Every AB26, AB27 and AB28 frame matches the frozen board at 1440 px and 390 px (screenshots in the PR).
- [ ] Grace → paused → resume works as specified (free state at once; the brief only on a click and cached per pause; "Still no credit" charges nothing); after 24 h → `incomplete`, unscored, out of trends, no AI call without a click (scenario B).
- [ ] The proposal needs an **explicit** accept (no auto-submit); re-propose once; the server's `scored_by` label shown; `pause_exposure` → `self` (scenario C).
- [ ] Accessibility preferences persist, are erased with the account, and set the setup's default multiplier; the server scales the rail and timers.
- [ ] The P0 audit table is complete; every gap is fixed or recorded as a v2.1.0 blocker.
- [ ] `v2.0.N` is tagged and verified per the checklist (incl. **no live interviews** before the tag); status.md updated.

## Release

**Tag `v2.0.N`** — the next free **patch** ([ADR-0034 §1.1](../../adr/0034-v2-release-labelling-gating-and-rollback.md#11-scheme): "From `v2.0.0` on, the minor moves only at a GA flip"; [§1.6](../../adr/0034-v2-release-labelling-gating-and-rollback.md#16-indicative-tag-timeline): "v2.0.x … M6a/M6b dark"). M6a stays dark behind the cohort gate; [m6b-04](sprint-m6b-04.md)'s `v2.1.0` is the interviewer GA.

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
- **Next free version:** a **patch** `v2.0.N` (major 2 = `.release-line`); never `v2.1.0` (the minor must not move, ADR-0034 §1.1). Content-wave patches from peer sessions may have taken numbers — check.
- **ACL line:** expected **n/a** — mock evaluations stay silent and M6a adds no stream or consumer. Confirm with `git diff <last tag>..HEAD -- internal/platform/events/topology.go`; if anything changed, its ACL PR merges first.
- **New service image:** n/a (no new service; judge, coach, gateway, assessment are existing images).
- **Contract / snapshot lines:** expected **n/a** — M6a migrations are expand-only. Confirm: `git diff <last tag>..HEAD -- '**/migrations/*.sql' | grep -n 'xlearn:contract'` is empty; if not, rehearse in compose, mark the floor, run `host-verify --cluster`, take the snapshot.
- **No live interviews:** `ssh sujaykumar-vps 'sudo k3s kubectl exec -n xlearn deploy/xlearn-coach -- coach admin interviews --live'` reports **`live=0`** (no interview lines; m6a-01's output ends with `live=N`) right before the tag (the owner's dogfood must not be running); log the exec in status.md.
- **NetworkPolicy line:** m6a-01 task 7's recorded outcome (none expected: `pause_exposure` goes through the gateway; any callee PR it did open is merged); m6a-04 and this sprint add no caller.
- **Flags:** none added.

**Rollback** ([ADR-0034 §4](../../adr/0034-v2-release-labelling-gating-and-rollback.md#4-rollback)): **R-c** (revert + the next patch tag) is the default; **R-b** may narrow the policies to `!=2.0.N` (the floor is unchanged, so the previous patch runs on this schema). The blast radius is the owner/tester cohort. Check `coach admin interviews --live` before any rollback: in text mode a short coach roll is usually survived ([m6a-01](sprint-m6a-01.md): the interview stays `live`, the clock is in Postgres); a roll longer than the 30 s heartbeat window leads to `interrupt(network)` → grace → paused, which is recoverable.

## Definition of Done

CI green (web typecheck/lint/tests/build, bundle check, `go test ./...`, `sqlc diff`, OpenAPI drift and route-enumeration tests, the exit e2e) · the PR merged, then `v2.0.N` tagged, deployed by Flux and verified by looking (no hand `kubectl apply`) · the post-tag record docs PR merged (task 9), with `ev-m6a-dogfood` listed in status.md's owner events (task 8; its result lands later where that PR names, and gates nothing) · acceptance criteria met · statuses updated (this file + [`../status.md`](../status.md)) · no open PR left behind · local `main` synced in xlearn (and `../infra` if task 7 opened a PR).

## Risks / watch-outs

- **The patch must not move the minor** (ADR-0034 §1.1): `v2.0.N`, never `v2.1.0`; the cohort gate stays on.
- **Tagging mid-interview.** Every tag rolls coach, and there is no make-before-break yet (M6b). In text mode a short roll is usually survived (m6a-01's SIGTERM checkpoint; the clock is in Postgres; the sweeper interrupts only after 30 s without heartbeats); a long one leads to `interrupt(network)` → grace → paused. The ADR-0034 §6 rule stands: check `--live` reports `live=0` right before the tag, and don't tag while the dogfood runs.
- **Coach at 128 Mi.** M6a adds streams, in-memory snapshots and brain calls; measure (task 7) and resize through the memory-sum rule if needed, rather than waiting for mi-13.
- **A long e2e.** Drive time with the injected clock; a real-time 45-minute test is never acceptable in CI.
- **Grace copy vs reality.** Top-ups rarely register within 5 minutes; the copy sets expectations and pausing is the normal, cheap outcome ([t6 §14](../research/t6-realtime-interviewer.md#14-risks)).
- **The label is the server's.** `ai-byo` needs every condition of [t6 §6](../research/t6-realtime-interviewer.md#6-assessment) step 5; the UI never infers it.
- **Audit creep.** Fix only small gaps; larger ones become explicit v2.1.0 blockers so this sprint still lands and tags.
- **Parallel patch tags.** Content waves (D6) and fixes also cut `v2.0.x`; re-check the next free patch immediately before tagging.

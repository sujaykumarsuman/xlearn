# xLearn v2 — Product Requirements Document

> **Status:** Draft (all v2 topics T0–T7 settled; next: the build plan) · **Owner:** @sujaykumarsuman ·
> **Last updated:** 2026-09-24
> **Builds on:** the v1 PRD ([`xlearn-prd.md`](xlearn-prd.md)), which stays authoritative except where
> amended below. **Design and feasibility:** [`../v2/feasibility.md`](../v2/feasibility.md). **Decisions:**
> ADR-0026 onward in [`../adr/`](../adr/).

## 1. Why v2

v1 (live, v1.5.2) enforces the learning **method**: gated stages, five-touch revision, the mistake
journal, timed mocks and a BYO-key coach. But every correctness signal in it is **self-reported**:
- the outcome picker;
- the revision auto-score checkboxes;
- the mock rubric sliders;
- the mistake categories.

v1 also serves one course only. v2 makes the loop **evaluated** and **multi-course**:
- **Evaluated:** a real judge and AI evaluation produce the signals the method runs on.
- **Multi-course:** each course brings its own solving ground and evaluator, while the method stays common.

## 2. Goals (v2)

| # | Goal | Success signal |
|---|------|----------------|
| V1 | **Per-course extensibility.** Navigation, content, solving ground and evaluator vary by course. Today, Roadmap, Revision, Mistakes, Mock and Progress stay common. | A course that reuses existing answer and grader kinds ships as content plus a manifest only, with no code change ([ADR-0026](../adr/0026-per-course-extensibility-model.md)). |
| V2 | **Online judge.** Read a full statement, pick a language, **Run** on sample tests, **Submit** on the full hidden set, and see per-test results. | A DSA item with tests is graded by execution. Its outcome is derived by the server, not picked by the learner. |
| V3 | **Evaluated learning loop.** In the guided course, the platform AI analyses each submission, which yields structured feedback, pre-filled mistakes and revision scheduling, **including for unsolved attempts**. | Every concluded attempt produces a learning signal with provenance (`self` / `auto` / `ai`). The mistake category is pre-filled. |
| V4 | **Courses beyond DSA.** Other judge types exist: a system-design canvas with written answers, MCQ/fill-in and an AI rubric; the SQL runner; the Go race-detector runner. | A second course ships on the shared frame (after the judge milestone). |
| V5 | **Two-tier AI.** A **platform key** (the owner's budget) covers limited platform tasks. The **user's BYO key** covers coach chat and live mock interviews. | Platform spend is metered and capped. Running out of budget degrades to self-report and never blocks. (Detail: T5.) |
| V6 | **Live AI mock interviewer (M6 · v2.1).** A 45–60 minute interview with **two-way voice** (OpenAI key; text mode for any key). The AI **shares the session like a second interviewer**: it follows the on-screen answer (editor, canvas, fields) as it changes and steers the interview from it. It assesses **what is said and built**, never voice, face or emotion. It ends with improvement areas. **No video goes to the AI.** It runs on the BYO key, with a 5-minute top-up grace, a pause of up to 24 h, and a resume summary. | Ships in v2.1, after a voice spike (S6). Detail: T6 / ADR-0032. |
| V7 | **Richer public dashboard.** `/xlearn/u/<username>` shows per-course stats, grade provenance and judge stats, with per-course visibility. | The public payload stays non-PII (ADR-0024/0025). (Data: T1; tasks: T7.) |

## 3. Non-goals (v2)

- A multi-tenant org or classroom product, billing or subscriptions. (Unchanged from v1.)
- A cross-course optimizing scheduler. The Today planner is a deterministic budget split (§5.4).
- Per-course revision intervals or per-course grade scales. The method's ladder and grades are universal (§5.1).
- An in-app authoring CMS. _(Revisit in T1: content source and authoring pipeline.)_
- No alerting or metrics stack in v2; the owner monitors via landscape and kubescope (D34).
- Real learners in v2. v2 is used by the owner (and testers) only; invited learners arrive in v3, by the owner's choice (D35, §5.7).

## 4. Personas

The v1 personas carry over. In v2 the owner is also the **course author** for six courses (about
345 items needing statements, tests, answer keys, rubric anchors and revision probes). Content
authoring, not code, is the critical path.

## 5. The extensibility model (from T0 · [ADR-0026](../adr/0026-per-course-extensibility-model.md))

- **A course is data.** A settings-only **course manifest** is compiled into every image. It covers:
  - nav labels, timers and the grading strategy with its thresholds;
  - the revision format per level band;
  - the mistake taxonomy;
  - the mock rail, rubric and targets;
  - `est_minutes`, public-stats picks and the coach persona.
- **A capability is code.** Answer widgets are chosen by **part type** (`code`, `text`, `choice`/`blank`, `canvas`; `audio` later). Graders are chosen by **kind** (`code`, `key`, `ai_rubric`, `composite`, plus `analyzer`). Neither is chosen by course.
- **judge produces evidence. practice decides.** judge returns passed / failed / inconclusive plus checks. practice alone turns evidence into a grade and emits the learning signal. Everything downstream is course-blind.
- **Private evaluation data** (hidden tests, answer keys, rubric anchors) is readable only by the judge. It is never in the public repo, the images, or any learner-facing API.
- **Routes.** Courses live at `/xlearn/<course>/…` and profiles at `/xlearn/u/<username>`. A course slug may not equal a static top-level segment (`u`, `auth`, `settings`) or a gateway-owned path (`api`, `assets`, `healthz`, `readyz`, `.well-known`).

### 5.1 Method: universal vs per course

| Universal (every course) | Per course (manifest) |
|---|---|
| Revision ladder **Day 1 · 3 · 7 · 21 · 45**; a fail resets to Day 1 | Touch **format** per level band (e.g. early touches are short recall checks; Day 21/45 run the full format under mock conditions) and its timers |
| Grades **Clean · Rough · Assisted · Miss**, plus a passed flag and an optional score | The grading strategy (from a closed set) and its thresholds |
| Stage roles: attempt (hard, server-timed limit) → hint → conclude; the solution unlocks after conclusion or give-up. No re-implement stage (T4/D16) | Time limit, hint-unlock time, grade windows, labels |
| Mistake mechanics (one open entry per item; closes after 2 clean revisits); core categories: misread, time management, communication | Additional categories |
| Mock lifecycle (server timer, score-once) | Rail, rubric (N dimensions × 1–5), targets, item pools; or no mock |

### 5.2 Amended requirements

- **R-SR1 (amended).** Every **concluded, counted** attempt on a revisable item, **of any grade including miss or give-up**, is scheduled for Day 1·3·7·21·45 from the moment its grade was determined (`anchor_at`, T4), and a below-Clean grade also opens a Mistake Journal entry. _v1: scheduled only from the first Clean solve, so below-Clean problems were never revised and their mistakes could not close._ New courses start on this rule. DSA switches in the M2 release.
- **R-SR2 (generalized).** A touch is re-solved under the course's touch format. It **passes** when the evaluation passes and every required criterion is met. For DSA the criteria stay: pattern named in under 2 minutes, correct within the timer, complexity stated.
- **R-OL1 (generalized).** The outcome is **derived by the server** from evaluation evidence plus stage and timer facts, via the course's grading strategy. Self-report stays available per course (the manifest's self-report policy) until that course's items all have evaluators.
- **R-MK2 (generalized).** The mock rubric is per course (N dimensions × 1–5, stored as total and max). DSA keeps its 7 dimensions (out of 35).

### 5.3 Screens

- **Course-scoped:** Roadmap, Week, Concept, Problems, the solving workspace, Revision (with an all-courses toggle), Mistakes and weak area, Mock, Progress.
- **Account-wide:** Catalog and the agenda, Settings, onboarding, the public profile header (streak and heatmap).
- **New screen patterns (need v2 artboards):**
  - the workspace's part widgets (code editor with Run/Submit and test results; canvas; text; MCQ/blank);
  - the evaluation and feedback panel;
  - per-course nav;
  - the interviewer.

### 5.4 Today and budget across courses

- **One study budget per account, in minutes.** Each item format has an estimated duration from its course manifest.
- **Reviews come first.** Due touches from **all** enrolled courses come first, oldest first.
- **Overdue reviews block new work only in their own course.**
- **The remaining minutes are split across active enrollments.**

### 5.5 Content, arena, profile, privacy (from T1 · [ADR-0027](../adr/0027-content-evalpack-and-user-data-model.md))

- **R-CT1 (content rights).**
  - Every statement, example, test and editorial is written from scratch.
  - LeetCode and NeetCode appear only as outbound links and are never fetched.
  - Every item records its `provenance`.
  - Content and code are **MIT**. The private eval pack is proprietary.
- **R-CT2 (answer secrecy).**
  - Hidden tests, answer keys and rubric anchors are never exposed to learners.
  - Feedback on hidden cases is the passed count plus the first failure class only.
  - Hints and editorials go live only once the owner has stamped them.
- **R-AR1 (Problems arena).** The arena is untimed study.
  - **Submits are kept as a per-problem submission history.** Runs on sample tests are not kept.
  - It has its **own done marker, separate from course completion**.
  - It has an optional **manual timer, off by default**.
  - Arena activity never counts toward course progress, grades or revision.
- **R-PF5 (course attempt).** Starting a guided-course attempt starts its **server-authoritative timer** and opens the full question view with the coding area.
- **R-PP1 (public profile).**
  - Each enrolled course shows its stats only if visible. Visibility is on by default except for behavioral.
  - The header totals (solved, streak, heatmap, mocks) count visible courses only.
  - Grades show their provenance (self / judge / AI).
  - Code, answers, feedback, mistakes and notes are never public.
- **R-AC5 (account deletion).**
  - A learner can delete their account. Every service removes that learner's data.
  - There are no off-node backups in v2 (D12). If a Hostinger snapshot is ever restored ([ADR-0034](../adr/0034-v2-release-labelling-gating-and-rollback.md) R-d), the owner re-runs through the admin CLI every erase completed after the snapshot.
- **v2.x content scope** (D6; since T7 the scope of the v2.x *line*, delivered in waves; `v2.0.0` doesn't wait for it, D35).
  - All 151 DSA items at the self-report tier.
  - Full test packs for weeks 1–4.
  - A ~10-item second-course pilot after the judge ships.
  - In `v2.0.0` itself: the pilot course and the 14 DSA packs M3 needs.

### 5.6 Judge, timing and AI suggestions (from T4 · [ADR-0029](../adr/0029-judge-contract-and-learning-signal.md))

**Amended problem flow** (v1 PRD §6.1):
- **R-PF1 (amended).** Starting an attempt shows the **full statement** and the solving workspace immediately. The **hint unlocks at 15 minutes** (course default). The solution unlocks only after the attempt concludes, or on give-up.
- **R-PF2 (amended).** Revealing the solution before passing **is a give-up** and grades as Miss.
- **R-PF3 (amended).** The attempt timer is **server-authoritative and a hard limit**.
  - The DSA default is **45 minutes** per problem. Per-problem budgets will come from a separate research session.
  - **No passing submit by the limit concludes the attempt as Miss.** It is re-attempted through the revision schedule, starting on Day 1.
- **R-PF4 (replaced).** There is no re-implement stage.
  - **A passing submit finishes the attempt.**
  - After a failure, the solution unlocks for study, and the Day-1 revision is the from-memory re-attempt.

**Grading** (server-derived):

| Grade | DSA default rule |
|---|---|
| **Clean** | Pass within 20 minutes, with no hint, no coach and at most 3 failed submits |
| **Rough** | Pass within 45 minutes, without hint or coach |
| **Assisted** | Pass after the hint, **or with AI-coach help during the attempt** |
| **Miss** | Timeout or give-up |

The UI warns before a hint or a coach message caps the grade.

**Judge:**
- **R-JG1.** Run on sample and custom inputs shows full detail.
- **R-JG2.** Submit runs the hidden tests. The learner sees:
  - the number of hidden tests passed;
  - a single "too slow on large inputs" flag;
  - the first failure type.

  Hidden inputs are never shown.
- **R-JG3.** Compile errors don't count as failed submits.
- **R-JG4.** Mistake categories and up to 3 concepts to revise are pre-filled from the evaluation. The learner can always change them.

**AI suggestions:**
- **R-AI1.** An AI-graded result (system design, LLD, behavioral, and recall answers matched loosely against a key) is a **suggestion with its reasoning**: grade, mistake, notes, concepts. The learner can:
  - **accept** it;
  - **edit** it (not above what the timer and hint facts allow; labelled as self-set);
  - request **one fresh re-grade**.

  **Submitting** the result triggers the mistake entry and revision scheduling. An untouched suggestion auto-submits after 24 hours.
- **R-AI2.** **Manual entry is always available** when AI is unreachable, unavailable or over budget.
- **R-AI3.** The AI also reviews **passing** solutions. If the logic can be improved, it attaches **pointer notes to the problem**, shown on later attempts and revisions, and may suggest an **optional** revisit. This never changes the grade or the required revision schedule.

- **R-JG5 (languages).** At launch, learners can solve code problems in **Go, C++ or Python**. Starters are generated from each problem's signature. Time limits are scaled per language.

**Platform AI and the coach** (from T5 · [ADR-0031](../adr/0031-platform-ai-and-two-tier-keys.md)):
- **R-AI4.** "xLearn AI (included)" analyses and grades with the owner's key. It needs no learner key.
- **R-AI5.** A monthly allowance applies, shown to the learner as a percentage. When it is exhausted, xLearn AI pauses and manual entry takes over.
- **R-AI6.** Accepting an invite asks for two separate, unticked consents: xLearn AI may review graded work, and it may review passing solutions. Behavioral answers need their own opt-in. All three can be switched off in Settings, which falls back to manual grading.
- **R-AI7.** Learners confirm they are 18 or older.
- **R-AI8.** The privacy notice says content is processed by Anthropic outside India, may be kept up to 30 days, and is not used for training.
- **R-AI3 (refined).** The AI reviews **course passes**. It also reviews **revision passes whose solution differs materially** from the last reviewed one; if that solution is correct but improvable, the revision item is marked "correct, with improvements" and gets pointer notes. Neither changes the grade or the revision schedule.
- **R-AC6 (coach).**
  - Chatting with "Your AI coach (your key)" **about the problem you're attempting**, during a counted attempt, asks for confirmation and caps that attempt at **Assisted** (per problem).
  - The coach is off during revision touches and mocks.
  - Learners see their own-key usage.

**Mock interviewer** (from T6 · [ADR-0032](../adr/0032-realtime-ai-mock-interviewer.md); ships in v2.1):
- **R-MI1.** The learner chooses a **text or voice** interview. Voice needs an OpenAI key and Chrome or Edge.
- **R-MI2.** The interviewer is disclosed as an AI. It watches the answer widgets, not the learner, and asks follow-ups based on the work in progress.
- **R-MI3.** **Top-up grace.** If credits run out, the interview freezes. The learner gets 5 minutes to top up, and it resumes where it stopped.
- **R-MI4.** **Pause.** Otherwise it pauses, resumable within **24 hours** (at most 3 pauses). Resuming shows the saved state immediately, plus an optional AI summary paid from the learner's key.
- **R-MI5.** **Expiry.** A pause not resumed within 24 hours becomes **incomplete**: unscored and left out of trends.
- **R-MI6.** **Scoring.** The AI proposes rubric scores with quoted evidence. The learner **explicitly** accepts or edits them before they're saved.
- **R-MI7.** **Public profile.** It shows **how many** mock interviews were done, never their scores.
- **R-MI8.** **Retention.** Audio and video are never stored. Transcripts are deleted 30 days after scoring unless the learner chooses to keep them.
- **Future.** Peer-to-peer mock interviews between learners, with video.

**Arena and canvas:**
- **R-AR1 (amended).** The arena is **unrestricted**. It isn't locked during live attempts or reviews. An early solution reveal is recorded but doesn't cap the course grade.
- **R-CV1.** System-design and LLD diagrams use an **Excalidraw** canvas with typed shapes. Freehand drawing is allowed but not graded.

### 5.7 Audience, admission and releases (from T7 · [ADR-0033](../adr/0033-invite-only-admission-and-owner-admin.md) · [ADR-0034](../adr/0034-v2-release-labelling-gating-and-rollback.md) · [ADR-0035](../adr/0035-v2-operations-nats-auth-limits-capacity.md))

**Audience and admission:**
- **R-AD1 (audience).** v2 is used by the owner, plus owner-minted testers, only. Real learners are invited from **v3**; that is the owner's choice, not a limit of v2. The whole learner gate below still ships in v2 and is rehearsed on production with a tester (D35).
- **R-AD2 (signup closed).** Public signup has been closed since v1.5.2 (2026-09-24); sign-in keeps working. GitHub sign-in never links into an account that has a password: the learner signs in with the password, then connects GitHub from Settings.
- **R-AD3 (invite links).** The owner mints **single-use invite links**. A link expires after 7 days by default (at most 30), can be revoked, and works for both email and GitHub sign-up. Invalid, expired and used links show the same error. Without an invite, no one can sign up (only the owner's admin CLI can create `tester` accounts, R-AD6). xLearn sends no email; the owner shares the link. The invite code is removed from the address bar once read.
- **R-AD4 (acceptance).** Before using the app, a new account confirms it is **18 or older**, accepts the current **privacy notice** (versioned; it states the data-loss window of having no off-node backups) and confirms its **region**. From M4 it also sees the two unticked AI consents (R-AI6).
- **R-AD5 (seat cap).** Learner accounts are capped by a **seat cap** (15 to start, set by the platform-AI budget). The owner and testers don't take seats. The cap is re-sized from M4 data before any opening.
- **R-AD6 (tester role).** The owner can create **tester** accounts to exercise non-owner paths (acceptance, erase). The invite round-trip is run by a tester in a fresh browser; it creates a learner account, which then erases itself. Testers can erase their own account on the web. The owner account can only be erased from the admin CLI.
- **R-AD7 (admin).** Admin actions (invites, seats, list, suspend, reactivate, set role, revoke sessions, erase) are an **owner CLI run inside the cluster**, and every action is audited. There is **no web admin page**. Roles live in the database and are never carried in the session token.
- **R-AD8 (no waitlist).** There is **no in-app waitlist form**. The sign-in page offers a `mailto:` "request an invite" link.

**Releases:**
- **R-RL1.** During the build, every milestone ships as a **1.x minor** release. A tag with the wrong major can never auto-deploy.
- **R-RL2.** **`v2.0.0` is the v2.0 GA** for the owner: it turns on the v2 defaults (judge, platform AI, the pilot course) once the infra track, M1–M4, the pilot and the learner gate are complete.
- **R-RL3.** **`v2.1.0` is the interviewer GA** (M6a + M6b). DSA evaluator-only (M5) is a later 2.x release.
- **R-RL4.** The HTTP API stays **`/api/v1`** throughout 2.x; changes are additive.

## 6. Scope by topic (filled as topics settle)

| Area | Topic | Status |
|------|-------|--------|
| Extensibility frame | T0 | ✅ settled (ADR-0026) |
| Content & data model, private content, authoring, public-dashboard data | T1 | ✅ settled (ADR-0027) |
| Object storage + off-node backups | T2 | ✅ settled (ADR-0028; backups deferred) |
| Judge types & the common evaluation contract | T4 | ✅ settled (ADR-0029) |
| Code sandbox / online-judge engine + cluster hardening | T3 | ✅ settled (ADR-0030; spike pending) |
| Platform AI + two-tier keys | T5 | ✅ settled (ADR-0031) |
| Live AI mock interviewer | T6 | ✅ settled (ADR-0032; M6 / v2.1) |
| Rollout, milestones, release labelling, admission, operations | T7 | ✅ settled (ADRs 0033–0035; plan: [rollout-plan.md](../v2/rollout-plan.md)) |

## 7. Open questions (routed to topics)

| # | Question | Topic |
|---|----------|-------|
| ~~Q1~~ | ~~Who has the final say on an AI-graded outcome?~~ **Resolved (D14):** an editable AI suggestion plus one re-grade; manual entry is the fallback. | T4 ✅ |
| ~~Q2~~ | ~~What happens to abandoned attempts?~~ **Resolved (D15):** the timer is a hard limit, so timeout = Miss, re-attempted via the schedule. | T4 ✅ |
| ~~Q3~~ | ~~Where does the platform LLM key live and meter?~~ **Resolved (T5):** in judge, as a WIF credential with a spend ledger. | T5 ✅ |
| ~~Q4~~ | ~~Where do public content and the private eval pack live? How is authoring done?~~ **Resolved in T1 (D5–D6):** public content stays in this repo; the eval pack goes in a private repo, shipped as a private image readable only by judge. Authoring: AI drafts, machines verify, the owner stamps. | T1 ✅ |
| Q5 | Which course goes second (candidates: system-design or go-concurrency; behavioral is not a pilot)? **Recommended (T7): go-concurrency; confirm at the pilot milestone's entry.** | Pilot (P) entry |
| ~~Q6~~ | ~~Which languages at launch?~~ **Resolved (D20): Go, C++ and Python at M3.** | T3 ✅ |
| Q7 | What is the ideal time budget per problem? The default is 45 minutes with the hint at 15; this needs its own research and analysis session. | Future session |

# Sprint ds-l-01 — Design L: invite acceptance ★, privacy notice, erase account (AB19 ★, AB20, AB21)

> **Milestone:** L — learner gate (design track)   ·   **Track:** design (parallel; order 7)
> **Prereqs:** none (no prerequisite beyond `depends_on`)   ·   **Unblocks:** [l-02](sprint-l-02.md) (AB21 frozen before L-E's UI) · [l-05](sprint-l-05.md) (AB19–AB20 frozen before the L-A/L-C front door)
> **Release action:** **land-and-sync; the merge is the design freeze** — the board PR squash-merges on CI green; no tag, nothing deploys (launching the prompt is the owner's approval, [D40](../feasibility.md#decisions-log-newest-first); the owner may review after the merge, and any change to a frozen board is a follow-up design PR)
> **Calendar:** week 1–2 (2026-09-25 → 10-09); the freeze lands with this session's merge, well before [l-02](sprint-l-02.md) starts (early November). No owner event (`ev-freeze-ds-l-01` is automatic at the merge and needs no tick) and no owner design hours (BP3).
> **Execute with:** [`../prompts/prompt-ds-l-01.md`](../prompts/prompt-ds-l-01.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | Board scaffolding (own files only; never `index.html` or `board.css`) | X | ⬜ |
| 2 | AB19 ★ invite acceptance (hero): auth-page states, invite errors, acceptance step | X | ⬜ |
| 3 | AB20 privacy notice / terms (agent-drafted notice text; ships as drafted, D40) | X | ⬜ |
| 4 | AB21 erase account (Settings) | X | ⬜ |
| 5 | Self-review checklist (run before merging) + screenshots (1440 px, 390 px) | X | ⬜ |
| 6 | Open the design PR (screenshots, frame lists, the ticked self-review checklist, "Decisions to confirm") | X | ⬜ |
| 7 | Freeze: squash-merge on CI green (the merge is the freeze) → status → sync `main` | X | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly. **Land and sync ([D40](../feasibility.md#decisions-log-newest-first)):** the board PR
> merges on CI green and the merge is the freeze, so this sprint records its own close-out. Once the PR number is known, a last
> commit on the PR branch sets tasks 1–7 and _Overall_ ✅ here (task 7: "frozen: merged in PR #N, <date>") and, in
> [`../status.md`](../status.md), this sprint's Sprint-board row ✅, the Artboards rows AB19, AB20 and AB21 → ✅ "frozen (merged,
> PR #N, <date>)", and the Snapshot's artboard count (`ev-freeze-ds-l-01` is automatic: no tick). If the
> merge slips to another day or fails after that commit, correct the rows in a follow-up docs PR merged the same way.
> Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] **Notice inputs known.** The inputs are **owner decisions D12 / D24 / D33** in the [feasibility decisions log](../feasibility.md#decisions-log-newest-first),
  recorded in ADR-0028 and ADR-0033 (Accepted) and ADR-0031 (still **Proposed** under BP2 — accepted at the WIF spike before M4 — but its
  §4 only records D24 and won't change with the spike), so nothing waits:
  - D12's data-loss window: with no off-node backups, up to about **7 days** of data can be lost with the node or its disk, and Hostinger's weekly images are the only safety net ([ADR-0028 §2](../../adr/0028-object-storage-and-backups.md#2-off-node-backups-deferred) and its Consequences, [feasibility D12](../feasibility.md#decisions-log-newest-first));
  - provider retention **≤ 30 days** (flagged content up to 2 years), **no training**, processing **outside India** (owner decision D24, as recorded in [ADR-0031 §4](../../adr/0031-platform-ai-and-two-tier-keys.md#4-retention-and-privacy-owner-d24), Proposed);
  - the notice contents and acceptance items of [ADR-0033 §6](../../adr/0033-invite-only-admission-and-owner-admin.md#6-the-acceptance-step-onboarding-step-0-l-a) and the `mailto:` rule of [§10](../../adr/0033-invite-only-admission-and-owner-admin.md#10-no-web-admin-no-waitlist).

_Informational, not a gate:_ `design-system/screens/v2/index.html` and `board.css` are written once by [ds-m1-01](sprint-ds-m1-01.md).
If they are on `main`, link `board.css` and use the file names the index links. If ds-m1-01's PR is still open, **do not create or edit
either file**: put the same `bd-*` chrome in each board's own `<style>` block and use the Task 1 file names (they match the index).

## Goal

Draft the three learner-gate boards — including the **AB19 invite-acceptance hero** — as static, preview-only HTML on
[`theme.css`](../../../design-system/theme.css), with every frame, state and final copy, so that:
- [l-02](sprint-l-02.md) builds the Settings erase section against a frozen **AB21** (freeze rule: AB21 before L-E);
- [l-05](sprint-l-05.md) builds the auth page's invite state, the acceptance step and the privacy page against frozen **AB19–AB20** (AB19–AB20 before L-A).

Per **BP3** (owner, 2026-09-24) agents draft every v2 board, heroes included. Per [D40](../feasibility.md#decisions-log-newest-first)
the board PR merges on CI green and the merge is the freeze; the owner may review afterwards, and any change to a frozen board is
a follow-up design PR. The rollout's "owner designs the heroes in Claude Design" ([rollout §9](../rollout-plan.md#9-artboards-by-milestone))
is superseded by BP3.

## Scope

**In**
- AB19 ★, AB20 and AB21 as static HTML under `design-system/screens/v2/`, one file per board.
- Every frame/state listed in Tasks 2–4, with final copy, a frame label citing its decision(s), and a behaviour-notes aside.
- AB20 carries the **full notice text**, drafted by the agent. Per D40 it lands as drafted and [l-05](sprint-l-05.md) ships it;
  the owner may revise it later with a content PR, and the owner's review of the notice is a
  [v3 opening gate](../rollout-plan.md#11-opening-gates-v3), not a step in any v2 sprint.
- 1440 px and 390 px screenshots of every board in the PR.
- Freeze = the board PR merged on CI green (D40); plus this sprint's own rows in `docs/v2/status.md` (Status note).

**Out**
- Any `web/` code → [l-02](sprint-l-02.md) (AB21) and [l-05](sprint-l-05.md) (AB19, AB20).
- The invite backend (`invite` table, redeem, `POST /api/invites/check`, `GET /api/v1/auth/config`) → [l-03](sprint-l-03.md); the boards only draw its states.
- The Settings AI-consent toggles (withdrawal) and the allowance meter → AB18 in [ds-m4-01](sprint-ds-m4-01.md); AB19 shows only the two consents at acceptance.
- Profile visibility toggles → AB22 in [ds-m2-01](sprint-ds-m2-01.md).
- Editing `design-system/screens/v2/index.html` or `board.css` → owned by [ds-m1-01](sprint-ds-m1-01.md) (written once).
- The L-exit rehearsal runbook → [l-04](sprint-l-04.md).
- Any web admin or waitlist surface: there is none (D33, [ADR-0033 §10](../../adr/0033-invite-only-admission-and-owner-admin.md#10-no-web-admin-no-waitlist)); AB23 is dropped.
- Shipping boards (preview-only) and owner design hours (BP3; any owner review happens after the merge).

## Tasks

### 1 · Board scaffolding [X]

Static HTML boards under `design-system/screens/v2/`, one file per board, named as the [ds-m1-01](sprint-ds-m1-01.md) index links them:
`AB19-invite-acceptance.html`, `AB20-privacy-notice.html`, `AB21-erase-account.html`.
- Each file links `../../theme.css`, then `board.css` when it is on `main`, plus the Google Fonts `<link>` the v1 boards use (Inter,
  JetBrains Mono). It reuses the tokens and components **verbatim**: dark "landscape console"; `ds-btn`, `ds-card`, `ds-field`,
  `ds-input`, `ds-seg`, `ds-toggle`, `ds-modal--sm|--md`, `ds-badge`, `ds-chip`, `xl-app`/`xl-side`/`xl-topbar`, `xl-brand`. Difficulty
  tokens where shown: Easy = `--ds-ok`, Medium = `--ds-warn`, Hard = `--ds-err`. Board-only layout CSS goes in the file's `<style>`, uses
  `--ds-*` tokens only (no new colours; danger styling = `--ds-err`), and never overrides a `ds-*`/`xl-*` rule. `theme.css` has no
  checkbox class: style native `<input type="checkbox">` board-locally with tokens.
- Every frame has a label (`AB19-F9 · Acceptance step`), the decision refs it implements, final copy (no lorem ipsum, no "TBD"), and
  a **behaviour notes** aside: gates, exact error codes (`403 acceptance_required`, `invite_invalid`, …), a11y (focus order, contrast
  ≥ 4.5:1 on text, keyboard, `aria-live` for status changes, reduced motion) and the < 1024 px layout. Each board ends with a
  `.bd-narrow` section (390 px) for its key frames.
- **Preview-only:** static HTML, no JS runtime (at most trivial static toggles), never imported by `web/`, never shipped. v1
  [`Auth.dc.html`](../../../design-system/screens/Auth.dc.html) and [`Settings.dc.html`](../../../design-system/screens/Settings.dc.html)
  are references only; draw from today's shipped `web/src/screens/Auth.tsx` and `Settings.tsx` structure (the Sign in / Sign up
  `ds-seg` pill, the Settings section rail), which have moved on from the `.dc.html` boards.
- **Touch only this sprint's own board files** (and their screenshots, plus this sprint's own rows in `docs/v2/status.md`).
  Parallel design PRs ([ds-m1-01](sprint-ds-m1-01.md), [ds-m2-01](sprint-ds-m2-01.md)) never conflict on `index.html`.

### 2 · AB19 ★ invite acceptance (hero) [X]

`AB19-invite-acceptance.html` — the pre-auth auth page in its v2 states and the acceptance step every account passes once.
Sources: [ADR-0033 §3, §5, §6, §10](../../adr/0033-invite-only-admission-and-owner-admin.md#5-invites-mint-and-redeem),
[ADR-0031 §4](../../adr/0031-platform-ai-and-two-tier-keys.md#4-retention-and-privacy-owner-d24) (consents),
[PRD §5.7](../../prd/xlearn-v2-prd.md#57-audience-admission-and-releases-from-t7--adr-0033--adr-0034--adr-0035) R-AD3/R-AD4/R-AD8 and R-AI6,
[t7 §2.3](../research/t7-cross-cutting-and-rollout.md#23-admission-design-identity-owns-it). Built in [l-05](sprint-l-05.md)
against [l-03](sprint-l-03.md)'s backend.

| Frame | Content and final copy | Decisions |
|---|---|---|
| F1 Auth · invite-only (`closed`, or `invite` with no code) | `GET /api/v1/auth/config` → `{signup: "closed"}`, or `{signup: "invite"}` with no stashed invite code (e.g. a visitor during the L-exit rehearsal without a link) — the same frame for both. **No Sign in / Create account pill**: a sign-in card only (email or username, password, **[Log in]**, **[Continue with GitHub]** for existing accounts). Below: "xLearn is invite-only." and a **[Request an invite]** link (`mailto:` to the owner's contact address from config, subject "xLearn invite request"). | D13, D33, ADR-0033 §10, R-AD8 |
| F1b Invite link while `closed` | `#invite=<code>` arrives while the mode is `closed`: the code is stashed and dropped from the address bar as in F2, but **not checked** (no `/api/invites/check` call). The F1 sign-in card plus a note: "Invites aren't being accepted right now. Sign in if you already have an account." **[Request an invite]**. No Create account. | ADR-0033 §5, §10 ([l-05](sprint-l-05.md)'s mode table) |
| F2 Invite link · checking | The SPA reads `#invite=<code>`, keeps it in `sessionStorage`, calls `history.replaceState` (address bar shows `/xlearn/auth`), then `POST /api/invites/check`. Banner skeleton "Checking your invite…" (`aria-live="polite"`). | ADR-0033 §5 (fragment, never logged) |
| F3 Invite link · valid (`invite` mode) | Opens on **Create account**. Banner: "You've been invited to xLearn. This invite expires on 12 Oct 2026." Email, password ("at least 8 characters"), **[Create account]**; "or" **[Continue with GitHub]**. Small print: "Next, you'll confirm you're 18 or older and accept the privacy notice." | ADR-0033 §5, R-AD3 |
| F4 `invite_invalid` (uniform) | For an unknown, expired, used or revoked code — at check or at redeem: "This invite link isn't valid. It may have expired, already been used, or been revoked. Ask the person who invited you for a new link." **[Request an invite]**. Create account hidden; Sign in stays. One frame, **identical for every cause**. | ADR-0033 §5, §9 (enumeration) |
| F5 `invite_required` | `/xlearn/auth?error=invite_required` (a new GitHub user without an invite, or an email signup with no code): "You need an invite to create an account. Open the invite link you were sent, or sign in if you already have an account." | ADR-0033 §5 |
| F6 No seats (`no_seats`) | Redeem refused because seats are full (the cap re-check at redeem; also the R0 seat-freeze lever): "xLearn is full right now, so this invite can't be used yet. It stays valid until 12 Oct 2026 — try again later, or ask the person who invited you." The error code is defined by [l-03](sprint-l-03.md); the board uses `no_seats` and flags it for confirmation. | ADR-0033 §3 |
| F7 Email already registered | The invite is **not** consumed (the transaction rolls back): "An account with this email already exists. Sign in instead — your invite hasn't been used." | ADR-0033 §5, §9 (accepted leak to invite holders) |
| F8 GitHub path with an invite | Behaviour frame: the Continue-with-GitHub form carries a hidden `invite` field (kept in the HttpOnly `oauthTx` cookie, 10 min); a **new** account redeems inside the create transaction and lands on F9; an **existing** account just signs in (invite untouched). If the invite names an email and GitHub's verified email differs → the **uniform F4 error** (conservative reading; listed under "Decisions to confirm"). | ADR-0033 §5, §2 (no auto-link into password accounts) |
| **F9 ★ Acceptance step** (onboarding step 0) | Full-screen, before the path step. Title "Before you start", stepper "Step 1 of 5" (the acceptance step joins today's 4 onboarding steps — path, budget, username, coach — in `web/src/screens/Auth.tsx`). Three required items and one optional group:<ul><li>☐ **"I'm 18 or older."**</li><li>☐ **"I've read the [privacy notice] (version 1, effective 1 Dec 2026) and agree to it."** — the link opens AB20 in a new tab.</li><li>**Region:** "Where do you live?" select, pre-filled **India** from the invite (default `IN`); helper "Some features depend on local law — for example, voice mock interviews."</li><li>Fieldset **"xLearn AI (optional)"**, both **unticked**: ☐ "xLearn AI reviews my graded work — grade suggestions and feedback on work you submit." ☐ "…and also reviews my passing solutions for improvement notes." (disabled until the first is ticked). Helper: "Off unless you tick them. If on, your work is sent to Anthropic, processed outside India, kept by them for up to 30 days and never used for training. Change this any time in Settings. With these off, you grade your own work."</li></ul>**[Continue]** · secondary **[Sign out]**. | ADR-0033 §6, ADR-0031 §4, R-AD4, R-AI6, D24 |
| F10 Acceptance · validation | Submit with a required box unticked: inline errors "Please confirm you're 18 or older." / "Please accept the privacy notice to continue."; focus moves to the first invalid item. **[Continue]** stays enabled (errors on submit, not a silently disabled button). | a11y |
| F11 Acceptance · owner/tester | Same form for accounts with no invite (the owner and CLI-minted testers pass it once too): region defaults to India; if onboarding is otherwise complete the stepper reads "One more step". **Consents reflect existing live grants:** an account that already granted a consent in Settings (the owner during M4) sees that box ticked — draw this variant (first consent ticked from a live grant, helper "On — you turned this on in Settings."); a box is never pre-ticked without a live grant, and unticking one withdraws it. | ADR-0033 §6, §7, ADR-0031 §4 ([l-05](sprint-l-05.md) task 4) |
| F12 Re-acceptance (`403 acceptance_required`) | Any non-onboarding API returns `403 acceptance_required` after a notice-version bump → the SPA routes here. Banner: "We've updated the privacy notice (version 2, effective 1 Mar 2027). Please review what changed and accept to keep using xLearn." **[What changed]** opens AB20-F2. Only the notice checkbox is asked again; 18+, region and the consents keep their recorded values (conservative reading; listed under "Decisions to confirm"). | ADR-0033 §6 |
| F13 Save failed | "Couldn't save your answers — check your connection and try again." Ticked boxes stay ticked. | — |
| F14 < 1024 px / 390 px | Auth card full width; acceptance items stacked; the consent fieldset below the required items; **[Continue]** pinned at the bottom. | — |

Behaviour notes must state: the code rides only in the URL fragment and is dropped from the address bar once read; invite errors never
say *why* an invite is invalid; the create-account path is hidden (not just disabled) while `closed`; the consents are unticked by
default, pre-filled only from the account's existing live grants (F11), and never pre-ticked without a grant (re-acceptance included); the acceptance is recorded with the notice version (`account_consent`/`accepted_at`)
and the gateway returns `403 acceptance_required` on every non-onboarding API until it is done; 18+ and region are attestations,
not verification (ADR-0033 Consequences). Keyboard: every checkbox reachable in order, Space toggles, Enter submits.

### 3 · AB20 privacy notice / terms [X]

`AB20-privacy-notice.html` — the versioned page at `/xlearn/privacy` (a new static SPA segment: `privacy` joins the course-slug guard,
[ADR-0033 §6](../../adr/0033-invite-only-admission-and-owner-admin.md#6-the-acceptance-step-onboarding-step-0-l-a)). Reachable before
sign-in (from the auth page and the acceptance step) and from Settings. Sources: [ADR-0033 §6](../../adr/0033-invite-only-admission-and-owner-admin.md#6-the-acceptance-step-onboarding-step-0-l-a),
[ADR-0031 §4](../../adr/0031-platform-ai-and-two-tier-keys.md#4-retention-and-privacy-owner-d24), [ADR-0028 §2](../../adr/0028-object-storage-and-backups.md#2-off-node-backups-deferred) (D12),
[ADR-0027 §6](../../adr/0027-content-evalpack-and-user-data-model.md#6-account-erase-v20) as amended by [ADR-0028 §3](../../adr/0028-object-storage-and-backups.md#3-amendments-to-adr-0027),
[ADR-0034 §4.3](../../adr/0034-v2-release-labelling-gating-and-rollback.md#43-snapshot-rule) (1-day snapshots),
[PRD §5.5](../../prd/xlearn-v2-prd.md#55-content-arena-profile-privacy-from-t1--adr-0027) R-PP1/R-AC5 and R-AI8.

| Frame | Content and final copy | Decisions |
|---|---|---|
| F1 Notice page (desktop, signed out) | Header "Privacy notice" · "Version 1 · effective 1 Dec 2026" · a table of contents. **The notice text** (agent-drafted; it ships as drafted, D40), one short section each:<ol><li>**Who runs xLearn** — one person, the owner, as a personal project; contact through the address in [Request an invite].</li><li>**What xLearn stores** — account (email, display name, username, password hash, a linked GitHub id), learning data (attempts, the code you submit, timings, grades, revision schedule, mistakes, mock scores), coach chats, your own AI keys (encrypted), region, your 18+ confirmation and your choices. One session cookie; no analytics, ads or tracking.</li><li>**How AI is used** — *xLearn AI* only if you opt in: your work goes to **Anthropic**, is processed **outside India**, may be kept by Anthropic for **up to 30 days** (flagged content up to 2 years) and is **never used for training**. *Your coach* uses your own OpenAI or Anthropic key under your account with them.</li><li>**What's public** — your profile at `/xlearn/u/<username>` shows stats for the courses you leave visible; never your code, answers, mistakes or notes. Toggles in Settings.</li><li>**Where data lives, and the data-loss window** — one server; **no off-node backups**; the hosting provider keeps a weekly image of the whole server. If the server or its disk fails, **up to about 7 days** of your data may be lost.</li><li>**Your choices** — the AI toggles, profile visibility, and erasing your account (what is erased; your username is held for 60 days; a weekly server image kept by the host may hold your data for **up to about 7 days**, and a pre-release snapshot for up to 1 day — if one is ever restored, the owner re-runs the erases made since it was taken; some internal event records keep a random account id, and older sign-up records also keep your display name, until the owner removes them on request).</li><li>**Age** — xLearn is for people 18 or older.</li><li>**Changes** — every version is numbered; after a change you're asked to accept again before continuing.</li></ol>Board annotation under the page (not page copy): "Notice text v1 as drafted by the agent — it ships as drafted (D40); the owner may revise it with a content PR (owner review: v3 opening gates)." | ADR-0033 §6, ADR-0031 §4, D12, D24 |
| F2 "What changed" block | Shown for version ≥ 2 at the top of the page and from AB19-F12: "What changed in version 2" with 2–4 plain bullets and the previous version's date. | ADR-0033 §6 |
| F3 Terms section | Same page, below the notice: "Terms" — xLearn is invite-only; one account per person; invites are single-use and not to be shared; the owner may suspend accounts; xLearn is provided as is, without warranty; course content is MIT-licensed. | D33, R-CT1 |
| F4 In-app view | The same page inside the app shell when signed in (reached from Settings); identical text. | — |
| F5 390 px | Single column; the table of contents collapses to a "Contents" disclosure. | — |

Behaviour notes: the page is public (no session needed); the version string and date come from one constant the acceptance step also
records; long-form text is ≥ 16 px with a readable line length (~70 ch); headings are real `h2`s for screen-reader navigation.

### 4 · AB21 erase account [X]

`AB21-erase-account.html` — the Settings erase section. Sources: [ADR-0033 §7, §9](../../adr/0033-invite-only-admission-and-owner-admin.md#9-erase-and-abuse-controls),
[ADR-0035 §4](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#4-limits-inventory) L8, [ADR-0027 §6](../../adr/0027-content-evalpack-and-user-data-model.md#6-account-erase-v20)
(as amended: no erase ledger, D12), [ADR-0034 §4.4](../../adr/0034-v2-release-labelling-gating-and-rollback.md#44-reversibility-by-step) (erase is irreversible),
[t1 §6.6](../research/t1-content-data-model.md#66-erase-path-none-exists-today-no-delete-apime-in-bffgo), PRD R-AC5 and R-AD6. Built in [l-02](sprint-l-02.md)
(`DELETE /api/me`). Placement: the last card of the existing **"Sign-in & security"** section of `Settings.tsx`'s rail (listed under
"Decisions to confirm": the alternative is its own rail item).

| Frame | Content and final copy | Decisions |
|---|---|---|
| F1 Erase card (tester, eligible) | Heading **"Erase account"**. "Permanently erase your xLearn account and everything in it. This can't be undone — xLearn keeps no off-node backups." A "What's erased" list: **Sign-in** (email, password, GitHub link, sessions) · **Profile and settings** (name, username, time zone, study budget, reminders) · **Learning** (attempts, timers, outcomes, revision schedule, mistake journal, weak areas) · **Mocks and progress** (mock sessions, scores, progress, your public profile) · **Coach** (your saved AI keys, coach chats). "Your username @asha is held for 60 days, then anyone can claim it." **[Erase my account…]** (danger button). | ADR-0033 §9, R-AC5, D12 |
| F2 Sign in again | Session older than 5 minutes (`/api/me` says it is no longer fresh, or the server returns `403 reauth_required`): "For your security, sign in again to erase your account. You'll come straight back here." **[Sign in again]** (signs out, then returns to Settings → Sign-in & security). | L8 |
| F3 Confirm modal | `ds-modal--md`, title **"Erase your account?"** Body: "This permanently erases your account, progress, revision schedule, mistakes, mock scores, coach chats and saved AI keys. Your public profile disappears immediately." Field: "Type your username **asha** to confirm" (accounts with no username type their email), mono input. **[Cancel]** (default focus) · **[Erase account]** (danger; enabled only on an exact, case-insensitive match). | L8 (typed confirmation) |
| F4 Mismatch | Server `400 confirmation_mismatch`: "That doesn't match your username." | L8 |
| F5 Erasing | Button "Erasing…", modal locked, `aria-busy`. | — |
| F6 Erased (signed out) | The auth page with a banner: "Your account has been erased. Your data is removed from xLearn's services within minutes; server images kept by the host may hold it for up to about 7 days. Your username is held for 60 days." | ADR-0033 §9, P11, D12 |
| F7 Owner refused | Card body, **no button**: "The owner account can't be erased from the web. Use the admin CLI: `identity admin account erase`." (`403 erase_owner_cli_only` if called anyway.) | ADR-0033 §7 (owner: never on the web) |
| F8 Tester-only note (learner before the L exit) | Card body, **no button**: "Erasing from the web is open to tester accounts for now. To erase this account, [contact the owner]." (`403 erase_not_available`.) [l-04](sprint-l-04.md) opens it to every non-owner. | ADR-0033 §7, §9 (testers at L-E) |
| F9 Other errors | `409 erase_in_progress`: "An erase is already in progress for this account." · `429`: "Too many requests — wait a moment and try again." · network: "Couldn't erase your account. Nothing was changed — try again." | L8 (1 open per account), L5 |
| F10 390 px | The card full width; the modal as a bottom sheet with the input above the buttons. | — |

Behaviour notes must state: the owner is always refused on the web (no XSS can wipe the owner); the fresh-session window is 5 minutes;
the typed value is checked on the server too; after success the session is gone and the public profile 404s at once (P11); what stays
(ADR-0027 §6 note; l-01/l-02 risk lists — the wording is listed under "Decisions to confirm"): "Some internal event records keep a random
account id, and older sign-up records also keep your display name, until the owner removes them on request" (identity's historic
`account_created` events carry `display_name`; the re-seal is on request); backups: "a weekly server image kept by the host may hold
your data for up to about 7 days, and a pre-release snapshot for up to 1 day; if one is ever restored, the owner re-runs the erases
made since it was taken" (D12: Hostinger's weekly images are the only safety net; ADR-0034 §4.2 step 3). Never claim the leftovers
"can't be linked back to you". a11y: focus trap
in the modal, Esc = Cancel, Enter never submits before the match, the danger button meets contrast on `--ds-err`.

### 5 · Self-review checklist (run before merging) [X]

The session's own gate before the merge (D40: nothing waits on the owner); its ticked result goes in the PR body.
Walk every frame against its cited decisions and the [rollout §9](../rollout-plan.md#9-artboards-by-milestone) L row
(AB19: 18+, notice, region, 2 unticked consents, `invite_required`, `invite_invalid`, no seats, `acceptance_required`, `mailto:`; AB20;
AB21: typed confirmation, "sign in again", what's deleted, cooldown). Check that:
- `theme.css` is linked (+ `board.css` when on `main`), never copied or overridden: `--ds-*` tokens only, no new colours, danger =
  `--ds-err`, difficulty tokens where shown; the canonical index file names are used;
- invite errors are uniform and never reveal why (F4), and no board shows an invite code anywhere after it is read;
- both consents are unticked by default everywhere, including re-acceptance, and ticked only where F11 shows an existing live grant;
- no board offers a web admin, a waitlist form or an owner erase button;
- the notice states Anthropic, ≤ 30 days, no training, outside India, the toggles and the ~7-day data-loss window, and AB20 §6 / AB21
  state the weekly-image (~7 days) and pre-release-snapshot (1 day) retention of erased data and the display-name leftover — no
  "no backups" or "can't be linked back to you" claims;
- text contrast ≥ 4.5:1 for every colour pair used.

Fix what fails, then re-check. Screenshot every board full-page at **1440 px** and **390 px** into
`design-system/screens/v2/shots/AB19@1440.png`, `AB19@390.png` (and AB20, AB21), each ≲ 500 KB (ds-m1-01's convention).

### 6 · Open the design PR [X]

Branch `design/ds-l-01`, conventional commit `docs(design): L boards AB19★ AB20 AB21` with the attribution lines, PR titled
"design: AB19★ AB20 AB21 (L)". The body: the screenshots (embedded from the branch), a frame list per board with decision cites, the
ticked self-review checklist (task 5), and **"Decisions to confirm"**. The list doesn't block the merge: each item states the default
the merge freezes (the boards as drawn), and the owner may revisit any item after the merge through a follow-up design PR (recorded
by the build sprint that implements it):
1. the no-seats error code `no_seats` (l-03 defines it);
2. a GitHub-verified email that differs from the invite's email → the uniform `invite_invalid`;
3. re-acceptance asks only the notice (18+, region and consents keep their values);
4. AB21's placement (last card of "Sign-in & security" vs its own rail item);
5. the typed confirmation value (the username; the email when there is none);
6. the "what stays" wording in AB21 and the notice text in AB20 (it lands as drafted, D40; the owner's review of the notice is a
   v3 opening gate, and a revision is a later content PR);
7. erased data in the host's **weekly server image for up to about 7 days** (and a pre-release snapshot for 1 day), with "if one is
   ever restored, the owner re-runs the erases made since it was taken" — after a whole-node loss the erase list may itself be lost,
   so this promise's wording is flagged; the wording as drawn is what the merge freezes (D12, ADR-0034 §4.2);
8. the **display-name leftover**: older `account_created` events keep the display name until the owner re-seals on request — drawn:
   keep the disclosure; the alternative is that [l-02](sprint-l-02.md) re-seals by default (ADR-0027 §6 note).

### 7 · Freeze: merge on CI green [X]

Once the PR number is known, push the status commit (the Status note). When CI is green (fix, then merge, on failure),
squash-merge: **the merge is the freeze** (D40), and it gates [l-02](sprint-l-02.md) (AB21) and [l-05](sprint-l-05.md)
(AB19–AB20). Never enable auto-merge. Then sync `main` (`git checkout main && git pull`). The owner may review after the merge;
any change is a follow-up design PR.

## Acceptance criteria

- [ ] Every frame listed for AB19, AB20, AB21 is present with final copy and states, and each frame cites its decision(s).
- [ ] Boards link `../../theme.css` and use only its tokens/components; no new colours; preview-only; `index.html`, `board.css` and `theme.css` untouched.
- [ ] Invite errors are uniform; consents are unticked by default everywhere and never pre-ticked without a live grant (F11); the owner has no web erase; the notice covers Anthropic, ≤ 30 days, no training, outside India, the toggles and the ~7-day data-loss window.
- [ ] The self-review checklist passed and is ticked in the PR body, with the eight "Decisions to confirm" (each item's frozen default stated).
- [ ] PR **merged on CI green** (the freeze) with 1440 px and 390 px screenshots; `docs/v2/status.md` changed only in this sprint's rows (ds-l-01 ✅, AB19–AB21 "frozen (merged)").

## Release

**Land-and-sync; the merge is the design freeze** — the board PR squash-merges on CI green; no tag. Nothing ships: boards are preview-only and
never imported by `web/`. Launching the prompt is the owner's approval ([D40](../feasibility.md#decisions-log-newest-first)), so
nothing waits on the owner; the owner may review after the merge, and any change to a frozen board is a follow-up design PR.
The merge is the freeze that gates [l-02](sprint-l-02.md) (AB21 before L-E) and [l-05](sprint-l-05.md) (AB19–AB20 before L-A).

## Definition of Done

Three board files (+ screenshots) merged on `main` (CI green; the freeze) · every frame present with final copy and decision
cites · the ticked self-review checklist and "Decisions to confirm" in the PR description · this file's Status all ✅ and this
sprint's `docs/v2/status.md` rows updated · no edits outside the three boards, their screenshots, this sprint file and those
status rows · local `main` synced.

## Risks / watch-outs

- **A board contradicting an Accepted decision.** For L the relevant ones are D12 (no backups), D13/D33 (invite-only, no web admin, no
  waitlist), D24 (retention), D35 (owner-only v2) and ADR-0031 §4 (consents; the ADR is Proposed under BP2, §4 records D24); the generic list (D15/D16/D17/D18/D27/D31) doesn't bite
  here. Cite the decision per frame; when a decision is ambiguous, draw the conservative reading and flag it (Task 6 list).
- **Uniform errors leaking.** A frame that says "this invite expired" instead of the single `invite_invalid` text breaks ADR-0033 §5.
- **Pre-ticked or bundled consent.** The two AI consents must be separate, unticked, optional and withdrawable (t5 critique fix 6).
- **Promising behaviour that isn't built yet.** AB19's consents only matter once M4 ships (they land with [l-05](sprint-l-05.md) in
  v1.17.0, after M4); AB21-F8's learner note disappears at [l-04](sprint-l-04.md). Label both.
- **Notice copy ships as drafted.** Per D40 the AB20 text lands as the agent drafted it and [l-05](sprint-l-05.md) ships it, so
  write it as final-quality text, faithful to D12, D24 and ADR-0033 §6. Label it on the board as the agent-drafted version 1; the
  owner reviews it at the [v3 opening gates](../rollout-plan.md#11-opening-gates-v3), and any revision is a later content PR.
- **Parallel design PRs** ([ds-m1-01](sprint-ds-m1-01.md), [ds-m2-01](sprint-ds-m2-01.md)) — touch only your three boards and their
  screenshots; rebase on `main` before pushing. They also edit `docs/v2/status.md` when they land: rebase before the status commit,
  touch only this sprint's rows and keep theirs.

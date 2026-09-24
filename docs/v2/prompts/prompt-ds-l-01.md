# Prompt — Sprint ds-l-01 · Design L: invite acceptance ★, privacy notice, erase account (AB19 ★, AB20, AB21)

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-ds-l-01.md`](../sprints/sprint-ds-l-01.md)   ·   **Milestone:** L (design track)   ·   **Prereqs:** none
> **This is a design sprint: it ends with an open PR and a STOP for owner review. You never merge it.**

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md) — repo conventions (the land-and-sync
  directive does **not** apply to design sprints: see the last line).
- The plan: [`../sprints/sprint-ds-l-01.md`](../sprints/sprint-ds-l-01.md) — the frame tables in Tasks 2–4 are the brief.
- Board conventions: [`../sprints/sprint-ds-m1-01.md`](../sprints/sprint-ds-m1-01.md) Task 1 (the index and its file names) and
  Task 2 (`board.css`, `bd-*` chrome, frame labels, behaviour notes, `.bd-narrow`, screenshots under `shots/`).
- Design system: [`../../../design-system/README.md`](../../../design-system/README.md) and
  [`../../../design-system/theme.css`](../../../design-system/theme.css) (use verbatim).
- v1 references (not runnable; layout and copy only): [`Auth.dc.html`](../../../design-system/screens/Auth.dc.html),
  [`Settings.dc.html`](../../../design-system/screens/Settings.dc.html); today's shipped screens `web/src/screens/Auth.tsx`
  (the `ds-seg` Sign in / Sign up pill, error copy in `emailAuthErrorMessage`, the `signup_closed` handling) and
  `web/src/screens/Settings.tsx` (the section rail `RAIL`, the "Sign-in & security" card, `Card`/`Field` helpers).
- Decisions:
  - [ADR-0033](../../adr/0033-invite-only-admission-and-owner-admin.md) §3 (seats), §5 (invites: fragment, uniform `invite_invalid`,
    `invite_required`, GitHub path), §6 (acceptance step, `403 acceptance_required`, notice contents, `privacy` slug), §7 (roles:
    owner never erased on the web), §9 (erase: typed confirmation, session < 5 min, testers at L-E), §10 (no web admin, `mailto:`);
  - [ADR-0031 §4](../../adr/0031-platform-ai-and-two-tier-keys.md#4-retention-and-privacy-owner-d24) (Proposed; records owner decision
    D24: retention ≤ 30 days, no training, outside India, the two unticked consents) and [t5 "Consent"](../research/t5-platform-ai.md) (the consent wording);
  - [ADR-0028 §2–§3](../../adr/0028-object-storage-and-backups.md#2-off-node-backups-deferred) and its Consequences (D12: no off-node
    backups — Hostinger's weekly images stay on as the only safety net, hence the ~7-day window; no erase ledger);
    [ADR-0027 §6](../../adr/0027-content-evalpack-and-user-data-model.md#6-account-erase-v20) (erase mechanics);
    [ADR-0034 §4.3](../../adr/0034-v2-release-labelling-gating-and-rollback.md#43-snapshot-rule) (1-day snapshots);
  - [ADR-0035 §4](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#4-limits-inventory) L8 (erase limits);
  - [PRD §5.5](../../prd/xlearn-v2-prd.md#55-content-arena-profile-privacy-from-t1--adr-0027) (R-PP1, R-AC5) and
    [§5.7](../../prd/xlearn-v2-prd.md#57-audience-admission-and-releases-from-t7--adr-0033--adr-0034--adr-0035) (R-AD1–R-AD8), R-AI6/R-AI8;
  - D12, D13, D24, D33, D35 in the [feasibility decisions log](../feasibility.md#decisions-log-newest-first).
- Boards list and freeze rule: [rollout §9](../rollout-plan.md#9-artboards-by-milestone) (L row: AB21 before L-E, AB19–AB20 before L-A).

## Context

L is the learner gate, **built and rehearsed in v2, never opened** (D35): invites, `SEAT_CAP`, the acceptance step (18+, notice
version, region, two unticked AI consents), the privacy notice, erase, and the `tester` role. Production stays `SIGNUP_MODE=closed`
except during one L-exit rehearsal. Two build sprints consume your boards:
- [l-02](../sprints/sprint-l-02.md) (early November) builds web erase for testers — **AB21 must be frozen before it starts**;
- [l-05](../sprints/sprint-l-05.md) (December, after M4) builds the auth page's invite state, the acceptance step and the privacy
  page — **AB19–AB20 must be frozen before it**.

Per **BP3** (owner, 2026-09-24) agents draft every v2 board, heroes included (AB19 is a hero ★); the owner only reviews. You draft
three boards; the owner's approval + merge is the freeze. The notice text on AB20 is a **draft**: the owner approves the final wording
at `ev-notice-text` in l-05.

## Entry gates — verify first (stop and report if any is unmet)

- [ ] **Notice inputs known** — D12's ~7-day data-loss window (Hostinger's weekly images as the only safety net), provider retention
      ≤ 30 days (flagged up to 2 years), no training, processing outside India, ADR-0033 §6's notice contents and §10's `mailto:` rule.
      The inputs are **owner decisions D12 / D24 / D33** (feasibility decisions log): ADR-0028 and ADR-0033 are Accepted; **ADR-0031 is
      still Proposed** (BP2: accepted at the WIF spike before M4), but its §4 only records D24 and doesn't change — a Proposed status
      there is expected, not a reason to stop. Confirm the facts by reading D12/D24 and those sections.

Then check (information, not a gate): `git fetch && git ls-tree origin/main design-system/screens/v2/`.
- If `index.html` and `board.css` exist (ds-m1-01 merged): link `board.css` and use the file names the index links for AB19–AB21.
- If not: **do not create or edit them.** Use `AB19-invite-acceptance.html`, `AB20-privacy-notice.html`, `AB21-erase-account.html`
  and inline the `bd-*` chrome from ds-m1-01's Task 2 in each board's `<style>`; say so in the PR.

## Do this (in order)

1. **[X] Branch** `design/ds-l-01` from an up-to-date `main`. Check peers first (`gh pr list`, `git worktree list`, ListAgents): other
   design PRs may be open — you touch only your three boards, their screenshots and this sprint's plan file.
2. **[X] Scaffold** the three boards under `design-system/screens/v2/`: `<link rel="stylesheet" href="../../theme.css">` (+ `board.css`
   when on `main`), the Google Fonts `<link>` for Inter + JetBrains Mono, a `<style>` block for board-only layout using `--ds-*` tokens
   only (danger = `--ds-err`; native checkboxes styled with tokens). Frames: full-screen frames stack vertically; small states sit side
   by side. Every frame: a label (`AB19-F9 · Acceptance step`), decision refs, final copy, and a **behaviour notes** aside (gates, exact
   error codes, focus order, contrast, keyboard, `aria-live`, < 1024 px layout). Each board ends with a 390 px `.bd-narrow` section.
   Static HTML only; nothing imported by `web/`.
3. **[X] AB19 ★ invite acceptance** — frames F1, F1b and F2–F14 exactly as the plan's Task 2 table lists: the `closed` auth page, also
   used for `invite` mode with no code (no create path, "xLearn is invite-only", `mailto:` **[Request an invite]**); an invite link while
   `closed` (stashed, not checked; "Invites aren't being accepted right now"); invite checking; the valid-invite create state with the expiry banner;
   the **one uniform** `invite_invalid` frame; `invite_required`; no seats (`no_seats`, flagged); email already registered (invite not
   consumed); the GitHub path behaviour; the **acceptance step hero** (18+, notice version link, region pre-filled India, the two
   **unticked** AI consents with the second disabled until the first is ticked, stepper "Step 1 of 5", **[Continue]**, **[Sign out]**);
   validation; owner/tester variant ("One more step"; consents pre-filled only from existing live grants — draw the ticked-from-a-grant case); re-acceptance on `403 acceptance_required`; save failed; mobile.
4. **[X] AB20 privacy notice / terms** — frames F1–F5 of Task 3: the public `/xlearn/privacy` page with version + effective date, the
   eight draft sections (who runs it; what's stored; how AI is used — Anthropic, outside India, ≤ 30 days, flagged up to 2 years, no
   training, and the BYO coach; what's public; one server, no off-node backups (the host's weekly image), **up to about 7 days** of data
   loss; your choices incl. erase, the 60-day username hold, erased data in the weekly image for up to ~7 days and in a pre-release
   snapshot for ≤ 1 day (erases re-run if one is restored), and the display-name leftover in older sign-up records; 18+; changes →
   re-acceptance), the "What changed" block, the Terms section,
   the in-app view, and 390 px. Mark the text **"Draft notice text — the owner approves the final wording before release."**
5. **[X] AB21 erase account** — frames F1–F10 of Task 4, placed as the last card of Settings → "Sign-in & security": the eligible tester
   card ("xLearn keeps no off-node backups" — never plain "no backups") with the "What's erased" list and the 60-day username hold; "Sign in again" (session older than 5 minutes /
   `403 reauth_required`); the typed-confirmation modal (username, or email when none; **[Cancel]** default focus; the danger button
   enabled only on an exact match); `400 confirmation_mismatch`; erasing; the signed-out "Your account has been erased" banner (with the
   ~7-day server-image line); the
   **owner-refused** card with no button (`403 erase_owner_cli_only`, "Use the admin CLI: `identity admin account erase`"); the
   **tester-only note** for learners (`403 erase_not_available`); other errors (`409 erase_in_progress`, 429, network); mobile.
6. **[X] Self-review** — walk every frame against its cited decisions and the rollout §9 L row. Confirm: invite errors uniform, no invite
   code visible after it's read, consents unticked by default everywhere and ticked only from a live grant (F11), no web admin / waitlist /
   owner erase button, the notice's required statements present (incl. the weekly-image and display-name disclosures; no "can't be
   linked back to you"), text contrast ≥ 4.5:1. Take full-page screenshots at **1440 px** and **390 px** (serve statically, e.g.
   `python3 -m http.server 5198 --directory design-system`, then `npx playwright screenshot --full-page --viewport-size=1440,900 …`
   or headless Chrome `--headless=new --screenshot=… --window-size=1440,900`) into
   `design-system/screens/v2/shots/AB19@1440.png`, `AB19@390.png` (and AB20, AB21), each ≲ 500 KB.
7. **[X] Update the plan's Status** (`docs/v2/sprints/sprint-ds-l-01.md`): tasks 1–5 ✅, _Overall_ 🔄 "PR open, awaiting owner review",
   task 7 ⬜.
8. **[X] Commit, open the PR, then STOP.** Conventional commit `docs(design): L boards AB19★ AB20 AB21` with the attribution lines from
   the session's system reminder; push; `gh pr create` titled "design: AB19★ AB20 AB21 (L)". The body: screenshots embedded from the
   branch (`…/blob/design/ds-l-01/design-system/screens/v2/shots/<file>?raw=true`), a frame list per board with decision cites, the
   self-review checklist, and the eight **"Decisions to confirm"** from the plan's Task 6. Mark task 6 ✅ in the plan (amend or add a
   commit before the PR is final). **Do not merge. Do not enable auto-merge.** Report the PR URL and stop.

## Constraints

- **Preview-only boards.** Static HTML on `theme.css` verbatim; no new colours or tokens; never imported by `web/`; never shipped.
  v1 `.dc.html` files are references only — do not edit them.
- **Touch only your files:** the three boards, their screenshots under `shots/`, and this sprint's plan file. Never `index.html` or
  `board.css` (ds-m1-01 owns them), never `theme.css`, never `docs/v2/status.md` (l-02 records the freeze).
- **Decisions win.** Every frame cites the decisions it implements; if the plan and an ADR disagree, the ADR wins — note the
  discrepancy in the PR instead of inventing behaviour.
- **Uniformity and consent:** one `invite_invalid` frame for every cause; consents unticked by default, separate, optional and
  withdrawable; never pre-ticked without an existing live grant (F11 shows the only ticked case).
- **No web admin, no waitlist, no owner web erase** (D33, ADR-0033 §7/§10). No alerting surface anywhere (D34).
- **No `web/`, service, infra or deploy changes.** No `kubectl` of any kind.
- **Parallel sessions:** other design PRs (ds-m1-01, ds-m2-01) may be open; rebase on `main` before pushing and resolve nothing
  outside your files.

## Deliverables

- `design-system/screens/v2/AB19-invite-acceptance.html` (★), `AB20-privacy-notice.html`, `AB21-erase-account.html`
  (or the names `index.html` links).
- `design-system/screens/v2/shots/AB19@1440.png`, `AB19@390.png`, `AB20@1440.png`, `AB20@390.png`, `AB21@1440.png`, `AB21@390.png`.
- An open PR with the screenshots, per-board frame lists, the self-review checklist and "Decisions to confirm".
- This sprint's plan Status updated.

## Update status

- In [`../sprints/sprint-ds-l-01.md`](../sprints/sprint-ds-l-01.md): tasks 1–6 ✅ as they land, _Overall_ 🔄 "PR #N open, awaiting
  owner review"; leave task 7 ⬜.
- **Do not edit [`../status.md`](../status.md)** from the design PR. After the owner merges, [l-02](../sprints/sprint-l-02.md) sets
  AB19/AB20/AB21 to "frozen (PR #N, date)", marks task 7 ✅ and this sprint ✅.
- No ADR is expected; if the owner's review changes a decision, the build sprint that implements it records the change.

## Done when (acceptance)

- [ ] Every frame listed for AB19, AB20, AB21 is present with final copy and states, each citing its decision(s).
- [ ] Boards use `../../theme.css` tokens/components only; preview-only; `index.html`, `board.css`, `theme.css` and `docs/v2/status.md` untouched.
- [ ] Invite errors uniform; consents unticked by default everywhere (ticked only from a live grant, F11); no owner web erase; the notice states Anthropic, ≤ 30 days, no training, outside India, the toggles and the ~7-day data-loss window.
- [ ] PR open with 1440 px and 390 px screenshots; **not merged by the agent**.

**Shipping:** this is a **design sprint** — the AGENT.md land-and-sync directive is replaced by this sprint's release action:
**open the PR and STOP for owner review.** Do not merge, do not enable auto-merge, do not tag. The owner's approval + merge is the freeze.

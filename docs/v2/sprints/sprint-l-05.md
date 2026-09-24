# Sprint l-05 — L-A/L-C front door: auth-page invite state, acceptance step★, privacy notice, AI consents (AB19★, AB20)

> **Milestone:** L — learner gate (**L-A** front door + **L-C** consents) · **Track:** product · **Order:** 67
> **Prereqs:** [l-03](sprint-l-03.md) (merged: invites, `account_consent`, `/api/auth/config`, `/api/invites/check`) · [m4-07](sprint-m4-07.md) (`v1.16.0` live, so M4's consent kinds exist and this doesn't ride an M4 tag) · [ds-l-01](sprint-ds-l-01.md) (AB19★ + AB20 frozen) · via `v1.16.0`: [m4-02](sprint-m4-02.md) (the kind constants in `internal/platform/consent`, identity's internal account read, judge's `POST /internal/accounts/{id}/refresh`), [m4-05](sprint-m4-05.md) (the append-only `SetConsents` write path and the gateway's judge-refresh call) and [m4-06](sprint-m4-06.md) (Settings consent toggles, AB18, the exported consent strings)
> **Unblocks:** [l-04](sprint-l-04.md) (tags `v1.17.0` and prepares the L-exit rehearsal)
> **Release action:** **merge only. It ships in `v1.17.0`**, tagged by [l-04](sprint-l-04.md), and never in an M3 or M4 tag ([ADR-0034 §1.6](../../adr/0034-v2-release-labelling-gating-and-rollback.md#16-indicative-tag-timeline)). Merging deploys nothing, because `main` is build-only.
> **Calendar:** December, after `v1.16.0`. Owner event **`ev-notice-text`** (~30 min): approve the drafted notice in this sprint's PR before it merges.
> **Execute with:** [`../prompts/prompt-l-05.md`](../prompts/prompt-l-05.md). One prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | Auth page: signup mode, `#invite=` fragment, invite states and errors (AB19) | X | ⬜ |
| 2 | identity acceptance: the `accept` step (first and notice-only re-acceptance), `accepted` tied to the notice version, `/me.acceptance`, order guard | X | ⬜ |
| 3 | Gateway `403 acceptance_required` in the shared session-validate path, an explicit exempt set, the judge refresh after `accept` | X | ⬜ |
| 4 | SPA acceptance step (AB19★ F9–F14), AuthedShell and global 403 handling | X | ⬜ |
| 5 | Privacy notice page `/privacy` (AB20), version parity, course-slug guard check | X | ⬜ |
| 6 | L-C: the two unticked AI consents at acceptance through m4-05's `SetConsents`; judge honours them | X | ⬜ |
| 7 | Existing fixtures default to accepted; end-to-end tests and compose smoke | X | ⬜ |
| 8 | Notice text approved (`ev-notice-text`) | O | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row, the L milestone, owner events). The AB19/AB20 artboard rows were set to frozen by [l-02](sprint-l-02.md); only confirm them.
> Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] **AB19★ and AB20 are frozen:** [ds-l-01](sprint-ds-l-01.md)'s PR has been merged by the owner. The boards are at `design-system/screens/v2/AB19-invite-acceptance.html` and `AB20-privacy-notice.html`. The one ds-l-01 merge froze AB19, AB20 and AB21 together, and [l-02](sprint-l-02.md) (task 9), the first consuming build sprint, already set all three rows in [`../status.md`](../status.md) to "frozen (PR #, date)". **Confirm** the AB19/AB20 rows read frozen; don't rewrite them.
- [ ] **[l-03](sprint-l-03.md) is merged on `main`.** That provides invite redeem on both paths, `account_consent`, `GET /api/auth/config`, `POST /api/invites/check` and `SIGNUP_MODE=invite`.
- [ ] **`v1.16.0` is live** ([m4-07](sprint-m4-07.md)). On `main`:
  - m4-02's `internal/platform/consent` (`AIReviewGraded`, `AIReviewPassing`, `AIBehavioral`; `consent.Version`), identity's `/internal/accounts/{id}` returning live rows as `consents: {kind: {version, granted_at}}`, and judge's 5-minute account cache with `POST /internal/accounts/{id}/refresh`;
  - m4-05's store function **`SetConsents`** (append-only history, withdrawing `ai_review_graded` also withdraws `ai_review_passing`, 422 `requires_ai_review_graded`), `PATCH /api/me/consents`, and the gateway's judge-refresh call after each consent change;
  - m4-06's Settings toggles and its exported consent strings.
- [ ] **No peer tag is planned between this merge and [l-04](sprint-l-04.md)'s `v1.17.0`.** Check with `git ls-remote --tags origin`, `gh pr list` and ListAgents. A patch tag in between would carry the acceptance step early.
- [ ] **Merge gate (not an entry gate): the notice text is approved** (`ev-notice-text`). Task 5 drafts it in this PR, and the PR waits for the owner's explicit approval before it merges.

## Goal

Ship the owner-visible half of L-A and L-C, to go out together in `v1.17.0`:
- the **auth page reads the signup mode**: it hides "Sign up" while `closed` and shows "xLearn is invite-only" with a `mailto:` "Request an invite". It stashes the `#invite=` code and removes it from the URL, and it shows AB19's invite and error states;
- the **acceptance step** (onboarding step 0) that **every account passes once**, the owner and testers included. It asks for 18+, agreement to the privacy notice at its version, and the region, plus the **two unticked AI consents** (L-C), written through m4-05's `SetConsents`. After a notice-version bump it asks **only** for the notice again (AB19-F12);
- the gateway's **`403 acceptance_required`** on every non-onboarding API until the step is done, and again after any notice-version change;
- the **privacy notice** at `/xlearn/privacy`.

Sources: [ADR-0033 §5, §6, §10](../../adr/0033-invite-only-admission-and-owner-admin.md#6-the-acceptance-step-onboarding-step-0-l-a), [ADR-0031 §4](../../adr/0031-platform-ai-and-two-tier-keys.md#4-retention-and-privacy-owner-d24) (consents, retention, 18+), [rollout §4](../rollout-plan.md#4-per-milestone-detail) (L-A, L-C rows).

## Scope

**In**
- `web/src/screens/Auth.tsx` and new `web/src/components/auth/*`: the three modes, the fragment capture, `/invites/check`, the error copy, the `mailto:` link, the privacy link, and the acceptance step.
- identity: the `accept` onboarding step (first acceptance and the notice-only re-acceptance), `accepted` = `accepted_at` plus a live `privacy_notice` consent at the current version, `/me.acceptance`, the region list, and the consent kinds `privacy_notice` and `age_18`.
- gateway: the acceptance gate in the shared session-validate path that `authAccount` and `authSession` both use, with a four-route exempt set; the judge account-cache refresh after a successful `accept`.
- `web/src/screens/Privacy.tsx` with `web/src/content/privacy-notice.md`; the notice-version parity test; a check that `privacy` is in the course-slug guard.
- The two AI consents at acceptance, written through m4-05's `SetConsents` with m4-02's constants, with a judge-side test that it honours them.
- Existing test fixtures (gateway fakes, `internal/e2e` helpers, web `/me` fixtures) default to an accepted account, so the gate doesn't break unrelated suites.

**Out**
- The invite backend, redeem, the CLI and `SEAT_CAP` → [l-03](sprint-l-03.md) (done).
- Opening web erase to every non-owner, tagging `v1.17.0`, the rehearsal runbook and infra PRs → [l-04](sprint-l-04.md).
- The behavioral opt-in (Settings only, a separate opt-in per ADR-0031 §4), the Settings consent toggles, and `/api/me/ai-allowance` → [m4-05](sprint-m4-05.md) / [m4-06](sprint-m4-06.md) (done).
- The erase UI (AB21) → [l-02](sprint-l-02.md) (done). Any email sending or waitlist form: none exists (ADR-0033 §10).
- Any infra change. `SIGNUP_MODE` stays `closed` on production.

## Tasks

### 1 · Auth page (AB19) [X]

Sources: [ADR-0033 §5](../../adr/0033-invite-only-admission-and-owner-admin.md#5-invites-mint-and-redeem) (the fragment, `sessionStorage`, `history.replaceState`, open on Sign up), [§10](../../adr/0033-invite-only-admission-and-owner-admin.md#10-no-web-admin-no-waitlist) (invite-only notice, `mailto:`); board `design-system/screens/v2/AB19-invite-acceptance.html`.
- `web/src/lib/auth.ts`: `useAuthConfig()` → `GET /auth/config` (react-query, `staleTime` 30 s) and `useInviteCheck(code)` → `POST /invites/check`. `Me` gains `acceptance` (task 2).
- `web/src/lib/invite.ts`:
  - `captureInviteFromHash()` runs once on `/auth` mount. It matches `location.hash` against `^#invite=([A-Za-z0-9_-]{22})$` and stores the code in `sessionStorage["xl_invite"]`, wrapped in try/catch with an in-memory fallback when storage is blocked. It **always** calls `history.replaceState(null, "", location.pathname + location.search)` when the hash starts with `#invite=`, valid or not, so no code lingers in the address bar or history.
  - `readInvite()`, and `clearInvite()` after a successful sign-up or redeem, or on `invite_invalid`.
- `web/src/lib/contact.ts`: `INVITE_REQUEST_MAILTO` is the owner's public contact address, confirmed by the owner in the `ev-notice-text` review. It is a build-time constant, so no infra or env change is needed. The notice reuses it.
- `SignIn` by mode, following AB19's frames:

| Mode | No stashed code | With a stashed code |
|---|---|---|
| `closed` | Sign-in only: **no Sign up segment**, plus "xLearn is invite-only" and "Request an invite" (`mailto:`) | the same, with AB19's "invites aren't being accepted right now" note. **No `/invites/check` call.** |
| `invite` | the same as `closed` | check it. **Valid:** open on **Sign up** with "Invite valid until <date>"; the email sign-up sends `invite`; the GitHub form gets `<input type="hidden" name="invite" value=…>`. **Invalid:** AB19's `invite_invalid` frame, no Sign up, `clearInvite()` |
| `open` (compose and dev) | as today | as today (the invite is ignored) |

- **Errors** (AB19 copy): the redirects `?error=invite_required`, `?error=invite_invalid` and `?error=no_seats`, the API codes `invite_required`, `invite_invalid` and `no_seats`, and the existing `signup_closed` and `account_exists_password`. Extend `oauthErrorMessage` and `emailAuthErrorMessage`.
- A footer link to `/privacy` on the sign-in card.
- `Auth.tsx` is already over 800 lines. New pieces go in `web/src/components/auth/` (`InviteNotice.tsx`, `Acceptance.tsx`). Use `theme.css` classes verbatim. m1-04's CSP allows React `style` props but no `<style>` element and no inline script.
- **Tests** (vitest, `Auth.test.tsx` plus the new components):
  - the Sign up segment's visibility for each mode;
  - the fragment is stashed and `replaceState` is called (and `location.hash` is empty afterwards);
  - the hidden `invite` input is present only with a valid code;
  - each error code renders AB19's copy;
  - no check call in `closed`;
  - storage throwing falls back to memory.

### 2 · identity: acceptance [X]

Sources: [ADR-0033 §6](../../adr/0033-invite-only-admission-and-owner-admin.md#6-the-acceptance-step-onboarding-step-0-l-a) (every account once; 18+, notice version, region pre-filled from the invite, default `IN`; the M4 consents; `403` on a notice-version change), §4, §7; [ADR-0019](../../adr/0019-account-settings-onboarding-and-reminder-gating.md) (onboarding, which gains a step 0).
- `internal/identity/notice.go` defines `NoticeVersion = "privacy-notice@1"` and `AgeAttestVersion = "age-18@1"`. The `consent.go` registry (l-03) gains `privacy_notice` and `age_18`; m4-05 already registered the AI kinds and the behavioral opt-in, whose constants live in **m4-02's `internal/platform/consent`** (`consent.AIReviewGraded`, `consent.AIReviewPassing`, `consent.Version`). Import them; invent no names.
- **`accepted`** in session-validate (`GetValidSession`, extended by m1-04) becomes `a.accepted_at IS NOT NULL AND EXISTS (SELECT 1 FROM identity.account_consent c WHERE c.account_id = a.id AND c.kind = 'privacy_notice' AND c.version = $2 AND c.withdrawn_at IS NULL)`, with `$2 = NoticeVersion`. Bumping the constant makes every account re-accept. The lookup hits l-03's partial unique index; measure `/sessions/validate` p95 in compose before and after.
- **Two modes**, derived per account (AB19-F9/F11 vs AB19-F12):
  - **`first`**: no live `age_18` row at `AgeAttestVersion`, or `accepted_at IS NULL`. The full form.
  - **`renotice`**: the account accepted an earlier notice (`accepted_at` set and a live `age_18` row at `AgeAttestVersion`) and only `NoticeVersion` changed. **Only the notice is asked again**; 18+, region and the AI consents keep their recorded values (AB19-F12, "conservative reading; flagged for the owner" on the board).
- **`POST /onboarding/step {"step":"accept", "notice_version", "agree_notice", "age_18", "region", "consents": {"ai_review_graded": bool, "ai_review_passing": bool}}`** in `handleOnboardingStep`:
  - `notice_version` ≠ `NoticeVersion` → 409 `notice_changed` (the SPA refetches `/me`);
  - `agree_notice` not `true` → 422 `notice_required` (AB19-F9's required notice checkbox);
  - in `first` mode: `age_18` not `true` → 422 `age_required`; `region` not in the ISO 3166-1 alpha-2 list (`internal/identity/region.go`, a static table) → 422 `invalid_region`; `consents` present with a key other than the two review kinds → 422;
  - in `renotice` mode: only `notice_version` and `agree_notice` are read, and any other field is ignored, so 18+, region and the consents keep their recorded values (the SPA sends none, per F12). Consent changes go through Settings (`PATCH /api/me/consents`);
  - otherwise `store.AcceptTerms`, **in one transaction**:
    - write a live `privacy_notice` row (`version = NoticeVersion`), superseding any earlier live `privacy_notice` row by setting `withdrawn_at = now()`; in `first` mode also a live `age_18` row (`version = AgeAttestVersion`), superseded the same way;
    - in `first` mode, the two AI kinds go through **m4-05's `SetConsents`** with `{AIReviewGraded: bool, AIReviewPassing: bool}`, **inside this same transaction**. If m4-05's function opens its own transaction, split it into a tx-scoped core (`setConsentsTx(ctx, q, …)`) that `SetConsents` and `AcceptTerms` both call, with no behaviour change. This inherits, rather than re-implements: the append-only history (grant inserts a row at `consent.Version`, withdraw sets `withdrawn_at`, unchanged writes nothing), **withdrawing `ai_review_graded` also withdraws `ai_review_passing`**, and 422 `requires_ai_review_graded` for passing without graded;
    - `account.accepted_at = now()`; in `first` mode also `account.region = $region`;
    - repeating the call with the same body is idempotent.
- **Order guard:** a `path`, `budget` or `finish` step while acceptance is required → 409 `acceptance_required`. identity holds the rule as well as the gateway.
- **`GET /accounts/{id}`** (→ `/api/me`) gains the additive block `"acceptance": {"required": bool, "mode": "first"|"renotice"|null, "notice_version": "privacy-notice@1", "region": "IN"|null, "consents": {"ai_review_graded": bool, "ai_review_passing": bool}}` (`mode` is null when not required). `region` is `account.region` (set from the invite at redeem, l-03), and the SPA defaults to `IN` when it is null. `consents` reports whether each review kind has a live grant.
- **Tests** (PG 18 plus handler tests):
  - the validation matrix, both modes (including `notice_required`, and a notice-only body accepted in `renotice` but refused in `first`);
  - rows written and superseded; `renotice` writes only a new `privacy_notice` row and leaves the `age_18` row, `region` and the AI rows untouched;
  - the AI rows at acceptance come from `SetConsents`: a withdraw and re-grant keeps the history; unticking graded also withdraws passing; passing without graded → 422 `requires_ai_review_graded`;
  - a version bump flips `accepted` to false and `mode` to `renotice`;
  - the order guard;
  - `/me` shape;
  - erase still removes every consent row (l-03's cascade).

### 3 · Gateway: `403 acceptance_required` [X]

Sources: [ADR-0033 §6](../../adr/0033-invite-only-admission-and-owner-admin.md#6-the-acceptance-step-onboarding-step-0-l-a) ("every non-onboarding API"), §7 (roles and status are never cached).
- `internal/gateway/acceptance.go`: `acceptanceExempt` is keyed by the matched `r.Pattern`:
  - `GET /api/me`;
  - `POST /api/onboarding/step`;
  - `POST /api/auth/logout`;
  - `DELETE /api/me`, as an **API-level safeguard only**: the gate must never be what blocks an erase. The SPA offers **no** erase path while acceptance is required (AuthedShell redirects to `/auth`, so Settings and AB21 are unreachable, and AB19 offers only Sign out). Someone who declines a changed notice signs out and asks the owner, through the notice's contact address, for a CLI erase (`identity admin account erase`). A web erase entry point on the acceptance step would be a board delta on the frozen AB19; flag it to the owner for the v3 opening, don't build it here. Record all of this in the decisions log.
- **Where the check lives:** in the **one session-validate path that both `authAccount` and `authSession` go through**. Today `authAccount` (`internal/gateway/bff.go`) calls `g.identity.validateSession` directly, and m1-04 kept its signature for existing callers while adding `authSession`. If `authAccount` still bypasses `authSession`, make it delegate to it (signature unchanged), so no handler can skip the gate. After session-validate: if `!info.Accepted` and `r.Pattern` isn't exempt → **403** `{"error":{"code":"acceptance_required","message":"Review and accept the privacy notice to continue."}}`. The `accepted` value is never cached (m1-04's rule), so a notice bump or an acceptance takes effect on the next request. The route-walk test proves every session route is covered.
- **Judge refresh after `accept`:** `handleOnboardingStep` (`internal/gateway/bff.go`), after identity answers 200 to a `"step":"accept"` body, calls judge `POST /internal/accounts/{id}/refresh` when `JUDGE_BASE_URL` is set. Reuse m4-05's refresh helper (the one `PATCH /api/me/consents` uses): one retry, then an ERROR log (D34: no alert), and the 200 is returned anyway, since the consent is stored and the 5-minute TTL bounds the staleness. So a consent withdrawn at acceptance applies to the very next platform-AI call, as it does from Settings ([t5 §8](../research/t5-platform-ai.md#8-privacy-and-residency)). gateway → judge `/internal/*` already exists (m4-05, MI-5a), so this adds no caller and no NetworkPolicy change.
- Untouched: the session-less routes (`/api/auth/*`, `/api/invites/check`, `/api/auth/config`, `/api/u/{username}`, JWKS, healthz) and the internal service routes.
- **Route-walk test** over `apiRoutes()`:
  - an unaccepted session → 403 on every session route outside the exempt set, whether its handler uses `authAccount` or `authSession`, and the exempt ones pass;
  - an accepted session → no 403 anywhere;
  - `POST /api/coach/chat` returns the JSON 403 before any SSE bytes;
  - a fake identity that flips `accepted` (a version bump) → 403 again.
- **Refresh test** (fake judge): a successful `accept` calls `/internal/accounts/{id}/refresh` once; a failing judge is retried once, logged at ERROR, and the client still gets 200; another onboarding step and a failed `accept` call nothing; with `JUDGE_BASE_URL` unset nothing is called.

### 4 · SPA acceptance step (AB19★) [X]

Board: `design-system/screens/v2/AB19-invite-acceptance.html`, acceptance frames **F9–F14**, which are frozen and win over this summary where they differ; tokens from [`theme.css`](../../../design-system/theme.css). Use AB19's copy verbatim.
- `web/src/components/auth/Acceptance.tsx`, laid out per **AB19-F9** (`first` mode): title "Before you start", stepper "Step 1 of 4" ("One more step" when onboarding is otherwise complete, **F11**), then:
  - ☐ **"I'm 18 or older."**, required;
  - ☐ **"I've read the [privacy notice] (version 1, effective <date>) and agree to it."**, required. The link opens `/privacy` (AB20) in a new tab. The version and date come from `lib/notice.ts`, the constant the server checks. It sends `agree_notice: true`;
  - **region**: "Where do you live?", a select of ISO codes with names from `Intl.DisplayNames` (no dependency), pre-filled from `me.acceptance.region`, else `IN`, with F9's helper line;
  - the fieldset **"xLearn AI (optional)"** with the **two AI consents** and F9's helper text (Anthropic, outside India, up to 30 days, never used for training, change any time in Settings, off means you grade your own work):
    - "xLearn AI reviews my graded work — grade suggestions and feedback on work you submit.";
    - "…and also reviews my passing solutions for improvement notes."

    The second is disabled until the first is ticked, and unticking the first unticks the second (ADR-0031 §4; m4-05's graded ⇒ passing rule). Reuse m4-06's exported consent strings where AB19's text matches. **Defaults:** for an account with no live grant (every invitee, and anyone who never used Settings) both are **unticked**, as F9 draws. For an account that already granted them in Settings during M4 (the owner, testers), the boxes show those **live grants**, and unticking one is a withdrawal through `SetConsents`. The boxes must show what is stored; showing an unticked box over a live grant would misstate the consent. This differs from F9/F11's "both unticked" only for accounts with existing grants, so it is a **flagged deviation**: record it in the decisions log and call it out in the PR for the owner's review;
  - primary **[Continue]**, secondary **[Sign out]** (AB19).
- **F10 validation:** submit with a required box unticked → the inline errors "Please confirm you're 18 or older." / "Please accept the privacy notice to continue."; focus moves to the first invalid item; **[Continue]** stays enabled. Server 422s (`notice_required`, `age_required`, `invalid_region`, `requires_ai_review_graded`) map to the same inline slots.
- **F12 re-acceptance** (`mode: "renotice"`): F12's banner "We've updated the privacy notice (version N, effective <date>). Please review what changed and accept to keep using xLearn." with **[What changed]** → `/privacy#what-changed` (AB20-F2), and **only the notice checkbox**. No 18+, region or consent controls, and nothing is pre-ticked. The payload is `{step: "accept", notice_version, agree_notice: true}`.
- **F13** save failed: "Couldn't save your answers — check your connection and try again."; ticked boxes stay ticked. **F14** < 1024 px / 390 px: items stacked, the consent fieldset below the required items, **[Continue]** pinned at the bottom.
- In `Auth.tsx`, `Onboarding` renders step 0 when `me.acceptance.required`, before `firstUnfinishedStep`. After success it invalidates `me`: if onboarding is complete, `navigate("/")`, else step 1. **The owner and testers** (already onboarded) see only step 0, once.
- In `web/src/components/RequireAuth.tsx`, `AuthedShell` sends `me.data.acceptance?.required` → `<Navigate to="/auth" replace />`.
- `web/src/lib/api.ts` gets `ApiRequestError.isAcceptanceRequired` (a 403 with that code). `web/src/lib/queryClient.ts` gets a global `onError` for queries and mutations that invalidates `me`, so AuthedShell redirects. That covers a notice bump in the middle of a session.
- **Tests:**
  - step 0 appears before the path step and for an already-onboarded account (F11's stepper text);
  - both required boxes, F10's errors and focus on submit, and [Continue] never disabled;
  - the defaults are unticked with no live grant; live grants show ticked; the second consent is disabled until the first, and unticking the first unticks the second;
  - the `first` and `renotice` payloads; `renotice` renders only the notice box, the F12 banner and the [What changed] link;
  - a 409 `notice_changed` → refetch;
  - the AuthedShell redirect;
  - a 403 from any API → back to `/auth`.

### 5 · Privacy notice (AB20) [X] · text approval [O]

Sources: [ADR-0033 §6](../../adr/0033-invite-only-admission-and-owner-admin.md#6-the-acceptance-step-onboarding-step-0-l-a) (contents; a new SPA segment joins the course-slug guard), [ADR-0031 §4](../../adr/0031-platform-ai-and-two-tier-keys.md#4-retention-and-privacy-owner-d24), [feasibility](../feasibility.md#decisions-log-newest-first) D12 (no backups; ~7-day loss window), [rollout §8](../rollout-plan.md#8-what-ships-where-content-hours-the-d6-reading) (DPDP ~2027-05-13); board `design-system/screens/v2/AB20-privacy-notice.html`.
- **Route:** `{ path: "/privacy", element: <Privacy /> }` goes in `web/src/router.tsx` **outside** `AuthedShell` (public, like `/auth` and `/u/:username`). A test shows it outranks m1-03's `/:course`.
- `web/src/screens/Privacy.tsx` follows AB20's layout (F1, F3 Terms, F4 in-app view, F5 390 px). The body is `web/src/content/privacy-notice.md` (`?raw` import), rendered by m1-06's Markdown renderer (`web/src/components/Markdown.tsx`, CSP-safe). The header shows `privacy-notice@1` and the effective date. AB20-F2's "What changed" block renders at `#what-changed` for version ≥ 2 (AB19-F12's [What changed] link targets it); at version 1 it is absent.
- **Version parity:** `web/src/lib/notice.ts` sets `NOTICE_VERSION = "privacy-notice@1"`. `internal/identity/notice_parity_test.go` reads that file and asserts it equals `NoticeVersion`, the same pattern as m1-03's slug-guard parity test.
- **Course-slug guard:** `privacy` is already reserved (m1-09's Go guard, m1-03's `web/src/lib/courseSlugGuard.ts`). Confirm both lists contain it and the parity test is green; add it to any list that lacks it.
- **Links:** the auth card footer, the acceptance step, and a small "Privacy notice" link in Settings next to the AB18 consents and the AB21 erase section.
- **Draft the text** (agent) as `privacy-notice.md`, in plain words, starting from AB20-F1's eight draft sections and F3's Terms. **Verify each claim against the code before writing it.** It must cover:
  - **who** runs xLearn (the owner, as an individual) and how to reach him (`INVITE_REQUEST_MAILTO`);
  - **what** is stored: account (email, username, display name, password hash, GitHub id), learning activity, submissions and code, coach chats, the encrypted BYO key, grading records, region, the 18+ confirmation and your choices; one session cookie, and no analytics, ads or tracking;
  - **xLearn AI**, if consented: provider **Anthropic**, provider retention **up to 30 days** (flagged content longer, per ADR-0031 §4), **no training**, and **processing outside India**. Pack material is never sent to Feedback or Analyze;
  - **the toggles:** the two AI consents at acceptance and in Settings, and the behavioral opt-in in Settings. All are off by default and withdrawable at any time; off means manual or self grading;
  - **BYO coach:** your key, your provider, and its terms apply;
  - **18+ only** (attestation);
  - **region:** its use (the EU/EEA voice gate later);
  - **the data-loss window:** there are no backups (D12), so **up to about 7 days** of data can be lost with the node or its disk;
  - **erase:** in Settings, for every non-owner; what it deletes; the username held for 60 days; provider copies expiring on the provider's schedule; and **"a server snapshot taken just before a release may hold your data for up to 1 day"** (AB20-F1 item 6; [ADR-0034 §4.3](../../adr/0034-v2-release-labelling-gating-and-rollback.md#43-snapshot-rule), kept for 1 day). The "every non-owner" claim is **not yet true on this sprint's code**: web erase is still testers-only (l-02) until [l-04](sprint-l-04.md) task 1 opens it to learners, and both land in the same `v1.17.0`. Write the claim for the tagged state, note it in the PR, and l-04's pre-tag check reconfirms it on the commit it tags;
  - **the public profile:** what is public and the private toggle;
  - **no email** is ever sent;
  - **terms** (AB20-F3): invite-only, one account per person, invites single-use and not to be shared, the owner may suspend accounts, provided as is, course content MIT-licensed;
  - rights and contact; the version and effective date.
- **Version rule:** a substantive text change bumps **both** constants, and everyone re-accepts. A typo fix doesn't (decisions log).
- **[O] `ev-notice-text`:** the owner reads the rendered page in the PR (screenshots at 1440 px and 390 px) and approves the text and the contact address. **The PR does not merge before that approval.**

### 6 · L-C: consents at acceptance; judge honours them [X]

Sources: [ADR-0031 §4](../../adr/0031-platform-ai-and-two-tier-keys.md#4-retention-and-privacy-owner-d24), [t5](../research/t5-platform-ai.md) ("Consent (fixed: free, specific, withdrawable)"), [ADR-0033 §12](../../adr/0033-invite-only-admission-and-owner-admin.md#12-authz-deltas-across-v2) row 13.
- Use **m4-02's constants** from `internal/platform/consent` (`AIReviewGraded`, `AIReviewPassing`, `consent.Version`) for the two AI kinds (import them; invent no names). The behavioral opt-in stays Settings-only.
- **One write path:** the acceptance step writes the AI kinds only through m4-05's `SetConsents` (task 2), so acceptance and Settings share the history, the graded ⇒ passing withdraw rule and the 422. **One freshness path:** the gateway's judge refresh after `accept` (task 3) is the same call m4-05 makes after `PATCH /api/me/consents`, so a withdrawal at acceptance applies to the next call rather than after the 5-minute TTL.
- No judge code change is expected: judge's per-purpose gate in `Reserve`, its account cache and `POST /internal/accounts/{id}/refresh` exist (m4-02). **Tests:**
  - identity, against m4-02's internal-read shape (`consents` lists **live rows only**, as `{kind: {version, granted_at}}`): accept with both unticked → neither AI kind appears; with the first ticked → only `ai_review_graded`, at `consent.Version`; an account with both granted in Settings that unticks the first at acceptance → neither appears (the cascade);
  - judge: m4-05's "no platform-AI call without consent" test gains a case built from acceptance-written rows (m4-02's fake provider; no `llm_call` row, the manual badge), including a withdrawal at acceptance followed at once by a conclusion (no call, no 5-minute wait, via the refresh). If the `-tags e2e` harness boots identity, gateway and judge, add the cross-service case there too.

### 7 · Existing fixtures, end-to-end tests and compose smoke [X]

- **Keep the existing suites green.** After this sprint every account or session with `accepted=false` gets 403 on every non-exempt route, so fixtures that don't opt in would fail for unrelated reasons:
  - gateway: the fake identity's validate answer (`internal/gateway/*_test.go`, including `mock_test.go`) defaults to `accepted: true`; only the gate's own tests set `false`;
  - `internal/e2e`: a helper (for example `acceptTerms(t, client)`) posts a `first`-mode `accept` for every account a flow creates, and the existing flows (`coreloop_test.go` and the rest) call it right after sign-up;
  - web: the vitest/MSW `/me` fixtures gain `acceptance: {required: false, mode: null, …}`, so only the acceptance tests exercise the redirect;
  - the compose dev login (`internal/identity/devauth.go` `handleDevLogin`, which completes onboarding) is **not** changed to auto-accept: the dev account must show step 0 in the smoke below.
- `-tags e2e` (or a compose script if the harness lacks identity):
  - `SIGNUP_MODE=invite` → mint via the CLI → email sign-up with the invite → `GET /api/dashboard` → 403 `acceptance_required`;
  - `POST /api/onboarding/step accept` (`first` mode) → 200 everywhere;
  - bump `NoticeVersion` in the test → 403 again → a notice-only `accept` (`renotice`) → 200, with 18+, region and the consents unchanged.
- Compose, with the browser console open:
  - the auth page in `open`, in `invite` (with and without a code) and in `closed` (compose overrides);
  - `/xlearn/privacy` logged out;
  - the acceptance step for an already-onboarded dev account;
  - then every screen: **no CSP violation**, and no call other than the exempt four before acceptance.
- Screenshots at 1440 px and 390 px beside AB19 and AB20 in the PR.

## Acceptance criteria

- [ ] **`403 acceptance_required` until acceptance** on every non-exempt session route (route-walk test). After `v1.17.0` the owner passes it once ([l-04](sprint-l-04.md) records it).
- [ ] **A notice-version change re-prompts** for the notice only (AB19-F12): 18+, region and the consents keep their recorded values (identity plus gateway tests, and e2e).
- [ ] **Both required boxes** (18+ and the notice agreement) are enforced in the SPA (F10) and the server (`age_required`, `notice_required`).
- [ ] **The consents are unticked by default** and the second is disabled until the first is ticked; they are written **only through m4-05's `SetConsents`** (history, graded ⇒ passing withdraw, `requires_ai_review_graded`); the gateway refreshes judge's cache after `accept`; **judge honours them** (no platform-AI call without consent, tested from acceptance-written rows, including an immediate withdrawal).
- [ ] Existing gateway, e2e and web suites stay green with fixtures that default to accepted.
- [ ] **The notice page is reachable** logged out at `/xlearn/privacy`. The `privacy` slug is reserved in both guards, and the version parity test is green.
- [ ] **Auth page:** no Sign up while `closed`; the invite-only notice and `mailto:`; `#invite=` is stashed and removed from the URL; AB19's error states render.
- [ ] Matches AB19★ and AB20 at 1440 px and 390 px; no CSP violation in the compose smoke.
- [ ] **Notice text approved by the owner (`ev-notice-text`)** before the merge.
- [ ] `sqlc diff` clean; CI green; merged to `main` (not tagged).

## Release

**Merge only. It ships in `v1.17.0`**, which [l-04](sprint-l-04.md) tags right after this merge, per rollout §7: "v1.17.0 | L-A: invite flow, acceptance, notice, 18+, region, consents …".
- No migration is expected: `account_consent` comes from l-03, and `accepted_at` and `region` from M1a. If one turns out to be needed, it is expand only.
- No new NATS subject, no new in-cluster caller (gateway → identity, gateway → judge `/internal/*` from m4-05, and judge → identity already exist), and no pod, so the memory sum is unchanged. No infra change.
- **Rollback:** there is no env switch for the acceptance gate. If it misfires after `v1.17.0`, use R-c (revert plus a patch tag) or R-b to `v1.16.0`. With no contract, that stays above the floor. The route-walk test and the compose smoke exist to prevent a lock-out.

## Definition of Done

CI green (including `sqlc diff` and the OpenAPI drift test for `/me`'s `acceptance` block and the new error codes) · PR squash-merged after `ev-notice-text` · screens match AB19★ and AB20 · acceptance criteria met · statuses updated (this file, plus [`../status.md`](../status.md): the board row "merged, ships in v1.17.0"; **confirm** the AB19/AB20 artboard rows read "frozen" (set by [l-02](sprint-l-02.md)); owner event `ev-notice-text` ✅ with the date; the L milestone "front door merged") · decisions log:
- the exempt set, with `DELETE /api/me` as an API-level safeguard only (no SPA erase path while acceptance is required; decliners ask the owner for a CLI erase; a web entry point is a board delta flagged for v3);
- the AI consents at acceptance go through m4-05's `SetConsents`, and the gateway refreshes judge after `accept`;
- **flagged deviation from AB19-F9/F11:** accounts with live Settings grants see them ticked, and an untick is a withdrawal (for the owner's confirmation in the PR);
- the notice-only re-acceptance per AB19-F12 (18+, region and consents keep their values);
- the notice-version bump rule; the `mailto:` constant.

## Risks / watch-outs

- **The acceptance step hits the owner on deploy (once).** This is expected, and it lands with the L tag. Tell the owner in the `v1.17.0` release notes ([l-04](sprint-l-04.md)).
- **Lock-out:** a missing exempt route (for example `GET /api/me`) would trap every account in a redirect loop. The route-walk test, the e2e test and the compose smoke all guard it.
- **Consent defaults must be unticked.** Showing an account's live grants is the flagged exception (decisions log); defaulting to ticked is never allowed, and re-acceptance shows no consent controls at all. It is tested.
- **Two consent write paths would drift.** Never write AI consent rows outside `SetConsents`; a second implementation would miss the graded ⇒ passing withdrawal or the judge refresh.
- **The notice must be true.** Every sentence is checked against the code and the ADRs, and anything that isn't built by the tag that ships it (`v1.17.0`) isn't claimed. The one claim that depends on [l-04](sprint-l-04.md) (web erase for learners) is reconfirmed by l-04 before the tag.
- **Riding the wrong tag:** a peer patch tag between this merge and `v1.17.0` would ship the acceptance step early. Coordinate (entry gate 4), and merge only when [l-04](sprint-l-04.md) can follow at once.
- **`sessionStorage` is blocked** in some private modes: fall back to memory for the page's lifetime (the code is lost on reload, and the owner re-sends the link).
- **The invite code in URLs:** the fragment never reaches Traefik or the `Referer` header, and `replaceState` removes it from history. Never copy it into a query string, a log or an analytics call.

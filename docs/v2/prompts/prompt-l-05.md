# Prompt — Sprint l-05 · L-A/L-C front door: auth-page invite state, acceptance step★, privacy notice, AI consents (AB19★, AB20)

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-l-05.md`](../sprints/sprint-l-05.md)   ·   **Milestone:** L (L-A front door + L-C)   ·   **Prereqs:** [l-03](../sprints/sprint-l-03.md) (merged), [m4-07](../sprints/sprint-m4-07.md) (`v1.16.0` live), [ds-l-01](../sprints/sprint-ds-l-01.md) (AB19★ + AB20 frozen)

## Before you launch (owner)

Launching this prompt attests these are done (D40). If one turns out to be missing, land everything that doesn't depend on it and record the gap as ⛔ in `status.md`; don't wait.

- [ ] Optional: the public contact address for the invite-only `mailto:` link and the notice (`INVITE_REQUEST_MAILTO`), in your launch message. Without one, the session uses the public address already on this repo's commits and records the choice. The notice text lands as the session drafts it (D40); your review of it is a v3 opening gate ([rollout §11](../rollout-plan.md#11-opening-gates-v3)), and any later change is a content PR.

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md).
- The plan: [`../sprints/sprint-l-05.md`](../sprints/sprint-l-05.md). It has the mode table, the acceptance payload, the exempt set, the notice checklist and the tests.
- [ADR-0033](../../adr/0033-invite-only-admission-and-owner-admin.md) (Accepted):
  - **§5** (the fragment link, `sessionStorage`, `replaceState`, open on Sign up);
  - **§6** (the acceptance step: every account once, 18+, notice version, region, the M4 consents, `403 acceptance_required`, the notice contents, the course-slug guard);
  - **§10** (invite-only notice, `mailto:`, no waitlist);
  - §7 (the owner and testers pass it too).
- [ADR-0031 §4](../../adr/0031-platform-ai-and-two-tier-keys.md#4-retention-and-privacy-owner-d24) (the two unticked consents, the behavioral opt-in, retention, no training, outside India, 18+). ADR-0031 was accepted at the WIF spike (mi-12); don't edit it.
- [t5](../research/t5-platform-ai.md) ("Consent (fixed: free, specific, withdrawable)"; the onboarding-step copy).
- [ADR-0019](../../adr/0019-account-settings-onboarding-and-reminder-gating.md) (onboarding; this adds step 0); [ADR-0025](../../adr/0025-public-profiles-under-u-prefix.md) update + [ADR-0026](../../adr/0026-per-course-extensibility-model.md) (the course-slug guard).
- [ADR-0034 §1.6](../../adr/0034-v2-release-labelling-gating-and-rollback.md#16-indicative-tag-timeline) (`v1.17.0` carries L-A/L-C); [rollout §4](../rollout-plan.md#4-per-milestone-detail) (L-A, L-C rows).
- Boards (frozen): `design-system/screens/v2/AB19-invite-acceptance.html`, `design-system/screens/v2/AB20-privacy-notice.html`; [`theme.css`](../../../design-system/theme.css); v1 reference [`Auth.dc.html`](../../../design-system/screens/Auth.dc.html).
- The plans of [l-03](../sprints/sprint-l-03.md) (the endpoints and error codes), [m4-02](../sprints/sprint-m4-02.md) (`internal/platform/consent`, identity's internal account read, judge's cache and `POST /internal/accounts/{id}/refresh`), [m4-05](../sprints/sprint-m4-05.md) (**`SetConsents`**: append-only, graded ⇒ passing withdraw, `requires_ai_review_graded`; the gateway's judge refresh after every consent change), [m4-06](../sprints/sprint-m4-06.md) (the exported consent strings), [l-02](../sprints/sprint-l-02.md) (the erase UI lives in Settings) and [m1-04](../sprints/sprint-m1-04.md) (`authSession`, session-validate's `accepted`, CSP).
- Board frames that win over the plan where they differ: AB19 **F9–F14** (both required boxes, [Continue], F10 validation, F12 notice-only re-acceptance) and AB20 **F1–F5**.
- Code:
  - `web/src/screens/Auth.tsx`, `web/src/components/RequireAuth.tsx`, `web/src/router.tsx`, `web/src/lib/{auth.ts,api.ts,queryClient.ts,courseSlugGuard.ts}`, `web/src/components/Markdown.tsx`, `web/src/screens/Settings.tsx`, the vitest/MSW `/me` fixtures;
  - `internal/identity/{handlers.go,session.go,consent.go,devauth.go}`, `internal/identity/store/queries/session.sql`, m4-05's `SetConsents` in `internal/identity/store`;
  - `internal/gateway/{bff.go,gateway.go}` (`authAccount` at `bff.go` calls `validateSession` directly; m1-04's `authSession`; `handleOnboardingStep`; m4-05's judge-refresh helper), the gateway test fakes, `internal/e2e/`;
  - `internal/platform/consent` (m4-02); judge's consent read and refresh (m4-02).

## Context

[l-03](../sprints/sprint-l-03.md) shipped the invite backend dark: redeem on both create paths, `account_consent`, `POST /api/invites/check` and `GET /api/auth/config`. Production is still `SIGNUP_MODE=closed`. `v1.16.0` (M4) added the AI consent kinds and the Settings toggles, and judge spends only with consent.

This sprint builds the **owner-visible half of L-A and L-C**:
- the auth page reads the signup mode (no Sign up while `closed`; invite-only plus `mailto:`), stashes and strips `#invite=`, and shows AB19's states;
- **the acceptance step** (step 0: 18+, agreement to the notice at its version, region and the two unticked AI consents), which **every account passes once, the owner included**. After a notice bump only the notice is asked again (AB19-F12);
- the gateway's `403 acceptance_required`, and again on a notice-version change;
- the **privacy notice** at `/xlearn/privacy`.

It merges only. [l-04](../sprints/sprint-l-04.md) tags it as `v1.17.0` right after, so it never rides an M3 or M4 tag. The notice text lands as you draft it (D40): the owner's review of it (`ev-notice-text`) moves to the v3 opening gates ([rollout §11](../rollout-plan.md#11-opening-gates-v3)).

## Entry gates — verify first (stop and report if any is unmet)

- [ ] AB19★ and AB20 frozen: ds-l-01's PR is merged, the files exist under `design-system/screens/v2/`, and status.md's AB19/AB20 rows already read "frozen" (l-02 set them).
- [ ] [l-03](../sprints/sprint-l-03.md) is merged: `/api/auth/config`, `/api/invites/check`, `invite` on signup and the GitHub start form, and `account_consent` are on `main`.
- [ ] `v1.16.0` is live (`/xlearn/api/v1/healthz`). On `main`: m4-02's `internal/platform/consent` and judge's `POST /internal/accounts/{id}/refresh`; m4-05's `SetConsents` and the gateway's judge-refresh helper; m4-06's Settings toggles.
- [ ] No peer tag is planned before l-04's `v1.17.0` (`git ls-remote --tags origin`, `gh pr list`, ListAgents).

## Do this (in order)

1. **[X] Branch** `feat/la-front-door` from an up-to-date `main`.
2. **[X] identity acceptance** (plan task 2):
   - `notice.go` (`NoticeVersion = "privacy-notice@1"`, `AgeAttestVersion = "age-18@1"`); the `privacy_notice` and `age_18` kinds in `consent.go`; `region.go` (the ISO 3166-1 alpha-2 table). The AI kind constants come from m4-02's `internal/platform/consent`.
   - session-validate `accepted` = `accepted_at IS NOT NULL` plus a live `privacy_notice` consent at `NoticeVersion`.
   - Two modes: `first` (full form) and `renotice` (an earlier acceptance exists with a live `age_18` row; only the notice is asked again, AB19-F12).
   - The `accept` onboarding step in `handleOnboardingStep`, body `{step, notice_version, agree_notice, age_18, region, consents}`: 409 `notice_changed`; 422 `notice_required`; in `first` mode 422 `age_required` / `invalid_region`; in `renotice` mode only `notice_version` and `agree_notice` are read. Then `AcceptTerms` in one transaction: it supersedes the `privacy_notice` row (and in `first` mode the `age_18` row), sets `accepted_at` (and in `first` mode `region`), and in `first` mode writes the two AI kinds **only by calling m4-05's `SetConsents` inside the same transaction** (split out a tx-scoped core if needed, with no behaviour change). That inherits the append-only history, the graded ⇒ passing withdrawal and 422 `requires_ai_review_graded`. Never write AI consent rows any other way.
   - The order guard (other steps → 409 `acceptance_required`), and `/me.acceptance` (`required`, `mode`, `notice_version`, `region`, `consents`). Measure the `/sessions/validate` p95 in compose before and after.
3. **[X] Gateway gate** (plan task 3):
   - `internal/gateway/acceptance.go` with the exempt set `GET /api/me`, `POST /api/onboarding/step`, `POST /api/auth/logout`, `DELETE /api/me` (an API-level safeguard only; the SPA has no erase path while acceptance is required).
   - Apply it in the **one session-validate path both `authAccount` and `authSession` use** (make `authAccount` delegate if it still calls `validateSession` directly) → 403 `acceptance_required`.
   - After a 200 `accept`, `handleOnboardingStep` calls judge `POST /internal/accounts/{id}/refresh` when `JUDGE_BASE_URL` is set, reusing m4-05's helper: one retry, then an ERROR log (D34), and the 200 is returned anyway.
   - Tests: the route-walk over `apiRoutes()` (handlers on `authAccount` and `authSession`, coach SSE, a version bump); the refresh with a fake judge (called once on success, retried and logged on failure, not called otherwise or without `JUDGE_BASE_URL`).
4. **[X] Auth page** (plan task 1): `useAuthConfig`, `useInviteCheck`, `lib/invite.ts` (capture, `replaceState`, storage fallback, clear), `lib/contact.ts` (`INVITE_REQUEST_MAILTO`: the address from the launch message, else the public one on this repo's commits; record which), and the three modes per the plan's table. Hide Sign up while `closed`, add the hidden `invite` input on the GitHub form, the AB19 error copy, and the privacy link. New components go in `web/src/components/auth/`.
5. **[X] Acceptance step** (plan task 4):
   - `Acceptance.tsx` per AB19 **F9–F14**, with AB19's copy verbatim: the required **"I'm 18 or older."** box; the required **"I've read the privacy notice (version 1, effective …) and agree to it."** box (the link opens `/privacy` in a new tab; sends `agree_notice: true`); the region select via `Intl.DisplayNames`; the "xLearn AI (optional)" fieldset with the two AI consents, the second disabled until the first and cleared when the first is unticked. Buttons: **[Continue]** / **[Sign out]**.
   - Consent defaults: **unticked** with no live grant; an account with live Settings grants sees them ticked and an untick withdraws. This is a **flagged deviation** from F9/F11's "both unticked": log it and call it out in the PR (the owner may revisit it after the merge; nothing waits on it, D40).
   - F10: inline errors on submit, focus on the first invalid item, [Continue] never disabled. F12 (`mode: "renotice"`): the "We've updated the privacy notice" banner with [What changed] → `/privacy#what-changed`, and **only** the notice box (no 18+, region or consent controls). F13 save-failed, F14 390 px.
   - Step 0 in `Onboarding`, before `firstUnfinishedStep`; already-onboarded accounts go straight into the app after it.
   - `AuthedShell` redirects on `acceptance.required`; `api.ts` gets `isAcceptanceRequired`; `queryClient.ts` gets a global `onError` → invalidate `me`.
6. **[X] Privacy notice** (plan task 5):
   - A public `/privacy` route outside AuthedShell (test that it ranks above `/:course`); `Privacy.tsx` per AB20 F1–F5 (the F2 "What changed" block at `#what-changed`, absent at version 1; the F3 Terms section), rendering `web/src/content/privacy-notice.md` with m1-06's `Markdown.tsx`; `lib/notice.ts` with the Go parity test.
   - Confirm `privacy` is in both course-slug guard lists. Add the links in the auth footer, the acceptance step and Settings.
   - **Draft the notice text** from AB20-F1's sections against the plan's checklist, verifying every claim in code and the ADRs (Anthropic, ≤ 30-day retention with the flagged-content caveat, no training, outside India, the toggles, 18+, region, the D12 ~7-day loss window, erase and its 60-day username hold, **a pre-release snapshot may hold data for up to 1 day** (ADR-0034 §4.3), the public profile, no email, the terms, contact, version).
   - The erase claim ("in Settings, for every non-owner") is true only once [l-04](../sprints/sprint-l-04.md) task 1 opens web erase to learners in the same `v1.17.0`. Say so in the PR; l-04 reconfirms it before the tag.
7. **[X] L-C** (plan task 6): acceptance writes the AI kinds only through m4-05's `SetConsents`, with m4-02's `internal/platform/consent` constants, and the gateway refreshes judge after `accept`. Tests, against m4-02's internal-read shape (live rows only, `{kind: {version, granted_at}}`): both unticked → neither kind present; first ticked → only `ai_review_graded`; a Settings grant unticked at acceptance → both gone (cascade). judge's no-consent case is built from acceptance-written rows, including an immediate withdrawal (and a cross-service e2e if the harness has identity, gateway and judge).
8. **[X] Fixtures** (plan task 7): default the gateway's fake identity to `accepted: true`; add an `internal/e2e` helper that posts a `first`-mode `accept` for each created account and call it in the existing flows; add `acceptance: {required: false, …}` to the web `/me` fixtures. Leave the compose dev login un-accepted so the smoke shows step 0.
9. **[X] Verify:**
   - `gofmt -l`, `go vet ./...`, `go test -race ./...`, `go test -tags e2e ./internal/e2e/...` (including the plan's task 7 flow), `sqlc diff`, the OpenAPI drift test, and web typecheck/lint/test/build.
   - `docker compose up --build`: the auth page in `open`, `invite` (with and without a code; mint with `docker compose exec identity identity admin invite create --note test`) and `closed` (compose override); `/xlearn/privacy` logged out; the acceptance step for the dev account (then bump the notice version locally and check the F12 notice-only form); then every screen with the console open. Expect **zero CSP violations**, and no API call before acceptance outside the exempt four (network panel).
   - Screenshots at 1440 px and 390 px beside AB19 and AB20.
10. **[X] PR** with conventional commits (`feat(web): …`, `feat(identity): …`, `feat(gateway): …`, `docs: …`) and the attribution lines, with screenshots, a rendered-notice link, the `mailto:` address used, and the two flagged items (the consent pre-fill deviation from AB19-F9/F11, and the learner-erase claim that l-04 makes true), logged for the owner's later look. The notice lands as drafted (D40): CI green → squash-merge (see Ship). **No tag:** l-04 tags `v1.17.0`.

## Constraints

- **Service boundaries ([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)):** acceptance state lives in identity (`account_consent`, `account.accepted_at`, `account.region`). The gateway only enforces using session-validate's `accepted`, and judge reads consents over identity's internal HTTP. No cross-schema read.
- **goose + sqlc:** no migration is expected. If one is needed, it is expand only at the next free version. **`sqlc diff` clean.**
- **No outbox event is added.** Acceptance and consent changes aren't events; judge reads them on demand with a 5-minute cache, and the gateway's refresh call after `accept` (m4-05's pattern) drops the cached entry at once. gateway → judge `/internal/*` already exists (m4-05), so no NetworkPolicy change.
- **Frontend:** `theme.css` tokens and components verbatim, the dark theme, AB19★ and AB20 as frozen. m1-04's CSP: no `<style>` element, no inline script, no `'unsafe-inline'`.
- **Consents:** unticked by default, withdrawable, specific (two separate boxes; the behavioral opt-in only in Settings). Never pre-tick for a new account, and show no consent controls on re-acceptance. Write AI consent rows **only** through m4-05's `SetConsents`.
- **Invite code hygiene:** it is read only from the fragment, stripped with `replaceState`, and never put in a query string, a log or storage beyond `sessionStorage`.
- **GitOps:** no infra change, and never `kubectl apply`. Production `SIGNUP_MODE` stays `closed`.
- **D34:** no alerting. Nothing here adds a log-based alert.
- **Memory-sum rule:** no new pod or container, so it is unaffected.
- **Release order:** merge only, riding `v1.17.0`. Check peers' tags and PRs, and merge only when no peer tag would carry this before l-04. Check peers' ADR numbers before claiming one; none is expected.

## Deliverables

- identity: `notice.go`, `region.go`, the consent kinds, the `accept` step (`first` and `renotice`) with `AcceptTerms` calling `SetConsents`, session-validate `accepted` tied to the notice version, `/me.acceptance`, and the order guard.
- gateway: `acceptance.go` (the exempt set and the 403) in the shared session-validate path, the judge refresh after `accept`, the route-walk and refresh tests, and fakes defaulting to accepted.
- web: the invite-aware auth page, `lib/invite.ts`, `lib/contact.ts`, `components/auth/{InviteNotice,Acceptance}.tsx`, the AuthedShell and API 403 handling, the `/privacy` route, `Privacy.tsx`, `content/privacy-notice.md`, and `lib/notice.ts` with the parity test.
- Tests: vitest, Go unit and integration, e2e (with the new accept helper in the existing flows).
- Docs: `api.md` + `openapi.yaml` (`acceptance` on `/me`, the `accept` step and its two modes, `acceptance_required`, `notice_changed`, `notice_required`, `age_required`, `invalid_region`, `requires_ai_review_graded`); `data-model.md` (the consent kinds).

## Update status

- The plan's Status table ([`../sprints/sprint-l-05.md`](../sprints/sprint-l-05.md)): tasks 🔄 → ✅ (task 8 ✅ at the merge: the notice lands as drafted, D40); _Overall_ ✅.
- [`../status.md`](../status.md):
  - the Sprint board row: l-05 ✅ "merged, ships in v1.17.0";
  - **confirm** the artboard rows AB19 and AB20 read "frozen" (set at ds-l-01's merge; don't rewrite them);
  - owner event `ev-notice-text` → "moved to the v3 opening gates (rollout §11, D40)"; no v2 approval;
  - the L milestone note "front door merged".
- Decisions log:
  - the exempt set, with `DELETE /api/me` as an API-level safeguard only (no SPA erase path while acceptance is required; decliners ask the owner for a CLI erase; a web entry point is a board delta flagged for v3);
  - the AI consents at acceptance go through m4-05's `SetConsents`, plus the judge refresh after `accept`;
  - **flagged deviation from AB19-F9/F11:** live Settings grants are shown ticked, and an untick is a withdrawal;
  - the notice-only re-acceptance per AB19-F12;
  - the notice-version bump rule (substantive change bumps; typo doesn't);
  - `INVITE_REQUEST_MAILTO` as a build-time constant, and where its address came from;
  - no invite check in `closed`;
  - the notice landing as drafted, with its owner review moved to the v3 opening gates (D40).

## Done when (acceptance)

- [ ] 403 `acceptance_required` until acceptance on every non-exempt session route (`authAccount` and `authSession` handlers alike); a notice-version change re-prompts for the notice only (AB19-F12).
- [ ] Both required boxes (18+, the notice agreement) are enforced in the SPA and the server; the button is [Continue].
- [ ] The consents are unticked by default and written only through `SetConsents`; judge is refreshed after `accept` and honours them (tested from acceptance-written rows).
- [ ] Existing gateway, e2e and web suites are green with fixtures that default to accepted.
- [ ] The notice page is reachable logged out; `privacy` is reserved in both guards; the version parity test is green.
- [ ] Auth page: no Sign up while `closed`; invite-only plus `mailto:`; the fragment is stashed and stripped; AB19's error states.
- [ ] Screens match AB19★ and AB20; no CSP violation.
- [ ] The notice text landed as drafted (D40); its owner review (`ev-notice-text`) is recorded as a v3 opening gate.
- [ ] CI green (`sqlc diff`); merged to `main`, untagged.

## Ship (land-and-sync — owner approval pre-granted)

> Launching this prompt is the owner's approval for every change it makes (D40); don't stop for review.

1. Branch `feat/la-front-door`, then conventional commit(s) with the attribution lines, then push, then the PR. This repo only: no `../infra` PR.
2. Once CI is green (fix, then merge, on failure), squash-merge — with the drafted notice as written; there is no notice-approval stop. Never enable auto-merge.
3. **Release action — merge only:** nothing deploys (`main` is build-only). It ships in **`v1.17.0`**, which [l-04](../sprints/sprint-l-04.md) tags right after; never in an M3 or M4 tag. No tag here.
4. Update status: the sprint file and `docs/v2/status.md`, in the same PR or a follow-up docs PR merged the same way.
5. Run `git checkout main && git pull`. If a clean peer worktree holds `main`, use `git -C <worktree> merge --ff-only origin/main` and then `git switch --detach main`.

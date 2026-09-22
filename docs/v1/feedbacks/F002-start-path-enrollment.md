# F002 — Start-path enrollment (current day · streak · today)

**Status:** ✅ done (local review) · **Opened:** 2026-09-22

## Feedback

Don't keep DSA **default-active**. Give the option to **start the path**; once started,
display the **current day**, the **active streak**, and — if something is scheduled for that
path today (a revision or interview practice per the curriculum schedule) — surface it.

## Decision

**Server-backed enrollment** (chosen over derive-from-onboarding / frontend-only):

- New per-user **`path_enrollment`** owned by **identity** (`account_id`, `path_slug`,
  `started_at`, `status`) — durable, cross-device, and multi-path-ready.
- `POST /paths/{slug}/start` (JWT-scoped, idempotent upsert) enrolls; enrollments ride
  `GET /me` so every screen knows started-state without a second call.
- **Current day** = whole days since `started_at` + 1 (learner-local), computed from the
  enrollment. **Streak** and **today's scheduled** reuse the existing `/dashboard` agg
  (assessment streak + review due-queue) — no new aggregation.
- The Catalog DSA card becomes enrollment-aware: **Start path** when not enrolled →
  **Day N · streak · today** + **Continue** once enrolled. The global seed `status:"active"`
  now means *"this path is available to start"*, not *"pre-started for everyone"*.
- Onboarding's `path_chosen` is left as-is (interest signal); enrollment is the explicit
  Catalog action. Not auto-enrolling from onboarding this batch (revisit if desired).

Dev-login + the enrollment migration are captured in **ADR-0022**.

## Scope / changes

- identity: `store/migrations/000NN_path_enrollment.sql`, sqlc queries, handlers
  (`POST /paths/{slug}/start`, enrollments in `GET /me`), config.
- gateway: proxy `POST /paths/{slug}/start`; pass enrollments through `/me`.
- web: `lib/auth.ts` (enrollments on `me`), `lib/enrollment.ts` (start mutation, day calc),
  `screens/Catalog.tsx` (Start vs started card), `screens/Dashboard.tsx` (Day N header).

## Changes done

- identity: `store/migrations/00002_path_enrollment.sql`, `store/queries/enrollment.sql`
  (idempotent upsert), regenerated sqlc, `Enrollment` type + `StartEnrollment`/`ListEnrollments`
  on the store, `handleStartEnrollment` (`POST /paths/{slug}/start`) + enrollments on `GET /me`.
- gateway: `handleStartPath` + `identityClient.startEnrollment`; the `POST /api/paths/{slug}/start`
  route (added to `openapi.yaml`, drift check green); enrollments pass through `/me`.
- web: `lib/enrollment.ts` (`useStartPath`, `currentDay`, `enrollmentFor`), `lib/auth.ts`
  (`enrollments` on `Me`), Catalog Start↔started card, Dashboard "Day N" header.
- Dev login for local review: `internal/identity/devauth.go` + `DEV_AUTH` config + the gateway
  `/api/auth/dev/*` passthrough + the SPA "Dev sign in" button (`useDevAuthEnabled`/`useDevLogin`).
- ADR: [ADR-0022](../../adr/0022-path-enrollment-and-dev-login.md).

**Verified live** (docker-compose): `POST /paths/dsa/start` enrolls (idempotent — re-start keeps
`started_at`); Catalog flips Start → **Day 1 · 0-day streak · New problems ready** + Continue; the
Dashboard header reads "Day 1 · …".

## Notes

- A freshly-started path honestly shows **Day 1 · 0-day streak · nothing scheduled** until the
  learner solves; no faked progress.
- The `/dashboard` agg is DSA-pinned in the gateway today; fine while DSA is the only real path.

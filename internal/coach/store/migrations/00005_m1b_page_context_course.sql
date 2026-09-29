-- v2 M1b DATA (sprint m1-03 task 5; t0 §7, t1 §4 coach): every course-scoped coach page
-- context becomes `<course>:<ctx>`, and coach_thread / coach_message carry the course in
-- path_slug (m1-02 added both columns, nullable).
--
-- The thread key is coach_thread UNIQUE (account_id, page_context): without the prefix,
-- one course's week 3 (or its Revision, Mock, …) would share a thread with another
-- course's. v1 was DSA-only, so every v1 course-scoped context is the DSA course's:
--
--   * path_slug = 'dsa' on the course-scoped threads (concept:<slug>, week:<n>, roadmap,
--     dashboard, revision, mistakes, mock, progress, and an already-prefixed dsa:<ctx>)
--     and on their messages. problem:<id> and the account-wide contexts (catalog,
--     settings, general) stay NULL here; v1.7.0 labels a problem thread on its next chat.
--   * page_context <ctx> → dsa:<ctx> for the v1 forms: the key v1.7.0's dual parser
--     (course.NormalizeCoachContext) maps them to. The patterns mirror that parser, so a
--     context it passes through unchanged (e.g. `concept:` with no slug) is left alone.
--
-- Idempotent: each statement skips the rows it has already done (path_slug IS NULL,
-- NOT LIKE 'dsa:%'), so a re-run is a no-op. Data only, no contract statement: v1.6.0
-- (the R-b target) reads the same columns, and after an R-b its coach no longer finds a
-- rewritten thread under the v1 key, so that page starts a fresh thread. Production had
-- 0 coach rows (read-only check, 2026-09-25).

-- +goose Up

-- (a) The course-scoped threads are the DSA course's …
-- +goose StatementBegin
UPDATE coach.coach_thread
SET path_slug = 'dsa'
WHERE path_slug IS NULL
  AND page_context ~ '^(dsa:)?((concept|week):[^:]+|roadmap|dashboard|revision|mistakes|mock|progress)$';
-- +goose StatementEnd

-- … and so are their messages: a message carries its thread's course.
-- +goose StatementBegin
UPDATE coach.coach_message m
SET path_slug = t.path_slug
FROM coach.coach_thread t
WHERE m.thread_id = t.id
  AND m.path_slug IS NULL
  AND t.path_slug IS NOT NULL;
-- +goose StatementEnd

-- (b) The v1 forms take the DSA prefix. A v1 row whose dsa: twin already exists for the
-- account would break UNIQUE (account_id, page_context), so it is skipped and kept as it
-- is: v1.7.0 never reads it again (its context maps to the twin's key), nothing is
-- deleted, and a v1.6.0 coach still finds it. A twin appears when, during the rolling
-- update, a v1.7.0 gateway (which normalizes contexts) reached the v1.6.0 coach for a
-- page that already had a v1 thread; or on a manual re-run.
-- +goose StatementBegin
UPDATE coach.coach_thread t
SET page_context = 'dsa:' || t.page_context
WHERE t.page_context NOT LIKE 'dsa:%'
  AND t.page_context ~ '^((concept|week):[^:]+|roadmap|dashboard|revision|mistakes|mock|progress)$'
  AND NOT EXISTS (
      SELECT 1 FROM coach.coach_thread twin
      WHERE twin.account_id = t.account_id
        AND twin.page_context = 'dsa:' || t.page_context
  );
-- +goose StatementEnd

-- +goose Down
-- Nothing to undo (ADR-0034 §3: roll forward): the prefixed keys and path_slug stay, and
-- v1.6.0 reads neither.

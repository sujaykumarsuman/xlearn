-- v2 M1a EXPAND (sprint m1-02; ADR-0034 §3, t1 §4 practice): the per-user rows gain
-- their course (path_slug) BESIDE the v1 shape. Additive only — a constant-default
-- column, nullable columns and set-based, idempotent backfills — so the v1.5.2 image
-- (the R-b target while the rollback floor is "none") still runs on this schema: its
-- INSERTs omit the new columns (defaults / NULL apply) and its `SELECT *` queries were
-- expanded to explicit column lists by sqlc, so the extra columns are never scanned.
--
-- attempt.{account_id, path_slug, problem_id} denormalise the owning
-- user_problem_state row (M2a's touch attempts carry them without a problem state);
-- v1.6.0 writers set all three on insert. outbox.account_id is erase prep (ADR-0027 §6):
-- it is backfilled from the envelope and set by the outbox writer on insert.

-- +goose Up

-- +goose StatementBegin
ALTER TABLE practice.user_problem_state
    ADD COLUMN IF NOT EXISTS path_slug text NOT NULL DEFAULT 'dsa';
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE practice.attempt
    ADD COLUMN IF NOT EXISTS account_id uuid,
    ADD COLUMN IF NOT EXISTS path_slug text,
    ADD COLUMN IF NOT EXISTS problem_id text;
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE practice.attempt AS a
SET account_id = s.account_id,
    path_slug  = s.path_slug,
    problem_id = s.problem_id
FROM practice.user_problem_state AS s
WHERE s.id = a.user_problem_state_id
  AND a.account_id IS NULL;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE practice.outbox ADD COLUMN IF NOT EXISTS account_id uuid;
-- +goose StatementEnd

-- Every v1 envelope carries a uuid account_id; the regex guard keeps a malformed row
-- from failing the migration (it just stays NULL).
-- +goose StatementBegin
UPDATE practice.outbox
SET account_id = (payload_json ->> 'account_id')::uuid
WHERE account_id IS NULL
  AND payload_json ->> 'account_id' ~* '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$';
-- +goose StatementEnd

-- +goose Down
-- Never run in production (ADR-0034 §3: roll forward). Local/dev only.
-- +goose StatementBegin
ALTER TABLE practice.outbox DROP COLUMN IF EXISTS account_id;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE practice.attempt
    DROP COLUMN IF EXISTS problem_id,
    DROP COLUMN IF EXISTS path_slug,
    DROP COLUMN IF EXISTS account_id;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE practice.user_problem_state DROP COLUMN IF EXISTS path_slug;
-- +goose StatementEnd

-- v2 M1b EXPAND (sprint m1-07 task 2; t5 §9, ADR-0031 §7, ADR-0034 §3): every chat turn
-- records HOW it was produced, so a reply can always be traced to the prompt that shaped it
-- and to the attempt it was asked during.
--
--   * prompt_v   — the system-prompt version the turn was built with (coach-prompt@2 from
--                  v1.7.0 on). NULL on every v1 row: v1's prompt had no version.
--   * attempt_id — the open counted attempt the turn belongs to, set only when the gateway
--                  recorded a D27 coach assist on it (X-Coach-Attempt). NULL otherwise. It
--                  is practice's id, deliberately NOT a foreign key: coach never reads
--                  another service's schema (ADR-0005).
--
-- m1-02 already added path_slug, so the user row and the assistant row of a turn now carry
-- all three. Both are written on BOTH rows of a turn with the same values.
--
-- Additive only, both columns nullable, no backfill: v1.6.0 (the R-b floor) neither reads
-- nor writes them, and its InsertMessage simply leaves them NULL. This is an EXPAND, so it
-- deliberately carries no contract marker and no rollback floor; hack/lint-migrations.sh
-- enforces that in both directions.

-- +goose Up

-- +goose StatementBegin
ALTER TABLE coach.coach_message ADD COLUMN IF NOT EXISTS prompt_v text;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE coach.coach_message ADD COLUMN IF NOT EXISTS attempt_id uuid;
-- +goose StatementEnd

-- +goose Down
-- Never run in production (ADR-0034 §3: roll forward). Local/dev only.
-- +goose StatementBegin
ALTER TABLE coach.coach_message DROP COLUMN IF EXISTS attempt_id;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE coach.coach_message DROP COLUMN IF EXISTS prompt_v;
-- +goose StatementEnd

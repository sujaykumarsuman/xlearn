-- v2 M1a EXPAND (sprint m1-02; ADR-0034 §3, t1 §4 coach, t5 §9): coach's per-course and
-- per-feature shape lands BESIDE the v1 columns.
--
--   * coach_thread / coach_message gain a nullable path_slug (m1-03 writes it; m1-07
--     then adds only prompt_v and attempt_id).
--   * key_default(account_id, feature) is the per-feature default key that replaces
--     api_key_config.is_default (m1-10 makes it the only source; M1c drops is_default).
--     It is backfilled from the is_default rows (model from default_model), every v1.6.0
--     writer of is_default dual-writes it, and readers prefer it, falling back to
--     is_default. ON DELETE CASCADE removes the row with its key.
--   * api_key_config.is_default becomes write-optional (keeps DEFAULT false), so v1.7.0
--     can stop writing it while an R-b to v1.6.0 still reads its rows.
--
-- Additive only: the v1.5.2 image (the R-b target while the floor is "none") keeps
-- writing is_default and never sees key_default.

-- +goose Up

-- +goose StatementBegin
ALTER TABLE coach.coach_thread ADD COLUMN IF NOT EXISTS path_slug text;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE coach.coach_message ADD COLUMN IF NOT EXISTS path_slug text;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS coach.key_default (
    account_id uuid        NOT NULL,
    feature    text        NOT NULL DEFAULT 'coach' CHECK (feature IN ('coach', 'interview')),
    key_id     uuid        NOT NULL REFERENCES coach.api_key_config (id) ON DELETE CASCADE,
    model      text        NOT NULL DEFAULT '',
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (account_id, feature)
);
-- +goose StatementEnd

-- The cascade looks rows up by key_id.
-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS key_default_key_id_idx ON coach.key_default (key_id);
-- +goose StatementEnd

-- One coach default per account: the is_default row (the earliest if a bug ever left
-- two, matching PromoteEarliestDefault's order).
-- +goose StatementBegin
INSERT INTO coach.key_default (account_id, feature, key_id, model)
SELECT DISTINCT ON (account_id) account_id, 'coach', id, default_model
FROM coach.api_key_config
WHERE is_default
ORDER BY account_id, created_at, provider
ON CONFLICT (account_id, feature) DO NOTHING;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE coach.api_key_config ALTER COLUMN is_default DROP NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- Never run in production (ADR-0034 §3: roll forward). Local/dev only.
-- +goose StatementBegin
UPDATE coach.api_key_config SET is_default = false WHERE is_default IS NULL;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE coach.api_key_config ALTER COLUMN is_default SET NOT NULL;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE IF EXISTS coach.key_default;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE coach.coach_message DROP COLUMN IF EXISTS path_slug;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE coach.coach_thread DROP COLUMN IF EXISTS path_slug;
-- +goose StatementEnd

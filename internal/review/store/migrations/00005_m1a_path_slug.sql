-- v2 M1a EXPAND (sprint m1-02; ADR-0034 §3, t1 §4 review, ADR-0026 §3): every per-user
-- review row gains its course (path_slug) BESIDE the v1 shape. Additive only —
-- constant-default and nullable columns plus a set-based, idempotent backfill — so the
-- v1.5.2 image (the R-b target while the rollback floor is "none") still runs on this
-- schema: its INSERTs omit the new columns (the 'dsa' default applies) and sqlc expanded
-- its `SELECT *` queries to explicit column lists.
--
-- reminder.path_slug stays nullable (a reminder can be account-wide). outbox.account_id
-- is erase prep (ADR-0027 §6): backfilled from the envelope, set by the outbox writer.
--
-- The (account_id, path_slug, week_of) unique for weak_area_snapshot is built
-- CONCURRENTLY in 00006 (NO TRANSACTION). The v1 UNIQUE (account_id, week_of) STAYS:
-- v1.5.2's upsert targets it, and dropping it is contract work (m1-08's drop list).

-- +goose Up

-- +goose StatementBegin
ALTER TABLE review.revision_item ADD COLUMN IF NOT EXISTS path_slug text NOT NULL DEFAULT 'dsa';
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE review.mistake_entry ADD COLUMN IF NOT EXISTS path_slug text NOT NULL DEFAULT 'dsa';
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE review.weak_area_snapshot ADD COLUMN IF NOT EXISTS path_slug text NOT NULL DEFAULT 'dsa';
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE review.reminder ADD COLUMN IF NOT EXISTS path_slug text;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE review.outbox ADD COLUMN IF NOT EXISTS account_id uuid;
-- +goose StatementEnd

-- Every v1 envelope carries a uuid account_id; the regex guard keeps a malformed row
-- from failing the migration (it just stays NULL).
-- +goose StatementBegin
UPDATE review.outbox
SET account_id = (payload_json ->> 'account_id')::uuid
WHERE account_id IS NULL
  AND payload_json ->> 'account_id' ~* '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$';
-- +goose StatementEnd

-- +goose Down
-- Never run in production (ADR-0034 §3: roll forward). Local/dev only.
-- +goose StatementBegin
ALTER TABLE review.outbox DROP COLUMN IF EXISTS account_id;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE review.reminder DROP COLUMN IF EXISTS path_slug;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE review.weak_area_snapshot DROP COLUMN IF EXISTS path_slug;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE review.mistake_entry DROP COLUMN IF EXISTS path_slug;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE review.revision_item DROP COLUMN IF EXISTS path_slug;
-- +goose StatementEnd

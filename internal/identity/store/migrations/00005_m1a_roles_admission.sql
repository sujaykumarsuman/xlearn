-- v2 M1a EXPAND (sprint m1-02; ADR-0033 §4, §7, §13; t1 §4 identity; D7): the account's
-- role, status, admission provenance and public visibility, the per-course visibility
-- flag on enrollments, and the owner admin audit log. Additive only — constant-default
-- or nullable columns, a new table and one set-based, idempotent backfill — so the
-- v1.5.2 image (the R-b target while the rollback floor is "none") still runs here: its
-- INSERTs omit the new columns (defaults / NULL apply; every CHECK accepts them) and
-- sqlc expanded its `SELECT *` queries to explicit column lists.
--
-- NOTHING READS these columns in v1.6.0: sessions, RequireRole, the admin CLI and the
-- public filters are M1b (m1-04, m1-05). Roles and status never enter the JWT (§7).
--
-- admitted_via: every row that exists when this runs is 'grandfathered' — the owner's
-- included (ADR-0033 §2: accounts created before v1.5.2 are marked grandfathered in
-- M1a; production has exactly one account, the owner's, created before v1.5.2).
-- set-role (m1-04) changes role only, never admitted_via, and nothing gates on it. New
-- rows: dev login and open-mode signup (compose only) write 'dev'; 'cli' arrives with
-- m1-04 and 'invite' with l-03. invite_id has no FK until L-A creates identity.invite.

-- +goose Up

-- +goose StatementBegin
ALTER TABLE identity.account
    ADD COLUMN IF NOT EXISTS role text NOT NULL DEFAULT 'learner'
        CONSTRAINT account_role_check CHECK (role IN ('learner', 'tester', 'owner')),
    ADD COLUMN IF NOT EXISTS status text NOT NULL DEFAULT 'active'
        CONSTRAINT account_status_check CHECK (status IN ('active', 'suspended')),
    ADD COLUMN IF NOT EXISTS admitted_via text
        CONSTRAINT account_admitted_via_check CHECK (admitted_via IN ('grandfathered', 'invite', 'cli', 'dev')),
    ADD COLUMN IF NOT EXISTS invite_id uuid,
    ADD COLUMN IF NOT EXISTS accepted_at timestamptz,
    ADD COLUMN IF NOT EXISTS region text,
    ADD COLUMN IF NOT EXISTS profile_visibility text NOT NULL DEFAULT 'public'
        CONSTRAINT account_profile_visibility_check CHECK (profile_visibility IN ('public', 'private'));
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE identity.account SET admitted_via = 'grandfathered' WHERE admitted_via IS NULL;
-- +goose StatementEnd

-- The per-course public flag (D7): StartEnrollment writes the course manifest's
-- public_stats.default_visible on insert; existing rows are DSA (default_visible true).
-- +goose StatementBegin
ALTER TABLE identity.path_enrollment
    ADD COLUMN IF NOT EXISTS public_visible boolean NOT NULL DEFAULT true;
-- +goose StatementEnd

-- The owner admin CLI's audit log (m1-04 writes it; ADR-0033 §8). target is the
-- affected id (an account or invite id), never an email.
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS identity.admin_audit (
    id     uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    at     timestamptz NOT NULL DEFAULT now(),
    verb   text        NOT NULL,
    target text        NOT NULL,
    detail jsonb       NOT NULL DEFAULT '{}'::jsonb
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS admin_audit_at_idx ON identity.admin_audit (at DESC);
-- +goose StatementEnd

-- +goose Down
-- Never run in production (ADR-0034 §3: roll forward). Local/dev only.
-- +goose StatementBegin
DROP TABLE IF EXISTS identity.admin_audit;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE identity.path_enrollment DROP COLUMN IF EXISTS public_visible;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE identity.account
    DROP COLUMN IF EXISTS profile_visibility,
    DROP COLUMN IF EXISTS region,
    DROP COLUMN IF EXISTS accepted_at,
    DROP COLUMN IF EXISTS invite_id,
    DROP COLUMN IF EXISTS admitted_via,
    DROP COLUMN IF EXISTS status,
    DROP COLUMN IF EXISTS role;
-- +goose StatementEnd

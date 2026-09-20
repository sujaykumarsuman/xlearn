-- curriculum schema: the read-heavy content rail — paths, phases, weeks, concepts,
-- problems and their stage-scoped content sections (ADR-0005; data-model.md
-- schema `curriculum`). All DDL is schema-qualified so the xlearn_curriculum role
-- only ever touches schema `curriculum`. Curriculum emits/consumes no events, so
-- there is NO outbox here (unlike identity).
--
-- Curriculum is static, seeded content (from the versioned files under curriculum/,
-- applied idempotently on startup) — not user-mutated records — so the rows carry
-- no created_at/updated_at audit columns.
--
-- Schema OWNERSHIP is a platform concern, not this migration's: schema `curriculum`
-- is provisioned by the CNPG Database CR (spec.schemas, owner xlearn_curriculum) so
-- the least-privilege role can create tables in it without CREATE on the database.
-- For local/dev, create it once out-of-band (mirrors prod):
--   CREATE SCHEMA curriculum AUTHORIZATION xlearn_curriculum;
--   ALTER ROLE xlearn_curriculum SET search_path = curriculum;

-- +goose Up

-- A learning path (e.g. `dsa`). `summary` + `sort_order` are content-supporting
-- columns beyond data-model.md's key set so Catalog is fully API-driven (the card
-- blurb + display order come from the API, not hard-coded in the SPA).
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS curriculum.path (
    slug          text    PRIMARY KEY,
    title         text    NOT NULL,
    status        text    NOT NULL CHECK (status IN ('active', 'coming_soon')),
    summary       text    NOT NULL DEFAULT '',
    problem_total integer NOT NULL DEFAULT 0,
    week_total    integer NOT NULL DEFAULT 0,
    sort_order    integer NOT NULL DEFAULT 0
);
-- +goose StatementEnd

-- A phase groups a contiguous week range. `name` is the display name
-- ("Fundamentals"); `theme` is the one-line subtitle. "order" is a reserved word,
-- so it is always quoted. (data-model.md: order, theme, week_from, week_to.)
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS curriculum.phase (
    id        uuid    PRIMARY KEY DEFAULT gen_random_uuid(),
    path_slug text    NOT NULL REFERENCES curriculum.path (slug) ON DELETE CASCADE,
    "order"   integer NOT NULL,
    name      text    NOT NULL,
    theme     text    NOT NULL DEFAULT '',
    week_from integer NOT NULL,
    week_to   integer NOT NULL,
    UNIQUE (path_slug, "order")
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS curriculum.week (
    id        uuid    PRIMARY KEY DEFAULT gen_random_uuid(),
    path_slug text    NOT NULL REFERENCES curriculum.path (slug) ON DELETE CASCADE,
    n         integer NOT NULL,
    title     text    NOT NULL,
    thesis    text    NOT NULL DEFAULT '',
    UNIQUE (path_slug, n)
);
-- +goose StatementEnd

-- A concept/pattern reading. slug is globally unique (the read API is
-- GET /concepts/{slug}, no path scope); path_slug is retained for grouping.
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS curriculum.concept (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    path_slug      text NOT NULL REFERENCES curriculum.path (slug) ON DELETE CASCADE,
    slug           text NOT NULL UNIQUE,
    title          text NOT NULL,
    body_md        text NOT NULL DEFAULT '',
    when_to_use_md text NOT NULL DEFAULT '',
    code_template  text NOT NULL DEFAULT ''
);
-- +goose StatementEnd

-- Week -> concept links (M:N).
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS curriculum.week_concept (
    week_id    uuid NOT NULL REFERENCES curriculum.week (id) ON DELETE CASCADE,
    concept_id uuid NOT NULL REFERENCES curriculum.concept (id) ON DELETE CASCADE,
    PRIMARY KEY (week_id, concept_id)
);
-- +goose StatementEnd

-- A problem carries a NATURAL id (its curriculum sequence number, e.g. "16" = 3Sum,
-- the hero). week_n is a soft reference to a week number (no FK). difficulty drives
-- the UI tokens: easy=green, med=amber, hard=red.
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS curriculum.problem (
    id               text    PRIMARY KEY,
    path_slug        text    NOT NULL REFERENCES curriculum.path (slug) ON DELETE CASCADE,
    week_n           integer NOT NULL,
    title            text    NOT NULL,
    difficulty       text    NOT NULL CHECK (difficulty IN ('easy', 'med', 'hard')),
    pattern          text    NOT NULL DEFAULT '',
    leetcode_url     text    NOT NULL DEFAULT '',
    neetcode_url     text    NOT NULL DEFAULT '',
    is_reinforcement boolean NOT NULL DEFAULT false,
    sort_order       integer NOT NULL DEFAULT 0
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS problem_path_week_idx
    ON curriculum.problem (path_slug, week_n);
-- +goose StatementEnd

-- Stage-scoped problem content. The read API returns content sections keyed by
-- `stage` (attempt/hint/solution); per-user "only unlocked stages" gating lands in
-- practice (S05). "order" is the position within a stage (quoted; reserved word).
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS curriculum.problem_section (
    id         uuid    PRIMARY KEY DEFAULT gen_random_uuid(),
    problem_id text    NOT NULL REFERENCES curriculum.problem (id) ON DELETE CASCADE,
    stage      text    NOT NULL CHECK (stage IN ('attempt', 'hint', 'solution')),
    kind       text    NOT NULL,
    "order"    integer NOT NULL,
    body_md    text    NOT NULL DEFAULT '',
    code       text    NOT NULL DEFAULT '',
    UNIQUE (problem_id, stage, "order")
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS problem_section_problem_idx
    ON curriculum.problem_section (problem_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS curriculum.problem_section;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE IF EXISTS curriculum.problem;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE IF EXISTS curriculum.week_concept;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE IF EXISTS curriculum.concept;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE IF EXISTS curriculum.week;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE IF EXISTS curriculum.phase;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE IF EXISTS curriculum.path;
-- +goose StatementEnd

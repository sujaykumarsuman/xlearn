-- +goose NO TRANSACTION
-- v2 M1a EXPAND, part 2 (sprint m1-09; ADR-0034 §3): the new composite uniques, built
-- CONCURRENTLY (so no write lock on the table) and therefore outside a transaction.
-- IF NOT EXISTS makes a re-run a no-op; a failed CONCURRENTLY build leaves an INVALID
-- index that IF NOT EXISTS would then skip, so the store test asserts every curriculum
-- index is valid (pg_index.indisvalid).
--
-- The v1 uniques UNIQUE(concept.slug) and problem_section (problem_id, stage, "order")
-- stay until M1c (m1-08's drop list): v1.5.2 upserts target them.

-- +goose Up
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS path_id_prefix_key
    ON curriculum.path (id_prefix);

CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS concept_path_slug_slug_key
    ON curriculum.concept (path_slug, slug);

CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS problem_section_problem_id_stage_order_language_key
    ON curriculum.problem_section (problem_id, stage, "order", language);

-- +goose Down
-- Never run in production (ADR-0034 §3: roll forward). Local/dev only.
DROP INDEX CONCURRENTLY IF EXISTS curriculum.problem_section_problem_id_stage_order_language_key;
DROP INDEX CONCURRENTLY IF EXISTS curriculum.concept_path_slug_slug_key;
DROP INDEX CONCURRENTLY IF EXISTS curriculum.path_id_prefix_key;

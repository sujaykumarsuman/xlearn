-- v2 M3 EXPAND (sprint m3-01; ADR-0027 §2, t1 §3.4, ADR-0034 §3): an item's grading
-- contract hash and its answer-free grading summary, both written by the seed.
--   contract_hash    canon.ContractHash: only what hidden data depends on; '' for a
--                    self-path item. judge (m3-05) and practice (m3-08) compare it with
--                    the pack's accepts_contract_hashes and an attempt's pinned hash.
--   grading_summary  {"mode": "self"|"auto"|"mixed", "parts": [{id, type, grading,
--                    cadence}], "grader_kinds": [...], "languages": [...]}; {"mode":
--                    "self"} for a self-path item. Public, never stage-gated.
-- Additive only, constant defaults, no index: the v1.5.2 image (the R-b target) still
-- migrates-noop, seeds (its INSERT omits both columns) and serves.

-- +goose Up
-- +goose StatementBegin
ALTER TABLE curriculum.problem
    ADD COLUMN IF NOT EXISTS contract_hash text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS grading_summary jsonb NOT NULL DEFAULT '{}'::jsonb;
-- +goose StatementEnd

-- +goose Down
-- Never run in production (ADR-0034 §3: roll forward). Local/dev only.
-- +goose StatementBegin
ALTER TABLE curriculum.problem
    DROP COLUMN IF EXISTS grading_summary,
    DROP COLUMN IF EXISTS contract_hash;
-- +goose StatementEnd

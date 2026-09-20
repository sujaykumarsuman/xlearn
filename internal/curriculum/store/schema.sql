-- Schema declaration for sqlc analysis ONLY. It is NOT a goose migration (goose
-- embeds ./migrations, never this file) and is never executed by the service.
--
-- In production the `curriculum` schema is provisioned + owned by the CNPG Database
-- CR (spec.schemas, owner xlearn_curriculum); the migrations create only tables in
-- it. sqlc, however, needs to know the schema exists to resolve curriculum.* names,
-- so it is declared here and listed first in sqlc.yaml's `schema`.
CREATE SCHEMA IF NOT EXISTS curriculum;

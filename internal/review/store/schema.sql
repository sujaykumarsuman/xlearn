-- Schema declaration for sqlc analysis ONLY. It is NOT a goose migration (goose
-- embeds ./migrations, never this file) and is never executed by the service.
--
-- In production the `review` schema is provisioned + owned by the CNPG Database
-- CR (spec.schemas, owner xlearn_review); the migrations create only tables in
-- it. sqlc, however, needs to know the schema exists to resolve review.* names,
-- so it is declared here and listed first in sqlc.yaml's `schema`.
CREATE SCHEMA IF NOT EXISTS review;

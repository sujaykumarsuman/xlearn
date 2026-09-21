-- Schema declaration for sqlc analysis ONLY. It is NOT a goose migration (goose
-- embeds ./migrations, never this file) and is never executed by the service.
--
-- In production the `coach` schema is provisioned + owned by the CNPG Database CR
-- (spec.schemas, owner xlearn_coach) so the least-privilege role can create tables in
-- it without CREATE on the database. sqlc, however, needs to know the schema exists to
-- resolve coach.* names, so it is declared here and listed first in sqlc.yaml's `schema`.
CREATE SCHEMA IF NOT EXISTS coach;

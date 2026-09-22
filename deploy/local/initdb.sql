-- Local docker-compose Postgres bootstrap. In prod each service's schema + a
-- least-privilege role are provisioned by the CNPG Database CR (../infra); locally we
-- run every service as the single POSTGRES_USER and just pre-create the schemas each
-- service migrates into (its search_path is pinned per service via PGSEARCHPATH). This
-- runs once, on an empty data directory (docker-entrypoint-initdb.d).
CREATE SCHEMA IF NOT EXISTS identity;
CREATE SCHEMA IF NOT EXISTS curriculum;
CREATE SCHEMA IF NOT EXISTS practice;
CREATE SCHEMA IF NOT EXISTS review;
CREATE SCHEMA IF NOT EXISTS assessment;
CREATE SCHEMA IF NOT EXISTS coach;

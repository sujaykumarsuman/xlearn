# ADR-0011 — JWKS publication & shared-DB schema ownership

- **Status:** Accepted
- **Date:** 2026-09-20
- **Deciders:** @sujaykumarsuman
- **Related:** refines [0005](0005-data-ownership-and-migrations.md), [0006](0006-authn-authz.md); realized in S02

## Context

S02 implemented identity + auth and hit two points the higher-level docs left slightly
under-specified. This ADR records the concrete calls so later services follow the same pattern.

1. **Who publishes JWKS?** [ADR-0006](0006-authn-authz.md) says the **gateway** holds the RSA
   signing key (`xlearn-jwt`), mints the JWT, and that "only the gateway holds the signing key" —
   yet [`services.md`](../architecture/services.md) lists `/.well-known/jwks.json` under the
   `identity` service. Having identity publish JWKS would mean shipping the (public half of the)
   signing material to a second service, splitting key custody.

2. **How does a per-service role own its schema in one shared `xlearndb`?** ADR-0005 mandates one
   database, schema-per-service, and a per-service role with privileges only on its own schema —
   but a plain login role cannot `CREATE SCHEMA` without `CREATE` on the database, and granting
   that would let it create objects anywhere.

## Decision

1. **The gateway publishes JWKS.** The JWT **issuer** is the JWKS **publisher**: the gateway serves
   `/.well-known/jwks.json` (at the pod root so in-cluster services reach it on the ClusterIP, and
   harmlessly under `/xlearn` too — the public key is not secret). Downstream services verify via
   the shared `internal/platform/auth` JWKS verifier pointed at the gateway. This supersedes the
   `services.md` note placing JWKS under identity; identity is a **verifier**, not a publisher.

2. **Schema ownership is provisioned declaratively by CNPG, not by migrations.** The `xlearndb`
   `Database` CR (owner = bootstrap role `xlearn`) declares each service schema with its owning
   role via `spec.schemas[].owner` (CNPG ≥ 1.25; the cluster runs v1.30). The service role
   (`xlearn_identity`) therefore **owns** schema `identity` and can create tables in it with no
   `CREATE` on the database and no access to any other schema. Migrations create only tables
   (schema-qualified); they never `CREATE SCHEMA`. `search_path` is pinned to the service schema on
   every pooled connection (belt-and-suspenders; all SQL is schema-qualified regardless).

## Consequences

- ✅ Single key custody: only the gateway ever holds signing material; services stay stateless
  verifiers (matches ADR-0006's intent exactly).
- ✅ Hard schema isolation without a privileged migration step or `CREATE` on the database.
- ✅ The same JWKS-verifier + `Database`-CR-schema pattern drops in for curriculum/practice/review/
  assessment/coach unchanged.
- ⚠️ Requires CNPG ≥ 1.25 for `spec.schemas[].owner`; noted in `xlearn-database.yaml`.
- ⚠️ The gateway is a soft dependency for downstream verification (JWKS fetch). Mitigated by the
  verifier's kid-keyed cache + rotation overlap, so a brief gateway blip does not fail verification.

### Key rotation procedure

Because every gateway replica publishes its own JWKS, a seamless signing-key rotation needs an
**overlap window** where both the outgoing and incoming public keys are published. The gateway
supports this via `JWT_ADDITIONAL_PUBLIC_KEYS` (extra public keys published in JWKS but never used
for minting). Rotate in three full rollouts:

1. Publish `{old, new}` (set `JWT_ADDITIONAL_PUBLIC_KEYS=new.pub`) while still minting with `old`.
2. Switch minting to `new` (`JWT_PRIVATE_KEY=new`), keep publishing `{new, old}`
   (`JWT_ADDITIONAL_PUBLIC_KEYS=old.pub`).
3. After ≥ one JWT TTL, drop `old` (clear `JWT_ADDITIONAL_PUBLIC_KEYS`).

Each step is a normal Flux rollout, so all replicas converge on the same multi-key set and no token
is ever signed by a kid absent from the served JWKS.

## Alternatives considered

| Option | Why not |
|--------|---------|
| **identity publishes JWKS** (per `services.md`) | Splits key custody — identity would need the signing key's public half distributed separately; contradicts "only the gateway holds the signing key." |
| **Migration runs `CREATE SCHEMA`** | Needs `CREATE` on the database for the service role, which also lets it create objects in any schema — breaks the isolation ADR-0005 requires. |
| **Service role owns the whole database** (the infra 4-step default) | Fine for a dedicated DB, but xLearn shares one `xlearndb` across services; the owner-per-schema split is what keeps roles scoped. |
| **Grant CREATE via role membership in the owner** | The service role would inherit the owner's rights over every schema — same isolation break. |

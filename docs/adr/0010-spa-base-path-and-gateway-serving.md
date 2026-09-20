# ADR-0010 — SPA base path & gateway serving model

- **Status:** Accepted
- **Date:** 2026-09-20
- **Deciders:** @sujaykumarsuman
- **Related:** [0008](0008-frontend-stack.md), [0009](0009-deployment-and-gitops.md)

## Context

xLearn is served at `projects.sujaykumar.dev/xlearn`. The gateway embeds the React SPA
(`//go:embed web/dist`) and serves it plus `/xlearn/api/*`, while k8s probes it at `/healthz` and
`/readyz`. Three layers must agree on the `/xlearn` prefix or deep links and asset URLs break:

1. **Traefik** route (`charts/project`): `route.pathPrefix: /xlearn`, with an optional `stripPrefix`
   middleware that removes the prefix before the request reaches the pod.
2. **Vite** `base` — the prefix baked into built asset URLs (`/xlearn/assets/...`).
3. **React Router** `basename` — the prefix the client router matches/strips.

The tension: in production Traefik strips `/xlearn` (the platform convention, matching airlift), so the
pod sees `/`. But the DoD requires the **same binary**, run locally without Traefik, to answer
`/xlearn`, `/xlearn/api/healthz` and `/readyz`. A gateway hard-coded to serve `/xlearn/*` would break
behind the stripping proxy; one hard-coded to serve `/*` would break in local/direct use.

## Decision

**Traefik `stripPrefix: true` (pod sees `/`), Vite `base: /xlearn/`, Router `basename: /xlearn`, and a
base-path-tolerant gateway.**

- **Traefik** route: `pathPrefix: /xlearn`, `stripPrefix: true`, `redirectSlash: true`, `priority: 100`
  (in `infra/apps/xlearn-gateway.yaml`). The `StripPrefix` middleware removes `/xlearn`; the pod
  receives `/`, `/api/healthz`, `/assets/...`.
- **Vite** `base: "/xlearn/"` → built asset URLs are absolute under the prefix. The browser requests
  `/xlearn/assets/x.js`; Traefik strips it to `/assets/x.js` for the pod.
- **Router** `basename: "/xlearn"` → the client matches the browser's `/xlearn/...` URLs; deep links
  round-trip.
- **Gateway** is **base-path tolerant**: it strips a leading `BASE_PATH` (`/xlearn`) from the request
  path if present (a no-op behind the stripping proxy), then routes on the remainder. So the one binary
  serves correctly both behind Traefik (`/api/healthz`) and in local/direct use (`/xlearn/api/healthz`).
  k8s probes (`/healthz`, `/readyz`) are always served at the pod root, prefix-independent.
- **Caching:** hashed assets under `assets/` → `Cache-Control: public, max-age=31536000, immutable`;
  `index.html` and SPA-fallback responses → `no-cache`. Unknown non-asset paths fall back to
  `index.html` (client-side routing); unknown `/api/*` paths return `404` (never the shell).
- **Read-only rootfs:** the gateway serves entirely from the embedded FS and writes nothing to disk,
  satisfying the chart's `readOnlyRootFilesystem: true` default ([0009](0009-deployment-and-gitops.md)).

`BASE_PATH` is a gateway env var (default `/xlearn`), set in the HelmRelease, so the mount prefix is
config, not a constant.

## Consequences

- ✅ One binary works identically in local dev (direct `:8080/xlearn`) and in-cluster (behind Traefik
  stripping) — no build-time or env-time base-path divergence beyond the single `BASE_PATH` value.
- ✅ Matches the platform's `stripPrefix` convention (airlift/landscape) — the pod's router is rooted at
  `/`, so services stay prefix-agnostic internally.
- ✅ Correct cache semantics: immutable hashed assets, always-fresh `index.html`.
- ⚠️ The gateway's prefix-tolerance is a deliberate small redundancy (it strips a prefix Traefik has
  usually already removed). Documented here so it isn't "cleaned up" into a prod-only assumption.
- ⚠️ The chart exposes a single `probes.path`; readiness and liveness both use `/healthz` this sprint
  (readiness is trivially OK with no DB). A distinct `/readyz` DB check lands when a service gains a
  datastore.

## Alternatives considered

| Option | Why not |
|--------|---------|
| **Traefik `stripPrefix: false`** (pod sees `/xlearn/...`) | Simpler gateway, but diverges from the platform's stripping convention and still needs the gateway to special-case root-level k8s probes; the tolerant handler subsumes it at no real cost. |
| **Gateway serves `/xlearn/*` only** (no stripping) | Breaks behind the platform's `stripPrefix: true` route — the pod would never see `/xlearn`. |
| **Serve SPA from disk / a separate static service** | Extra deployable + route, and disk writes fight the read-only rootfs; embedding in the gateway is one image, same origin, no CORS ([0008](0008-frontend-stack.md)). |
| **Relative Vite base + injected `<base href>`** (airlift's model) | Works, but a runtime HTML rewrite is more moving parts than an absolute `base` + `basename`; xLearn's fixed single prefix doesn't need per-request base injection. |

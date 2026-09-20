# Sprint 11 — AI coach (BYO key)

> **Milestone:** M6 — feature-complete v1   ·   **Design phase:** D6 Account + coach
> **Prereqs:** [S10](sprint-10.md), [S05](sprint-05.md)   ·   **Unblocks:** [S12](sprint-12.md)
> **Execute with:** [`../prompts/prompt-s11.md`](../prompts/prompt-s11.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Status |
|---|------|--------|
| 1 | coach service + schema + secrets | ⬜ |
| 2 | BYO-key envelope encryption | ⬜ |
| 3 | provider fan-out + context prompts | ⬜ |
| 4 | coach panel wired across screens | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row + the
> M6 milestone). Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Goal

Stand up the `coach` service end to end and wire the AI coach into every screen using each user's own
provider API key, encrypted at rest. This completes the D6 Account + coach phase begun in [S10](sprint-10.md)
(Settings) and consumes the page context that `practice` exposes ([S05](sprint-05.md)). Landing it makes
v1 **feature-complete (M6)** — all 12 screens are now backed by real services.

## Scope

**In**
- New `coach` service: `cmd/coach` + `internal/coach`; schema `coach` (`api_key_config`, `coach_thread`,
  `coach_message`) via goose migrations + sqlc/pgx; per-service DB role/schema through the infra 4-step pattern.
- BYO-key **envelope encryption** in `internal/platform/secrets`: XChaCha20-Poly1305 per-record data key
  wrapped by a SOPS-managed master key; store `enc_key`/`enc_data_key`/`masked_key`; `PUT`/`GET`/`DELETE
  /coach/key` where only the masked value ever leaves the service.
- Provider-agnostic interface implementing **OpenAI + Anthropic** Messages APIs, called with the decrypted
  user key; server-built page-context prompts (Socratic during an attempt, reviewer post-solve);
  `POST /coach/chat` streaming over **SSE**; `GET /coach/thread?context=` history.
- Wire the persistent `xl-coach`/`xl-fab` bubble -> side panel across **every** screen with the
  current-page context chip; empty state (no key) routes to Settings; make the [S10](sprint-10.md)
  Settings API-keys section fully functional (store/delete).
- Infra: `infra/apps/xlearn-coach.yaml` HelmRelease + Flux image-automation entry; SOPS master-key secret
  `infra/apps/secrets/xlearn-coach.enc.yaml` mounted to the coach pod; gateway routes `/coach/*` and
  proxies the SSE stream (ClusterIP, no Traefik route of its own).

**Out (later sprints)**
- Additional providers beyond OpenAI/Anthropic (kept behind the interface, added later).
- Hardening / perf / e2e / 1.0 cut — [S12](sprint-12.md).

## Tasks

### 1 — coach service + schema + secrets
Scaffold `cmd/coach` (config, slog JSON logger, pgx pool, goose **embedded** migrations run at startup under
an advisory lock, HTTP server with `/healthz` + `/readyz`) and `internal/coach` (handlers, service, sqlc
queries). Add schema `coach` with the three tables from [data-model](../../architecture/data-model.md#schema-coach):
`api_key_config` (`id`, `account_id`, `provider`, `enc_key`, `enc_data_key`, `masked_key`, `default_model`,
`enabled`), `coach_thread` (`id`, `account_id`, `page_context`, `created_at`), `coach_message` (`id`,
`thread_id`, `role`, `content`, `created_at`). The service touches **only** schema `coach`
([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)); it **emits no events** (no `outbox`) and
**consumes none via NATS** (page context is read through the gateway API), per [services.md](../../architecture/services.md).
Add a per-service DB role via the infra 4-step pattern (role, role-owned schema, grants, `search_path`),
`deploy/coach.Dockerfile`, and a CI path filter. Provision infra by copying `infra/apps/airlift.yaml` /
`infra/apps/landscape.yaml` as the HelmRelease template into `infra/apps/xlearn-coach.yaml`: GHCR image
`ghcr.io/sujaykumarsuman/xlearn-coach`, the Flux image-automation `ImagePolicy` (semver `>=0.1.0`) with the
`# {"$imagepolicy": "flux-system:xlearn-coach:tag"}` setter, deployed **ClusterIP / internal** (no route).

### 2 — BYO-key envelope encryption
Implement envelope encryption in `internal/platform/secrets`
([ADR-0007](../../adr/0007-ai-coach-byo-key-and-secrets.md)): generate a per-record 32-byte data key,
encrypt the raw provider key with **XChaCha20-Poly1305** (NaCl `secretbox` / `golang.org/x/crypto`), wrap
the data key with the service **master key**, and persist `enc_key` (ciphertext) + `enc_data_key` (wrapped)
+ `masked_key` (e.g. `sk-...3f2a`). The master key is a 32-byte secret from the SOPS-managed
`infra/apps/secrets/xlearn-coach.enc.yaml`, mounted to the coach pod (same age-encrypted, Flux-decrypted
path) — never in Git plaintext, logs, or the DB. Endpoints: `PUT /coach/key` (store/replace; validate +
mask), `GET /coach/key` (masked value + provider + `default_model` + `enabled` only — **never** the raw
key), `DELETE /coach/key`. Decrypt **in memory only** for the duration of a provider call and zero the
buffer afterward; never log the key or the decrypted buffer; redact in all error paths. A key that fails
provider auth flips `enabled=false` so the client routes the user back to Settings.

### 3 — provider fan-out + context prompts
Define a provider-agnostic `Coach` interface with two v1 implementations — **OpenAI** and **Anthropic**
Messages APIs — each called from the coach service with the decrypted user key (provider + `default_model`
come from `api_key_config`). Build the page-context prompt **server-side** from the fields practice /
curriculum expose (problem, stage, recent outcome, weak area): **Socratic + spoiler-free while the stage is
an active attempt**, and a **code reviewer post-solve**, with behaviour enforced by the system prompt plus
the stage passed from `practice` ([ADR-0007](../../adr/0007-ai-coach-byo-key-and-secrets.md), R-AC3). The
stage gate must never leak the solution during an attempt. `POST /coach/chat` streams the reply over **SSE**
(`text/event-stream`) and persists `coach_thread`/`coach_message`; `GET /coach/thread?context=` returns
history for a page context. Verify SSE passes cleanly through the gateway + Traefik (no response buffering,
adequate idle timeout).

### 4 — coach panel wired across screens
Wire the persistent coach bubble -> side panel using `xl-coach` / `xl-fab` from `theme.css` on **every**
screen (Dashboard, Catalog, Roadmap, Week, Concept, Problem, Revision, Mistakes, Mock, Progress, Settings),
matching `design-system/screens/Problem.dc.html` and `Concept.dc.html`. Render the current-page context chip
(`xl-coach__ctx`) reflecting the page (e.g. `Context — Problem 16 — 3Sum — <stage>` on Problem,
`Context — Concept — Sliding Window` on Concept) and send the same context to `POST /coach/chat` and
`GET /coach/thread?context=`. The empty state (no key) shows the "add a key" prompt and routes to Settings
(`/settings`). Make the [S10](sprint-10.md) Settings API-keys section fully functional against
`PUT`/`GET`/`DELETE /coach/key` (store/delete; masked display; provider + default model; `enabled` toggle).
Add the gateway `/coach/*` routes and proxy the SSE stream to the coach ClusterIP — the gateway never sees
the raw key.

## Acceptance criteria
- [ ] A user stores a provider key: only a **masked** value (e.g. `sk-...3f2a`) is ever returned by
      `GET /coach/key`, and the coach answers using the stored (encrypted) key.
- [ ] The coach is **Socratic / spoiler-free during an active attempt** and a **reviewer post-solve**; the
      context chip reflects the current page.
- [ ] No key -> empty state routes to **Settings**; a key that fails provider auth flips `enabled=false` and
      routes back to Settings.
- [ ] The `xl-coach` / `xl-fab` panel is wired on **every** screen, matching the artboards.
- [ ] `coach` is deployed to prod (ClusterIP) with the SOPS master key mounted; **v1 is feature-complete —
      M6 reached**.

## Definition of Done
CI green · deployed to prod via Flux (no hand `kubectl`) · screens match the artboards · acceptance
criteria met · statuses updated (this file + [`../status.md`](../status.md)) · notable decisions recorded as ADRs.

## Risks / watch-outs
- **NEVER return or log the raw key.** Decrypt in-memory only, zero the buffer after, and redact in every
  error path.
- Master-key handling must match [ADR-0007](../../adr/0007-ai-coach-byo-key-and-secrets.md) (SOPS-managed,
  separate from the DB; losing it means users must re-enter their keys — acceptable, the keys are theirs).
- **Spoiler control:** the stage from `practice` must gate the coach so it cannot leak the solution during
  an active attempt.
- **SSE through the gateway + Traefik:** verify buffering/timeouts (disable response buffering; set an
  adequate idle timeout) or the stream stalls.

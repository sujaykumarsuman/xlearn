# Prompt — Sprint 11 — AI coach (BYO key)

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-11.md`](../sprints/sprint-11.md)   ·   **Milestone:** M6 — feature-complete v1   ·   **Prereqs:** [S10](../sprints/sprint-10.md), [S05](../sprints/sprint-05.md)

## Read first
- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md) — repo conventions: reuse `theme.css`, respect service boundaries, do not commit/push unless asked.
- [`../../adr/0007-ai-coach-byo-key-and-secrets.md`](../../adr/0007-ai-coach-byo-key-and-secrets.md) — **the governing ADR**: envelope encryption, SOPS master key, provider access, handling rules.
- [`../../adr/0006-authn-authz.md`](../../adr/0006-authn-authz.md) — gateway-minted RS256 JWT + JWKS; how coach trusts inbound calls.
- [`../../architecture/services.md`](../../architecture/services.md) — the `coach` section (owns schema `coach`; API; emits no events) and the service<->screen matrix.
- [`../../architecture/data-model.md`](../../architecture/data-model.md) — schema `coach` tables (`api_key_config`, `coach_thread`, `coach_message`).
- [`../../architecture/api.md`](../../architecture/api.md) — the **Coach** endpoints (`/coach/key`, `/coach/chat` SSE, `/coach/thread`) and the SSE/error conventions.
- [`../../prd/xlearn-prd.md`](../../prd/xlearn-prd.md) — **6.6 AI coach** (R-AC1..R-AC4).
- [`../../../design-system/README.md`](../../../design-system/README.md) — coach mechanics + `xl-coach`/`xl-fab` classes; reuse `design-system/theme.css` verbatim.
- `design-system/screens/Problem.dc.html` and `design-system/screens/Concept.dc.html` — coach panel + context chip layout/copy (design reference only; not runnable).
- Sibling infra repo: `infra/apps/airlift.yaml` and `infra/apps/landscape.yaml` (HelmRelease template to copy), `infra/apps/secrets/` (SOPS/age pattern), and the Flux image-automation config. Inspect before adding any deploy config.

## Context
Settings + onboarding are done (S10) and `practice` provides page context (S05). This sprint wires the AI
coach with the user's own provider key (encrypted at rest) across every screen, making v1 feature-complete
(M6). `coach` is a new **internal** service; the gateway proxies its responses (including the SSE stream)
and never sees the raw key.

## Do this (in order)
1. **coach service + schema + secrets** — Scaffold `cmd/coach` (config, slog JSON logger, pgx pool, goose
   **embedded** migrations at startup under an advisory lock, HTTP server with `/healthz` + `/readyz`) and
   `internal/coach` (handlers, service, sqlc queries). Create schema `coach` migrations for `api_key_config`
   (`id`, `account_id`, `provider`, `enc_key`, `enc_data_key`, `masked_key`, `default_model`, `enabled`),
   `coach_thread` (`id`, `account_id`, `page_context`, `created_at`), and `coach_message` (`id`, `thread_id`,
   `role`, `content`, `created_at`); generate the sqlc/pgx query layer. Touch **only** schema `coach` — no
   `outbox` (emits no events), no NATS consumer (reads context via the gateway API). Add `deploy/coach.Dockerfile`,
   a CI path filter, the per-service DB role via the infra 4-step pattern, and `infra/apps/xlearn-coach.yaml`
   by copying `airlift.yaml`/`landscape.yaml`: GHCR image `ghcr.io/sujaykumarsuman/xlearn-coach`, the Flux
   `ImagePolicy` (semver `>=0.1.0`) with the `# {"$imagepolicy": "flux-system:xlearn-coach:tag"}` setter,
   deployed **ClusterIP** (no Traefik route).
2. **BYO-key envelope encryption** — Implement `internal/platform/secrets`: a per-record 32-byte data key
   encrypts the raw provider key with **XChaCha20-Poly1305** (NaCl `secretbox` / `golang.org/x/crypto`); the
   data key is wrapped by the service **master key**. Persist `enc_key` + `enc_data_key` + `masked_key`
   (e.g. `sk-...3f2a`). Load the 32-byte master key from the SOPS-managed
   `infra/apps/secrets/xlearn-coach.enc.yaml` mounted to the pod (env/file) — never in Git plaintext, logs,
   or the DB. Implement `PUT /coach/key` (store/replace), `GET /coach/key` (masked value + provider +
   `default_model` + `enabled` only — never the raw key), `DELETE /coach/key`. Decrypt **in memory only** for
   the provider call and zero the buffer after; never log the key or buffer; redact on error. On a provider
   auth failure, flip `enabled=false`.
3. **provider fan-out + context prompts** — Define a provider-agnostic `Coach` interface with **OpenAI** and
   **Anthropic** Messages implementations, called with the decrypted user key (provider + model from
   `api_key_config`). Build the page-context prompt **server-side** from problem, stage, recent outcome, and
   weak area: **Socratic + spoiler-free while the stage is an active attempt**, **code reviewer post-solve** -
   enforced by the system prompt + the stage passed from `practice`; the gate must not leak the solution
   mid-attempt. `POST /coach/chat` streams over **SSE** (`text/event-stream`) and persists
   `coach_thread`/`coach_message`; `GET /coach/thread?context=` returns history. Verify the stream survives
   the gateway + Traefik (no response buffering, adequate idle timeout).
4. **coach panel wired across screens** — Wire the persistent `xl-coach`/`xl-fab` bubble -> side panel on
   **every** screen (Dashboard, Catalog, Roadmap, Week, Concept, Problem, Revision, Mistakes, Mock, Progress,
   Settings), matching `Problem.dc.html`/`Concept.dc.html`. Render the current-page context chip
   (`xl-coach__ctx`) and send the same context to `/coach/chat` and `/coach/thread?context=`. Empty state
   (no key) shows the "add a key" prompt and routes to `/settings`. Make the S10 Settings API-keys section
   fully functional against `PUT`/`GET`/`DELETE /coach/key` (store/delete; masked display; provider + default
   model; `enabled` toggle). Add the gateway `/coach/*` routes and proxy the SSE stream to the coach ClusterIP.

## Constraints
- Reuse `design-system/theme.css` **verbatim** (no Tailwind); the coach UI uses `xl-coach`/`xl-fab`; keep the
  dark theme; difficulty tokens Easy=`--ds-ok` / Medium=`--ds-warn` / Hard=`--ds-err` wherever difficulty appears.
- **Schema-per-service:** coach touches only schema `coach`; goose **embedded** migrations (startup +
  advisory lock) + sqlc/pgx; per-service DB role via the infra 4-step pattern.
- **Server-authoritative behaviour gate:** the Socratic-vs-reviewer mode is decided server-side from the
  stage passed by `practice` — never trust the client.
- **Secrets:** envelope encryption (XChaCha20-Poly1305 per-record data key wrapped by the SOPS master key);
  decrypt in-memory only and zero after; never log/return the raw key; redact on error. Master key comes only
  from `infra/apps/secrets/xlearn-coach.enc.yaml`.
- coach **emits no events** (no outbox) and reads page context through the gateway API (no NATS consumer).
- **Match infra conventions:** copy `airlift.yaml`/`landscape.yaml` as the HelmRelease; GHCR image
  `ghcr.io/sujaykumarsuman/xlearn-coach`; image-automation setter `# {"$imagepolicy": "flux-system:xlearn-coach:tag"}`;
  `ImagePolicy` semver `>=0.1.0`; read-only rootfs; deploy **ClusterIP** (only the gateway is routed, at `/xlearn`).
- **Pull-based GitOps:** never `kubectl apply` by hand — changes land via Flux from `main` (build-semver
  auto-deploy). Do not commit or push unless asked.

## Deliverables
- `cmd/coach` + `internal/coach` (service, handlers, sqlc queries) with `/healthz` + `/readyz` and slog JSON logs.
- schema `coach` goose migrations (`api_key_config`, `coach_thread`, `coach_message`) + sqlc query layer.
- `internal/platform/secrets` envelope-encryption package (encrypt/decrypt/mask; master-key load; buffer
  zeroing; error redaction).
- Provider-agnostic `Coach` interface + OpenAI and Anthropic implementations.
- Gateway: `/coach/key` (PUT/GET/DELETE), `/coach/chat` (SSE), `/coach/thread?context=` wired and the SSE proxied.
- Web: `xl-coach`/`xl-fab` panel + context chip on every screen; empty state -> Settings; a functional
  Settings API-keys section (S10).
- `deploy/coach.Dockerfile`, a CI path filter, and the infra additions: `infra/apps/xlearn-coach.yaml`
  HelmRelease + image-automation entry + coach DB role/schema + `infra/apps/secrets/xlearn-coach.enc.yaml`
  (SOPS master key).

## Update status
- As each task lands, set its row in [`../sprints/sprint-11.md`](../sprints/sprint-11.md) to ✅ (🔄 while in
  progress); set _Overall_ when all four tasks are ✅.
- Mirror the sprint's state into [`../status.md`](../status.md): the **Sprint board** row, and the
  **Milestones** table (M6). Add a **Decisions log** line for any notable call.
- Record notable technical decisions as ADRs under `docs/adr/` (append-only, MADR-style).

## Done when (acceptance)
- [ ] A user stores a provider key: only a **masked** value (e.g. `sk-...3f2a`) is ever returned, and the
      coach answers using the stored key.
- [ ] The coach is **Socratic / spoiler-free during an active attempt** and a **reviewer post-solve**; the
      context chip reflects the current page.
- [ ] No key -> empty state routes to **Settings**; a key that fails provider auth flips `enabled=false` and
      routes back to Settings.
- [ ] The `xl-coach`/`xl-fab` panel is wired on **every** screen, matching the artboards.
- [ ] `coach` is deployed to prod (ClusterIP) with the SOPS master key mounted; **v1 is feature-complete (M6)**.
- Do not commit or push unless asked.

# ADR-0007 — AI coach BYO-key & secret handling

- **Status:** Accepted
- **Date:** 2026-09-20
- **Deciders:** @sujaykumarsuman
- **Related:** [0003](0003-service-decomposition.md), [0006](0006-authn-authz.md), [0009](0009-deployment-and-gitops.md)

## Context

The AI coach uses **each user's own provider API key** (`ApiKeyConfig`: provider, masked key,
default model, enabled). This is user-supplied *secret* material at rest in our DB — the highest-value
data we hold. We must store it safely, never expose it, and keep the handling consistent with the
platform's SOPS/age model. Note the platform safety rule: xLearn never asks users to type keys
anywhere but their own Settings form, and the key is never echoed back.

## Decision

### Storage: envelope encryption, decrypt in-memory only

- The **coach** service owns `ApiKeyConfig` in its `coach` schema. The provider key is stored
  **encrypted**, never in plaintext.
- **Envelope encryption:** a per-record data key encrypts the API key with **XChaCha20-Poly1305**
  (NaCl `secretbox` / `golang.org/x/crypto`); the data key is wrapped by a **service master key**.
- The **master key** is a 32-byte secret delivered as a **SOPS-managed Kubernetes secret**
  (`apps/secrets/xlearn-coach.enc.yaml`), mounted to the coach pod as env/file — same age-encrypted,
  Flux-decrypted mechanism the platform already uses. It is **never** in Git plaintext, logs, or the DB.
- The key is decrypted **in memory only**, for the duration of a provider call, and zeroed after. It
  is **never** returned to the client — the API exposes only `maskedKey` (e.g. `sk-...3f2a`).

### Provider access

- The coach exposes a provider-agnostic interface; v1 implements **OpenAI** and **Anthropic** Messages
  APIs. Requests are made **from the coach service** using the decrypted user key; the gateway only
  proxies coach responses (it never sees the key).
- Prompts carry **page context** (problem, stage, recent outcome, weak area) built server-side.
  During an active attempt the coach is **Socratic / spoiler-free**; post-solve it is a code reviewer
  (behaviour enforced by system prompt + the stage passed from `practice`).
- Rate/refusal handling and provider errors surface as coach messages; keys that fail auth flip
  `enabled=false` and route the user back to Settings.

### Handling rules (codified in `internal/platform/secrets`)

- Never log the key or the decrypted buffer; redact in error paths.
- Master key rotation: re-wrap data keys with the new master (background re-encrypt); overlap window.
- Backups: the encrypted column is safe in DB backups; the master key is **not** in the DB, so a DB
  leak alone does not expose keys.

## Consequences

- ✅ A database compromise does not reveal user API keys (master key is separate, SOPS-guarded).
- ✅ Consistent with the platform's existing SOPS/age secret flow — no new secret system.
- ✅ Clear blast-radius: only the coach pod can decrypt; the key never crosses to the browser or gateway.
- ⚠️ Master-key rotation requires a re-encrypt routine (documented; low frequency).
- ⚠️ Losing the master key makes stored keys unrecoverable (acceptable — users re-enter; keys are theirs).

## Alternatives considered

| Option | Why not |
|--------|---------|
| **Plaintext key column** | Unacceptable — a DB/backup leak exposes users' provider keys. |
| **Reversible app-wide symmetric key in code/env-plaintext** | Key would end up in Git or unencrypted config; envelope + SOPS master key is stronger and on-platform. |
| **External KMS (Vault / cloud KMS)** | Another stateful system / external dependency + cost for a solo single-node deploy; SOPS master key achieves the goal here. |
| **Store the key in the browser, proxy blind** | Key would live in JS/localStorage (XSS-reachable) and travel on every request; server-side encrypted storage is safer and enables server-built context. |
| **No storage — ask each session** | Poor UX for a daily-use coach; the design shows a persistent configured key. |

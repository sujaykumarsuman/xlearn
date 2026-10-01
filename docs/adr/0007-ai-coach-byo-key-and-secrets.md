# ADR-0007 — AI coach BYO-key & secret handling

- **Status:** Accepted. **Narrowed by [ADR-0031](0031-platform-ai-and-two-tier-keys.md) (v2, Proposed)** to BYO keys only; platform AI is owned by judge. ADR-0031 §7 lists the P0/P1 fixes to this ADR. **Amended by [ADR-0032](0032-realtime-ai-mock-interviewer.md) (v2, Proposed):** realtime sessions use server-side SDP brokering (no credential in the browser), and the key is decrypted only while an interview segment is live.
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

## Update — 2026-10-01: the gateway transits the PUT; AEAD associated data + a KEK keyring

Sprint [m1-10](../v2/sprints/sprint-m1-10.md) implements the P1 fixes [ADR-0031](0031-platform-ai-and-two-tier-keys.md) §7 asks of this ADR. Three
clarifications to the handling rules above.

**The gateway TRANSITS the key; it never stores or logs it — and that is now test-backed.**
This ADR's "the key never crosses to the browser or gateway" is accurate about *storage* but reads as
if the gateway never sees the bytes at all. It does: `PUT /api/coach/key` arrives at the gateway and
is forwarded verbatim to coach, which seals it. The raw key exists in the gateway only as bytes in
flight — never in a log, a response body or an error path. That is the one claim here a careless
change could falsify invisibly (a debug log of a request body; an echoed upstream error), so
`internal/gateway/coach_key_transit_test.go` now sends a sentinel key through the real middleware
chain and asserts it appears in what coach received and in **no** log record at DEBUG (access log
included), no response body, and no error path (coach down → a clean 502).

**Sealed material is bound to its owner (associated data).** A sealed pair alone says nothing about
whose it is, so anyone with database write access could swap two rows' ciphertext and have the coach
answer one learner with another's key. Keys are now also sealed with AEAD associated data
`xlearn/coach/key/v1|<account_id>|<provider>` on **both** envelope layers, so a pair moved between
accounts or providers fails to open instead of decrypting into the wrong context.

Because v1.6.0 opens the sealed material with **nil** associated data, re-sealing the existing
columns in place would have made every key unreadable by the previous release — silently raising
coach's rollback floor while `v1.7.0` declares 1.6.0 ([ADR-0034](0034-v2-release-labelling-gating-and-rollback.md) §3). So M1b stores **two** pairs: the
legacy unbound pair (dual-written, still readable by v1.6.0) and the AD-bound pair, tied to the
legacy pair it was sealed beside by `ad_src_digest = sha256(enc_data_key)`. The digest is what
detects a key replaced by a v1.6.0 image during a rollback, which rewrites only the legacy columns:
without it, chat would answer from the stale AD pair after rolling forward — billing the old key, or
disabling the new one on a 401.

**The binding is therefore not yet ENFORCED.** While the legacy pair exists, a reader still falls
back to it when the AD pair is absent or stale, so the swap above is still reachable by forcing that
path. Enforcement arrives when the legacy pair is contracted in two steps once the floor passes
1.7.0 — [l-01](../v2/sprints/sprint-l-01.md) (`v1.11.0`) stops reading and writing it, [l-02](../v2/sprints/sprint-l-02.md) (`v1.12.0`) drops it. The residual risk is
accepted until then: v2 is owner-only (D35) and the database is not reachable from outside the
cluster (MI-5).

**Master-key rotation is now code, not a procedure.** This ADR's "re-wrap data keys with the new
master (background re-encrypt); overlap window" is realised as a **KEK keyring**
(`COACH_MASTER_KEYS="k1:<b64>,k0:<b64>"`, first entry active) plus a background job
(`internal/coach/rewrap.go`) that moves rows onto the active entry in batches while the retiring
entry can still open what it sealed. Each pair records its `kek_id`, so the overlap window needs no
coordination. Production runs a one-entry keyring (`k0` = today's `COACH_MASTER_KEY`): this sprint
ships the machinery without performing a rotation and changes no SOPS secret. A row the job cannot
decrypt is **skipped and reported, never deleted or disabled** — it is most likely sealed under a key
retired too early, and the key itself may be perfectly valid.

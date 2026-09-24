# ADR-0031 — Platform AI & two-tier keys

- **Status:** Proposed. The WIF spike (≤ ½ day) and judge's egress NetworkPolicy gate M4. **Amended by [ADR-0035](0035-v2-operations-nats-auth-limits-capacity.md) and [ADR-0033](0033-invite-only-admission-and-owner-admin.md) (T7, 2026-09-24):** no push channel in v2 (D34), and the "before learners" items apply at the v3 opening (D35); see §8.
- **Date:** 2026-09-24
- **Deciders:** @sujaykumarsuman
- **Related:** narrows [0007](0007-ai-coach-byo-key-and-secrets.md) to BYO keys; builds on [0029](0029-judge-contract-and-learning-signal.md)
  (the `Scorer` interface, D14/D16/D18), [0027](0027-content-evalpack-and-user-data-model.md) (pack material rules),
  [0030](0030-runner-technology-and-host-hardening.md) (egress, host). v2 topic T5:
  [feasibility § T5](../v2/feasibility.md#t5--platform-ai--two-tier-keys) and the full
  [research appendix](../v2/research/t5-platform-ai.md).

## Context

v2 adds **platform AI**, which the owner pays for. It is used for:
- submission analysis: mistake category, concepts, pointer notes on passing solutions;
- rubric scoring and feedback for the system-design, LLD and behavioral courses.

The v1 **BYO-key coach** (ADR-0007) stays, and T6 interviews also run on BYO keys.

Constraints:
- The eval pack is judge-only (D5).
- There is no service auth for internal endpoints (ADR-0016).
- Invite-only signup (D13).
- A single node with no metrics stack.
- An India-based owner and node (DPDP).
- Neither Anthropic nor OpenAI offers zero data retention (ZDR) without a sales agreement.

## Decision

### 1. Who holds what

- **judge owns platform AI.** `internal/judge/ai` holds the `Scorer`, the spend ledger and all policy. `internal/platform/llm` holds policy-free raw-HTTP adapters (no SDK auto-retries, no `tools` field), the model catalog and a dated price table.
- **coach owns BYO keys only.** Nothing else uses them.
- **No new service.**
- **BYO keys never do platform work in v2.0**: no fallback when the budget runs out, and no learner opt-in. Revisit on open signup, for Feedback and Analyze only, never Score.

### 2. Credential and egress

- **Workload Identity Federation.** A k3s projected service-account token is exchanged for a short-lived Anthropic token with scope `workspace:inference`, bound to the dedicated workspace `xlearn-platform-prod`.
  - A **spike of ≤ ½ day before M4** gates this.
  - **Fallback and break-glass:** a single-workspace service-account key with a 90-day provider-side expiry.
- **SOPS secret `xlearn-judge-llm`** holds the end-user HMAC salt and, only in break-glass, the key.
- **Provider-side controls:**
  - a dedicated org if one is shared;
  - a workspace monthly hard limit plus Console alerts at 50% and 80%;
  - low per-model rate limits;
  - prepaid credits with auto-reload **off**.
- **Workspace pinning:** judge checks the `anthropic-workspace-id` response header.
- **Egress (M4 gate):** judge is default-deny egress except DNS, Postgres, NATS, the runner and TCP 443 to non-cluster addresses. Chart 0.3.0 adds the egress template.

### 3. Models

- **One provider and one model in v2.0: `claude-sonnet-5`**, always with an explicit `effort`. `thinking` display is omitted, and no temperature is sent.
  - The analyzer's effort (`low` or `medium`) and `max_tokens` are set by the **analyzer acceptance run** on a frozen test set.
  - **Opus 5.5 `medium`** is used only as a per-rubric escalation and as the **independent blind re-grade** configuration.
- **Excluded:** Haiku 4.5 (retires as early as 2026-10-15) and Fable 5.1 (a Covered Model: mandatory 30-day retention, 5× the price).
- **Pinned catalog.** Model classes resolve to IDs only through a passed calibration row, and the resolved model is recorded on every call.
- **Structured outputs use a portable schema subset.** Enums replace numeric ranges, and every size cap is enforced in judge's Go validator.

### 4. Retention and privacy (owner: D24)

- **`RetentionPolicy` replaces "zero retention":**
  - Messages API only;
  - no Batch, Files, tools or Covered Models;
  - provider retention ≤ 30 days (flagged content up to 2 years);
  - no training;
  - disclosed to learners;
  - ZDR requested opportunistically.
- **Pack material goes only into `Score`.** Feedback and Analyze never see it (enforced at compile time).
- **Consent:**
  - two unticked opt-ins at invite acceptance, "xLearn AI reviews my graded work" and "…my passing solutions";
  - a separate opt-in for behavioral answers;
  - all three are toggles in Settings, and switching one off falls back to manual entry or v1 self-scoring.
- **Invite terms:** 18+ by attestation.
- **Logs** never contain prompts, completions or learner content. A canary test enforces this.
- **Processing is outside India** (`inference_geo: global`). This is lawful now; DPDP duties apply from ~2027-05-13, but the privacy notice ships before M4 opens to learners.

### 5. Spend (owner: D25)

Four layers:

| Layer | Limit | Notes |
|---|---|---|
| Provider workspace hard limit | **$100/month** | Enforced by Anthropic |
| App global cap | **$80** (80%) | From one append-only `judge.llm_call` ledger, with an admission check at job claim (no holds) |
| Per account | **$6/month and $1/day**, ≤ 20 analyses/day | Re-set from dogfood P90 |
| Advisory work | Pace-based shedding | D16 pass notes are dropped first |

- **Degrade order:** advisory → counted analyzer and Feedback → final Score. **Nothing blocks the loop:** the fallbacks are deterministic pre-fill, D14 manual entry and v1 self-scoring.
- **Circuit breaker:** any spend-cap or billing error from the provider opens a global breaker, shown in-app as a degradation badge. _(T7, D34: no push alert in v2; see §8.)_
- **Learner view:** the allowance is shown as a **percentage**, never in dollars.

**Expected cost:** DSA-only runs **about $1.3–3.2 per active learner-month**. At 20 active learners that is **about $26–63/month**. The one-off analyzer acceptance run (~$10–30) and calibration (~$20–60 per rubric) are billed to the separate `xlearn-calib` workspace.

### 6. Scope of AI review of passing solutions (owner: D26, refines D16)

- **Reviewed:** **course passes**, plus a **revision-touch pass whose solution is materially different** from the last reviewed solution for that problem. "Materially different" means the normalized-code fingerprint differs by more than a threshold tuned in the acceptance run.
- **Outcome when the solution is correct but improvable:** the revision item is marked **"correct, with improvements"** and gets pointer notes, which are learner-private (C3), per (account, item), and erased with the account.
- **No effect** on pass/fail, grade or ladder.
- **Tier:** advisory, so it is shed first.

### 7. Coach (BYO) changes

- **Per-course persona** from the manifest.
- **Mode gate:**
  - `locked` during live touches and mocks;
  - `attempt` withholds the pattern, concepts and solution facts;
  - `review` after conclusion.
- **Coach-assist cap (owner: D27):**
  - **Per problem.** A coach chat about the **same problem** during its open counted attempt needs an explicit confirmation, and the gateway records it on the attempt (fails closed).
  - The attempt is then **capped at Assisted**.
  - Asking from another page is an accepted, honor-based bypass.
- **ADR-0007 fixes:**
  - P0: quota and billing errors never disable a key; `max_tokens` and effort are set so replies aren't truncated; the onboarding key step actually saves the key.
  - P1: OpenAI `store:false`; a server-side model catalog (no user base URLs); AEAD associated data bound to account and provider; a master-key keyring with `kek_id`; per-feature default keys (`coach`, `interview`); a derived-credentials clause for T6 realtime.
  - Doc: the gateway transits the PUT body in memory.
- **Usage and limits:** usage is shown on the learner's own key, with limits of 20 messages/min, 2 concurrent streams and 300/day.
- **UI names:** **"xLearn AI (included)"** and **"Your AI coach (your key)"**.

### 8. Amended by T7 (2026-09-24)

- **No push channel (D34, [ADR-0035](0035-v2-operations-nats-auth-limits-capacity.md)).** v2 has no alerting. Spend protection is the provider console's own hard limit and 50%/80% alerts (external to the cluster) plus judge's in-app caps, breaker and degradation badges. The owner reviews spend on demand. The `retire_not_before` check becomes a manual check before each model change.
- **"Before learners" means the v3 opening (D35, [ADR-0033](0033-invite-only-admission-and-owner-admin.md)).** v2 is used by the owner and testers only. The privacy notice, 18+ attestation and consents still ship with the learner gate (L) in v2. The two-week dogfood re-measurement and a `SEAT_CAP` re-sized from M4 data gate the opening.
- **Limits in owner-only v2 (D25, D35).** D25's dogfood defaults (provider $15, app cap $12) apply while only the owner and testers use platform AI, which is all of v2. The §5 values ($100 / $80) apply from the v3 opening, re-sized with `SEAT_CAP` ([rollout plan](../v2/rollout-plan.md) MI-14, §11).
- **The WIF spike** is scratch-only, so it can run any time before M4 (e.g. in the spike week); only the production JWKS paste needs the cluster.

## Consequences

- ✅ **Grading and analysis work for every learner without a key.** Spend is bounded four ways, and the loop never blocks.
- ✅ **No standing platform secret in the cluster** under WIF. The inference-only scope blocks Files, Batch and agents.
- ✅ **One provider** keeps calibration, disclosures and secrets small. D14 manual entry covers outages.
- ⚠️ **The Hostinger weekly images contain the k3s service-account signing key**, so WIF does not remove that exposure. The Hostinger login stays top-tier sensitive.
- ⚠️ **A judge compromise can leak the pack** through judge's own output. This is accepted: pack secrecy protects signal integrity only.
- ⚠️ **Analyzer cost depends on the effort the acceptance run needs** (`low` vs `medium` is ×2.2). Caps are sized after the run, within the $100 ceiling.
- ⚠️ **The coach-assist cap is honor-based beyond same-problem in-app use** (D27).
- ⚠️ **Model churn** (new GPT-6 models; Haiku retiring) is handled by the calibration gate and alerts at `retire_not_before`.

## Alternatives considered

| Option | Why not |
|---|---|
| The coach as the single LLM egress | Pack material would leave judge (breaks D5); every secret would sit in one pod |
| A new `ai` gateway service, or a credential-injecting egress proxy now | A ninth service; needs service auth; deferred to triggers (open signup, runner off-node, a second consumer) |
| A static SOPS key as the primary credential | WIF removes the standing secret and adds a scope; a static key is the fallback only |
| BYO fallback, or BYO opt-in for platform tasks | ADR-0007 decrypt boundary, the pack rule, and uncalibrated models |
| Two providers active/active | Doubles calibration and disclosures; D14 covers outages |
| Batch or Flex API | Batch isn't ZDR-eligible (29-day retention); saving is ~$3–5/month |
| Reviewing every touch pass, or the first pass per item only | The owner chose course passes plus materially different touch passes |
| Account-wide coach-assist cap | The owner chose per problem |

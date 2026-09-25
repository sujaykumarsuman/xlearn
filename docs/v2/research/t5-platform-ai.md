> **T5 research appendix.** Method: four parallel research slices (key architecture and secrets; cost and budgets; quality, safety and privacy; coach and BYO integration) were synthesized into one draft. The draft faced two adversarial critiques: cost, abuse and ops (6/10) and security, privacy and grading quality (7/10). This is the revised final. Prices, retention terms and features were verified on the web on 2026-09-24. Decided in [ADR-0031](../../adr/0031-platform-ai-and-two-tier-keys.md) (Proposed).
>
> **Status:** settled with the owner 2026-09-24 (**D24–D27**, §12). §12 overrides the body where they conflict, notably:
> - D16 scope = course passes plus *materially different* touch passes marked "correct, with improvements";
> - D18 = per problem, with no mock or realtime lock during attempts.
>
> **WIF spike (spk-03, 2026-09-25): WIF GO** with `check_jti=false` on the one-rule issuer ([§15](#15-wif-spike-result-spk-03-2026-09-25)). §15 overrides §3 where they conflict: notably the rule scope is `workspace:developer`, not `workspace:inference`.

# T5: Platform AI and two-tier keys

This final version incorporates the draft and both critiques. Claims were re-verified on the web on **2026-09-24**; the sources are at the end. `(inferred)` marks my own reasoning. Nothing in the repo or the cluster was changed. The cost re-run is at `scratchpad/t5final/cost.py`.

## 1. Recommendation in one paragraph

**judge owns platform AI and coach owns BYO keys.** One owner-paid Anthropic credential is used only for counted course and touch evaluation and analysis, and only judge uses it. `internal/judge/ai` holds the `Scorer`, the spend ledger and all policy. `internal/platform/llm` holds the shared, policy-free raw-HTTP adapters, the model catalog and the price table. The credential is **Workload Identity Federation (WIF)**: a k3s projected service-account token is exchanged for a short-lived Anthropic token with scope `workspace:inference`, bound to the dedicated workspace `xlearn-platform-prod`. A spike of half a day or less before M4 gates it. The fallback and break-glass option is a single-workspace service-account key with a provider-side expiry. **judge gets default-deny egress as a hard M4 gate**, and its request builder cannot send `tools`. v2.0 runs **one provider and one model, `claude-sonnet-5`**, always with an explicit effort. The analyzer's effort (`low` or `medium`) and its `max_tokens` are picked by the analyzer acceptance run, and **the caps are sized from that run, not before it**. DSA-only spend is about **$1.3–1.9 per active learner-month at `low`, or $2.8–3.2 at `medium`** — about $26–63 a month at 20 learners. Four layers bound spend:
- the provider workspace hard limit, which is the owner's ceiling (Q2);
- an app cap at 80% of that limit, enforced from one append-only `llm_call` ledger;
- per-account day and month caps;
- **pace-based shedding** of advisory work (D16 pass notes).

None of them ever blocks the loop: every AI failure ends in deterministic pre-fill, D14 manual entry or v1 self-scoring. **There is no BYO fallback and no BYO opt-in for platform tasks in v2.0.** Learners can switch xLearn AI off. "Zero retention" becomes a **`RetentionPolicy`**: Messages API only; no Covered Models, Batch, Files or tools; provider retention of 30 days or less, disclosed; ZDR if it is ever granted. The owner signs it off before any pack-bearing `Score` (Q1). M4's analyzer carries no pack material. A new **ADR-0031** records all of this. ADR-0007 is narrowed to BYO keys and gains:
- the P0 fix so quota errors no longer disable keys;
- `store:false`;
- a model catalog;
- AEAD binding;
- a master-key keyring;
- a derived-credentials clause for T6.

## 2. Two tiers: when each key is used

| Task | Key | Model tier | Why | Degrade when unavailable |
|---|---|---|---|---|
| **Analyze**, below-clean course conclusion (DSA, M4) | platform | `claude-sonnet-5`, effort set by the acceptance run (§4) | Counted signal: mistake pre-fill and concepts (T4 §6.3). No pack material. Every learner gets it without a key. | `analysis_unavailable`. Deterministic pre-fill still runs, and the learner picks the category. |
| **Analyze**, failed code touch | platform | same | same | same |
| **Analyze on pass (D16)**, scope: **the first passing solution per (account, item)**. That is the course pass, or the first passing touch after a Miss. Later touch passes are **not** analyzed; existing notes still show. (Q3) | platform | same, **advisory tier** | Pointer notes and an optional revisit; no effect on grade or ladder | **Shed first**, when spend runs ahead of pace (§6); no notes are attached |
| **`ai_rubric` Score** (course final; touches at L4–5), including k escalation | **platform only** | Sonnet 5 at its calibrated effort. Opus 5.5 only for a rubric Sonnet 5 fails. | Pack material goes only here (T1 §10). Must pass the gate for that exact configuration. | `inconclusive(budget_exhausted \| ai_unavailable)`. The learner chooses **"Wait for xLearn AI (resets <date>)"** or **"Grade it myself"** (D14). A touch falls back to v1 self-scoring. |
| **Blind re-grade** (the one D14 dispute) | platform | An **independent calibrated configuration**: a second `prompt@v` with reordered anchors and different exemplars, or Opus 5.5 at `medium`; k=5 | Errors not correlated with the original grade (§7) | Budget exhausted → manual override within the deterministic ceilings |
| **Feedback** (B finals, including give-up) | platform | Sonnet 5 · `low` | No pack material; gets Score's verified learner quotes and bands. It is the only learner-visible "reasoning" (D14). | Show bands, verified quotes and public descriptors (`feedback_unavailable`) |
| Behavioral (C4), any platform call | platform, **only after a separate recorded opt-in** | same | T1 consent rule | No opt-in → `self@1`. Never sent to Analyze. |
| Learner turned **"xLearn AI reviews my work" off** (§8) | none | — | DPDP consent must be free and easy to withdraw | D14 manual entry and v1 self-scoring; D16 skipped |
| Arena submit, Run | none | — | T0, T4 | `not_evaluated_here` |
| Coach chat (modes in §9) | **BYO** | The learner's choice from the catalog | Learner pays (ADR-0007) | No key: "connect" prompt. Quota error: top-up message, and **the key stays enabled**. |
| "Ask your AI coach about this result" | BYO | Learner's choice | Review mode; learner-visible DTO only | Hidden when no key is connected |
| T6 interviews and the mock AI-score proposal | BYO (`interview` default key) | Realtime-capable catalog model | Settled (T4 §11.2) | No key → self-scored mock. **Refused while a counted course attempt is open** (§9). |
| Calibration and analyzer acceptance runs | Owner's **`xlearn-calib`** workspace: WIF from private CI, or a short-expiry personal key; **never in the cluster** | Model under test | Keeps this spend out of learner budgets | — |
| Content-authoring drafts (T1) | Owner's separate key, offline | any | Never in the cluster | — |
| Platform budget exhausted | **No BYO fallback** | — | See below | See the Score and Analyze rows |

**Why BYO never does platform work in v2.0:**

| Reason | Detail |
|---|---|
| Decrypt boundary | Only coach can decrypt BYO keys (ADR-0007). judge would need a money-spending coach endpoint plus service auth that doesn't exist yet (ADR-0016). T0 forbids unauthenticated spending endpoints. |
| Pack rule | Pack material may not go into BYO calls (T1 §10). |
| Trust | The learner's model is uncalibrated, so its output is `trust=honor` — the same trust as manual entry, which is free. |
| Saving | About $1.3–3.2 per learner-month. |

**Revisit trigger:** open signup (D13 lifted, also a D21 trigger), or the app cap reached in 2 of 3 months. Even then, allow BYO for **Feedback and Analyze only, never Score**. It would go through a coach `POST /internal/complete` with service auth, labelled `ai-byo` and `trust=honor`.

## 3. Where the platform key lives, and secrets

**The choice.** judge owns platform AI: `Scorer`, ledger, policy, and credential use. No new service is added.

| Alternative | Why not |
|---|---|
| Coach as the single egress | Pack material would pass through coach, which breaks D5. The BYO master key, every BYO key and the platform key would sit in one 128 Mi pod, so a key-selection bug could spend the owner's money on chat or the learner's money on grading (inferred). |
| New `ai` service | A ninth service on one node for one consumer, and it needs service auth. |
| Credential-injecting egress proxy now | Deferred with triggers (§11). The main reason: a compromised judge can leak the pack through its own learner-visible fields (feedback, notes), so the proxy adds little protection for the pack. WIF already makes a stolen credential short-lived. |

**Credential: Workload Identity Federation, gated by a spike.** Verified: Anthropic supports self-managed Kubernetes, including k3s, with **`inline` JWKS** for issuers that can't be reached publicly, and it offers an **`workspace:inference`** scope (Messages, Models, OpenAI-compatible chat only; anything else returns 403).

| Item | Setting |
|---|---|
| Anthropic resources | Service account `xlearn-judge` (member of `xlearn-platform-prod` only). Issuer = the cluster's `iss`, `jwks.type: inline` (keys from `kubectl get --raw /openid/v1/jwks`). Rule: `subject_prefix: system:serviceaccount:xlearn:xlearn-judge` (no `*`), `audience: https://api.anthropic.com`, `oauth_scope: workspace:inference`, `token_lifetime_seconds: 3600`, a single workspace. |
| Pod | A projected `serviceAccountToken` (audience as above, `expirationSeconds: 3600`) at `/var/run/secrets/anthropic.com/token`. `automountServiceAccountToken` stays off. The kube-apiserver rejects a token with that audience (inferred). |
| Code | Raw-HTTP `POST /v1/oauth/token` in `platform/llm/auth`. Re-exchange only when the projected file's `iat` has changed, so every exchange presents a fresh `jti`. **Never set `ANTHROPIC_API_KEY`**; it would shadow federation in any SDK. |
| **Spike (≤ ½ day, before M4)** | 1. Do k3s tokens carry `jti`, and how often are they rotated? 2. An in-place container restart re-presents a used `jti` (`jti_reused`). If it does, set `check_jti=false` on this one-rule issuer; that is acceptable because the tokens are audience-bound and last ≤ 1 h (inferred). 3. Measure end-to-end refresh. If the spike fails, use the fallback. |
| Maintenance | Inline JWKS has no auto-refresh. **When the k3s service-account signing key rotates (rare; check at k3s upgrades), paste the new JWKS.** Until then, exchanges fail with 401: the breaker opens, an alert fires, and grading goes to manual. |
| **Fallback and break-glass** | A **single-workspace service-account key** with a **provider-side expiry of 90 days**, so a forgotten rotation fails safe (401 → manual). It is created only when needed. |
| Calibration | Private CI (the `xlearn-evalpack` repo) uses GitHub Actions OIDC WIF into `xlearn-calib`. Owner-machine runs use a personal key with a 7–30-day expiry. |

**Blast radius (corrected).**
- A compromised judge exposes the pack, in-flight submissions, and the ability to spend through the credential until it is found. Spend is bounded by the provider limit.
- **WIF does not remove the Hostinger-image exposure.** The images contain the k3s SA signing key (T2), and with it an attacker can mint judge tokens.
- What WIF does remove: the standing secret string in etcd, SOPS and env dumps; any use of Files, Batch or Managed Agents with our credential; and the rotation chore.
- Pack secrecy protects signal integrity only (T1 §10). Treat a judge compromise as "pack burned": rotate items per T1.

**Spending paths are authenticated or internal to judge:**
- **Submissions and closes:** gateway-minted JWT with audience `judge` (T4 §2.2).
- **Re-grade:** amend T4 §2.4 step 8. The gateway asks practice to authorize the dispute, then calls judge `POST /evaluations/{id}/regrade` with the learner's judge-audience JWT. judge checks ownership and `UNIQUE(regrade_of)`.
- **The analyzer** is triggered only by judge's own NATS durables (`problem_solved`, `touch_concluded`). NATS auth before M3 closes the forged-event spend path.
- **Invariant tests:**
  - no `/internal/*` route enqueues llm-lane work;
  - `Reserve` refuses any context other than `course` or `touch`;
  - a CI lint allows credential use only from `internal/judge/ai`;
  - the coach binary cannot load `LLM_*` configuration;
  - **the request builder has no `tools`, `mcp_servers` or `container` fields** (a golden request-shape test).

**SOPS layout.** File `infra/apps/secrets/xlearn-judge-llm.enc.yaml`, covered by the existing `apps/secrets/.*\.enc\.ya?ml$` rule and loaded with `envFrom` (convention in `apps/xlearn-coach.yaml:54-62`).
```yaml
stringData:
  LLM_ENDUSER_SALT: …            # 32 B; HMAC(account_id) → metadata.user_id / safety_identifier
  # LLM_ANTHROPIC_API_KEY: …     # break-glass only; absent under WIF
  # LLM_OPENAI_API_KEY: …        # only once an OpenAI challenger passes the gate
```
- **Non-secret HelmRelease values:**
  - `LLM_PLATFORM_ENABLED`;
  - `LLM_ANTHROPIC_ORG_ID`, `…_WORKSPACE_ID`, `…_SERVICE_ACCOUNT_ID`, `…_FEDERATION_RULE_ID`;
  - `LLM_KEY_LABEL` (written to each ledger row);
  - model routes (§4) and caps (§6).
- Kept separate from `xlearn-judge-db`, the runner bearer and the evalpack pull secret.
- Rotating the salt or the break-glass key bumps `podAnnotations: {xlearn.dev/llm-rev: "N"}` (precedent: `xlearn.dev/oauth-rev`, `apps/xlearn-identity.yaml:36-38`).

**Provider-side controls (mandatory; runbook in T7).**

| Control | Setting |
|---|---|
| Org | A **dedicated xLearn org** if the owner's Claude Code or other API use shares the current one. Verified: the Claude Code workspace auto-creates in the same org, and the tier cap (Start $500), credits and abuse enforcement are org-wide. A new org may start in the Evaluation tier. |
| Workspace | `xlearn-platform-prod`. Verified: the Default workspace can't carry limits. |
| **Spend limit plus alerts** | The workspace monthly limit is the owner's **ceiling (Q2)**. Also configure the Console's spend alerts at 50% and 80% (verified: the Spend limits tab offers them). Together they form a zero-code alarm that is independent of the app. |
| Rate limits | Low workspace RPM and OTPM on Sonnet 5 and Opus 5.5 to bound runaway loops (verified per workspace and per model). |
| Billing | Prepaid credits with **auto-reload OFF** and a balance of about 1–2 months of the ceiling. Verified: auto-reload has no monthly cap, so "capped" isn't an option, and credits expire after 1 year. |
| Workspace check | judge refuses the platform path unless the response header `anthropic-workspace-id` equals `LLM_ANTHROPIC_WORKSPACE_ID`. It checks on the first call and then on each response. A mismatch opens the breaker and alerts. |
| No Admin keys in the cluster | Reconcile against the Console monthly. |
| OpenAI (only when a challenger passes) | Dedicated project with a hard project limit and spend alerts; a restricted service key, inference only; model allowlist (from the slice, not re-verified). |

**`RetentionPolicy` (replaces `ZeroRetention bool`).** Every platform call:
- **Anthropic:** Messages API only. No Batch (29-day retention), Files, code execution, server tools or Covered Models (verified).
- **OpenAI:** explicit `store:false`. No `previous_response_id`, background mode, Files or Batch.
- `workspace:inference` enforces most of this at the credential; the request builder enforces the rest.
- `retention_class ∈ {provider_std, zdr}` is written to the ledger.
- ZDR is requested opportunistically (verified: per org, through sales).
- `Score` refuses to run unless `zdr=true` **or** the owner has attested `LLM_ACCEPT_STD_RETENTION=true` (Q1).

**Egress: a hard M4 gate (fixes the gap).** Verified: `infra/charts/project/templates/networkpolicy.yaml` (chart **0.2.2**) is ingress-only ("Egress is untouched"), and T3 Track B gates nothing.
- Chart 0.3.0 adds an egress template.
- judge gets **default deny**, then allows only: kube-dns :53, PG, NATS, the runner service, and **TCP 443 to `0.0.0.0/0` except** the pod, service and node CIDRs and `169.254.0.0/16`.
- This moves judge's egress out of Track B and into the **M4 gate**.
- The runner keeps no egress and no LLM path (T3).
- Residual risks: DNS tunnelling through CoreDNS forwarding, and exfiltration through judge's own output channel (inferred). Both are accepted for invite-only.

**Kill switches, fastest first:**

| # | Action | Takes effect |
|---|---|---|
| 1 | **Console:** disable the WIF rule or key, or drop the workspace limit to the minimum → 401 or 400 → breaker → manual | Seconds; no git |
| 2 | `judge admin breaker open` (`kubectl exec`) | Seconds |
| 3 | `LLM_PLATFORM_ENABLED=false` through git and Flux | Minutes |
| 4 | Per-account `ai_disabled` (owner) | Immediate |

**Triggers to split out a credential-injecting `xlearn-llm-egress`** (the judge image with a subcommand, around 32 Mi, the only pod with :443, which forwards only `POST /v1/messages` and the token exchange, overwrites auth and strips tools):
- open signup;
- the runner moves off-node (D21);
- a second in-cluster consumer of the platform credential;
- judge running on more than one node.

## 4. Provider and model choice per task

| Task | Primary (model · effort · `max_tokens`) | Fallback | Gate |
|---|---|---|---|
| **Analyze** (M4) | `claude-sonnet-5`. Effort is swept on the **dev** split over {`low` with `thinking:{type:"disabled"}`, `low`, `medium`} (inferred candidates). `max_tokens` = p99(thinking + visible) × 1.5 from the run; **provisional 4,000**. | No automatic fallback → `analysis_unavailable`. Pre-approved lever: `gpt-6-luna` once it passes. | **Analyzer acceptance** on a frozen **test** split of ≥ 40 (15 optimal, where "no pointer" is correct; 15 passing but suboptimal; 10 failing with a known category), plus a dev split of ≥ 30. Thresholds: category accuracy ≥ 80%; false-pointer rate ≤ 10%; line ranges 100% valid; schema-valid after ≤ 1 retry 100%; **truncation ≤ 2%**; **p95 $/analysis recorded**; bootstrap lower bound on accuracy ≥ 70%. |
| **Score** (first rubric course) | Sonnet 5 · calibrated effort · 4,000 | Opus 5.5 · `medium`, per rubric. Challenger `gpt-6-sol`. Nothing uncalibrated ever runs. | §7 gate per (rubric@v, prompt@v, schema@v, model, effort, api_version) |
| **Re-grade** | An independent passed configuration (§7) | Manual override | Same gate |
| **Feedback** | Sonnet 5 · `low` · 3,000 | → `feedback_unavailable` | Golden set ≥ 20 per archetype; quote verification ≥ 90%; consistency rules (§7) |
| Recall probes | none (`key` graders plus alias tables) | — | — |

**Why Anthropic, and one provider (verified 2026-09-24):**
- **Price:** Sonnet 5 at $2/$10 is now standard; the scheduled move to $3/$15 was cancelled.
- **Lifecycle:** Sonnet 5 retires not sooner than 2027-06-30 and Opus 5.5 not sooner than 2027-09-22, with at least 60 days' notice.
- **Controls:** workspace spend and rate limits, WIF with an inference-only scope, and no stateful defaults.
- **OpenAI comparison:** `gpt-6-sol` has the same list price and is about 15–20% cheaper per task after the Claude tokenizer, but it launched 2026-09-22 and Responses are stored by default.
- **A second provider** doubles calibration, processor disclosures and secrets, for availability that D14's manual path already covers.
- **Revisit when:** `ai_unavailable` exceeds 5% of AI evaluations over 30 days, or Anthropic's terms or price change.

**Structured outputs: a portable subset.** Verified: Anthropic's `output_config.format` is GA on Sonnet 5 and Opus 5.5. It doesn't support `minimum`/`maximum`, `minLength`/`maxLength`, `pattern` or `maxItems`, and `minItems` accepts only 0 or 1. OpenAI strict mode requires every field in `required` plus `additionalProperties:false`. So the schemas use:
- types, `enum`, `const`, `$defs`/`$ref`;
- `anyOf:[T,null]` for optional values;
- every field required;
- `additionalProperties:false`.

All caps are enforced in judge's Go validator (§7).

**Request parameters (every platform call):**

| Parameter | Setting | Why |
|---|---|---|
| `output_config.effort` | Always explicit | Sonnet 5 defaults to `high` and Opus 5.5 to `medium` (verified). Effort is part of `prompt@v`. Changing it invalidates the cache. |
| `thinking` | `{type:"adaptive", display:"omitted"}` sent explicitly. It is already the default display on Sonnet 5 and Opus 5.5 (verified), so thinking text never reaches judge. Discard stays as a backstop. | Thinking is billed as output and counts toward `max_tokens` (verified) |
| `temperature`, `top_p`, `top_k` | **Never sent** | Corrected: they return 400 only when **set to a non-default value** on 4.7+ (verified). k-sampling relies on default variance. |
| `inference_geo` | `global` | `us` costs ×1.1 with no DPDP benefit (verified) |
| `metadata.user_id` | HMAC pseudonym | Lets Anthropic "pinpoint violations"; enforcement stays org-level (§6) |
| `tools`, `mcp_servers`, `container` | Never present | Retention and exfiltration (§3) |

**Pinning and deprecation.**
- A compiled-in catalog holds `{provider, id, class, caps(realtime, effort, structured), price@v, retention: standard|covered, retire_not_before}`.
- Classes resolve to pinned IDs **only through a passed calibration or acceptance row**. An unknown combination goes to manual.
- The response `model` is recorded on every call. A mismatch with the calibrated ID stops `Score` or `Analyze` (→ manual) and alerts.
- An alert fires 90 days before `retire_not_before`. The successor must pass at least 30 days before retirement.
- **Excluded:**
  - Haiku 4.5: retires not sooner than 2026-10-15; no `effort`; no `inference_geo`.
  - Fable 5.1: a Covered Model with mandatory 30-day retention and no ZDR; 5× the price.

## 5. Cost model

**Prices per MTok (verified 2026-09-24; corrected cache columns).**

| Model | Input | 5 m cache write | Cache read | Output | Notes |
|---|---|---|---|---|---|
| `claude-sonnet-5` | $2 | $2.50 | $0.20 | $10 | Batch −50% |
| `claude-opus-5-5` | $4 | $5 | $0.20 (0.05×) | $20 | Batch −50%; thinking always on |
| `claude-fable-5-1` | $10 | $12.50 | **$0.25** | $50 | Excluded |
| `claude-haiku-4-5-20251001` | $1 | $1.25 | **$0.10** | $5 | Excluded |
| `gpt-6-sol` | $2 | — | $0.20 | $10 | Batch/Flex −50% |
| `gpt-6-luna` | $0.10 | — | **$0.01** | $0.50 | Batch/Flex −50% |
| `gpt-6-astra` | $10 | — | $1 | $50 | BYO catalog only |
| `gpt-5.6-sol` | $4 | — | $0.40 | $20 | Promo "through November 21, 2026" |
| `gpt-5.6-terra` / `-luna` | $2 / $0.20 | — | $0.20 / $0.02 | $12 / $1.20 | |

- Claude 4.7+ tokenizer: "approximately 30% more tokens", so the model applies ×1.3.
- Thinking is billed as output. `inference_geo:"us"` costs ×1.1. OpenAI regional processing costs +10%.

**Per-task unit cost** (Sonnet 5; ×1.03 retry allowance).

| Task | Input | Visible out | Thinking | Calls | $ per unit |
|---|---|---|---|---|---|
| Analyze, `low` | 4–5k | 0.5–0.8k | 0.3–0.6k | 1 | **$0.021–0.030** |
| Analyze, `medium` | 5k | 0.8k | 2.0–2.5k | 1 | **$0.045–0.050** |
| Analyze, worst case (16 KiB input cap, `max_tokens` 4,000, **no truncation retry**) | ≈ 8k | — | — | 1 | $0.061 |
| SD final: Score (k=3.4, cached) + Feedback | 8.5–9k / 4–8k | 0.6–0.8k / 1–1.2k | 0.5–1.5k / 0.3k | 4.4 | $0.11–0.17 |
| Blind re-grade (**k=5**) + ½ Feedback | | | | 5.5 | **$0.12–0.19** (corrected) |
| SD worst-case admission (k=3 + Feedback at `max_tokens`) / escalation | | | | | **$0.25** / $0.13 (corrected) |

**Activity model** (T1 §6.2; grade mix inferred):
- 250 active days a year; 2 concluded attempts and 4 touches per learner-day.
- Clean 35%, rough 25%, assisted 15%, miss 25%; 20% of touches fail on code; 70% of Misses pass at Day 1.
- Analyses per learner-day: **3.05** (course conclusions + failed touches + first pass per item).
- If every touch pass were analyzed, it would be **5.9 (×1.9)**.

| Scenario ($ per active learner-month) | $/lrn-mo | Owner only | 20 | 100 | 1000* |
|---|---|---|---|---|---|
| **DSA, Sonnet 5 analyzer at `low`** | **1.30–1.92** | $1–2 | **$26–38** | $130–192 | $1.3–1.9k |
| **DSA, Sonnet 5 analyzer at `medium`** | **2.84–3.17** | $3 | **$57–63** | $284–317 | $2.8–3.2k |
| D16 on every touch pass (Q3b), at `low` / `medium` | 2.52–3.72 / 5.49–6.13 | — | $50–74 / $110–123 | — | — |
| + SD/LLD at 10% of attempts (after the SD course and canvas) | adds ≈ 0.7–1.1 | | +$14–22 | +$70–110 | |
| SD at 30% (stress) | adds ≈ 2.1–3.3 | | +$42–66 | | |
| Lever: `gpt-6-luna` analyzer (gated) | 0.05–0.14 | ~$0 | $1–3 | $5–14 | $50–140 |
| `gpt-6-sol` analyzer, `low` / `medium` | 1.05–1.57 / 2.49–2.81 | | $21–31 / $50–56 | | |

\* N = 1000 requires open signup (a D21 trigger), so it is outside v2.0.

- **One-off costs (corrected; billed to `xlearn-calib`):**
  - Analyzer acceptance: about **$10–30**, covering the effort sweep, prompt iteration and dev plus test runs.
  - Calibration: about **$20–60 per rubric**, from 5–10 prompt passes, the effort sweep, twins, and the independent re-grade configuration.
  - Budget `xlearn-calib` at about **$50/month during M4 bring-up**, with its own limit and alerts.
- **Caching:** the analyzer is one call, with a shared system prefix of 512+ tokens (verified: that is the minimum cacheable length on Sonnet 5 and Opus 5.5). **Sample-1-first k-sampling is deferred to the SD-course milestone** and saves about $0.04 per SD final there. No Batch or Flex in v2.0.
- **Sensitivity:**
  - analyzer `low` → `medium`: ×2.2;
  - D16 touch passes: ×1.9;
  - effort left at the default (`high`): about +34% or more;
  - Opus 5.5 for Score: +30% (SD at 10%).
- **Tax:** about +18% GST on Indian invoices (third-party source), plus card FX. Budgets are in pre-tax USD.
- **Re-measure** the grade mix, touch-fail rate, reasoning tokens and truncation from the first two weeks of M4 dogfood, then set the invite-phase caps.

## 6. Budgets, metering and abuse controls

**Cap layers (all monthly, UTC).**

| Layer | Value | Enforced by |
|---|---|---|
| Provider workspace hard limit | **The owner's ceiling (Q2)** | Anthropic (a 400 "specified workspace API usage limits") |
| Prepaid credit, auto-reload off | About 1–2 × the ceiling | Anthropic (credit exhausted) |
| App global cap | **80% of the provider limit** (OpenAI cap enforcement "is not instantaneous") | judge ledger |
| App global daily guard | 15% of the app cap | judge ledger |
| Per account | **$6/month, $1/day** at launch. Re-set to 2 × the measured P90 learner-month after dogfood. | judge ledger |
| Per account (counts) | ≤ 20 analyses/day; ≤ 6 AI-scored finals/day | judge ledger |
| Owner account | Separate limit (for example 2 × a learner's) | `judge.llm_account_limit` override |

**Sizing rule (goes into ADR-0031; numbers go into HelmRelease values after the acceptance run):**
- app cap ≥ 1.2 × the P90 projection for N active learners at the measured unit cost;
- provider limit = app cap ÷ 0.8;
- if the result exceeds the Q2 ceiling, pull a lever in order: analyzer `low`, then Luna (gated), then cut D16 scope.

**Dogfood defaults (M4, owner only):** provider $15, app $12.

**Ledger: one append-only table** (replaces the draft's `llm_budget` and `llm_reservation`; money in µUSD `bigint`, numbers and enums only).

```sql
judge.llm_call(id uuidv7, account_id NULL, evaluation_id NULL, job_id, path_slug, purpose, tier /*final|counted|advisory*/,
  provider, model_requested, model_resolved, effort, prompt_v, schema_v, prices_v, key_label, retention_class,
  est_micros, in_tok, cache_write_tok, cache_read_tok, out_tok, reasoning_tok /*informational*/, cost_micros,
  status /*inflight|ok|invalid_output|refusal|truncated|rate_limited|budget_blocked|provider_error|timeout|usage_unknown*/,
  provider_request_id, workspace_id_seen, latency_ms, created_at, settled_at)
judge.ai_sample(evaluation_id, call_id, parsed jsonb)   -- C3; deleted at finalize; erased with the account
judge.llm_breaker(scope, open_until, reason)            judge.llm_calibration(...)  -- §7
```

| Step | Rule |
|---|---|
| **Admission (`Reserve`, at job claim)** | Sum `cost_micros` over the global-month, global-day, account-month and account-day periods (inflight rows count at their estimate). Admit if **sum + worst-case job ≤ limit − headroom(tier)**. Otherwise return `ErrBudgetExhausted{scope, resets_at}`. There are no holds, so nothing can leak. Two workers mean overshoot is at most one job (≈ $0.06 while only the analyzer runs, inferred). |
| **Headroom** | `final` = 0. `counted` = $0.35 at account-day and $2 at global-month, so cheap analyses can't push a learner's final into self-grading. `advisory` = 20% **and** pace-gated. |
| **Pace gate (advisory only)** | Allowed while global month-to-date ≤ app cap × min(1, 1.15 × elapsed-fraction + 0.05). At linear spend below 1.15 × cap it never sheds; it sheds only when spend runs ahead of pace (re-run). |
| **Per call** | Insert the row as `inflight` at `est_micros` → call → update from the provider's usage. The cost is always recorded, even above the estimate. |
| **Call cap** | `COUNT(llm_call WHERE evaluation_id=?) < 8`, checked before every call (persisted, so it survives a re-claim). |
| **Resume** | Parsed samples persist in `ai_sample`. A re-claimed job resumes. **At most 2 re-claims per job**, then `ai_unavailable`. For SD, Feedback runs as its **own llm job** after Score, so each job fits the 150 s budget. |
| **Sweeper** | An `inflight` row older than its lease + 60 s → `usage_unknown` at the estimate (fails safe). |
| **Prices** | `platform/llm/prices.go` with effective dates. A CI test requires a price for every catalog model; an unknown price is refused. |
| **Usage semantics** | `In` = uncached input only (Anthropic `input_tokens`; OpenAI `input_tokens − cached_tokens`). `CacheRead` and `CacheWrite` are separate; a 1 h write is priced at 2×. `Out` = all billed output including thinking. `Reasoning` is informational and **never priced**. Golden usage→cost tests per adapter. |
| **Truncation** | **Analyze (learner-controlled input): never retried** → `analysis_unavailable(truncated)`. Score and Feedback: one retry at 1.5 × `max_tokens`, **priced into admission**. The bound is "`max_tokens` × calls", not effort (corrected). |

**Degrade order** (utilisation is the max over the four scopes):

| Stage | What stops | What remains |
|---|---|---|
| 1 | `advisory` (D16 pass notes), when running ahead of pace or above 80% | — |
| 2 | `counted` analyzer and Feedback, at limit − headroom | Deterministic pre-fill; bands + verified quotes + descriptors |
| 3 | `final` Score, at the limit | "Wait for AI (resets …)" or D14 manual; v1 self-scoring for touches |

**Nothing blocks the learning loop.**

**Provider error mapping (platform credential).**

| Signal (verified unless marked) | Action |
|---|---|
| 400 "You have reached your specified **workspace** API usage limits" · 400 "You have reached your specified API usage limits" (org) · 429 `enforced_spend_limit_reached` (no `retry-after`) · **402 `billing_error`** · credit exhausted (message "credit balance is too low", inferred) · OpenAI 429 `project_/organization_spend_limit_exceeded` / `insufficient_quota` | **Open the global breaker** until reset or owner action; `ErrBudgetExhausted`; **immediate push**. If app month-to-date is under 70% of the provider limit → alert **"external spend or ledger bug: rotate or disable the credential"**. |
| Any other 400 on a locally validated request, on 3 consecutive calls | Open the breaker, immediate push |
| 401 or 403, or a token-exchange 401 | Open the breaker (queued jobs stop retrying); `ErrUnavailable`; immediate push with the Console auth-history hint |
| 429 with `retry-after`, 529 | Re-queue with `run_after`; counts inside the 8-call cap |
| `stop_reason: refusal` / OpenAI refusal | `ErrInvalidOutput`, `review_flag` |
| `anthropic-workspace-id` mismatch | Breaker, immediate push |

**Abuse and rate controls.**
- Invite-only (D13) is the primary control.
- T4 limits: ≤ 8 calls per evaluation, one re-grade, ≤ 3 releases, per-account serialization, lane queue ≤ 8.
- One analysis per `attempt_id`. **Skip** analysis when the draft is blank or equals the starter, the conclusion comes less than 60 s after start, or learner input exceeds **16 KiB** (`analysis_skipped(too_large)`).
- The analyzer calls no tools and requests no Runs.
- **HMAC pseudonym** in `metadata.user_id` / `safety_identifier`. Corrected: this helps the provider pinpoint a violation, but **enforcement stays org-level**. Real isolation is the dedicated org, plus Anthropic's advice to "warn, throttle, or suspend": **≥ 3 `review_flag`/refusal/injection hits in 7 days → `ai_disabled` pending owner review** (the learner is routed to manual and notified).

**Learner view.** `GET /api/me/ai-allowance` returns `{used_pct, resets_at, state: ok|low|paused, global_paused}`, as a percentage, never dollars. Erase sets `account_id`/`evaluation_id` to NULL on `llm_call` rows and deletes `ai_sample` rows. Global totals survive.

**Alerting (no metrics stack).**

| Channel | What |
|---|---|
| **Immediate, pushed by judge itself** (the T7 push channel, ntfy or email; opscheck is a daily CronJob and can't do this) | Breaker opened (with reason); credential 401 or token-exchange failure; workspace mismatch; global use ≥ 90% |
| **Console, zero code** | Workspace spend alerts at 50% and 80% of the provider limit |
| **opscheck daily digest (trimmed)** | Month-to-date vs cap, with a forecast; breaker events; `ai_unavailable` and invalid-output rates; truncation rate; `review_flag` list; top accounts above 80%; `usage_unknown` rows; calibration, JWKS and fallback-key age; `retire_not_before` within 90 days |
| **Monthly, manual** | Console vs ledger (±5%) |

## 7. Quality: calibration, regression and injection defences

**T4 §7 is confirmed:** `Scorer{Reserve, Score, Feedback, Analyze}`, the three schemas, k=3→5, ≤ 8 calls, quote verification, `review_flag`, QWK ≥ 0.70. Refinements:

**1. Where things live.**
- `Scorer` belongs to `internal/judge/ai`, the consumer.
- `platform/llm` holds only adapters, auth (WIF exchange or key), catalog and prices:
  - `Complete(ctx, Cred, Req) (Resp, error)`;
  - `Stream(ctx, Cred, Req, sink) (Resp, error)`;
  - `Req{Model, System[]Block, Messages, Schema, MaxOutputTokens, Effort, ThinkingDisplay, EndUser, NoStore}`, with no tools field;
  - `Resp{JSON|Text, Stop, Usage, ModelResolved, RequestID, WorkspaceID}`.
- Raw HTTP, **with no SDK auto-retries** (verified: SDKs retry twice by default). `ResponseHeaderTimeout` is 30 s.
- `ScoreRequest.ZeroRetention bool` becomes `Retention RetentionPolicy`.

**2. Schema changes** (not built yet, so no version bump):
- `band` becomes `enum:[1,2,3,4,5]`.
- `confidence` becomes `enum:["low","medium","high"]`, and the analyzer precedence "≥ 0.6" becomes "≥ medium".
- Optional values become `anyOf:[T,null]`, with every field required.
- `xlearn.code_analysis@1` gains `pass_review:{pointer_notes[{note,lines|null}], revisit{suggested,reason|null}}|null`, filled only on a pass.
- Must-cover and red-flag enums use **opaque ids** (`m1…`, `r1…`), because schemas are cached for up to 24 h outside the conversation (verified).

**3. The Go validator** enforces what the subset can't:
- string lengths (480/240/200/160);
- `findings ≤ 5`, `concepts ≤ 3`, `pointer_notes ≤ 3`;
- the complexity regex;
- ids exist in the manifest.

Over-long prose is truncated at a rune boundary. Quotes are verified or dropped, never truncated. **Validator errors carry field paths and ids only, never values.**

**4. Calibration gate, strengthened.** Enforced in code by `judge.llm_calibration(…, split, n_test, qwk, qwk_lb, within1, smd, passfail, passfail_lb, twins_ok, schema_ok, trunc_rate, p95_ms, configs_tried, passed)`.

| Item | Rule |
|---|---|
| Sets | **Dev** (≥ 30, used for prompt and effort tuning) and a **frozen test** split (**≥ 50 per rubric**, oversampled at the pass/fail boundary), both in private `xlearn-evalpack`. Each configuration touches the test split **once**; `configs_tried` is logged. |
| Sources | Owner-written responses plus synthetics from **≥ 2 model families**, reported by source. None from the grader's own family alone (self-preference). |
| QWK | Point estimate ≥ 0.70 **and bootstrap 95% lower bound ≥ 0.60**, per required criterion and overall, **reported on a production-weighted band mix** (for example 5/25/40/25/5) as well as on the test mix |
| Pass/fail agreement | ≥ 90%, lower bound ≥ 85% |
| Within-1, SMD, must-cover κ | ≥ 90%; ≤ 0.15; ≥ 0.6 |
| **Injection gate (outcome-based)** | On **held-out red-team twins written after the heuristic is frozen** (paraphrased, encoded, canvas-label, code-comment): **the band never rises** above the base response. `review_flag` recall is reported, not gated. |
| Padded twins | The band never rises |
| Other | `low_confidence` ≤ 20%; schema-valid 100%; truncation ≤ 2%; p95 per call ≤ 45 s |

Why the bound: at n≈30, a grader whose true QWK is 0.60 passes a single test about 18% of the time, and a uniform band mix inflates QWK (critic simulations).

**5. Independent re-grade.**
- The dispute runs on a **different passed configuration**: a second `prompt@v` with reordered anchors and different exemplars, or Opus 5.5 `medium`, at k=5. It never uses the appeal text.
- Any disagreement with the original goes to the owner audit list.
- **The 24 h auto-submit (D14) is skipped** when `review_flag` or `low_confidence` is set.
- **k-median is relabelled as variance control, not an injection defence.**

**6. Feedback consistency.**
- Feedback receives Score's **verified learner quotes and bands** (learner text only; still no pack).
- Gate: no "strength" on a criterion at band ≤ 2; an "improve" item for every criterion at band ≤ 3.

**7. Regression triggers.**
- Any change to model, effort, `prompt@v`, `rubric@v`, `schema@v`, the normalizer or delimiter code, or `anthropic-version`.
- A quarterly re-run on the pinned model.
- A resolved-model mismatch, which stops `Score`.
- **Production drift** (≥ 20 evaluations per rubric): edit/override rate > 30%, a spike in flags or invalid output, or an independent re-grade disagreement rate > 25%.
- **The owner blind-labels a small monthly sample of real evaluations (with consent) as the ground-truth drift check.**

**8. Injection and extraction defences.** The T4 controls stay: random-boundary delimiting, quote verification with a clamp to ≤ 2, `review_flag` → `trust=honor`, and "AI never flips a deterministic verdict". Added:
- Strip and flag Unicode tag characters (U+E0000–E007F), bidi controls and zero-width characters.
- Escape the boundary prefix and role markers.
- Add datamarking only if the twins fail.
- **`FeedbackRequest` cannot import `PackView`** (compile time) plus a contract test.
- `display:"omitted"`.
- Prompt templates are public; only the pack payload is secret.
- **Canary test:** a unique string injected into learner parts, dispute text and the pack payload must appear in no log line, ledger row, error string, event or opscheck output.

## 8. Privacy and residency

**Data classes.**

| Class | May go to |
|---|---|
| Pack material | Platform `Score` only |
| C3 learner content (code, text, canvas text export) | Score, Feedback, Analyze, **only while the learner's "xLearn AI reviews my work" is on** |
| C4 behavioral | Only after a separate opt-in; never Analyze |
| Pointer notes | **Stored per (account_id, item_id), class C3, learner-private.** Never on the public route or in events; erased with the account; withheld during live touches. |
| Identifier | Only the HMAC pseudonym. Never name, email, username, raw `account_id` or IP. |

**Retention and training (verified).**
- **Anthropic:**
  - The docs say content "is not retained by default" except for Covered Models. The Privacy Center says deletion happens "within 30 days", with flagged content kept up to 2 years. **Plan to and disclose the 30-day bound.**
  - Retained data is "never used for model training without your express permission".
  - ZDR is sales-only and per org.
  - Structured-output schemas are cached for 24 h (hence opaque ids).
- **OpenAI:** no training by default; abuse logs kept up to 30 days; Responses are stored for at least 30 days unless `store:false`; ZDR and Modified Abuse Monitoring need approval.

**Residency.**
- There is no Indian inference option. Anthropic offers `global` or `us` (×1.1). OpenAI's India region needs approval.
- Use `global`; all processing is outside India.

**DPDP (verified).**
- s.16 and Rule 15 are expected to take effect **2027-05-13**, on a negative-list model, and no restricted-country list has been notified, so the transfer is lawful.
- Notice, consent, security and rights duties apply from about 2027-05-13. **Ship them before M4 opens to learners anyway.**
- Invite terms require **18+ by attestation** (DPDP s.9 requires verifiable parental consent for minors).
- CERT-In obligations may not apply to an individual owner (inferred); the logs are in India anyway.

**Consent (fixed: free, specific, withdrawable).**
- At invite acceptance there are **two unticked, explicit choices**:
  1. "xLearn AI reviews my graded work" (Analyze, Score, Feedback);
  2. "…and also reviews my passing solutions for improvement notes" (D16).
- A **third opt-in** covers behavioral answers.
- All three are toggles in Settings. Turning one off routes to D14 manual entry and v1 self-scoring, effective immediately.
- Consent version and timestamp are stored.
- Copies already at the provider expire on the provider's schedule.

**Logging.**
- Never log prompts, completions, thinking, learner parts or pack text.
- Allowed fields: request id, provider `request-id`, workspace id, model, `prompt@v`, `schema@v`, effort, usage, µUSD, latency, outcome enum, account HMAC.
- A `slog.LogValuer` renders `[redacted N bytes]`. Provider error bodies are parsed (≤ 8 KiB) and never echoed.
- No panics with content in the value (the `httpx` recoverer logs `rec`); this is a T1 log-lint item.
- The canary test (§7) enforces all of the above.

**Privacy-notice items:**
- which features send what to Anthropic, as a processor under its DPA, and why — **naming pass-solution review explicitly**;
- processing is outside India;
- no training;
- provider retention of ≤ 30 days, up to 2 years if flagged, and the ZDR status;
- the three toggles and what switching each off does;
- AI results are editable suggestions with a manual path (D14);
- BYO chat and interviews go to **the learner's own** provider account, under the learner's terms;
- how to access and erase;
- a grievance contact.

## 9. Coach and BYO changes in v2

**Persona and mode gate.**
- The manifest gains `coach{persona ≤ 600 chars, primary_language, off_during:[touch, mock]}`. The gateway resolves `path_slug` server-side.
- **Prompt order:**
  1. the fixed frame, `coach-prompt@2`;
  2. the persona;
  3. **the mode rules, last, as hard constraints**;
  4. the context.
- `coach_message` gains `prompt_v`, `path_slug` and `attempt_id`.
- A new `GET /attempts/open` (practice, JWT-scoped) replaces `/state/{id}`. Modes come from T1's `withhold()`:

| Mode | When | What the prompt gets |
|---|---|---|
| `locked` | Any live touch or mock on the account | 409 `coach_paused`; nothing is sent |
| `attempt` | Open counted attempt, due touch, or never solved | No pattern, concepts or solution facts. Fixes `gateway/coach.go:313` and `prompt.go:84-86`. |
| `review` | Concluded, no touch due | Final submission (≤ 16 KiB), xLearn AI feedback and pointer notes. Never hidden data or anchors. |
| `general` | Not an item | A guard line naming live items as off-limits |

**D18 capture (Q4).**
- While a counted course attempt is open, `POST /api/coach/chat` without `assist_ack:<attempt_id>` returns **409 `assist_confirm_required`**.
- With the acknowledgement, the gateway calls practice `POST /attempts/{id}/assist` (idempotent) **before** forwarding. If that fails, the gateway returns 503 and sends nothing (fails closed).
- practice's strategy rule: `coach_assist_at ≤ lock_at` → ceiling Assisted. `problem_solved` v2 gains `assist{hint, coach}`.
- **Closes the T6 bypass:**
  - while any counted course attempt is open, coach **refuses to mint realtime or derived credentials** and T6 **refuses to start a mock** (409 `attempt_open`);
  - pointer notes stay collapsed during a live touch.
- **Honesty wording:** the "no coach" part of Clean and Rough is **self-attested**; only in-app use is detected. The UI copy is an honesty prompt ("Using any AI during a counted attempt caps it at Assisted"), not a claim of enforcement. ADR-0029 notes the same.
- A failed provider call after the acknowledgement still caps the attempt (accepted: the learner chose to ask).

**Per-feature keys.** `coach.key_default(account_id, feature ∈ {coach, interview}, provider, model)`. Backfill from `is_default`, then drop that column in the contract step. An `interview` default must point at a catalog model with the `realtime` capability.

**Providers and catalog.**
- A registry, `map[string]llm.Provider`, from which validation is derived. The DB `CHECK` stays and is widened by migration.
- v2.0 providers: Anthropic and OpenAI. **Remove "Google" from onboarding** (`Auth.tsx:699`).
- **No user-supplied base URLs** (SSRF).
- `GET /api/coach/models` returns id, capabilities, price, `as_of` and a recommended default. It replaces the stale SPA list and adds `claude-opus-5-5`, `gpt-6-sol`, `gpt-6-luna` and `gpt-6-astra`.
- Custom ids must match `^[A-Za-z0-9][A-Za-z0-9._:-]{0,63}$` and show as "cost unknown".
- **Covered Models are allowed on BYO** (the learner's own org), labelled "your provider keeps these chats 30 days". Corrected from "block".
- **Defaults:** `claude-sonnet-5`, and `gpt-6-sol` (half the price of v1's `gpt-5.6-sol`) after a smoke test.

**Usage display and limits.**
- Parse usage and `stop_reason` from both streams (OpenAI `stream_options.include_usage`).
- Store tokens and `est_cost_micros` on `coach_message`.
- Settings shows "This month on your keys: messages · tokens · ≈ $x (estimate)", for display only.
- **Limits:** 20 messages a minute, 2 concurrent streams, **300 messages a day per account**, and a history cap of the last 20 turns or 32 KiB. This bounds what a stolen xLearn session can spend from a learner's key.
- Onboarding copy recommends setting provider-side limits on the key.

**ADR-0007 fixes.**

| Priority | Fix |
|---|---|
| **P0 (M1)** | Typed errors. **Only** 401, `authentication_error` and `invalid_api_key` disable a key. `insufficient_quota`, 402 `billing_error`, spend-limit 400/429, rate-limit, model and region errors never do. Today `providers.go:133` maps quota errors to `ErrProviderAuth`, which becomes `enabled=false` (`handlers.go:381`). |
| **P0 (M1)** | Truncation. `max_tokens` 1024 applies to Anthropic only today, while Sonnet 5 thinks at `high` by default. Send `max_tokens` 4096 with `effort:"low"` (OpenAI: `max_completion_tokens` 4096 plus `reasoning_effort:"low"`), check `stop_reason`, and mark cut-short replies. |
| **P0 (M1)** | The onboarding key step discards the key: the `<input>` is uncontrolled (`Auth.tsx:708`). Wire it to `PUT /api/coach/key` and drop `onboarding.key_added`. |
| P1 | `store:false` on OpenAI BYO calls |
| P1 | Server catalog; unknown models → 422 unless the id format is valid and flagged as custom |
| P1 | AEAD associated data `xlearn/coach/key/v1\|<account_id>\|<provider>` (`nil` today, `secrets.go:118,133`) |
| P1 | A `kek_id` column plus a `COACH_MASTER_KEYS` keyring, with background re-wrap |
| **P1 (T6)** | **Derived credentials:** shortest provider TTL; one per mock session; session configuration set server-side; each mint logged; never minted while the account is `locked` or a counted attempt is open |
| Doc | "The gateway never sees the key" becomes "the gateway transits the PUT body in memory, never logging or storing it" (`gateway/coach.go:153`), backed by a test |
| Doc | ADR-0020 is stale: it says `UNIQUE(account_id)`, but migration `00003` uses per-(account, provider) keys plus `is_default` |

**Onboarding step 3/4.** "Optional: bring your own AI coach." Copy: "xLearn AI already reviews your submissions — no key needed." Buttons: **[Connect & finish] [Skip for now]**. The **consent toggles (§8) sit on the invite-acceptance step, not here.**

**Names in the UI.** **"xLearn AI (included)"** is the platform AI. **"Your AI coach (your key)"** is BYO.

## 10. What T5 constrains downstream

**T6 (interviews and mocks)**
- BYO only, through the `interview` default key. No platform key for mocks in v2.0 (confirms T4 §11.2).
- The raw BYO key never leaves coach. Realtime uses **derived credentials minted by coach** under the ADR-0007 clause, or streams proxied by coach.
- **No mock starts, and no credential is minted, while a counted course attempt is open.**
- Public rubric descriptors only. Output is labelled `ai-byo`, `trust=honor`. `ScoreMock` runs once. Coach is `locked` during a mock.
- Shared typed errors: `ErrQuota` means a top-up message, never a key disable. The catalog's `realtime` capability picks the model.

**T7 (build, rollout, ops)**

| When | Items |
|---|---|
| **M1** | P0 coach fixes; mode gate plus `withhold()`; D18 capture (`assist_confirm_required`, practice assist endpoint, event field); BYO daily cap |
| **Before M4 (gates)** | **WIF spike (≤ ½ day)**; **chart 0.3.0 egress template plus judge default-deny egress**; provider runbook (dedicated org if shared, workspaces, spend limit **and alerts**, rate limits, credits with auto-reload off, WIF issuer, rule and service account); push channel callable from judge |
| **M4** | `platform/llm` extraction (adapters, WIF auth, catalog, prices, usage golden tests); `internal/judge/ai`; `llm_call` + `ai_sample` + `llm_breaker` + `llm_calibration`; analyzer acceptance set (dev/test) and **caps sized from it**; `/api/me/ai-allowance` and UI; consent toggles; opscheck AI digest; canary log test |
| **Before learners** | Privacy notice; 18+ attestation; invites carry an allowance tier; two weeks of dogfood re-measurement |
| **SD-course milestone** | Score/Feedback split into separate jobs; sample-1-first caching; dev/test calibration sets (≥ 80 labelled per rubric); independent re-grade configuration; Q1 sign-off |
| **Artboards** | xLearn AI suggestion card (accept / edit / re-grade; no auto-submit when flagged); "Wait for AI vs grade it myself"; allowance meter and paused state; consent toggles; coach assist warning (honesty wording) and paused banner; collapsed pointer notes |

**Open signup (D21)** triggers a re-look at per-account budgets, BYO for Feedback and Analyze, the `xlearn-llm-egress` proxy, and extracting an `ai` service.

**Amendments to settled documents**

| Doc | Amendment |
|---|---|
| **T4 §7 / ADR-0029** | `Scorer` in `internal/judge/ai`; `RetentionPolicy`; portable subset, enums and `anyOf` null; `pass_review`; `Reserve` = admission check (no holds); persisted call cap and samples; re-claim cap 2; Feedback as its own job for SD; truncation rule; effort and thinking in `prompt@v`, no temperature; re-grade on an independent configuration through the gateway JWT; no auto-submit when flagged; analyzer skip rules; record resolved model and `provider_request_id`; §9.3 SD cost becomes $0.11–0.17 and re-grade $0.12–0.19; D18 honesty wording |
| **T1 §10 / ADR-0027** | "Retention off" becomes `RetentionPolicy`; no Covered Models, Batch, Files or tools for pack calls; pointer notes C3 per (account, item) |
| **T0 §10** | judge is the key holder (the credential is WIF) |
| **T3** | judge egress moves from Track B to an **M4 gate** |
| **D16 / D18 wording** | Q3 and Q4 |
| **Dependency matrix** | Rows for the WIF issuer/rule, the provider workspace and alerts, the `xlearn-judge-llm` SOPS secret (salt and break-glass), and "no Admin keys in the cluster" |
| **New ADR-0031** | "Platform AI and two-tier keys" |

## 11. Alternatives considered

| Alternative | Verdict | Why |
|---|---|---|
| Platform key in coach (single egress) | Rejected | Breaks D5; puts all secrets in one pod; grading would depend on a 128 Mi chat pod |
| New `ai` gateway service (custom or LiteLLM-style) | Deferred | A ninth service for one consumer, and it needs service auth; §3 triggers |
| **Credential-injecting `xlearn-llm-egress` now** (security critique) | **Deferred with triggers** | A compromised judge can leak the pack through its own learner-visible output regardless; WIF already makes a stolen credential short-lived; judge egress is closed to the cluster at M4 anyway |
| CONNECT allowlist proxy to `api.anthropic.com` (draft's upgrade path) | **Rejected** | TLS pass-through can't strip tools or overwrite auth, so an attacker's own key passes straight through |
| Static SOPS key with 90-day manual rotation as the primary | Fallback only | WIF removes the standing secret and adds the `workspace:inference` scope (verified) |
| Reserve→commit ledger with `llm_budget` + `llm_reservation`, 5 bands, lock ordering | **Rejected for v2.0** | Protects about $1 of overshoot on a 2-worker lane while adding hold-leak bugs; replaced by one append-only table |
| Fixed-percentage shedding bands | **Rejected** | Normal spend would shed D16 notes for the last ~9 days of every month; pace-based instead |
| D16 on every touch pass | Owner option (Q3b) | ×1.9 analyzer volume; notes already persist |
| Same-configuration blind re-grade | **Rejected** | Correlated errors; measures variance, not correctness |
| Single n=30 calibration set, point-estimate QWK | **Rejected** | Too noisy and overfit-prone; dev/test plus bootstrap lower bound |
| BYO fallback when the budget is exhausted | Rejected for v2.0 | ADR-0007 decrypt boundary, T1 pack rule, uncalibrated models |
| BYO opt-in for Feedback and Analyze | Rejected for v2.0 | Needs coach egress for judge plus service auth; revisit on open signup |
| Two providers active/active with failover | Rejected | Doubles calibration, disclosures and secrets; D14 covers outages |
| `gpt-6-sol` as primary | Challenger | About 15–20% cheaper per task, but 2 days old and stored by default |
| `gpt-6-luna` analyzer | Pre-approved lever | About 25× cheaper; a second processor; must pass the gate |
| Opus 5.5 for all Score calls | Escalation and re-grade configuration only | +30% at SD 10% |
| Fable 5.1 / Haiku 4.5 on the platform | Excluded | Covered Model (30-day retention, 5× price) / retires not sooner than 2026-10-15, no effort |
| Block Covered Models on BYO | Rejected | The learner's own org and retention; label it instead |
| Batch or Flex; sample-1-first in M4 | Deferred | Saves about $3–4.5/month / only pays off for SD |
| `inference_geo:"us"` | Declined | +10% with no DPDP gain |
| Admin or cost-API key in the cluster | Rejected | Full access, no scopes (verified); reconcile manually |
| SDKs with built-in retries | Rejected | Break the call cap and the ledger |
| Allowance shown in dollars | Rejected | A percentage is clearer and hides pricing |

## 12. Owner decisions — resolved 2026-09-24 (they OVERRIDE the body where they conflict)

| # | Question | Decision |
|---|----------|----------|
| **D24** | Q1: retention for pack-bearing `Score` | **Accept `RetentionPolicy`.**<br>• Provider retention ≤ 30 days (up to 2 years only if flagged); no training.<br>• Messages API only: no Batch, Files, tools or Covered Models.<br>• Disclosed in the privacy notice.<br>• ZDR is requested opportunistically.<br>• `LLM_ACCEPT_STD_RETENTION=true` is the owner's attestation. |
| **D25** | Q2: monthly platform-AI ceiling (invite-only) | **$100/month** as the provider workspace hard limit. The app cap is **$80** (80%). The dogfood defaults (provider $15 / app $12) apply while only the owner uses it. Caps are re-sized from the analyzer acceptance run, within this ceiling. |
| **D26** | Q3: D16 pass-review scope | **Course passes, plus a revision-touch pass only when its solution is *materially different*** from the last reviewed solution for that problem.<br>• "Materially different" = a normalized-code fingerprint differs by more than a threshold. Whitespace, comments and renames are normalized away; the threshold is tuned in the analyzer acceptance run.<br>• If the review finds the different touch solution **correct but improvable**, the revision item is marked **"correct, with improvements"** and gets pointer notes (and optionally a suggested revisit).<br>• **No effect on the touch's pass/fail, the grade or the ladder.**<br>• Identical or near-identical touch resubmissions are not re-reviewed; the existing notes still show.<br>• It stays `advisory` tier: shed first under budget pressure.<br>• This replaces Q3(a) "first pass per item". |
| **D27** | Q4: D18 coach-assist scope | **Per problem.** Only coach chats whose context is **that problem** during its open counted attempt require the confirm and cap the grade at **Assisted**. Asking from another page or item is an **accepted bypass**; "no outside AI" is honor-based. **Dropped** from §9: the "no mock or realtime while any counted attempt is open" rule. **Kept:** the coach is `locked` during live touches and mocks. The `withhold()` mode gate still keeps pattern and concepts out of chats about live items. |

## 13. Risks

| Risk | Mitigation |
|---|---|
| The analyzer needs `medium`: cost ×2.2 | Caps sized after the acceptance run; Q2 ceiling; levers in order (`low` with thinking off, then Luna, then D16 scope) |
| WIF spike fails, or `jti_reused` after an in-place restart | `check_jti=false` on the one-rule issuer, or the expiring service-account key fallback |
| k3s SA signing key rotates; inline JWKS goes stale | Breaker plus immediate alert naming the auth-history reason; runbook re-paste; check at every k3s upgrade |
| ZDR never granted and Q1a rejected | `ai_rubric` stays off; M4 is unaffected; D14 manual path |
| Model churn (GPT-6 is 2 days old, Haiku retires, prices move) | Model classes; the gate; `retire_not_before` alerts; price table with effective dates and a CI test |
| Cost drift from tokenizer, thinking or grade mix | Explicit effort; worst-case admission; no truncation retries on Analyze; re-measure after two weeks |
| Provider cap is shared org-wide or enforced late | Dedicated org; app cap at 80%; Console alerts; breaker on the first cap error; the "ledger < 70% at a cap error" alert |
| External spend on a leaked credential goes unseen | WIF short-lived tokens; the ledger-vs-cap alert; Console alerts; monthly reconciliation; Hostinger login treated as tier-0 |
| A judge compromise leaks the pack through its own output | Accepted (signal integrity only); egress gate; rotate the pack per T1; proxy on the triggers |
| Injection launders a grade into `trust=checked` | Delimiting, character filter, quote clamp, outcome-based held-out twins, independent re-grade, no auto-submit when flagged |
| Calibration overfits a small set | Dev/test split, n_test ≥ 50, bootstrap lower bound, production-mix reporting, monthly owner blind labels |
| A cheap analyzer invents "improvements" to optimal code | False-pointer gate ≤ 10%; notes are advisory, collapsed during touches, and can be switched off |
| D18 is gamed with outside AI | Stated as honor-based; in-app detection only; mocks and realtime refused during attempts |
| Recording coach use fails closed when practice is down | Accepted; coaching outside attempts is unaffected |
| The two Anthropic retention sources disagree | Plan to and disclose the 30-day bound |
| The authoring load for calibration sets (≥ 80 per rubric) | Only at the SD-course milestone; synthetic plus owner-written; this is the price of a valid gate |
| GST and FX raise the owner's real cost | Budgets in pre-tax USD; forecast in the digest |

## 14. Critic findings addressed

**Critic 1: cost, abuse and solo ops (6/10).**

| # | Finding (severity) | Disposition |
|---|---|---|
| 1 | Fixed-percentage bands shed D16 and analysis under normal spend (major) | **Fixed.** Pace-based gate for advisory work; hard cap only at 100% minus headroom (§6); re-run confirms no shedding at linear spend below 1.15 × cap |
| 2 | Analyzer token profile unmeasured; `medium` breaks the caps; `max_tokens` 2,500 truncates (major) | **Fixed.** `medium` row added ($2.84–3.17/lrn-mo); effort and `max_tokens` from the acceptance run (p99 × 1.5, provisional 4,000); truncation and $/analysis in the gate; caps sized after the run (§4–§6) |
| 3 | D16 scope silently narrowed (major) | **Fixed.** Scope made explicit: first pass per item (+13%); all touch passes priced at ×1.9; **owner Q3** |
| 4 | "Effort pinned so injection can't inflate spend" is wrong; 2× retry on truncation (major) | **Fixed.** Bound restated as `max_tokens` × calls; no truncation retry for Analyze; Score/Feedback retry at 1.5× priced into admission; input cap 16 KiB |
| 5 | Call cap and samples not persisted; re-claims re-spend; 150 s budget tight (major) | **Fixed.** Inflight row first; `COUNT` over `llm_call`; `ai_sample` resume; ≤ 2 re-claims; SD Feedback as its own job |
| 6 | Error table misses the org-limit 400, 402 and credit exhaustion (major) | **Fixed.** All mapped to breaker plus immediate push; repeated-400 rule; 401 → breaker (§6) |
| 7 | Provider cap assumed, never checked; "auto-reload capped" not possible (major) | **Fixed.** `anthropic-workspace-id` check; Console spend alerts; ledger-vs-cap external-spend alert; auto-reload OFF, balance about 1–2 months |
| 8 | Reserve→commit machinery over-built (major) | **Fixed.** Single append-only `llm_call`; claim-time admission check; three tiers via headroom; `llm_budget` and `llm_reservation` dropped |
| 9 | Sample-1-first pays only for SD (minor) | **Adopted.** Deferred to the SD milestone |
| 10 | opscheck is daily, so it can't send "immediate" alerts; digest bloated (minor) | **Fixed.** judge pushes immediate alerts itself; digest trimmed |
| 11 | Kill switch slow; rotation depends on a reminder (minor) | **Fixed.** Console disable is step 1; WIF removes rotation; fallback key carries a provider-side expiry |
| 12 | Re-grade priced at k=3.4; reservation used 3,000 not 4,000 tokens (minor) | **Fixed.** Re-grade at k=5 ($0.12–0.19); SD worst case $0.25 derived from §4's `max_tokens` |
| 13 | One-off calibration costs understated (minor) | **Fixed.** $20–60 per rubric, $10–30 for acceptance; `xlearn-calib` about $50/month during bring-up |
| 14 | Usage field semantics undefined (minor) | **Fixed.** Definitions plus golden tests; reasoning never priced |
| 15 | BYO drain by a stolen account (minor) | **Fixed.** 300 messages/day, history cap, onboarding advice |

**Critic 2: security, privacy and grading quality (7/10).**

| # | Finding (severity) | Disposition |
|---|---|---|
| 1 | Static long-lived key; WIF not considered (major) | **Fixed.** WIF with inline JWKS and `workspace:inference` is primary behind a ½-day spike; expiring service-account key as fallback; GitHub OIDC for calibration CI. **Nuance:** Hostinger images still hold the k3s SA signing key, so WIF doesn't remove that exposure (§3) |
| 2 | judge gets open :443 egress; chart is ingress-only; CONNECT allowlist doesn't stop exfiltration (major) | **Fixed in part.** Default-deny egress is now an **M4 gate**; CONNECT upgrade rejected; tools stripped by construction. **Why the proxy isn't adopted now:** a compromised judge can leak the pack through its own learner-visible output anyway; the proxy is deferred with triggers (§3, §11) |
| 3 | D18 bypassed via T6 realtime; adverse selection (major) | **Fixed.** No realtime mint and no mock start during open counted attempts; derived-credentials clause; honesty wording in ADR and UI |
| 4 | Calibration gate statistically weak; twins tautological; self-preference conflict (major) | **Fixed.** Dev/test split, n_test ≥ 50, bootstrap lower bound, production-weighted mix, held-out outcome-based twins, ≥ 2 model families, monthly owner blind labels |
| 5 | Re-grade uses the same configuration; k-median isn't an injection defence (major) | **Fixed.** Independent passed configuration; k-median relabelled; no auto-submit when flagged |
| 6 | Consent bundled at invite; no withdrawal (major) | **Fixed.** Three explicit toggles, withdrawable in Settings; off → manual; consent version stored; pass review named in the notice |
| 7 | HMAC user-id claim overstated (minor) | **Fixed.** Reworded; flag-based throttle → `ai_disabled` pending review |
| 8 | Thinking text reaches judge (minor) | **Adopted.** `display:"omitted"` explicit; noted it is already the default on Sonnet 5 and Opus 5.5 |
| 9 | Logging rules unenforced (minor) | **Fixed.** Canary test; validator errors carry paths and ids only |
| 10 | Pointer-note ownership unspecified (minor) | **Fixed.** Per (account_id, item_id), C3, private |
| 11 | Cheap analyses push a final into self-grading (minor) | **Fixed.** Headroom for finals; "Wait for AI" option |
| 12 | Feedback can contradict Score (minor) | **Fixed.** Score quotes and bands passed in; consistency gate |
| 13 | Blocking Covered Models on BYO adds nothing (minor) | **Fixed.** Allowed with a label |

**Verified-wrong claims in the draft, corrected:**

| Claim | Correction (source, seen 2026-09-24) |
|---|---|
| Cache read "—" for Fable 5.1, Haiku 4.5, gpt-6-luna | $0.25, $0.10 and $0.01 respectively (Anthropic and OpenAI pricing) |
| temperature, top_p and top_k "return 400 on 4.7+" | Only when set to a **non-default** value (model deprecations) |
| "Effort is pinned, so injection can't inflate spend" | "Effort is a behavioral signal, not a strict token budget"; `max_tokens` is the hard limit (Effort) |
| "Auto-reload off or capped" | No monthly cap exists; only OFF is a hard stop (Help Center) |
| The error table listed only the workspace-limit 400 | Org 400 "You have reached your specified API usage limits" and 402 `billing_error` also exist (Rate limits; Errors) |
| "Immediate" opscheck alerts | opscheck is a daily CronJob (feasibility row 126); judge pushes instead |
| Re-grade $0.10–0.15; k=3 reservation $0.22 | $0.12–0.19 at k=5; $0.25 at `max_tokens` 4,000 |
| $1.2–1.7 per learner-month as "expected" | Valid only at `low` without D16 touch passes; now a range of $1.3–3.2 |
| Provider enforcement "hits one learner" through `user_id` | IDs help Anthropic "pinpoint violations"; enforcement is org-level (Safeguards article) |
| Static key and rotation are the only option | WIF with inline JWKS and an inference-only scope exists (WIF docs) |
| Egress described as an addition to a restricted list | The chart NetworkPolicy is ingress-only (chart **0.2.2**, not 0.2.1); Track B gates nothing |

### Sources

**Fetched or checked for this final version, 2026-09-24:**
- [Anthropic pricing](https://platform.claude.com/docs/en/about-claude/pricing): all Claude prices and cache multipliers; Sonnet 5 $2/$10 made standard; tokenizer +30%; `inference_geo` ×1.1; Batch −50%.
- [Workload Identity Federation](https://platform.claude.com/docs/en/manage-claude/workload-identity-federation), [WIF reference](https://platform.claude.com/docs/en/manage-claude/wif-reference), [WIF with Kubernetes](https://platform.claude.com/docs/en/manage-claude/wif-providers/kubernetes): inline JWKS, `workspace:inference`, lifetimes 60–86400 s, `jti` single use and `check_jti`, `ANTHROPIC_API_KEY` precedence.
- [Authentication](https://platform.claude.com/docs/en/manage-claude/authentication): service-account keys; key expiration; expired key → 401.
- [Rate and spend limits](https://platform.claude.com/docs/en/api/rate-limits): tier caps (Start $500); 429 `enforced_spend_limit_reached`; 400 org and workspace limit messages; no limits on the Default workspace; Evaluation tier; `anthropic-workspace-id`.
- [Workspaces](https://platform.claude.com/docs/en/manage-claude/workspaces): Spend limits tab with alerts; Claude Code workspace in the same org; per-workspace cache isolation.
- [API errors](https://platform.claude.com/docs/en/api/errors): 402 `billing_error`; SDK auto-retries; thinking can't be disabled on Opus 5.5 or Fable.
- [Effort](https://platform.claude.com/docs/en/build-with-claude/effort): defaults (Sonnet 5 `high`, Opus 5.5 `medium`); "behavioral signal, not a strict token budget"; Sonnet 5 `low` for "chat and non-coding".
- [Thinking](https://platform.claude.com/docs/en/build-with-claude/thinking): `display:"omitted"` is the default on Sonnet 5 and Opus 5.5; billed as output; counts toward `max_tokens`; Sonnet 5 allows disabling.
- [Model deprecations](https://platform.claude.com/docs/en/about-claude/model-deprecations): retirement dates; ≥ 60 days' notice; sampling parameters 400 only when non-default.
- [API and data retention](https://platform.claude.com/docs/en/manage-claude/api-and-data-retention): Covered Models 30-day retention; ZDR via sales, per org; Batch 29 days; schemas cached 24 h; no training.
- [Structured outputs](https://platform.claude.com/docs/en/build-with-claude/structured-outputs) and [Prompt caching](https://platform.claude.com/docs/en/build-with-claude/prompt-caching): unsupported keywords; 512-token minimum; "a cache entry only becomes available after the first response begins"; usage-field semantics.
- [How do I pay for Claude API usage](https://support.claude.com/en/articles/8977456-how-do-i-pay-for-my-claude-api-usage): prepaid credits; auto-reload threshold and amount; credits expire after 1 year.
- [API safeguards tools](https://support.claude.com/en/articles/9199617-api-safeguards-tools): hashed user IDs; "warn, throttle, or suspend".
- [OpenAI pricing](https://developers.openai.com/api/docs/pricing): GPT-6 Sol, Luna and Astra; GPT-5.6; Batch/Flex −50%; regional +10%.
- [OpenAI spend limits](https://developers.openai.com/api/docs/guides/spend-limits): error codes; "Enforcement is not instantaneous"; alerts.
- [OpenAI data controls](https://developers.openai.com/api/docs/guides/your-data): no training by default; 30-day abuse logs; Responses stored unless `store:false`; ZDR by approval; India residency.
- [TechCrunch, GPT-6 Sol and Luna launch, 2026-09-22](https://techcrunch.com/2026/09/22/openai-launches-gpt-6-sol-and-luna/) and [DigitalApplied](https://www.digitalapplied.com/blog/gpt-6-sol-luna-launch-pricing-benchmarks-2026).
- DPDP s.16 / Rule 15 (2027-05-13, no restricted list): [Legal500](https://www.legal500.com/intelligence/india/privacy/cross-border-data-transfers-under-indias-dpdp-act-rules-restrictions-and-compliance-roadmap-for-2027), [dpdpa.com Rule 15](https://www.dpdpa.com/dpdparules/rule15.html), [Bar & Bench](https://www.barandbench.com/columns/the-dpdp-cross-border-transfer-rules-arent-live-yet-so-why-are-contracts-being-redrafted-as-if-they-are).

**From the slices and critiques, not re-fetched (seen 2026-09-24):**
- [Anthropic Privacy Center, 30-day deletion](https://privacy.claude.com/en/articles/7996866-how-long-do-you-store-my-organization-s-data)
- [OpenAI structured outputs](https://developers.openai.com/api/docs/guides/structured-outputs)
- [OpenAI error codes](https://developers.openai.com/api/docs/guides/error-codes)
- [OpenAI safety identifier](https://help.openai.com/en/articles/5428082-how-to-incorporate-a-safety-identifier)
- [OpenAI Realtime](https://developers.openai.com/api/docs/guides/realtime)
- [GST on foreign AI tools (third-party)](https://startuptalky.com/getting-wrong-gst-chatgpt-claude-foreign-ai-tools/)
- Papers: [Spotlighting](https://arxiv.org/abs/2403.14720), [JudgeDeceiver](https://arxiv.org/abs/2403.17710), [self-preference bias](https://arxiv.org/abs/2404.13076)
- Critic simulations: `scratchpad/t5crit/qwk_sim.py` and `qwk_skew.py`

**Repo spot-checks:**
- `../infra/charts/project/templates/networkpolicy.yaml` (ingress-only; `Chart.yaml` version 0.2.2)
- `docs/v2/feasibility.md` rows 14–25 and 126
- `docs/v2/research/t4-judge-contract.md` §7, §9.1 (llm lane: 2 workers, 150 s, 30 s lease), §2.4, §13
- `docs/v2/research/t2-object-storage-backups.md` L91 (Hostinger images hold k3s Secrets and `sops-age`)
- `docs/v2/research/t3-sandbox.md` L587 (Track B egress)
- Coach and secrets line references as cited in the draft (checked by critic 2): `internal/coach/providers.go:133`, `handlers.go:381`, `internal/platform/secrets/secrets.go:118,133`, `web/src/screens/Auth.tsx:699,708`

---

## 15. WIF spike result (spk-03, 2026-09-25)

Run 2026-09-25 on a throwaway multipass VM (D41 pulled the spike forward; launching the prompt was the WIF go-ahead, D23/D40). Everything on the VM was thrown away. Only this section and the status rows are committed. **No token value was printed, logged or committed.** The projected tokens stayed in the pods and on the VM. The access tokens existed only in a pod's `/tmp` for the length of one script. Every value below is a decoded claim or a non-secret response field.

**Environment**

| Item | Value |
|---|---|
| VM | `xlearn-wif`: multipass on the owner's Mac, Ubuntu 24.04 **arm64**, 2 CPU / 4 GiB. The architecture doesn't matter here: token issuance and rotation are Kubernetes behaviour |
| k3s | `v1.36.4+k3s1` (production's pin), installed with `--disable traefik`; containerd `2.3.4-k3s1.36` |
| Issuer / JWKS | `iss` = `https://kubernetes.default.svc.cluster.local` (the same string production uses); one RSA-2048 `RS256` key, kid `W2B3g78jobYPZv69l_I-FYASHgMHbSzqcSdX_woZANY` |
| Org | **Sujay's Individual Org** (`ced627ab-c84c-4f9c-b126-a252462a1ae0`), the owner's current org; no dedicated xLearn org exists. WIF works in it |
| Console objects (the owner's, created before launch) | Workspace `xlearn-wif-spike` (`wrkspc_017ZKXnJ1a9BxwcEm7e9rhGi`; $1 monthly limit; auto-reload off). Service account `xlearn-wif-spike` (`svac_019eVYj3HcGdou7MMvmDWzZN`; org role Developer). Issuer `xlearn-wif-spike-k3s` (`fdis_01VKrjGxwBjDDrCASJS1CUQw`; inline JWKS; **JTI replay protection on**, the default; max JWT lifetime 1 h). Rule `xlearn-wif-spike-probe` (`fdrl_01JM53aDwbkXy1RJ2fhG9LCf`; subject exactly `system:serviceaccount:wif-spike:wif-probe`; audience `https://api.anthropic.com`; the one workspace; **scope `workspace:developer`**; token lifetime 1 h) |
| Probes | Namespace `wif-spike`; ServiceAccount `wif-probe` (`automountServiceAccountToken: false`); pods `wif-probe-3600` and `wif-probe-600` with the plan's spec (projected token at `/var/run/secrets/anthropic.com/token`, audience `https://api.anthropic.com`). Runs that need a fresh `jti` use fresh pods with the same spec. Raw `curl` + `jq`; `ANTHROPIC_API_KEY` was never set |
| Spend | 5 Messages calls on `claude-sonnet-5`, each 16 input / 4 output tokens with thinking disabled: under $0.001 |

**Results**

| Q | Check | Result |
|---|---|---|
| **Q-W1** | Claims of the projected token | Header: `alg` `RS256` and `kid` (no `typ`). Payload: `aud` (an array: `["https://api.anthropic.com"]`), `exp`, `iat`, `nbf` (= `iat`), `iss`, `sub` (`system:serviceaccount:wif-spike:wif-probe`), **`jti` present (a UUID)** ✅, and `kubernetes.io` {`namespace`, `node` {name, uid}, `pod` {name, uid}, `serviceaccount` {name, uid}}. `exp − iat` is exactly 3600 (or 600), within the issuer's 1 h maximum |
| **Q-W2** | Rotation, sampled every 60 s on the VM host | **600 s:** 5 rotations, at 486–544 s after `iat` (**81.0–90.7 %** of TTL; median 517 s). **3600 s:** one rotation at **2881 s (80.0 %)**, inside the in-place-restarted container. The kubelet marks a token due at 80 % of TTL (minus up to 10 s of jitter). It rewrites the file at its next pod sync, so a rotation lands 0 to ~90 s after the 80 % point: at most ~2970 s for 3600 s. A k3s (kubelet) restart re-issues every projected token at once; seen twice, each time with a new `iat` and `jti` within seconds of k3s coming back |
| Q-W2 | Access-token lifetime | `expires_in` = **3600** for a fresh 3600 s token (5/5). It follows the documented `min(rule token lifetime, 2 × (JWT exp − now))`: an aged 600 s token with 132 s left got **261**. **The invariant holds:** the access-token lifetime (3600 s) is longer than the rotation interval (2881 s observed; at most ~2970 s), so re-exchanging on each new `iat` never leaves judge without a valid token |
| **Q-W3 (a)** | The same file exchanged twice | First exchange 200. Second, 56 s later: **401** `authentication_error` "Authentication failed" (`req_011CfQ39tCfE955xhrid6FLX`). The response is opaque by design: the deny reason (`jti_reused`) appears only in the Console's authentication history |
| **Q-W3 (b)** | In-place restart (`kubectl exec … -- kill 1`) | `restartCount` 0 → **1**: same pod uid, new container id, exit code 1 (the trap). The token file's `jti` and `iat` are **unchanged**, so the boot exchange gets **401** again (`req_011CfQ3CmAaV2YgZinWeGxtG`). The lock-out ends only when the kubelet rotates the file: here it ran from the restart at 10:57:29Z to the rotation at 11:39:16Z (~42 min). The same restarted container then exchanged **200** (`req_011CfQ6NsZfyMLQKH19KfjRt`) |
| **Q-W3 (c)** | Pod deleted and recreated | New pod uid, new token, fresh `jti` → **200** with `expires_in` 3600 (`req_011CfQ6SFg3FsMcDzV1sfbPp`) |
| **Q-W4** | Exchange + first Messages call, 5 runs, a fresh `jti` each | Exchange **p50 0.332 s, max 0.473 s**. First `POST /v1/messages` (16 in / 4 out tokens): **p50 1.613 s, max 1.824 s**. Together: **p50 1.926 s, max 2.297 s**, well under 5 s. Measured from the owner's Mac through multipass NAT, not from the VPS |
| Q-W4 | Workspace pin | `anthropic-workspace-id: wrkspc_017ZKXnJ1a9BxwcEm7e9rhGi` on **5/5** calls, and `anthropic-organization-id` = the org UUID. The exchange response also carries `workspace_id` (the same value), so judge can pin before its first call |
| Scope | Files and Batches with the access token | **200 / 200** for `GET /v1/files` and `GET /v1/messages/batches` (list calls, status only). Not 403, because the rule's scope is `workspace:developer` (finding 1 below) |
| Side | Audience binding: the projected token against the **throwaway** API server's `/api`, from inside `wif-probe-600` | **401**. Control: a default-audience TokenRequest token for the same SA → 200. A leaked judge token can't be used against the cluster |
| Side | Rule matchers: TokenRequest tokens exchanged from the VM | A default-audience token (`aud` = `["https://kubernetes.default.svc.cluster.local","k3s"]`) → **401**. Another SA's subject → **401**. `workspace_id: default` → **401**: the service account's Default-workspace membership doesn't widen the rule. The same token, then sent with the rule's workspace → **200**, so a refused attempt doesn't burn the `jti`. A fresh control → 200 |
| Side | JWKS stability (`/openid/v1/jwks`) | **Unchanged** (same kid, same JWKS hash, `service.key` untouched) after `systemctl restart k3s`, and again after `k3s certificate rotate` + restart. That command rotates every component certificate but not the service-account key. An exchange after both still returned 200 with the originally pasted inline JWKS. Only `k3s certificate rotate-ca` with a new `service.key` (per the k3s docs; not run here) or a node rebuilt without the old `/var/lib/rancher/k3s/server/tls/service.key` changes it |

**Exchange request shape (for m4-01; field names only)**

- `POST https://api.anthropic.com/v1/oauth/token` with `content-type: application/json`. No `anthropic-version` or `anthropic-beta` header was needed.
- Body (the RFC 7523 `jwt-bearer` grant):
  - `grant_type` = `urn:ietf:params:oauth:grant-type:jwt-bearer`;
  - `assertion`: the projected JWT verbatim, trailing newline trimmed, at most 16 KiB;
  - `federation_rule_id` (`fdrl_…`), `organization_id` (UUID), `service_account_id` (`svac_…`);
  - `workspace_id` (`wrkspc_…`): optional for a one-workspace rule, but send it.
- **200:** `access_token` (`sk-ant-oat01-…`), `token_type` `Bearer`, `expires_in`, `scope`, `workspace_id`. It also carries two undocumented fields, `next_challenge` (an empty string) and `next_challenge_expires_in` (0), which belong to another grant, so decode leniently and ignore unknown fields.
- **Every assertion denial is the same 401:** `{"type":"error","error":{"type":"authentication_error","message":"Authentication failed"},"request_id":…}`, whether the cause is `jti` reuse, audience, subject or workspace. A malformed request is a 400 `invalid_request_error` (per the WIF reference; not exercised here).
- The access token shares the `sk-ant-oat01-` prefix with other Anthropic OAuth tokens. Add that prefix to judge's log canary and redaction patterns.
- Then `POST /v1/messages` with `authorization: Bearer <access_token>` and `anthropic-version: 2023-06-01` (no `x-api-key`, no beta header).

**Decision: WIF GO, with `check_jti=false` on the one-rule issuer.** This is the pre-decided path: (a) rejected the reuse and (b) re-presented a used `jti`. Why production must set it:
- **An in-place restart locks judge out.** A container restart (OOM kill, crash, liveness failure) keeps the pod's token file. judge's boot exchange then re-presents a used `jti` and is refused until the kubelet's next rotation: up to ~48–50 min at 3600 s (~42 min in this run).
- **The refusal can't be told apart.** It is an opaque 401, the same as JWKS drift or an archived rule. m4-01's planned `jti_reused` branch can't be built from the response, so every restart would open the breaker as an auth failure.
- **Retries hit it too.** A retry after a lost 200 (a timeout after the server accepted) would re-present a used `jti` (inferred). A *refused* attempt doesn't burn the `jti` (the matcher row above).
- **The weakening is bounded.** A stolen projected token is replayable only until its `exp` (≤ 1 h), only against this rule (exact subject, audience, one workspace), and any access token it mints lives ≤ 1 h anyway. Production has one rule on the issuer, so no other rule loses replay protection.

Alternatives considered and not taken (mi-12 confirms):

| Alternative | Why not |
|---|---|
| Cache the access token in a memory-backed `emptyDir` | It survives container restarts, but it puts a live bearer token on a volume, adds cache-invalidation code, and doesn't cover the lost-response retry |
| `expirationSeconds: 600` with `check_jti` on | Shortens the lock-out to about 9 minutes, but every restart still opens the breaker, and judge exchanges about 7 times an hour |
| judge mints fresh tokens through the TokenRequest API | Needs an API-server credential and RBAC for judge's service account; automount stays off |

The confirming re-run of (a) and (b) under `check_jti=false` is handed to mi-12, because it is a Console change and the spike never touched the Console. Expected: (a) 200 and (b) 200.

**Values mi-12 sets**

| Where | Setting |
|---|---|
| Pod | Projected `serviceAccountToken`: `audience: https://api.anthropic.com`, **`expirationSeconds: 3600`**, `path: token`, mounted read-only at `/var/run/secrets/anthropic.com`; `automountServiceAccountToken: false`. Keep `expirationSeconds` at or below the issuer's max JWT lifetime (1 h by default); `exp − iat` is exactly 3600 |
| Issuer | `issuer_url` = `https://kubernetes.default.svc.cluster.local`. JWKS **inline**: paste the **`keys` array**, because the Console field takes the array, not the `{"keys": […]}` wrapper. **`check_jti` off**: untick "Enforce single-use tokens (JTI replay protection)", which defaults on. Max JWT lifetime 1 h (the default) |
| Rule | `subject_prefix` exactly `system:serviceaccount:xlearn:xlearn-judge` (no `*`); `audience` = `https://api.anthropic.com`; the one workspace `xlearn-platform-prod`; **token lifetime 1 h, set explicitly** (the Console's options are 1 m, 5 m, 10 m (default), 1 h, 24 h or custom); scope per finding 1 below |
| judge | `LLM_ANTHROPIC_ORG_ID`, `…_WORKSPACE_ID`, `…_SERVICE_ACCOUNT_ID` and `…_FEDERATION_RULE_ID` as non-secret values. `ANTHROPIC_API_KEY` is never set |

**Re-exchange rule (m4-01):**
- **Boot:** exchange lazily on the first platform call. This works even if the file was already exchanged before the restart, because `check_jti=false` allows it.
- **Afterwards:** re-exchange only when the file's `iat` is newer than the one last exchanged. That happens every ~48–50 min (and after every k3s restart), 10 or more minutes before the cached 3600 s token expires.
- **Errors:** retry transport errors and 5xx a bounded number of times. Treat every 401/403 as `llm.ErrAuth` (the breaker); the reason is in the Console's authentication history.
- **Pin:** refuse the token unless the exchange response's `workspace_id` equals `LLM_ANTHROPIC_WORKSPACE_ID`, then check `anthropic-workspace-id` on every response.

**Lifetime invariant:** the access-token lifetime is **longer than** the rotation interval. A fresh file gets `min(3600, 2 × 3600)` = 3600 s; rotation comes at 80 % of 3600 s plus at most one kubelet sync (2881 s observed; at most ~2970 s). A re-presented aged file keeps the invariant too: `min(3600, 2 × (3600 − a)) > 2970 − a` for every age `a` before the rotation point.

**Findings that amend ADR-0031** (proposed; mi-12 folds them in when it accepts the ADR, and this sprint doesn't edit it):
1. **Scope (§2, Consequences).**
   - **What the Console offers:** the rule form has only `workspace:developer` and `org:admin`, no `workspace:inference` (owner, 2026-09-25).
   - **What the docs say:** the WIF reference and the Admin API reference (fetched 2026-09-25) still list `workspace:inference`, settable only by an `org:admin` OAuth caller through `POST /v1/organizations/federation_rules`. Untested here: the spike holds no admin credential.
   - **What was measured:** under `workspace:developer` the token reached Files and Batches (200 / 200).
   - **Proposal:** use **`workspace:developer`**, the least privilege the Console offers. mi-12 may instead create the rule through the Admin API with `workspace:inference`, and keeps it only if a Files call then returns 403.
   - **Either way, the credential no longer guarantees "Messages only".** judge's request builder is the enforcement: Messages only; no `tools`, `mcp_servers` or `container`; the golden request-shape test and the CI lint. Drop "The inference-only scope blocks Files, Batch and agents" from Consequences. t5 §3's `RetentionPolicy` line "`workspace:inference` enforces most of this at the credential" no longer holds.
   - The extra reach is small: judge stores nothing in the workspace, and the workspace limit caps spend.
2. **Credential (§2).** WIF GO. `check_jti=false` on the one-rule issuer; rule token lifetime 1 h, set explicitly; `expirationSeconds: 3600`; plus the re-exchange rule and the invariant above.
3. **Workspace pinning (§2).** Check the exchange response's `workspace_id` before first use, as well as `anthropic-workspace-id` on each response; both matched 5/5. The service account is always a member of the Default workspace (the Console locks it). The rule refuses `workspace_id: default` (401), so that membership doesn't widen the credential.
4. **Maintenance (§2).** Paste the JWKS inline as the `keys` array. It survives k3s restarts and `k3s certificate rotate`. It changes only with `k3s certificate rotate-ca` carrying a new `service.key`, or on a node rebuilt without the old `service.key`. After either, re-paste it: exchanges 401 until then (breaker → manual grading). `host-verify --cluster`'s kid check shows the drift (mi-12 task 6). A k3s restart does re-issue every projected token, but that is only a newer `iat` for judge to re-exchange.
5. **Provider-side controls (§2, §5, §8).** The org's monthly spend limit is **$5** today (owner, 2026-09-25). No workspace can spend past it, so before mi-12 the org limit must rise to at least the sum of the workspace limits it holds: $15 for `xlearn-platform-prod` (dogfood) plus about $50 for `xlearn-calib` during M4 bring-up, and enough for $100 at the v3 opening. This is an mi-12 before-launch item.
6. **Org.** WIF works in the owner's current org. The dedicated-org question stays with mi-12 (t5 §3's "Org" row); if a new org is created, re-check WIF there.

**Throwaway resources:** the namespace `wif-spike` and the VM `xlearn-wif` were deleted on 2026-09-25 (`multipass delete --purge`; `multipass list` shows no instances, so `xl-spike` is gone too). The Console objects (rule, issuer, service account, workspace) are left for the owner's clean-up (status.md → Open owner items). The issuer is inert once the VM is purged, because its signing key went with the VM. It must still be deleted before mi-12 creates the production issuer, which uses the same `issuer_url`.

Sources, fetched 2026-09-25: [Workload Identity Federation](https://platform.claude.com/docs/en/manage-claude/workload-identity-federation) (exchange flow, token lifetime and refresh, `jti` single use); [WIF reference](https://platform.claude.com/docs/en/manage-claude/wif-reference) (request and response fields, OAuth scopes, JWT verification, errors); [WIF with Kubernetes](https://platform.claude.com/docs/en/manage-claude/wif-providers/kubernetes) (inline `keys` array, rule shape); the Admin API references for [federation issuers](https://platform.claude.com/docs/en/api/admin/federation_issuers) (`check_jti`, `max_jwt_lifetime_seconds`) and [federation rules](https://platform.claude.com/docs/en/api/admin/federation_rules) (`oauth_scope` accepts `workspace:inference` from OAuth callers); [k3s `certificate`](https://docs.k3s.io/cli/certificate) (`rotate` covers client and server certificates only; the service-account issuer key rotates with `rotate-ca`).

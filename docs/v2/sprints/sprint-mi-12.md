# Sprint mi-12 — M4 gates: judge 443 egress, LLM secret, provider runbook (MI-14) + ADR-0031 → Accepted

> **Milestone:** MI — rollout step **MI-14** (the M4 gates; sprint id `mi-12` ≠ rollout step MI-12, the runner) · **Track:** infra · **Order:** 56
> **Prereqs:** [m3-07](sprint-m3-07.md) (judge on prod, born default-deny egress) · [spk-03](sprint-spk-03.md) (WIF spike result)
> **Unblocks:** [m4-01](sprint-m4-01.md) (M4 entry: "MI-14 done") · also satisfies the gates in [m4-02](sprint-m4-02.md) (judge → identity :8081 confirmed — born in m3-07, reused by M4) and [m4-07](sprint-m4-07.md) (provider runbook executed)
> **Release action:** **infra PR(s) only** (three `../infra` PRs) — plus an xlearn **docs** PR (ADR-0031, runbook, status) and an `xlearn-evalpack` template PR. **No tag**; nothing here ships in an xlearn image.
> **Calendar:** December 2026, before M4 · owner event **`ev-provider-runbook`** (Anthropic Console, ~45 min), done **before launch** (D40) · prepares **`ev-acceptance-set`** (owner labels ≥ 70 examples, 5–10 h, before m4-01)
> **Execute with:** [`../prompts/prompt-mi-12.md`](../prompts/prompt-mi-12.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | Accept ADR-0031 (fold the spk-03 result and the T7 amendments) | X | ⬜ |
| 2 | Provider runbook `docs/v2/runbooks/platform-ai-provider.md` | X | ⬜ |
| 3 | Provider Console setup (`ev-provider-runbook`) — before launch | O | ⬜ |
| 4 | judge egress: TCP 443 to non-cluster (own PR); confirm m3-07's `xlearn-identity` :8081 rule | I | ⬜ |
| 5 | `xlearn-judge-llm` secret + LLM values + projected SA token | I | ⬜ |
| 6 | `host-verify --cluster`: WIF JWKS drift check | H | ⬜ |
| 7 | Analyzer acceptance-set template | E | ⬜ |
| 8 | Verify + record | H + X | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row, MI table row MI-14, flag inventory, events).
> Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] **MI-13 done** ([m3-07](sprint-m3-07.md)): `xlearn-judge` Ready on prod, born with default-deny egress (DNS, PG, NATS, runner, `xlearn-gateway` :8080 for JWKS, `xlearn-identity` :8081 for the erase re-verify)
- [ ] **WIF spike reported** ([spk-03](sprint-spk-03.md)): WIF **GO** with the `check_jti` decision and the measured refresh time, **or** the fallback (single-workspace key, 90-day expiry) chosen; the throwaway workspace and rule deleted
- [ ] **MI-1 done** (`ev-mi1`): 2FA on the owner's Anthropic account (rollout §2: MI-1 unblocks the M4 provider accounts). This is a before-launch owner item, attested by launching the prompt
- [ ] **Chart 0.3.0 egress template** in use by `apps/xlearn-judge.yaml` ([mi-01](sprint-mi-01.md), set by m3-07)
- [ ] **Parallel sessions:** no open peer PR edits `../infra/apps/xlearn-judge.yaml` (e.g. [mi-11](sprint-mi-11.md)'s MI-15 egress, an evalpack bump) or `docs/adr/0031-*` (`gh pr list` in both repos, `git worktree list`, ListAgents) — else sequence with it

## Goal

Open exactly what platform AI needs before M4 and nothing more: judge's **TCP 443** egress to non-cluster
addresses (its **`xlearn-identity` :8081** rule was born in [m3-07](sprint-m3-07.md) for the erase re-verify and
carries M4's `/internal/accounts/{id}` read unchanged — confirmed here, not added), the **`xlearn-judge-llm`** SOPS secret, the non-secret
LLM values (with the kill switch `LLM_PLATFORM_ENABLED=false`), and the **projected service-account token**
that Workload Identity Federation exchanges; wire in the owner's before-launch Console setup and write it up
as the **provider runbook** at D25's dogfood limits ($15 provider / $12 app, D35 owner-only v2); ship the
**acceptance-set template** the
owner labels before m4-01; and move **[ADR-0031](../../adr/0031-platform-ai-and-two-tier-keys.md) to
Accepted** with the [spk-03](sprint-spk-03.md) result. No platform-AI code lands here (that is M4).

## Scope

**In**
- ADR-0031 → **Accepted** (WIF GO or the break-glass-key fallback), with the T7 amendments (§8: D34 no push
  channel, D35 v3 opening, D25 dogfood limits) folded into its body (unless the BP2 sign-off already did); ADR index row.
- `../infra/apps/xlearn-judge.yaml` egress: **+ TCP 443** to `0.0.0.0/0` except the cluster, node and
  link-local ranges — the one new rule. `xlearn-identity` :8081 already exists since [m3-07](sprint-m3-07.md)
  (erase re-verify) and serves m4-02's `/internal/accounts/{id}` too: verified, added only if missing. Its own PR, merged
  before `v1.16.0` ([ADR-0035](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#2-nats-auth-nkey-users-fine-acls-server-first) standing rule).
- `../infra/apps/secrets/xlearn-judge-llm.enc.yaml` (`LLM_ENDUSER_SALT`; the break-glass key only on the
  fallback) via `envFrom`; non-secret values; the projected token at `/var/run/secrets/anthropic.com/token`.
- `docs/v2/runbooks/platform-ai-provider.md` (new): the `xlearn-platform-prod` workspace, limits, alerts,
  billing, WIF issuer/rule/service account, the `xlearn-calib` workspace, break-glass, kill switches,
  maintenance, the v3 raise.
- A read-only **JWKS drift check** in `host-verify --cluster` (the one ops tool, D34).
- `xlearn-evalpack`: `acceptance/{dev,test}/` + label schema + labelling README.

**Out**
- Any push ping (judge → healthchecks.io `judge-ai`), Flux Alert or opscheck — **dropped by D34**, no sprint takes it.
- `internal/platform/llm`, the WIF token exchange, `RetentionPolicy`, the request-shape guards → [m4-01](sprint-m4-01.md).
- `internal/judge/ai` (Scorer, ledger, llm lane, caps, breaker), identity `/internal/accounts/{id}` fields → [m4-02](sprint-m4-02.md).
- Running the acceptance set, sizing caps, `LLM_PLATFORM_ENABLED=true` for the cohort → [m4-07](sprint-m4-07.md).
- Labelling the ≥ 70 examples → owner event `ev-acceptance-set` (before [m4-01](sprint-m4-01.md)).
- `xlearn` egress for the other services (MI-15) → [mi-11](sprint-mi-11.md).
- Raising the limits to $100 / $80 → the v3 opening ([rollout §11](../rollout-plan.md#11-opening-gates-v3)), not v2.
- An OpenAI platform credential (only once a challenger passes calibration) and the credential-injecting
  `xlearn-llm-egress` proxy (trigger-based, [t5 §3](../research/t5-platform-ai.md#3-where-the-platform-key-lives-and-secrets)) — not v2.0.

## Tasks

### 1 · Accept ADR-0031 [X]

Inputs: the [spk-03](sprint-spk-03.md) result, recorded as `## 15. WIF spike result` in [t5](../research/t5-platform-ai.md) (appended by spk-03); [ADR-0031 §8](../../adr/0031-platform-ai-and-two-tier-keys.md#8-amended-by-t7-2026-09-24) (if still present);
[ADR-0035 §6](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#6-amendments) (the ADR-0031 §5 row);
[ADR-0033](../../adr/0033-invite-only-admission-and-owner-admin.md) (D35: "before learners" = the v3 opening);
[t5 §3](../research/t5-platform-ai.md#3-where-the-platform-key-lives-and-secrets).

Edit `docs/adr/0031-platform-ai-and-two-tier-keys.md`. **First check whether the BP2 ADR sign-off (2026-09-24) already
folded §8 into the body** (0031 stayed Proposed, but 0033/0035's amendments may have been folded then): if §8 is gone or
already a dated history, skip the §4/§5/Consequences/§8 bullets below and only add the spike result, the egress list, the
Maintenance note and the Status line; otherwise fold as described.
- **Status** → `Accepted (2026-12-DD). WIF spike (spk-03, <date>): GO, check_jti=<true|false>` — or
  `… fallback: a single-workspace service-account key (90-day expiry) is the primary credential`.
- **§2 Credential** gains a *Spike result* list: is `jti` present in k3s projected tokens; the rotation
  cadence (the kubelet refreshes at ≈ 80% of the 3600 s TTL, so ≈ every 48 min); does an in-place container
  restart re-present a used `jti` (`jti_reused`) and therefore `check_jti=false` on this one-rule issuer;
  the measured exchange + first Messages call time. On the fallback: the key is created only in the
  `xlearn-platform-prod` workspace, 90-day provider-side expiry, SOPS-held, expiry date as a manual check in
  status.md (D34), and WIF is retried at the next k3s minor.
- **§2 Egress** lists judge's full allow-list: kube-dns :53, PG :5432, NATS :4222, the runner, `xlearn-gateway`
  :8080 (JWKS, since m3-07), **`xlearn-identity` :8081** (since M3 — m3-07, the erase re-verify — reused by M4's
  `/internal/accounts/{id}` read, m4-02), and **TCP 443** to non-cluster addresses (the only M4 addition, this sprint).
- **§2 Maintenance:** inline JWKS has no auto-refresh; re-paste at a k3s SA-signing-key rotation; the
  `host-verify --cluster` kid check (task 6) is how the owner notices (no alert, D34).
- **§4 / §5:** fold §8 into the body — the §5 table reads provider **$15** / app **$12** in owner-only v2 (D25
  dogfood, D35) and $100 / $80 from the v3 opening; the breaker row says in-app badge + the provider's own
  Console alerts and hard limit, **no push channel** (D34); "before M4 opens to learners" → "before the v3 opening".
- **Consequences:** "alerts at `retire_not_before`" → "a manual `retire_not_before` check before each model change (D34)".
- **§8** shrinks to a dated amendment history (three lines, pointing to 0033/0035) now that the body carries the text.
- `docs/adr/README.md`: row 0031 → `Accepted (2026-12-DD; WIF GO | fallback key)`.

### 2 · Provider runbook [X]

New `docs/v2/runbooks/platform-ai-provider.md` (owner-executable, one screen per section; it records the
before-launch setup of task 3 with its date and non-secret IDs, and carries the maintenance, break-glass and
v3-raise procedures; source:
[t5 §3 provider-side controls](../research/t5-platform-ai.md#3-where-the-platform-key-lives-and-secrets),
[t5 §6](../research/t5-platform-ai.md#6-budgets-metering-and-abuse-controls), [ADR-0031 §2/§5](../../adr/0031-platform-ai-and-two-tier-keys.md#2-credential-and-egress)):

| Section | Content |
|---|---|
| Org | A **dedicated xLearn org** if the current org is shared with Claude Code or other API use (tier cap, credits and abuse enforcement are org-wide). A new org may start in the Evaluation tier: check its RPM/OTPM allow the llm lane (cap 8, ≤ 8 calls per evaluation). 2FA on. |
| Workspace | `xlearn-platform-prod` (the Default workspace can't carry limits). Monthly **hard limit $15**; Console spend alerts at **50% and 80%**; the app cap **$12** lives in judge (m4-02). |
| Rate limits | Low per-model RPM / OTPM on `claude-sonnet-5` and Opus 5.5 (start low, *inferred*: ~20 RPM, ~40k OTPM; m4-07 re-sizes after the acceptance run). |
| Billing | Prepaid credits ≈ 1–2 × the ceiling ($15–30); **auto-reload OFF** (it has no monthly cap); credits expire after 1 year. |
| WIF | Service account `xlearn-judge`, member of `xlearn-platform-prod` only. Issuer = the cluster `iss` (`https://kubernetes.default.svc.cluster.local`, read live: `ssh vps 'sudo k3s kubectl get --raw /.well-known/openid-configuration'`), `jwks.type: inline` = the output of `ssh vps 'sudo k3s kubectl get --raw /openid/v1/jwks'` (public keys). Rule: `subject_prefix: system:serviceaccount:xlearn:xlearn-judge` (no `*`), `audience: https://api.anthropic.com`, `oauth_scope: workspace:inference`, `token_lifetime_seconds: 3600`, one workspace, `check_jti` per spk-03. |
| Hand-over | The non-secret IDs the infra values need: org, workspace, service account, federation rule; the `LLM_KEY_LABEL` (`wif-xlearn-platform-prod`); the JWKS `kid`s pasted (→ `hack/expected-jwks-kids.txt` and its embedded copy in `host-verify.sh`, task 6). |
| `xlearn-calib` | A separate workspace for the analyzer acceptance run (~$10–30, m4-07) and later calibration (~$20–60 per rubric): its own hard limit of **~$50/month during M4 bring-up** ([t5 §5](../research/t5-platform-ai.md#5-cost-model)) with its own Console alerts — separate from the $15 prod limit — and a personal key with a 7–30-day expiry for owner-machine runs, **never in the cluster**. |
| Break-glass | Only on the spk-03 fallback or a WIF outage: a single-workspace service-account key, **90-day expiry**; the **owner** writes it into the SOPS file (`sops ../infra/apps/secrets/xlearn-judge-llm.enc.yaml`) — no agent handles it; expiry date into status.md (manual check); bump `xlearn.dev/llm-rev`. Never name it `ANTHROPIC_API_KEY`. |
| Kill switches (fastest first) | Console: disable the rule/key or drop the limit → 401/400 → breaker → manual · `judge admin breaker open` (`kubectl exec`) · `LLM_PLATFORM_ENABLED=false` (infra PR) · per-account `ai_disabled` (owner). |
| Maintenance | JWKS re-paste when `host-verify --cluster` WARNs on kid drift (check after every k3s upgrade and in the monthly window); monthly ledger vs Console (procedure added by m4-07); `retire_not_before` before any model change; no Admin keys in the cluster; ZDR requested opportunistically (sales). |
| v3 opening | Raise to $100 / $80, re-sized with `SEAT_CAP` ([rollout §11](../rollout-plan.md#11-opening-gates-v3)). |

### 3 · Provider Console setup [O, before launch]

Calendar event `ev-provider-runbook`, done by the owner **before launching this sprint's prompt** (~45 min;
provider-console work is owner-only, D40). The checklist is the prompt's *Before you launch (owner)* block,
built from the task 2 table rows:
- the org, with 2FA on;
- `xlearn-platform-prod` with the $15 hard limit, alerts at 50%/80%, low RPM/OTPM, and prepaid credits with
  auto-reload off;
- the service account `xlearn-judge`, the inline-JWKS issuer and the rule;
- `xlearn-calib` with its own limit and alerts.

The owner reads the issuer URL and the JWKS JSON (public keys) live, with the two read-only commands in the
task 2 WIF row. At launch they hand over the non-secret IDs (org, workspace, service account, federation
rule), the `LLM_KEY_LABEL` and the pasted `kid`s. Launching attests the setup is done: the session can't see
the Console, so it records the date and the IDs in the runbook and status.md.

**If the setup turns out to be missing at launch:** tasks 1, 2, 4, 6 and 7 still land. Task 5 isn't opened,
so no draft PR is left behind. Tasks 3 and 5 go ⛔ "waiting on ev-provider-runbook" in status.md, and
[m4-01](sprint-m4-01.md) stays gated. A re-run of this prompt after the setup lands task 5.

On the spk-03 fallback, the owner writes the break-glass key into the SOPS file once task 5 has created it;
no agent handles it. Record that as a pending owner item in status.md. It's needed before
[m4-07](sprint-m4-07.md) enables the cohort, and nothing here waits for it.

### 4 · judge egress: TCP 443 (+ confirm identity :8081) [I]

One `../infra` PR, `feat(xlearn-judge): M4 egress — TCP 443 to non-cluster (MI-14)`, editing only
the NetworkPolicy values of `apps/xlearn-judge.yaml` (chart 0.3.0 egress template, same schema m3-07 used):

| Rule | To | Port |
|---|---|---|
| **new** | `ipBlock 0.0.0.0/0` **except** `10.0.0.0/8` (covers the pod `10.42.0.0/16` and service `10.43.0.0/16` CIDRs), `172.16.0.0/12`, `192.168.0.0/16`, `100.64.0.0/10`, `169.254.0.0/16` and the **node IP `/32`** (read live: `k3s kubectl get node -o wide`) — a superset of [t5 §3](../research/t5-platform-ai.md#3-where-the-platform-key-lives-and-secrets)'s list | TCP 443 |
| unchanged (verify) | pods `app.kubernetes.io/instance: xlearn-identity`, same namespace — born in [m3-07](sprint-m3-07.md) for the erase re-verify; m4-02's `/internal/accounts/{id}` read reuses it (identity's MI-5a ingress already admits `xlearn`) | TCP 8081 |
| unchanged | kube-dns :53, PG :5432, NATS :4222, the runner, `xlearn-gateway` :8080 | — |

- Read the live policy first (`k3s kubectl get networkpolicy -n xlearn xlearn-judge -o yaml`). The identity :8081 rule
  must already be there (m3-07). **Only if it is missing** (m3-07 drifted — judge's erase consumer would be dead-lettering)
  add it in this PR and say why in the description.
- PR description: the caller-matrix delta (443 is new with m4-01; judge → identity is unchanged — it exists since m3-07
  and gains m4-02's read with no policy change), the ADR-0035 §2 standing rule ("merged before `v1.16.0`"), and the before/after
  `k3s kubectl get networkpolicy -n xlearn xlearn-judge -o yaml` diff: **exactly one egress rule added (TCP 443)**.
- Merge early (any time before `v1.16.0`); fail-open revert = `git revert`.

### 5 · `xlearn-judge-llm` secret, LLM values, projected token [I]

One `../infra` PR; it needs the IDs from task 3's before-launch setup:
- **`apps/secrets/xlearn-judge-llm.enc.yaml`** — Secret `xlearn-judge-llm`, namespace `xlearn`, `stringData.LLM_ENDUSER_SALT`
  (32 random bytes, base64; generated and encrypted in one pipe, never echoed); `LLM_ANTHROPIC_API_KEY` **only**
  on the fallback, written by the owner. Covered by the existing `apps/secrets/.*\.enc\.ya?ml$` SOPS rule.
- **`apps/xlearn-judge.yaml`** values:
  - `envFrom` += `secretRef: {name: xlearn-judge-llm}` (convention: `apps/xlearn-coach.yaml`);
  - `env` += `LLM_PLATFORM_ENABLED: "false"` (permanent kill switch, [ADR-0034 §2](../../adr/0034-v2-release-labelling-gating-and-rollback.md#2-feature-gating-three-tiers-no-flag-service)),
    `LLM_ANTHROPIC_ORG_ID`, `LLM_ANTHROPIC_WORKSPACE_ID`, `LLM_ANTHROPIC_SERVICE_ACCOUNT_ID`,
    `LLM_ANTHROPIC_FEDERATION_RULE_ID`, `LLM_KEY_LABEL` (all non-secret);
  - the projected token (WIF path only; `automountServiceAccountToken` stays `false`):
    ```yaml
    extraVolumes:          # appended to m3-07's evalpack, nats-seed, runner-auth and scratch volumes — keep all four (Helm replaces lists)
      - name: anthropic-token
        projected:
          sources:
            - serviceAccountToken:
                audience: https://api.anthropic.com
                expirationSeconds: 3600
                path: token
    extraVolumeMounts:
      - name: anthropic-token
        mountPath: /var/run/secrets/anthropic.com   # the file m4-01 reads: …/token
        readOnly: true
    ```
  - `podAnnotations` += `xlearn.dev/llm-rev: "1"` (precedent `xlearn.dev/oauth-rev` in `apps/xlearn-identity.yaml`).
- The chart's `serviceAccount.create: true` names the SA `xlearn-judge` → subject `system:serviceaccount:xlearn:xlearn-judge`; confirm it matches the rule.
- Merging restarts judge once (no v1.15 code reads `LLM_*`); merge when no cohort evaluation is running.

### 6 · `host-verify --cluster`: WIF JWKS drift check [H]

A small `../infra` PR to `hack/host-verify.sh` (read-only, the MI-8 pattern, [ADR-0035 §3](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#3-no-alerting-in-v2-owner-d34) "cheap later additions"):
- **The expected kids travel with the script** ([mi-02](sprint-mi-02.md)'s convention for its TSVs): the owner runs it piped
  (`ssh vps 'bash -s -- --cluster' < hack/host-verify.sh`) or from `/root/host-verify.sh`, and neither has a file beside it.
  So add `hack/expected-jwks-kids.txt` (the kids pasted into the Console in task 3, one per line) **and** embed it in the
  script between `# >>> expected-jwks-kids.txt` / `# <<< expected-jwks-kids.txt` markers; `hack/host-lint.sh` fails when the
  embedded copy differs from the file (extend its existing embedded-TSV check).
- `--cluster` compares the `kid` set of the live JWKS (mi-02's `kc_raw /openid/v1/jwks` + `jq`) with the embedded block →
  **PASS** on a match, **WARN** `WIF JWKS drift — re-paste per docs/v2/runbooks/platform-ai-provider.md` on a difference;
  INFO-skip **only when the embedded block is empty** (the fallback path).
- Prove it fires: pipe a scratch copy with one embedded kid altered (`sed … hack/host-verify.sh | ssh vps 'bash -s -- --cluster'`)
  → the WARN; nothing is written on the node. Paste both runs in the PR.
- Keeps `hack/host-lint.sh`'s read-only grep and shellcheck green. After merge, refresh the node copy the sanctioned way
  (`scp hack/host-verify.sh vps:/root/`, [mi-02](sprint-mi-02.md) task 7), so the owner's `ssh vps 'bash /root/host-verify.sh --cluster'` runs it.
  That node write is specified here, so launching the prompt pre-approves it (D40). Not an alert, timer or CronJob (D34).

### 7 · Analyzer acceptance-set template [E]

A PR in the private `xlearn-evalpack` repo, checked out at `../xlearn-evalpack` (no evalpack tag; the template changes no pack).
The acceptance artefacts are solutions to pack items: **never copy anything from `../xlearn-evalpack` into the public xlearn
repo** (the AGENT.md never-copy rule, [m3-01](sprint-m3-01.md)) — not into the PR, status.md or this file.
- `acceptance/README.md` — how to label **≥ 70** examples: **test ≥ 40** (15 *optimal*, where "no pointer" is
  correct; 15 *passing but suboptimal*; 10 *failing* with a known category) and **dev ≥ 30** (same mix; used for
  prompt/effort tuning and the D26 "materially different" fingerprint threshold). The gate thresholds m4-07
  applies ([t5 §4](../research/t5-platform-ai.md#4-provider-and-model-choice-per-task)): category accuracy ≥ 80%,
  false-pointer rate ≤ 10%, line ranges 100% valid, schema-valid after ≤ 1 retry 100%, truncation ≤ 2%,
  p95 $/analysis recorded, bootstrap lower bound ≥ 70%. The **test split is frozen**: each configuration touches
  it once and `configs_tried` is logged ([t5 §7](../research/t5-platform-ai.md#7-quality-calibration-regression-and-injection-defences)).
  Artefacts are owner-written solutions to **stamped pilot items** (Go, C++, Python) — no tester or learner data.
- `acceptance/schema/label.schema.json` + one YAML per example:
  `id`, `split` (dev|test), `item` (course/item id), `language`, `artefact` (path under `acceptance/<split>/artefacts/`),
  `verdict` (pass|fail, the deterministic judge verdict), `kind` (optimal|suboptimal|failing),
  `expected.category` (one of the 8 mistake categories, [PRD R-MJ2](../../prd/xlearn-prd.md#64-mistake-journal); `null` on passes),
  `expected.concepts[]` (the item's pack concept tags), `expected.pointers[]` (`[]` = "no pointer is correct"; else `{lines: "12-18", class}`),
  `pair_of` (dev only, near-duplicate for the fingerprint threshold), `notes` (never sent to the model), `labelled_at`.
- `acceptance/dev/`, `acceptance/test/` (with `configs_tried.log`), one `example: true` file excluded from counts.
- **Excluded from the pack image** (`.dockerignore` / build context) — judge never mounts the acceptance set;
  a CI step in evalpack validates every label against `label.schema.json` with a JSON-schema checker (e.g. `check-jsonschema`;
  [m3-02](sprint-m3-02.md)'s `packspec` validator derives case-input checks from `constraints[]` and is not a JSON-schema validator).

### 8 · Verify + record [H + X]

- `ssh vps 'bash /root/host-verify.sh --cluster'` (after the task 6 refresh; before it, `ssh vps 'bash -s -- --cluster' < ../infra/hack/host-verify.sh`)
  green, with the kid check reporting **PASS** — not INFO-skip — on the WIF path (memory sum unchanged: no new pod).
- judge Ready after both infra PRs; a cohort **judged submit** still grades end to end (JWKS, PG, NATS, runner intact);
  `k3s kubectl get pod -n xlearn -l app.kubernetes.io/instance=xlearn-judge -o jsonpath='{.items[0].spec.volumes[*].name}'`
  lists `anthropic-token`; `…spec.automountServiceAccountToken` is `false`; no secret value printed anywhere.
- Functional reachability of `api.anthropic.com:443` **from the judge pod** can't be exercised before M4 code exists (no
  client in the image; `kubectl debug` is forbidden). It is proven by `judge admin llm-smoke` from the pod —
  [m4-01](sprint-m4-01.md) task 10, run as [m4-07](sprint-m4-07.md)'s first step after `v1.16.0` rolls out. A dial timeout there
  reopens task 4. `xlearn-identity:8081` is already exercised by judge's erase re-verify (m3-07's smoke).
- `docs/v2/status.md`: MI table MI-14 ✅ (PR numbers); flag inventory — `LLM_PLATFORM_ENABLED` present, `false`
  (kill switch; m4-07 flips it); events — `ev-provider-runbook` ✅ date, `ev-acceptance-set` ⬜ (template ready);
  manual checks — JWKS kids + date (and the break-glass key expiry on the fallback); accepted-risk register —
  inline JWKS rotation, DNS tunnelling / own-output exfiltration (accepted, owner-only v2); decisions log — ADR-0031
  Accepted (WIF GO/fallback, `check_jti`), "M4 adds only TCP 443 (judge → identity :8081 exists since m3-07)".
- [rollout §12](../rollout-plan.md#12-downstream-constraints-for-the-build-plan-session)'s "M4 adds only 443" stands
  (m3-07 moved judge → identity forward to M3) — no rollout edit.

## Acceptance criteria

- [ ] [ADR-0031](../../adr/0031-platform-ai-and-two-tier-keys.md) **Accepted** with the spk-03 result and the T7 amendments folded into its body; ADR index row updated
- [ ] judge's NetworkPolicy adds exactly **one** egress rule — **TCP 443 to non-cluster addresses** — nothing else new; m3-07's **`xlearn-identity` :8081** rule present (before/after diff in the PR); judge Ready and a cohort judged submit still grades. Policy verified here; functional 443 reachability is proven by `judge admin llm-smoke` from the pod ([m4-01](sprint-m4-01.md) task 10, m4-07 step 1)
- [ ] `xlearn-judge-llm` loaded via `envFrom`; `LLM_PLATFORM_ENABLED=false`; the projected token (audience `https://api.anthropic.com`, 3600 s) mounted at `/var/run/secrets/anthropic.com/token` with automount still off
- [ ] Runbook merged; Console limit **$15**, alerts **50%/80%**, auto-reload **off**, WIF issuer/rule/service account set (the owner's before-launch setup, attested at launch and recorded with its date and IDs)
- [ ] `host-verify --cluster` green, with the JWKS kid check reporting **PASS** from the embedded kid block (not INFO-skip; WIF path), a forced-drift run showing the WARN, and `/root/host-verify.sh` refreshed
- [ ] Acceptance-set template merged in `xlearn-evalpack` (schema, README, dev/test dirs, CI validation, excluded from the pack image)

## Release

**Infra PR(s) only — no tag.** Three `../infra` PRs, each its own task and never folded into a tag
([rollout §2.2](../rollout-plan.md#22-operating-rules-every-mi-step-and-every-tag)): (A) egress — merge first,
any time before `v1.16.0`; (B) secret + values + token — with the IDs from the owner's before-launch Console
setup; (C) the host-verify check. Plus one xlearn **docs** PR (ADR-0031 accepted in-session, runbook,
status.md) and one `xlearn-evalpack` PR (template; no `evalpack v1.x` tag). Launching the prompt is the
owner's approval for all of it (D40). Flux applies A and B within minutes; B restarts judge once, still AI-dark.
The M4 code that uses these gates ships in **`v1.16.0`** ([m4-07](sprint-m4-07.md)), which flips
`LLM_PLATFORM_ENABLED=true` for the cohort in its own infra PR. Rollback: `git revert` of A or B (fail-open for A).

## Definition of Done

All infra PRs merged via GitOps (no hand `kubectl`) and reconciled · judge Ready · `host-verify --cluster` green ·
docs PR and evalpack PR merged with CI green · acceptance criteria met (the Console setup attested at launch) ·
statuses updated (this file + [`../status.md`](../status.md)) · ADR-0031 Accepted · local `main` synced in xlearn, `../infra` and `xlearn-evalpack`.

## Risks / watch-outs

- **Inline JWKS has no auto-refresh.** A k3s SA-signing-key rotation breaks every exchange (401 → breaker → manual
  grading). Mitigation: the task 6 kid check after every k3s upgrade and in the monthly window, and the runbook re-paste.
- **DNS tunnelling through CoreDNS and exfiltration through judge's own output** remain possible with 443 open.
  Accepted for owner-only v2 ([t5 §3](../research/t5-platform-ai.md#3-where-the-platform-key-lives-and-secrets));
  the `xlearn-llm-egress` proxy is the trigger-based answer.
- **A missing secret crash-loops judge** (`CreateContainerConfigError`): the Secret and its `envFrom` go in the same
  PR; check it decrypts **without printing** (`sops -d apps/secrets/xlearn-judge-llm.enc.yaml >/dev/null && echo decrypts`)
  and read only its structure from the encrypted file (`yq '.stringData | keys'` — values stay ciphertext).
- **Shared org with Claude Code:** spend, tier and abuse enforcement are org-wide — use a dedicated org. A new org may
  sit in the Evaluation tier; check it before m4-07's acceptance run.
- **Fallback key expiry** is forgotten → it fails safe (401 → manual); the expiry date is a status.md manual check.
- **Double-adding judge → identity:** [m3-07](sprint-m3-07.md) already has it — diff the live policy before editing.
- **The acceptance set in the pack image** would widen what judge mounts — exclude it and confirm the image layer list.
- **Never** set `ANTHROPIC_API_KEY` (it would shadow federation in any SDK); agents never see the salt or any key.

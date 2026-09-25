# Prompt — Sprint mi-12 · M4 gates: judge 443 egress, LLM secret, provider runbook (MI-14) + ADR-0031 → Accepted

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-mi-12.md`](../sprints/sprint-mi-12.md)   ·   **Milestone:** MI (rollout step MI-14, the M4 gates)   ·   **Prereqs:** [m3-07](../sprints/sprint-m3-07.md), [spk-03](../sprints/sprint-spk-03.md)

## Before you launch (owner)

Launching this prompt attests these are done (D40). If one turns out to be missing, land everything that doesn't depend on it and record the gap as ⛔ in `status.md`; don't wait.

- [ ] **MI-1** (`ev-mi1`, owner hygiene; it gates the M4 provider accounts): Hostinger 2FA; two offline copies of the age key; 2FA on GitHub, Anthropic and OpenAI.
- [ ] **Anthropic Console setup** (`ev-provider-runbook`, ~45 min; plan task 3, with the plan's task 2 table as the checklist):
  - a dedicated xLearn org if the current one is shared with Claude Code or other API use, with 2FA on;
  - workspace `xlearn-platform-prod` with a **$15** monthly hard limit, Console spend alerts at **50% and 80%**, low per-model RPM/OTPM, and prepaid credits with **auto-reload off**;
  - WIF: service account `xlearn-judge` (member of `xlearn-platform-prod` only), the issuer and inline JWKS, and the federation rule as the plan's WIF row specifies (`check_jti` per spk-03). Read the issuer URL and the JWKS (public keys) live with `ssh sujaykumar-vps 'sudo k3s kubectl get --raw /.well-known/openid-configuration'` and `… get --raw /openid/v1/jwks`;
  - workspace `xlearn-calib` with its own ~$50/month limit and alerts; its personal key, if any, never goes in the cluster.
- [ ] At launch, hand the session the non-secret IDs (org, workspace, service account, federation rule), the `LLM_KEY_LABEL` and the JWKS `kid`s you pasted. Never hand over a key or secret.

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md) — conventions, GitOps, land-and-sync.
- The plan: [`../sprints/sprint-mi-12.md`](../sprints/sprint-mi-12.md) (the rule table, values snippet, runbook sections and label schema are spelled out there).
- [ADR-0031](../../adr/0031-platform-ai-and-two-tier-keys.md) — all of it; you move it to **Accepted**. §2 (credential, egress), §5 (spend), §8 (the T7 amendments you fold in — unless the BP2 sign-off already folded them; check first).
- [ADR-0035](../../adr/0035-v2-operations-nats-auth-limits-capacity.md) §2 (the NetworkPolicy standing rule), §3 (no alerting, D34; `host-verify` additions), §4 row L17, §6 (the ADR-0031 §5 amendment).
- [ADR-0034](../../adr/0034-v2-release-labelling-gating-and-rollback.md) §2 (`LLM_PLATFORM_ENABLED` is a permanent kill switch); [ADR-0033](../../adr/0033-invite-only-admission-and-owner-admin.md) §12 row 13 (judge reads identity internals, MI-5a is the fence).
- **The spike input:** `## 15. WIF spike result` in [t5](../research/t5-platform-ai.md) (appended by [spk-03](../sprints/sprint-spk-03.md)) — the decision, `check_jti`, refresh time, exchange shape.
- Research [t5 §3](../research/t5-platform-ai.md#3-where-the-platform-key-lives-and-secrets) (WIF, SOPS layout, provider controls, egress, kill switches), [t5 §4](../research/t5-platform-ai.md#4-provider-and-model-choice-per-task) (analyzer acceptance thresholds), [t5 §5](../research/t5-platform-ai.md#5-cost-model) (`xlearn-calib` ≈ $50/month in bring-up), [t5 §6](../research/t5-platform-ai.md#6-budgets-metering-and-abuse-controls) (dogfood $15/$12), [t5 §7](../research/t5-platform-ai.md#7-quality-calibration-regression-and-injection-defences) (dev/test discipline).
- [`../rollout-plan.md`](../rollout-plan.md) §2 row **MI-14**, §3 row **M4**, §6 (calendar), §12 ("M4 adds only 443" — correct as written, since m3-07 gave judge identity :8081 at birth).
- The [spk-03](../sprints/sprint-spk-03.md) sprint (what the spike measured) and [m3-07](../sprints/sprint-m3-07.md) (how judge's HelmRelease and default-deny egress were born — incl. `xlearn-identity` :8081 for the erase re-verify, and the four `extraVolumes` you must keep).
- Infra (edit only through PRs): `../infra/apps/xlearn-judge.yaml`, `../infra/apps/xlearn-coach.yaml` (`envFrom` convention), `../infra/apps/xlearn-identity.yaml` (`xlearn.dev/oauth-rev` annotation precedent), `../infra/charts/project/templates/{deployment,networkpolicy}.yaml` + `values.yaml` (chart 0.3.0 egress template, `extraVolumes`, `serviceAccount`), `../infra/.sops.yaml`, `../infra/hack/host-verify.sh` (its embedded-table markers and `kc_raw` wrapper from [mi-02](../sprints/sprint-mi-02.md)), `../infra/hack/host-lint.sh` (the embedded-copy check), `../infra/hack/expected-netpol.tsv`.
- `../xlearn-evalpack` (private) repo layout and CI (from [m3-02](../sprints/sprint-m3-02.md)) and the AGENT.md **never-copy rule** for it (added by [m3-01](../sprints/sprint-m3-01.md)); the 8 mistake categories: [v1 PRD R-MJ2](../../prd/xlearn-prd.md#64-mistake-journal).

## Context

M3 is live (judge dark → Run/Submit for the owner/tester cohort, `v1.14.0`), and the pilot course is in flight.
M4 (platform AI) may not start until MI-14 is done: judge needs a way out to Anthropic, its credential plumbing,
and a provider account with hard limits — and ADR-0031 must be Accepted with the WIF spike's answer. v2 is
**owner-only** (D35), so the limits are D25's dogfood defaults: **$15** provider hard limit, **$12** app cap.
There is **no alerting** in v2 (D34): spend protection is the provider's own limit and Console alerts plus
judge's in-app caps and breaker (M4 code). This sprint writes **no Go code**: it is ADR + runbook (X), three
infra PRs (I/H), the owner's Console setup (O, done before launch) and an evalpack template (E). ADR-0031's
acceptance happens in this session; there's no owner sign-off (D40).

## Entry gates — verify first (stop and report if any is unmet)

- [ ] [m3-07](../sprints/sprint-m3-07.md) done: `ssh sujaykumar-vps 'sudo k3s kubectl get deploy,networkpolicy -n xlearn xlearn-judge'` shows judge Ready and its policy — incl. the `xlearn-identity` :8081 rule (erase re-verify)
- [ ] [spk-03](../sprints/sprint-spk-03.md) reported: WIF **GO** (+ `check_jti` decision, refresh time) or the fallback chosen; throwaway workspace deleted
- [ ] MI-1 done (`ev-mi1`): 2FA on the owner's Anthropic account (a before-launch item, attested by launching this prompt)
- [ ] `apps/xlearn-judge.yaml` uses the chart 0.3.0 egress template ([mi-01](../sprints/sprint-mi-01.md))
- [ ] No open peer PR touches `../infra/apps/xlearn-judge.yaml` or `docs/adr/0031-*` (`gh pr list` in xlearn and `../infra`, `git worktree list`, ListAgents)

## Do this (in order)

1. **(X) Parallel-sessions check + branches.** Confirm no peer is editing the same files; branch `docs/mi-12-adr-0031-runbook` in xlearn and `feat/xlearn-judge-m4-egress` in `../infra`.
2. **(X) Accept ADR-0031** (plan task 1): fold the spk-03 result (t5 §15) into §2 (*Spike result* list; on the fallback, the 90-day key as primary + WIF retry at the next k3s minor), rewrite §2 *Egress* as the full allow-list (DNS, PG, NATS, runner, gateway :8080, **identity :8081 — since M3/m3-07 for the erase re-verify, reused by M4's `/internal/accounts/{id}`**, **TCP 443 non-cluster** — the only M4 addition), add §2 *Maintenance* (JWKS re-paste; the kid check). **If §8 is still a separate section** (the BP2 sign-off may already have folded it — check), fold it into §4/§5 and Consequences (dogfood $15/$12 now, $100/$80 at the v3 opening; no push channel; manual `retire_not_before`) and shrink §8 to a dated history; otherwise skip that part. Set **Status → Accepted (date)**; update the row in `docs/adr/README.md`.
3. **(X) Write the provider runbook** `docs/v2/runbooks/platform-ai-provider.md` with every section of the plan's task 2 table (org, workspace $15 + alerts 50/80%, rate limits, billing auto-reload OFF, WIF issuer/rule/SA, hand-over IDs, `xlearn-calib` at ~$50/month in bring-up with its own alerts, break-glass, kill switches, maintenance, v3 raise). Record the owner's before-launch setup in it (date, non-secret IDs). Read the issuer live (`ssh sujaykumar-vps 'sudo k3s kubectl get --raw /.well-known/openid-configuration'`) and fetch the JWKS (`… get --raw /openid/v1/jwks`), public keys only, and check they match what the owner pasted (the handed-over `kid`s).
4. **(I) Egress PR** (plan task 4): read judge's live policy first and confirm m3-07's `xlearn-identity` :8081 rule is present (add it **only if missing**, and say why in the PR). In `apps/xlearn-judge.yaml` add exactly **one** egress rule — TCP 443 to `0.0.0.0/0` except `10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`, `100.64.0.0/10`, `169.254.0.0/16` and the node IP `/32` (read `k3s kubectl get node -o wide`; confirm the pod/service CIDRs sit inside `10.0.0.0/8`). Paste the before/after rendered-policy diff and the caller-matrix delta (443 new; judge → identity unchanged since m3-07) in the PR. Merge → Flux reconciles → judge still Ready.
5. **(O, before launch) Check the Console hand-over** (plan task 3, `ev-provider-runbook`): the owner did the Console setup before launch and handed over the non-secret IDs, the `LLM_KEY_LABEL` and the pasted `kid`s. Record them (never a key or secret). **If they're missing:** carry on with steps 7–8, don't open step 6 (no draft PR is left behind), mark plan tasks 3 and 5 ⛔ "waiting on ev-provider-runbook" in status.md, and land the rest; a re-run of this prompt after the setup lands step 6.
6. **(I) Secret + values + token PR** (plan task 5): create `apps/secrets/xlearn-judge-llm.enc.yaml` with `LLM_ENDUSER_SALT` generated and SOPS-encrypted in one pipe (never echoed); on the fallback the **owner** adds `LLM_ANTHROPIC_API_KEY` with `sops` themselves after this PR creates the file (no agent handles the key; record it as a pending owner item in status.md, needed before m4-07's enable, and don't wait). In `apps/xlearn-judge.yaml` add the `envFrom` secretRef, the non-secret env (`LLM_PLATFORM_ENABLED: "false"`, the four `LLM_ANTHROPIC_*_ID`s, `LLM_KEY_LABEL`), the projected-token `extraVolumes`/`extraVolumeMounts` (WIF only; audience `https://api.anthropic.com`, 3600 s, mount `/var/run/secrets/anthropic.com`), keep `automountServiceAccountToken: false`, add `podAnnotations: xlearn.dev/llm-rev: "1"`. Keep m3-07's four `extraVolumes` (evalpack, nats-seed, runner-auth, scratch) and their mounts — Helm replaces lists, so append, never overwrite. Check the Secret decrypts **without printing it** (`sops -d apps/secrets/xlearn-judge-llm.enc.yaml >/dev/null && echo decrypts`; `yq '.stringData | keys'` on the encrypted file for its structure); `helm template` the release (values only — no Secret rendered). Merge when no cohort evaluation is running (judge restarts once).
7. **(H) host-verify kid check PR** (plan task 6): add `hack/expected-jwks-kids.txt` (step 5's hand-over) **and embed it in `hack/host-verify.sh`** between `# >>> expected-jwks-kids.txt` / `# <<< expected-jwks-kids.txt` markers (mi-02's convention: the script runs piped or from `/root`, with no file beside it); extend `hack/host-lint.sh` to fail when the embedded copy differs from the file. `--cluster` compares the live JWKS `kid`s (`kc_raw /openid/v1/jwks` + `jq`) with the embedded block → PASS / WARN with the runbook path on drift; INFO-skip **only when the embedded block is empty** (fallback path). Test with `ssh sujaykumar-vps 'bash -s -- --cluster' < hack/host-verify.sh` (PASS) and a piped scratch copy with one kid altered (WARN); paste both. `hack/host-lint.sh` (read-only grep, shellcheck) green. After merge, refresh the node copy: `scp hack/host-verify.sh sujaykumar-vps:/root/` (mi-02 task 7's sanctioned refresh; a node write this prompt specifies, so it's pre-approved by launching it, D40).
8. **(E) Acceptance-set template PR** in `../xlearn-evalpack` (plan task 7; never copy its content into xlearn): `acceptance/README.md` (≥ 40 test = 15 optimal / 15 suboptimal / 10 failing; ≥ 30 dev; thresholds; frozen-test rules; owner-written solutions to stamped items only), `acceptance/schema/label.schema.json`, `acceptance/{dev,test}/` (+ `test/configs_tried.log`), one `example: true` file, JSON-schema validation of every label in evalpack CI (e.g. `check-jsonschema`; m3-02's `packspec` validator is for case inputs, not JSON schema), and `acceptance/` excluded from the pack image build context. No evalpack tag.
9. **(H) Verify** (plan task 8): `ssh sujaykumar-vps 'bash /root/host-verify.sh --cluster'` green with the kid check reporting **PASS** (not INFO-skip); judge Ready; a cohort judged submit grades end to end; the pod spec lists the `anthropic-token` volume (next to evalpack, nats-seed, runner-auth, scratch) and automount `false`; print no secret value. Functional 443 reachability is deferred to `judge admin llm-smoke` from the pod ([m4-01](../sprints/sprint-m4-01.md) task 10, m4-07 step 1) — a dial timeout there reopens plan task 4.
10. **(X) Record**: status.md (below); [rollout §12](../rollout-plan.md#12-downstream-constraints-for-the-build-plan-session)'s "M4 adds only 443" stays as written. Open the xlearn docs PR (ADR-0031, runbook, status), CI green, merge.

## Constraints

- **GitOps only:** never `kubectl apply`/`edit`/`debug`/`run` by hand; `ssh sujaykumar-vps` reads (`get`, `get --raw`, `top`, `host-verify.sh`) only; `kubectl exec` only for the admin CLIs (D33). Mirror `../infra` conventions (chart values, SOPS rule, file names).
- **Infra PRs stand alone** and are never folded into a tag; the egress change is **its own PR**, merged before `v1.16.0` ([ADR-0035 §2](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#2-nats-auth-nkey-users-fine-acls-server-first) standing rule).
- **Secrets:** SOPS only; never print, log or commit the salt or a key in plaintext (no bare `sops -d` to the terminal — redirect to `/dev/null`); **agents never handle the break-glass key** (the owner edits the SOPS file); never use the env name `ANTHROPIC_API_KEY`; no Admin keys anywhere in the cluster.
- **D34 — no alerting:** no push channel, healthchecks.io ping, Flux `Provider`/`Alert`, opscheck, timer or CronJob. The `host-verify --cluster` extension is the only ops tooling.
- **Service boundaries ([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)):** judge reaches identity only through `/internal/accounts/{id}` over HTTP behind the MI-5a fence — never identity's schema.
- **No code:** nothing in `internal/`, `cmd/`, `web/` this sprint (goose/sqlc/outbox rules don't apply; M4 code is m4-01…m4-07).
- **Memory-sum rule (ADR-0035 §5):** no new pod or container here; judge only restarts. Confirm `host-verify --cluster` stays inside the rule.
- **D35 / D25:** limits are $15 provider / $12 app; $100 / $80 only at the v3 opening.
- **Parallel sessions:** check peers' PRs, tags and worktrees before editing `apps/xlearn-judge.yaml` or ADR-0031; this sprint claims no ADR number.
- **Private eval pack:** never copy anything from `../xlearn-evalpack` (acceptance artefacts are solutions to pack items) into the public xlearn repo — PRs, status.md and docs included.

## Deliverables

- `docs/adr/0031-platform-ai-and-two-tier-keys.md` **Accepted**; `docs/adr/README.md` row updated.
- `docs/v2/runbooks/platform-ai-provider.md`.
- `../infra` PR A: judge egress (TCP 443 non-cluster; identity :8081 confirmed from m3-07).
- `../infra` PR B: `apps/secrets/xlearn-judge-llm.enc.yaml` + `apps/xlearn-judge.yaml` (envFrom, LLM values with `LLM_PLATFORM_ENABLED=false`, projected token, `llm-rev` annotation).
- `../infra` PR C: `hack/host-verify.sh` kid check (embedded kid block) + `hack/expected-jwks-kids.txt` + the `host-lint.sh` embedded-copy check; `/root/host-verify.sh` refreshed.
- `xlearn-evalpack` PR: `acceptance/` template + schema + CI validation.
- The owner's before-launch Console setup, recorded (date, non-secret IDs) in the runbook and status.md.

## Update status

- This sprint's Status table in [`../sprints/sprint-mi-12.md`](../sprints/sprint-mi-12.md) (🔄 / ✅ / ⛔ per task; _Overall_).
- [`../status.md`](../status.md): Sprint board row; **MI table** row MI-14 ✅ with the PR numbers; **flag inventory** (kill switches) — `LLM_PLATFORM_ENABLED` present = `false`; **events** — `ev-provider-runbook` ✅ (date), `ev-acceptance-set` ⬜ "template ready"; **manual checks** — JWKS kids + date (and the break-glass key expiry on the fallback); **accepted-risk register** — inline JWKS rotation; DNS tunnelling / own-output exfiltration; **decisions log** — "ADR-0031 Accepted (WIF GO | fallback; `check_jti`=…)" and "M4 adds only TCP 443 (judge → identity :8081 exists since m3-07)".
- No new ADR; ADR-0031's own acceptance is the record.

## Done when (acceptance)

- [ ] ADR-0031 **Accepted** with the spk-03 result and the T7 amendments folded into its body; ADR index row updated
- [ ] judge's NetworkPolicy adds exactly **one** egress rule — **TCP 443 to non-cluster addresses** — nothing else new; m3-07's **`xlearn-identity` :8081** rule present; judge Ready and a cohort judged submit still grades (functional 443 reachability: `judge admin llm-smoke`, m4-01 task 10 / m4-07 step 1)
- [ ] `xlearn-judge-llm` loaded via `envFrom`; `LLM_PLATFORM_ENABLED=false`; the projected token mounted at `/var/run/secrets/anthropic.com/token` with automount still off
- [ ] Runbook merged; Console limit **$15**, alerts **50%/80%**, auto-reload **off**, WIF issuer/rule/service account set (the owner's before-launch setup, attested at launch and recorded with its date and IDs)
- [ ] `host-verify --cluster` green, with the JWKS kid check reporting **PASS** from the embedded kid block (a forced-drift run shows the WARN); `/root/host-verify.sh` refreshed
- [ ] Acceptance-set template merged in `xlearn-evalpack` (schema, README, dev/test dirs, CI validation, excluded from the pack image)

## Ship (land-and-sync — owner approval pre-granted)

> Launching this prompt is the owner's approval for every change it makes (D40); don't stop for review.

1. Branch, then conventional commit(s) with the attribution lines, then push, then a PR in every repo touched (`../infra` PRs first where the order requires it; infra PRs are never folded into a tag).
2. Once CI is green (fix, then merge, on failure), squash-merge. Never enable auto-merge. (infra has no CI: paste the local checks, such as the rendered-policy diff, `host-lint.sh` and both `host-verify` runs, into each PR body and merge on them.)
3. **Release action — infra PR(s) only, no tag:**
   - Merge the three `../infra` PRs, each on its own: (A) egress first, any time before `v1.16.0`; (B) secret + values + token, only with the owner's handed-over IDs (if they're missing, B isn't opened: see *Do this* step 5); (C) the host-verify kid check, then refresh `/root/host-verify.sh`.
   - Let Flux reconcile, and verify live (*Do this* step 9).
   - Then merge the `xlearn-evalpack` template PR (no evalpack tag), and the xlearn docs PR (ADR-0031 Accepted, the runbook, status).
   - Nothing ships in an xlearn image; the M4 code rides `v1.16.0` ([m4-07](../sprints/sprint-m4-07.md)).
4. Update status: the sprint file and `docs/v2/status.md`, in the same PR or a follow-up docs PR merged the same way.
5. Run `git checkout main && git pull` in every repo touched (xlearn, `../infra`, `../xlearn-evalpack`). If a clean peer worktree holds `main`, use `git -C <worktree> merge --ff-only origin/main` and then `git switch --detach main`.

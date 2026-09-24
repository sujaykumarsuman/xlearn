# Prompt — Sprint spk-03 · WIF spike (≤ ½ day, throwaway)

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-spk-03.md`](../sprints/sprint-spk-03.md) · **Milestone:** MI (MI-14's WIF item, MI-10 spike week) · **Prereqs:** [spk-01](../sprints/sprint-spk-01.md) (its VM, or any throwaway k3s)

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md) — repo conventions, the land-and-sync directive.
- [`../sprints/sprint-spk-03.md`](../sprints/sprint-spk-03.md) — the plan: questions Q-W1…Q-W4, the probe-pod manifest, the decision rule.
- [ADR-0031 §2](../../adr/0031-platform-ai-and-two-tier-keys.md#2-credential-and-egress) (WIF credential, fallback key, workspace pinning) and [§8](../../adr/0031-platform-ai-and-two-tier-keys.md#8-amended-by-t7-2026-09-24) (the spike is scratch-only). ADR-0031 stays **Proposed** in this sprint (BP2).
- [t5 §3](../research/t5-platform-ai.md#3-where-the-platform-key-lives-and-secrets) — the WIF settings table (issuer, rule, pod, code, **spike row**, maintenance, fallback) and [§13](../research/t5-platform-ai.md#13-risks) (the `jti_reused` risk row).
- [Rollout §2](../rollout-plan.md#2-mi-infra-track) (MI-10, MI-14) and [§6](../rollout-plan.md#6-critical-path-parallel-tracks-owner-calendar) (calendar: WIF in the spike week if it fits, else before M4).
- [t3 §15](../research/t3-sandbox.md) — the "S0 results" section, the format precedent for the t5 §15 you will add.
- The Anthropic WIF docs: [overview](https://platform.claude.com/docs/en/manage-claude/workload-identity-federation), [reference](https://platform.claude.com/docs/en/manage-claude/wif-reference), [Kubernetes provider](https://platform.claude.com/docs/en/manage-claude/wif-providers/kubernetes) — the token-exchange request shape, `jti` single use, `check_jti`, token lifetimes.
- `../infra/hack/host-bootstrap.sh` — the pinned k3s version (`PIN_K3S_VERSION`, `v1.36.4+k3s1`) to mirror.

## Context

Platform AI (M4) authenticates judge to Anthropic with **Workload Identity Federation**: a k3s projected
service-account token (audience `https://api.anthropic.com`, 3600 s, at `/var/run/secrets/anthropic.com/token`)
is exchanged for a short-lived `workspace:inference` token bound to one workspace. The issuer uses an **inline
JWKS** because the cluster's issuer isn't publicly reachable. Before [mi-12](../sprints/sprint-mi-12.md) wires
production (December) and [m4-01](../sprints/sprint-m4-01.md) writes the client, this ≤ ½-day spike answers:
does the token carry a `jti`, how does it rotate, does an **in-place container restart** re-present a used
`jti` (`jti_reused`), and how long does a refresh take? The answer is **WIF GO** (with a `check_jti` decision)
or the **fallback**: a single-workspace service-account key with a 90-day provider-side expiry. Production's
issuer string is `https://kubernetes.default.svc.cluster.local` (read-only check, 2026-09-24) — the same default
every k3s cluster uses, including the throwaway one.

## Entry gates — verify first (stop and report if any is unmet)

- [ ] **Owner go-ahead for the WIF spike** (ev-spike-goahead). If there is no explicit yes in `docs/v2/status.md` or from the owner in this session, stop.
- [ ] A throwaway k3s: the [spk-01](../sprints/sprint-spk-01.md) VM **`xl-spike`** (`multipass list`; stopped, handed over by [spk-02](../sprints/sprint-spk-02.md)'s teardown — this sprint is its last user and purges it) **or** you can launch a fresh multipass VM **`xlearn-wif`**.
- [ ] The owner is available for ≈ 30 min in the Anthropic Console (step 2, right after step 1 prints the issuer and JWKS; step 8 at the end). The agent never logs into the Console and never handles Console credentials.
- [ ] *(soft)* MI-1: 2FA on the Anthropic account — ask; if not done, note it in the report.

## Do this (in order)

**Every `kubectl`, `curl` and `systemctl` runs inside the VM** (`multipass exec <vm> -- sudo k3s kubectl …`,
`<vm>` = `xl-spike` or `xlearn-wif`). The laptop's kubectl context and `127.0.0.1:6443` tunnel to
**production**; keep `KUBECONFIG` unset.

1. **[H] Throwaway k3s + probes.** `multipass start xl-spike` (already on `v1.36.4+k3s1`), or
   `multipass launch 24.04 --name xlearn-wif --cpus 2 --memory 4G --disk 10G` + k3s pinned to `v1.36.4+k3s1`.
   Print `get --raw /.well-known/openid-configuration` and `get --raw /openid/v1/jwks` for step 2. On the
   **throwaway** cluster only, apply namespace `wif-spike`, ServiceAccount `wif-probe`
   (`automountServiceAccountToken: false`) and the two probe pods from the plan (`wif-probe-3600`,
   `wif-probe-600`), with the plan's command exactly: PID 1 is `sh` with a `trap 'exit 1' TERM` loop, so step 5's
   `kill 1` works. Keep every manifest and script in the session scratchpad. Never set `ANTHROPIC_API_KEY`.
2. **[O] Throwaway workspace + WIF objects.** With step 1's output in hand, ask the owner to create, in the org
   production will use (record which): workspace `xlearn-wif-spike` (lowest spend limit, auto-reload off);
   service account `xlearn-wif-spike` (that workspace only); a WIF issuer (issuer URL and inline JWKS from
   step 1); a rule `subject_prefix: system:serviceaccount:wif-spike:wif-probe`,
   `audience: https://api.anthropic.com`, `oauth_scope: workspace:inference`, `token_lifetime_seconds: 3600`,
   `check_jti` at its default. Get back only the **non-secret ids** (org, workspace, service account, rule).
   No API key.
3. **[H] Q-W1/Q-W2.** Decode the token payload (claim names + non-secret values; `jti` present?). Sample
   `iat`/`exp`/`jti` every 60 s: the 600 s pod ≥ 20 min, the 3600 s pod ≥ 60 min if the box allows. Record the
   rotation offsets as a % of TTL.
4. **[H] Q-W4.** From `wif-probe-3600`, `POST /v1/oauth/token` with the documented request shape; then one
   minimal `POST /v1/messages` (`claude-sonnet-5`, `max_tokens` ≈ 16). Record `expires_in`, the
   `anthropic-workspace-id` header (must match the throwaway workspace), and `curl -w '%{time_total}'` for 5
   runs (p50/max). Then one Files or Batches request with the access token → expect 403 (status only).
5. **[H] Q-W3.** (a) exchange twice with the same file → record the second response; (b) in-place restart:
   `kubectl -n wif-spike exec wif-probe-3600 -- kill 1` (the trapping `sh` exits; if it doesn't, from the VM
   host `sudo k3s crictl ps --name probe --label io.kubernetes.pod.name=wif-probe-3600` → `sudo k3s crictl stop
   <id>`); confirm restartCount rose by exactly 1 and `jti`/`iat` are unchanged, exchange → record; (c) recreate the
   pod → exchange succeeds. If (a) rejects and (b) re-presents a used `jti`, ask the owner to set
   `check_jti=false` on the issuer and re-run (a)/(b). Note the rejected alternative (emptyDir token cache).
6. **[H] Side checks.** The projected token against the **throwaway** API server, from inside the probe pod
   (`kubectl -n wif-spike exec wif-probe-600 -- sh -c 'curl -sk -o /dev/null -w "%{http_code}\n" -H "Authorization: Bearer $(cat /var/run/secrets/anthropic.com/token)" https://kubernetes.default.svc/api'`)
   → expect 401; never against the laptop's `127.0.0.1:6443` (production). If time allows, after steps 3–5 and
   only on a VM no other spike needs (`xl-spike` after spk-02's hand-over, or a fresh `xlearn-wif`): JWKS before
   and after `sudo systemctl restart k3s` and after `sudo k3s certificate rotate` + restart, inside the VM;
   record which operations change it.
7. **[X] Report (docs PR).** On a `docs/…` branch in xlearn: add **`## 15. WIF spike result (spk-03, <date>)`** to
   [`../research/t5-platform-ai.md`](../research/t5-platform-ai.md) — results table, decision (GO +
   `check_jti` + re-exchange rule + "access-token lifetime > rotation interval", or fallback), the exchange
   request shape (endpoint + field names only), the values mi-12 sets, the org used, the teardown date. Update
   [`../status.md`](../status.md) (Sprint board row, the MI table's MI-14 row WIF-spike item, Decisions log) and this sprint's
   Status table.
8. **[O + H] Tear down.** Owner deletes the rule, issuer, service account and workspace (required: the issuer
   shares production's default `iss`). You delete the `wif-spike` namespace and `multipass delete --purge <vm>`.
   This sprint is `xl-spike`'s last user (spk-01 and spk-02 hand it on; it also holds spk-02's pull secret). On a
   fresh `xlearn-wif`, purge that and confirm `xl-spike` is already gone. Record the purge in `status.md`. Grep
   the scratchpad and the branch for any token or access token before pushing.

## Constraints

- **Throwaway only.** No command touches the VPS or the production cluster (the laptop's kubectl context and
  `127.0.0.1:6443` *are* production: go through `multipass exec <vm>`); nothing is applied to any real
  cluster; no infra PR. The only production fact used (the issuer string) is already recorded.
- **Secrets hygiene.** Decode claims; never paste a whole projected token or access token into the transcript,
  a doc, a PR or a commit. Record field names, not values. No `ANTHROPIC_API_KEY` anywhere.
- **Spend:** the throwaway workspace carries the lowest limit; a handful of ~16-token calls only. Never use a
  production or calibration workspace.
- **ADR-0031 stays Proposed.** Don't edit ADRs; [mi-12](../sprints/sprint-mi-12.md) accepts ADR-0031 with this
  result (BP2).
- **D34:** no alerting of any kind is added. **GitOps:** not applicable (no cluster change); never `kubectl apply`
  against production.
- **Parallel sessions:** before pushing the docs PR, check peers (`gh pr list --state all`, `git worktree list`,
  ListAgents) — t5 and `status.md` may be open in a peer's branch; rebase onto a moved `main`.
- n/a in this sprint: service boundaries, goose/sqlc, outbox/inbox, `theme.css`, memory-sum rule,
  consumers-before-producers, ACL-before-tag.
- **Time box:** ≤ ½ day. If Q-W1…Q-W4 aren't answered in the box, record what is known and choose the fallback
  or re-book before M4.

## Deliverables

- `docs/v2/research/t5-platform-ai.md` §15 (the one-page result) and the `status.md` rows — one docs PR.
- A GO / fallback decision with `check_jti`, the re-exchange rule, the lifetime invariant and the request shape
  that [mi-12](../sprints/sprint-mi-12.md) and [m4-01](../sprints/sprint-m4-01.md) consume.
- Owner confirmation that every throwaway Anthropic object is deleted; the VM gone.

## Update status

- Set each task in [`../sprints/sprint-spk-03.md`](../sprints/sprint-spk-03.md) 🔄 / ✅ / ⛔ as you go;
  _Overall_ ✅ when all eight are ✅.
- [`../status.md`](../status.md): the **Sprint board** row, the **MI** table's **MI-14** row, WIF-spike item
  (spk-03: GO / fallback, date, link to t5 §15; MI-14 stays open until mi-12), and a **Decisions log** line (e.g. "WIF GO, `check_jti=false` on the one-rule issuer").
- No ADR is written or accepted here; if the result contradicts ADR-0031 §2, say so in the Decisions log so
  mi-12 amends it at acceptance.

## Done when (acceptance)

- [ ] Q-W1…Q-W4 answered with numbers (`jti`, rotation %, reuse/restart behaviour, exchange + first-call
      p50/max, workspace header).
- [ ] WIF GO (with `check_jti`, re-exchange rule, lifetime invariant) **or** fallback chosen — recorded in t5 §15
      and `status.md`.
- [ ] The exchange request shape (endpoint + field names) is recorded for m4-01.
- [ ] Throwaway workspace, issuer, rule and SA deleted (owner confirms); namespace and VM gone; no secret in
      any committed file.

**Shipping:** spike — the VM, manifests and scripts are **throwaway and never committed**. Per AGENT.md
land-and-sync, ship **only the results docs PR** (branch `docs/…` → conventional commit with the attribution
lines → push → PR → merge once green → `git checkout main && git pull`, via the peer-worktree route if `main`
is checked out elsewhere). No tag, no infra PR, no deploy.

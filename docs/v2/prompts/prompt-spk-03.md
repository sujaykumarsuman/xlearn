# Prompt — Sprint spk-03 · WIF spike (≤ ½ day, throwaway)

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-spk-03.md`](../sprints/sprint-spk-03.md) · **Milestone:** MI (MI-14's WIF item, MI-10 spike week) · **Prereqs:** [spk-01](../sprints/sprint-spk-01.md) (its VM, or any throwaway k3s). Launching this prompt is the WIF go-ahead (D23, D40)

## D41 changes (read first; they override the text below where they conflict)

> **D41 (owner, 2026-09-25): spikes first.** All four spikes run **before any build sprint**, so every design yes/no is answered before M1 starts: spk-01 and spk-02 on Fri 2026-09-25 (agent-only, after the MI-0 reboot), spk-03 and spk-04 on Sat 2026-09-26 (spk-03 after the owner's Console step; spk-04 with the owner present). For this sprint: the calendar line below is superseded (run Sat 2026-09-26). The throwaway VM is **`xlearn-wif`**, created on 2026-09-25 by the build-plan session (2 CPU, 4 GiB, k3s `v1.36.4+k3s1`, issuer `https://kubernetes.default.svc.cluster.local`); its public JWKS was handed to the owner for the Console step. Run `multipass start xlearn-wif` if it's stopped; purge it at teardown.

## Before you launch (owner)

Launching this prompt attests these are done (D40). If one turns out to be missing, land everything that doesn't depend on it and record the gap as ⛔ in `status.md`; don't wait.

- [ ] **The throwaway VM is up and you have its public JWKS.** If spk-02 handed `xl-spike` over: `multipass start xl-spike`. Otherwise launch a fresh one, `multipass launch 24.04 --name xlearn-wif --cpus 2 --memory 4G --disk 10G`, and install k3s pinned to `v1.36.4+k3s1` the way `../infra/hack/host-bootstrap.sh` does (`INSTALL_K3S_VERSION`). Then print the JWKS (public keys only): `multipass exec <vm> -- sudo k3s kubectl get --raw /openid/v1/jwks`.
- [ ] **Anthropic Console (≈ 20 min), in the org production will use** (the dedicated xLearn org if it exists, else the current one): workspace `xlearn-wif-spike` (the lowest monthly spend limit, auto-reload off); service account `xlearn-wif-spike` (that workspace only); a WIF issuer with URL `https://kubernetes.default.svc.cluster.local` and `jwks.type: inline` holding the JWKS above; a rule with `subject_prefix: system:serviceaccount:wif-spike:wif-probe`, `audience: https://api.anthropic.com`, `oauth_scope: workspace:inference`, `token_lifetime_seconds: 3600`, the one workspace, and `check_jti` at its default. No API key.
- [ ] **In your launch message:** the VM's name (`xl-spike` or `xlearn-wif`), which org you used, and the non-secret ids the exchange needs (org, workspace, service account, federation rule). Never a key or a token.

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

- [ ] **The WIF go-ahead is this launch** (D23, D40); record it under `ev-spike-goahead`. Don't ask for another one.
- [ ] The throwaway k3s named in the launch message is up (`multipass list`): the [spk-01](../sprints/sprint-spk-01.md) VM **`xl-spike`** (handed over by [spk-02](../sprints/sprint-spk-02.md)'s teardown — this sprint is its last user and purges it) **or** a fresh **`xlearn-wif`**. If it's stopped, start it.
- [ ] The launch message carries the Console objects' non-secret ids (the before-launch block). The agent never logs into the Console and never handles Console credentials. **If the ids are missing,** run steps 1, 3 and 6 (they need no Console object), record Q-W3/Q-W4 ⛔ "Console objects missing" in `status.md`, and land the partial results; don't wait.
- [ ] *(soft)* MI-1: read the Anthropic 2FA state from status.md's MI-1 row; if it isn't recorded as done, note that in the report.

## Do this (in order)

**Every `kubectl`, `curl` and `systemctl` runs inside the VM** (`multipass exec <vm> -- sudo k3s kubectl …`,
`<vm>` = `xl-spike` or `xlearn-wif`). The laptop's kubectl context and `127.0.0.1:6443` tunnel to
**production**; keep `KUBECONFIG` unset.

1. **[H] Throwaway k3s + probes.** The VM from the launch message (`xl-spike` on `v1.36.4+k3s1`, or
   `xlearn-wif` with k3s pinned to `v1.36.4+k3s1`), started before launch.
   Print `get --raw /.well-known/openid-configuration` and `get --raw /openid/v1/jwks` for the report. On the
   **throwaway** cluster only, apply namespace `wif-spike`, ServiceAccount `wif-probe`
   (`automountServiceAccountToken: false`) and the two probe pods from the plan (`wif-probe-3600`,
   `wif-probe-600`), with the plan's command exactly: PID 1 is `sh` with a `trap 'exit 1' TERM` loop, so step 5's
   `kill 1` works. Keep every manifest and script in the session scratchpad. Never set `ANTHROPIC_API_KEY`.
2. **[O] Throwaway workspace + WIF objects — done before launch** (the before-launch block): workspace
   `xlearn-wif-spike`, service account `xlearn-wif-spike`, the WIF issuer with the VM's inline JWKS, and the rule
   (`check_jti` at its default). Take the **non-secret ids** and the org used from the launch message and record
   which org it is. No API key. Don't ask for anything; a missing piece follows the entry gates.
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
   pod → exchange succeeds. If (a) rejects and (b) re-presents a used `jti`, the decision is **`check_jti=false`**
   on the one-rule issuer (t5 §3's accepted trade; the pre-decided path). Setting it is a Console action, so don't
   wait for it (D40): record the decision and hand the confirming (a)/(b) re-run under `check_jti=false` to mi-12,
   which creates the production issuer with the flag (`status.md` → Hand-offs). Note the rejected alternative
   (emptyDir token cache).
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
8. **[H] Tear down.** Record in `status.md` → Open owner items: "delete the spk-03 rule, issuer, service account and
   workspace `xlearn-wif-spike` in the Console — required before mi-12 creates the production issuer (it shares
   production's default `iss`)". That's an owner follow-up, not a wait. You delete the `wif-spike` namespace and
   `multipass delete --purge <vm>`.
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
  (the pre-decided path, D40) or re-book before M4.

## Deliverables

- `docs/v2/research/t5-platform-ai.md` §15 (the one-page result) and the `status.md` rows — one docs PR.
- A GO / fallback decision with `check_jti`, the re-exchange rule, the lifetime invariant and the request shape
  that [mi-12](../sprints/sprint-mi-12.md) and [m4-01](../sprints/sprint-m4-01.md) consume.
- The VM gone; the deletion of every throwaway Anthropic object recorded as an owner follow-up.

## Update status

- Set each task in [`../sprints/sprint-spk-03.md`](../sprints/sprint-spk-03.md) 🔄 / ✅ / ⛔ as you go;
  _Overall_ ✅ when all eight are ✅.
- [`../status.md`](../status.md): the **Sprint board** row, the **MI** table's **MI-14** row, WIF-spike item
  (spk-03: GO / fallback, date, link to t5 §15; MI-14 stays open until mi-12), a **Decisions log** line (e.g. "WIF GO, `check_jti=false` on the one-rule issuer"), `ev-spike-goahead` (WIF: the launch, D40), the `check_jti` re-run hand-off to mi-12 if the flag is needed, and the Console clean-up under **Open owner items**.
- No ADR is written or accepted here; if the result contradicts ADR-0031 §2, say so in the Decisions log so
  mi-12 amends it at acceptance.

## Done when (acceptance)

- [ ] Q-W1…Q-W4 answered with numbers (`jti`, rotation %, reuse/restart behaviour, exchange + first-call
      p50/max, workspace header).
- [ ] WIF GO (with `check_jti`, re-exchange rule, lifetime invariant) **or** fallback chosen — recorded in t5 §15
      and `status.md`.
- [ ] The exchange request shape (endpoint + field names) is recorded for m4-01.
- [ ] Namespace and VM gone; no secret in any committed file; the throwaway workspace, issuer, rule and SA
      deletion recorded as an owner follow-up (mi-12's entry checks it).

## Ship (land-and-sync — owner approval pre-granted)

> Launching this prompt is the owner's approval for every change it makes (D40); don't stop for review.

1. Branch `docs/spk-03-wif-results`, then a conventional commit (e.g. `docs(v2): WIF spike result (spk-03, MI-14)`) with the attribution lines, then push, then the PR. This repo only: no `../infra` PR. Grep the branch for any token or access token before pushing.
2. Once CI is green (fix, then merge, on failure), squash-merge. Never enable auto-merge.
3. **Release action — spike (throwaway):** the VM, manifests and probe scripts are never committed; only this results docs PR lands. No tag, no deploy. ADR-0031 stays Proposed (mi-12 accepts it).
4. Update status: the sprint file and `docs/v2/status.md`, in the same PR or a follow-up docs PR merged the same way.
5. Run `git checkout main && git pull`. If a clean peer worktree holds `main`, use `git -C <worktree> merge --ff-only origin/main` and then `git switch --detach main`.

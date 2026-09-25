# Sprint spk-03 — WIF spike (≤ ½ day, throwaway)

> **Milestone:** MI — the WIF item of rollout step **MI-14**, run in the **MI-10** spike week if it fits · **Track:** spike · **Order:** 18
> **Prereqs:** [spk-01](sprint-spk-01.md) (its throwaway multipass k3s, or any throwaway k3s) · the owner's before-launch Console setup (task 2). Launching this prompt is the WIF spike's own go-ahead (D23, D40)
> **Unblocks:** [mi-12](sprint-mi-12.md) (ADR-0031 → Accepted, production WIF config), and through it [m4-01](sprint-m4-01.md) (`platform/llm` WIF auth)
> **Release action:** **no merge (spike, throwaway).** Nothing from the VM is committed; only a **docs PR** recording the result (t5 §15 + `status.md`) merges, on CI green with no owner stop (D40).
> **Calendar:** **Fri 2026-10-16** if it fits the spike week (ev-spike-week, after [spk-02](sprint-spk-02.md)); otherwise any day before M4 — it must report before [mi-12](sprint-mi-12.md) (December).
> **Execute with:** [`../prompts/prompt-spk-03.md`](../prompts/prompt-spk-03.md) — one prompt, one session.

## D41 changes (read first; they override the text below where they conflict)

> **D41 (owner, 2026-09-25): spikes first.** All four spikes run **before any build sprint**, so every design yes/no is answered before M1 starts: spk-01 and spk-02 on Fri 2026-09-25 (agent-only, after the MI-0 reboot), spk-03 and spk-04 on Sat 2026-09-26 (spk-03 after the owner's Console step; spk-04 with the owner present). For this sprint: the calendar line below is superseded (run Sat 2026-09-26). The throwaway VM is **`xlearn-wif`**, created on 2026-09-25 by the build-plan session (2 CPU, 4 GiB, k3s `v1.36.4+k3s1`, issuer `https://kubernetes.default.svc.cluster.local`); its public JWKS was handed to the owner for the Console step. Run `multipass start xlearn-wif` if it's stopped; purge it at teardown.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | Throwaway k3s (`xl-spike`, or a fresh `xlearn-wif`; started before launch) + probe pods (projected token, audience `https://api.anthropic.com`); print issuer + JWKS | H | ⬜ |
| 2 | Throwaway Anthropic workspace + WIF issuer (inline JWKS from the VM), rule and service account | O (before launch) | ⬜ |
| 3 | Q-W1/Q-W2: token claims (`jti`?) and rotation cadence | H | ⬜ |
| 4 | Q-W4: token exchange + first Messages call, timed; workspace header; scope check | H | ⬜ |
| 5 | Q-W3: `jti` reuse and the in-place restart → `check_jti` decision | H | ⬜ |
| 6 | Side checks: audience rejected by the API server; JWKS stability | H | ⬜ |
| 7 | Report: GO (+ `check_jti`) or fallback → t5 §15 + `status.md` (docs PR) | X | ⬜ |
| 8 | Tear down: VM purged (spk-03 is `xl-spike`'s last user); the workspace, issuer, rule and SA deletion recorded as an owner follow-up (not a wait) | H | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row + the MI table's MI-14 row: its WIF-spike item, the way spk-01/spk-02 record MI-10).
> Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] **The WIF go-ahead is this launch** (D23, D40). D23's go-ahead covers P0–P3 and the image volume; WIF is its own optional yes ([rollout §13](../rollout-plan.md#13-open-owner-items)), given by launching this prompt. Record it under `ev-spike-goahead`
- [ ] A throwaway k3s is up (started before launch): the [spk-01](sprint-spk-01.md) VM **`xl-spike`**, handed over by [spk-02](sprint-spk-02.md)'s teardown when both run in the spike week — spk-01 and spk-02 leave its purge to this sprint, the last user — **or** a fresh multipass VM **`xlearn-wif`** (≈ 20 min, task 1's commands)
- [ ] The owner's Console objects exist (task 2, done before launch) and the launch message carries their non-secret ids. If they don't, run the tasks that need no Console object (1, 3 and 6), record Q-W3/Q-W4 ⛔ "Console objects missing" in `status.md`, and land the partial results (D40)
- [ ] *(soft)* MI-1 owner hygiene done — 2FA on the Anthropic account ([rollout §2](../rollout-plan.md#2-mi-infra-track), MI-1), read from `status.md`

## Goal

Decide the **platform-AI credential** before M4 ([ADR-0031 §2](../../adr/0031-platform-ai-and-two-tier-keys.md#2-credential-and-egress),
[t5 §3](../research/t5-platform-ai.md#3-where-the-platform-key-lives-and-secrets)): can judge exchange a k3s
**projected service-account token** for a short-lived Anthropic token (scope `workspace:inference`, one
workspace) through **Workload Identity Federation with an inline JWKS**, and on what terms? Four questions, each
answered with a number:

| Q | Question | Expected (inferred — verify) |
|---|---|---|
| **Q-W1** | Do k3s `v1.36.4+k3s1` projected tokens carry a `jti`? Which claims (`iss`, `sub`, `aud`, `iat`, `exp`, `kubernetes.io`)? | `jti` present (ServiceAccountTokenJTI is GA on 1.36) |
| **Q-W2** | How often does the kubelet rotate the file for `expirationSeconds: 3600` (and 600)? Is the Anthropic access-token lifetime ≥ the rotation interval, so every re-exchange has a fresh `jti`? | rotation at ≈ 80 % of TTL (≈ 48 min for 3600 s) |
| **Q-W3** | Does an **in-place container restart** (same pod) re-present an already-used `jti` and get `jti_reused`? If so, does `check_jti=false` on the one-rule issuer fix it? | same file after restart → `jti_reused` → `check_jti=false` |
| **Q-W4** | End-to-end refresh: token exchange + the first Messages call, p50/max over 5 runs; does `anthropic-workspace-id` match? | ≪ 5 s; header matches |

**Outcome:** **WIF GO** — with the `check_jti` value, the re-exchange rule (re-exchange only when the file's
`iat` changes) and the exact exchange request shape for [m4-01](sprint-m4-01.md) — **or the fallback**: a
single-workspace service-account key with a **90-day provider-side expiry** (ADR-0031 §2 break-glass),
created only in [mi-12](sprint-mi-12.md). ADR-0031 stays **Proposed** here; [mi-12](sprint-mi-12.md) accepts it
with this result (BP2).

## Scope

**In**
- A throwaway k3s (the spk-01 multipass VM `xl-spike`, or a fresh `xlearn-wif`) running the **same k3s version
  as production**.
- A throwaway Anthropic workspace with its own WIF issuer (inline JWKS from the VM), rule and service account,
  created by the owner before launch and deleted by the owner afterwards (a recorded follow-up, D40).
- Q-W1…Q-W4, two side checks (task 6), and a one-page result.

**Out**
- The production issuer, rule, service account, workspace `xlearn-platform-prod`, spend limits, judge's 443
  egress, the `xlearn-judge-llm` secret and the projected-token values → [mi-12](sprint-mi-12.md).
- Accepting ADR-0031 → [mi-12](sprint-mi-12.md) (its task 1).
- The WIF client code (`internal/platform/llm/auth`, raw-HTTP `POST /v1/oauth/token`) → [m4-01](sprint-m4-01.md).
- The sandbox spike P0–P3 and the image-volume spike → [spk-01](sprint-spk-01.md), [spk-02](sprint-spk-02.md).
- **Anything on the VPS or the production cluster.** The only production read allowed is the issuer string,
  already recorded below (read-only, 2026-09-24).

## Tasks

**The VM, `<vm>` below:** `xl-spike` (spk-01's, handed over by spk-02: `multipass start xl-spike`) or, when this
sprint runs outside the spike week or `xl-spike` is gone, a fresh `xlearn-wif`. The owner starts it before launch,
because task 2's issuer needs its JWKS; the launch message names it. **Run every `kubectl`, `curl`
and `systemctl` inside it** (`multipass exec <vm> -- sudo k3s kubectl …`). The laptop's kubectl context and
`127.0.0.1:6443` tunnel to **production** ([spk-01](sprint-spk-01.md)), so `KUBECONFIG` stays unset.

### 1 · Throwaway k3s + probe pods [H]

- **VM:** up since before launch (task 2); these are the commands the owner used, and a stopped VM restarts the
  same way. `multipass start xl-spike` if it is still there (it already runs production's `v1.36.4+k3s1`); else
  `multipass launch 24.04 --name xlearn-wif --cpus 2 --memory 4G --disk 10G` and install k3s pinned to
  production's `v1.36.4+k3s1` (the `INSTALL_K3S_VERSION` pin from `../infra/hack/host-bootstrap.sh`). No sandbox
  host files are needed. arm64 vs amd64 does not matter here: token issuance and rotation are Kubernetes
  behaviour, not CPU behaviour.
- Print `multipass exec <vm> -- sudo k3s kubectl get --raw /.well-known/openid-configuration` (→ the issuer for
  task 2) and `… get --raw /openid/v1/jwks` (→ the inline JWKS for task 2; public keys only).
- Apply (to the **throwaway** cluster only, through `multipass exec <vm>`) a namespace `wif-spike`, a
  ServiceAccount `wif-probe` (`automountServiceAccountToken: false`) and **two** probe pods that differ only in
  TTL. Token issuance doesn't depend on the Anthropic side, so the pods can run before task 2:

```yaml
apiVersion: v1
kind: Pod
metadata: { name: wif-probe-3600, namespace: wif-spike }   # and wif-probe-600 with expirationSeconds: 600
spec:
  serviceAccountName: wif-probe
  automountServiceAccountToken: false          # as judge will run (mi-12)
  restartPolicy: Always
  containers:
    - name: probe
      image: alpine:3                          # throwaway; curl + jq added at start
      # PID 1 is sh with a TERM trap, so `kill 1` restarts the container in place (task 5b). A bare
      # `exec sleep infinity` as PID 1 installs no handler, and the kernel drops signals sent to a
      # PID-namespace init from inside the namespace, so `kill 1` would be a no-op.
      command: ["sh", "-c", "apk add --no-cache curl jq >/dev/null; trap 'exit 1' TERM; while :; do sleep 1; done"]
      volumeMounts:
        - { name: anthropic-token, mountPath: /var/run/secrets/anthropic.com, readOnly: true }
  volumes:
    - name: anthropic-token
      projected:
        sources:
          - serviceAccountToken:
              audience: https://api.anthropic.com
              expirationSeconds: 3600            # 600 on the short pod (the minimum), to see rotations quickly
              path: token
```

- **Never** set `ANTHROPIC_API_KEY` anywhere in the VM or pods (it would shadow federation in any SDK; the spike
  uses raw `curl`).

### 2 · Throwaway workspace + WIF issuer, rule and service account [O, before launch]

**Before launch** (D40: Console work is owner-only, so it isn't a mid-run wait), the owner starts the VM and prints
its public JWKS with task 1's commands (`multipass start xl-spike`, or launch `xlearn-wif` and install the pinned
k3s; then `multipass exec <vm> -- sudo k3s kubectl get --raw /openid/v1/jwks`). Then, in the Anthropic Console
(the agent never holds Console credentials):
- In the org production will use (the dedicated xLearn org if it already exists, else the current one — record
  which; WIF availability can differ by org or tier), create workspace **`xlearn-wif-spike`** with the
  **lowest monthly spend limit** the Console allows and auto-reload off. The spike makes a handful of
  ~16-token calls (< $0.05).
- Create service account `xlearn-wif-spike` (member of that workspace only).
- Create a WIF **issuer**: issuer URL = the VM's `iss` from task 1 (k3s' default is
  `https://kubernetes.default.svc.cluster.local`, the **same string production uses** — verified read-only
  2026-09-24 via `get --raw /.well-known/openid-configuration`), `jwks.type: inline` with task 1's
  `/openid/v1/jwks` JSON.
- Create a **rule** mirroring production's shape ([t5 §3](../research/t5-platform-ai.md#3-where-the-platform-key-lives-and-secrets)):
  `subject_prefix: system:serviceaccount:wif-spike:wif-probe` (no `*`), `audience: https://api.anthropic.com`,
  `oauth_scope: workspace:inference`, `token_lifetime_seconds: 3600`, the one workspace. Leave `check_jti` at
  its default for now (task 5 changes it).
- Hand the agent the **non-secret identifiers** the exchange needs (org, workspace, service-account and
  federation-rule ids) in the launch message, with the VM's name. No API key is created.

### 3 · Q-W1 / Q-W2: claims and rotation [H]

- Decode the payload of `/var/run/secrets/anthropic.com/token` (base64url, no signature check needed) and
  record the claim **names** and the non-secret values: `iss`, `sub`, `aud`, `iat`, `exp`, `nbf`, **`jti`
  present?**, the `kubernetes.io` block (pod, SA, node). Never paste a whole token anywhere.
- Rotation: sample `iat`, `exp` and `jti` of both pods' files every 60 s (a loop on the VM host over
  `k3s kubectl exec … cat`). Keep the 600 s pod ≥ 20 min (expect ≥ 2 rotations) and the 3600 s pod ≥ 60 min if
  the ½-day box allows (expect one at ≈ 48 min). Record the observed rotation offsets as a % of TTL.
- Record the exchange response's access-token lifetime (`expires_in`) next to the rotation interval: the
  invariant for [m4-01](sprint-m4-01.md) is **access-token lifetime > rotation interval**, so re-exchanging on
  each new `iat` never leaves judge without a valid token and never needs a used `jti`.

### 4 · Q-W4: exchange + first Messages call, timed [H]

- From inside `wif-probe-3600`: `POST https://api.anthropic.com/v1/oauth/token` with the projected token as the
  assertion, using **exactly the request shape the WIF reference documents** (grant type, assertion, and the
  org / service-account / federation-rule identifiers it requires —
  [WIF with Kubernetes](https://platform.claude.com/docs/en/manage-claude/wif-providers/kubernetes),
  [WIF reference](https://platform.claude.com/docs/en/manage-claude/wif-reference)). Record the **field names
  and endpoint** (never the values of the token or the access token) for m4-01's raw-HTTP client.
- With the access token: one minimal `POST /v1/messages` (`claude-sonnet-5`, `max_tokens` ≈ 16). Record the
  response header **`anthropic-workspace-id`** (must equal `xlearn-wif-spike`'s id — judge pins it, ADR-0031 §2).
- Time both with `curl -w '%{time_total}'`, 5 runs (a fresh pod per exchange, or wait for a rotation — see
  task 5 on reuse): p50 and max for the exchange and for the first call.
- **Scope check (cheap):** a Files or Batches API request with the same access token → expect **403**
  (`workspace:inference` blocks them, t5 §3). Record the status only.

### 5 · Q-W3: `jti` reuse and the in-place restart → `check_jti` decision [H]

- **(a) Reuse:** exchange twice with the *same* file content → record the second response (expect
  `jti_reused` with `check_jti` on).
- **(b) In-place restart:** `multipass exec <vm> -- sudo k3s kubectl -n wif-spike exec wif-probe-3600 -- kill 1`.
  PID 1 is the trapping `sh` (task 1), so the container exits and restarts **inside the same pod**. If `kill 1`
  doesn't take, stop the container from the VM host instead:
  `sudo k3s crictl ps --name probe --label io.kubernetes.pod.name=wif-probe-3600` → `sudo k3s crictl stop <id>`
  (restartPolicy Always restarts it in the same pod). Check that restartCount went up by **exactly 1**
  (`… get pod wif-probe-3600 -o jsonpath='{.status.containerStatuses[0].restartCount}'`) and that the token
  file's `jti`/`iat` are **unchanged**. Then exchange again (this is what judge does on boot) and record the
  result.
- **(c) Pod recreate:** delete and recreate the pod → new token, fresh `jti` → exchange succeeds (control).
- **Decision:** if (a) rejects and (b) re-presents a used `jti`, the decision is **`check_jti=false`** on the
  one-rule issuer, the pre-decided path. This is acceptable per
  [t5 §3](../research/t5-platform-ai.md#3-where-the-platform-key-lives-and-secrets): the token is
  audience-bound and lives ≤ 1 h. Setting the flag is a Console action, so the session doesn't wait for it (D40):
  it records the decision, and the confirming re-run of (a)/(b) under `check_jti=false` is handed to
  [mi-12](sprint-mi-12.md), which creates the production issuer with the flag (`status.md` → Hand-offs). Record
  the alternative considered and not taken (cache the access token in a memory-backed `emptyDir`, which survives
  container restarts) for mi-12 to confirm.

### 6 · Side checks [H]

- **Audience binding:** present the projected token to the **throwaway** API server from inside the probe pod,
  so the token never leaves it:
  `multipass exec <vm> -- sudo k3s kubectl -n wif-spike exec wif-probe-600 -- sh -c 'curl -sk -o /dev/null -w "%{http_code}\n" -H "Authorization: Bearer $(cat /var/run/secrets/anthropic.com/token)" https://kubernetes.default.svc/api'`
  → expect **401**, so a leaked judge token cannot be used against the cluster. **Never** run this against
  `127.0.0.1:6443` on the laptop: that is the production tunnel.
- **JWKS stability** (only if time allows, and only after tasks 3–5 are done, because a changed key breaks
  further exchanges): restarting k3s and rotating certificates is disruptive, so run it only on a VM no other
  spike still needs, meaning `xl-spike` after spk-02's teardown has handed it over (its status.md teardown
  line), or a fresh `xlearn-wif`. If spk-02 still needs `xl-spike`, skip this check or run it on a fresh
  `xlearn-wif`. Inside the VM, compare `get --raw /openid/v1/jwks` before and after
  `multipass exec <vm> -- sudo systemctl restart k3s`, and after `sudo k3s certificate rotate` + restart. Record which operations change it
  (the signing key is `/var/lib/rancher/k3s/server/tls/service.key`). This feeds the mi-12 runbook line
  "re-paste the JWKS at k3s SA-key rotation" and the DR runbook: a **rebuilt** node gets a new key unless that
  file is restored, so WIF 401s until the re-paste (breaker → manual grading).

### 7 · Report [X]

A docs PR in xlearn (the only thing this sprint merges):
- **`docs/v2/research/t5-platform-ai.md` — new `## 15. WIF spike result (spk-03, <date>)`** (precedent:
  t3 §15 "S0 results"): a results table (Q-W1…Q-W4 with numbers, the two side checks), the **decision**
  (WIF GO + `check_jti` value + re-exchange rule, or fallback), the exact exchange request shape (field names
  only), the TTL / lifetime values mi-12 sets (`expirationSeconds: 3600`, `token_lifetime_seconds`), the org
  used, and "throwaway resources deleted on <date>".
- **`docs/v2/status.md`:** the MI table's **MI-14** row, WIF-spike item (spk-03: GO / fallback, date, link to
  t5 §15; MI-14 itself stays open until [mi-12](sprint-mi-12.md)), the Sprint
  board row, a Decisions-log line, `ev-spike-goahead` (WIF: the launch, D40), the `check_jti` re-run hand-off to
  mi-12 if the flag is needed, and the owner follow-up from task 8.
- ADR-0031 is **not** edited here — [mi-12](sprint-mi-12.md) folds this result in and accepts it.

### 8 · Tear down [H]

- **Owner follow-up (recorded, not a wait):** the owner deletes the rule, the issuer, the service account and the
  workspace `xlearn-wif-spike` in the Console. The issuer shares production's default `iss` string, so it **must**
  be gone before mi-12 creates the production issuer: record it in `status.md` → Open owner items (mi-12's entry
  checks it is gone).
- **H:** delete the `wif-spike` namespace, then `multipass delete --purge <vm>`. spk-03 is **`xl-spike`'s last
  user**: spk-01 hands it on and spk-02's teardown hands it to this sprint ("spk-03 deletes it"), so purge it
  here. It also still holds spk-02's evalpack pull secret. If this sprint ran on a fresh `xlearn-wif` instead,
  purge that one, and check `multipass list` that `xl-spike` is already gone (spk-02 purges it when spk-03 isn't
  in the spike week). Record the purge in `status.md`.
- Confirm no token, access token or identifier-bearing log is left in any file that could be committed.

## Acceptance criteria

- [ ] Q-W1…Q-W4 answered with numbers: `jti` present or not; rotation offset (% of TTL) for 600 s and 3600 s;
      reuse and in-place-restart behaviour; exchange and first-call p50/max; workspace header matches.
- [ ] **WIF GO** — with the `check_jti` decision, the re-exchange rule and the lifetime invariant — **or the
      fallback chosen** (90-day single-workspace key, created in mi-12); recorded in t5 §15 and `status.md`.
- [ ] The exact exchange request shape (endpoint + field names, no secrets) is in t5 §15 for m4-01.
- [ ] Probe namespace and VM gone; no token or access token in any committed file; the deletion of the throwaway
      workspace, issuer, rule and service account recorded as an owner follow-up (mi-12's entry checks it).

## Release

**No merge (spike, throwaway).** The VM, manifests and probe scripts live only in the session scratchpad and
the VM, and are never committed. The **results docs PR** (t5 §15 + `status.md` + this file's statuses) is the
only merge — squash-merged on CI green, with no owner stop (D40; docs only; no tag, no deploy). ADR-0031 moves
to Accepted in [mi-12](sprint-mi-12.md), not here.

## Definition of Done

Every question answered with numbers · GO or fallback decided and recorded (t5 §15, `status.md`) · the VM and
namespace deleted, and the Console objects' deletion recorded as an owner follow-up · nothing from the spike
committed except the results docs PR · statuses updated (this file + [`../status.md`](../status.md)).

## Risks / watch-outs

- **It may not fit the spike week** (spk-01 hard-stops at 1.5 days; spk-02 runs Thu–Fri). Then run it any day
  before M4 on a fresh VM — it needs nothing from production, and only mi-12's JWKS paste needs the cluster
  ([ADR-0031 §8](../../adr/0031-platform-ai-and-two-tier-keys.md#8-amended-by-t7-2026-09-24)).
- **The laptop points at production.** Its kubectl context and `127.0.0.1:6443` tunnel to the VPS, so a
  `kubectl` or `curl` typed on the laptop hits production. Every command goes through `multipass exec <vm>`.
- **Same issuer string as production.** k3s' default `iss` is identical on every cluster; a leftover
  throwaway issuer could collide with mi-12's. The owner deletes it after this sprint (task 8's recorded
  follow-up); mi-12's entry checks it is gone.
- **Tokens are secrets for up to an hour.** Decode claims, never paste a whole token or access token into the
  transcript, a doc or a PR.
- **Org / tier differences.** A new dedicated org may start in the Evaluation tier; if the spike runs in the
  current org, note it — mi-12 re-checks WIF availability in the org production uses.
- **`check_jti=false` is a deliberate weakening** (a stolen token is replayable until it expires, ≤ 1 h). It is
  scoped to the one-rule issuer and is the t5-accepted trade; the rejected alternative (emptyDir cache) is
  recorded for mi-12.
- **Inline JWKS never auto-refreshes.** Any k3s SA-key rotation or node rebuild breaks exchanges (401 → breaker
  → manual grading) until the re-paste — the task-6 result makes that runbook line concrete.

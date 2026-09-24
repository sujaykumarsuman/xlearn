# Prompt — Sprint mi-11 · Track B finish: xlearn egress, PG connection limits, Renovate, N4 (MI-15)

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-mi-11.md`](../sprints/sprint-mi-11.md)   ·   **Milestone:** MI (rollout step MI-15, second half)   ·   **Prereqs:** [m3-07](../sprints/sprint-m3-07.md), [l-01](../sprints/sprint-l-01.md) (also [mi-03](../sprints/sprint-mi-03.md), [mi-06](../sprints/sprint-mi-06.md))

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md): repo conventions and the land-and-sync rule.
- [`../sprints/sprint-mi-11.md`](../sprints/sprint-mi-11.md): the plan, including the caller matrix and destination table (task 1), the three egress batches and their smoke tests (task 2), and the N4 rehearsal and fallback (task 7).
- [`../rollout-plan.md`](../rollout-plan.md):
  - §2, the MI-15 row;
  - §2.2, the operating rules;
  - §12, the downstream constraints (its "M4 adds only 443" line stands: judge → identity :8081 moved forward to m3-07).
- [ADR-0035](../../adr/0035-v2-operations-nats-auth-limits-capacity.md):
  - §2, the NATS model, the N1–N4 table and the NetworkPolicy standing rule. Its sentence says only "updates the policy"; every tag sprint's release checklist words it as ingress **and** egress, and that's the reading this sprint makes explicit;
  - §3, D34, no alerting;
  - §4, L21 (PG pools and role limits);
  - §6, the amendments.
- [ADR-0030](../../adr/0030-runner-technology-and-host-hardening.md): §5 Track B; §6, the patch cadence (D22: monthly reboot, monthly k3s patch, Renovate).
- [ADR-0034](../../adr/0034-v2-release-labelling-gating-and-rollback.md): §4.3, the snapshot rule (no off-node dump); §4.4, the NetworkPolicy and NATS rows.
- The research:
  - [t3 §8.4, §8.6, §8.8](../research/t3-sandbox.md): v1 policies, Track B, the rollout order;
  - [t7](../research/t7-cross-cutting-and-rollout.md): the C3 row, the N4 reload behaviour on 2.10.22.
- The neighbour plans:
  - [mi-03](../sprints/sprint-mi-03.md), the MI-5/5a caller list (reuse its evidence);
  - [mi-06](../sprints/sprint-mi-06.md), the N1–N3 config;
  - [mi-01](../sprints/sprint-mi-01.md), `networkPolicy.egress` semantics;
  - [mi-02](../sprints/sprint-mi-02.md), the `--nats-stage` flag and `expected-netpol.tsv` (plus its embedded copy in `host-verify.sh`);
  - [mi-09](../sprints/sprint-mi-09.md), `--with-sandbox`/`--expect-sandbox` and the `pg_dumpall` question;
  - [mi-12](../sprints/sprint-mi-12.md), judge's 443 except-list (task 4); don't touch judge;
  - [mi-13](../sprints/sprint-mi-13.md), coach WSS;
  - [m3-08](../sprints/sprint-m3-08.md), [m3-09](../sprints/sprint-m3-09.md), [m3-13](../sprints/sprint-m3-13.md): the future judge callers;
  - **the matrix hand-offs** (each of these sprints recorded its hand-off in status.md):
    [m2-01](../sprints/sprint-m2-01.md) task 8 (practice → review :8084, `REVIEW_BASE_URL`, touch start);
    [l-01](../sprints/sprint-l-01.md) (practice, assessment, coach → identity :8081 for the erase re-verify; coach → NATS);
    [m3-05](../sprints/sprint-m3-05.md) and [m3-07](../sprints/sprint-m3-07.md) (judge → identity :8081, already in judge's policy);
  - [l-02](../sprints/sprint-l-02.md), the `identity admin account erase` and `identity admin erasures` CLI verbs used by the erase smoke.
- xlearn code: `internal/*/config.go` (env reads, outbound clients); `internal/identity/oauth.go`; `internal/coach/config.go`; `deploy/*.Dockerfile`, including `deploy/runner.Dockerfile` (m3-15).
- `../infra`, all read before editing:
  - `apps/xlearn-*.yaml`: env, the MI-5a `networkPolicy` values;
  - `infrastructure/database/cluster/cluster.yaml`: managed roles;
  - `infrastructure/messaging/release.yaml`: the N3 authorization block;
  - `hack/host-verify.sh`, `hack/expected-netpol.tsv`;
  - README: "Kernel reboot runbook", "Rebuild order";
  - `infrastructure/**/repository.yaml` and `release.yaml`, `infrastructure/controllers/*.yaml`: the HelmRepository sources and chart pins Renovate's `flux` manager reads.

## Context

- MI-15 is Track B hygiene. It doesn't gate M3, but it belongs in v2.0 before GA.
- mi-08 already landed the PSA labels and SA tokens off. This sprint finishes the rest:
  - egress NetworkPolicies for the seven v1 `xlearn` services. judge was born with its own policy in m3-07;
  - per-role Postgres connection limits;
  - Renovate for the D22 pins;
  - **N4**, which removes the NATS `legacy` bridge and `no_auth_user`, so `auth_required: true`;
  - the monthly-window runbook.
- Everything is fail-open on revert except N4, whose revert needs a restart.
- The main risk is **cutting a caller silently**. Build the matrix from evidence, and smoke after every batch.
- Facts from 2026-09-24:
  - only identity (GitHub OAuth) and coach (OpenAI/Anthropic) dial external hosts;
  - selectors: kube-dns is `k8s-app: kube-dns`, NATS pods are `app.kubernetes.io/name: nats`, CNPG pods are
    `cnpg.io/cluster: projects-pgstore`;
  - the pod CIDR is under `10.42.0.0/16` and the service CIDR is `10.43.0.0/16`;
  - xlearn login roles have `connectionLimit: -1`.
- Owner time: ~5 min for the Renovate app (step 8) and ~10 min for the erase smoke (steps 2–3). Book it first.
- Each step below is tagged with the plan task it ticks in the plan's Status table.

## Entry gates — verify first (stop and report if any is unmet)

- [ ] status.md records l-01's N3 ≥ 24 h re-check (no `legacy` connection).
- [ ] `host-verify --cluster --nats-stage=n3`: `/connz?auth=true` shows every client on its own nkey, judge and coach included, and zero on `legacy`.
- [ ] MI-5a ingress is live on every `xlearn-*` release, and chart 0.3.0's `networkPolicy.egress` is available.
- [ ] The last Hostinger weekly image is ≤ 7 days old.
- [ ] The owner is booked for the erase smoke and the Renovate app install.
- [ ] Parallel sessions: no open peer PR touches `apps/xlearn-*.yaml`, `cluster.yaml` or `messaging/release.yaml` (`gh pr list -R sujaykumarsuman/infra`, `git worktree list`, ListAgents). mi-12 may be editing `apps/xlearn-judge.yaml`; you don't touch that file.

## Do this (in order)

1. **[H] Caller matrix** (plan task 1). Build it from:
   - the live `apps/xlearn-*.yaml` env (`*_BASE_URL`, `JWKS_URL`, `NATS_URL`, `PG*`);
   - `main`'s config/env reads and outbound clients. Callers merged but not yet tagged get
     **forward-declared**: gateway → judge 8087 (m3-09/m3-13) and practice → judge 8087 (m3-08);
   - `/connz?auth=true`;
   - the MI-5a ingress values;
   - the hand-offs in status.md: **practice → review 8084** (m2-01), **practice, assessment and coach →
     identity 8081** (l-01's erase re-verify; review already has it), coach → NATS (l-01). judge → identity
     8081 is already in judge's own policy (m3-07); leave it there.

   Fill the plan's table and **verify every cell**. Then read the destinations live: the kube-dns labels,
   the NATS and CNPG labels, the CIDRs, and the node IP (`k3s kubectl get node -o wide`). The 443 rule is
   `0.0.0.0/0` except `10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`, `100.64.0.0/10`, `169.254.0.0/16`
   and the node `/32`: the same except-list as mi-12's judge rule (its task 4).

   **Erase smoke prep [O]:** before PR a merges, the owner mints two throwaway testers with the CLI
   (`identity admin account create --role tester --email …`, as in ev-first-tester).

2. **[I] Egress PR a** (plan task 2): curriculum, assessment, review, practice. Set `networkPolicy.egress` for DNS UDP+TCP
   53, PG 5432 and NATS 4222 where used, and the in-namespace targets from the matrix. In the same PR, bump a
   `podAnnotations` key on review so Flux rolls it.
   - After the merge, check:
     - the dashboard and week views;
     - a review background worker run (identity + curriculum);
     - outboxes drain;
     - review re-fetches JWKS, and an authed call through it works (the mistakes page);
     - **a touch start** (practice → review 8084): start a due touch if the owner has one; otherwise
       `POST …/touches/<random UUID>/start` must answer 404, never 503 `touches_unavailable`;
     - **erase round trip #1 [O]**: the owner erases one throwaway tester with
       `identity admin account erase <email> --confirm <email>`. Within a few minutes, a read-only
       `identity admin erasures --since <today>` must show the request **closed with every expected ack**
       (5: practice, review, assessment, coach, judge), with no new `event_dead_letter` rows. A request still
       open means a consumer can't reach identity 8081.
   - If v1.14.0 is live, smoke the practice reconciler (a judged submit in the cohort). If not, record that
     the rule is forward-declared.
   - Run `host-verify --cluster`. On any failure: `git revert`.

3. **[I] Egress PR b** (plan task 2): identity and coach, each with TCP 443 (the except-list above).
   - After the merge:
     - a **GitHub OAuth login round trip**;
     - an email login;
     - a coach chat that streams;
     - identity's outbox drains to `XLEARN_IDENTITY`;
     - coach is on `/connz` with its nkey;
     - **erase round trip #2 [O]** with the second throwaway tester, with the same pass rule as #1.
   - Log the `kubectl exec` uses and both erases in status.md.
   - Run `host-verify --cluster`.

4. **[I] Egress PR c** (plan task 2): gateway (every service port, plus the forward-declared judge 8087).
   - After the merge: login, the dashboard, every screen, `/xlearn/u/<owner>`, coach SSE.
   - Run `host-verify --cluster`.
   - Only if an object name changed: update `hack/expected-netpol.tsv` **and** its byte-identical embedded
     copy (between the `# >>> expected-netpol.tsv` / `# <<< expected-netpol.tsv` markers in
     `hack/host-verify.sh`) in the same PR, then run `hack/host-lint.sh`.
   - Each PR body lists the known upcoming callers that will need egress + ingress rules before their tags:
     gateway → judge (m3-13), practice → judge (v1.14.0), judge TCP 443 (mi-12, judge's own policy),
     coach WSS (mi-13).

5. **[I] PG role limits** (plan task 3). In `infrastructure/database/cluster/cluster.yaml`, set `connectionLimit: 20` on
   `xlearn`, `xlearn_identity`, `_curriculum`, `_practice`, `_review`, `_assessment`, `_coach` and `_judge`.
   Merge it, then check the Cluster's `.status.managedRolesStatus`. There's no restart.

6. **[X] Renovate (xlearn)** (plan task 4). Add `renovate.json`:
   - the `dockerfile` manager (`pinDigests`, grouped);
   - `gomod` enabled only for the go-sandbox pin;
   - a custom regex manager for `deploy/runner.Dockerfile`'s `ARG *_VERSION` pins (label `runner`);
   - everything else disabled;
   - `automerge: false`, `platformAutomerge: false`, label `deps`, weekly, a low `prConcurrentLimit`;
   - `vulnerabilityAlerts: {enabled: false}` and `osvVulnerabilityAlerts: false` (D34; vulnerability PRs
     are on by default and bypass the schedule).

   `renovate-config-validator` must pass.

7. **[I] Renovate (infra)** (plan task 5). Add `renovate.json`:
   - the `flux` manager with `managerFilePatterns: ["infrastructure/**/*.yaml"]` (by default it matches only
     `gotk-components.yaml`), for the chart versions (label `restart`);
   - a regex manager for `PIN_K3S_VERSION` in **both** host scripts as one PR (patch only, no prereleases);
   - a regex manager for the CNPG `imageName` (minor only);
   - `ignorePaths`: `apps/**`, `runner/**`, `clusters/vps/flux-system/**`, `charts/project/**`;
   - no automerge; `vulnerabilityAlerts: {enabled: false}`, `osvVulnerabilityAlerts: false`.

   The validator must pass. Merge it.

8. **[O] Owner installs the Renovate app** (plan task 6) on `xlearn` and `infra` only (~5 min). Wait for
   Renovate to read the configs and open at least one PR. Confirm none merges, and that the infra run lists
   the four charts (Longhorn, cert-manager, the CNPG operator, NATS). If the owner declines, record that and
   leave the configs dormant.

9. **[I] N4** (plan task 7).
   - **Local rehearsal.** In a `nats:<version from /varz>` container, run with the live N3 config, connect a
     client, apply N4, then `--signal reload`. Record whether it applied, and whether
     `auth_required: true`. Then rehearse the revert, which is expected to need a restart.
   - **PR:** in `infrastructure/messaging/release.yaml`, remove the `legacy` user **and** `no_auth_user` in
     one commit. In the same PR, set `PIN_NATS_STAGE=n4` in `hack/host-verify.sh` (mi-02's constant), so a
     plain `--cluster` run checks the live stage. Merge it.
   - **Verify:** `host-verify --cluster --nats-stage=n4` (`auth_required: true`); all six nkeys on
     `/connz?auth=true`; outbox `unsent` 0; consumer pending 0; no permission-violation logs.
   - **If the reload is refused:** revert at once (git matches the running config again), and prepare the
     N4 + pod-template annotation bump PR for the monthly window.

10. **[X] Monthly-window runbook** (plan task 8) `docs/v2/runbooks/monthly-window.md`, with the contents of
    the plan's task 8:
    - the pre-checks: the weekly image; **a manual snapshot before every monthly reboot**, and the VNC
      console open (the infra README kernel runbook's step 1, kept); no off-node dump (the only part of that
      step 1 skipped, pending the owner's answer);
    - the kernel steps (`host-bootstrap.sh --with-sandbox`), linking the infra README kernel runbook steps 3–5;
    - `host-verify --cluster --expect-sandbox --nats-stage=<live>`;
    - the monthly k3s patch via Renovate's pin PR and `--with-sandbox --with-k3s --replace-k3s`;
    - restart-inducing Renovate PRs, one at a time;
    - the N4 fallback (a PR from the branch status.md records);
    - the on-demand reads (PAT expiry, TR-* lines, Renovate PRs);
    - a log line.

    Record the owner's answer to mi-09's `pg_dumpall` question. If the owner confirms, drop the dump from the
    infra README step 1 in a small infra PR, keeping the snapshot and VNC parts.

11. **[H] Verify** (plan task 9). `host-verify --cluster --nats-stage=n4` (or `n3` if N4 was deferred) is
    green with the expected-policy list (`.tsv` and its embedded copy in sync, `host-lint.sh` green). The memory
    sum is unchanged. No unexpected restarts. Renovate PRs are open and none is merged.

12. **[X] Record** (plan task 10). Update status.md (see Update status), then open the xlearn PR (`renovate.json` +
    runbook + status) and merge it.

## Constraints

- **GitOps:** never `kubectl apply`, never `kubectl rollout restart`. A restart is a `podAnnotations` bump in
  git. Every egress PR reverts cleanly with `git revert` (fail-open).
- **Standing rule**, as every tag sprint's release checklist carries it (it extends the
  [ADR-0035 §2](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#2-nats-auth-nkey-users-fine-acls-server-first)
  sentence, which says only "updates the policy"): from now on, every new in-cluster HTTP or NATS caller
  needs its **ingress and egress** change in its own infra PR, merged before its tag. The ACL PR still goes before the consuming tag, and consumers still ship
  before producers.
- **Don't touch judge's policy** (m3-07/mi-12), the `xlearn-runner` namespace (mi-14), or PSA and tokens (mi-08).
- **N4 is all or nothing:** `legacy` and `no_auth_user` are removed together. If the reload is refused,
  revert immediately. Git must never differ from the running server.
- **Renovate is PRs only:** no automerge, and it never touches Flux-managed tags (`apps/**`, `runner/**`).
  Restart-inducing bumps merge only in a window.
- **D34, no alerting:** no Flux Alert, CronJob, timer, push channel, healthchecks.io, or Renovate
  vulnerability alerting (`vulnerabilityAlerts.enabled: false` in both configs, OSV alerts off). `host-verify`
  stays on demand.
- **Erases are final** (D12, no backups): the erase smoke uses only the two throwaway testers the owner
  minted for it. Never erase the owner or any real account, and never send `DELETE /api/me` from the owner's
  session.
- **The memory-sum rule:** no new pod here. If anything adds one, check it against
  [ADR-0035 §5](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#5-capacity-the-memory-sum-rule-triggers-and-ordered-responses).
- **Service boundaries** ([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)): PG egress goes only
  to the DB-owning services. The gateway gets none.
- **Not applicable, since there's no service code here:** goose + sqlc (`sqlc diff`), outbox/inbox changes,
  and `theme.css`. The only xlearn changes are `renovate.json`, the runbook and status. If you find yourself
  editing `internal/`, `cmd/` or `web/`, stop.
- **Parallel sessions:** re-check peers' infra PRs (especially mi-12 on judge) and worktrees right before
  each merge, and rebase first. The Flux image bots commit to infra `main` often. No ADR number is needed;
  if you write one, check peers' numbers first.
- Conventional commits with the required attribution lines, and squash merges. Don't enable auto-merge.

## Deliverables

- infra PRs (merged in order): egress a, b and c; PG role limits; `renovate.json`; N4 (or N4 reverted, with
  the monthly-window PR prepared); the README alignment if the owner confirmed it.
- xlearn PR (merged, no tag): `renovate.json`, `docs/v2/runbooks/monthly-window.md`, the status rows and this
  sprint's Status table.
- Evidence in the PR bodies: the caller matrix and its sources, the smoke results per batch (including both
  erase round trips and the touch start), the N4 local rehearsal and its live verification.

## Update status

- Set each task row in [`../sprints/sprint-mi-11.md`](../sprints/sprint-mi-11.md) to 🔄 or ✅ as you go (each step above names its plan task; steps 2–4 all tick task 2), and _Overall_ to ✅ when all ten plan tasks are done.
- Mirror it in [`../status.md`](../status.md):
  - the **Sprint board** row;
  - the **MI track** MI-15 row ✅ (with mi-08's slice);
  - the **NATS** rows: the N4 date, or "deferred to the monthly window <date>";
  - the **MI milestone** row: set it ✅ when its exit criteria hold (they may already hold since mi-10).
- Add **Decisions log** lines for:
  - the egress matrix (a link to the PR);
  - the forward-declared callers;
  - `connectionLimit` 20 including the owner role;
  - Renovate's scope and `ignorePaths`;
  - the N4 reload outcome on 2.14.6;
  - the owner's `pg_dumpall` answer.
- Add the erase log lines (the two throwaway-tester erases: request id, date, via `cli`, acks) and the `kubectl exec` uses.

## Done when (acceptance)

- [ ] Egress policies present on all seven v1 releases; OAuth, coach, the JWKS re-fetch, review workers and the practice reconciler are unaffected (the reconciler if v1.14.0 is live, else forward-declared and recorded)
- [ ] Both erase round trips close with every expected ack and no new dead-letter rows (after PR a and after PR b); a touch start reaches review (never 503 `touches_unavailable`)
- [ ] `auth_required: true` and no anonymous connection possible (`--nats-stage=n4`), or N4 reverted cleanly and scheduled for the monthly window
- [ ] Renovate opens PRs and merges none automatically, in both repos
- [ ] Every xlearn login role is at `connectionLimit: 20`, with no PG restart
- [ ] `docs/v2/runbooks/monthly-window.md` merged
- [ ] `host-verify --cluster` green with the updated expected-policy list
- Ship at session end per AGENT.md land-and-sync, with this sprint's release action: **infra PR(s) only**.
  - Merge the infra PRs in order (egress a → b → c, PG limits, Renovate, N4 last), putting the smoke and check output in each PR body (infra has no CI).
  - Merge the xlearn PR (`renovate.json` + runbook + status); no tag.
  - If N4 was reverted, push its restart change as a branch (e.g. `feat/nats-n4-restart`) with **no PR**, and record the branch name in status.md. The monthly window opens and merges it, so no PR is left hanging.
  - Sync local `main` in both repos.

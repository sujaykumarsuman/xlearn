# Prompt — Sprint m3-07 · M3-1 on prod: evalpack v1.0.0, judge ACL, tag v1.13.0, judge HelmRelease (MI-13)

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-m3-07.md`](../sprints/sprint-m3-07.md)   ·   **Milestone:** M3 (M3-1 on prod; rollout step MI-13)   ·   **Prereqs:** [m3-14](../sprints/sprint-m3-14.md), [m3-02](../sprints/sprint-m3-02.md), [mi-06](../sprints/sprint-mi-06.md), [l-01](../sprints/sprint-l-01.md), [mi-03](../sprints/sprint-mi-03.md), [mi-07](../sprints/sprint-mi-07.md) (preferred: [mi-10](../sprints/sprint-mi-10.md))

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md) — conventions, land-and-sync (this sprint tags a release and opens infra PRs).
- The plan: [`../sprints/sprint-m3-07.md`](../sprints/sprint-m3-07.md) — the order, every file, the HelmRelease values table and the NetworkPolicy are spelled out there.
- [ADR-0034](../../adr/0034-v2-release-labelling-gating-and-rollback.md) **§1.4 (range order), §1.5 (judge and evalpack streams)**, §1.6 (tag timeline), §2 (kill switches), **§6 (release checklist)**.
- [ADR-0035](../../adr/0035-v2-operations-nats-auth-limits-capacity.md) §1 (topology), **§2 (nkeys, fine ACLs, reload vs restart, the standing rules)**, §3 (D34, `host-verify`), §5 (memory-sum rule).
- [ADR-0005](../../adr/0005-data-ownership-and-migrations.md) (the 4-step), [ADR-0027](../../adr/0027-content-evalpack-and-user-data-model.md) §1–§2, §5 (public/private rule, image volume), [ADR-0033](../../adr/0033-invite-only-admission-and-owner-admin.md) §12 row 10 (practice → judge `/internal/*`).
- [`../rollout-plan.md`](../rollout-plan.md) §2 (MI-9, MI-13 rows), §2.2 (operating rules), §7 (`v1.13.0`, evalpack `v1.0.0`).
- [t1 §3.3](../research/t1-content-data-model.md) (evalpack image, image volume, Ready-with-zero), [t4 §9.3](../research/t4-judge-contract.md#93-throughput-and-memory-inferred-t3-measures) (judge 128/256 Mi), [t3 §8](../research/t3-sandbox.md) (runner namespace, `judge-to-runner`).
- Peer plans this sprint builds on: [mi-01](../sprints/sprint-mi-01.md) (chart 0.3.0 knob names), [mi-02](../sprints/sprint-mi-02.md) (`host-verify`, `expected-netpol.tsv`), [mi-03](../sprints/sprint-mi-03.md) (live selectors: pods carry only `name` + `instance`), [mi-06](../sprints/sprint-mi-06.md) (nkey + seed procedure), [mi-07](../sprints/sprint-mi-07.md) (evalpack repo, pull secrets), [mi-10](../sprints/sprint-mi-10.md) (runner bearer: one value, two files), [spk-02](../sprints/sprint-spk-02.md) (image-volume result).
- Code: `internal/judge/config.go` (the env names judge reads, incl. the kill switch), `cmd/judge/main.go` (`judge admin`), `internal/platform/events/{topology.go,consumer.go}`, `cmd/identity/main.go` (consumer start order), `.github/workflows/deploy.yml` (the 8th job), `.release-line`, `Makefile` (`nats-acl-render`).
- Infra (`../infra`, via PRs only): `apps/image-automation.yaml`, `apps/xlearn-practice.yaml` (a release to mirror), `apps/secrets/`, `infrastructure/messaging/release.yaml`, `infrastructure/database/{README.md,cluster/cluster.yaml,cluster/xlearn-database.yaml}`, `charts/project/values.yaml`, `hack/{host-verify.sh,expected-netpol.tsv,host-lint.sh}`, `.sops.yaml`.
- Private: `../xlearn-evalpack` (never copy anything from it into xlearn).

## Context

M3-1 puts **judge — the 8th service — on production dark**. Its code is merged: [m3-05](../sprints/sprint-m3-05.md)
(skeleton, schema, a pack loader that is Ready with 0 evaluable rather than crash-looping, the `XLEARN_IDENTITY` erase
consumer), [m3-06](../sprints/sprint-m3-06.md) (queue, runner lane, graders, contexts, internal endpoints) and
[m3-14](../sprints/sprint-m3-14.md) (admission, learner API, arena progress, telemetry, `judge admin`). The MI fences are
live (MI-5/5a, NATS N3), the evalpack plumbing exists (MI-9), and the owner has stamped at least one pilot pack. This
session cuts the first pack, grants judge its NATS identity, tags `v1.13.0` (which builds `xlearn-judge` for the first
time) and then deploys judge through two infra PRs. Order is everything: **pack image → pack policy → judge ACL → tag →
DB 4-step → HelmRelease**. Nothing becomes learner-visible: the gateway's `JUDGE_BASE_URL` stays unset until
[m3-13](../sprints/sprint-m3-13.md). Production has one owner account (D35); there is no alerting (D34), so every check
is a read you perform.

## Entry gates — verify first (stop and report if any is unmet)

- [ ] MI-3 chart 0.3.0 merged ([mi-01](../sprints/sprint-mi-01.md)); read the NetworkPolicy/probe knob names in `../infra/charts/project/values.yaml`
- [ ] MI-5 + MI-5a live ([mi-03](../sprints/sprint-mi-03.md)); `nats-ingress` and `projects-pgstore-ingress` list `xlearn-judge`
- [ ] MI-7 N3 applied and its **≥ 24 h re-check recorded** in status.md ([mi-06](../sprints/sprint-mi-06.md), [l-01](../sprints/sprint-l-01.md) task 1)
- [ ] MI-9 done ([mi-07](../sprints/sprint-mi-07.md)): `ImageRepository xlearn-evalpack` Ready, `xlearn-evalpack-pull` in `xlearn` + `flux-system`, no ImagePolicy yet
- [ ] m3-05, m3-06, m3-14 and m3-02 merged; the `xlearn-judge` job exists in `.github/workflows/deploy.yml`
- [ ] ≥ 1 pilot pack stamped and green in the evalpack CI (ev-packs-14)
- [ ] spk-02's image-volume result recorded (GO, or the fallback recipe)
- [ ] `v1.11.0` (l-01) live; mi-10 (runner dark) preferred, not required
- [ ] Parallel sessions: no open peer PR touches the infra files above; peers' xlearn tags and PRs checked (`git ls-remote --tags origin`, `gh pr list`, `gh pr list -R sujaykumarsuman/infra`, `git worktree list`, ListAgents)

## Do this (in order)

1. **[H] Pre-flight** (plan task 1):
   - Compose check at the candidate commit: start the stack **without judge** → identity is Ready and serving while its
     `XLEARN_JUDGE` ack consumer waits; then start judge → the durable binds. If identity blocks before
     `ListenAndServe`, **stop and fix first** (a merge-only X PR starting that consumer in the background).
   - Memory sum: `ssh sujaykumar-vps 'bash -s -- --cluster --nats-stage=n3' < ../infra/hack/host-verify.sh` green (N4 is mi-11's,
     after this sprint); add judge's 256 Mi limit + 256 Mi surge by hand; still ≤ capacity − 0.5 GiB. Record the numbers.
2. **[E] evalpack `v1.0.0`** (plan task 2): with `../xlearn` checked out at the commit you'll tag `v1.13.0`, run
   `make packcheck` in `../xlearn-evalpack` (it builds packlint via `go -C ../xlearn run ./cmd/packlint`) and
   `(cd ../xlearn && go run ./cmd/packlint check --public . --pack ../xlearn-evalpack)`; TLs stay provisional unless the
   runner baselines × prod multipliers exist (then re-run the TL gate). Tag `v1.0.0`; CI must build and the anonymous
   manifest GET must be 401/403. Record the digest; confirm the PAT pull works, anonymous fails, and the ImageRepository
   lists `1.0.0`.
3. **[I] evalpack ImagePolicy PR** (plan task 3): `xlearn-evalpack`, `^\d+\.\d+\.\d+$`, `>=1.0.0 <2.0.0`,
   `digestReflectionPolicy: IfNotPresent` (check the field against the live Flux CRD); header comment on the marker and
   judge restarts. Merge; verify latest = `1.0.0@sha256:…`.
4. **[I] judge nkey + ACL PR** (plan task 4): generate the nkey offline (seed never printed) →
   `apps/secrets/xlearn-nats-judge.enc.yaml` (`sops -e -i`); `make nats-acl-render` on `main`; the diff vs the live
   `messaging` block must be **only** the new judge user and identity's `XLEARN_JUDGE` ack durable — anything else, stop.
   Paste it (public key plaintext), **no** pod-template annotation bump (reload, not restart). Confirm MI-5 already admits
   `xlearn-judge`. Merge; verify the reload (`config_load_time` moved, `nats-0` not restarted), no `legacy`, no
   permission-violation logs, `host-verify --cluster --nats-stage=n3` green.
5. **[X] Tag `v1.13.0`** (plan task 5): run every release-checklist line in the plan first (peers, next free minor,
   major = `.release-line`, ACL merged). Title `v1.13.0 — v2 build · M3-1 judge dark`. After the tag: 7 releases on the
   new version, their ImagePolicies' latest = the tag, healthz reports it, smoke login/dashboard/coach (never enter
   credentials: use an already-signed-in browser session if you have one, else run the credential-free checks and leave
   "owner login smoke pending" as a pending-smoke note in status.md); the `xlearn-judge` image exists; identity's
   `XLEARN_JUDGE` consumer is waiting, not failing.
6. **[X] GHCR check** (plan task 6): `crane ls ghcr.io/sujaykumarsuman/xlearn-judge` anonymously lists `1.13.0` → carry on.
   If it returns 401/403, **don't wait**: making **that package** public and linking it to the repo is an owner-only
   GitHub-settings action (it holds only public code), and the package only exists since your tag push. Still do steps 7
   and 10, record ⛔ "owner: set the `xlearn-judge` package public and link it to the repo" in status.md, set plan task 8 ⛔
   and skip steps 8–9. A re-run of this prompt after the owner has done it skips the ✅ steps (never re-tag) and picks up at
   step 8. Never touch `xlearn-evalpack`'s visibility.
7. **[I] DB 4-step PR** (plan task 7): `pg-xlearn-judge.enc.yaml`, the `xlearn_judge` managed role (`connectionLimit: -1` like its neighbours), schema `judge` in
   `xlearn-database.yaml`, `apps/secrets/xlearn-judge-db.enc.yaml` (same password, generated offline, never echoed).
   Merge; verify role + schema owner + CNPG healthy (read-only psql via exec).
8. **[I] HelmRelease PR** (plan task 8): the runner bearer `apps/secrets/xlearn-judge-runner-auth.enc.yaml` (re-encrypt
   the runner's value if mi-10 ran, else generate and note it for mi-10); ImageRepository + ImagePolicy `xlearn-judge`;
   `apps/xlearn-judge.yaml` exactly per the plan's values table (env names as merged in `internal/judge/config.go`;
   `JUDGE_ADMISSION=open` explicitly, `RUNNER_BASE_URL`/`RUNNER_TOKEN_FILE`, `IDENTITY_BASE_URL`, `JUDGE_SCRATCH_DIR` on an
   `emptyDir`; the evalpack image volume with its `$imagepolicy` marker — or spk-02's fallback);
   the NetworkPolicy (gateway + practice ingress on 8087; egress DNS, PG, NATS, runner :8090, gateway :8080, identity :8081);
   `xlearn	xlearn-judge` in `hack/expected-netpol.tsv` and its embedded copy. Before merge: `helm template`, a
   `--dry-run=server` of the Deployment and NetworkPolicy, and the selector proof. Merge.
9. **[H] Verify** (plan task 9): judge 1/1 Ready; `judge admin status` → pack `1.0.0`, N evaluable; NATS (judge's key, no
   `legacy`, `XLEARN_JUDGE` exists, erase durable and identity's ack durable bound); PG sessions ≤ 8; JWKS OK; memory
   sum and NetworkPolicy checks green in `host-verify --cluster`; the erase durable replayed past requests with no dead
   letters; `JUDGE_BASE_URL` still unset on the gateway; smoke again (step 5's credential rule).
10. **[X] Record** (plan task 10) in an xlearn docs PR, then merge it.

## Constraints

- **Order is the contract** ([ADR-0034 §1.4–§1.5](../../adr/0034-v2-release-labelling-gating-and-rollback.md#15-other-release-streams)):
  image before policy (pack and judge), ACL before the tag, HelmRelease after the tag. Never widen or touch another range.
- **GitOps only:** every cluster change is a `../infra` PR; never `kubectl apply` (a `--dry-run=server` persists nothing
  and is allowed); read-only `ssh sujaykumar-vps` for verification; `judge admin` via `kubectl exec` is a sanctioned path (D33).
  **Infra PRs stand alone** — never folded into the tag.
- **Secrets:** nkey seed, DB password and runner bearer are generated offline and piped straight into `sops -e -i`; never
  echoed, logged, or committed in plaintext. Public nkeys are plaintext in `messaging` (no SOPS decryption there).
- **Service boundaries ([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)):** role `xlearn_judge` owns only schema `judge`.
- **NATS standing rules (ADR-0035 §2):** ACL PR merged before the tag; a NetworkPolicy change for any new caller in its
  own PR before the tag (here: judge's pod carries its own policy; PG/NATS ingress already forward-declared).
- **Memory-sum rule (ADR-0035 §5):** judge carries a memory limit (256 Mi); the sum with it is checked before and after.
- **D34:** no alert, no Flux `Alert`/`Provider`, no opscheck, no healthchecks.io, no push channel.
- **Evalpack privacy:** never make `xlearn-evalpack` public; never copy pack content into xlearn; the probe must stay green.
- **Parallel sessions:** re-check peers' tags and PRs immediately before the tag push; never move or re-push a tag.

## Deliverables

- evalpack `v1.0.0` (digest recorded) and its ImagePolicy.
- infra PR: judge nkey seed + ACL (and identity's ack durable), merged before the tag.
- Tag `v1.13.0`, verified; `xlearn-judge` image pullable anonymously.
- infra PR: judge DB 4-step. infra PR: runner bearer copy, ImageRepository/ImagePolicy `xlearn-judge`, `apps/xlearn-judge.yaml`, `expected-netpol.tsv`.
- judge Ready dark on prod; an xlearn docs PR updating status.

## Update status

- This plan's Status table ([`../sprints/sprint-m3-07.md`](../sprints/sprint-m3-07.md)): each task 🔄 → ✅; _Overall_ ✅ at the end.
- [`../status.md`](../status.md): Sprint board row m3-07 ✅; **MI row MI-13 ✅**; M3 milestone row (M3-1 live dark);
  **milestone → tag → floor → snapshot** `M3-1 → v1.13.0 → floor unchanged → n/a`; **evalpack stream** row (`1.0.0`,
  digest, `validated_against`, items, TL provisional); **content status** (stamped items per tier); NATS rows (judge nkey,
  ACL PR, reload); runner-auth provenance (never the value); **flag inventory** (judge kill switch = permanent kill
  switch; `JUDGE_BASE_URL` unset); memory-sum numbers.
- Decisions log: judge's gateway :8080 (JWKS) and identity :8081 (erase re-verify) egress added to MI-13's list (mi-11 leaves judge's policy as is; mi-12 then adds only TCP 443 and verifies :8081 is present); the DB 4-step as its own PR before the
  HelmRelease; the identity/`XLEARN_JUDGE` startup finding. ADR only for a call beyond ADR-0034/0035 (check peers' ADR numbers).
- If step 6 found the package private: task 8 ⛔ and the ⛔ "owner: set the `xlearn-judge` package public and link it to the
  repo" (Sprint board row and **Blocked / needs input**); MI-13 stays 🔄 until the re-run lands step 8 and completes the record.

## Done when (acceptance)

- [ ] judge Ready on prod with the pack mounted (`judge admin status` N evaluable); 0 learner exposure.
- [ ] Order respected and recorded: pack image → pack policy → ACL → tag → DB 4-step → HelmRelease.
- [ ] judge on NATS with its own nkey; erase consumer and identity's ack durable bound; `XLEARN_JUDGE` exists.
- [ ] Born default-deny egress (DNS, PG, NATS, runner, gateway :8080, identity :8081); expected-netpol updated; `host-verify --cluster` green incl. the memory sum.
- [ ] `v1.13.0` verified per the release checklist; evalpack `v1.0.0` + ImagePolicy live.

## Ship (land-and-sync — owner approval pre-granted)

> Launching this prompt is the owner's approval for every change it makes (D40); don't stop for review.

1. Branch, then conventional commit(s) with the attribution lines, then push, then a PR in every repo touched (`../infra` PRs first where the order requires it; infra PRs are never folded into a tag). Here: the four `../infra` PRs, each its own, in the release action's order below; the xlearn docs/status PR (plus a merge-only fix PR if the pre-flight needed one); `../xlearn-evalpack` gets a tag, not a PR.
2. Once CI is green (fix, then merge, on failure), squash-merge. Never enable auto-merge. (infra has no CI: paste each PR's local checks from the plan, e.g. the ACL render diff, `helm template`, `--dry-run=server` and the selector proof, into its body and merge on them; verify after each merge.)
3. **Release action — evalpack `v1.0.0` + tag `v1.13.0` (ACL PR before, DB 4-step and HelmRelease after):** in this order, never folding an infra PR into the tag:
   - evalpack `v1.0.0` by the stream's own procedure (step 2 above): `make packcheck` and `packlint check` against the commit you'll tag, tag `v1.0.0`, CI builds it and the anonymous manifest GET is 401/403; record it under **release streams** (digest, `validated_against`);
   - the evalpack ImagePolicy PR (step 3 above), then the judge nkey + NATS ACL PR (step 4 above), both merged and verified **before** the tag;
   - walk the release checklist (ADR-0034 §6; the plan's Release checklist), push the tag `v1.13.0` (the next free version), let Flux deploy, then verify live by looking (step 5 above); no snapshot (not a contract, erase or GA tag);
   - the anonymous GHCR check (step 6 above), then the judge DB 4-step PR (step 7 above), then the judge HelmRelease PR (step 8 above). If step 6 found the package private, step 8 is ⛔ in status.md and a re-run lands it; the session never waits;
   - verify (step 9 above) and record (step 10 above).
4. Update status: the sprint file and `docs/v2/status.md`, in the same PR or a follow-up docs PR merged the same way.
5. Run `git checkout main && git pull` in every repo touched (xlearn, `../infra` and `../xlearn-evalpack`). If a clean peer worktree holds `main`, use `git -C <worktree> merge --ff-only origin/main` and then `git switch --detach main`.

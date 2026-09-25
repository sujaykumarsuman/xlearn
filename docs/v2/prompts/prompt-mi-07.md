# Prompt — Sprint mi-07 · Evalpack plumbing (MI-9)

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-mi-07.md`](../sprints/sprint-mi-07.md)   ·   **Milestone:** MI (rollout step MI-9)   ·   **Prereqs:** none (owner event `ev-machine-user`, before launch); must land by Fri 2026-10-09
>
> **Run twice.** The package grant and the helper run are owner steps, and they can only follow this prompt's own `v0.1.0` push. A session never waits on the owner (D40). So the **first launch** does steps 1–3, the helper PR (step 5a) and the record, and sets plan tasks 4–6 ⛔. The **re-run**, launched once the owner has done the re-run items below, does steps 4, 5b, 6 and the record.

## Before you launch (owner)

Launching this prompt attests these are done (D40). If one turns out to be missing, land everything that doesn't depend on it and record the gap as ⛔ in `status.md`; don't wait.

**First launch** (`ev-machine-user`, about 20 minutes):
- [ ] A GitHub **machine user** with 2FA on, and an email address you control.
- [ ] A **classic** PAT on it with **only `read:packages`**, expiring in at most 1 year. At launch, give the session the machine-user **name** and the **expiry date**, never the PAT value.

**Re-run** (after the first session has pushed `v0.1.0` and merged the helper):
- [ ] In the `xlearn-evalpack` package settings, the package shows **Private** and is linked to `xlearn-evalpack`. Under "Manage access", the machine user has **Read**, and `xlearn-evalpack` keeps **Write** under "Manage Actions access" (the fallback is a read collaborator on the repo).
- [ ] Locally: `docker login ghcr.io -u <machine-user> --password-stdin` (typed), then `docker pull ghcr.io/sujaykumarsuman/xlearn-evalpack:0.1.0` succeeds, then `docker logout ghcr.io`. An anonymous pull fails. Tell the session the result.
- [ ] In `../infra` on an up-to-date `main`, run `hack/evalpack-pull-secret.sh`. It writes `apps/secrets/xlearn-evalpack-pull.enc.yaml` and `apps/secrets/xlearn-evalpack-pull-flux-system.enc.yaml`, encrypted; leave them uncommitted.

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md): repo conventions and the land-and-sync rule.
- [`../sprints/sprint-mi-07.md`](../sprints/sprint-mi-07.md): the plan, including the skeleton table, the probe table (with its negative control) and the ImageRepository YAML.
- [ADR-0027](../../adr/0027-content-evalpack-and-user-data-model.md):
  - §1: the public/private rule;
  - §2: the private eval pack, image volume and `contract_hash`;
  - §5: the leak controls.
- [t1 §3.3 and §3.5](../research/t1-content-data-model.md): the pack repo layout, the image, "Private CI … anonymous manifest GET must return 401 or 403", the machine user and classic `read:packages` PAT, and rollback.
- [ADR-0034](../../adr/0034-v2-release-labelling-gating-and-rollback.md):
  - §1.4: the range-change order, where **a new policy goes tag first, then merge**;
  - §1.5: the evalpack stream, `>=1.0.0 <2.0.0`, a major is a contract break.
- [ADR-0035](../../adr/0035-v2-operations-nats-auth-limits-capacity.md) §3: D34, no alerting. The evalpack CI probe and the PAT expiry date in status.md are the only safeguards.
- [`../rollout-plan.md`](../rollout-plan.md): §2 (the MI-9 row, "depends on —") and §2.2 (image before HelmRelease or policy; infra PRs stand alone).
- `../infra`:
  - `README.md` "Secrets (SOPS + age)" and the image-automation notes;
  - `.sops.yaml` (the `apps/secrets/*.enc.yaml` rule encrypts `data`/`stringData`);
  - `apps/image-automation.yaml` (the existing ImageRepositories and the header comment);
  - `hack/host-lint.sh`.
- The consumers: [spk-02](../sprints/sprint-spk-02.md) (mounts `0.1.0` on Thu 2026-10-15), [m3-02](../sprints/sprint-m3-02.md) (the pipeline), [m3-07](../sprints/sprint-m3-07.md) (`v1.0.0`, the ImagePolicy, judge's volume).

## Context

- judge grades against hidden cases and keys that must **never** be public. The xlearn repo and its
  images are public, so the hidden data lives in a **fresh private repo** `xlearn-evalpack`. Its CI
  pushes a private `FROM scratch` data image to GHCR. Flux scans it with a machine-user PAT, and from M3
  judge mounts it as an image volume.
- This sprint builds only the pipe: a `v0.1.0` scaffold image with no content. That's enough for the
  image-volume spike and for authoring to start.
- Two facts measured 2026-09-24 shape the probe:
  - an **unauthenticated** manifest GET on GHCR returns **401 even for a public image**;
  - a **missing** package answers 403 exactly like a private one.

  So the probe must use the anonymous token exchange, include a public negative control (→ 200), and
  pair the privacy check with an authenticated existence check (→ 200).
- The ImagePolicy is **not** in this sprint. It lands in m3-07 once evalpack `v1.0.0` exists (image before policy).

## Entry gates — verify first (stop and report if any is unmet)

- [ ] None from the MI track: MI-9 depends on no other MI step. The owner's machine user + PAT (task 1) is a before-launch item.
- [ ] `gh repo view sujaykumarsuman/xlearn-evalpack` → not found, and no peer is creating it (ListAgents, `gh repo list sujaykumarsuman`). If it exists, stop: verify it's private, not a fork or template, and that its package isn't public, then report. **On the re-run** it exists from the first launch: run the same checks (`PRIVATE/false/false`, the probe green), then carry on.
- [ ] No open peer PR in `../infra` touches `apps/image-automation.yaml`, `apps/secrets/`, `hack/host-lint.sh` or `README.md`. If [mi-01](../sprints/sprint-mi-01.md) or [mi-02](../sprints/sprint-mi-02.md) is open (both edit `host-lint.sh` in the same week), rebase onto it and keep **all** shellcheck additions and README sections.

## Do this (in order)

1. **[O, before launch] Machine user + PAT.** Check that the owner has given you the machine-user **name** and the PAT's **expiry date** (a before-launch item: a GitHub machine user with 2FA on and a **classic** PAT with **only `read:packages`**, expiring in at most 1 year). If they're missing, carry on with steps 2–3 and 5a, and set plan tasks 1 and 4–6 ⛔ in status.md.

   **Never ask for, or accept, the PAT value in chat.** Creating accounts is owner-only.

2. **[E] Create the repo and scaffold.**
   - Run `gh repo create sujaykumarsuman/xlearn-evalpack --private` (fresh, never `--template` or a fork), then verify `gh repo view --json visibility,isFork,isTemplate` gives `PRIVATE/false/false`.
   - Clone it to `../xlearn-evalpack`, as a sibling, never inside xlearn.
   - Add the plan's skeleton: `README.md` (the hard rules), `pack.json` `{"format_major": 0, "version": "0.1.0"}`, `courses/dsa/items/.gitkeep`, `tests.lock` `{}`, `hack/build.sh`, the `Dockerfile` (`FROM scratch`, the `org.opencontainers.image.source` label, `COPY build/ /`), and `.gitignore`.
   - Add `.github/workflows/build.yml`, with third-party actions pinned to SHAs:
     - **`probe`,** on every push, pull request and `workflow_dispatch`, and after `build`. Implement the plan's probe table exactly:
       - **The probed tag.** On a `v*` tag build, it's the pushed tag without the `v`. Otherwise it's the highest semver from an **authenticated** `tags/list` (`GITHUB_TOKEN`, `sort -V`). Before the first tag, the existence and manifest checks are skipped with a notice.
       - **Negative control:** an anonymous exchange on `ghcr.io/sujaykumarsuman/xlearn-gateway`, then `tags/list` → 200.
       - **Existence:** `GITHUB_TOKEN` (`packages: read`) GET of `manifests/<tag>` → 200.
       - **Privacy:** an anonymous token from `https://ghcr.io/token?scope=repository:sujaykumarsuman/xlearn-evalpack:pull&service=ghcr.io`. A **401/403 here ends the check as PASS**. If a token is issued (200), the manifest and `tags/list` with it must both be **401/403**.
       - **Any 200 on a privacy fetch fails the job loudly** (the package is public). **Any other code** (404, 5xx, curl error) at any step is a probe error and also fails the job. Never treat "non-200" as private.
     - **`build`,** on `v*` tags only, with `packages: write`. It validates the tag, runs `hack/build.sh` (the version must equal the tag), and pushes `ghcr.io/sujaykumarsuman/xlearn-evalpack:${TAG#v}` with `provenance: false` and `sbom: false`. It writes the digest to the job summary.
   - Make the initial commit on `main` (conventional, with the attribution lines) and push. The first `probe` run must be green (existence skipped).

3. **[E] Tag `v0.1.0`.** Push the tag and watch the run:
   - build ✅ and the digest;
   - existence 200;
   - the privacy check's 401/403 (at the token step, or on the manifest and `tags/list`);
   - negative control 200.

   The probe's 401/403 is the privacy evidence here. Before the re-run, the owner checks in the package settings that the package shows **Private** and is linked to `xlearn-evalpack` (a re-run before-launch item). Record the digest.

4. **[O, before the re-run] Package access + PAT pull** (plan task 4).
   - **First launch:** skip it. Set plan task 4 ⛔ "owner: package grant + PAT pull, then re-run". The package only exists since step 3, so this can't be a before-launch item of this launch.
   - **Re-run:** the owner has granted the machine user **Read** on the package ("Manage access"), keeping `xlearn-evalpack` at **Write** under "Manage Actions access" (the fallback is a read collaborator on the repo). They've verified locally with `docker login ghcr.io -u <machine-user> --password-stdin` (typed), `docker pull ghcr.io/sujaykumarsuman/xlearn-evalpack:0.1.0` and `docker logout ghcr.io`; an anonymous pull must fail. Record their result, then re-run `probe` with `workflow_dispatch` to show the access change broke nothing.

5. **[I] Pull secrets + helper** (plan task 5):
   - **5a, first launch** (branch `feat/mi-9-evalpack-pull-helper` in `../infra`): the helper, the README rotation section and the host-lint change, as their own PR, merged (it's inert: a script and docs). The owner runs the helper before the re-run.
   - **5b, re-run:** the owner's helper run left the two encrypted files uncommitted in `../infra`. Check them (below) and commit them with step 6 on branch `feat/mi-9-evalpack-pull`.
   - Write `hack/evalpack-pull-secret.sh` (5a). It:
     - reads the machine-user name and PAT with `read -rs`, never from argv, and uses `printf` builtins;
     - builds the `kubernetes.io/dockerconfigjson` Secret `xlearn-evalpack-pull` in memory;
     - encrypts it through `sops encrypt --filename-override <target> --input-type yaml --output-type yaml /dev/stdin` (check `sops encrypt --help`; local sops is 3.13);
     - writes `apps/secrets/xlearn-evalpack-pull.enc.yaml` (ns `xlearn`) and `apps/secrets/xlearn-evalpack-pull-flux-system.enc.yaml` (ns `flux-system`);
     - never leaves plaintext on disk;
     - `unset`s the PAT.
   - Add it to host-lint's shellcheck set.
   - **The owner runs it** in `../infra`, between the two launches.
   - On the re-run, check that `git diff` shows only `ENC[` values under `data`. Don't decrypt to the terminal.
   - Add the rotation runbook to the infra README "Secrets" (5a):
     1. new PAT;
     2. re-run the helper;
     3. PR and merge;
     4. the ImageRepository re-scans Ready;
     5. revoke the old PAT;
     6. update the status.md expiry row.

6. **[I] ImageRepository (re-run), in the same PR as the two Secrets (5b):**
   - Add `xlearn-evalpack` (namespace `flux-system`, `image: ghcr.io/sujaykumarsuman/xlearn-evalpack`, `interval: 5m`, `secretRef: {name: xlearn-evalpack-pull}`) to `apps/image-automation.yaml`, and extend its header comment: the evalpack policy (`>=1.0.0 <2.0.0`) lands after `v1.0.0`, in m3-07. **Add no ImagePolicy.**
   - Open the PR with the checks in the body (infra has no CI), and merge it.
   - Verify read-only over `ssh sujaykumar-vps`:
     - the ImageRepository reports `Ready=True`, `tagCount 1`, `latestTags [0.1.0]`;
     - both Secrets exist with type `kubernetes.io/dockerconfigjson` (`-o jsonpath='{.type}'`; never print `data`);
     - `k3s kubectl get kustomization apps -n flux-system` is Ready;
     - `ssh sujaykumar-vps 'bash -s -- --cluster' < ../infra/hack/host-verify.sh` shows no FAIL.

7. **[X] Record** (branch `docs/mi-07-status` here, after each launch):
   - the Status table in [`../sprints/sprint-mi-07.md`](../sprints/sprint-mi-07.md);
   - in [`../status.md`](../status.md):
     - the **MI track** row MI-9 ✅, with repo date, CI run link, infra PRs, and the note "ImagePolicy → m3-07 (image before policy)". After the first launch it's 🔄 instead, with plan tasks 4–6 ⛔ "owner: package grant + PAT pull + helper run, then re-run";
     - the **evalpack PAT expiry** row: the expiry and a rotate-by date 14 days earlier (manual check, D34, event `ev-pat-expiry`);
     - the **evalpack stream** row: `0.1.0` scaffold, its digest, "below every range; spk-02 input";
     - the **Sprint board** row.

## Constraints

- **Never public.** Never make the repo or the package public, and never create the repo as a fork or template. Public is **irreversible** for a GHCR package. A 200 on the privacy probe means: stop, report to the owner, and don't push anything else.
- **The PAT never touches the transcript, argv, shell history, a plaintext file or git.** Only the owner handles it. The committed files are SOPS-encrypted (`.sops.yaml` rule).
- **Image before policy** ([ADR-0034](../../adr/0034-v2-release-labelling-gating-and-rollback.md) §1.4): no evalpack ImagePolicy in this sprint, and **no `>=1.0.0` tag**. `0.1.0` only.
- **No content.** The scaffold carries no items, cases, keys or anything copied from xlearn or derived from answers.
- **GitOps:** the Secrets and the ImageRepository reach the cluster only through a merged `../infra` PR. `ssh sujaykumar-vps` is read-only. No `kubectl create secret` against the cluster.
- **D34, no alerting:** no scheduled or notification workflow. The probe runs on push, pull request, dispatch and tag only, and the PAT expiry is a status.md date.
- **Not applicable here, since there's no xlearn code:** goose + sqlc (`sqlc diff`), outbox/inbox, service boundaries ([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)), `theme.css` verbatim, consumers-before-producers, the ACL-PR-before-consuming-tag rule, and the memory-sum rule (no new pod).
- **Parallel sessions:** check peers' infra PRs and the `gh repo list` before creating the repo and before merging. Peers' `hack/host-lint.sh` and README edits (mi-01, mi-02) are the likeliest conflicts: rebase and keep every shellcheck addition. Rebase on infra `main`, where the image-automation bot commits often.
- **Rollback is the infra revert only.** Deleting the `0.1.0` package version (permanent) and revoking the PAT are **[O] owner actions**. Never do them yourself.
- Conventional commits with the attribution lines, and squash merges in `../infra`. Don't enable auto-merge.

## Deliverables

- `sujaykumarsuman/xlearn-evalpack` (private): the skeleton, `hack/build.sh`, the `FROM scratch` Dockerfile, and `build.yml` (probe with negative control plus existence and privacy checks; build on `v*`). The `v0.1.0` tag and its private image (digest recorded).
- infra PR 1 (first launch):
  - `hack/evalpack-pull-secret.sh`;
  - the README rotation section;
  - the host-lint shellcheck set.
- infra PR 2 (re-run):
  - `apps/secrets/xlearn-evalpack-pull.enc.yaml` and `apps/secrets/xlearn-evalpack-pull-flux-system.enc.yaml` (from the owner's helper run);
  - the `xlearn-evalpack` ImageRepository.
- xlearn docs PR after each launch: the status rows (MI-9, PAT expiry, evalpack stream) and this sprint's Status table.

## Update status

- Set each task row in [`../sprints/sprint-mi-07.md`](../sprints/sprint-mi-07.md) to 🔄 or ✅ (⛔ while waiting on an owner step, with the reason), and _Overall_ to ✅ when all seven are done.
- Mirror the state in [`../status.md`](../status.md): the **Sprint board** row, the **MI track** row MI-9, the **evalpack PAT expiry** row, and the **evalpack stream** row.
- Add a **Decisions log** line for:
  - the probe design (token exchange, negative control, existence check);
  - the package access model you ended up with;
  - `interval: 5m`;
  - `format_major: 0` for the scaffold.
- No ADR is expected. If the delivery mechanism changes from ADR-0027 §2 (e.g. the spike forces ORAS), that's a later sprint's ADR. Check peers' ADR numbers first.

## Done when (acceptance)

- [ ] `xlearn-evalpack` exists: PRIVATE, not a fork or template, and cloned at `../xlearn-evalpack`.
- [ ] CI's privacy check PASSes on an exact **401/403** (at the anonymous token step, or on both the manifest and `tags/list` with an anonymous token). The authenticated existence check returns 200, and the public negative control returns 200. Any other code fails the job. The probe runs on every push.
- [ ] `ghcr.io/sujaykumarsuman/xlearn-evalpack:0.1.0` is pullable with the machine-user PAT and not anonymously, and its digest is recorded (spk-02 input).
- [ ] The ImageRepository `xlearn-evalpack` is Ready and scans the private package with `secretRef` (`0.1.0` listed). There's no ImagePolicy, and `apps` is Ready.
- [ ] Both pull secrets are committed SOPS-encrypted only and exist in `xlearn` and `flux-system`.
- [ ] `docs/v2/status.md` shows MI-9 ✅, the PAT expiry and rotate-by dates, and the evalpack stream row.
- The first launch is done when steps 1–3, 5a and 7 have landed and plan tasks 4–6 are ⛔ for the owner. The re-run ticks the rest.

## Ship (land-and-sync — owner approval pre-granted)

> Launching this prompt is the owner's approval for every change it makes (D40); don't stop for review.

1. Branch, then conventional commit(s) with the attribution lines, then push, then a PR in every repo touched (`../infra` PRs first where the order requires it; infra PRs are never folded into a tag). (`xlearn-evalpack` starts empty: its scaffold is the initial commit on `main`, so there's no PR.)
2. Once CI is green (fix, then merge, on failure), squash-merge. Never enable auto-merge. (infra has no CI: paste the checks into each PR body and merge on them.)
3. **Release action — infra PR(s) only, plus the evalpack scaffold:**
   - In `xlearn-evalpack`, push only `main` and the **`v0.1.0` tag**. No `>=1.0.0` tag.
   - **First launch:** merge infra PR 1 (the helper, the README rotation section, the host-lint change).
   - **Re-run:** merge infra PR 2 (the two SOPS pull secrets with the ImageRepository, no ImagePolicy), then verify read-only (step 6).
   - No xlearn tag.
4. Update status: the sprint file and `docs/v2/status.md`, in the same PR or a follow-up docs PR merged the same way (after each launch).
5. Run `git checkout main && git pull` in every repo touched (xlearn, `../infra`, `../xlearn-evalpack`). If a clean peer worktree holds `main`, use `git -C <worktree> merge --ff-only origin/main` and then `git switch --detach main`.

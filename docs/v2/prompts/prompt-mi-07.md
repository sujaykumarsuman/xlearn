# Prompt — Sprint mi-07 · Evalpack plumbing (MI-9)

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-mi-07.md`](../sprints/sprint-mi-07.md)   ·   **Milestone:** MI (rollout step MI-9)   ·   **Prereqs:** none (owner event `ev-machine-user`); must land by Fri 2026-10-09

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

- [ ] None from the MI track: MI-9 depends on no other MI step. Ask the owner to do task 1 (machine user + PAT) now. It can run in parallel with task 2.
- [ ] `gh repo view sujaykumarsuman/xlearn-evalpack` → not found, and no peer is creating it (ListAgents, `gh repo list sujaykumarsuman`). If it exists, stop: verify it's private, not a fork or template, and that its package isn't public, then report.
- [ ] No open peer PR in `../infra` touches `apps/image-automation.yaml`, `apps/secrets/`, `hack/host-lint.sh` or `README.md`. If [mi-01](../sprints/sprint-mi-01.md) or [mi-02](../sprints/sprint-mi-02.md) is open (both edit `host-lint.sh` in the same week), rebase onto it and keep **all** shellcheck additions and README sections.

## Do this (in order)

1. **[O] Machine user + PAT.** Ask the owner to:
   - create a GitHub machine user (2FA on);
   - create a **classic** PAT with **only `read:packages`** and an expiry of at most 1 year;
   - tell you the machine-user **name** and the **expiry date**.

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

   Confirm with the owner that the package page shows **Private**, linked to `xlearn-evalpack`. Record the digest.

4. **[O] Package access + PAT pull.**
   - Ask the owner to grant the machine user **Read** on the package ("Manage access"), keeping `xlearn-evalpack` at **Write** under "Manage Actions access". The fallback is a read collaborator on the repo.
   - The owner verifies locally: `docker login ghcr.io -u <machine-user> --password-stdin` (typed), `docker pull ghcr.io/sujaykumarsuman/xlearn-evalpack:0.1.0`, `docker logout ghcr.io`. An anonymous pull must fail.
   - Re-run `probe` with `workflow_dispatch` so the access change broke nothing.

5. **[I] Pull secrets + helper** (branch `feat/mi-9-evalpack-pull` in `../infra`):
   - Write `hack/evalpack-pull-secret.sh`. It:
     - reads the machine-user name and PAT with `read -rs`, never from argv, and uses `printf` builtins;
     - builds the `kubernetes.io/dockerconfigjson` Secret `xlearn-evalpack-pull` in memory;
     - encrypts it through `sops encrypt --filename-override <target> --input-type yaml --output-type yaml /dev/stdin` (check `sops encrypt --help`; local sops is 3.13);
     - writes `apps/secrets/xlearn-evalpack-pull.enc.yaml` (ns `xlearn`) and `apps/secrets/xlearn-evalpack-pull-flux-system.enc.yaml` (ns `flux-system`);
     - never leaves plaintext on disk;
     - `unset`s the PAT.
   - Add it to host-lint's shellcheck set.
   - **The owner runs it** in `../infra`.
   - Check that `git diff` shows only `ENC[` values under `data`. Don't decrypt to the terminal.
   - Add the rotation runbook to the infra README "Secrets":
     1. new PAT;
     2. re-run the helper;
     3. PR and merge;
     4. the ImageRepository re-scans Ready;
     5. revoke the old PAT;
     6. update the status.md expiry row.

6. **[I] ImageRepository, in the same PR:**
   - Add `xlearn-evalpack` (namespace `flux-system`, `image: ghcr.io/sujaykumarsuman/xlearn-evalpack`, `interval: 5m`, `secretRef: {name: xlearn-evalpack-pull}`) to `apps/image-automation.yaml`, and extend its header comment: the evalpack policy (`>=1.0.0 <2.0.0`) lands after `v1.0.0`, in m3-07. **Add no ImagePolicy.**
   - Open the PR with the checks in the body (infra has no CI), and merge it.
   - Verify read-only over `ssh vps`:
     - the ImageRepository reports `Ready=True`, `tagCount 1`, `latestTags [0.1.0]`;
     - both Secrets exist with type `kubernetes.io/dockerconfigjson` (`-o jsonpath='{.type}'`; never print `data`);
     - `k3s kubectl get kustomization apps -n flux-system` is Ready;
     - `ssh vps 'bash -s -- --cluster' < ../infra/hack/host-verify.sh` shows no FAIL.

7. **[X] Record** (branch `docs/mi-07-status` here):
   - the Status table in [`../sprints/sprint-mi-07.md`](../sprints/sprint-mi-07.md);
   - in [`../status.md`](../status.md):
     - the **MI track** row MI-9 ✅, with repo date, CI run link, infra PR, and the note "ImagePolicy → m3-07 (image before policy)";
     - the **evalpack PAT expiry** row: the expiry and a rotate-by date 14 days earlier (manual check, D34, event `ev-pat-expiry`);
     - the **evalpack stream** row: `0.1.0` scaffold, its digest, "below every range; spk-02 input";
     - the **Sprint board** row.

## Constraints

- **Never public.** Never make the repo or the package public, and never create the repo as a fork or template. Public is **irreversible** for a GHCR package. A 200 on the privacy probe means: stop, report to the owner, and don't push anything else.
- **The PAT never touches the transcript, argv, shell history, a plaintext file or git.** Only the owner handles it. The committed files are SOPS-encrypted (`.sops.yaml` rule).
- **Image before policy** ([ADR-0034](../../adr/0034-v2-release-labelling-gating-and-rollback.md) §1.4): no evalpack ImagePolicy in this sprint, and **no `>=1.0.0` tag**. `0.1.0` only.
- **No content.** The scaffold carries no items, cases, keys or anything copied from xlearn or derived from answers.
- **GitOps:** the Secrets and the ImageRepository reach the cluster only through a merged `../infra` PR. `ssh vps` is read-only. No `kubectl create secret` against the cluster.
- **D34, no alerting:** no scheduled or notification workflow. The probe runs on push, pull request, dispatch and tag only, and the PAT expiry is a status.md date.
- **Not applicable here, since there's no xlearn code:** goose + sqlc (`sqlc diff`), outbox/inbox, service boundaries ([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)), `theme.css` verbatim, consumers-before-producers, the ACL-PR-before-consuming-tag rule, and the memory-sum rule (no new pod).
- **Parallel sessions:** check peers' infra PRs and the `gh repo list` before creating the repo and before merging. Peers' `hack/host-lint.sh` and README edits (mi-01, mi-02) are the likeliest conflicts: rebase and keep every shellcheck addition. Rebase on infra `main`, where the image-automation bot commits often.
- **Rollback is the infra revert only.** Deleting the `0.1.0` package version (permanent) and revoking the PAT are **[O] owner actions**. Never do them yourself.
- Conventional commits with the attribution lines, and squash merges in `../infra`. Don't enable auto-merge.

## Deliverables

- `sujaykumarsuman/xlearn-evalpack` (private): the skeleton, `hack/build.sh`, the `FROM scratch` Dockerfile, and `build.yml` (probe with negative control plus existence and privacy checks; build on `v*`). The `v0.1.0` tag and its private image (digest recorded).
- infra PR:
  - `apps/secrets/xlearn-evalpack-pull.enc.yaml` and `apps/secrets/xlearn-evalpack-pull-flux-system.enc.yaml`;
  - `hack/evalpack-pull-secret.sh`;
  - the `xlearn-evalpack` ImageRepository;
  - the README rotation section;
  - the host-lint shellcheck set.
- xlearn docs PR: the status rows (MI-9, PAT expiry, evalpack stream) and this sprint's Status table.

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
- Ship at session end per AGENT.md land-and-sync, with this sprint's release action: **infra PR(s) only**, plus the evalpack scaffold.
  - Merge the `../infra` PR yourself, with the checks in the body (infra has no CI).
  - In `xlearn-evalpack`, push only `main` and the **`v0.1.0` tag**. No `>=1.0.0` tag.
  - Merge the xlearn status docs PR.
  - No xlearn tag.
  - Sync local `main` in xlearn, `../infra` and `../xlearn-evalpack`.

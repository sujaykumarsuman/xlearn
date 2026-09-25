# Sprint mi-07 — Evalpack plumbing (MI-9)

> **Milestone:** MI — infra-first track (rollout step **MI-9**; sprint ids `mi-NN` ≠ rollout steps `MI-N`) · **Track:** infra + evalpack · **Order:** 5
> **Prereqs:** none. MI-9 depends on no other MI step ([rollout §2](../rollout-plan.md)); only the owner's machine-user action (calendar event `ev-machine-user`), done before launch.
> **Unblocks:** [m3-02](sprint-m3-02.md) (the evalpack pipeline) · [spk-02](sprint-spk-02.md) (the image-volume spike needs `0.1.0` + the PAT) · [m3-07](sprint-m3-07.md) (evalpack `v1.0.0`, the ImagePolicy, judge's image volume)
> **Release action:** **infra PR(s) only**, plus the new evalpack repo scaffold. **Only a `v0.1.0` image; no `>=1.0.0` tag.** Two `../infra` PRs: the helper PR (first launch), then the pull secrets + ImageRepository PR (the re-run, after the owner's package-grant and helper steps). Plus an xlearn docs PR for status after each launch.
> **Calendar:** weeks 1–2. **Must land by Fri 2026-10-09**, re-run included: spk-02 runs the image-volume spike on Thu 2026-10-15, and pack authoring starts at the ≈ 2026-10-05 schema freeze.
> **Execute with:** [`../prompts/prompt-mi-07.md`](../prompts/prompt-mi-07.md) — one prompt, run **twice**. The owner's package grant (task 4) and helper run (task 5) can only follow the first session's own `v0.1.0` push, and a session never waits on the owner (D40). So the first launch lands tasks 1–3, the helper and the record, then ⛔s tasks 4–6. The re-run, launched once the owner has done those steps, lands tasks 4–6.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | Machine user + classic `read:packages` PAT — before launch | O | ⬜ |
| 2 | Create `xlearn-evalpack` (fresh, private) + scaffold + CI with the anonymous-GET probe | E | ⬜ |
| 3 | Tag `v0.1.0` → private scaffold image, probe green | E | ⬜ |
| 4 | Grant the machine user read on the package; verify a PAT pull — before the re-run's launch | O | ⬜ |
| 5 | SOPS pull secrets (`xlearn`, `flux-system`) + rotation helper (the helper merges on the first launch; the owner runs it before the re-run) | I | ⬜ |
| 6 | `xlearn-evalpack` ImageRepository with `secretRef` (no ImagePolicy), in the re-run's PR with the Secrets | I | ⬜ |
| 7 | Record (after each launch) | X | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row + MI track row MI-9 + the evalpack stream and PAT-expiry rows).
> Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] None from the MI track: MI-9 depends on no other MI step ([rollout §2](../rollout-plan.md)). The only prerequisite is the owner's machine-user action (task 1), done before launch.
- [ ] `sujaykumarsuman/xlearn-evalpack` doesn't exist yet (`gh repo view` → not found; true on 2026-09-24), and no peer session is creating it (ListAgents, `gh repo list`). If it exists, stop and verify it is private, not a fork or template of xlearn, and that its package isn't public, before reusing it. **On the re-run** it exists from the first launch: run the same privacy checks, then carry on.
- [ ] No open peer PR in `../infra` touches `apps/image-automation.yaml`, `apps/secrets/`, `hack/host-lint.sh` or `README.md` (parallel sessions). If [mi-01](sprint-mi-01.md) or [mi-02](sprint-mi-02.md) is open (both edit `host-lint.sh` in the same week), rebase onto it and keep **all** shellcheck additions and README sections.

## Goal

Create the private eval-pack delivery path **early and on no other MI step**, so the image-volume spike
(Thu 2026-10-15) and pack authoring (from the ≈ 2026-10-05 schema freeze) start on time. That path is:
- a machine user;
- a **fresh private** `xlearn-evalpack` repo whose CI builds a `FROM scratch` data image and then proves, on every push, that the package **isn't public**;
- a private GHCR package;
- a PAT held only as SOPS pull secrets in `xlearn` and `flux-system`;
- a Flux `ImageRepository` that can scan it.

Hidden test data never enters the public repo or the public images ([ADR-0027](../../adr/0027-content-evalpack-and-user-data-model.md) §1–2). This sprint builds the pipe only: a `v0.1.0` scaffold with no content.

## Scope

**In**
- MI-9:
  - the machine user and a classic `read:packages` PAT (owner, before launch);
  - `xlearn-evalpack`, created fresh and private: skeleton, `FROM scratch` Dockerfile, CI build on `v*` tags, and the **anonymous-GET probe on every push**;
  - a `0.x` scaffold image `ghcr.io/sujaykumarsuman/xlearn-evalpack:0.1.0`;
  - the PAT as SOPS dockerconfigjson pull secrets in `xlearn` and `flux-system`, plus a rotation helper;
  - an `ImageRepository` with `secretRef`.
- The PAT expiry date in `docs/v2/status.md` as a **manual check** (D34, [ADR-0035](../../adr/0035-v2-operations-nats-auth-limits-capacity.md) §3). No alert.

**Out**
- **The evalpack `ImagePolicy`** → [m3-07](sprint-m3-07.md). Rollout §2 lists the policy under MI-9, but **image before policy** ([ADR-0034](../../adr/0034-v2-release-labelling-gating-and-rollback.md) §1.4) puts it after evalpack `v1.0.0` exists. A `>=1.0.0 <2.0.0` policy today would match nothing and sit not-Ready.
- Pack CI gates beyond build + probe, the real pack format, one layer per course, `cases.jsonl.zst`, and the compose fixture pack → [m3-02](sprint-m3-02.md).
- Any pack content, and the 14 pilot packs (owner event `ev-packs-14`).
- The public-repo guards: the pre-push fingerprint hook, `.gitignore`/`.dockerignore`, and the AGENT.md rule *"never copy content from `../xlearn-evalpack`"* → [m3-01](sprint-m3-01.md).
- judge's image volume, `imagePullSecrets` and `EVALPACK_DIR` → [m3-07](sprint-m3-07.md). The mount and credential-isolation spike → [spk-02](sprint-spk-02.md).
- PSA labels and SA tokens off → [mi-08](sprint-mi-08.md). The N3 ≥ 24 h re-check → [l-01](sprint-l-01.md).

## Tasks

### 1 · Machine user + classic `read:packages` PAT [O, before launch]

The owner does this as calendar event `ev-machine-user` (about 20 minutes), before launching the prompt:
- Creates a GitHub **machine user** with 2FA on, and an email address the owner controls.
- Creates a **classic** PAT on that account with **only `read:packages`**. Fine-grained PATs don't cover the GHCR registry.
- Sets an expiry of at most 1 year. At launch, gives the agent the machine-user **name** and the **expiry date**. The PAT value itself never goes in chat, a terminal argument or git.

Creating accounts is owner-only. If the name and date are missing at launch, tasks 2–3 and the helper still
land (they don't need them); tasks 1 and 4–6 go ⛔ in status.md.

### 2 · Create `xlearn-evalpack` (fresh, private) + scaffold + CI [E]

- **Create it:** `gh repo create sujaykumarsuman/xlearn-evalpack --private`.
  - **Never** a fork or template of xlearn ([t1 §3.3](../research/t1-content-data-model.md)).
  - Verify `gh repo view --json visibility,isFork,isTemplate` gives `PRIVATE/false/false`.
  - Clone it as the **sibling** `../xlearn-evalpack`, never nested inside xlearn.
- **Skeleton** (initial commit on `main`; the repo is empty, so there's no PR):

| Path | Content |
|---|---|
| `README.md` | Purpose and hard rules: never public (package or repo); never a fork or template; never copy content into xlearn; no `>=1.0.0` tag before [m3-07](sprint-m3-07.md); the probe must stay green |
| `pack.json` | `{"format_major": 0, "version": "0.1.0"}`. `0` marks a scaffold judge must never load; [m3-02](sprint-m3-02.md) sets the real format |
| `courses/dsa/items/.gitkeep`, `tests.lock` (`{}`) | the t1 §3.3 layout, empty |
| `hack/build.sh` | assembles `build/`: `/manifest.json` = `{format_major, version, validated_against: null, items: {}}`, from `pack.json`. Fails if `version` ≠ the tag without `v`. No generators, oracles or answers |
| `Dockerfile` | `FROM scratch` · `LABEL org.opencontainers.image.source=https://github.com/sujaykumarsuman/xlearn-evalpack` (links the package to the private repo) · `COPY build/ /` |
| `.github/workflows/build.yml` | the jobs below; third-party actions pinned to commit SHAs |
| `.gitignore` | `build/` |

- **`build.yml`, the `probe` job.** Runs on every `push` (all branches), `pull_request` and `workflow_dispatch`, and after `build`.

  It uses the **anonymous token exchange**. A bare unauthenticated GET returns 401 even for a **public**
  image (verified 2026-09-24 on `xlearn-gateway`), so a naive probe would pass while the package was public.

  **The probed tag `<tag>`.**
  - On a `v*` tag build, it's the pushed tag without the `v`.
  - On any other push, pull request or dispatch, it's the highest semver tag from an **authenticated** `GET /v2/sujaykumarsuman/xlearn-evalpack/tags/list` (`GITHUB_TOKEN`, then `sort -V`).
  - Before the first `v*` tag exists, the existence and manifest checks are skipped with a notice. The token step still runs.

  | Step | Request | Pass | Fail |
  |---|---|---|---|
  | **Negative control** | the same anonymous exchange against a **public** image: `GET /v2/sujaykumarsuman/xlearn-gateway/tags/list` | 200 | anything else: the probe mechanism is broken |
  | **Existence** | `GITHUB_TOKEN` (`packages: read`) GET of `manifests/<tag>` | 200 | anything else |
  | **Privacy, token step** | `GET https://ghcr.io/token?scope=repository:sujaykumarsuman/xlearn-evalpack:pull&service=ghcr.io` without credentials | **401 or 403 ends the privacy check as PASS.** A missing package also answers 403 here, which is why the existence step is needed | a 200 (a token was issued) goes on to the next row. Any other code, or a curl error, fails the job |
  | **Privacy, fetch step** (only after a 200 token) | `GET /v2/…/manifests/<tag>` and `/tags/list` with that anonymous token | **401 or 403** on both | **200** on either: the package is public. **Stop everything; report to the owner.** Any other code (404, 5xx, curl error) is a **probe error**: fail the job, never PASS |

  Every step checks the exact code: nothing is read as "non-200 so private". Measured on 2026-09-25: the anonymous
  token for `xlearn-gateway` is 200 and its `tags/list` 200 (the control works); the token for the not-yet-created
  `xlearn-evalpack` is 403, and a bare `tags/list` without a token is 401 or 404.
- **`build.yml`, the `build` job.** Runs on `v*` tags only, with `contents: read` and `packages: write`.
  - Validate the tag `^v\d+\.\d+\.\d+$` and run `hack/build.sh`.
  - Build and push `ghcr.io/sujaykumarsuman/xlearn-evalpack:${TAG#v}`, stripping the `v` as the fleet does, with `provenance: false` and `sbom: false`.
  - Print the digest to the job summary.
  - Then `needs: build` → run `probe` against the pushed tag.

### 3 · Tag `v0.1.0` → private scaffold image, probe green [E]

- Tag and push `v0.1.0`.
- The run must show:
  - build ✅ and the digest;
  - existence 200;
  - the privacy check's 401/403 (at the token step, or on the manifest and `tags/list`);
  - the negative control 200.
- The package must be **Private** and linked to `xlearn-evalpack`. In this session the probe's 401/403 is the
  privacy evidence. The owner checks both in the package settings as part of task 4, before the re-run.
- This is the image [spk-02](sprint-spk-02.md) mounts. `0.1.0` sits **below** every future `>=1.0.0 <2.0.0` range, so it can never be selected.

### 4 · Grant the machine user read on the package; verify a PAT pull [O, before the re-run's launch]

The package only exists once task 3 has pushed `v0.1.0`, so this can't come before the first launch. The owner
does it after the first session and before re-launching the prompt. The first session sets this task ⛔
"owner: package grant + PAT pull, then re-run" and doesn't wait (D40).
- **Grant access.** In the package settings, under "Manage access", add the machine user with **Read** on the package.
  - Keep `xlearn-evalpack`'s Actions access at **Write** under "Manage Actions access", so tag builds keep pushing.
  - If a package-level grant isn't possible, fall back to a read collaborator on the repo. The PAT's `read:packages` scope still can't read the code.
- **Verify.** The owner, locally (the PAT is typed, never pasted into the transcript), runs `docker login ghcr.io -u <machine-user> --password-stdin`, then `docker pull ghcr.io/sujaykumarsuman/xlearn-evalpack:0.1.0`, then `docker logout ghcr.io`. The pull must succeed. A pull without credentials must fail.
- **On the re-run,** the session re-runs `probe` with `workflow_dispatch` to show the access change broke nothing, and records the owner's pull result.

### 5 · SOPS pull secrets (`xlearn`, `flux-system`) + rotation helper [I]

- **Two launches.**
  - **First launch:** the helper, the README rotation section and the host-lint change go in their own infra
    PR and merge. They're inert: a script and docs.
  - **Before the re-run:** the owner runs the helper in `../infra` on an up-to-date `main`. It writes the two
    encrypted files, which stay uncommitted.
  - **Re-run:** check the files (below), then commit them with task 6 in one PR.
- **The helper:** a new `../infra/hack/evalpack-pull-secret.sh`, which the **owner runs** at creation (before the re-run) and at every rotation. It:
  - reads the machine-user name and the PAT from the tty with `read -rs`, so neither enters argv or history;
  - builds the `kubernetes.io/dockerconfigjson` Secret `xlearn-evalpack-pull` in memory, for `ghcr.io`;
  - pipes it through `sops encrypt --filename-override <target> --input-type yaml --output-type yaml /dev/stdin` (sops ≥ 3.9; confirm the flags with `sops encrypt --help`);
  - writes two files, never plaintext:
    - `apps/secrets/xlearn-evalpack-pull.enc.yaml` (namespace `xlearn`, for judge's `imagePullSecrets` at m3-07);
    - `apps/secrets/xlearn-evalpack-pull-flux-system.enc.yaml` (namespace `flux-system`, for the ImageRepository).

  The existing `.sops.yaml` rule `apps/secrets/*.enc.yaml` encrypts `data`. Add the helper to host-lint's shellcheck set.
- **Before committing,** check that `git diff` shows only `ENC[` values under `data`, and that no plaintext file exists anywhere in the tree. Don't decrypt to the terminal.
- **Rotation runbook.** Add a short section to the infra README "Secrets":
  1. new PAT;
  2. re-run the helper;
  3. PR and merge;
  4. the ImageRepository re-scans Ready;
  5. revoke the old PAT;
  6. update the expiry row in status.md.

### 6 · `xlearn-evalpack` ImageRepository with `secretRef` (no ImagePolicy) [I]

- In `apps/image-automation.yaml`, add the object below, and extend the file's header comment to note the evalpack stream and that its `>=1.0.0 <2.0.0` policy lands after `v1.0.0` (m3-07).

  ```yaml
  apiVersion: image.toolkit.fluxcd.io/v1
  kind: ImageRepository
  metadata:
    name: xlearn-evalpack
    namespace: flux-system
  spec:
    image: ghcr.io/sujaykumarsuman/xlearn-evalpack
    interval: 5m            # packs change rarely; keeps PAT-authenticated scans low
    secretRef:
      name: xlearn-evalpack-pull
  ```

- **Ship it in the re-run's infra PR, together with task 5's two Secrets,** so the Secret exists when the ImageRepository first reconciles. After the merge (read-only, `ssh sujaykumar-vps`):
  - `k3s kubectl get imagerepository xlearn-evalpack -n flux-system -o jsonpath='{.status.conditions[?(@.type=="Ready")].status} {.status.lastScanResult.tagCount} {.status.lastScanResult.latestTags}'` gives `True 1 [0.1.0]`;
  - both Secrets exist, with type `kubernetes.io/dockerconfigjson` (`-o jsonpath='{.type}'` only; never print `data`);
  - the `apps` Kustomization is Ready;
  - `host-verify --cluster` has no FAIL. If [mi-02](sprint-mi-02.md) has merged, its Flux check covers ImageRepositories too.
- **No ImagePolicy yet.**

### 7 · Record [X]

In [`../status.md`](../status.md):
- **MI track:** MI-9 ✅ (evalpack repo created date, CI run link, infra PRs). Note: *ImagePolicy deferred to m3-07 (image before policy)*.
  After the first launch it's 🔄 instead, with tasks 4–6 ⛔ "owner: package grant + PAT pull + helper run, then re-run".
- **PAT expiry row:** the expiry date and a **rotate-by** date 14 days earlier (event `ev-pat-expiry`, manual check, D34).
- **Evalpack stream row:** `0.1.0` scaffold, its digest, "below every policy range; spk-02 input".
- The **Sprint board** row.

Also update this file's Status table. Ship it as an xlearn docs PR after each launch.

## Acceptance criteria

- [ ] `sujaykumarsuman/xlearn-evalpack` exists: **PRIVATE**, not a fork or template, and checked out at `../xlearn-evalpack`.
- [ ] In CI, the privacy check PASSes on an exact **401/403**: at the anonymous token step, or, if a token is issued, on both the manifest and `tags/list`. The authenticated existence check returns 200, and the negative control on a public xlearn image returns 200. Any other code fails the job. The probe runs on every push, against the tag rule above.
- [ ] `ghcr.io/sujaykumarsuman/xlearn-evalpack:0.1.0` is **pullable with the machine-user PAT** (spk-02 input) and not anonymously, and its digest is recorded.
- [ ] The `xlearn-evalpack` ImageRepository is **Ready and scanning** the private package through `secretRef` (`lastScanResult` lists `0.1.0`). No ImagePolicy exists. `apps` is Ready.
- [ ] Both pull secrets are committed SOPS-encrypted only (no plaintext in git history) and exist in `xlearn` and `flux-system`.
- [ ] `docs/v2/status.md` shows MI-9 ✅, the PAT expiry and rotate-by dates, and the evalpack stream row.

## Release

**infra PR(s) only**, plus the evalpack repo scaffold. Launching the prompt, and re-launching it, is the
owner's approval for every change (D40); nothing waits on the owner mid-run.
- Two `../infra` PRs, each merged on its own:
  - **first launch:** `hack/evalpack-pull-secret.sh`, the README rotation section and the host-lint change;
  - **re-run:** the two SOPS pull secrets (from the owner's helper run) and the ImageRepository, together.
- In `xlearn-evalpack`: the initial scaffold commit and the **`v0.1.0` tag only**. **No `>=1.0.0` tag:** the first real pack, `v1.0.0`, is cut in [m3-07](sprint-m3-07.md), after which its ImagePolicy merges (image before policy).
- The status rows ship as an xlearn docs PR after each launch.
- No xlearn tag.
- **Rollback:**
  - the agent reverts the infra PR (removes the ImageRepository and the Secrets). That's the agent's whole rollback;
  - **[O]** deleting the `0.1.0` package version, if ever needed. It's permanent and owner-only;
  - **[O]** revoking the PAT. It's owner-held.

## Definition of Done

- The repo is private with a green probe.
- `0.1.0` is pushed, private, and pullable with the PAT (the owner's local check before the re-run, recorded).
- Both infra PRs are merged (the helper first, then the Secrets with the ImageRepository), with the ImageRepository Ready and no ImagePolicy.
- Statuses and the expiry row are updated (this file + [`../status.md`](../status.md)), and the xlearn docs PRs are merged.
- Local `main` is synced in xlearn, `../infra` and `../xlearn-evalpack`.

## Risks / watch-outs

- **Making the package or repo public is irreversible.** A public GHCR package can't be made private again, and the hidden cases would leak for good. The probe runs on **every push**, and a 200 stops everything. Never change visibility settings.
- **A vacuous probe is the realistic failure.** An unauthenticated GET without the token exchange returns 401 for public images too, and a missing package answers 403 like a private one. The negative control (public → 200) and the authenticated existence check (→ 200) are what make the probe mean something. So does the exact-code rule: a 404, 5xx or network error is a probe error, never "private".
- **PAT hygiene.** It's typed into the helper's `read -rs` and exists only SOPS-encrypted. Never in chat, argv, shell history, a plaintext file or git history. If it leaks, revoke it and rotate.
- **A bad or expired PAT** makes the ImageRepository not-Ready. The `apps` Kustomization (`wait: true`) then goes not-Ready too, and from M3 judge pulls fail at the next restart (its `evaluable = 0` badge). Verify the pull (task 4) before merging, and rotate before the rotate-by date. There is no alert (D34), only the status.md row.
- **Don't tag `>=1.0.0` here.** m3-07's `>=1.0.0 <2.0.0` policy would select a scaffold as "the first pack".
- **The scaffold holds no content**: `format_major: 0`, no items, nothing copied from xlearn, nothing derived from answers.
- **Package access model.** Removing the repo's Actions write access while tightening package access would break tag builds. Re-run a build (a `workflow_dispatch` probe is enough) after changing access.

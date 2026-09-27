# ADR-0036 — AI Builder: execution-manager dashboard, agent API and the manager's D40 delegation (build tooling)

- **Status:** Accepted (2026-09-26). The owner's launch of the AI Builder build session is the approval (D40): the build
  brief (`sujaykumarsuman/ai-builder` `seed/brief-2026-09-26.md`) specified this ADR and D44, and the owner settled the
  credential, second-factor, backup and edge questions in that session. Recorded with **D44**.
- **Date:** 2026-09-26
- **Deciders:** @sujaykumarsuman
- **Related:** [0033](0033-invite-only-admission-and-owner-admin.md) (§10 "no web admin on the shared origin" and §11
  console isolation: this ADR hosts off that origin, so neither is amended), [0034](0034-v2-release-labelling-gating-and-rollback.md)
  (tag train, snapshot rule and rollback: AI Builder is never on the `v*` train and gates tag sprints through
  attestations and holds), [0035](0035-v2-operations-nats-auth-limits-capacity.md) (§3 D34 "no alerting": scoped here to
  exclude this build-tooling channel; §5 memory-sum rule: not engaged, zero footprint on the node),
  [0027](0027-content-evalpack-and-user-data-model.md) (the evalpack never-copy rule, repeated in the hygiene rules),
  [0028](0028-object-storage-and-backups.md) (production backups: unrelated; AI Builder has its own).
- **Links:** [feasibility D44](../v2/feasibility.md#decisions-log-newest-first) · [status.md](../v2/status.md) ·
  [AGENT.md "Manager delegation (D44)"](../../AGENT.md) · [manager charter](../v2/manager/MANAGER.md) ·
  [launch prompt](../v2/prompts/prompt-manager.md) · [owner guide](../v2/manager/OWNER.md) · design and API:
  `sujaykumarsuman/ai-builder` `docs/DESIGN.md`, `docs/api.md` (private repo) · live: https://ai-builder.skriptvalley.com

## Context

The owner asked for the static "xLearn v2 Run Order" page to evolve into a **live execution-manager dashboard** for the
whole v2 plan (89 sprints, 689 tasks), for a **manager Claude session** that runs the v2 build end to end by
dispatching subagents on the existing sprint prompts, and for **steering from any device**: status, remarks, typed
questions ("asks") the manager posts and polls, and a brake. An adversarially verified audit of the v2 plan against the
request (`seed/audit/v2-audit-2026-09-26.{md,json}`: 179 rows) found:

- **67 new-decision rows** that deduplicate to 24 decisions, plus 5 forced by the brief and the verifier notes
  (AIB-01…AIB-29, `docs/research/decisions-dedup.md`). All `blocks_build_start` rows are settled by the brief and the
  owner's decisions below.
- **63 doc-fold rows** (27 groups): decisions already made that xlearn docs don't reflect yet (pre-spike, pre-D41,
  pre-D42 text). They are **not** folded here; they are seeded into AI Builder as doc debt, linked to the sprints they
  block, and the manager's first job is the fold pass.
- **7 deferred / in-sprint rows** seeded as owner events and gates (`ev-q5`, `ev-s6-recheck` and its missing
  prompt/harness, the agent-billing question before the first dispatch, `ev-provider-runbook`, AB06 judge stats before
  ds-m2-01, WIF `check_jti` re-run, D29 cadence copy).

Why not at `projects.sujaykumar.dev/xlearn/ai-builder`, where the request first pointed: ADR-0033 §10 forbids a web
admin on the shared projects origin (an XSS there reaches kubescope's cluster-admin); the xlearn `Path=/xlearn` session
cookie would reach it; the production node has ≈ 0 memory margin before MI-11a (ADR-0035 §5); the release-tag train and
rollbacks (ADR-0034) would couple to it; mi-04 moves consoles off that origin; mi-03's NetworkPolicy fences would cut
it off; `/xlearn/ai-builder` collides with `/:course/*` and the reserved-slug guard; an R-d restore would rewind it.

The owner decided in the build session (quoted where their words matter):

1. **Owner login reuses "projects-secret"** (infra's `projects-admin` `ADMIN_PASSWORD`): *"copy the projects-secret into
   skriptvalley-vps as well and use that."* Only an argon2id hash may live on the box; the plaintext is piped from
   `sops -d` on the owner's Mac straight into a hashing command — never printed, on disk, in argv or in a transcript.
2. **No second factor.**
3. **Backups:** hourly on-box snapshots, age-encrypted; off-host both by a launchd pull on the Mac over the existing
   ssh alias and by a nightly GitHub Actions pull over HTTPS with a backup-only `aib_` token (`backup:read`,
   ciphertext only; no SSH key); a restore drill.
4. **No hPanel snapshot** is needed before the proxy change.
5. **Keep `skriptvalley.com`'s landing page up** through one shared Caddy.

## Decision

### 1. Classification: build tooling, not product, not a sprint (AIB-02, AIB-04, AIB-24)

- AI Builder is **build tooling outside xLearn**: no xlearn namespace, database, NATS, JWT, service call or admin verb;
  not in the PRD; not a GA criterion; the architecture docs don't change. The v2 build proceeds without it (degraded
  mode, §11).
- It is registered as **tooling** (D44 plus a status.md "Build tooling (not sprints, D44)" row), **not** one of the 89
  sprints and never a prerequisite. build-plan.md and the execution-order data are untouched.
- Numbering: **ADR-0036 and D44 only**. D43 is reserved for the owner's $20 Anthropic org limit (recorded by the
  planning session on 2026-09-27; if D44 merges first, D43 is inserted between D44 and D42 in the newest-first log).
  The finders' proposed D45–D49 become sections of this ADR. Later ADR/D numbers go through the manager's
  `adr-number`/`d-number` locks after the peer check (`gh pr list`, `git worktree list`, ListAgents).

### 2. Placement: host, repo, release stream, isolation, edge (AIB-01, 03, 05, 07, 08, 29)

- **Host:** `https://ai-builder.skriptvalley.com` on `skriptvalley-vps` (Hostinger KVM 1: 1 vCPU, 3.8 GiB), a different
  registrable domain on a different machine. This satisfies ADR-0033 §10's "revisit only on a separate host" intent and
  §11's isolation goal **without amending either**: it isn't even same-site with kubescope, landscape or xLearn, and
  mi-04's scope is unchanged. `skriptvalley.com` *is* same-site, so cookie mutations require an exact `Origin` and
  `Sec-Fetch-Site: same-origin`.
- **Repo and release stream:** private `sujaykumarsuman/ai-builder`, its own CI (build/test/lint only) and version
  stream, deployed by `make deploy` from the Mac (`docker save | zstd | ssh docker load`, the last 3 image tarballs kept
  against the host's 24 h image-prune cron). Never on xlearn's `v*` tags, `deploy.yml` or `.release-line`; no GHCR, no
  registry, no new deploy key.
- **Zero footprint on the xLearn node:** no pod, no memory-budget row, no host-verify change; ADR-0035 §5 isn't engaged.
  The v2 sprints need no accommodation for it (mi-01…mi-05 unchanged; `projects-hub` `sharedSecret` untouched); the
  `Path=/xlearn` cookie never reaches another registrable domain; nothing is added to the reserved-slug list.
- **Runtime:** one static Go 1.26 binary (CGO off) serving the API and the embedded React 19 SPA, with the importer and
  backups in-process; SQLite (WAL) via `modernc.org/sqlite`; Go deps limited to the stdlib, `modernc.org/sqlite`,
  `golang.org/x/crypto` and `filippo.io/age`. Container caps: app 384 MiB / 0.8 CPU, Caddy 128 MiB / 0.5 CPU.
- **Edge (owner decision 5):** a new compose project `edge` runs stock Caddy (pinned by digest) at a fixed address
  `172.30.0.2` on a fixed-subnet network; it serves `skriptvalley.com` from the landing's built files **vendored** into
  the ai-builder repo and proxies `ai-builder.skriptvalley.com` to the app (no published app port). The old landing
  container is **stopped, never removed** (one-command rollback), its certificates copied, not re-issued. Ports are
  published on IPv4 only; there is no AAAA record. The app trusts `X-Forwarded-For` only from `172.30.0.2/32`, and Caddy
  overwrites it. (This replaces AIB-29's first wording, "proxy the existing landing container".)

### 3. Owner authentication: the reuse decision, compensating controls, sessions (AIB-09, 10, 11)

- **Decision (owner):** the dashboard login reuses projects-secret. This **overrides the build brief's default** (a
  dedicated credential) **and its instruction "never read it from SOPS"**, at the owner's explicit choice, in the owner's
  words above.
- **How it is stored:** the browser derives `PBKDF2-SHA256(password, "ai-builder.skriptvalley.com/owner/v1", 600000)`
  with WebCrypto and sends only that; the box stores only `argon2id(derived)` (m = 64 MiB, t = 3) as a PHC string in a
  **0400 file outside the database** (so no backup ever contains it). The KDF constants are compiled into the SPA and
  the binary; nothing about them is fetched from the network.
- **How it is set (owner runs it in their own Terminal):** `make owner-verifier` pipes `sops -d` → `yq` → a local
  hasher that computes PBKDF2 **and** argon2id on the Mac → `ssh skriptvalley-vps` → a one-shot container that
  validates the PHC and renames it into place. Only the verifier crosses ssh; the plaintext is never printed, on disk,
  in argv, in shell history or in a transcript. `../infra` is read, never edited.
- **No second factor (owner decision 2).** Compensated by a **step-up** (password re-entry, valid 5 minutes) for:
  answering attests on snapshot/irreversible-class sprints (m1-08, l-02, l-04, ga-02, m6b-04, mi-09 and asks about
  `ev-host-window`, `ev-snap-*`, `ev-ga`) and any ask flagged `step_up`; minting or revoking tokens; sign-out-everywhere;
  settings and caps; export and redaction; and **resuming a system hold** (§5). Hold never needs it.
- **Sessions:** `__Host-aib_session` (HttpOnly, Secure, SameSite=Strict, `Path=/`), server-side and revocable, 12 h
  absolute, ≤ 10 live, all revoked when the verifier changes; sign-out-everywhere. The SPA shell and static assets are
  served unauthenticated (a link from GitHub never looks signed out); only `/api/*` needs auth. Cookie mutations need an
  exact `Origin`, `Sec-Fetch-Site: same-origin` and JSON; bearer requests ignore cookies; no CORS.
- **Throttling without self-lockout:** in-memory buckets, then a per-IP lockout (5 free consecutive failures, then
  30 s doubling to 1 h), a global slow-down (never a global lockout, so a stranger can't lock the owner out), one
  argon2 verification at a time; unlock over ssh only. Unauthenticated failures are aggregated (bounded audit growth).
- **Rotation procedure:** any rotation of infra's `projects-admin` `ADMIN_PASSWORD` does **not** reach AI Builder by
  itself — the old password keeps working here — so the owner re-runs `make owner-verifier` after every rotation (the
  new fingerprint signs out every session). Settings shows "Owner credential set <date> · fingerprint". The manager
  kit reminds the owner whenever a sprint touches `projects-admin` (no v2 sprint rotates it). Switching to a dedicated
  passphrase later is the same target fed from 1Password and closes risk R-A1.
- **Owner sign-in rule:** the owner signs in only on their phone or in a browser profile no agent drives; the manager
  session denies every browser-driving tool. Verification screenshots come from a localhost replica restored from the
  live backup with a throwaway test credential.

### 4. Agent tokens and role separation (AIB-12)

- Format `aib_<role>_<52 base32>` (256-bit), stored only as SHA-256, looked up per request (instant revocation), never
  logged; the `aib_` prefix itself is on the hygiene denylist.
- **Roles:** `manager` (owner-minted in the UI with step-up, project-bound, 90 d default); `run` (minted by the manager
  per dispatch, a **narrowing-only child** bound to one run and its sprint, 24 h default, ≤ 72 h, revoked when the run
  ends) — recorded here as *not* "minting an owner-level thing"; `backup` (`backup:read` only, for the GitHub pull).
- **Agents can:** create and end runs, write heartbeats, task and gate state, remarks, PR/CI/tag facts, spend
  estimates, asks and stop notices (manager), locks, doc-debt state, the decision journal; trigger imports; read
  everything in their project, including answers and directives.
- **Agents can never:** answer asks, issue or lift directives, change settings or caps, mint manager or backup tokens,
  read the audit log or tokens, export or redact, or declare that the owner did something (an owner-action, presence
  or snapshot event completes only through an acknowledged owner answer to its attest or seeded ask, or a chat decision
  the manager journals; no bearer token completes one, the chip or Terminal session dispatched for that event
  included, since its run token also sits in the manager's Keychain: that session reports evidence). A golden authorization-matrix test over every route proves it, and the
  database enforces the same rules with CHECKs and triggers (answers only from an owner session, system-hold resume
  only with a step-up, immutable ask content, permanent revocations, run tokens only as children of a live manager
  token), so a handler bug can't open them; run tokens additionally can't create attests, stop notices or
  `global`/`action:*` asks.
- **The box is off-limits to agents,** by layers together: the manager's deny rules block the direct routes to
  `skriptvalley-vps` (ssh, scp, rsync, the ai-builder wrapper `aib-remote.sh`, `launchctl` for the backup job, docker
  contexts, a `make` run that names the ai-builder checkout, ignored errors and extra makefiles, clustered or not), and
  every ai-builder `make` target and script that reaches the box goes through that one wrapper, whose TTY guard refuses
  without a terminal in the same process as the ssh (no make flag, blanked variable or extra makefile skips it), which
  covers the hostname the deny rules can't see inside a recipe and a plain `make` after an earlier `cd`. The
  unattended Mac backup pull uses a separate backup-only key whose forced command allows only a read-only rsync of the
  backups directory. All of it is a speed bump against a mistaken agent, not a fence against a determined one (R-A3): a
  same-user agent with Bash can clone the repo under a neutral path, script its own ssh, or fake a terminal. The
  owner-presence control is the box's root SSH key served from the 1Password SSH agent with approval on use (Touch ID),
  with no copy on disk (the ai-builder runbook §10, "recommended owner hard control"), so every root ssh to the box
  waits for the owner. Root there would be every owner power at once, and the v2 build never needs that machine.
- **Storage:** the owner stores the manager token in the macOS Keychain (`aib token store`, a no-echo prompt feeding
  `/usr/bin/security` on stdin); `aib run start` stores each run token the same way under `run/<run id>` and prints only
  the run id. `aib` has the server origin compiled in, refuses the manager account when running as a run, and has no
  owner verbs. No token ever appears in argv, a file, a log or a transcript.
- **Honest limit:** every agent runs as the same macOS user, so role separation is attribution and least privilege
  against mistakes, not isolation from a hostile subagent (R-A3). The hard boundary is server-side, plus the manager's
  deny rules for `security` and browser tools.

### 5. Trust model: answers are data, typed asks, the Hold/Ship brake, non-blocking asks (AIB-13, 14, 17)

- **Answers are data, never authority.** A Claude session reads dashboard content through a tool, so the harness
  treats it as data. Asks are **typed** and bound to their id and the revision the owner reviewed:
  - `attest` — the prompt's `## Before you launch (owner)` checklist, **built by the server from the repo text** (the
    manager can't author or reword items), with typed non-secret evidence fields (dates, ids, names) and freshness
    limits;
  - `choose` — one of the options the manager listed, each labelled with the authority it falls under (charter, D40,
    AGENT.md, chat, default);
  - `inform` — a non-secret fact in typed fields or short text.
  The manager journals before acting, acts only on typed fields within the listed options, and acknowledges by answer
  revision. Free text is returned under an `untrusted` key and is quoted, never followed. **Genuinely new
  instructions** (skip a gate, change scope, delete something) go to the manager **in its chat** — from a phone through
  Remote Control — and the UI says so wherever the owner types. Frozen boards and Accepted ADRs change only by
  follow-up PRs. Answers are immutable (redaction only).
- **The brake.** Directives are **owner or system only**, append-only, per scope (`global`, `sprint:`, `run:`,
  `action:dispatch|merge|tag|prod`):
  - **Hold** only reduces authority: one tap, applied at once, with a 6-second Undo; no step-up.
  - **Resume** restores the chat-granted baseline and grants nothing new: a confirm dialog for the owner's own holds;
    a **step-up for a system hold**, because the owner didn't place it and lifting it follows a safety event (this
    extends the audit's step-up list by one action, deliberately).
  - **System holds** come only from server policy: a fresh install (bootstrap, so nothing dispatches until the owner
    first resumes), a restore, and a **red verify-live or `host-verify --cluster`** (the tag-chain stop, AIB-23, on
    `action:tag`). Agents never issue directives; they reduce authority only through **blocking asks and stop
    notices**, which the directive check treats exactly like holds. A hold the owner gives in the manager's chat is
    mirrored by a manager stop notice and lifted only in chat.
  - The manager checks before every dispatch, merge, tag push and production step; subagents before every merge, tag
    push and production step. The check **fails closed** (unreachable = don't proceed). `POST /runs` re-checks
    server-side. This is D40's in-session "hold / don't ship", now also available from the phone.
- **Asks never stall the build:** each ask has a `blocking_scope` and a D40-safe default ("not dispatched; other lanes
  keep moving"), never "do the risky thing". Subagents never wait for the owner (they end ⛔ "awaiting owner"); the
  manager waits through one background long-poll waiter and keeps other lanes moving.

### 6. D40 delegation and attestations (AIB-15, 16)

- **The owner's launch of the manager** (`docs/v2/prompts/prompt-manager.md`, with the charter
  `docs/v2/manager/MANAGER.md`) authorizes dispatching the planned sprint prompts; **D40 then applies unchanged to each
  dispatched prompt**, but only when (1) every item of its `## Before you launch (owner)` block is attested by the
  owner's acknowledged answer to that prompt's `attest` ask (31 prompts, 61 items; 28/53 still live), which the subagent
  verifies itself, and (2) no hold, blocking ask or stop covers it. AGENT.md's "Manager delegation (D44)" paragraph
  carries this authority, because a subagent can't take approval from an agent message: **the dispatch envelope grants
  nothing by itself.**
- **One attest per dispatch attempt**, re-runs included (mi-07's First-launch/Re-run groups; the conditional re-runs of
  mi-04, mi-12, m3-07, m3-15, p-01, m4-07 and m6b-04). **Unanswered means not dispatched** (the sprint shows "Needs
  you"; other lanes continue). Snapshot-class sprints (m1-08, l-02, l-04, ga-02, m6b-04; mi-09/`ev-host-window`) get
  their attest just in time with same-day evidence (≤ 6 h at dispatch, ≤ 24 h at the tag), a step-up, and one snapshot
  in flight at a time (Hostinger keeps one manual snapshot).
- **What the dashboard never carries:** keys exported for a session (ENVKEY items: m4-07, m6a-03; optional m1-10, m4-03)
  run as **owner-launched Terminal sessions** with the key in that shell only; the command the owner pastes is built by
  the dashboard from typed values (run id, variable names, the checkout path), never from agent-written text. PII
  items (l-04's email, l-05's mailto, mi-04's IP ranges) go to the manager's chat; the checkbox attests only "given in
  chat" / "I'll start it that way".
- **Owner-present work** (m6b-04, the host window, `ev-s6-recheck`, `ev-l-rehearsal`) runs as a spawn_task chip the
  owner clicks, reporting through `aib` and SendMessage.
- Agents never perform owner-only actions (hPanel, DNS, provider consoles, accounts, keys, GitHub settings), whatever an
  answer says. Gate waivers are chat decisions only, recorded with a journal reference.

### 7. Source of truth, importer, reporting contract (AIB-18, 22)

- **Git stays canonical** for the plan and durable status (sprint Status tables and status.md through PRs, the existing
  protocol, which dispatched subagents keep). **AI Builder is the live layer:** runs, heartbeats, remarks, asks,
  directives, locks, spend and the derived ready/queued states, which are never written back.
- **Importer:** in the server, anonymous (the xlearn repo is public): a git v2 `ls-refs` poll of `main` every 5 minutes
  plus manual and manager triggers; the codeload tarball by full sha with the pax-comment check; parsing is a
  golden-tested Go port of `tools/ref_import.py` (89 sprints, 689 tasks, 520 entry gates, 161 hard edges, 13 waves, 19
  workstreams, 49 events, 31 prompts / 61 before-launch items, 20 release cuts); one transaction per import; upsert by
  stable ids (sprint id, `sprint#task`, `ev-*`, tag label); `prompt-manager.md` excluded; a `build_tooling` status.md
  table key; imports only ever soft-delete repo-origin rows, never seeded or dashboard rows; repo text is scanned for
  secrets report-only.
- **Precedence:** while a run is in flight the DB wins for in-progress state; after the merge (an import covering the
  merged sha) the repo wins; differences show as drift badges. **A sprint is done only when** DB tasks, repo tasks at
  the merged sha and the release (tag verified live, where applicable) all agree. The four spikes import as done.
- **Robustness rules** while the doc debt is unfolded: the leading status emoji only; waves from the data file; events
  from status.md; never flag a prerequisite on a ✅ sprint; never derive state from ADR status lines; spend caps are
  configuration, not parsed.
- **Reporting contract:** a fixed run-report schema (`ai-builder/run-report@1`: per-task state, gates, PRs, tags,
  release action, pending smoke, hand-offs, asks, a ⛔ reason code). The manager decides only on structured fields and
  re-derives PR and CI state with `gh` before a merge or tag. The server independently confirms public facts (xlearn PR
  merges and check runs; the gateway healthz version for app tags) and labels the rest "reported".

### 8. Concurrency, locks, lanes, worktrees (AIB-19, 20)

- Lease locks with TTL and fencing, renewed by heartbeats, released when the holder's run ends: `adr-number`,
  `d-number`, `tag` (+ `tag:runner`, `tag:evalpack`), `goose:<service>`, `infra:charts/project` (and the other
  `infra:*` paths), `status.md`, `host:sujaykumar-vps`, `mac:compose`, `mac:vm`, `snapshot`, `freeze:main`,
  `prod:restart`, `owner:present`, `owner:browser`, `infra:checkout`, and `manager` (one live manager at a time).
- The manager is the single allocator of ADR/D numbers, tag versions and merge order; at most ~3 lanes; **at most one
  tag, contract, erase or GA sprint in flight**; status.md merges serialized.
- Agent-only sprints run as in-process background Agent-tool subagents in worktrees, one per prompt, running the
  existing prompts unchanged plus a short dispatch envelope. `../infra` doesn't resolve from a worktree, so the envelope
  gives absolute paths and each run makes its own **sibling** infra worktree under the `infra:checkout` lock.

### 9. Content hygiene and retention (AIB-21)

- The server rejects secret-shaped text with **422** and never echoes it: Anthropic, OpenAI, GitHub (`ghp_`,
  `github_pat_` …), AWS, age, PEM, JWT, Slack, Google, Stripe, NATS seeds, our own `aib_` tokens, bearer headers, URL
  credentials, `password=`-style assignments, plus **learner PII** (emails, `#invite=` fragments, public IPs) and
  private `xlearn-evalpack` paths, and an owner-configurable denylist (e.g. evalpack canaries once mi-07 defines them).
  Commit shas, digests, `age1…` recipients and public NATS keys pass. Bidi and zero-width tricks are stripped first.
  The UI and `aib` run the same rules before sending. Asks never request secret values. The evalpack never-copy rule
  (ADR-0027) is repeated in the kit because content can't be pattern-matched.
- Remarks are never mirrored into git. Data is kept for the project's life, exportable as JSONL and redactable on
  request (owner, step-up).

### 10. Operating safety: the tag-chain stop; notifications and D34's scope (AIB-23, 26)

- After every tag or host change the subagent reports verify-live (healthz version, images, HelmReleases) and
  `host-verify --cluster` as structured facts. **Any red result makes the server hold further tags** (a system hold on
  `action:tag`) until the owner resumes it; the manager posts an ask and pushes a notification. Nothing new runs in the
  cluster.
- **Notifications:** the only push channel is the **manager session's PushNotification** (Remote Control, the Claude
  app on the owner's phone): new asks, blockers, system holds, chips to start, answers acted on, AI Builder unreachable.
  Content is the ask id and title, never a secret. The server sends nothing and holds no bot token or notification key.
  **D34 ("no alerting in v2", ADR-0035 §3) governs xLearn production-ops alerting and does not apply to this
  build-tooling channel**, which runs off the production node and alerts on the build, not on production.

### 11. Durability: backups, restore, restore epoch, degraded mode (AIB-06, 27)

- **On-box:** hourly consistent snapshots (`VACUUM INTO`), integrity-checked, **scrubbed** of sessions, idempotency
  keys and throttle rows, gzip, **age-encrypted to two public recipients** — a dedicated AI Builder identity (its
  private key kept only on the owner's Mac) and the infra SOPS recipient — so either key restores and the box never
  holds a private key. 48 hourly + 30 daily kept. The audit hash-chain head travels with each snapshot.
- **Off-host (owner decision 3):** a Mac launchd pull every 6 hours with a backup-only SSH key (its forced command on
  the box allows only a read-only rsync of the backups directory and one marker touch; the owner installs it with
  `make backup-key-install`), with an at-least-weekly automated decrypt-and-count check; and a nightly GitHub Actions
  pull (30-day artifacts, ciphertext only).
- **The GitHub pull is HTTPS with a backup-only `aib_` token** (one endpoint, `backup:read` only, ciphertext only, 12
  pulls a day): no SSH key, no sshd change and no firewall change. The box's SSH stays IP-filtered upstream.
- **Restore** bumps a restore epoch, signs out every session, re-applies every token revocation made after the snapshot
  (a revocation journal kept outside the database), inserts a system hold and forces re-confirmation of
  answered-but-unacknowledged asks, so a restore never undoes a Hold, a revocation or a sign-out, and a stale answer is
  never acted on twice. A restore drill (both keys, a throwaway volume, a local replica) is required before go-live and
  monthly. RPO ≤ 1 h on the box, ≤ 6 h off it; git holds the durable plan status.
- **Degraded mode:** if AI Builder is unreachable, merges and tags pause (fail-closed) and reports queue locally; after
  30 minutes the manager pushes a notification, and **only the owner's chat reply** switches it to working from git and
  status.md until the service returns.

### 12. Manager runtime, pacing, continuity; web hardening (AIB-25, 20, 11)

- The manager is a long-running Claude Code desktop session on the owner's Mac (the sprints need `ssh sujaykumar-vps`,
  the SOPS age key, the sibling `../infra` checkout, gh and docker), cwd the xlearn main checkout, `/loop` in dynamic
  self-paced mode (≈ 20–30 minutes when idle), re-armed weekly; the app's keep-awake request is active and the Mac stays
  on power with the lid open or in clamshell mode; the login keychain stays unlocked; permission mode `auto` with an
  allow list and a deny list (`security`, `find-generic-password`, every browser-driving tool, every route to
  `skriptvalley-vps`) that the owner sets.
- **Re-launchable from git plus the dashboard alone:** `GET /resume` (open runs, asks, holds and stops, locks, the
  server-computed ready queue with blockers, doc debt to fold first, upcoming owner events, import state), a manager
  heartbeat (shown stale after 35 minutes, "asleep" after 90), a decision journal promotable to status.md's Decisions
  log by PR, and an orphan protocol for subagents that died with a previous session.
- **Billing: subscription only.** The build (the manager, its subagents and workflows, and the chip and Terminal
  sessions it starts) runs on the owner's Claude subscription through the Claude Code sign-in, not on the API org
  under the $20 cap (D43); that org pays only for the xLearn product's AI (the platform-AI sprints m4-*, calibration)
  under its own key names (`LLM_ANTHROPIC_API_KEY`, `LLM_CALIB_API_KEY`). A guard enforces it: at start and at every
  loop iteration the manager counts `ANTHROPIC_API_KEY`, `ANTHROPIC_AUTH_TOKEN`, `CLAUDE_CODE_USE_BEDROCK` and
  `CLAUDE_CODE_USE_VERTEX` in its environment (a count, never a value), and any hit stops dispatch until the owner
  relaunches from a clean shell. With extra usage off, a plan usage window pauses the build; the manager reschedules
  and never switches to API billing (MANAGER.md §1).
- **Web hardening:** a strict CSP (`default-src 'none'`, `'self'` for script, style, font, img, connect and manifest;
  no inline anything; **Trusted Types enforced**; `frame-ancestors 'none'`), HSTS, nosniff, `Referrer-Policy:
  no-referrer`, COOP/CORP, `X-Robots-Tag: noindex` and `robots.txt` disallow; self-hosted Inter and JetBrains Mono;
  agent and owner text rendered as text through a React-node Markdown subset with links only to
  `github.com/sujaykumarsuman/*` and the dashboard itself.

### 13. The Run Order artifact (AIB-28)

The old "xLearn v2 Run Order" artifact (https://claude.ai/artifact/FaQ1Gstv3mMoxWaASPbCdD) is updated in place with a
prominent "Superseded by the live dashboard" link to the Board view, its pre-D41 spike-week data corrected, and a data
stamp. It is **not** the host: ArtifactData writes as the owner, so agent writes would be indistinguishable from owner
answers and per-role tokens would be impossible, breaking §4 and §5.

### Accepted-risk register

| Id | Risk | Accepted by | Compensating controls | Revisit when |
|---|---|---|---|---|
| R-A1 | Owner login reuses projects-secret, which also opens kubescope (cluster-admin), landscape, the Longhorn UI and airlift's admin token | owner, 2026-09-26 | §3: plaintext never reaches the box; only argon2id(PBKDF2) stored, outside the DB and backups; throttling; step-up; 12 h sessions; rotation re-sync | any projects-admin rotation; any sign of box compromise; another tenant on the box; the owner adopts a dedicated credential; projects-secret found low-entropy |
| R-A2 | No second factor | owner, 2026-09-26 | step-up set (§3); a push for every answer acted on; device labels on answers | a forged or unexplained answer; a new irreversible action class |
| R-A3 | Agents share one macOS user: role separation is attribution, not isolation. A same-user agent with Bash can't be fully fenced by command patterns and a TTY check: they are speed bumps (a neutral-path clone, a script, a faked terminal get past them) | design panel | server-side hard boundary (§4); deny rules; the TTY guard of the one ai-builder wrapper every box call goes through (`aib-remote.sh`); pinned `aib` origin; one live manager; **recommended owner steps:** the `skriptvalley-vps` root SSH key served from the 1Password SSH agent with approval on use (Touch ID), no key file on disk, and a backup-only key with a forced command for the launchd pull (`make backup-key-install`) | agents move to separate users or machines; an unexpected Touch ID prompt for the box key |
| R-A4 | Root on skriptvalley-vps could capture the password at the owner's next login (altered JS or binary) | owner (implied by R-A1) | recommended sshd hardening (keys only); a box running only Caddy and AI Builder; pinned image digests; deploys only from the Mac | as R-A1 |
| R-A5 | Encrypted backups stored as GitHub Actions artifacts | owner (decision 3) | ciphertext only; two independent restore keys; 30-day retention; private repo | the owner prefers Mac-only copies |
| R-A6 | Single box, no HA | design panel | tooling, not product; degraded mode (§11); RPO ≤ 1 h | the build depends on the dashboard for more than steering |
| R-A7 | A chat decision (`source: chat_decision` + a `decision` journal entry) completes an owner-governed event or waives a gate on the manager's word: it rests on the manager relaying the owner's chat faithfully, and no server check can tell a relayed decision from an invented one | owner (review round 3) | the journal entry is append-only and shown in the dashboard; every decision acted on is pushed to the owner; the chat itself is the owner's own record; no bearer token completes an owner-governed event any other way (a run's own `done` is refused, the schema too) | a decision the owner doesn't recognise; agents gain a second channel to the owner |

### Decision map (the audit's deduplicated decisions)

| AIB | Decision | Section |
|---|---|---|
| 01 | Served at `ai-builder.skriptvalley.com` on `skriptvalley-vps`; ADR-0033 §10/§11 not engaged | §2 |
| 02 | Build tooling outside xLearn | §1 |
| 03 | Private repo, own CI and release stream, `make deploy` from the Mac | §2 |
| 04 | Registered as tooling (D44 + status.md), not a sprint or prerequisite | §1 |
| 05 | Production-node capacity moot; its own budget on the box | §2 |
| 06 | Fate sharing removed; degraded mode | §11 |
| 07 | No cluster or infra collisions; no sprint accommodates it | §2 |
| 08 | No `/xlearn` route, slug-guard or cookie-path collisions | §2 |
| 09 | Owner login reuses projects-secret (accepted risk, controls) | §3, R-A1 |
| 10 | No second factor; step-up | §3, R-A2 |
| 11 | Sessions and web hardening | §3, §12 |
| 12 | Agent tokens and role separation | §4 |
| 13 | Answers are data; typed asks | §5 |
| 14 | Hold/Ship brake | §5 |
| 15 | D40 delegation | §6 |
| 16 | Before-launch items become attest asks | §6 |
| 17 | Asks never stall the build | §5 |
| 18 | Source of truth, importer, drift | §7 |
| 19 | Locks and the single allocator | §8 |
| 20 | Subagent execution model and paths | §8, §12 |
| 21 | Content hygiene and retention | §9 |
| 22 | Structured reporting; remarks untrusted | §7 |
| 23 | Tag-chain stop rule (now a server-policy system hold) | §5, §10 |
| 24 | Numbering and packaging | §1 |
| 25 | Manager runtime and continuity | §12 |
| 26 | Notifications vs D34 | §10 |
| 27 | Backups and restore (two pulls) | §11 |
| 28 | The Run Order artifact is superseded, not the host | §13 |
| 29 | One shared edge Caddy (vendored landing, old container stopped) | §2 |

## Consequences

- ✅ **Zero footprint on the xLearn production node**, its release train and its rollbacks; a dashboard fix never needs
  a fleet tag, and a fleet tag never restarts the dashboard.
- ✅ **D40's approval model survives an autonomous manager:** every dispatch still rests on the owner's launch plus an
  owner attestation per prompt, and every merge and tag is behind a brake the owner can pull from a phone.
- ✅ **The dashboard can't become an instruction channel:** answers are typed data bound to listed options; no agent
  token can answer, direct or change settings; free text is labelled and never followed.
- ✅ **The manager can be relaunched from git plus `resume`** at any point of a months-long build.
- ✅ **Drift between the live layer and the repo is visible**, and "done" means three-way agreement.
- ✅ **Backups are encrypted end to end** and restorable with either of two owner-held keys; restores can't undo a brake,
  a revocation or a sign-out.
- ⚠️ **The owner's cluster-admin password now also guards a public login form on a smaller box** (R-A1, R-A4); the
  rotation re-sync is a manual step.
- ⚠️ **No second factor** (R-A2); step-up covers only the irreversible class.
- ⚠️ **Fail-closed checks pause merges and tags while AI Builder is down**, until the owner authorizes degraded mode in
  chat.
- ⚠️ **In-process subagents die with the manager session**; the orphan protocol re-derives from git and GitHub and may
  redo idempotent steps.
- ⚠️ **Spend is entered by hand** (the Anthropic Admin API is likely unavailable for an individual org); the $20 cap
  shows as pending until D43 is recorded. That cap covers the product's AI only; the build is subscription-billed
  (§12).
- ⚠️ **The importer is coupled to doc shapes**; a parse error keeps the last good snapshot and a nightly differential
  job watches `main`.

## Alternatives considered

| Option | Why not |
|---|---|
| `projects.sujaykumar.dev/xlearn/ai-builder` with an ADR-0033 §10 amendment | same-origin XSS reaches kubescope; the `Path=/xlearn` cookie leaks; ≈ 0 node memory margin; couples to the tag train; collides with `/:course/*` and the slug guard; rewound by an R-d restore |
| `ops.sujaykumar.dev/ai-builder` after mi-04 | still same-site with the consoles; needs `ev-mi5b-dns` early; mi-04's fallback IP allowlist would lock out a phone |
| A private claude.ai Artifact with a shared database as the host | ArtifactData writes as the owner: agent writes indistinguishable from owner answers; no per-role tokens |
| On the cluster (`build.sujaykumar.dev`) | node capacity and fate sharing; infra PRs racing week-1 sprints |
| A dedicated owner passphrase (the brief's default) | the owner chose reuse; recorded as R-A1 with controls; switching later is one command |
| Map `projects-admin` in through `sharedSecret`, or landscape ForwardAuth | cluster-only or same-host only; not applicable to a separate machine |
| TOTP or passkeys | declined by the owner (decision 2) |
| Manager-issued auto-holds (the audit's AIB-23 wording) | agents would issue directives; replaced by a server-policy system hold plus agent blocking asks |
| SSE for live updates | a held connection per tab on 1 vCPU, Caddy buffering pitfalls, iOS background kills; short polling with ETags is cheaper and resyncs itself |
| Postgres | fits in RAM but buys nothing at this scale; SQLite WAL on the box's disk is simpler to back up and restore |
| SSH forced-command pull for the GitHub backup job | not chosen: it needs sshd hardening and port 22 open to GitHub's runners, and a backup-only HTTPS token gives the same ciphertext with one revocable scope (§11) |
| Proxying the old landing container behind the new Caddy | breaks the one-command rollback and keeps a second Caddy resident; vendoring the built files is reproducible |
| A server-side push channel (web push, email, a bot) | a notification credential on the box and a channel outside the owner's chat; the manager's PushNotification suffices |
| Registering AI Builder as sprints `ab-01`/`ab-02` | it is tooling, not v2 product scope; would change the 89-sprint plan and its counts |

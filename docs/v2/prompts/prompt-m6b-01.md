# Prompt — Sprint m6b-01 · VoiceShell adapter + SDP broker + sideband

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-m6b-01.md`](../sprints/sprint-m6b-01.md)   ·   **Milestone:** M6b (voice, one shell)   ·   **Prereqs:** [mi-13](../sprints/sprint-mi-13.md), [ds-m6b-01](../sprints/sprint-ds-m6b-01.md), [m6a-06](../sprints/sprint-m6a-06.md)

## Before you launch (owner)

Launching this prompt attests these are done (D40). If one turns out to be missing, land everything that doesn't depend on it and record the gap as ⛔ in `status.md`; don't wait.

- [ ] *Optional, only for step 11's ≤ 2-minute live check (≈ $0.10):* your own OpenAI key, entered by you in the local compose stack's Settings and set as the `interview` default (the agent never types a key). Without it, the live leg is recorded as deferred to m6b-04, not ⛔.

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md) — conventions, service boundaries, land-and-sync.
- The plan: [`../sprints/sprint-m6b-01.md`](../sprints/sprint-m6b-01.md) — the interface sketch, the precondition table with its typed errors,
  the decoder rules and the test list are spelled out there. Follow them.
- Research [t6 §3](../research/t6-realtime-interviewer.md#3-architecture--media-path) (media path, credential model, sideband and audio copies,
  browser control surface), [§4](../research/t6-realtime-interviewer.md#4-session-state-machine--the-owners-failsafes) (classifier, failure
  table: second tab, unconfirmed SDP), [§5](../research/t6-realtime-interviewer.md#5-the-coding-round) (director push, hints, runs),
  [§6](../research/t6-realtime-interviewer.md#6-assessment) (brain-authored evaluative speech, distress on words),
  [§7](../research/t6-realtime-interviewer.md#7-privacy-consent-retention--accessibility) (data inventory, consent, logging, S1–S4),
  [§11](../research/t6-realtime-interviewer.md#11-what-t6-constrains-downstream) (ADR-0007 and catalog amendments),
  [§13](../research/t6-realtime-interviewer.md#13-owner-decisions--resolved-2026-09-24-they-override-the-body-where-they-conflict) (D29 overrides
  §5's 20 s code debounce: 2–3 s + every turn, one replaceable current-screen item), and the **S6 results** — t6 **§16** "S6 results (spk-04, <date>)"
  appended by [spk-04](../sprints/sprint-spk-04.md), the scrubbed fixtures in `docs/v2/research/t6-s6-fixtures/` (`index.json`), and the
  spike-results row in [`../status.md`](../status.md).
- [ADR-0032](../../adr/0032-realtime-ai-mock-interviewer.md) §2 (architecture), §3 (D29), §6 (EU/EEA, caps), §7 (ADR-0007 amendments);
  [ADR-0007](../../adr/0007-ai-coach-byo-key-and-secrets.md) (key handling); [ADR-0005](../../adr/0005-data-ownership-and-migrations.md)
  (schema-per-service, goose); [ADR-0034 §2](../../adr/0034-v2-release-labelling-gating-and-rollback.md#2-feature-gating-three-tiers-no-flag-service)
  (T-3 cohort, dark launch); [ADR-0035 §2](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#2-nats-auth-nkey-users-fine-acls-server-first)
  (NetworkPolicy standing rule), [§4](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#4-limits-inventory) (L6, L19).
- Neighbours: [mi-13](../sprints/sprint-mi-13.md) (the `/segments` route contract: 20 KiB, 20 s, no retry, no body logs),
  [m6a-01](../sprints/sprint-m6a-01.md) / [m6a-02](../sprints/sprint-m6a-02.md) / [m6a-04](../sprints/sprint-m6a-04.md) (FSM, segment log,
  consent, brain + classifier + probe, snapshots, bounded SSE), [m1-10](../sprints/sprint-m1-10.md) (catalog, `key_default`, typed errors),
  [m6b-02](../sprints/sprint-m6b-02.md) (what is deliberately left for the next session).
- Boards (frozen; read for states and copy only): `design-system/screens/v2/AB29-voice-preflight-notices.html`, `AB30-voice-live-hud.html`.
- Code: `cmd/coach/main.go`, `internal/coach/{service.go,handlers.go,providers.go,catalog.go}`, `internal/coach/interview/` (M6a),
  `internal/coach/store/{migrations,queries,store.go}`, `internal/platform/secrets`, `internal/identity/service.go`
  (`GET /internal/accounts/{id}`), the gateway interview proxy in `internal/gateway/` (M6a + mi-13's `/segments` row),
  `internal/gateway/openapi_drift_test.go`, `docs/architecture/{services,api,data-model}.md`, `docs/architecture/openapi.yaml`, `docker-compose.yml`.

## Context

M6a shipped the **text** interviewer dark in a `v2.0.x` patch: coach owns the interview FSM, clock, transcript, segments, consent and the
director brain on the learner's `interview` key. [mi-13](../sprints/sprint-mi-13.md) then shipped the voice gates: the SPA's
`Permissions-Policy`, the sibling camera/mic deny, coach at 500m / 256 Mi with a 60 s grace and make-before-break rolling, coach's 443
egress, and the gateway's **20 s** `POST /api/interviews/{id}/segments` route that proxies to coach (whose handler is still a 404). S6 picked
**one** OpenAI shell. This sprint gives coach voice: the browser sends its SDP offer; coach decrypts the key, creates the provider session
server-side and returns the answer — **no credential ever reaches the browser** — then holds an outbound sideband that feeds transcripts,
usage and errors into the FSM while **dropping audio copies by event type, unparsed**. Audio flows browser ↔ OpenAI only. The lease,
drain, cost, caps and modes are [m6b-02](../sprints/sprint-m6b-02.md); the UI is [m6b-03](../sprints/sprint-m6b-03.md).

## Entry gates — verify first (stop and report if any is unmet)

- [ ] [mi-13](../sprints/sprint-mi-13.md)'s `v2.0.x` patch is live: as the owner, `POST /xlearn/api/v1/interviews/<nonexistent>/segments` returns
      coach's 404 through the gateway; status.md records coach 500m / 256 Mi, grace 60, 1/0 and the 443 egress. (Use an already-signed-in
      browser session for the owner call if you have one, never credentials; otherwise an anonymous POST → 401 plus mi-13's status.md
      record is the check.)
- [ ] [ds-m6b-01](../sprints/sprint-ds-m6b-01.md)'s PR is **merged** (AB29–AB30 on `main`; the merge is the freeze, D40).
- [ ] The S6 results exist and the decision is **not** "both fail"; they name the shell, M7, M13, the event filter, the current-screen
      finding, the sideband URL, the audio event names and the fixture path. [ADR-0032](../../adr/0032-realtime-ai-mock-interviewer.md) reads Accepted.
- [ ] `account.region` exists and the acceptance step sets it (v1.17.0 live).
- [ ] [m6a-06](../sprints/sprint-m6a-06.md)'s patch is live (interview FSM, `interview_segment`, `interview_turn`, `interview_consent`, brain,
      classifier, probe, bounded SSE, snapshots, interview proxy + cohort gate, `coach admin interviews --live`).
- [ ] Parallel sessions: `gh pr list`, `git worktree list`, ListAgents — no open PR adds a coach migration or edits `internal/coach/interview/`,
      `internal/coach/catalog.go`, the gateway interview proxy or identity's internal account handler.

## Do this (in order)

1. **[X] Branch** `feat/m6b-01-voice-broker` off an up-to-date `origin/main`.
2. **[X] Reconcile** (plan task 1): write the S6 fact table into your PR draft; confirm the live route name (`/segments`); if
   `docs/adr/0007-ai-coach-byo-key-and-secrets.md` lacks its *Amendment (ADR-0032)* section, add it with ds-m6a-01's wording (SDP brokering
   only, each creation logged; the Realtime `client_secrets` fallback, 30–60 s TTL, **only if S6 showed brokering failing**; the key as a
   `[]byte` per live segment, dropped on `interrupted`/`paused`, never in panics or logs); record the design freeze if ds-m6b-01's session
   didn't (idempotent; skip any edit already done) — `sprint-ds-m6b-01.md` Status rows + _Overall_ ✅, status.md Artboards AB29–AB30
   "frozen (PR #, date)", and `ev-freeze-ds-m6b-01` ✅ ("automatic at the merge") if status.md still lists it.
3. **[X] Migration + sqlc** (plan task 4, last bullets): the next free coach goose version (expand only) — `interview_segment` gains
   `purpose` (`live`\|`preflight`, default `live`), `client_id`, `confirmed_at`; **`interview_event.kind` is widened with `caption` and
   `segment`** (drop and re-add m6a-01's CHECK with the two values; or extend the Go enum if the set is only enforced there), and
   m6a-04's **S1 kind allowlist + its test** gain both kinds (`segment` never carries `provider_session_id`, model, SDP or key material).
   Voice transcripts are written as `interview_turn.source='server'` — `source` keeps m6a-01's two values (`client` is m6b-02's mirror).
   Everything else exists from m6a-01 (`kind`, `model`, `provider_session_id`, `opened_at`, `closed_at`, `close_reason`, `usage`,
   `cost_micros`; the voice states are already in the FSM enum). **No SDP column.** `sqlc generate`; `sqlc diff` clean. Add the voice
   consent kinds to `internal/coach/interview/consent.go` with the AB29 F2 strings.
4. **[X] Catalog + transport + `voice.Segment` + adapter** (plan task 2): finalise the winner's `voice_shell` row (billing shape,
   `session_cap_s`, `voices`, `min_tier`, `billing_url`, `probe_model`, `as_of` = today's pricing page); delete the loser's row. Create
   `internal/coach/interview/voice/` with the `VoiceShell`/`Sideband` transport (incl. `Detach` ≠ `HangUp`), `voice.Segment` implementing
   m6a-02's `Segment` (`Open`/`Turn`/`Cue`/`Close` — so every FSM exit that closes a segment hangs up), the **winner's adapter only** (from the
   S6 fixture shapes, on coach's `OPENAI_BASE_URL`, sideband scheme derived from it), and the test fakes. Validate the snapshot's
   `interviewer.voice_default` against the catalog `voices` at a voice start (422 `voice_unavailable`). Extend the setup/pre-flight DTO with
   `voice{available, reason, shell, voice, data_channel}`.
5. **[X] Segment key** (plan task 3): extend m6a-02's segment-owned `[]byte` to the sideband, director pushes, reaper and supersede paths;
   add the `[redacted]` formatters if missing; sideband goroutines `recover()` and log only the panic type.
6. **[X] SDP broker** (plan task 4): coach `POST /interviews/{id}/segments` with the checks **in the plan's order** and their typed errors
   (`not_found`; `lease_lost`; `illegal_transition` — with the `interrupted` → `cleared` rule, fired only after `CreateSession` succeeds so a failed re-prime keeps the grace; `consent_required {kinds}`;
   `voice_unavailable_region` — identity `GET /internal/accounts/{id}` `region`, add the field there if missing, cache 60 s, NULL refused, and
   the same refusal on `POST /interviews` with `mode=voice`; `voice_key_required`; `sdp_invalid`; the `voiceAdmission` no-op hook; an
   unconfirmed same-client segment replaced); `CreateSession` under 15 s; the mint row + one slog line without SDP/key/IP; attach the
   sideband; `201 {segment_n, sdp, shell, confirm_by}` with `no-store`; classifier mapping to HTTP (`voice_model_access`, `provider_quota`,
   `provider_auth`, `provider_unavailable`, `sdp_timeout`) with the FSM effect **by source state** (the plan's second table:
   `preflight` → `preflight_fail(model_access|quota|auth)` → `setup`; `connecting` → `interrupt(reason)` through a new **voice-only**
   `connecting —interrupt→ interrupted` data-table row; `interrupted` → stays, a failed re-prime; auth → key disabled, `paused`);
   **no retry**. The `preflight` purpose speaks AB29 F5's fixed greeting and hangs up at 30 s. **Do not build the `client_secrets`
   fallback** — ADR-0032 §7 accepts it only conditionally (Realtime, only if S6 showed brokering failing); if S6 says brokering fails, stop
   and report (a gate failure, not a review: ⛔ in status.md): it needs mi-13's `connect-src` widened in its own gateway PR, in a follow-up
   sprint.
7. **[X] Confirm, reaper, supersede** (plan task 5): coach `POST /interviews/{id}/segments/{n}/confirm` + one gateway interview-proxy row
   (default budget, cohort gate, `aud=coach`) + `openapi.yaml`/`api.md`; confirmation also from the first media event (then m6a-01's
   `segment_open` → `live`); the reaper as a job in m6a-01's sweeper (closes and hangs up segments unconfirmed after 15 s; once at startup);
   supersede through m6a-01's client lease — another tab's `attach` hangs up the open voice segment and fires `interrupt(tab_closed)`.
   Write the `segment{event: opened|confirmed|closed, n, purpose, reason}` event-log kind (added in step 3) with each segment change.
8. **[X] Sideband** (plan task 6): `go get github.com/coder/websocket`; dial with the segment key (compression disabled); **`conn.SetReadLimit`
   right after the dial** — the library's default is 32 KiB per message on `Reader` too, and a bigger `session.updated` (instructions echo)
   or audio frame would close the sideband with `StatusMessageTooBig`; use the larger of 1 MiB and 2× the largest S6 M8 fixture frame, and
   state it in the PR; stream frames via `conn.Reader`, each read to its end; the **512-byte prefix scanner** — audio types →
   `io.Copy(io.Discard, r)`, allowlisted types → `io.LimitReader(256 KiB)` + unmarshal (longer ones drained and dropped), everything else
   dropped; dropped frames tallied per segment and logged once at close (no metric); the event filter too if S6 found one; tamper →
   reconfigure, then hang up + `session_tampered` caveat.
9. **[X] Director, current screen, captions, ask** (plan task 7): the voice sink for m6a-02's director (context vs speak per shell;
   brain-authored evaluative speech; a **voice-mode fixed disclosure turn** with AB30 F1's text spoken as the opener — never m6a-02's text
   disclosure, which says "can't see or hear you"; test it); `POST /interviews/{id}/ask {repeat|rephrase}`
   (+ gateway row); the single replaceable current-screen item on snapshot change (2–3 s, never unchanged) and every final candidate turn — or
   diffs-only if S6 found no in-place replace; hints recorded before push; run echo as a note; final transcripts → `interview_turn`
   (`source='server'`, `truncated`) + the `turn` event + `last_input_at`; the distress text check and the never-list scan on voice turns;
   partial captions as the `caption` event-log kind added in step 3 (coalesced ≥ 300 ms, ≤ 4 KiB, C4 retention) — **one caption path for
   both shells**.
10. **[X] Tests** (plan task 8): unit (offer table, precondition order, per-source-state error effects + the new FSM row in the table test and
    model check, decoder garbage-after-prefix + allocation benchmark at 32 KiB and 512 KiB, tamper, the S1 allowlist with `caption`/`segment`,
    the voice disclosure), **> 32 KiB `session.updated` and audio frames through the fake shell's real WebSocket** (no `StatusMessageTooBig`,
    sideband still attached), integration on real Postgres with the fake shell (201, every typed error, mint row clean, 15 s reaper with a fake clock, lease takeover → hang-up +
    `lease_lost`, `interrupted` → `cleared` (and a failed re-prime keeps `grace_until`), every FSM exit from `live` hangs up, key zeroed, `caption` → `turn` order), **the no-audio-persisted
    canary scan** over every coach table + logs + panics, the extended canary for `/api/interviews/*` and the decoder, the fixture contract,
    the gateway confirm/ask routes + OpenAPI drift. `go test ./...`, `go vet`, lint, `sqlc diff`, the web suite.
11. **[X] Optional live check** (plan task 8, last bullet), pre-approved by launching this prompt (D40): verify the before-launch item —
    the owner's own OpenAI key in the local compose Settings as the `interview` default (you never type a key). If it's there, record the
    voice consent for your test interview (compose test data; the audio is a fake clip), drive Chrome with fake media and the DevTools
    snippet (`purpose:'preflight'`), ≤ 2 minutes, ≈ $0.10; confirm `connected`, the greeting caption on SSE, the hang-up,
    `coach admin interviews --live` empty, and repeat the no-audio scan. Otherwise record "live leg deferred to m6b-04" (not ⛔).
12. **[X] Docs + status** (plan task 9), then ship: see **Ship** below (**no tag**).

## Constraints

- **Service boundaries ([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)):** coach owns the interview, the key and every provider
  call; the gateway only proxies (no credential, no SDP parsing, no provider origin); identity only gains a read field. No cross-schema reads.
- **goose + sqlc:** expand-only migration on schema `coach`, next free version at rebase; commit generated code; `sqlc diff` clean in CI.
- **No new NATS events:** coach's outbox/inbox and NATS subjects are unchanged (the new `caption`/`segment` kinds live on coach's own
  per-interview event log), so no ACL PR; consumers-before-producers is not in play.
- **Privacy:** SDP, audio, the key and prompts/completions are never logged or persisted; transcripts only in `interview_turn`; the canary
  test is the gate. The browser never receives a token.
- **One shell; the fallback is not built here:** only the S6 winner's adapter. ADR-0032 §7's `client_secrets` fallback is accepted but
  conditional (Realtime only, only if S6 showed brokering failing) and needs mi-13's `connect-src` widened to the provider origin in its own
  gateway PR, in a follow-up sprint — so if it's needed, stop and report (a gate failure: ⛔ in status.md); mi-13's CSP stays as it is in
  this session.
- **No retries** on session creation anywhere in coach.
- **Dark:** everything behind M6a's T-3 cohort gate; no new flag; no UI (`web/` untouched — `theme.css` rules apply to m6b-03).
- **GitOps / ops:** no infra change expected; if coach → identity `:8081` is missing from coach's egress, that is its own infra PR merged via
  Flux (never `kubectl apply`). D34: no metrics, alerts or opscheck — `coach admin interviews --live` is the only ops read (the decoder's
  per-segment dropped-frame tally is one log line at segment close, nothing exported).
- **Memory-sum rule:** no new pod here; the decoder must stay inside S6 M8 (record the benchmark).
- **Keys:** never type or paste an API key into any field; the live check runs only on a key the owner entered himself before launch.
- **Parallel sessions:** re-check peers' PRs and worktrees before opening the migration PR; claim an ADR number only after that check.

## Deliverables

- `internal/coach/interview/voice/` (transport interfaces, `voice.Segment`, winner adapter, decoder, sideband, fakes) + the broker, confirm and
  ask handlers, the reaper job, the lease-takeover hook, the voice consent kinds.
- `internal/coach/catalog.go` final `voice_shell` row; the coach migration + sqlc output; identity's internal account `region` (if missing).
- Gateway: the confirm and ask route rows; the `voice` block passthrough; `openapi.yaml` + `api.md`; `events.md` (`caption` and `segment` kinds).
- The PR description: the S6 fact table and the two fake-provider endpoints [m6b-03](../sprints/sprint-m6b-03.md)'s compose walk needs.
- `go.mod`/`go.sum` with `github.com/coder/websocket`.
- Tests listed in the plan; ADR-0007 carrying the ADR-0032 §7 amendment; `services.md`/`data-model.md` updates.

## Update status

- [`../sprints/sprint-m6b-01.md`](../sprints/sprint-m6b-01.md): each task 🔄 → ✅ (⛔ with a reason); _Overall_ ✅ when all are.
- [`../sprints/sprint-ds-m6b-01.md`](../sprints/sprint-ds-m6b-01.md): its Status rows and _Overall_ ✅ (the freeze), only if its session didn't already.
- [`../status.md`](../status.md): Sprint board rows (ds-m6b-01 ✅ if not already, m6b-01 ✅); **Milestones** M6b 🔄; **Artboards** AB29–AB30
  "frozen (PR #, date)" if not already; owner event `ev-freeze-ds-m6b-01` ✅ ("automatic at the merge") if status.md still lists it; flags —
  none new; **Decisions log** — the S6 winner + catalog `as_of`, precondition order and error
  codes (with their per-source-state FSM effects), confirm = browser call or first media event, supersede through the client lease, the
  current-screen mechanism (or diffs-only), the `caption` and `segment` event kinds (widened `kind` set), voice turns as `source='server'`,
  the sideband read limit, the `session_tampered` caveat, "brokering works; `client_secrets` fallback not needed", whether the live check ran (or deferred to
  m6b-04); a **hand-off line for m6b-03** — the `voice` block fields, the routes (`segments`, `confirm`, `ask`), the event kinds and the
  fake-provider endpoints for its compose walk.
- No ADR expected (ADR-0032/0007 cover it). If you must diverge from t6 §3 (e.g. brokering impossible), stop and report rather than wait
  (a gate failure, not a review): record the options and a recommendation, and ⛔ "needs owner decision" in status.md. An ADR number is
  claimed only after the parallel-sessions check.

## Done when (acceptance)

- [ ] Browser ↔ provider WebRTC with the SDP brokered via coach: the fixture contract and fake-shell integration pass, and the live check
      connected (or is recorded as deferred to m6b-04); no browser response carries a credential
- [ ] **No audio bytes persisted or logged** (canary scan = 0 hits); audio frames are discarded unparsed
- [ ] Every segment creation is a mint row with no SDP/IP; SDP never in logs
- [ ] Unconfirmed sessions hung up at 15 s; a second tab's lease takeover hangs up the first tab's session; ≤ 1 open segment; every FSM exit
      from `live` hangs up
- [ ] The key is a `[]byte` per live segment, zeroed on every path that ends it, never in logs or panics
- [ ] Server-side preconditions: client lease, FSM state, voice consent (`consent_required`), EU/EEA (`voice_unavailable_region`, NULL refused,
      voice creates too), OpenAI interview key + catalog shell, audio-only offer
- [ ] Captions reach the event log as `caption` → `turn` (one path for both shells)
- [ ] Tamper re-applied once, hung up on the second
- [ ] The sideband survives > 32 KiB frames (`SetReadLimit`; real-socket test); `caption`/`segment` in the widened `kind` set and the S1 allowlist
- [ ] CI green; merged; no tag

## Ship (land-and-sync — owner approval pre-granted)

> Launching this prompt is the owner's approval for every change it makes (D40); don't stop for review.

1. Branch, then conventional commit(s) with the attribution lines, then push, then a PR in every repo touched (`../infra` PRs first where the order requires it; infra PRs are never folded into a tag). The xlearn branch is `feat/m6b-01-voice-broker`.
2. Once CI is green (fix, then merge, on failure), squash-merge. Never enable auto-merge.
3. **Release action — merge only (ships dark in m6b-03's `v2.0.x` patch):** Nothing deploys; it ships in the `v2.0.x` patch cut by [m6b-03](../sprints/sprint-m6b-03.md) (or rides an earlier peer `v2.0.x` patch, still dark: cohort-gated API, no UI). Don't tag. No infra PR, unless coach → identity `:8081` was missing from coach's egress: then that is its own `../infra` PR (ADR-0035 §2 standing rule), merged on its own; infra has no CI, so paste the rendered-policy diff into its PR body and merge on it.
4. Update status: the sprint file and `docs/v2/status.md`, in the same PR or a follow-up docs PR merged the same way.
5. Run `git checkout main && git pull` in every repo touched (xlearn, and `../infra` if the egress PR was needed). If a clean peer worktree holds `main`, use `git -C <worktree> merge --ff-only origin/main` and then `git switch --detach main`.

# Prompt — Sprint mi-13 · M6b gates: Permissions-Policy, camera/mic deny, coach sizing, SDP route, WSS egress (MI-16) → v2.0.x patch

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-mi-13.md`](../sprints/sprint-mi-13.md)   ·   **Milestone:** MI (rollout step MI-16, the M6b gates)   ·   **Prereqs:** [mi-11](../sprints/sprint-mi-11.md), [spk-04](../sprints/sprint-spk-04.md), [m6a-06](../sprints/sprint-m6a-06.md)

## Before you launch (owner)

Launching this prompt attests these are done (D40). If one turns out to be missing, land everything that doesn't depend on it and record the gap as ⛔ in `status.md`; don't wait.

- [ ] Your xLearn account has an OpenAI API key saved in Settings (coach BYO), for step 8's coach-chat smoke on `api.openai.com:443`. Only that smoke depends on it.
- [ ] The last Hostinger weekly image in hPanel is ≤ 7 days old: the coach-sizing PR restarts coach (rollout §2.2). If it's older, land the rest and mark that PR ⛔.

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md) — conventions, GitOps, land-and-sync.
- The plan: [`../sprints/sprint-mi-13.md`](../sprints/sprint-mi-13.md) (the route table, mechanism order, WebRTC snippet and release checklist are spelled out there).
- Research [t6 §3](../research/t6-realtime-interviewer.md#3-architecture--media-path) (media path, credential model, **infra-needs table**: SDP route, coach resources and rollout, Permissions-Policy, CSP, sibling deny), [t6 §9](../research/t6-realtime-interviewer.md#9-phased-plan) row P1, [t6 §11](../research/t6-realtime-interviewer.md#11-what-t6-constrains-downstream).
- [ADR-0032](../../adr/0032-realtime-ai-mock-interviewer.md) §2 (architecture, small changes), §6 (caps); [ADR-0033 §12](../../adr/0033-invite-only-admission-and-owner-admin.md#12-authz-deltas-across-v2) row 15 (same origin → Permissions-Policy at M6b).
- [ADR-0035](../../adr/0035-v2-operations-nats-auth-limits-capacity.md) §2 (NetworkPolicy standing rule), §4 (L5, L6, L19), **§5 (memory-sum rule; the coach P1 row)**.
- [ADR-0034](../../adr/0034-v2-release-labelling-gating-and-rollback.md) §1.1 (patches after GA), §2 (cohort gate), §4.1 (rollback), **§6 (release checklist)**.
- [`../rollout-plan.md`](../rollout-plan.md) §2 row **MI-16**, §3 row **M6b**, §4 (M6a / M6b gates), §2.2 (operating rules).
- The S6 results: `## 16. S6 results` in [t6](../research/t6-realtime-interviewer.md) (appended by [spk-04](../sprints/sprint-spk-04.md)) — shell, M7, **M8 (coach limits confirmed or revised, `GOMEMLIMIT`)**, browser list, harness CSP — and [m6a-06](../sprints/sprint-m6a-06.md) / [m6a-01](../sprints/sprint-m6a-01.md) (the interview proxy, cohort gate, `coach admin interviews --live`).
- Code: `internal/gateway/` — `security.go` (m1-04's security-header middleware, applied in `Handler()` in `gateway.go`; confirm with `grep -rn Content-Security-Policy internal/gateway`), `gateway.go` (`serveStatic`'s SPA fallback → `serveIndex`), `bff.go` (`apiRoutes()` route table, `Doc` field), the m6a interview proxy, `coach.go`'s 10 s `httpc`; [m1-04](../sprints/sprint-m1-04.md) task 6 (the CSP it sets, on SPA shell responses only), `internal/gateway/openapi_drift_test.go`, `internal/platform/httpx/httpx.go` (server timeouts), `docs/architecture/openapi.yaml`, `docs/architecture/api.md`.
- Infra (edit only through PRs): `../infra/apps/xlearn-coach.yaml`, `../infra/apps/{airlift,projects-hub,landscape,kubescope}.yaml`, `../infra/infrastructure/storage/ui.yaml` (raw Longhorn route; the no-cross-namespace comment), [mi-04](../sprints/sprint-mi-04.md)'s raw routes — `../infra/apps/airlift-admin-routes.yaml`, `../infra/apps/ops-redirects.yaml`, and `../infra/apps/kubescope-route.yaml` if its task 5 fired — `../infra/charts/project/templates/ingressroute.yaml` + `values.yaml` (does chart 0.3.0 from [mi-01](../sprints/sprint-mi-01.md) carry `route.responseHeaders`?), [rollout §2.1](../rollout-plan.md#21-chart-030-knob-list) (the frozen knob union), `../infra/hack/chart-diff.sh`, `../infra/infrastructure/configs/traefik-config.yaml`, the MI-15 egress values from [mi-11](../sprints/sprint-mi-11.md).

## Context

v2.0.0 (owner-facing GA) is live and M6a — the text interviewer — has shipped **dark** to the cohort in a `v2.0.x`
patch. M6b adds voice: the browser talks WebRTC **directly to OpenAI** (no media on the node), coach brokers the
SDP with the learner's BYO key and holds an outbound sideband WebSocket. Before any voice code, five small platform
changes must be live (MI-16): the SPA's `Permissions-Policy`, a camera/mic deny on the sibling apps that share the
`projects.sujaykumar.dev` origin, coach sized for voice, coach's WSS egress, and a gateway SDP route with a **20 s**
budget (today's coach client times out at 10 s). This sprint ships them — the gateway parts in a `v2.0.x` **patch**
with no interviewer behaviour change — so [m6b-01](../sprints/sprint-m6b-01.md) can build its SDP broker on a live route.

## Entry gates — verify first (stop and report if any is unmet)

- [ ] MI-3 merged ([mi-01](../sprints/sprint-mi-01.md)): the chart has `terminationGracePeriodSeconds` and `strategy.rollingUpdate`
- [ ] MI-15 merged ([mi-11](../sprints/sprint-mi-11.md)): `k3s kubectl get networkpolicy -n xlearn` shows coach's egress policy
- [ ] MI-5b live ([mi-04](../sprints/sprint-mi-04.md)): console hosts final
- [ ] S6 result known ([spk-04](../sprints/sprint-spk-04.md)) and ADR-0032 Accepted ([ds-m6a-01](../sprints/sprint-ds-m6a-01.md))
- [ ] [m6a-06](../sprints/sprint-m6a-06.md)'s `v2.0.x` patch live (interview proxy + cohort gate + `coach admin interviews --live` on prod)
- [ ] `.release-line` = `2`; every `xlearn-*` ImagePolicy `>=1.0.0 <3.0.0` ([ga-02](../sprints/sprint-ga-02.md))
- [ ] `ssh sujaykumar-vps 'bash /root/host-verify.sh --cluster'` (or `ssh sujaykumar-vps 'bash -s -- --cluster' < ../infra/hack/host-verify.sh`): memory sum ≥ the coach resize's cost + 0.05 GiB inside the rule — cost = 2 × (S6 M8 memory limit − 128 Mi), so **≥ 0.3 GiB** at the default 256 Mi (record the number)
- [ ] No open peer PR on the gateway's headers/interview routes, `apps/xlearn-coach.yaml`, the sibling HelmReleases or the chart (`gh pr list` in both repos, `git worktree list`, ListAgents)

## Do this (in order)

1. **(X) Parallel-sessions check + branches:** `feat/mi-13-voice-gates` in xlearn; `feat/sibling-camera-mic-deny`, `feat/xlearn-coach-voice-sizing` (and, only if needed, `feat/xlearn-coach-wss-egress`) in `../infra`.
2. **(X) Permissions-Policy + CSP tests** (plan task 1): add `Permissions-Policy: camera=(self), microphone=(self)` in `internal/gateway/security.go` (m1-04's header middleware); keep the CSP exactly (`script-src 'self'`, `connect-src 'self'`, no `webrtc` directive). Tests: the Permissions-Policy value on `/`, a deep SPA route and `/api/v1/healthz`; the CSP **only on the SPA shell** (`/`, the deep route — m1-04 doesn't set it on API responses), parsed into directives and tokens: `connect-src` exactly `'self'` (no `openai` host), `script-src` exactly `'self'`, no `webrtc` directive, and no **bare scheme source** token (`https:`, `wss:`, `http:`, `ws:`; `data:` only in `img-src`) — m1-04's host sources (`https://fonts.googleapis.com`, `https://fonts.gstatic.com`, `https://github.com`) must pass, so compare tokens, not substrings.
3. **(X) The SDP route** (plan task 2): `POST /api/interviews/{id}/segments` in the m6a interview proxy (route-table entry in `bff.go`'s `apiRoutes()`) — `aud=coach` JWT, the m6a cohort gate (non-cohort → unknown-route 404), 20 KiB `MaxBytesReader` → typed 413 `payload_too_large`, its own `http.Client{Timeout: 20s}` + a 20 s context → typed 504 `sdp_timeout`, **no retry**, status/body/`Content-Type` passthrough with `Cache-Control: no-store`, **no body logging**. Route-table entry (`Doc: true`), `docs/architecture/openapi.yaml` path, `api.md` row. Tests with an injectable budget (plus one pinning 20 s): under/over budget, 413 without calling coach, one upstream request on 502, cohort vs non-cohort, no `v=0`/`a=candidate`/`a=fingerprint` in captured logs, the other coach routes still on the 10 s client.
4. **(X) WebRTC under the CSP** (plan task 3): check the S6 note (harness under `connect-src 'self'`; any page-side provider request → **stop and report**); run the compose loopback snippet in Chrome with fake media as the owner and paste `{state: 'connected', violations: [], mic: true}` into the PR.
5. **(X) CI + PR:** `go test ./...`, `go vet`, lint, the web suite, `sqlc diff` (no queries change — must stay clean); open the xlearn PR; don't tag yet.
6. **(I) Sibling deny PR** (plan task 4): re-read `k3s kubectl get ingressroute -A`; `curl -sI` each sibling first (merge any existing policy); list mi-04's raw routes too (`airlift-admin`, `airlift-admin-block`, `ops-redirects`, `kubescope-route` if present). Mechanism: (a) chart 0.3.0's `route.responseHeaders` knob (from mi-01) if present → set values on the chart-rendered siblings (airlift, projects-hub, landscape, kubescope unless its chart route is disabled), `hack/chart-diff.sh` showing only those releases' route + Middleware change, no chart bump → else (b) add that knob in a chart 0.3.x minor as a **deliberate, recorded exception** to rollout §2.1 (decisions log), chart-diff showing the `helm.sh/chart` label on every release plus the sibling routes/Middlewares only → plus (c) same-namespace references on the raw routes: a raw `headers` Middleware for `longhorn-system/longhorn-ui` and for `kubescope-route` (if present); the airlift admin routes reference the chart-rendered `airlift-headers`; redirect-only `ops-redirects` recorded n/a. No entrypoint middleware, no cross-namespace refs. Add the "every new app carries the deny" note to the infra README. Merge; verify each sibling (incl. `https://ops.sujaykumar.dev/airlift/admin`) answers `permissions-policy: camera=(), microphone=()` and xLearn is unchanged.
7. **(I + H) coach sizing PR** (plan task 5): limits = S6 M8's values (default 500m / 256Mi; today 250m / 128Mi), `GOMEMLIMIT` (S6 M8, default `200MiB`), `terminationGracePeriodSeconds: 60`, `strategy.rollingUpdate` maxSurge 1 / maxUnavailable 0. Merge **only when** `ssh sujaykumar-vps 'sudo k3s kubectl exec -n xlearn deploy/xlearn-coach -- coach admin interviews --live'` is empty; then `host-verify --cluster` — the memory sum (Δ = 2 × (M8 memory − 128 Mi): limits + surge; ≈ +0.25 GiB at the default) must stay inside the rule; record before/after and the margin left for m6b-02. Margin < Δ + 0.05 GiB before (0.3 GiB at the default) → stop, ⛔ "R0 trim first".
8. **(I) coach WSS egress** (plan task 6): read coach's rendered egress; TCP 443 to non-cluster already there → record "covered by MI-15 (infra#N)", no PR; narrower → extend in its own PR. Smoke: a coach chat on the owner's OpenAI key works (a signed-in check; see Constraints).
9. **(X) Merge + tag:** with the infra PRs reconciled, merge the xlearn PR (CI green), run the release checklist in the plan's **Release** section (incl. **no live interviews**, next free `v2.0.N`, major = `.release-line`), tag `v2.0.N`, release title `v2.0.N — v2.1 build · MI-16 voice gates`.
10. **(H) Verify on prod** (plan task 7): healthz version, images, ImagePolicies' latest, HelmReleases Ready, smoke login/dashboard/coach; `curl -sI` xLearn + siblings; **the signed-in DevTools `fetch`** to `/xlearn/api/v1/interviews/<nonexistent-id>/segments` in an already-signed-in browser session (you never enter credentials; without one, record "owner login smoke pending: SDP route `fetch`" in status.md and carry on) → coach's 404 (plain-text default today, not the gateway's typed JSON `not_found`), then prove it reached coach from the read-only access log (`ssh sujaykumar-vps 'k3s kubectl logs -n xlearn deploy/xlearn-coach --since=5m' | grep '/segments'` → `POST /interviews/<id>/segments`, status 404); signed out, your own `curl -X POST` → 401; the layered-timeout audit (Traefik args: no `respondingTimeouts` override; gateway 30 s read / 60 s write); coach deploy shows S6 M8's limits (default 500m/256Mi), grace 60, 1/0; `host-verify --cluster` green.
11. **(X) Record** (plan task 8) in a small docs PR if anything is left after the tag: status.md, decisions log, the monthly-window runbook line.

## Constraints

- **Service boundaries ([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)):** the gateway only proxies; coach owns interviews and the BYO key — no credential or provider call in the gateway, no new service.
- **CSP stays `connect-src 'self'`** — never add a provider origin or `wss:`; never the CSP3 `webrtc` directive. `theme.css`/web code is untouched (no UI in this sprint).
- **No behaviour change for learners:** the SPA doesn't call the new route; the route is cohort-gated (T-3); no new flag.
- **Release discipline:** a **patch**, never a minor ([ADR-0034 §1.1](../../adr/0034-v2-release-labelling-gating-and-rollback.md#11-scheme)); infra PRs merge first and are never folded into the tag; **no live interviews** before merging the coach PR and before tagging; parallel-sessions check (peers' tags, PRs, worktrees, ListAgents) before tagging; never move or re-push a tag.
- **GitOps only:** never `kubectl apply`/`edit`/`debug`/`run` by hand; `ssh sujaykumar-vps` reads and `host-verify.sh` only; `kubectl exec` only for the admin CLIs (`coach admin`). Mirror `../infra` conventions; one Middleware per namespace (no cross-namespace refs); no entrypoint-level header middleware.
- **Memory-sum rule ([ADR-0035 §5](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#5-capacity-the-memory-sum-rule-triggers-and-ordered-responses)):** checked before and after the coach resize, surge term included; every container keeps a memory limit.
- **NetworkPolicy standing rule (ADR-0035 §2):** no new in-cluster caller here (gateway → coach already fenced by MI-5a/MI-15); coach's 443 is confirmed, not assumed.
- **D34 — no alerting:** no interview counters, opscheck, Flux Alert or push channel; the release checklist's live-interview check is the only M6 ops step.
- **Privacy:** SDP bodies (ICE candidates = learner IPs) are never logged or persisted.
- **Signed-in checks** (step 10's SDP `fetch`, step 8's coach chat): an agent never enters credentials. Use an already-signed-in browser session if you have one; otherwise run the credential-free checks, record "owner login smoke pending" in status.md, and carry on; don't wait.
- goose/sqlc/outbox: no migration, query or event changes in this sprint (`sqlc diff` must stay clean).

## Deliverables

- `internal/gateway`: `Permissions-Policy` + CSP regression tests; `POST /api/interviews/{id}/segments` (20 s, 20 KiB, cohort, no retry, no body logs) + tests; `docs/architecture/openapi.yaml` + `api.md`.
- `../infra` PR: sibling camera/mic deny (chart 0.3.0 knob values, or a recorded 0.3.x knob exception; sibling values; same-namespace references on the raw routes — Longhorn UI, mi-04's airlift admin routes, `kubescope-route` if present) + infra README note.
- `../infra` PR: coach at S6 M8's limits (default 500m / 256Mi), `GOMEMLIMIT`, grace 60 s, rollingUpdate 1/0.
- `../infra` PR only if needed: coach 443 egress extension.
- Tag `v2.0.N` (MI-16 gateway parts), verified on prod; the WebRTC loopback result and the timeout audit in the PR/status.

## Update status

- This sprint's Status table in [`../sprints/sprint-mi-13.md`](../sprints/sprint-mi-13.md) (🔄 / ✅ / ⛔ per task; _Overall_).
- [`../status.md`](../status.md): Sprint board row; **MI table** row MI-16 ✅ (PR numbers); **milestone → tag → floor → snapshot**: `v2.0.N` (MI-16 gateway parts), floor unchanged, no snapshot (patch, no contract/erase); memory-sum before/after and the margin left for m6b-02; **decisions log** — SDP route path `/segments`, the sibling-middleware mechanism (knob name; a 0.3.x chart bump is recorded as an exception to rollout §2.1 with its reason), the coach limits and `GOMEMLIMIT` from S6 M8, "coach 443 covered by MI-15" (or the extension PR).
- `docs/v2/runbooks/monthly-window.md`: one line — re-read Traefik's args after a k3s upgrade (the SDP route needs ≥ 20 s everywhere).
- No new ADR expected; if one becomes necessary, claim its number only after the parallel-sessions check.

## Done when (acceptance)

- [ ] Every non-xLearn router that serves a document — chart-rendered (airlift, projects-hub, landscape, kubescope) and raw (Longhorn UI, mi-04's `airlift-admin`/`airlift-admin-block`, `kubescope-route` if present, any newer sibling) — answers `Permissions-Policy: camera=(), microphone=()` (redirect-only `ops-redirects` recorded n/a); xLearn documents answer `camera=(self), microphone=(self)` (self only)
- [ ] CSP unchanged — `script-src 'self'`, `connect-src 'self'` (no provider origin, no `wss:`), no `webrtc` directive — and the compose loopback check connects with zero CSP violations
- [ ] `POST /xlearn/api/v1/interviews/{id}/segments` is live on prod: cohort-only, 20 KiB cap (typed 413), **20 s** budget (typed 504), no retry, SDP never logged; a signed-in fetch gets coach's 404 and coach's access log shows the request (or it's recorded as owner login smoke pending); signed out → 401; the prod timeout audit finds nothing on the path below 20 s
- [ ] coach runs S6 M8's limits (default 500m / 256 Mi), grace 60 s, rollingUpdate 1/0; `host-verify --cluster` memory sum inside the rule (numbers recorded)
- [ ] coach egress admits TCP 443 to non-cluster addresses (MI-15 confirmed or extended); a coach chat on the owner's OpenAI key works (or it's recorded as owner login smoke pending)
- [ ] `v2.0.N` tagged with the release checklist and verified; no interviewer behaviour change

## Ship (land-and-sync — owner approval pre-granted)

> Launching this prompt is the owner's approval for every change it makes (D40); don't stop for review.

1. Branch, then conventional commit(s) with the attribution lines, then push, then a PR in every repo touched (`../infra` PRs first where the order requires it; infra PRs are never folded into a tag). Here: step 1's branches, `feat/mi-13-voice-gates` in xlearn and `feat/sibling-camera-mic-deny`, `feat/xlearn-coach-voice-sizing` (and only if needed `feat/xlearn-coach-wss-egress`) in `../infra`.
2. Once CI is green (fix, then merge, on failure), squash-merge. Never enable auto-merge. `../infra` has no CI: put the local checks (chart-diff, `host-verify --cluster`) in each infra PR body and merge on them.
3. **Release action — tag a `v2.0.x` patch (MI-16 gateway parts) + infra PR(s) first:** merge the `../infra` PRs first, each its own PR and never folded into the tag: the sibling deny; the coach sizing, only when `coach admin interviews --live` is empty; the coach egress only if needed. Let Flux reconcile them. Then merge the xlearn PR on CI green. Walk the release checklist (ADR-0034 §6; the plan's **Release** checklist, including "from M6: no live interviews", the next free `v2.0.N`, and major = `.release-line`), push the tag `v2.0.N` (release title `v2.0.N — v2.1 build · MI-16 voice gates`), let Flux deploy, then verify live by looking (step 10). No snapshot: a patch is not a contract, erase or GA tag. Anything recorded after the tag goes in step 11's small docs PR.
4. Update status: the sprint file and `docs/v2/status.md`, in the same PR or a follow-up docs PR merged the same way.
5. Run `git checkout main && git pull` in every repo touched (xlearn and `../infra`). If a clean peer worktree holds `main`, use `git -C <worktree> merge --ff-only origin/main` and then `git switch --detach main`.

# Sprint mi-13 — M6b gates: Permissions-Policy, camera/mic deny, coach sizing, SDP route, WSS egress (MI-16) → v2.0.x patch

> **Milestone:** MI — rollout step **MI-16** (the M6b gates; sprint id `mi-13` ≠ rollout step MI-13, the judge HelmRelease) · **Track:** infra (+ gateway code) · **Order:** 81
> **Prereqs:** [mi-11](sprint-mi-11.md) (MI-15: coach egress policy) · [spk-04](sprint-spk-04.md) (S6 result) · [m6a-06](sprint-m6a-06.md) (M6a shipped dark in a v2.0.x patch)
> **Unblocks:** [m6b-01](sprint-m6b-01.md) (entry gate: "the 20 s SDP route exists on prod")
> **Release action:** **tag a `v2.0.x` patch** (MI-16 gateway parts only; the next free patch, [ADR-0034 §1.1](../../adr/0034-v2-release-labelling-gating-and-rollback.md#11-scheme)) **+ infra PR(s)**, merged first and never folded into the tag
> **Calendar:** Q1 2027, before M6b (after [ds-m6b-01](sprint-ds-m6b-01.md) lands, before [m6b-01](sprint-m6b-01.md)) · no calendar event; no mid-run owner step (before launch, the owner's account has an OpenAI key saved for task 6's smoke; task 7's signed-in `fetch` uses an already-signed-in browser session, or is recorded as "owner login smoke pending" — agents never enter passwords)
> **Execute with:** [`../prompts/prompt-mi-13.md`](../prompts/prompt-mi-13.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | `Permissions-Policy` on the SPA; CSP stays `connect-src 'self'` | X | ⬜ |
| 2 | The 20 s SDP route `POST /api/interviews/{id}/segments` | X | ⬜ |
| 3 | WebRTC under the CSP (S6 note + compose loopback check) | X | ⬜ |
| 4 | Sibling-app camera/mic deny (Traefik `headers` middleware) | I | ⬜ |
| 5 | coach sizing: S6 M8 limits (default 500m / 256 Mi), grace 60 s, rollingUpdate 1/0 (memory-sum checked) | I + H | ⬜ |
| 6 | coach WSS egress (confirm MI-15's 443; extend only if needed) | I | ⬜ |
| 7 | Tag the `v2.0.x` patch + verify on prod | X + H | ⬜ |
| 8 | Record | X | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row, MI table row MI-16, milestone → tag → floor rows).
> Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] **MI-3 merged** ([mi-01](sprint-mi-01.md)): chart 0.3.0's `terminationGracePeriodSeconds` and `strategy.rollingUpdate` knobs exist
- [ ] **MI-15 merged** ([mi-11](sprint-mi-11.md)): the `xlearn` egress policies are live, coach's included (TCP 443 to non-cluster addresses)
- [ ] **MI-5b live** ([mi-04](sprint-mi-04.md)): the admin consoles' hosts are final, so the sibling-router list is too
- [ ] **S6 result known** ([spk-04](sprint-spk-04.md)): chosen shell, M7 deploy shape, M8 coach memory, browser list, and whether the harness page connected under `connect-src 'self'`; [ADR-0032](../../adr/0032-realtime-ai-mock-interviewer.md) Accepted ([ds-m6a-01](sprint-ds-m6a-01.md))
- [ ] **M6a shipped dark** ([m6a-06](sprint-m6a-06.md)): its `v2.0.x` patch is live, so the gateway's `/api/interviews/*` proxy, its cohort gate and `coach admin interviews --live` exist
- [ ] **Post-GA release line** ([ga-02](sprint-ga-02.md)): `.release-line` = `2`; every `xlearn-*` range `>=1.0.0 <3.0.0`
- [ ] **Weekly image fresh** (owner, before launch): the last Hostinger weekly image in hPanel is ≤ 7 days old, since the coach-sizing PR restarts coach ([rollout §2.2](../rollout-plan.md)); if not, that PR is ⛔ and the rest lands
- [ ] **Memory-sum margin read:** `host-verify --cluster` shows ≥ the coach resize's cost + 0.05 GiB inside the rule (cost = 2 × (S6 M8 memory limit − 128 Mi), limits + surge term; **≥ 0.3 GiB** at the default 256 Mi)
- [ ] **Parallel sessions:** no open peer PR on the gateway's security headers or interview routes, `../infra/apps/xlearn-coach.yaml`, the sibling HelmReleases or `../infra/charts/project` (`gh pr list` in both repos, `git worktree list`, ListAgents)

## Goal

Land the small platform changes voice needs before any voice code: the SPA's **`Permissions-Policy:
camera=(self), microphone=(self)`** with the CSP kept at `connect-src 'self'`; a **camera/mic deny** on every
sibling app that shares the `projects.sujaykumar.dev` origin (so a mic grant to the origin can't be used by
another app); **coach sized for voice** (S6 M8's limits, default 500m / 256 Mi; 60 s grace; make-before-break rolling update) inside
the memory-sum rule; coach's **WSS egress** confirmed; and the gateway's **20 s SDP route** live on prod — then
cut a **`v2.0.x` patch** carrying only the gateway parts, with **no interviewer behaviour change**.
[m6b-01](sprint-m6b-01.md) builds its SDP broker on this live route.

## Scope

**In**
- `internal/gateway`: `Permissions-Policy` on the SPA; CSP regression tests (X).
- `internal/gateway`: `POST /api/interviews/{id}/segments` — "the SDP route" ([t6 §3](../research/t6-realtime-interviewer.md#3-architecture--media-path): one POST = one new provider segment) — with its own 20 s budget, 20 KiB body cap, cohort gate, no retry, no body logging; `docs/architecture/openapi.yaml` + `api.md` (X).
- A WebRTC-under-CSP check (compose + the S6 note) (X).
- `../infra`: a Traefik `headers` middleware `Permissions-Policy: camera=(), microphone=()` on every non-xLearn router — chart-rendered and raw ([mi-04](sprint-mi-04.md)'s `ops` routes included) (I).
- `../infra/apps/xlearn-coach.yaml`: limits = S6 M8's values (default 500m / 256 Mi), `GOMEMLIMIT`, `terminationGracePeriodSeconds: 60`, `strategy.rollingUpdate` maxSurge 1 / maxUnavailable 0 (I), checked against the memory-sum rule (H).
- coach egress for the provider sideband WebSocket (TCP 443) — confirmed or extended (I).
- Tag the `v2.0.x` patch; verify on prod.

**Out**
- The `VoiceShell` adapter, coach's SDP broker handler, the 15 s orphan reaper, the sideband → [m6b-01](sprint-m6b-01.md).
- Lease, make-before-break re-attach, SIGTERM drain (the code that uses the 60 s grace), cost and $ cap, push-to-talk → [m6b-02](sprint-m6b-02.md).
- A `coach-interview` Deployment (only if S6 M7 failed) and its policies → [m6b-02](sprint-m6b-02.md).
- Voice UI (pre-flight, notices, live HUD, self-view) → [m6b-03](sprint-m6b-03.md); fake-media e2e and the interviewer GA flip → [m6b-04](sprint-m6b-04.md).
- UDP, TURN, Traefik WebSockets, ufw, namespaces, an object store — **none needed** ([ADR-0032 §2](../../adr/0032-realtime-ai-mock-interviewer.md#2-architecture)).
- Interview counters, alerts or opscheck — dropped by D34; the release checklist's live-interview check stays.

## Tasks

### 1 · `Permissions-Policy` on the SPA; CSP stays `'self'` [X]

In the gateway's security-header middleware that [m1-04](sprint-m1-04.md) added with the CSP —
`internal/gateway/security.go`, applied in `Handler()` (`internal/gateway/gateway.go`); confirm with
`grep -rn Content-Security-Policy internal/gateway`:
- Add **`Permissions-Policy: camera=(self), microphone=(self)`** to every response the gateway serves (the SPA shell
  — `serveIndex`, reached through `serveStatic`'s SPA fallback in `gateway.go` — is what matters; API responses carry it too).
- **CSP unchanged:** `script-src 'self'`, `connect-src 'self'`. The page never calls `api.openai.com` (coach brokers the
  SDP) and `connect-src` doesn't govern WebRTC ([t6 §3](../research/t6-realtime-interviewer.md#3-architecture--media-path)
  P1 gate, [ADR-0033 §12](../../adr/0033-invite-only-admission-and-owner-admin.md#12-authz-deltas-across-v2) row 15).
  Do **not** add the CSP3 `webrtc` directive (its default is allow; `'block'` would kill voice).
- Tests (next to m1-04's header tests):
  - **Permissions-Policy** on all three paths — `/`, a deep SPA route (e.g. `/dsa/week/2`) and `/api/v1/healthz` — equals
    `camera=(self), microphone=(self)` exactly.
  - **CSP** only on the SPA shell responses (`/` and the deep route; m1-04 sets it on the shell, not on API responses):
    parse it into directives; assert `connect-src` is exactly `'self'` (no `openai` host, no `wss:`), `script-src` is
    exactly `'self'`, no `webrtc` directive, and no directive carries a **bare scheme source** — a token that is only
    `scheme:` (`https:`, `wss:`, `http:`, `ws:`; `data:` allowed only in `img-src`). Host sources with a scheme
    (m1-04's `https://fonts.googleapis.com`, `https://fonts.gstatic.com`, `https://github.com` in `form-action`) are
    legitimate — test tokens, not substrings.

### 2 · The 20 s SDP route [X]

`POST /api/interviews/{id}/segments` (served as `/xlearn/api/v1/interviews/{id}/segments` through the `/api/v1`
alias), added to the interview proxy [m6a](sprint-m6a-01.md) built in `internal/gateway`, with an `apiRoutes()` entry
(`internal/gateway/bff.go`, `Doc: true`) and the matching `docs/architecture/openapi.yaml` path (the drift test) and `api.md` row.

| Aspect | Rule |
|---|---|
| Auth | session → an `aud=coach` JWT (the existing mint); m1-04's `Content-Type: application/json` + `Sec-Fetch-Site` checks on mutating calls |
| Gate | the same T-3 cohort gate m6a put on `/api/interviews/*` ([ADR-0034 §2](../../adr/0034-v2-release-labelling-gating-and-rollback.md#2-feature-gating-three-tiers-no-flag-service)); non-cohort → the same 404 as an unknown route, coach not called |
| Body | JSON (`{"sdp": "<offer>", …}` — [m6b-01](sprint-m6b-01.md) owns the fields; the gateway doesn't parse them); `http.MaxBytesReader` **20 KiB** (a ≤ 16 KiB SDP offer + envelope) → typed 413 `payload_too_large` (the L6 shape); coach validates the SDP itself (m6b-01) |
| Budget | its **own** `http.Client{Timeout: 20 * time.Second}` plus `context.WithTimeout(r.Context(), 20*time.Second)` — not the 10 s `httpc` (`gateway/coach.go`), not the no-timeout stream client; expiry → typed 504 `sdp_timeout` (the SPA falls back to text mode) |
| Upstream | coach `POST /interviews/{id}/segments`, mapped like m6a's other interview calls; coach's handler lands in m6b-01 — until then coach's mux answers its default 404 (`text/plain` `404 page not found`, unlike the gateway's typed JSON 404), which passes through |
| Retries | **none** — a retry opens a second provider session on the learner's key |
| Response | coach's status, body and `Content-Type` passed through; `Cache-Control: no-store` |
| Logging | never the request or response body — SDP ICE candidates carry the learner's IP addresses ([t6 §3](../research/t6-realtime-interviewer.md#3-architecture--media-path)); the access log keeps method, path, status, duration |
| Limits | L5 per-account bucket applies; L19 (≤ 2 starts/day, ≤ 3 live voice, 75 voice-min/day) is enforced in coach ([m6a-01](sprint-m6a-01.md), [m6b-02](sprint-m6b-02.md)) |

Server-side, `httpx.NewServer` already allows it (`ReadTimeout` 30 s, `WriteTimeout` 60 s). Tests against an
`httptest` fake coach, with the budget injectable (e.g. 200 ms) plus one test pinning the production value at 20 s:
answer just under the budget → 200 passthrough; just over → 504 `sdp_timeout`; body 20 KiB + 1 → 413 and coach never
called; upstream 502 → exactly one upstream request; non-cohort → 404, cohort → bearer with `aud=coach`; a captured
slog holds no `v=0`, `a=candidate` or `a=fingerprint`; the existing coach routes still use the 10 s client.
The SPA doesn't call the route yet ([m6b-03](sprint-m6b-03.md) does): **no interviewer behaviour change**.

### 3 · WebRTC under the CSP [X]

- **S6 note** ([spk-04](sprint-spk-04.md)): confirm the harness page connected to the provider with the page limited to
  `connect-src 'self'` (SDP via the local broker). If the note records any page-side request to a provider origin,
  **stop** — that contradicts [ADR-0032 §2](../../adr/0032-realtime-ai-mock-interviewer.md#2-architecture) and is
  resolved before anything is widened.
- **Compose loopback check** (proves the shipped headers don't block mic capture or a PeerConnection): Chrome with
  `--use-fake-ui-for-media-stream --use-fake-device-for-media-stream`, the owner (dev login) on the compose SPA
  (`http://localhost:…/xlearn/` is a secure context), DevTools console:
  ```js
  const v = []; document.addEventListener('securitypolicyviolation', e => v.push(e.violatedDirective));
  const s = await navigator.mediaDevices.getUserMedia({audio: true});
  const a = new RTCPeerConnection(), b = new RTCPeerConnection();
  a.onicecandidate = e => e.candidate && b.addIceCandidate(e.candidate);
  b.onicecandidate = e => e.candidate && a.addIceCandidate(e.candidate);
  s.getTracks().forEach(t => a.addTrack(t, s)); a.createDataChannel('oai-events');
  await a.setLocalDescription(await a.createOffer()); await b.setRemoteDescription(a.localDescription);
  await b.setLocalDescription(await b.createAnswer()); await a.setRemoteDescription(b.localDescription);
  await new Promise(r => setTimeout(r, 3000));
  ({state: a.connectionState, violations: v, mic: document.featurePolicy?.allowsFeature('microphone')})
  ```
  Expect `{state: 'connected', violations: [], mic: true}`; paste the result into the PR. The real provider leg is
  [m6b-01](sprint-m6b-01.md) (dev) and [m6b-04](sprint-m6b-04.md) (fake-media on prod).

### 4 · Sibling-app camera/mic deny [I]

Every app on the shared origin can use a mic permission the browser granted to `projects.sujaykumar.dev`; a
`Permissions-Policy: camera=(), microphone=()` on their documents stops that ([t6 §3](../research/t6-realtime-interviewer.md#3-architecture--media-path) P1 gate).
- **Routers.** The 2026-09-24 list (`airlift/airlift`, `projects-hub/projects-hub` — the `/` catch-all —
  `landscape/landscape`, `kubescope/kubescope`, `longhorn-system/longhorn-ui`, `xlearn/xlearn-gateway` excluded) predates
  [mi-04](sprint-mi-04.md) (MI-5b), which adds **raw** IngressRoutes: `apps/airlift-admin-routes.yaml` (`airlift-admin`,
  serving airlift's admin HTML on `ops.sujaykumar.dev`, and `airlift-admin-block`, a 403 on `projects`),
  `apps/ops-redirects.yaml` (302s only; the optional `ops` root redirect in `landscape`) and — only if mi-04 task 5 fired —
  `apps/kubescope-route.yaml` with kubescope's chart `route.enabled: false`. Re-read `k3s kubectl get ingressroute -A` live;
  **every non-xLearn router that can serve a document** gets the deny — including those on `ops.sujaykumar.dev` (another
  origin, so defence in depth only). Redirect-only routes serve no document: record them as n/a.
- **Before:** `curl -sI` each; if an app already sends a `Permissions-Policy`, merge the directives (Traefik's
  `customResponseHeaders` replaces the backend's value).
- **Mechanism, in this order:**
  - (a) **Chart-rendered routes, via a chart 0.3.0 knob.** If [mi-01](sprint-mi-01.md)'s chart 0.3.0 carries a default-off
    `route.responseHeaders` (the recommended home: it renders a `<release>-headers` Middleware with
    `headers.customResponseHeaders` in the release namespace and appends it to the route's chain), set it in the values
    of each chart-rendered sibling route — airlift, projects-hub, landscape, and kubescope unless its chart route is
    disabled. `hack/chart-diff.sh` must show only those releases' IngressRoute and new Middleware change; no chart bump,
    nothing restarts.
  - (b) **Otherwise, a chart 0.3.x minor adding that knob — a deliberate exception** to [rollout §2.1](../rollout-plan.md#21-chart-030-knob-list)
    ("one PR carries the union of every topic's asks"; T6 §3 listed the sibling deny but the 0.3.0 union missed it).
    Record the exception and its reason in the status.md decisions log. `hack/chart-diff.sh` then shows the
    `helm.sh/chart` label change on every release (pod templates carry selector labels only, so nothing restarts) plus
    the sibling routes and Middlewares — nothing else.
  - (c) **Raw routes** (no chart knob reaches them), each with a **same-namespace** `headers` reference:
    - Longhorn UI (`infrastructure/storage/ui.yaml`): a raw `headers` Middleware in `longhorn-system` plus its reference;
    - `airlift-admin` and `airlift-admin-block` (`apps/airlift-admin-routes.yaml`, namespace `airlift`): reference the
      chart-rendered `airlift-headers` Middleware from (a)/(b) — same namespace, no new object;
    - `apps/kubescope-route.yaml` (only if present): a raw `headers` Middleware in `kubescope` plus its reference (the
      chart renders no Middleware while `route.enabled: false`);
    - `apps/ops-redirects.yaml` (302 only, never a document): n/a — record it.
- **Not** an entrypoint middleware on `websecure`: it wraps xLearn's route too, and the outer modifier runs last, so it would
  overwrite the gateway's `camera=(self), microphone=(self)`. **Not** cross-namespace references: this Traefik's
  kubernetescrd provider doesn't allow them (see the comment in `infrastructure/storage/ui.yaml`).
- Add to the infra README's add-an-app notes: *every new app on `projects.sujaykumar.dev` carries the camera/mic deny.*
- Verify after merge: each sibling URL — including `https://ops.sujaykumar.dev/airlift/admin` and the kubescope raw route
  if present — answers `permissions-policy: camera=(), microphone=()`; on a sibling page in Chrome,
  `document.featurePolicy.allowsFeature('microphone')` is `false`; xLearn unchanged.

### 5 · coach sizing [I + H]

`../infra/apps/xlearn-coach.yaml` ([t6 §3](../research/t6-realtime-interviewer.md#3-architecture--media-path) infra table,
[ADR-0035 §5](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#5-capacity-the-memory-sum-rule-triggers-and-ordered-responses) coach P1 row), using chart 0.3.0's knob names.
The values come from **S6 M8** (sideband KB/s, harness RSS, decoder CPU), recorded as `## 16. S6 results` in
[t6](../research/t6-realtime-interviewer.md) by [spk-04](sprint-spk-04.md) — "coach 500m / 256 Mi confirmed or revised":
- `resources.limits` = S6 M8's values (**default cpu 500m, memory 256Mi**; today 250m / 128Mi; requests unchanged);
- `env` += `GOMEMLIMIT` = S6 M8's measured value (default `200MiB`, ≈ 80% of the memory limit);
- `terminationGracePeriodSeconds: 60` (harmless until m6b-02's drain; coach's own `shutdownTimeout` is 10 s today);
- `strategy: {type: RollingUpdate, rollingUpdate: {maxSurge: 1, maxUnavailable: 0}}` (the effective default for one
  replica, pinned so make-before-break never depends on it).

**Memory sum:** with M = the M8 memory limit, the resize adds (M − 128 Mi) of limits **and** the same to the
rollout-surge term (ADR-0035 §5 counts a fleet image-automation roll, so coach's surge pod grows to M) ⇒
**Δ = 2 × (M − 128 Mi)** — ≈ **+0.25 GiB** at the default 256 Mi — and (M8 CPU − 250m) of CPU limits (+250m at the
default). If M8 revised the values, note the new coach P1 row in the decisions log. Run `host-verify --cluster` before
(record the margin) and after (must stay inside **Σ limits + Σ p95 limitless + largest surge + host ≤ capacity − 0.5 GiB**).
If the margin before is < Δ + 0.05 GiB (0.3 GiB at the default): stop, ⛔ "R0 trim first" (ADR-0035 §5 standing rule). Record the margin left — [m6b-02](sprint-m6b-02.md)
needs it if S6 M7 failed and a `coach-interview` Deployment follows. The PR rolls coach: merge it only when
`coach admin interviews --live` is empty.

### 6 · coach WSS egress [I]

The provider sideband is an outbound `wss://` connection on **TCP 443** ([t6 §3](../research/t6-realtime-interviewer.md#3-architecture--media-path)).
[mi-11](sprint-mi-11.md)'s MI-15 policy gives coach TCP 443 to non-cluster addresses (for the BYO providers).
Read the rendered policy (`k3s kubectl get networkpolicy -n xlearn -o yaml`): if coach already has 443 to
`0.0.0.0/0` minus the cluster ranges, record "covered by MI-15 (infra#N)" and make **no change**; if it is narrower,
extend it to that block in its own PR. Smoke: a coach chat on the owner's OpenAI key (saved before launch) succeeds (coach →
`api.openai.com:443` under the policy), in an already-signed-in browser session; without one, record "owner login smoke
pending" in status.md and carry on. A `coach-interview` Deployment's egress, if ever needed, is [m6b-02](sprint-m6b-02.md)'s.

### 7 · Tag the `v2.0.x` patch + verify [X + H]

After the infra PRs (tasks 4–6) are merged and reconciled and the xlearn PR (tasks 1–3, 8's docs) is merged with CI
green: run the release checklist below, tag the **next free `v2.0.N`** (`git ls-remote --tags origin 'refs/tags/v2.0.*'`),
GitHub release title `v2.0.N — v2.1 build · MI-16 voice gates`. Then the MI-16 checks on prod:
- `curl -sI https://projects.sujaykumar.dev/xlearn/` → `permissions-policy: camera=(self), microphone=(self)` and the CSP unchanged; siblings → the deny (task 4).
- **SDP route live** (a signed-in check — agents never enter credentials): in an already-signed-in browser session, run in DevTools
  `fetch('/xlearn/api/v1/interviews/<nonexistent-id>/segments', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: '{"sdp":"v=0"}'}).then(async r => [r.status, r.headers.get('content-type'), await r.text()])`
  and record the result. Without a signed-in session, record "owner login smoke pending: SDP route `fetch`" in status.md
  and carry on with the credential-free checks below. Expect **404 from coach** — today coach's default mux answers `text/plain` `404 page not found`
  (its handler lands in m6b-01) — **not** the gateway's typed JSON `not_found` (which would mean the cohort gate or the
  route missed). The proof is the read-only coach access log, which tells the two apart even if m6a gave coach a typed
  404: `ssh vps 'k3s kubectl logs -n xlearn deploy/xlearn-coach --since=5m' | grep '/segments'` shows
  `POST /interviews/<id>/segments` with `status 404`. The `aud=coach` bearer is pinned by task 2's test (an unregistered coach route checks no JWT, so a
  live 404 can't prove it). The agent checks signed out: `curl -s -o /dev/null -w '%{http_code}' -X POST -H 'Content-Type: application/json' -d '{}' https://projects.sujaykumar.dev/xlearn/api/v1/interviews/x/segments` → `401`.
- **Layered-timeout audit:** Traefik (3.7.8 on 2026-09-24) has no `respondingTimeouts` override — read the live args
  (`k3s kubectl -n kube-system get deploy traefik -o jsonpath='{.spec.template.spec.containers[0].args}'`; defaults: read 60 s,
  write 0, idle 180 s); the gateway server allows 30 s read / 60 s write ⇒ nothing on the path is below 20 s; the 20 s
  budget itself is pinned by task 2's test. Record the audit.
- coach: `k3s kubectl get deploy -n xlearn xlearn-coach -o yaml` shows S6 M8's limits (default 500m / 256Mi) and `GOMEMLIMIT`, grace 60, rollingUpdate 1/0; `host-verify --cluster` green.

### 8 · Record [X]

`docs/v2/status.md`: MI table MI-16 ✅ (PR numbers); milestone → tag → floor → snapshot (`v2.0.N`, floor unchanged,
no snapshot needed — a patch without contract or erase); memory-sum before/after and the margin left for
m6b-02; decisions log — the route path `/segments` ("the SDP route"), the sibling-middleware mechanism, the
`GOMEMLIMIT` value; rollout §3 M6b row's MI-16 gate met. `docs/architecture/api.md` + `openapi.yaml` (task 2).
Add one line to `docs/v2/runbooks/monthly-window.md` (from [mi-11](sprint-mi-11.md)): after a k3s upgrade, re-read
Traefik's args — the SDP route needs every layer at ≥ 20 s.

## Acceptance criteria

- [ ] Every non-xLearn router that serves a document — chart-rendered (airlift, projects-hub, landscape, kubescope) and raw (Longhorn UI, mi-04's `airlift-admin`/`airlift-admin-block`, `kubescope-route` if present, any newer sibling) — answers `Permissions-Policy: camera=(), microphone=()` (redirect-only `ops-redirects` recorded n/a); the mechanism (0.3.0 knob, or the recorded 0.3.x exception) is in the decisions log; xLearn documents answer `camera=(self), microphone=(self)` (self only)
- [ ] CSP unchanged — `script-src 'self'`, `connect-src 'self'` (no provider origin, no `wss:`), no `webrtc` directive — and the compose loopback check connects with zero CSP violations
- [ ] `POST /xlearn/api/v1/interviews/{id}/segments` is live on prod: cohort-only, 20 KiB cap (typed 413), **20 s** budget (typed 504), no retry, SDP never logged; a signed-in fetch gets coach's 404 (plain text) and coach's access log shows the request (or it's recorded as owner login smoke pending); signed out → 401; the prod timeout audit finds nothing on the path below 20 s
- [ ] coach runs S6 M8's limits (default 500m / 256 Mi), grace 60 s, rollingUpdate 1/0; `host-verify --cluster` memory sum inside the rule (numbers recorded)
- [ ] coach egress admits TCP 443 to non-cluster addresses (MI-15 confirmed or extended); a coach chat on the owner's OpenAI key works (or it's recorded as owner login smoke pending)
- [ ] `v2.0.N` tagged with the release checklist and verified; no interviewer behaviour change

## Release

**Tag a `v2.0.x` patch (MI-16 gateway parts) + infra PR(s).** Order: the infra PRs (sibling deny; coach sizing;
coach egress only if needed) merge first, each its own PR ([rollout §2.2](../rollout-plan.md#22-operating-rules-every-mi-step-and-every-tag));
then the xlearn PR; then the tag. It is a **patch**: from `v2.0.0` on the minor moves only at a GA flip
([ADR-0034 §1.1](../../adr/0034-v2-release-labelling-gating-and-rollback.md#11-scheme)); `v2.1.0` is [m6b-04](sprint-m6b-04.md)'s.
Release checklist ([ADR-0034 §6](../../adr/0034-v2-release-labelling-gating-and-rollback.md#6-release-checklist), verbatim, plus the ADR-0035 §2 standing rule):

- [ ] Before the tag: peers' tags and PRs are checked (parallel sessions; `git ls-remote --tags origin`, `gh pr list`, `git worktree list`, ListAgents)
- [ ] Before the tag: it is the next free version, and its major equals `.release-line`
- [ ] Before the tag: ACL PRs for new streams and consumers are merged
- [ ] Before the tag: a new service's image comes before its policy
- [ ] Before the tag: for a contract: rehearsed in compose, floor marked
- [ ] Before the tag: for a contract, erase or GA tag: `host-verify --cluster` is green (ADR-0035), the host has settled, and the snapshot is taken
- [ ] Before the tag: from M6: no live interviews
- [ ] After the tag (by looking, D34): `/xlearn/api/v1/healthz` reports the version
- [ ] After the tag: `k3s kubectl get deploy -n xlearn` shows the new images
- [ ] After the tag: every `xlearn-*` ImagePolicy's latest equals the tag, and the HelmReleases are Ready
- [ ] After the tag: smoke-test login, the dashboard and coach
- [ ] Record milestone → tag → floor → snapshot and any flag changes in `docs/v2/status.md`
- [ ] (ADR-0035 §2 standing rule, not part of ADR-0034 §6) Every new in-cluster HTTP or NATS caller this tag introduces has its NetworkPolicy (ingress and egress) change in its own infra PR, merged before the tag

**For this tag:** ACL PRs — n/a (no stream or consumer); new service — n/a; contract / erase / GA — n/a, so no
snapshot; **M6: no live interviews** — `ssh vps 'sudo k3s kubectl exec -n xlearn deploy/xlearn-coach -- coach admin interviews --live'`
is empty (also before merging task 5's PR); major = `.release-line` = `2`; standing rule — no new caller (gateway → coach
exists under MI-5a/MI-15; coach → provider 443 is task 6); flags — none new (the route sits behind the existing cohort gate).
Extra after-tag reads: task 7's MI-16 checks. Rollback: R-c revert + the next patch (default) or R-b narrowing to the
previous `v2.0.x` (no migrations, floor unchanged); infra: `git revert`.

## Definition of Done

CI green (incl. the OpenAPI drift test) · infra PRs merged via GitOps (no hand `kubectl`) and reconciled · `v2.0.N`
tagged, deployed by Flux and verified live · acceptance criteria met · memory sum inside the rule · statuses updated
(this file + [`../status.md`](../status.md)) · local `main` synced in xlearn and `../infra`.

## Risks / watch-outs

- **Widening `connect-src` only adds exposure** — WebRTC isn't governed by it; keep `'self'`. A provider origin in the CSP
  is a sign the design drifted (the SDP must stay brokered by coach).
- **Traefik `customResponseHeaders` replaces** a sibling's own `Permissions-Policy` — merge directives; an entrypoint-level
  middleware would silently override xLearn's `(self)` grant and break voice.
- **Cross-namespace middleware references fail** here — one Middleware per release namespace.
- **Raw routes escape a chart knob** — mi-04's `airlift-admin`, `kubescope-route` and the Longhorn UI each need their own
  same-namespace reference; re-read `ingressroute -A` so a later raw route isn't missed.
- **Reopening the chart late** re-renders every release — prefer the 0.3.0 knob; a 0.3.x bump is a recorded exception.
- **The coach resize grows the surge term too** (+128 Mi) — read the memory sum before and after; the margin feeds m6b-02.
- **Merging the coach PR or tagging mid-interview** rolls coach — check `coach admin interviews --live` both times.
- **Retrying the SDP POST double-bills the learner** — the gateway never retries; the SPA owns the fallback.
- **Logging SDP leaks IP addresses** — body logging is tested away.
- **A k3s/Traefik upgrade could change the default timeouts** — task 8 adds the re-read to the monthly-window runbook ([mi-11](sprint-mi-11.md)).
- **The patch must not move the minor** (ADR-0034 §1.1).

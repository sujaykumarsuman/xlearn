# Sprint mi-04 — Admin consoles to ops.sujaykumar.dev (MI-5b)

> **Milestone:** MI — infra-first track, rollout step **MI-5b** · **Track:** infra (parallel) · **Kind:** infra
> **Prereqs:** none. MI-5b depends on no other MI step ([rollout §2](../rollout-plan.md#2-mi-infra-track)); the owner adds the DNS record before launch (event `ev-mi5b-dns`)
> **Unblocks:** [l-02](sprint-l-02.md) (hard gate: MI-5b live before the first `tester` is minted, `ev-first-tester`, prepared by [m1-04](sprint-m1-04.md)) · [l-04](sprint-l-04.md) (entry gate) · [mi-13](sprint-mi-13.md) (entry gate: the console hosts are final) · [m1-06](sprint-m1-06.md) / [m1-07](sprint-m1-07.md) (soft gate: before the M1 Markdown renderer ships in v1.7.0) · the v3 opening gate ([rollout §11](../rollout-plan.md#11-opening-gates-v3))
> **Release action:** infra PR(s) only: two PRs, plus a third only if the acceptance fails. No xlearn tag. The status update is an xlearn docs PR
> **Calendar:** weeks 2–4, target week 2 (Mon 2026-10-05 → Fri 2026-10-09). Avoid spike week (Mon 2026-10-12 → Fri 2026-10-16), when the owner is booked; the fallback is week 4 (by Fri 2026-10-23). MI-5b has float only until the first `tester` is minted (L-E, around the M2/M3 boundary; [rollout §6](../rollout-plan.md#6-critical-path-parallel-tracks-owner-calendar))
> **Execute with:** [`../prompts/prompt-mi-04.md`](../prompts/prompt-mi-04.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | DNS record for `ops.sujaykumar.dev` (`ev-mi5b-dns`) — before launch | O | ⬜ |
| 2 | Certificate `ops-tls` + TLSStore entry (infra PR 1) | I | ⬜ |
| 3 | Move the IngressRoutes, block airlift admin on the shared origin, add old-URL redirects, rewrite the README (infra PR 2) | I | ⬜ |
| 4 | Cross-origin acceptance: credential-free probes (4a), then the signed-in browser probe (4b; ⛔ for a follow-up if no signed-in session) | H | ⬜ |
| 5 | Interim guard: `ipAllowList` (infra PR 3, **only if task 4 fails**) | I | ⬜ |
| 6 | Record MI-5b in `docs/v2/status.md` | X | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, and ⛔ if it's blocked (say why).
> Task 5 becomes ✅ "n/a (acceptance passed)" if task 4 passes. Update the _Overall_ line to match, and mirror the sprint's state into
> [`../status.md`](../status.md) (the Sprint board row and the MI rows). Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] The DNS record exists: the owner adds it before launch (task 1, `ev-mi5b-dns`), and the session verifies it resolves before anything else (task 1).
- [ ] Local `../infra` `main` is synced. No open peer PR touches `apps/kubescope.yaml`, `apps/landscape.yaml`, `apps/airlift.yaml`, `infrastructure/storage/ui.yaml` or `infrastructure/configs/`. If one does, agree the merge order first (parallel sessions).
- [ ] No non-owner account exists on production yet: `identity admin account list` shows only the owner, or the M1b CLI isn't live yet. The session runs the CLI itself through the sanctioned admin-CLI `kubectl exec` path ([rollout §2.2](../rollout-plan.md#22-operating-rules-every-mi-step-and-every-tag)); launching the prompt pre-approves that one use (D40), and it's logged in `docs/v2/status.md`'s CLI-use log, as §2.2 requires. If a `tester` already exists, this sprint is overdue. Treat it as urgent and record the gap in the Decisions log.

## Goal

Get every admin console off xLearn's origin **before the first non-owner account exists**
([ADR-0033 §11](../../adr/0033-invite-only-admission-and-owner-admin.md#11-admin-console-isolation-mi-5b)). The consoles are kubescope (cluster-admin, `exec`), landscape, the read-write
Longhorn UI and airlift's admin surface. Today they all share one browser origin with xLearn, `projects.sujaykumar.dev`, and are separated only by path. So one XSS in learner-, tester- or
AI-rendered xLearn content, while the owner is signed in, can drive kubescope to cluster-admin. From there it can read `sops-age`, which decrypts every secret, and there are no backups (D12). After this sprint:
- `projects.sujaykumar.dev` serves no console;
- the consoles live on **`ops.sujaykumar.dev`**;
- the consoles are proven to refuse same-site cross-origin writes and WebSocket upgrades. If they don't, an IP allowlist fences them.

## Scope

**In**
- An owner DNS record, plus a separate Let's Encrypt certificate for `ops.sujaykumar.dev` adopted by Traefik's default TLSStore.
- Move three routes to `Host(`ops.sujaykumar.dev`)`: kubescope, landscape (with `LANDSCAPE_PUBLIC_URL`), and the Longhorn UI with its landscape ForwardAuth. Base paths stay the same.
- **airlift's `/api/admin` is exposed** (verified at planning). Its bearer token is the shared `projects-admin` password, the same one that opens kubescope. Its admin page keeps that token in `sessionStorage` on the shared origin. This sprint serves `/airlift/admin` and `/airlift/api/admin/*` from `ops` only and answers 403 for them on `projects`.
- Old-URL redirects: exact console roots on `projects` return a 302 to `ops`, so bookmarks and the hub's `/landscape/` link keep working.
- The cross-origin acceptance (`Origin` / `Sec-Fetch-Site` on mutating calls and WebSocket upgrades), and the interim `ipAllowList` if it fails.
- The infra README's "Admin consoles" section, rewritten.

**Out**
- The xLearn CSP and the gateway's `Sec-Fetch-Site` + JSON checks: [m1-04](sprint-m1-04.md) (M1b, ADR-0033 §12 row 15).
- Permissions-Policy and the camera/mic deny middleware on sibling apps: [mi-13](sprint-mi-13.md) (MI-16).
- Any web admin for xLearn. There is none ([ADR-0033 §10](../../adr/0033-invite-only-admission-and-owner-admin.md#10-no-web-admin-no-waitlist), D33).
- Minting the first `tester`: event `ev-first-tester`, after this sprint ([m1-04](sprint-m1-04.md) prepares the CLI).
- Decoupling airlift's admin token from `projects-admin`, rotating `projects-admin`, `__Host-` cookie prefixes in kubescope/landscape, and the hub's link text in `sujaykumarsuman.github.io/projects`. These belong to other repos or are owner options (see Risks). None of them is planned in v2.
  - The register's task 3 included "update projects-hub links". It's descoped here because the hub is another repo, and the old-URL redirects keep the hub's `/landscape/` link working. Editing the hub's links is an optional owner follow-up in that repo. Task 6 records the descope in the Decisions log.
- Fixing landscape's per-app public URLs after the move (see Risks). That's an upstream issue in the landscape repo, linked from the Decisions log.

## Tasks

### 1 · DNS record [O, before launch]

Owner action before launch, recorded as `ev-mi5b-dns` (with the bookmark update, see Risks). In the Hostinger DNS zone for `sujaykumar.dev` (nameservers `ns1/ns2.dns-parking.com`), add **`A ops`** with the same IPv4 value as `projects.sujaykumar.dev`'s A record and a TTL of 300. Add no AAAA record: `projects` has none, so keep parity. No CAA record exists, and none is needed.

The agent verifies propagation before task 2 merges, because Let's Encrypt resolves through public resolvers:
- `dig +short ops.sujaykumar.dev A` equals `dig +short projects.sujaykumar.dev A`, checked against the local resolver, `@1.1.1.1` and `@8.8.8.8`;
- the node sees the same (`ssh vps 'getent hosts ops.sujaykumar.dev'`).

If the record is missing (not just still propagating), don't wait: everything else depends on it (the HTTP-01
challenge needs it, and PR 2 needs `ops-tls`). Set task 1 ⛔ in this file and status.md, naming the owner action,
and land that as the docs PR; a re-run starts at task 1.

### 2 · Certificate [I] — infra PR 1 `feat(tls): certificate for ops.sujaykumar.dev (MI-5b)`

- `infrastructure/configs/certificate.yaml`: add a **second** `Certificate` named `ops-tls` in `kube-system`, with `secretName: ops-tls`, `dnsNames: [ops.sujaykumar.dev]` and `issuerRef` set to ClusterIssuer `letsencrypt-prod` (HTTP-01 through Traefik, like `projects-tls`).
- `infrastructure/configs/tlsstore.yaml`: add `certificates: [{secretName: ops-tls}]` to the `default` TLSStore. `defaultCertificate` stays `projects-tls`. Every `tls: {}` route then gets its certificate by SNI, so no route or chart change is needed.
- **Why not add a SAN to `projects-tls`:** a failed `ops` challenge would then block the renewal of the certificate xLearn is served with. Separate certificates renew independently.
- **Before merging.** infra has no CI, so paste the `yq` parse of both files and the `hack/host-lint.sh` output (a sanity run) into the PR body.
- Verify, read-only:
  - `ssh vps 'k3s kubectl -n kube-system get certificate ops-tls'` shows `Ready=True`;
  - `openssl s_client -connect ops.sujaykumar.dev:443 -servername ops.sujaykumar.dev </dev/null | openssl x509 -noout -subject -ext subjectAltName -issuer -dates` shows `ops.sujaykumar.dev` and a Let's Encrypt issuer;
  - `projects.sujaykumar.dev` still serves `projects-tls`;
  - `infra-configs` is Ready.
- Until PR 2 merges, `ops` answers Traefik's 404. That's expected.

### 3 · Move the IngressRoutes [I] — infra PR 2 `feat(ops): move admin consoles to ops.sujaykumar.dev (MI-5b)`

Merge only after `ops-tls` is Ready. **landscape and the Longhorn UI must move in the same PR.** Longhorn's ForwardAuth relies on landscape's `ls_session` cookie (`Path=/`, host-only), which exists only on the host landscape is served from.

| Console | File | Change |
|---|---|---|
| kubescope | `apps/kubescope.yaml` | `route.host: ops.sujaykumar.dev`. Chart 0.2.2 already has `route.host`, so no chart change is needed. Also update the header comment URL. `KUBESCOPE_BASE_PATH=/kubescope`, the host-only session cookie on `/kubescope/` and the Traefik-only NetworkPolicy are unchanged. kubescope's `hostGuard` passes any `Host` on a `0.0.0.0` bind |
| landscape | `apps/landscape.yaml` | `route.host: ops.sujaykumar.dev`; `LANDSCAPE_PUBLIC_URL: https://ops.sujaykumar.dev/landscape`. The ForwardAuth login redirect is absolute and built from this value, so if it's stale Longhorn's sign-in loops back to `projects`. **Known side effect:** landscape builds every owner app's "public URL" as the host of `LANDSCAPE_PUBLIC_URL` plus the app's ingress path, so its links to xlearn, airlift and the hub will point at `ops.sujaykumar.dev/<app>` and return 404. That's a cosmetic regression, accepted here (see Risks) |
| Longhorn UI | `infrastructure/storage/ui.yaml` | ``match: Host(`ops.sujaykumar.dev`) && PathPrefix(`/longhorn`)``. Keep the three middlewares in order (auth first); the redirect regex is host-agnostic. Update the header comment |
| airlift admin | new `apps/airlift-admin-routes.yaml` (namespace `airlift`) | (a) **`airlift-admin-block`** on ``Host(`projects.sujaykumar.dev`) && (Path(`/airlift/admin`) \|\| PathPrefix(`/airlift/api/admin/`))`` at priority 110, which beats airlift's 100. Its middleware is `ipAllowList` with `sourceRange: [127.0.0.1/32]`, which answers 403 to every client, and its backend is service `airlift:8443`, which is never reached. (b) **`airlift-admin`** on ``Host(`ops.sujaykumar.dev`) && (Path(`/airlift/admin`) \|\| PathPrefix(`/airlift/api/admin/`) \|\| PathPrefix(`/airlift/assets/`))``, then stripPrefix `/airlift`, then `airlift:8443`, with `tls: {}`. airlift writes a path-only `<base href="/airlift/">`, so the admin page served from `ops` calls `ops`. If the page needs another path (for example `/airlift/sw.js` or `/airlift/api/info`), add that exact path, **never the whole `/airlift` prefix**. The public airlift app stays on `projects` unchanged |
| old URLs | new `apps/ops-redirects.yaml` (namespace `projects-hub`) | An IngressRoute at priority 120 on ``Host(`projects.sujaykumar.dev`) && (Path(`/kubescope`) \|\| Path(`/kubescope/`) \|\| Path(`/landscape`) \|\| Path(`/landscape/`) \|\| Path(`/longhorn`) \|\| Path(`/longhorn/`))``. Its middleware is `redirectRegex` with `^https?://[^/]+/(kubescope\|landscape\|longhorn)/?$` → `https://ops.sujaykumar.dev/${1}/` and `permanent: false`, so a 302 that browsers don't cache. Its backend is the `projects-hub` service, which is never reached. **Only exact roots redirect.** Any deeper console path on `projects` (`/kubescope/api/…`) falls to the hub's catch-all, so nothing console-like answers there |
| *(optional)* ops root | `apps/ops-redirects.yaml` (namespace `landscape`) | ``Host(`ops.sujaykumar.dev`) && Path(`/`)`` → 302 `/landscape/` |

- Middlewares must sit in the same namespace as their route, because the k3s Traefik 3.7 rejects cross-namespace references. The Kustomization for `apps/` has no `kustomization.yaml`, so Flux applies every YAML file in the folder, and the new files need no registration.
- **No new pod and no NetworkPolicy change.** The memory sum is unchanged ([ADR-0035 §5](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#5-capacity-the-memory-sum-rule-triggers-and-ordered-responses)).
- **Render check before pushing.** infra has no CI, so paste these checks into the PR body:
  - `helm template kubescope ./charts/project -f <values extracted from apps/kubescope.yaml>` and the same for landscape, as a diff against `main`. The only differences are the `Host(...)` match and landscape's env.
  - `yq` parses every new and changed file.
  - `hack/host-lint.sh` stays clean. This sprint touches no host script, so this is a sanity run.
- **After the merge:** `ssh vps 'bash -s -- --cluster' < hack/host-verify.sh` (read-only) shows no FAIL.
- **Chart 0.3.0 ([mi-01](sprint-mi-01.md)):** these are value changes, so they're orthogonal to `hack/chart-diff.sh`, which compares charts under identical values. If both PRs are open, whichever merges second rebases and re-runs `chart-diff.sh`.
- **Rewrite the infra README's "Admin consoles" section** in the same PR:
  - the URL table on `ops.sujaykumar.dev`, with the airlift admin row;
  - the gate per console;
  - the cross-origin guarantees each console relies on:
    - kubescope: Go `CrossOriginProtection` runs *before* auth, and the exec socket checks `Origin` (coder/websocket, no dev origins on a non-loopback bind);
    - landscape: its ForwardAuth `crossOriginWrite` refuses same-site writes and WebSocket upgrades;
    - landscape's own API has only GET reads, protected by the same-origin policy (no CORS headers anywhere);
    - airlift: a bearer token, not a cookie;
  - the task 4 result;
  - replace the "Same-origin caveat" paragraph with the same-site caveat, and add a rule: **a console upgrade that drops these checks must bring back the `ipAllowList`**;
  - a known cosmetic regression: landscape's per-app public-URL links for apps that stay on `projects` (xlearn, airlift, the hub) point at `ops` and return 404, until the upstream landscape fix (task 6) lands;
  - the old-URL redirects, and a list of every router this sprint adds (`airlift-admin-block`, `airlift-admin`, the redirects), so [mi-13](sprint-mi-13.md)'s sibling-router deny can find them;
  - update the Layout lines for `storage/ui.yaml` and `landscape.yaml`.

### 4 · Cross-origin acceptance [H]

**4a — agent, credential-free, from the laptop** (none of these carries a session, and every write targets an object that doesn't exist):

| # | Request | Expect |
|---|---|---|
| 1 | `curl -sI https://projects.sujaykumar.dev/{kubescope,landscape,longhorn}/` | `302` → `https://ops.sujaykumar.dev/<console>/` |
| 2 | `curl -s https://projects.sujaykumar.dev/kubescope/api/v1/namespaces`, `/landscape/api/session`, `/longhorn/v1` | not a console (the hub's response); no `unauthenticated` / `cross_origin_rejected` JSON from kubescope, no landscape session JSON |
| 3 | `curl -s -o /dev/null -w '%{http_code}' https://projects.sujaykumar.dev/airlift/admin` and `/airlift/api/admin/config` | `403` |
| 4 | `https://projects.sujaykumar.dev/airlift/` and `/xlearn/api/v1/healthz` | unchanged (airlift page; xLearn version) |
| 5 | `https://ops.sujaykumar.dev/kubescope/healthz`, `/landscape/api/session`, `/airlift/admin` | `200` (signed-out session JSON for landscape) |
| 6 | `curl -sI https://ops.sujaykumar.dev/longhorn/` with `Accept: text/html` | `302` → `https://ops.sujaykumar.dev/landscape/?next=…` (proves `LANDSCAPE_PUBLIC_URL`) |
| 7 | `curl -s -X POST -H 'Origin: https://projects.sujaykumar.dev' -H 'Sec-Fetch-Site: same-site' https://ops.sujaykumar.dev/kubescope/api/v1/workloads/deployments/default/xl-probe-missing/restart`; repeat with `Sec-Fetch-Site: cross-site`, and with `Origin` only | `403 cross_origin_rejected` (kubescope's guard runs before auth) |
| 8 | the same POST with neither header | `401 unauthenticated` (the control) |

**4b — signed-in browser, about 5 minutes.** This is the only faithful test for the WebSocket and the Longhorn paths, because the consoles' cookies must ride the request. `SameSite=Strict` and `Lax` cookies are sent on same-site requests.

It needs a browser signed in to landscape and kubescope on `ops`, which exists only after PR 2 merges and needs
the owner's credentials (an agent never enters them). Run 4b in such a browser if the session can drive one: the
owner may sign in on `ops` once PR 2 merges, but the session never waits for that. **Otherwise (D40, blocking
case):** land PR 1, PR 2 (and PR 3 if a 4a row failed) and the docs PR; set task 4 ⛔ "owner signed-in probe (4b)
pending" in this file and status.md; keep MI-5b 🔄, not ✅, because `ev-first-tester` and l-02 need 4b to pass or
PR 3 to be live. A follow-up session picks up at 4b (the prompt's step 5), then task 5 if a row fails.

1. With landscape and kubescope signed in on `ops`: in kubescope, open a shell on an `xlearn-curriculum` pod. Its image is distroless with no shell, so nothing runs. Copy the exec WebSocket URL from the Network tab.
2. **Run the probe from a `projects` page that sends no CSP: the hub root `https://projects.sujaykumar.dev/`.** Never use an xLearn page. [m1-04](sprint-m1-04.md) gives every xLearn page `connect-src 'self'` (in v1.7.0, which may overlap this sprint), and a CSP blocks `fetch` and `WebSocket` to `ops` inside the browser, before any request leaves. (c)–(e) would then pass falsely, and (a)/(b) would show `(blocked:csp)`.
   - First confirm: `curl -sI https://projects.sujaykumar.dev/ | grep -i content-security-policy` prints nothing (true at planning). If the hub ever sends a CSP, pick another `projects` path that the same check shows sends none.
   - Open a tab on that page and paste the snippet from the prompt into devtools. It sends:
     - (a) the kubescope `POST …/restart` on `xl-probe-missing`, with credentials, `no-cors`;
     - (b) the Longhorn `POST /longhorn/v1/volumes/xl-probe-missing?action=snapshotCreate`;
     - (c) the kubescope exec WebSocket (the copied URL);
     - (d) the Longhorn WebSocket `/longhorn/v1/ws/1s/volumes`;
     - (e) a credentialed cross-origin `fetch` GET of `/landscape/api/graph`.
3. **Pass** needs a real server answer for every row:
   - (a) and (b) show a **server 403** in the Network tab. A 404 means the guard let the write through: **fail**.
   - (c) and (d) show a **403 handshake** (the WS row's status, or the console's `Unexpected response code: 403`) and never reach `open`.
   - (e) rejects with a CORS `TypeError`, and the console names the missing `Access-Control-Allow-Origin` header, not a CSP directive.
   - **Inconclusive, not a pass:** `(blocked:csp)`, `(failed)`, or a network error with no status. Fix the probe page and re-run.
4. **Evidence, read-only.**
   - **kubescope** logs JSON: `ssh vps 'k3s kubectl -n kubescope logs deploy/kubescope --since=15m'`. Look for an `"msg":"http request"` line with `"status":403` on the `…/xl-probe-missing/restart` path (the body `cross_origin_rejected` isn't logged), and an `"msg":"exec websocket upgrade failed"` line whose error says `not authorized`.
   - **Longhorn / landscape ForwardAuth:** landscape doesn't log `/api/forward-auth` (too chatty), so its logs show nothing. The evidence is the browser's 403 for (b) and (d), with the JSON body `{"error":"cross-origin request refused"}` where devtools shows the response.
5. The login check, in the same signed-in browser: landscape, kubescope, Longhorn (through the landscape session) and airlift admin all work on `ops`. The owner updated the bookmarks before launch (`ev-mi5b-dns`); the old-URL redirects cover any that weren't.

Record the 4a and 4b tables (request, expected, observed) in PR 2's description.
- **All pass:** there's no allowlist, and task 5 is n/a.
- **Any fail:** task 5 is mandatory, and the sprint isn't done until the allowlist is live.

### 5 · Interim guard (only if task 4 fails) [I] — infra PR 3 `feat(ops): ipAllowList on kubescope and Longhorn (MI-5b interim)`

- **`ipAllowList` middlewares.** Use Traefik 3.7 `traefik.io/v1alpha1` `Middleware` with `spec.ipAllowList.sourceRange` set to the owner's egress ranges, which the owner gives at launch (a before-launch item in the prompt). Traefik sees real client IPs because of `externalTrafficPolicy: Local` in `traefik-config.yaml`.
  - **Longhorn:** add a middleware in `longhorn-system`, first in `ui.yaml`'s chain.
  - **kubescope:** chart 0.2.2 renders only a ForwardAuth middleware, and chart 0.3.0's knob list is frozen ([rollout §2.1](../rollout-plan.md#21-chart-030-knob-list)). So set `route.enabled: false` in `apps/kubescope.yaml` and add a raw IngressRoute in a new file `apps/kubescope-route.yaml`, with redirect-slash, stripPrefix and `ipAllowList` middlewares.
  - Apply the same treatment to any other console that failed.
- **The ranges** live only in the private infra repo. Never copy them into xlearn, which is a public repo, and never into `status.md`.
- **If no stable ranges were given at launch:** set `KUBESCOPE_READ_ONLY=true`. That disables every kubescope mutation, `exec` included, because there's no exec-only switch. It stays set until the console is fixed upstream, and the README says so. Any other failed console with no such fallback is recorded ⛔ in status.md; don't wait for ranges.
- **Break-glass if the owner's IP changes:** `ssh vps` plus `k3s kubectl` (read-only unless the owner acts), then a one-line PR to update the range.

### 6 · Record [X]

In `docs/v2/status.md`:
- MI rows: **MI-5b ✅** with the date and the infra PR numbers. Say whether the allowlist was needed. If 4b couldn't run, MI-5b stays 🔄 with task 4 ⛔ "owner signed-in probe (4b) pending" (task 4).
- Mark the owner event `ev-mi5b-dns` done.
- The CLI-use log: the entry gate's `identity admin account list`, if the CLI was live.
- Add a note: *required before the first `tester` ([ADR-0033 §11](../../adr/0033-invite-only-admission-and-owner-admin.md#11-admin-console-isolation-mi-5b)), and an opening gate that must stay true ([rollout §11](../rollout-plan.md#11-opening-gates-v3))*.
- Decisions log:
  - a separate `ops-tls` certificate through the TLSStore;
  - airlift admin moved and blocked on the shared origin;
  - the allowlist decision;
  - the register's "update projects-hub links" descoped: the hub is another repo, the redirects cover its `/landscape/` link, and editing it is an optional owner follow-up;
  - landscape's per-app public URLs now point at `ops` (a cosmetic regression), with a link to the upstream issue the agent opens in the landscape repo (derive each app's host from its IngressRoute match). Open an issue for any other console that needs an upstream fix too.

## Acceptance criteria

- [ ] `ops.sujaykumar.dev` resolves to the node on public resolvers and serves a valid Let's Encrypt certificate (`ops-tls` Ready). `projects.sujaykumar.dev` still serves `projects-tls`.
- [ ] **No admin console answers on `projects.sujaykumar.dev`:**
  - console roots return a 302 to `ops`;
  - no console API responds there;
  - `/airlift/admin` and `/airlift/api/admin/*` return 403;
  - `/airlift/` and `/xlearn/` are unaffected.
- [ ] **The consoles refuse cross-origin mutating calls and WebSocket upgrades** from `https://projects.sujaykumar.dev`: 4a #7 returns 403, and 4b (a)–(e) pass. **Or** the `ipAllowList` (PR 3) is live on every console that failed.
- [ ] **Sign-in to landscape and kubescope works at the new URLs** (4b's signed-in browser). Longhorn opens through the landscape session on `ops`, and airlift admin works on `ops`.
- [ ] Flux `infra-configs`, `infra-storage` and `apps` are Ready, and `host-verify --cluster` is green after the merges.
- [ ] The infra README's "Admin consoles" section is rewritten, and `docs/v2/status.md` records MI-5b ✅.

## Release

**Infra PR(s) only.**
- PR 1 (certificate) → PR 2 (routes + README) → PR 3 only if task 4 fails, in that order. infra has no CI, so each PR body carries the pasted checks (the `helm template` diff where a chart value changes, the `yq` parse, and `hack/host-lint.sh`). Then merge it under the standing authority, and Flux applies it.
- After PR 2, and after PR 3 if it's needed, run `ssh vps 'bash -s -- --cluster' < hack/host-verify.sh` (read-only). It must show no FAIL.
- No xlearn tag: nothing here ships in a tag. The status update rides an xlearn docs PR.
- There's no snapshot, because this isn't a contract, erase or GA tag and it doesn't restart the host.

## Definition of Done

- Infra PRs are merged and reconciled by Flux; nothing is applied by hand ([ADR-0009](../../adr/0009-deployment-and-gitops.md)).
- The acceptance criteria are met, and the acceptance tables are in PR 2.
- Owner bookmarks were updated before launch (`ev-mi5b-dns`).
- Statuses are updated in this file and in [`../status.md`](../status.md).
- Local `main` is synced in both repos.

## Risks / watch-outs

- **Same-site, not same-origin.** `ops` and `projects` share a site, so the consoles' cookies ride XSS-triggered requests, and CORS doesn't cover WebSocket upgrades. The isolation holds only because each console checks `Origin` / `Sec-Fetch-Site` itself. That is why task 4 runs, and why a console upgrade that drops those checks needs the allowlist back (README rule).
- **The owner's monitoring moves.** landscape and kubescope are the D34 monitoring. The old-URL redirects soften the change. The owner updates the bookmarks before launch (`ev-mi5b-dns`).
- **landscape and Longhorn are coupled** through the `ls_session` cookie and `LANDSCAPE_PUBLIC_URL`. Moving one without the other breaks the Longhorn sign-in.
- **airlift's admin token is the shared admin password.** Blocking its admin page on `projects` closes the `sessionStorage` path. Rotating `projects-admin` afterwards is cheap (infra README "Rotate it"), but its consumer restart is a hand step, so rotation is the owner's call and isn't planned. Decoupling the token belongs in the airlift repo.
- **Cookie tossing.** Script on `projects` can set a `Domain=sujaykumar.dev` cookie that `ops` receives. It can shadow a console cookie and sign the owner out, but it can't forge a session, because tokens are HMAC-signed. `__Host-` cookie names would close this; that belongs in the console repos.
- **HTTP-01 needs propagated DNS.** If the challenge fails, `ops` shows a certificate error or a 404. `projects` is unaffected because its certificate is separate.
- **Ordering hazard.** The M1b `identity admin` CLI can mint a `tester` before this sprint lands. `ev-first-tester` and [l-02](sprint-l-02.md) gate on MI-5b, so check `status.md` before minting.
- **An allowlist with a dynamic home IP can lock the owner out.** Keep the break-glass path (`ssh vps`) and a one-line range PR.
- **A CSP on the probe page fakes a pass.** A page with `connect-src 'self'` (every xLearn page once [m1-04](sprint-m1-04.md) ships) blocks the 4b requests in the browser, so the WebSocket and CORS rows "pass" without reaching `ops`. Probe from a page that sends no CSP (the hub root), and count only real server 403s.
- **landscape's app links regress (cosmetic).** `LANDSCAPE_PUBLIC_URL` on `ops` makes landscape's public-URL links for xlearn, airlift and the hub point at `ops.sujaykumar.dev/<app>`, which returns 404. The consoles and the monitoring views still work. It's recorded in the infra README, and the fix is the upstream landscape issue.

# Prompt — Sprint mi-04 · Admin consoles to ops.sujaykumar.dev (MI-5b)

> **One self-contained prompt = one sprint = one session.** Paste it into a fresh coding session at the xlearn repo root.
> **Plan:** [`../sprints/sprint-mi-04.md`](../sprints/sprint-mi-04.md) · **Milestone:** MI (rollout step MI-5b) · **Prereqs:** none (the DNS record is a before-launch owner item)

## Before you launch (owner)

Launching this prompt attests these are done (D40). If one turns out to be missing, land everything that doesn't depend on it and record the gap as ⛔ in `status.md`; don't wait.

- [ ] Add the DNS record `A ops` in the Hostinger zone `sujaykumar.dev`: the same IPv4 as `projects.sujaykumar.dev`, TTL 300, no AAAA (`ev-mi5b-dns`).
- [ ] Update your bookmarks to `https://ops.sujaykumar.dev/{kubescope,landscape,longhorn}/` and `https://ops.sujaykumar.dev/airlift/admin` (`ev-mi5b-dns`). They answer once PR 2 merges; after that the old console roots 302 there.
- [ ] If you have stable egress IP ranges, give them in the launch message. PR 3 uses them only if the cross-origin acceptance fails, and they go only into the private infra repo, never xlearn or `status.md`. Without them, PR 3 falls back to `KUBESCOPE_READ_ONLY=true`.

## Read first

- [`../sprints/sprint-mi-04.md`](../sprints/sprint-mi-04.md): the plan. It holds the task 3 change table, the 4a/4b acceptance tables, the README rewrite list and the Risks. This prompt follows it step by step.
- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md): repo conventions, GitOps, and land-and-sync.
- [ADR-0033 §11 Admin-console isolation](../../adr/0033-invite-only-admission-and-owner-admin.md#11-admin-console-isolation-mi-5b) and [§12 row 15](../../adr/0033-invite-only-admission-and-owner-admin.md#12-authz-deltas-across-v2): the gate, the same-site caveat and the acceptance.
- [Rollout §2 (the MI-5b row)](../rollout-plan.md#2-mi-infra-track), [§2.2 operating rules](../rollout-plan.md#22-operating-rules-every-mi-step-and-every-tag), [§6 owner calendar](../rollout-plan.md#6-critical-path-parallel-tracks-owner-calendar) and [§11 opening gates](../rollout-plan.md#11-opening-gates-v3).
- [ADR-0035 §3](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#3-no-alerting-in-v2-owner-d34): landscape and kubescope are the owner's only monitoring (D34), so they must keep working.
- The sibling infra repo (read it all before editing): `../infra/README.md` "Admin consoles", `../infra/apps/{kubescope,landscape,airlift,projects-hub}.yaml`, `../infra/infrastructure/storage/ui.yaml`, `../infra/infrastructure/configs/{certificate,tlsstore,clusterissuers,traefik-config}.yaml`, `../infra/charts/project/templates/ingressroute.yaml` and `values.yaml` (`route.host`), and `../infra/clusters/vps/{apps,infrastructure,storage}.yaml`.
- The consoles' own guards (read-only; they explain why the acceptance should pass). The repos are local siblings, paths relative to the xlearn root:
  - kubescope (`../../skriptvalley/kubescope`): `internal/server/csrf.go`, `internal/server/server.go` (guard order, and `requestLogger`, which logs JSON), `internal/stream/exec.go` (`OriginPatterns`);
  - landscape (`../landscape`): `forwardAuthH`, `crossOriginWrite` and `logMW` in `internal/server/server.go`. `logMW` skips `/api/forward-auth`, and the app-detail code builds each app's public URL from `LANDSCAPE_PUBLIC_URL`'s host;
  - airlift (`../airlift`): the `/api/admin` routes, `admin.go` and `web/src/admin/main.ts` (token in `sessionStorage`).

## Context

- **The problem.** Every admin console sits on xLearn's origin, `projects.sujaykumar.dev`, separated only by path:
  - kubescope (cluster-admin, `exec`);
  - landscape;
  - the read-write Longhorn UI, gated by landscape's ForwardAuth;
  - airlift's admin surface. Its bearer token is the shared `projects-admin` password, and its admin page keeps that token in `sessionStorage` on the shared origin.

  v2 adds learner-, tester- and AI-rendered content to that origin. One XSS while the owner is signed in could then drive kubescope to cluster-admin, and from there to `sops-age`, with no backups (D12).
- **The fix (MI-5b).** Move the consoles to their own host, `ops.sujaykumar.dev`, and prove they refuse same-site cross-origin writes and WebSocket upgrades. The same-site part matters: sibling subdomains still carry cookies. If a console fails that check, fence it with an IP allowlist.
- **The deadline.** MI-5b is a hard gate before the **first non-owner account** (a CLI-minted `tester` included; `ev-first-tester`, [l-02](../sprints/sprint-l-02.md)) and an opening gate for v3. It depends on no other MI step.
- **The ops facts at planning time (2026-09-24):**
  - Traefik is 3.7.8 with `externalTrafficPolicy: Local`, so client IPs are real.
  - One shared certificate, `projects-tls`, sits in `kube-system` and is served through the `default` TLSStore.
  - DNS is on Hostinger (`dns-parking.com`), and no `ops` record exists.
  - Chart 0.2.2 already exposes `route.host`.

## Entry gates — verify first (stop and report if any is unmet)

- [ ] `ev-mi5b-dns` was done before launch (the DNS record and bookmarks); step 1 verifies the record resolves.
- [ ] `git -C ../infra fetch && git -C ../infra status` shows a clean tree on up-to-date `main`. `gh pr list -R sujaykumarsuman/infra --state open` shows no peer PR touching the files in the plan's task 3, or you've agreed an order. Also check `git worktree list` and `ListAgents` for peers.
- [ ] No non-owner account exists yet. If the M1b CLI is live, run `identity admin account list` yourself through the sanctioned admin-CLI `kubectl exec` path ([rollout §2.2](../rollout-plan.md#22-operating-rules-every-mi-step-and-every-tag)); launching this prompt pre-approves that one use (D40). Log it in `docs/v2/status.md`'s CLI-use log, as §2.2 requires; otherwise your `ssh sujaykumar-vps` access stays read-only. If a `tester` exists, continue anyway, but log the gap in the Decisions log as urgent.

## Do this (in order)

1. **[O, before launch] DNS (`ev-mi5b-dns`): verify it.**
   - The owner added `A ops` in the Hostinger zone `sujaykumar.dev` before launch, with the same IPv4 as `projects` and TTL 300. No AAAA.
   - Check that `dig +short ops.sujaykumar.dev A`, `dig @1.1.1.1 …` and `dig @8.8.8.8 …` all equal `dig +short projects.sujaykumar.dev A`, and `ssh sujaykumar-vps 'getent hosts ops.sujaykumar.dev'` agrees. Poll briefly if it's still propagating (TTL 300).
   - Don't open PR 1 for merge before that.
   - **If the record is missing:** everything else depends on it (the certificate's HTTP-01 challenge, then PR 2). Set task 1 ⛔ "DNS record missing (owner, before launch)" in the sprint file and `status.md`, land that as the docs PR (step 7), and end there; a re-run starts at step 1. Don't wait.
2. **[I] Infra PR 1: the certificate.**
   - Branch `feat/ops-tls` in `../infra`.
   - Add a `Certificate` `ops-tls` in `kube-system` (secret `ops-tls`, dnsNames `[ops.sujaykumar.dev]`, ClusterIssuer `letsencrypt-prod`) to `infrastructure/configs/certificate.yaml`.
   - Add `certificates: [{secretName: ops-tls}]` to the `default` TLSStore in `tlsstore.yaml`. Keep `defaultCertificate: projects-tls`.
   - Commit `feat(tls): certificate for ops.sujaykumar.dev (MI-5b)` and open the PR. infra has no CI, so paste the `yq` parse of both files and the `hack/host-lint.sh` output (a sanity run) into the PR body. Then merge it under the standing authority.
   - Let Flux reconcile. There's no `kubectl apply` and no `flux` write.
   - Verify read-only: `ssh sujaykumar-vps 'k3s kubectl -n kube-system get certificate ops-tls'` shows Ready, and `openssl s_client … -servername ops.sujaykumar.dev` shows the new certificate while `projects` still serves `projects-tls`.
3. **[I] Infra PR 2: move the consoles** (branch `feat/ops-consoles`, after `ops-tls` is Ready). Make exactly the changes in the plan's task 3 table:
   - `apps/kubescope.yaml`: `route.host`;
   - `apps/landscape.yaml`: `route.host` and `LANDSCAPE_PUBLIC_URL=https://ops.sujaykumar.dev/landscape`;
   - `infrastructure/storage/ui.yaml`: the `Host` match, with the middleware order unchanged;
   - new `apps/airlift-admin-routes.yaml` (namespace `airlift`): the 403 block on `projects` at priority 110, using an `ipAllowList` of `127.0.0.1/32`, and the admin route on `ops` with stripPrefix;
   - new `apps/ops-redirects.yaml`: exact-root 302s on `projects` in namespace `projects-hub` at priority 120, plus the optional `ops` root → `/landscape/` route in namespace `landscape`;
   - the README "Admin consoles" rewrite, as the plan lists it.

   - In the README rewrite, also note the known cosmetic regression: landscape's per-app public-URL links for apps that stay on `projects` (xlearn, airlift, the hub) now point at `ops` and return 404, until the upstream landscape fix lands.

   Before pushing (infra has no CI, so paste all three into the PR body):
   - `helm template` kubescope and landscape with their values, as a diff against `main`. Only the `Host(...)` match and landscape's env may change.
   - `yq` parses the new and changed files.
   - `hack/host-lint.sh` stays clean (a sanity run; no host script changes).
   - If mi-01's chart PR is open, coordinate the rebase and the `chart-diff.sh` re-run.

   Then commit `feat(ops): move admin consoles to ops.sujaykumar.dev (MI-5b)`, open the PR, merge it under the standing authority, and let Flux reconcile.
   - Confirm read-only: `ssh sujaykumar-vps 'k3s kubectl get ingressroute -A -o custom-columns=NS:.metadata.namespace,NAME:.metadata.name,MATCH:.spec.routes[*].match'` shows no console on `projects` other than the redirect and the 403 block.
   - Verify the host and cluster, read-only: `ssh sujaykumar-vps 'bash -s -- --cluster' < ../infra/hack/host-verify.sh`. It writes nothing on the node. Expect no FAIL.
4. **[H] Acceptance 4a** (credential-free, from the laptop).
   - Run the plan's table rows 1–8 with `curl` (`-s -o /dev/null -w '%{http_code} %{redirect_url}\n'`, or print the body where the table expects JSON).
   - Every write targets `xl-probe-missing`, so nothing changes even if a guard were missing.
   - Paste the observed column into PR 2's description.
5. **[H] Acceptance 4b** (a browser signed in to the consoles on `ops`).
   - It needs landscape and kubescope signed in on `ops`, which exists only after PR 2 and needs the owner's credentials; you never enter them. If this session can drive such a browser (the owner may sign in on `ops` once PR 2 merges; don't wait for that), run 4b there.
   - **Otherwise (D40, blocking case):** set task 4 ⛔ "owner signed-in probe (4b) pending" in the sprint file and `status.md`, and keep MI-5b 🔄, not ✅ (`ev-first-tester` and l-02 need 4b to pass or PR 3 to be live). Do step 6 only if a 4a row failed, then step 7. A follow-up session re-runs this step, then step 6 if any row fails.
   - In kubescope, open a shell on an `xlearn-curriculum` pod (distroless, no shell, so nothing runs) and copy the exec WebSocket URL from the Network tab into `EXEC_URL`.
   - **The probe page must send no CSP.** Use the hub root, `https://projects.sujaykumar.dev/`, **never an xLearn page**. [m1-04](../sprints/sprint-m1-04.md) gives every xLearn page `connect-src 'self'` (in v1.7.0, which may overlap this sprint). A CSP blocks `fetch` and `WebSocket` to `ops` inside the browser before any request leaves, so (c)–(e) would pass falsely.
     - First run `curl -sI https://projects.sujaykumar.dev/ | grep -i content-security-policy`. It must print nothing (true at planning). If the hub sends a CSP, find another `projects` path that the same check shows sends none.
   - On a tab at that page, paste this snippet into devtools:
   ```js
   const ops = 'https://ops.sujaykumar.dev';
   const EXEC_URL = 'wss://ops.sujaykumar.dev/kubescope/…copied…';
   const post = (n, u) => fetch(u, {method: 'POST', credentials: 'include', mode: 'no-cors'})
     .then(() => console.log(n, 'sent — read its status in the Network tab (expect 403)'));
   post('a kubescope write', `${ops}/kubescope/api/v1/workloads/deployments/default/xl-probe-missing/restart`);
   post('b longhorn write', `${ops}/longhorn/v1/volumes/xl-probe-missing?action=snapshotCreate`);
   for (const [n, u] of [['c kubescope exec ws', EXEC_URL], ['d longhorn ws', `wss://ops.sujaykumar.dev/longhorn/v1/ws/1s/volumes`]]) {
     const ws = new WebSocket(u);
     ws.onopen = () => { console.error(n, 'FAIL: opened'); ws.close(); };
     ws.onerror = () => console.log(n, 'ok: refused');
   }
   fetch(`${ops}/landscape/api/graph`, {credentials: 'include'})
     .then(() => console.error('e FAIL: cross-origin read succeeded'))
     .catch(() => console.log('e ok: blocked by CORS'));
   ```
   **Pass** means all of these, each backed by a real server answer (the console lines alone aren't enough):
   - (a) and (b) show a **server 403** in the Network tab. A **404 is a fail**: the write got through to the console.
   - (c) and (d) log `refused`, **and** their handshake shows **403** (the WS row's status in the Network tab, or the console's `Unexpected response code: 403`).
   - (e) logs `blocked`, and the console's error names the missing `Access-Control-Allow-Origin` header, not a CSP directive.
   - **Inconclusive, never a pass:** `(blocked:csp)`, `(failed)`, or a network error with no status. Fix the probe page and re-run.

   **Evidence, read-only:**
   - kubescope logs JSON: `ssh sujaykumar-vps 'k3s kubectl -n kubescope logs deploy/kubescope --since=15m'`. Look for an `"msg":"http request"` line with `"status":403` on the `…/xl-probe-missing/restart` path (the `cross_origin_rejected` code is only in the response body, never logged), and an `"msg":"exec websocket upgrade failed"` line whose error says `not authorized`.
   - Longhorn / ForwardAuth: landscape's `logMW` skips `/api/forward-auth`, so its logs show nothing. Use the browser's 403 for (b) and (d), with the JSON body `{"error":"cross-origin request refused"}` where devtools shows the response.

   Then, in the same signed-in browser, check that landscape, kubescope, Longhorn (via the landscape session) and airlift admin work on `ops`. Add the 4b table to PR 2.
6. **[I] Only if any 4a/4b row fails: infra PR 3.**
   - Add `ipAllowList` middlewares with the owner's egress ranges (given at launch; only in the private infra repo, never in xlearn).
   - **Longhorn:** a Middleware in `longhorn-system`, first in `ui.yaml`'s chain.
   - **kubescope:** set `route.enabled: false` and add a raw IngressRoute with redirect-slash, stripPrefix and `ipAllowList` in `apps/kubescope-route.yaml`. Don't add a chart knob; chart 0.3.0's list is frozen.
   - Do the same for any other console that failed.
   - Fallback when no stable ranges were given at launch: `KUBESCOPE_READ_ONLY=true`. Record any other failed console with no fallback ⛔ in `status.md`; don't wait for ranges.
   - Commit `feat(ops): ipAllowList on kubescope and Longhorn (MI-5b interim)`. infra has no CI, so paste the `yq` parse and the `hack/host-lint.sh` output into the PR body. Then merge it under the standing authority, reconcile, and re-run the failed rows. They must now return 403 from off-range addresses.
   - Re-run `ssh sujaykumar-vps 'bash -s -- --cluster' < ../infra/hack/host-verify.sh` (read-only). Expect no FAIL.
7. **[X] Record.**
   - Branch `docs/mi-04-status` in xlearn. Update this sprint's Status table and `docs/v2/status.md` (below).
   - Open an issue in the landscape repo (`sujaykumarsuman/landscape`): derive each app's public-URL host from its IngressRoute match rather than from `LANDSCAPE_PUBLIC_URL`. Link it from the Decisions log. Don't change landscape's code.
   - Commit `docs(v2): MI-5b live — admin consoles on ops.sujaykumar.dev`, open the PR, and merge it once xlearn's CI is green.

## Constraints

- **GitOps only.** Every cluster change is an infra PR that Flux reconciles. Never run `kubectl apply`, `kubectl edit`, `flux suspend` or `flux reconcile` against production. Use `ssh sujaykumar-vps 'k3s kubectl get|logs …'` for reads only.
  - **Your laptop's kubectl context may tunnel to production** (`127.0.0.1:6443`), so don't use it for anything.
  - Infra PRs are their own tasks ([rollout §2.2](../rollout-plan.md#22-operating-rules-every-mi-step-and-every-tag)).
- **No chart change.** Chart 0.2.2's `route.host` is enough, and chart 0.3.0's knob list is frozen ([rollout §2.1](../rollout-plan.md#21-chart-030-knob-list)). Use raw manifests for any extra route or middleware, in the route's own namespace.
- **Memory-sum rule** ([ADR-0035 §5](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#5-capacity-the-memory-sum-rule-triggers-and-ordered-responses)): this sprint adds no pod. If you find yourself adding one, stop.
- **No alerting (D34).** Add no Flux Alert, no healthcheck, no push channel. The acceptance is observed by hand and recorded.
- **Credentials.**
  - Never read, print or copy `projects-admin`, the session keys or the owner's IP ranges into xlearn, which is a public repo.
  - The owner types the console password in their own browser.
  - The probes you run carry no credentials.
- **Probes must be harmless.**
  - Every write targets a name that doesn't exist.
  - Exec targets a distroless pod.
  - Never probe with a real volume, deployment or node.
- **Parallel sessions.** Check peers' infra PRs, `git worktree list` and `ListAgents` before opening each PR, and rebase onto a moved `main` before merging. ([mi-01](../sprints/sprint-mi-01.md), [mi-03](../sprints/sprint-mi-03.md) and [mi-14](../sprints/sprint-mi-14.md) also edit `../infra`.)
- **Scope.** Don't touch xLearn's CSP or gateway checks ([m1-04](../sprints/sprint-m1-04.md)), Permissions-Policy ([mi-13](../sprints/sprint-mi-13.md)), the airlift, kubescope or landscape source repos, or the hub site repo.

## Deliverables

- `../infra` PR 1: `infrastructure/configs/certificate.yaml` (`ops-tls`) and `tlsstore.yaml` (a `certificates` entry).
- `../infra` PR 2:
  - `apps/kubescope.yaml`, `apps/landscape.yaml` and `infrastructure/storage/ui.yaml` moved to `ops`;
  - the new `apps/airlift-admin-routes.yaml` and `apps/ops-redirects.yaml`;
  - the README "Admin consoles" rewrite;
  - the acceptance tables in the PR description.
- *(Conditional)* `../infra` PR 3: `ipAllowList` middlewares, plus `apps/kubescope-route.yaml`.
- xlearn docs PR: this sprint's Status table and `docs/v2/status.md`.

## Update status

- In [`../sprints/sprint-mi-04.md`](../sprints/sprint-mi-04.md), move each task ⬜ → 🔄 → ✅, or ⛔ with a reason. Task 5 becomes "✅ n/a" when 4a/4b pass. Set _Overall_ ✅ when every task is done.
- In [`../status.md`](../status.md):
  - the **Sprint board** row for mi-04;
  - the **MI rows**: MI-5b ✅, with the date, the infra PR numbers and whether the allowlist was needed. If 4b couldn't run: MI-5b 🔄 and task 4 ⛔ "owner signed-in probe (4b) pending", so `ev-first-tester` and l-02 see it;
  - the owner calendar event `ev-mi5b-dns` done;
  - the note "required before the first `tester`; an opening gate that must stay true";
  - **Decisions log** lines: the separate `ops-tls` certificate through the TLSStore; airlift admin moved and blocked on `projects`; the allowlist decision; any console that needed a fallback; the register's "update projects-hub links" descoped (another repo, the redirects cover the hub's `/landscape/` link, an optional owner follow-up); landscape's per-app public URLs now point at `ops` (a cosmetic regression), with the upstream issue link;
  - the **CLI-use log**: the admin-CLI `kubectl exec` you ran for the entry gate (rollout §2.2);
- No new ADR is expected, because ADR-0033 §11 already decides this. If a console needs a code fix upstream, open an issue in its repo and link it from the Decisions log instead.

## Done when (acceptance)

- [ ] `ops.sujaykumar.dev` resolves publicly and serves a valid `ops-tls` certificate. `projects` still serves `projects-tls`.
- [ ] No admin console answers on `projects.sujaykumar.dev`: roots 302 to `ops`, no console API responds, airlift admin returns 403, and airlift and xLearn are unaffected.
- [ ] The consoles refuse cross-origin mutating calls and WebSocket upgrades from `projects` (4a #7, 4b a–e), or the `ipAllowList` is live on every console that failed.
- [ ] Sign-in to landscape and kubescope works at the new URLs (4b's signed-in browser); Longhorn and airlift admin work on `ops`.
- [ ] Flux `infra-configs`, `infra-storage` and `apps` are Ready, and `host-verify --cluster` is green.
- [ ] The infra README is rewritten, and `docs/v2/status.md` records MI-5b ✅.

## Ship (land-and-sync — owner approval pre-granted)

> Launching this prompt is the owner's approval for every change it makes (D40); don't stop for review.

1. Branch, then conventional commit(s) with the attribution lines, then push, then a PR in every repo touched (`../infra` PRs first where the order requires it; infra PRs are never folded into a tag). Here: `feat/ops-tls` and `feat/ops-consoles` (plus PR 3's branch if needed) in `../infra`, then `docs/mi-04-status` in xlearn.
2. Once CI is green (fix, then merge, on failure), squash-merge. Never enable auto-merge. infra has no CI: each PR body carries the pasted checks, and you merge on them.
3. **Release action — infra PR(s) only:** Merge the infra PRs in the plan's order (each its own PR, never folded into a tag): PR 1 (certificate) → PR 2 (routes + README) → PR 3 only if a 4a/4b row fails. Let Flux apply each one and verify live, with `host-verify --cluster` after PR 2 (and PR 3). Then the xlearn docs/status PR. No tag.
4. Update status: the sprint file and `docs/v2/status.md`, in the same PR or a follow-up docs PR merged the same way.
5. Run `git checkout main && git pull` in every repo touched (xlearn and `../infra`). If a clean peer worktree holds `main`, use `git -C <worktree> merge --ff-only origin/main` and then `git switch --detach main`.

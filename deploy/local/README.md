# Local review stack

A full local xLearn — the gateway + all six internal services on a local Postgres +
NATS/JetStream — for reviewing UI/UX changes without the k3s cluster. This is a **review
aid only**; production is GitOps via `../infra` + Flux.

## Run

```bash
docker compose up --build
```

First run builds seven images (the gateway also builds the SPA) and can take a few
minutes. Then open **http://localhost:8080/xlearn**.

## Sign in

There is no OAuth app registered for `localhost`, so use the **"Dev sign in (local)"**
button on the sign-in screen. It appears because `identity` runs with `DEV_AUTH=1` here
(the endpoint 404s — and the button is hidden — in the prod images). It mints a session
for a fixed local account and drops you on the Catalog home.

Email sign-up works here because compose sets `SIGNUP_MODE=open`. Without it identity
defaults to **closed** (prod: `403 signup_closed`, ADR-0023 §2).

To exercise the real GitHub flow instead, register
`http://localhost:8080/xlearn/api/auth/github/callback` as a callback on the OAuth app
and use "Continue with GitHub" (the client id/secret come from the repo `.env`).

## What to look at

- **Home** (`/xlearn`) — no left nav; the DSA card shows **Start path** until you start
  it, then **Day N · streak · today** with **Continue**.
- **Inside a curriculum** (`/xlearn/dsa/...`) — the left nav returns; the **curriculum
  selector** sits in the top bar (where search used to be); Settings/Progress are only in
  the avatar menu.

## Reset

```bash
docker compose down -v   # drops the Postgres + NATS volumes (fresh seed + a new dev account)
```

**PG major bump → `docker compose down -v`.** The stack runs `postgres:18-alpine` (prod parity). A
volume initialised by an older major (e.g. PG 16) is incompatible, and PG 18 images keep their data
under `/var/lib/postgresql/18/docker` (the volume mounts `/var/lib/postgresql`). If Postgres exits
on startup after a pull, drop the volumes once with `docker compose down -v`.

## Eval pack in compose

judge (m3-05) reads the private eval pack at `/evalpack` ([t1 §3.7](../../docs/v2/research/t1-content-data-model.md#37-local-dev-and-public-ci)).
The top-level anchor `x-evalpack-mount` in `docker-compose.yml` is that mount, read-only. m3-05's judge service
attaches it with `volumes: [*evalpack-mount]`.

- **Default:** `./internal/judge/testdata/pack`, the SYNTHETIC fixture pack. It is the built pack of the
  `fixture` course (`fx-001`…`fx-003`), rebuilt from `internal/judge/testdata/packsrc` by CI's `pack-fixture`
  job. It needs no secrets.
- **The real pack** is local only and never committed. In `../xlearn-evalpack`, run `make build`, then start
  compose with `EVALPACK_DIR=../xlearn-evalpack/build docker compose up`.
- **With only the default mount**, judge starts **Ready with 0 evaluable**. The compose services embed the real
  curriculum, which has no `fixture` course, so every pack item is `invalid` (no public item). That matches
  t1 §3.7's "no pack" behaviour.

**The dev-only content overlay (`FIXTURE_CONTENT_DIR`).** This is a contract for m3-05, which builds it; it is
not built yet. It makes the fixture items evaluable in compose, for m3-05's "N evaluable" check and the
m3-06, m3-11 and m3-12 compose e2e.

- `FIXTURE_CONTENT_DIR=<dir>` loads that content root **in addition to** the embedded curriculum.
- It is honoured only when `DEV_AUTH` is set (ADR-0033 §3's guard pattern). Without `DEV_AUTH` it is ignored and
  logged at ERROR, and the service still starts.
- judge, curriculum and practice all read the same variable:
  - judge resolves the pack's items;
  - curriculum seeds the `fixture` path, so the SPA lists it;
  - practice resolves the items for attempts.
- Compose mounts `./internal/judge/testdata/content` read-only on all three and sets the variable there.
- Ids can't collide: the items use the `fx-` prefix, the fixture root has its own `ids.lock.json`, and the
  loader's cross-course uniqueness guard applies.
- With the overlay, `GET /internal/evaluable?path=fixture` lists `fx-001…003` as `ok`. m3-05's compose check
  reads `path=fixture`, not `path=dsa`.

## Runner in compose (dev mode)

The runner ([`docs/architecture/runner.md`](../../docs/architecture/runner.md)) sits behind the compose profile
`runner`, so a plain `docker compose up` doesn't start it:

```bash
docker compose --profile runner up --build runner
```

- **Dev mode, privileged, compose only.** It runs with `RUNNER_MODE=dev`, `privileged: true` and a private cgroup
  namespace, because it builds cgroups and mounts for its jails. Production never runs privileged: mi-14's VAP
  forbids it, and the pod uses a user namespace, AppArmor and seccomp instead. `dev` only downgrades the failing
  *security* canaries (AppArmor label, UID map, userns, `fsopen`, SCTP) to warnings, so `/readyz` is 200 and jobs
  run. Limits: 3 GiB, 2 CPUs (the pod's).
- **Apple Silicon.** The image builds for arm64 from the same Dockerfile (the base is pinned by its multi-arch
  index digest). The jails then use the arm64 dev exec allowlists, which are KILL-default like amd64's. Docker
  Desktop's kernel has no Ubuntu AppArmor userns restriction, which is one more reason this is `dev` mode only.
- **Restarts are normal.** `restart: unless-stopped`: the runner drains and exits **on purpose** after any SIGSYS
  (a learner hit the seccomp filter), after `RUNNER_MAX_JOBS` jobs and after `RUNNER_MAX_AGE`. Each restart has a
  new `boot_epoch` in `GET /v1/profiles`.
- **Token.** `deploy/local/runner-token.dev` is a committed, obviously non-secret dev-only token, used nowhere but
  compose. judge's compose service (m3-05/m3-06) must send the **same value** (`RUNNER_TOKEN`, or mount the same
  file).
- **Reaching it.** Port 8090 stays on the compose network (`http://runner:8090`). To run the acceptance suite
  from the host, add the loopback-only port override:

  ```bash
  docker compose -f docker-compose.yml -f deploy/local/runner-ports.compose.yml --profile runner up -d --build runner
  make runner-acceptance RUNNER_URL=http://127.0.0.1:18090 RUNNER_TOKEN_FILE=deploy/local/runner-token.dev SUBSET=full
  ```

  `SUBSET` is required (`full` here; `prod` is mi-10's production run). The JSON report lands in `bin/`.

## Ports

- `8080` → gateway / SPA (the only one you need)
- `5433` → Postgres (host side; `psql postgres://xlearn:xlearn@localhost:5433/xlearndb`)
- `8090` → the runner (compose network only; `127.0.0.1:18090` with the override above)

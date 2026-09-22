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

## Ports

- `8080` → gateway / SPA (the only one you need)
- `5433` → Postgres (host side; `psql postgres://xlearn:xlearn@localhost:5433/xlearndb`)

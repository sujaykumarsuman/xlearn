# C4 L2 — Containers

```mermaid
graph TB
    browser["👤 Browser — React SPA (Vite, TS)"]
    traefik["Traefik · projects.sujaykumar.dev"]

    subgraph xlearn_ns["k3s · namespace: xlearn"]
      gw["gateway (BFF) · edge<br/>serves SPA + /xlearn/api · session auth · aggregation"]
      id["identity"]
      cur["curriculum"]
      prac["practice"]
      rev["review<br/>(scheduler + mistakes + sweep + notifications)"]
      asm["assessment<br/>(mock + progress projections)"]
      coach["coach (BYO-key)"]
      nats["NATS JetStream"]
    end

    subgraph db_ns["namespace: databases"]
      pg[("CloudNativePG projects-pgstore<br/>xlearndb: schema per service")]
    end

    llm["LLM provider (user key)"]
    oauth["GitHub / Google OAuth"]

    browser --> traefik -->|/xlearn| gw
    gw -->|JWT| id & cur & prac & rev & asm & coach
    id --> oauth
    coach --> llm

    prac -- publish --> nats
    id -- publish --> nats
    asm -- publish --> nats
    nats -- consume --> rev
    nats -- consume --> asm

    id & cur & prac & rev & asm & coach --- pg
```

- **Edge:** only `gateway` has a Traefik route (`/xlearn`); everything else is ClusterIP.
- **Sync:** gateway → services over HTTP/JSON with a propagated JWT.
- **Async:** producers (`practice`, `identity`, `assessment`) → NATS; consumers (`review`,
  `assessment`) react. See [`../events.md`](../events.md).
- **Data:** every service binds the shared `xlearndb` but only its own schema.

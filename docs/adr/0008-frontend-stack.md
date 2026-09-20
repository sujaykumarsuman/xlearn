# ADR-0008 — Frontend stack

- **Status:** Accepted
- **Date:** 2026-09-20
- **Deciders:** @sujaykumarsuman
- **Related:** [0003](0003-service-decomposition.md), [0006](0006-authn-authz.md), [0009](0009-deployment-and-gitops.md)

## Context

The frontend is **React + TypeScript** (given). We have a complete design system as **plain CSS**:
`design-system/theme.css` defines `--ds-*` tokens and `ds-*` / `xl-*` component classes (the dark
"landscape console" look). We need a build tool, routing, a server-state library, a way to consume
the design system without rewriting it, and an auth flow against the BFF — all servable as static
assets behind the gateway at `/xlearn`.

## Decision

| Concern | Choice | Why |
|---------|--------|-----|
| **Build tool** | **Vite** | Fast dev server + HMR, first-class TS, simple static `dist/` output to embed in the gateway. Matches sibling repos' `web/` + npm scripts (typecheck/lint/test/build). |
| **Language** | **TypeScript** (strict) | Given; strict mode for a data-heavy app. |
| **Routing** | **React Router** (data router) | Routes are already specified; nested layouts fit the sidebar+topbar shell; base path `/xlearn`. |
| **Server state** | **TanStack Query** (React Query) | The app is read-heavy over the BFF (dashboard, week, revision queue). Caching, invalidation, background refetch, and mutation states fit the mechanics (timers, outcome logging). |
| **Client state** | React state + Context; **Zustand** only if a genuinely global store appears (coach panel, active timer). | Avoid Redux ceremony; most state is server state. |
| **Design system** | **Port `theme.css` as-is — CSS variables + component classes. No Tailwind.** | The DS is already authored as tokens + `ds-*`/`xl-*` classes; reusing it verbatim keeps parity with the design source of truth and the sibling design-system repo. Tailwind would mean re-expressing the whole system. |
| **Styling method** | Global `theme.css` (tokens + shared components) + **CSS Modules** for screen-local styles. | Keeps the shared DS global and reusable; scopes per-screen CSS without a CSS-in-JS runtime. |
| **Forms/validation** | Native + **Zod** schemas shared with API DTOs where practical. | Lightweight; Zod doubles as runtime validation of API responses. |
| **Icons/fonts** | Inter + JetBrains Mono via the DS; icons as inline SVG per the artboards. | Matches `theme.css` (`--ds-font-sans`/`--ds-font-mono`). |
| **Testing** | **Vitest** + Testing Library; Playwright for a couple of critical flows later. | Vite-native; matches sibling `web` CI jobs. |

### Delivery & auth flow

- Built to static assets and **embedded into the gateway** binary (`//go:embed web/dist`), served
  under `/xlearn` with SPA fallback — one deployable for the web edge (mirrors airlift/landscape).
- **Auth:** unauthenticated → redirect to `identity` OAuth (server flow, [0006](0006-authn-authz.md));
  the browser holds only the HttpOnly session cookie. The SPA calls the BFF with `credentials: include`;
  a 401 triggers the login redirect. No tokens in JS.
- **API base:** `/xlearn/api` (same origin → cookie just works; no CORS).

## Consequences

- ✅ The design system is reused verbatim — pixel parity with the artboards, zero re-theming.
- ✅ One web deployable (embedded in gateway); same CI shape as sibling repos.
- ✅ TanStack Query matches the read-heavy, cache-friendly screen set.
- ⚠️ Server-authoritative timers ([R-PF3](../prd/xlearn-prd.md#61-guided-problem-flow-gated-stages))
  mean the client HUD mirrors server state — needs careful sync (poll or SSE), not a naive local timer.
- ⚠️ CSS Modules + a global DS require a naming discipline so screen CSS doesn't fight `xl-*`/`ds-*`.

## Alternatives considered

| Option | Why not |
|--------|---------|
| **Next.js** | SSR/RSC unneeded for an authed SPA behind a BFF; heavier deploy than static assets embedded in Go. |
| **Tailwind** | Would re-implement an existing, complete token+component CSS system; loses parity with the DS repo. |
| **Redux Toolkit** | Overkill — nearly all state is server state (TanStack Query) or trivial local UI state. |
| **Separate static web service** | Extra deployable/route; embedding in the gateway is one image, same origin, no CORS. |
| **CRA / Webpack** | Slower DX than Vite; not used by sibling repos. |

# ADR-0001 — Record architecture decisions

- **Status:** Accepted
- **Date:** 2026-09-20
- **Deciders:** @sujaykumarsuman

## Context

xLearn is design-complete but pre-code, and is built in phases by (mostly) an AI-assisted solo
developer. Decisions made now — service boundaries, comms, data ownership, deploy — will be
re-encountered every session and by every future agent working in the repo. Without a durable,
skimmable record, decisions get re-litigated, drift, or are silently reversed.

## Decision

We keep **Architecture Decision Records** in [`docs/adr/`](.), numbered and
[MADR](https://adr.github.io/madr/)-style, **append-only**:

- Global (span all versions); per-version execution lives under `docs/vX/`.
- One decision per file: **Status · Context · Decision · Consequences · Alternatives considered**.
- A decision is reversed by a **new** ADR that supersedes the old one — the old file stays.
- The [index](README.md) is the table of contents.

## Consequences

- ✅ Any agent/session can reconstruct *why* the system is shaped as it is from the repo alone.
- ✅ Cheap to add; forces the trade-off to be written down once.
- ⚠️ Requires discipline: notable decisions must actually be recorded (called out in
  [`AGENT.md`](../../AGENT.md) and the sprint DoD).

## Alternatives considered

| Option | Why not |
|--------|---------|
| Decisions in code comments / commit messages | Not discoverable; no trade-off record; scatter. |
| A single `DECISIONS.md` | Grows unreviewable; no stable per-decision anchor to link/supersede. |
| A wiki / external tool | Off-repo; agents can't read it; drifts from the code. |

# Diagrams

Mermaid sources (render on GitHub). Kept here as editable standalones; some are also embedded inline
in the architecture docs for reading flow.

| File | What |
|------|------|
| [`context.md`](context.md) | C4 L1 — system context (who/what talks to xLearn). |
| [`containers.md`](containers.md) | C4 L2 — services, broker, DB, edge. |
| [`deployment.md`](deployment.md) | GitOps deploy flow (repo → GHCR → Flux → k3s) + runtime placement. |
| revision loop | Sequence diagrams live in [`../events.md`](../events.md). |

Edit here; keep the inline copies in `overview.md` in sync when a diagram changes materially.

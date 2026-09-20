# xLearn v1 — status

Cross-sprint living tracker. Updated per the
[status protocol](sprints/README.md#status-protocol-way-of-working) whenever a task or sprint changes
state (it's baked into every `prompt-sNN.md`). Per-task detail lives in each
[`sprints/sprint-NN.md`](sprints/); newest decisions at the top of the log.

- **Build line:** v1 · **Phase:** Pre-code (docs & architecture scaffold complete)
- **Deploy mode:** build-semver auto-deploy from `main` ([ADR-0009](../adr/0009-deployment-and-gitops.md))
- **Last updated:** 2026-09-20

## Snapshot

| Area | State |
|------|-------|
| PRD | ✅ [drafted](../prd/xlearn-prd.md) |
| ADRs 0001-0009 | ✅ accepted |
| Architecture (overview/services/data/api/events/diagrams) | ✅ drafted |
| Git strategy | ✅ drafted |
| v1 build plan | ✅ done |
| Sprint plans + prompts (S01-S12) | ✅ all scaffolded |
| Application code | ⬜ not started |
| `infra` xlearn wiring | ⬜ not started |

## Sprint board

| Sprint | Focus | State |
|--------|-------|-------|
| [S01](sprints/sprint-01.md) | Bootstrap + app shell (→ M0 live at `/xlearn`) | ⬜ Next |
| [S02](sprints/sprint-02.md) | identity / auth (→ M1) | ⬜ Planned |
| [S03](sprints/sprint-03.md) | curriculum + Catalog/Roadmap | ⬜ Planned |
| [S04](sprints/sprint-04.md) | Week + Concept (→ M2) | ⬜ Planned |
| [S05](sprints/sprint-05.md) | practice / Problem ★ (→ M3) | ⬜ Planned |
| [S06](sprints/sprint-06.md) | review scheduler / Revision | ⬜ Planned |
| [S07](sprints/sprint-07.md) | mistakes + notifications (→ M4) | ⬜ Planned |
| [S08](sprints/sprint-08.md) | assessment / Mock | ⬜ Planned |
| [S09](sprints/sprint-09.md) | progress + Dashboard ★ (→ M5) | ⬜ Planned |
| [S10](sprints/sprint-10.md) | Settings | ⬜ Planned |
| [S11](sprints/sprint-11.md) | coach (→ M6) | ⬜ Planned |
| [S12](sprints/sprint-12.md) | hardening + 1.0 (→ M7) | ⬜ Planned |

Legend: ✅ done · 🔄 in progress · ⬜ planned/not started · ⛔ blocked. Update per the
[status protocol](sprints/README.md#status-protocol-way-of-working).

## Milestones

| ID | Target | State |
|----|--------|-------|
| M0 live at `/xlearn` | end S01 | ⬜ |
| M1 login | end S02 | ⬜ |
| M2 browse curriculum | end S04 | ⬜ |
| M3 guided problem | end S05 | ⬜ |
| M4 repetition+mistakes | end S07 | ⬜ |
| M5 mock+analytics | end S09 | ⬜ |
| M6 feature-complete | end S11 | ⬜ |
| M7 1.0 | S12 | ⬜ |

## Decisions log

Notable calls not (yet) worth a full ADR, newest first. Promote to an ADR if they harden.

| Date | Decision | Notes |
|------|----------|-------|
| 2026-09-20 | Docs/architecture scaffold created; ADRs 0001-0009 accepted. | This scaffold. See open assumptions below. |
| 2026-09-20 | `review` groups scheduler + mistakes; `assessment` groups mock + progress. | Node-frugal; split later if needed ([ADR-0003](../adr/0003-service-decomposition.md)). |
| 2026-09-20 | Build-semver auto-deploy from `main` for v1; switch to release tags at 1.0. | Matches `projects-hub`/`landscape`. |

## Open assumptions (confirm / correct)

Tracked from the PRD [open questions](../prd/xlearn-prd.md#10-open-questions) and scaffold assumptions:

- [ ] **Code execution:** re-implementation is **self/auto-assessed**, not sandbox-executed in v1.
- [ ] **Curriculum source:** versioned **seed in-repo** (`curriculum/`), not an in-app CMS.
- [ ] **LLM providers at launch:** **OpenAI + Anthropic**.
- [ ] **Staging:** **none** in v1 (single node; trunk-based auto-deploy).
- [ ] **NATS placement:** its own `messaging` namespace vs inside `xlearn` — default `xlearn`.
- [ ] **Data isolation:** schema-per-service in one `xlearndb` (not DB-per-service) for v1.

## Blocked / needs input

_(none — proceeding on the assumptions above until corrected.)_

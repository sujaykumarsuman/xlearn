# v1 sprints

Per-sprint **plans**. Each sprint is one flat file `sprint-NN.md` and pairs with exactly one
execution **prompt** in [`../prompts/`](../prompts/) — `prompt-sNN.md` — designed to run **one prompt
per sprint per session**. Phases/sequencing/milestones live in [`../build-plan.md`](../build-plan.md);
the live cross-sprint tracker is [`../status.md`](../status.md).

## Layout

```
docs/v1/
  sprints/
    README.md          this index + the status protocol
    sprint-01.md … sprint-12.md   one plan per sprint (goal · scope · tasks+status · acceptance · DoD · risks)
  prompts/
    README.md
    prompt-s01.md … prompt-s12.md one self-contained prompt per sprint (paste, run, done)
```

## Index

| Sprint | Focus | Milestone | Plan | Prompt |
|--------|-------|-----------|------|--------|
| 01 | Bootstrap + app shell | M0 (live at `/xlearn`) | [sprint-01](sprint-01.md) | [prompt-s01](../prompts/prompt-s01.md) |
| 02 | identity / auth | M1 | [sprint-02](sprint-02.md) | [prompt-s02](../prompts/prompt-s02.md) |
| 03 | curriculum + Catalog/Roadmap | — | [sprint-03](sprint-03.md) | [prompt-s03](../prompts/prompt-s03.md) |
| 04 | Week + Concept | M2 | [sprint-04](sprint-04.md) | [prompt-s04](../prompts/prompt-s04.md) |
| 05 | practice / Problem ★ | M3 | [sprint-05](sprint-05.md) | [prompt-s05](../prompts/prompt-s05.md) |
| 06 | review scheduler / Revision | — | [sprint-06](sprint-06.md) | [prompt-s06](../prompts/prompt-s06.md) |
| 07 | mistakes + notifications | M4 | [sprint-07](sprint-07.md) | [prompt-s07](../prompts/prompt-s07.md) |
| 08 | assessment / Mock | — | [sprint-08](sprint-08.md) | [prompt-s08](../prompts/prompt-s08.md) |
| 09 | progress + Dashboard ★ | M5 | [sprint-09](sprint-09.md) | [prompt-s09](../prompts/prompt-s09.md) |
| 10 | Settings | — | [sprint-10](sprint-10.md) | [prompt-s10](../prompts/prompt-s10.md) |
| 11 | coach | M6 | [sprint-11](sprint-11.md) | [prompt-s11](../prompts/prompt-s11.md) |
| 12 | hardening + 1.0 | M7 | [sprint-12](sprint-12.md) | [prompt-s12](../prompts/prompt-s12.md) |

## Status protocol (way of working)

Progress is tracked in **two places that must stay in sync**: each sprint's **Status** table
(`sprint-NN.md`) and the cross-sprint tracker ([`../status.md`](../status.md)). Statuses:

| Icon | Meaning |
|------|---------|
| ⬜ | Not started |
| 🔄 | In progress |
| ✅ | Done (its acceptance bullet passes) |
| ⛔ | Blocked (note why) |

**When a task changes state:**
1. Update that task's row in the sprint's **Status** table (⬜ → 🔄 when you pick it up; → ✅ when its
   acceptance criterion is met; → ⛔ if blocked, with a one-line reason).
2. Update the sprint's **_Overall_** line (🔄 once any task is in progress; ✅ when all tasks are ✅).
3. When the sprint's overall status changes, mirror it in [`../status.md`](../status.md): the **Sprint
   board** row and — if the sprint carries a milestone — the **Milestones** table. Add a line to the
   **Decisions log** there for any notable call.

`build-plan.md` is the *static* plan (it does not track live status); the sprint files + `status.md`
are the single source of progress truth. Executing a `prompt-sNN.md` includes doing these updates —
each prompt ends with an **Update status** step.

## Adding/adjusting a sprint

- Keep the flat `sprint-NN.md` + `prompt-sNN.md` pairing (one prompt = one session).
- Follow the shape of an existing sprint file and prompt. Record notable decisions as ADRs.
- Sprints are pre-scaffolded for the whole of v1; refine a sprint's detail as it approaches if the
  code has drifted from the plan.

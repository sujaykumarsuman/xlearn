# F004 — Add xLearn to projects-hub

**Status:** ✅ done (local review) · **Opened:** 2026-09-22

## Feedback

Add xLearn as a project on **projects-hub** (the `projects.sujaykumar.dev` landing page) and
keep it as the **first** project.

## Decision

- projects-hub is served from the **`sujaykumarsuman.github.io`** repo, `projects/` subdir
  (built as the `projects-hub` image, deployed by `../infra`) — confirmed via the `landscape`
  repo's source mapping. The entry is added there, ordered first.
- Match the existing project-card shape/tokens in that repo; link to
  `https://projects.sujaykumar.dev/xlearn`.

## Scope / changes

- `sujaykumarsuman.github.io/projects/**` — add the xLearn card as the first entry (exact file
  determined from that repo's structure).

## Changes done

- `sujaykumarsuman.github.io/projects/index.html` — xLearn added as card **01** (status `live`,
  route `/xlearn`, tags Go · React · spaced-repetition, source link to the xlearn repo);
  airlift → 02, landscape → 03; the "Running" count → **3 apps**. Matches the existing card markup.

**Verified** by rendering the page (static). It's a separate repo, so it lands in its own PR under
the review→ship gate and is not part of the xlearn docker-compose.

## Notes

- Separate repo → its own PR in the review→ship gate; won't appear in the xlearn
  docker-compose (shown via its own diff / local render).

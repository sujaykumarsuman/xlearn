# m1-03 DSA parity evidence (not for merge)

Screenshots for [sprint m1-03](https://github.com/sujaykumarsuman/xlearn/blob/main/docs/v2/sprints/sprint-m1-03.md)'s
DSA parity check (golden = v1). This branch holds evidence only and is never merged.

- `v1.6.0/`: the published `ghcr.io/sujaykumarsuman/xlearn-*:1.6.0` images in the local compose topology.
- `branch/`: `feat/m1b-course-resolution`, built from source in the same topology.
- `diff/`: pixelmatch diff images (threshold 0.1) for the screens that differ.

Both stacks were seeded with the same learner state through the public API, a few seconds apart:
dev login, start DSA, week-1 problems 1 (clean), 2 (rough), 3 (hint revealed, assisted), problem 5 in
progress, one journal entry, one scored mock. Each screen was shot at 1440×900 and 390×844 (full page,
Chromium headless shell, reduced motion). The course switcher was shot open.

Result (`diff-table.txt`): every DSA screen is pixel-identical. The only differences are:

- `coming-soon` (`/xlearn/system-design`): the new AB02-F4 teaser. v1 showed NotFound there. This is intended.
- `settings`: 17 px in the profile card's live clock (the two shots were taken a minute apart). Not a change.

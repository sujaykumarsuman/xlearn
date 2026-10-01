# Envelope fixtures (m1-02)

SYNTHETIC v1/v2 event envelopes — never copied from production. One account
(`0b5e7a3c-…`), `occurred_at` 2026-09-21T12:00:00Z.

- `<subject>.v1.json` — the v1 envelope every producer emits in v1.6.0.
- `<subject>.v2.json` — the SAME event as a v2 envelope (`version: 2`, `path_slug: "dsa"`;
  identity's account-scoped `account_created` carries no `path_slug`). Each consumer's twin
  test feeds both and requires identical store calls / rows.
- `<subject>.v2-nopath.json` — a course-scoped v2 event without `path_slug`: decodes to
  `events.ErrInvalidEnvelope`, which the consumer dead-letters.
- `problem_solved.v3-extra.json` — an unknown future version with extra fields: still decodes.
- `problem_solved.v2-assist.json` — `problem_solved.v2.json` plus m1-07's additive `data.assist`
  object (D27, `v1.7.0`): every consumer must make the same store call as for the v2 twin.
- `problem_solved.v1-noversion.json` — a v1 envelope without `version`: read as v1 (DSA).

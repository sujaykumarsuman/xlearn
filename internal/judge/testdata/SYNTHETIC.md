SYNTHETIC — hand-made test data, never derived from xlearn-evalpack.

# The fixture course and its eval pack (sprint m3-02)

Everything under `internal/judge/testdata/` is original and synthetic, written for the public pipeline's own
tests. Nothing here comes from, or relates to, the private eval pack (`xlearn-evalpack`).

- `content/` is a separate content root holding a `fixture` course (not real DSA items) with three items:
  `fx-001` (function mode, `exact`), `fx-002` (function mode, `unordered_deep`, a WA and a TLE wrong solution)
  and `fx-003` (class mode, `class-ops@1`). Each has Go, C++ and Python references under `_code/`. The root
  passes the curriculum loader and its guards (`paths.json`, `ids.lock.json`, `courses/fixture/…`). It is never
  embedded, so it is never seeded in production. A dev-only overlay (`FIXTURE_CONTENT_DIR`, built in m3-05)
  loads it beside the embedded curriculum in compose.
- `packsrc/` is that course's pack source: format 1 (`pack.json`, `tests.lock`, `tests/edge.jsonl`, `gen/`,
  `validate/`, `invalid/`, `submissions/{brute.go, wrong/*}`).
- `pack/` is the built pack, `packlint build --dockerfile=false` over `packsrc/` (`manifest.json` and one
  `cases.jsonl.zst` per item). Compose mounts it at `/evalpack` by default. `validated_against` is all zeros,
  because the fixture is validated against its own tree.

The CI job `pack-fixture` rebuilds `pack/` from `packsrc/` (all gates, through docker) and diffs it. The test
`internal/packspec/fixture_test.go` checks `pack/` without docker; `go test ./internal/packspec -run Fixture
-update` rewrites `pack/` from a fresh build. This marker lets the repo-wide pack-artefact lint accept
`tests.lock` and `cases.jsonl.zst` here.

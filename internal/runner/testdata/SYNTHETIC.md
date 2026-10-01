# Synthetic test data

Everything under `internal/runner/testdata/` is hand-made for the runner's own tests:

- `corpus/<name>/main.go`: the hostile corpus (t3 §9's P2 rows) that the `runner-it` lane compiles
  and runs inside the jail under the test-only `testgo@0` / `testgo-open@0` profiles (m3-03);
- `items/<slug>/`: seven synthetic code items for the launch profiles (m3-04): `item.json` (the
  frozen item-schema shape with public samples), `cases.jsonl` (hidden cases; when a line has no
  `expected`, the Go reference's output is the expected one), `refs/solution.{go,cpp,py}` and, for
  `pair-sum` and `reverse-list`, `wrong/<class>.{go,cpp,py}` whose first line names the verdict the
  jail and the test-only term mapper must give. `internal/runner/profile/refs_it_test.go` runs them
  through the real jail; the perf cases are generated in that test.

None of it comes from, or relates to, the private eval pack (`xlearn-evalpack`). `testdata` keeps
these programs out of `./...`; they are compiled only inside the jail.

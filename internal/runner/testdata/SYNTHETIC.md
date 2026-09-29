# Synthetic test data

Everything under `internal/runner/testdata/` is hand-made for the runner's own tests: the hostile
corpus (`corpus/<name>/main.go`, t3 §9's P2 rows) that the `runner-it` lane compiles and runs inside
the jail under the test-only `testgo@0` / `testgo-open@0` profiles. None of it comes from, or relates
to, the private eval pack (`xlearn-evalpack`).

`testdata` keeps these programs out of `./...`; they are compiled only inside the jail.

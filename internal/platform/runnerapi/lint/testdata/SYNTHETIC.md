# Synthetic test data

Everything under `internal/platform/runnerapi/lint/testdata/` is hand-made for the lint's golden
tests: one fixture per rule and per language, a clean file, and one evasion file per language that
the lint misses on purpose (the jail stops it; `internal/runner/profile/refs_it_test.go` runs the
same evasion programs). Each file's first line names the rules it must trip (`want:`). None of it
comes from, or relates to, the private eval pack (`xlearn-evalpack`).

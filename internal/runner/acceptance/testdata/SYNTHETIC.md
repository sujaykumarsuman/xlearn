SYNTHETIC — hand-made test data, never derived from xlearn-evalpack.

# Acceptance-suite probes and kernels

Everything under `internal/runner/acceptance/testdata/` is hand-made for the runner acceptance suite
(`make runner-acceptance`, m3-15):

- `probes/<lang>/<section>.<ext>`: hostile and probing programs written against `func-json@1` with the
  signature `probe(arg string) string`, so they take the normal profile path (`go@1.26`, `cpp@g++14`,
  `python@3.13`). `net` is section B (network), `syscalls` section C, `corpus` section D (t3 §9's P2
  rows, plus the no-op and spin programs sections A and F use) and `markers` section E (cross-job
  markers, t3 §5.10).
- `kernels/<lang>/<kernel>.<ext>`: the five calibration kernels (t3 §7.3: integer loop, sort 1e6,
  map-heavy, alloc/GC-heavy, BFS on 2e5 edges) with the signature `kernel(n int) int64`. The three
  languages do the same work and return the same checksum.

They are learner-style files (Go `package main` without `func main`, several per directory), so
`testdata` keeps them out of `go build/vet/test ./...`; the suite embeds them and the runner compiles
them inside its jail. None of it comes from, or relates to, the private eval pack (`xlearn-evalpack`).

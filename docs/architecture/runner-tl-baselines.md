# Runner time-limit baselines

Per-profile baselines for the pack TL gate ([m3-02](../v2/sprints/sprint-m3-02.md), re-gated in [m3-13](../v2/sprints/sprint-m3-13.md)).
The rule is [ADR-0030 §3](../adr/0030-runner-technology-and-host-hardening.md#3-timing-on-a-noisy-4-vcpu-vm) and
[t3 §7.2–§7.3](../v2/research/t3-sandbox.md#73-calibration). The numbers come from the acceptance suite's calibration
mode ([`runner.md`](runner.md#tests), `make runner-acceptance … CALIBRATE=1`).

> **Status: provisional.** The CI column (m3-15) is filled. The production column is **pending
> [mi-10](../v2/sprints/sprint-mi-10.md)**, which fills it in its docs PR. Until then every TL computed from this page is
> provisional ([m3-02](../v2/sprints/sprint-m3-02.md)'s flag), and [m3-13](../v2/sprints/sprint-m3-13.md)'s re-gate
> clears it.

## The rule

- **Pack TL** = max(3 × the Go reference's CPU time **on the production profile**, 1 s). The verdict clock is CPU
  time: the case cgroup's `usage_usec` delta.
- **A language's TL** = the pack TL × that profile's `tl_multiplier`, which `GET /v1/profiles` serves. The runner
  enforces only the TL it is sent.
- **CI references are scaled** by `speed_index(prod) / speed_index(CI)`. Here `speed_index` is the geometric mean of the
  five `go@1.26` kernel medians, in ms: higher is slower.
- **TLs only scale up.** They never shrink.
- **A patch bump** (a toolchain patch, a new snapshot date) re-runs the speed index:
  - within ±5% of this page's value for the same profile, TLs carry forward;
  - otherwise they scale up by the ratio and are flagged.

  A minor bump, or a CPU-model change, is a full recalibration.
- **Keys:** a baseline is `baseline@1` keyed to the profile's `profile_sha256`, not to the image digest. The hash
  covers the toolchain trees, the profile's commands, env, limits and seccomp lists, and the harness templates. It is
  commit-independent: GOROOT and the GOCACHE seed have fixed mtimes ([`runner.md`, Image](runner.md#image-deployrunnerdockerfile)),
  so a runner tag that changes no toolchain keeps every hash.

## How the numbers are made

- **Five kernels** (t3 §7.3; `internal/runner/acceptance/testdata/kernels/`) run with the same `n` in Go, C++ and
  Python, and return the same checksum in all three: intloop (2e7 LCG steps), sort (1e6 values), maphash (1e6 updates
  and 1e6 lookups), alloc (1e6 list nodes in 4 rounds) and BFS (2e5 edges).
- **30 runs each**, sequential, on one slot, in jobs of 10 cases, through the normal profile path (`func-json@1`).
- **Per kernel:** the median case CPU ms and the CV. Plus the steal over the window (`Telemetry.StealPct`) and the
  kernel ratios against `go@1.26`.
- **The ratios are informational.** `intloop` is a pure interpreter loop, so it bounds Python from above; the served
  multipliers are m3-04's, from the reference perf cases. mi-10 calibrates them on production.

## Pending

The per-profile tables (CI column) are filled from this PR's CI run `runner-image-acceptance` before merge.

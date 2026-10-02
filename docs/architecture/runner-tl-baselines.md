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

## Per profile

**The CI column** comes from xlearn#116's `runner-image-acceptance` lane, run 37029026421, on 2026-10-02:
- the image built from this Dockerfile, run in dev mode and privileged on a GitHub-hosted `ubuntu-24.04` runner;
- 2 CPUs and 3 GiB, on an AMD EPYC 7763;
- steal 0.00% during the window; `CanaryMedian` 144,480 µs (CV 0.067 over 139 samples).

**The production column** is mi-10's: the same profiles at the same `profile_sha256` (the image is
digest-identical), with `SUBSET=prod REQUIRE_PROD=1 CALIBRATE=1` on the node.

**The image:** `runner-v1.0.0` at `ghcr.io/sujaykumarsuman/xlearn-runner:1.0.0`. Its digest is recorded in
[`docs/v2/status.md`](../v2/status.md#release-streams) (runner stream).

**Checksums:** every kernel returned the same checksum in all three languages (the same work).

### `go@1.26` (the baseline profile)

| | CI (m3-15) | Production (mi-10) |
|---|---|---|
| `profile_sha256` | `f342dced42e8db4ccd0aed7bdde8329b54ec63ccec7692e4a82ab14cdcca694a` | pending mi-10 (expected unchanged) |
| Toolchain | go1.26.8 (tarball), GOCACHE seed `gocache-seed@1` | — |
| Memory baseline | 2,560 KiB | pending mi-10 |
| `tl_multiplier` (served) | 1.0, the baseline (`calibrated: false`) | 1.0 |
| intloop, n = 2e7 | 30.0 ms (CV 0.014) | pending mi-10 |
| sort, n = 1e6 | 91.0 ms (CV 0.009) | pending mi-10 |
| maphash, n = 1e6 | 262.0 ms (CV 0.060) | pending mi-10 |
| alloc, n = 1e6 | 46.0 ms (CV 0.027) | pending mi-10 |
| bfs, 2e5 edges | 18.5 ms (CV 0.060) | pending mi-10 |
| **`speed_index`** | **57.13 ms** | pending mi-10 |
| Steal during the run | 0.00% (max 0.00%) | pending mi-10 |
| `CanaryMedian` | 144,480 µs | pending mi-10 |

### `cpp@g++14`

| | CI (m3-15) | Production (mi-10) |
|---|---|---|
| `profile_sha256` | `a9b110d085309305c0e80edf118c8708e45be56861553d4f6a759552c92d243e` | pending mi-10 (expected unchanged) |
| Toolchain | g++ 14.2.0 (Debian trixie, snapshot `20261001T000000Z`); no PCH | — |
| Memory baseline | 256 KiB | pending mi-10 |
| `tl_multiplier` (served) | 1.0, provisional (m3-04: reference perf ratios 0.41–0.77) | pending mi-10 (calibrated) |
| intloop / sort / maphash / alloc / bfs | 26.0 / 75.0 / 482.5 / 26.0 / 17.0 ms (CV 0.019 / 0.008 / 0.103 / 0.040 / 0.123) | pending mi-10 |
| Kernel ratio vs Go | 0.87 / 0.82 / 1.84 / 0.57 / 0.92; geomean 0.93, max 1.84 | pending mi-10 |

The C++ maphash ratio (`std::unordered_map` vs Go's map) is the one kernel above 1. m3-04's reference perf cases
stay at or under 0.77.

### `python@3.13`

| | CI (m3-15) | Production (mi-10) |
|---|---|---|
| `profile_sha256` | `e676423af7c036f5008c19451cd26da7d94d6fc2f18e2dd7b7dd15e2c953d0f9` | pending mi-10 (expected unchanged) |
| Toolchain | CPython 3.13.5 (Debian trixie, snapshot `20261001T000000Z`); stdlib bytecode `checked-hash` | — |
| Memory baseline | 3,776 KiB | pending mi-10 |
| `tl_multiplier` (served) | 6.5, provisional (m3-04: reference perf ratios 2.50–6.32) | pending mi-10 (calibrated) |
| intloop / sort / maphash / alloc / bfs | 2,618.0 / 783.5 / 759.5 / 491.0 / 195.5 ms (CV 0.068 / 0.015 / 0.033 / 0.009 / 0.049) | pending mi-10 |
| Kernel ratio vs Go | 87.27 / 8.61 / 2.90 / 10.67 / 10.57; geomean 11.97, max 87.27 | pending mi-10 |

For Python, the kernels bracket the served 6.5:
- `intloop` is a pure interpreter loop (87×);
- `maphash` is mostly C (2.9×);
- the rest sit near 10×.

m3-04 measured 6.32 at most over the reference perf cases. Those cases, not the kernels, are what pack TLs apply to.
The multiplier stays provisional until mi-10 calibrates it on production.

## For pack authors

- Compute a pack TL from the Go reference on the **production** profile. Until the production column exists, take
  the CI reference and scale it by `speed_index(prod) / speed_index(CI)`, with the CI value **57.13 ms**. Every
  TL computed this way is **provisional** (m3-02's flag) until m3-13's re-gate.
- C++ and Python TLs are the Go TL × the served `tl_multiplier`. Don't hand-tune them per item.
- A runner patch that moves `speed_index` by more than ±5% scales every TL up and flags it. A TL never shrinks.

## Updating this page

- **mi-10** fills the production column in its docs PR. It confirms the three `profile_sha256` values are unchanged,
  records `baseline@1`, `CanaryMedian`, the steal and the calibrated multipliers, and drops the "provisional" banner
  for the profiles it calibrates.
- **A runner patch** (`runner-vX.Y.Z+1`) re-runs CI's lane. Compare its `speed_index` here, then mi-10's, against the
  previous production value.

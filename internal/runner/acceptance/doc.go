// Package acceptance is the runner acceptance suite (m3-15 task 4; t3 §5.10's A8 release gate,
// §5.3's /readyz canaries, §7.3's calibration): a black-box HTTP client that submits jobs straight
// to a running xlearn-runner (bypassing judge's lint on purpose: it tests the jail, not the lint)
// and reads GET /v1/stats and GET /v1/profiles.
//
// It lives behind the build tag runner_acceptance, so `go test ./...` never runs it. Run it with
//
//	make runner-acceptance RUNNER_URL=… RUNNER_TOKEN_FILE=… SUBSET=full|prod [REQUIRE_PROD=1] [CALIBRATE=1]
//
// SUBSET has no default: full is sections A–H (CI's in-image lane, the VM rehearsal), prod is A–F
// and H with t3 §5.10's A8 counts and no G (mi-10 on production; the rc smoke); REQUIRE_PROD=1
// adds section I and fails a dev-mode runner. The suite is rotation-aware: the runner drains and
// exits after any SIGSYS (m3-03), so every SIGSYS-expected case runs inside one job per language
// per section, and after each such job the suite waits (≤ 6 min) for /readyz and a new BootEpoch.
// An epoch change no SIGSYS explains, a container oom_kill, or /v1/stats going away outside an
// expected rotation aborts the run. It writes a JSON report (bin/, never committed) and prints a
// Markdown summary. docs/architecture/runner.md has the section table.
package acceptance

// Package it holds the runner's Linux integration tests (build tags linux && runner_it): the
// hostile corpus, the cleanup invariants, the L14 caps and the API contract, run against the
// real runner binary in a jail-capable environment (CI's runner-it job: a privileged
// debian:trixie-slim container with a private cgroup namespace; or `make runner-it` as root on
// a Linux VM). macOS can't run the jail.
package it

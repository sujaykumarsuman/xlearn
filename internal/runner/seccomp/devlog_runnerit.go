//go:build runner_it

package seccomp

// devLogBuild lets a runner_it build honour the arm64 dev-VM LOG default (with RUNNER_MODE=dev
// only) until m3-04 adds the arm64 name lists. Never in a release or image build.
const devLogBuild = true

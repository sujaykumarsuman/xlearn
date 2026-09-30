//go:build !runner_it

package seccomp

// devLogBuild is false in every release and image build: exec filters are KILL-default on
// every arch. Only a runner_it test build may honour the arm64 dev-VM LOG switch.
const devLogBuild = false

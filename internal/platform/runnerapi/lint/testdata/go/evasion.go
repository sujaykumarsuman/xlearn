// want: go_import
// The Go allowlist leaves no lint-clean path to a system call, so the Go evasion fixture is one
// the lint rejects; refs_it_test posts it past the lint to prove the jail stops it anyway
// (socket → SIGSYS under the KILL-default exec filter).
package main

import "syscall"

func pairSum(nums []int, target int) []int {
	fd, _ := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, 0)
	return []int{fd}
}

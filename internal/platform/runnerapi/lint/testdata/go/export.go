// want: go_directive go_directive
package main

//export pairSum
//go:noinline
func pairSum(nums []int, target int) []int { return nil }

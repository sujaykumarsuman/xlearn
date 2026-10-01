// want: TLE
package main

// pairSum tries every pair: O(n^2) on the perf cases.
func pairSum(nums []int, target int) []int {
	for i := range nums {
		for j := i + 1; j < len(nums); j++ {
			if nums[i]+nums[j] == target {
				return []int{i, j}
			}
		}
	}
	return nil
}

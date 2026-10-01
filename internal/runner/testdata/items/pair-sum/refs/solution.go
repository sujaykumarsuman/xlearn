package main

// pairSum remembers each value's index and looks up the complement: one pass, O(n).
func pairSum(nums []int, target int) []int {
	seen := make(map[int]int, len(nums))
	for i, x := range nums {
		if j, ok := seen[target-x]; ok {
			return []int{j, i}
		}
		if _, ok := seen[x]; !ok {
			seen[x] = i
		}
	}
	return nil
}

package main

// canSplit (wrong: TLE) recomputes the left sum for every cut.
func canSplit(nums []int) bool {
	total := 0
	for _, x := range nums {
		total += x
	}
	for cut := 1; cut < len(nums); cut++ {
		left := 0
		for i := 0; i < cut; i++ {
			left += nums[i]
		}
		if 2*left == total {
			return true
		}
	}
	return false
}

package main

// canSplit (wrong: WA) also tries the cut after the last element, so the right part may be empty.
func canSplit(nums []int) bool {
	total := 0
	for _, x := range nums {
		total += x
	}
	left := 0
	for i := 0; i < len(nums); i++ {
		left += nums[i]
		if 2*left == total {
			return true
		}
	}
	return false
}

package main

// canSplit keeps a running left sum and compares it with the rest of the total.
func canSplit(nums []int) bool {
	total := 0
	for _, x := range nums {
		total += x
	}
	left := 0
	for i := 0; i < len(nums)-1; i++ {
		left += nums[i]
		if 2*left == total {
			return true
		}
	}
	return false
}

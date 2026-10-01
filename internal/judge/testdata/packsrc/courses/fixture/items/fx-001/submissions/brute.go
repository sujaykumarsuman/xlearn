package main

// canSplit (oracle) tries every cut and sums both sides from scratch.
func canSplit(nums []int) bool {
	for cut := 1; cut < len(nums); cut++ {
		l, r := 0, 0
		for i, x := range nums {
			if i < cut {
				l += x
			} else {
				r += x
			}
		}
		if l == r {
			return true
		}
	}
	return false
}

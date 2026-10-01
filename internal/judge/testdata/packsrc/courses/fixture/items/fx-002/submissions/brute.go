package main

import "sort"

// trios (oracle) checks every triple of positions and keeps the distinct sorted trios.
func trios(nums []int, target int) [][]int {
	seen := map[[3]int]bool{}
	out := [][]int{}
	for i := 0; i < len(nums); i++ {
		for j := i + 1; j < len(nums); j++ {
			for k := j + 1; k < len(nums); k++ {
				if nums[i]+nums[j]+nums[k] != target {
					continue
				}
				t := []int{nums[i], nums[j], nums[k]}
				sort.Ints(t)
				key := [3]int{t[0], t[1], t[2]}
				if !seen[key] {
					seen[key] = true
					out = append(out, t)
				}
			}
		}
	}
	return out
}

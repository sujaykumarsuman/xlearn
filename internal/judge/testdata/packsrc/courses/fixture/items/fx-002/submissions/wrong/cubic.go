package main

import "sort"

// trios (wrong: TLE) sorts and tries every triple, skipping repeated values at each level.
func trios(nums []int, target int) [][]int {
	a := append([]int(nil), nums...)
	sort.Ints(a)
	out := [][]int{}
	for i := 0; i < len(a); i++ {
		if i > 0 && a[i] == a[i-1] {
			continue
		}
		for j := i + 1; j < len(a); j++ {
			if j > i+1 && a[j] == a[j-1] {
				continue
			}
			for k := j + 1; k < len(a); k++ {
				if k > j+1 && a[k] == a[k-1] {
					continue
				}
				if a[i]+a[j]+a[k] == target {
					out = append(out, []int{a[i], a[j], a[k]})
				}
			}
		}
	}
	return out
}

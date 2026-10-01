package main

import "sort"

// trios sorts, fixes the smallest value and closes in with two pointers, skipping repeats.
func trios(nums []int, target int) [][]int {
	a := append([]int(nil), nums...)
	sort.Ints(a)
	out := [][]int{}
	for i := 0; i+2 < len(a); i++ {
		if i > 0 && a[i] == a[i-1] {
			continue
		}
		l, r := i+1, len(a)-1
		for l < r {
			switch s := a[i] + a[l] + a[r]; {
			case s < target:
				l++
			case s > target:
				r--
			default:
				out = append(out, []int{a[i], a[l], a[r]})
				for l < r && a[l] == a[l+1] {
					l++
				}
				for l < r && a[r] == a[r-1] {
					r--
				}
				l++
				r--
			}
		}
	}
	return out
}

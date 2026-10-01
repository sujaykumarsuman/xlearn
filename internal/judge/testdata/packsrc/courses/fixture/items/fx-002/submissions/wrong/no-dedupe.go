package main

import "sort"

// trios (wrong: WA) uses two pointers but never skips repeated values, so a trio can repeat.
func trios(nums []int, target int) [][]int {
	a := append([]int(nil), nums...)
	sort.Ints(a)
	out := [][]int{}
	for i := 0; i+2 < len(a); i++ {
		l, r := i+1, len(a)-1
		for l < r {
			switch s := a[i] + a[l] + a[r]; {
			case s < target:
				l++
			case s > target:
				r--
			default:
				out = append(out, []int{a[i], a[l], a[r]})
				l++
				r--
			}
		}
	}
	return out
}

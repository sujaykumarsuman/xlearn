package main

// Calibration kernel "sort" (t3 §7.3): sort n 32-bit LCG values, then a position-weighted sum.

import "slices"

func kernel(n int) int64 {
	a := make([]int64, n)
	x := uint32(12345)
	for i := range a {
		x = x*1664525 + 1013904223
		a[i] = int64(x)
	}
	slices.Sort(a)
	var s int64
	for i, v := range a {
		s = (s + v%1000000007*int64(i+1)) % 1000000007
	}
	return s
}

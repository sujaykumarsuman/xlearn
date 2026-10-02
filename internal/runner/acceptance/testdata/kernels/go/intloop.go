package main

// Calibration kernel "intloop" (t3 §7.3): n steps of a 32-bit LCG, summing the high halves.

func kernel(n int) int64 {
	x := uint32(1)
	var s int64
	for i := 0; i < n; i++ {
		x = x*1664525 + 1013904223
		s += int64(x >> 16)
	}
	return s % 1000000007
}

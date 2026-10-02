package main

// Calibration kernel "maphash" (t3 §7.3, map-heavy): n updates then n lookups on a hash map with
// keys drawn from 4n values.

func kernel(n int) int64 {
	m := make(map[int64]int64)
	x := uint32(7)
	space := int64(4 * n)
	for i := 0; i < n; i++ {
		x = x*1664525 + 1013904223
		m[int64(x)%space] += int64(i)
	}
	var s, hits int64
	for i := 0; i < n; i++ {
		x = x*1664525 + 1013904223
		if v, ok := m[int64(x)%space]; ok {
			hits++
			s = (s + v) % 1000000007
		}
	}
	return (s + hits) % 1000000007
}

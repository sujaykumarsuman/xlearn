package main

// Calibration kernel "alloc" (t3 §7.3, alloc/GC-heavy): four rounds of building a linked list of
// n/4 heap nodes and walking it.

type cell struct {
	val  int64
	next *cell
}

func kernel(n int) int64 {
	x := uint32(99)
	var s int64
	for r := 0; r < 4; r++ {
		var head *cell
		for i := 0; i < n/4; i++ {
			x = x*1664525 + 1013904223
			head = &cell{int64(x >> 8), head}
		}
		for c := head; c != nil; c = c.next {
			s = (s + c.val) % 1000000007
		}
	}
	return s
}

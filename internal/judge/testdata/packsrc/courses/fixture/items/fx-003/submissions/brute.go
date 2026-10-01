package main

// FixedShelf (oracle) keeps (key, value) pairs in insertion order and scans them.
type FixedShelf struct {
	capacity int
	pairs    [][2]int
}

func Constructor(capacity int) FixedShelf {
	return FixedShelf{capacity: capacity}
}

func (s *FixedShelf) Put(key int, value int) {
	for i := range s.pairs {
		if s.pairs[i][0] == key {
			s.pairs[i][1] = value
			return
		}
	}
	if len(s.pairs) == s.capacity {
		s.pairs = append(s.pairs[:0:0], s.pairs[1:]...)
	}
	s.pairs = append(s.pairs, [2]int{key, value})
}

func (s *FixedShelf) Get(key int) int {
	for _, p := range s.pairs {
		if p[0] == key {
			return p[1]
		}
	}
	return -1
}
